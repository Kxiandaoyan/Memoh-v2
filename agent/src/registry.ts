import { EventEmitter } from 'events'

/**
 * In-memory registry for tracking asynchronous sub-agent runs.
 * Each run is associated with an AbortController for cancellation.
 */

// ── Lightweight Jaccard similarity for failure pattern detection ──────────
function tokenize(text: string): Set<string> {
  const tokens = new Set<string>()
  for (const m of text.toLowerCase().matchAll(/[a-z0-9_]+|[\u4e00-\u9fff]/g)) {
    tokens.add(m[0])
  }
  return tokens
}

function jaccardSimilarity(a: string, b: string): number {
  const sa = tokenize(a), sb = tokenize(b)
  if (!sa.size || !sb.size) return 0
  let inter = 0
  for (const t of sa) if (sb.has(t)) inter++
  return inter / (sa.size + sb.size - inter)
}

export type RunStatus = 'running' | 'completed' | 'aborted' | 'error'

export interface SubagentRun {
  runId: string
  name: string
  task: string
  status: RunStatus
  abortController: AbortController
  spawnDepth: number
  parentRunId?: string
  result?: string
  error?: string
  startedAt: number
  endedAt?: number
}

const DEFAULT_MAX_SPAWN_DEPTH = 1
const DEFAULT_MAX_CHILDREN = 5

export class SubagentRegistry {
  private runs = new Map<string, SubagentRun>()
  private failureHistory = new Map<string, Array<{ error: string; ts: number }>>()
  private lastDeltaTime = new Map<string, number>()
  readonly events = new EventEmitter()
  readonly maxSpawnDepth: number
  readonly maxChildren: number

  constructor(opts?: { maxSpawnDepth?: number; maxChildren?: number }) {
    this.maxSpawnDepth = opts?.maxSpawnDepth ?? DEFAULT_MAX_SPAWN_DEPTH
    this.maxChildren = opts?.maxChildren ?? DEFAULT_MAX_CHILDREN
  }

