/**
 * SSE payload truncation utilities.
 *
 * Large tool results or accumulated messages can exceed the Go server's
 * bufio.Scanner buffer limit, causing "token too long" errors.
 * These helpers truncate oversized fields before yielding SSE events.
 */

const DEFAULT_CONTEXT_WINDOW = 128_000
const TOOL_RESULT_CONTEXT_SHARE = 0.3
const CHARS_PER_TOKEN = 3.5
const HEAD_CHARS = 1500
const TAIL_CHARS = 1500
const MAX_TOOL_RESULT_CHARS = Math.floor(DEFAULT_CONTEXT_WINDOW * TOOL_RESULT_CONTEXT_SHARE * CHARS_PER_TOKEN)

export function computeMaxToolResultChars(contextWindow?: number): number {
  const cw = contextWindow && contextWindow > 0 ? contextWindow : DEFAULT_CONTEXT_WINDOW
  const maxChars = Math.floor(cw * TOOL_RESULT_CONTEXT_SHARE * CHARS_PER_TOKEN)
  return Math.max(maxChars, HEAD_CHARS + TAIL_CHARS + 200)
}

function truncateString(s: string, max: number): string {
  if (s.length <= max) return s
  if (max >= HEAD_CHARS + TAIL_CHARS + 100) {
    const head = s.slice(0, HEAD_CHARS)
    const tail = s.slice(-TAIL_CHARS)
    return head + `\n\n[... content trimmed, original ${s.length} chars ...]\n\n` + tail
  }
  return s.slice(0, max) + `\n...[truncated: ${s.length} total chars]`
}

/**
 * Truncate a single tool result value for SSE transport.
 * Accepts string, object, or array; returns a size-safe version.
 */
export function truncateToolResult(
  value: unknown,
  max = MAX_TOOL_RESULT_CHARS,
): unknown {
  if (value === null || value === undefined) return value
  if (typeof value === 'string') {
    return truncateString(value, max)
  }
  let serialized: string
  try {
    serialized = JSON.stringify(value)
  } catch {
    return value
  }
  if (serialized.length <= max) return value
  return truncateString(serialized, max)
}

/**
 * Strip large fields from a tool-result streaming chunk's metadata
 * to avoid duplicating oversized data in the SSE event.
 */
export function sanitizeToolChunkMetadata(
  chunk: Record<string, unknown>,
): Record<string, unknown> {
  if (!chunk || typeof chunk !== 'object') return {}
  const safe: Record<string, unknown> = {}
  for (const [key, val] of Object.entries(chunk)) {
    if (key === 'output' || key === 'result') continue
    safe[key] = val
  }
  return safe
}

/**
 * Truncate tool-role message contents within a messages array.
 * This keeps assistant text intact while capping tool results
 * so the serialised `agent_end` event stays within SSE limits.
 */
export function truncateMessagesForTransport<T>(
  messages: T[],
  max = MAX_TOOL_RESULT_CHARS,
): T[] {
  if (!Array.isArray(messages)) return messages
  return messages.map((msg: any) => {
    if (!msg || typeof msg !== 'object') return msg
    if (msg.role === 'tool') {
      return truncateToolMessage(msg, max)
    }
    return msg
  })
}

/**
 * Remove reasoning content parts from assistant messages.
 *
 * Vercel AI SDK stores reasoning as `{ type: "reasoning", text: "..." }` parts
 * inside assistant message content arrays. These are already sent separately in
 * the `reasoning` field of `agent_end`, so keeping them in `messages` only
 * bloats the SSE payload and risks leaking thinking text into channel replies.
 */
export function stripReasoningFromMessages<T>(messages: T[]): T[] {
  if (!Array.isArray(messages)) return messages
  return messages.map((msg: any) => {
    if (!msg || typeof msg !== 'object') return msg
    if (msg.role !== 'assistant') return msg
    if (!Array.isArray(msg.content)) return msg

    const filtered = (msg.content as any[]).filter((part: any) => {
      if (!part || typeof part !== 'object') return true
      const t = part.type
      return t !== 'reasoning'
    })
    if (filtered.length === msg.content.length) return msg
    if (filtered.length === 0) {
      return { ...msg, content: [{ type: 'text', text: '' }] }
    }
    return { ...msg, content: filtered }
  })
}

function truncateToolMessage(
  msg: Record<string, unknown>,
  max: number,
): Record<string, unknown> {
  const content = msg.content
  if (typeof content === 'string') {
    return { ...msg, content: truncateString(content, max) }
  }
  if (Array.isArray(content)) {
    return {
      ...msg,
      content: content.map((part: any) => {
        if (!part || typeof part !== 'object') return part
        if ('result' in part) {
          return { ...part, result: truncateToolResult(part.result, max) }
        }
        if ('text' in part && typeof part.text === 'string') {
          return { ...part, text: truncateString(part.text, max) }
        }
        return part
      }),
    }
  }
  return msg
}
