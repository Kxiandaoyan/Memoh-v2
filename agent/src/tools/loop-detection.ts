/**
 * Tool call loop detection.
 *
 * Wraps every tool with three detectors that prevent the agent from burning
 * tokens on stuck infinite-loop patterns:
 *
 *   1. repeatedNoProgress  – same tool + same params + same result N times
 *   2. pingPong            – two tools alternating with no result change
 *   3. globalCircuitBreaker – any 25 identical (tool+params) calls in window
 *
 * Each detector threshold is conservative; only kills loops that have *clearly*
 * made zero progress. Normal polling patterns (e.g. checking a build 3-4 times)
 * are not affected.
 */

import type { ToolSet } from 'ai'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ToolCallRecord {
  toolName: string
  paramsHash: string
  resultHash: string
  timestamp: number
}

interface LoopDetectResult {
  stuck: boolean
  detector?: 'repeatedNoProgress' | 'pingPong' | 'globalCircuitBreaker'
  message?: string
}

const WINDOW_SIZE = 40           // sliding window of recent calls
const REPEAT_NO_PROGRESS = 8    // same tool+params+result → stuck
const PING_PONG_PAIRS   = 5     // A→B→A→B... pairs with no change → stuck
const GLOBAL_BREAKER    = 25    // identical (tool+params) calls in window

// ---------------------------------------------------------------------------
// Per-session state
// ---------------------------------------------------------------------------

const sessionHistories = new Map<string, ToolCallRecord[]>()

function getHistory(sessionId: string): ToolCallRecord[] {
  if (!sessionHistories.has(sessionId)) {
    sessionHistories.set(sessionId, [])
  }
  return sessionHistories.get(sessionId)!
}

function appendRecord(sessionId: string, record: ToolCallRecord): void {
  const history = getHistory(sessionId)
  history.push(record)
  if (history.length > WINDOW_SIZE) {
    history.splice(0, history.length - WINDOW_SIZE)
  }
}

/** Clean up session state after a conversation round ends. */
export function clearLoopDetectionState(sessionId: string): void {
  sessionHistories.delete(sessionId)
}

// ---------------------------------------------------------------------------
// Hashing
// ---------------------------------------------------------------------------

function simpleHash(value: unknown): string {
  const str = typeof value === 'string' ? value : JSON.stringify(value)
  let h = 0x811c9dc5
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i)
    h = (h * 0x01000193) >>> 0
  }
  return h.toString(16)
}

// ---------------------------------------------------------------------------
// Detection logic
// ---------------------------------------------------------------------------

function detectLoop(
  history: ToolCallRecord[],
  toolName: string,
  paramsHash: string,
): LoopDetectResult {
  if (history.length < 2) return { stuck: false }

  const callKey = `${toolName}:${paramsHash}`

  // 1. Global circuit breaker: count identical (tool+params) in entire window
  const globalCount = history.filter(
    (r) => r.toolName === toolName && r.paramsHash === paramsHash,
  ).length
  if (globalCount >= GLOBAL_BREAKER) {
    return {
      stuck: true,
      detector: 'globalCircuitBreaker',
      message: `[LoopDetected] Tool "${toolName}" called ${globalCount} times with identical parameters. Aborting to prevent runaway execution.`,
    }
  }

  // 2. Repeated no-progress: last N calls same tool+params+result
  if (history.length >= REPEAT_NO_PROGRESS) {
    const tail = history.slice(-REPEAT_NO_PROGRESS)
    const allSame = tail.every(
      (r) => r.toolName === toolName && r.paramsHash === paramsHash && r.resultHash === tail[0].resultHash,
    )
    if (allSame) {
      return {
        stuck: true,
        detector: 'repeatedNoProgress',
        message: `[LoopDetected] Tool "${toolName}" repeated ${REPEAT_NO_PROGRESS} times with identical parameters and results — no progress detected. Stop and report what happened so far.`,
      }
    }
  }

  // 3. Ping-pong: two tools alternating with no progress
  if (history.length >= PING_PONG_PAIRS * 2) {
    const tail = history.slice(-(PING_PONG_PAIRS * 2))
    const evenKey = `${tail[0].toolName}:${tail[0].paramsHash}:${tail[0].resultHash}`
    const oddKey  = `${tail[1].toolName}:${tail[1].paramsHash}:${tail[1].resultHash}`
    // Check if entries alternate between exactly two keys
    let isPingPong = tail[0].toolName !== tail[1].toolName // must be two different tools
    if (isPingPong) {
      for (let i = 0; i < tail.length; i++) {
        const expected = i % 2 === 0 ? evenKey : oddKey
        const actual   = `${tail[i].toolName}:${tail[i].paramsHash}:${tail[i].resultHash}`
        if (actual !== expected) {
          isPingPong = false
          break
        }
      }
    }
    if (isPingPong) {
      return {
        stuck: true,
        detector: 'pingPong',
        message: `[LoopDetected] Tools "${tail[0].toolName}" and "${tail[1].toolName}" are alternating with no progress (${PING_PONG_PAIRS} cycles). Stop and report the current state.`,
      }
    }
  }

  return { stuck: false }
}

// ---------------------------------------------------------------------------
// Public wrapper
// ---------------------------------------------------------------------------

/**
 * Wraps every tool in `tools` with loop detection keyed by `sessionId`.
 * Throws a descriptive error when a loop is detected, which surfaces to the
 * LLM as a tool error — prompting it to stop and report rather than retry.
 */
export function wrapToolsWithLoopDetection(tools: ToolSet, sessionId: string): ToolSet {
  if (!sessionId) return tools

  const wrapped: ToolSet = {}

  for (const [name, tool] of Object.entries(tools)) {
    if (!tool || typeof (tool as { execute?: unknown }).execute !== 'function') {
      wrapped[name] = tool
      continue
    }

    const originalExecute = (tool as { execute: (...args: unknown[]) => Promise<unknown> }).execute

    wrapped[name] = {
      ...tool,
      execute: async (...args: unknown[]) => {
        const [, params] = args as [unknown, unknown, ...unknown[]]
        const paramsHash = simpleHash(params)

        const history = getHistory(sessionId)
        const loopCheck = detectLoop(history, name, paramsHash)

        if (loopCheck.stuck) {
          // Record the blocked call so it shows in history.
          appendRecord(sessionId, {
            toolName: name,
            paramsHash,
            resultHash: 'BLOCKED',
            timestamp: Date.now(),
          })
          throw new Error(loopCheck.message!)
        }

        // Execute the real tool.
        const result = await originalExecute(...args)

        appendRecord(sessionId, {
          toolName: name,
          paramsHash,
          resultHash: simpleHash(result),
          timestamp: Date.now(),
        })

        return result
      },
    } as typeof tool
  }

  return wrapped
}