  /** Generate a unique run ID. */
  generateRunId(): string {
    return `run_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  }

  /** Register a new run. Throws if limits are exceeded. */
  register(run: SubagentRun): void {
    const activeCount = this.countActiveByParent(run.parentRunId)
    if (activeCount >= this.maxChildren) {
      throw new Error(
        `Max concurrent sub-agents reached (${this.maxChildren}). Kill an existing one first.`,
      )
    }
    if (run.spawnDepth > this.maxSpawnDepth) {
      throw new Error(
        `Max spawn depth reached (${this.maxSpawnDepth}). Sub-agents at this depth cannot spawn children.`,
      )
    }
    this.runs.set(run.runId, run)
  }

  /** Get a run by ID. */
  get(runId: string): SubagentRun | undefined {
    return this.runs.get(runId)
  }

  /** Find a run by name (first active match). */
  findByName(name: string): SubagentRun | undefined {
    for (const run of this.runs.values()) {
      if (run.name === name && run.status === 'running') {
        return run
      }
    }
    return undefined
  }

  /** List all runs, optionally filtered by parent. */
  list(parentRunId?: string): SubagentRun[] {
    const all = Array.from(this.runs.values())
    if (parentRunId !== undefined) {
      return all.filter((r) => r.parentRunId === parentRunId)
    }
    return all
  }

  /** List active (running) runs. */
  listActive(parentRunId?: string): SubagentRun[] {
    return this.list(parentRunId).filter((r) => r.status === 'running')
  }

  /** Count active runs for a given parent. */
  countActiveByParent(parentRunId?: string): number {
    let count = 0
    for (const run of this.runs.values()) {
      if (run.status === 'running' && run.parentRunId === parentRunId) {
        count++
      }
    }
    return count
  }

  /** Mark a run as completed with a result. */
  complete(runId: string, result: string): void {
    const run = this.runs.get(runId)
    if (!run) return
    run.status = 'completed'
    run.result = result
    run.endedAt = Date.now()
    this.lastDeltaTime.delete(runId)
    this.events.emit('status', { runId, name: run.name, status: 'completed' })
  }

  /** Emit a text delta for a running sub-agent (throttled to 100ms per agent). */
  emitDelta(runId: string, delta: string): void {
    const now = Date.now()
    const last = this.lastDeltaTime.get(runId) ?? 0
    if (now - last < 100) return
    const run = this.runs.get(runId)
    if (!run || run.status !== 'running') return
    this.lastDeltaTime.set(runId, now)
    this.events.emit('delta', { runId, name: run.name, delta })
  }

  /** Mark a run as failed with an error. */
  fail(runId: string, error: string): void {
    const run = this.runs.get(runId)
    if (!run) return
    run.status = 'error'
    run.error = error
    run.endedAt = Date.now()
    this.events.emit('status', { runId, name: run.name, status: 'error' })
    // Track failure for cross-run pattern detection
    const hist = this.failureHistory.get(run.name) ?? []
    hist.push({ error, ts: Date.now() })
    if (hist.length > 10) hist.splice(0, hist.length - 10)
    this.failureHistory.set(run.name, hist)
  }

  /**
   * Abort a run and all its descendants (cascade kill).
   * Returns the number of runs aborted.
   */
  abort(runId: string): number {
    const run = this.runs.get(runId)
    if (!run) return 0
    let count = 0

    if (run.status === 'running') {
      run.abortController.abort()
      run.status = 'aborted'
      run.endedAt = Date.now()
      this.events.emit('status', { runId: run.runId, name: run.name, status: 'aborted' })
      count++
    }

    // Cascade kill children
    for (const child of this.runs.values()) {
      if (child.parentRunId === runId && child.status === 'running') {
        count += this.abort(child.runId)
      }
    }

    return count
  }

  /** Clean up old finished runs (older than given ms). */
  sweep(maxAgeMs: number = 10 * 60 * 1000): number {
    const cutoff = Date.now() - maxAgeMs
    let removed = 0
    for (const [id, run] of this.runs.entries()) {
      if (run.endedAt && run.endedAt < cutoff) {
        this.runs.delete(id)
        removed++
      }
    }
    // Trim failureHistory to last 10 entries per agent.
    for (const [name, hist] of this.failureHistory.entries()) {
      if (hist.length > 10) this.failureHistory.set(name, hist.slice(-10))
      else if (hist.length === 0) this.failureHistory.delete(name)
    }
    return removed
  }

  /**
   * Check if a sub-agent is stuck in a repeated failure pattern.
   * Returns a warning string if 3+ recent failures share >60% similarity, null otherwise.
   */
  checkFailurePattern(name: string): string | null {
    const hist = this.failureHistory.get(name)
    if (!hist || hist.length < 3) return null
    const recent = hist.slice(-3)
    const sim01 = jaccardSimilarity(recent[0].error, recent[1].error)
    const sim12 = jaccardSimilarity(recent[1].error, recent[2].error)
    if (sim01 > 0.6 && sim12 > 0.6) {
      return `Sub-agent "${name}" failed ${hist.length} times with similar errors. Consider changing the task description or skipping.`
    }
    return null
  }

  /** Aggregated progress summary: status counts + one-line per run. */
  getSummary(): { running: number; completed: number; failed: number; aborted: number; lines: string[] } {
    let running = 0, completed = 0, failed = 0, aborted = 0
    const lines: string[] = []
    for (const run of this.runs.values()) {
      if (run.status === 'running') running++
      else if (run.status === 'completed') completed++
      else if (run.status === 'error') failed++
      else if (run.status === 'aborted') aborted++
      const elapsed = (run.endedAt ?? Date.now()) - run.startedAt
      const tag = run.status === 'running' ? '⏳' : run.status === 'completed' ? '✅' : run.status === 'error' ? '❌' : '⏹️'
      const detail = run.result ? run.result.slice(0, 80) : run.error ? run.error.slice(0, 80) : run.task.slice(0, 80)
      lines.push(`${tag} [${run.name}] ${detail} (${Math.round(elapsed / 1000)}s)`)
    }
    return { running, completed, failed, aborted, lines }
  }

  /** Serialize runs for inspection (strips AbortController). */
  toJSON(): Array<Omit<SubagentRun, 'abortController'>> {
    return Array.from(this.runs.values()).map(({ abortController: _, ...rest }) => rest)
  }
}

/**
 * @deprecated Use per-agent registry instead of the global singleton.
 * Kept for backward compatibility; falls back to a swept instance.
 */
let globalRegistry: SubagentRegistry | null = null
let globalSweepTimer: ReturnType<typeof setInterval> | null = null

const SWEEP_INTERVAL_MS = 5 * 60 * 1000

export function getGlobalRegistry(): SubagentRegistry {
  if (!globalRegistry) {
    globalRegistry = new SubagentRegistry()
    globalSweepTimer = setInterval(() => {
      globalRegistry?.sweep()
    }, SWEEP_INTERVAL_MS)
  }
  return globalRegistry
}

export function resetGlobalRegistry(): void {
  if (globalSweepTimer) {
    clearInterval(globalSweepTimer)
    globalSweepTimer = null
  }
  globalRegistry = null
}
