/**
 * Resolve an unknown error into a human-readable message.
 * Tries Error.message, plain object .message/.error/.detail, and falls back to the given default.
 */
export function resolveErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  if (error && typeof error === 'object' && 'message' in error) {
    const msg = (error as { message?: string }).message
    if (msg && msg.trim()) return msg
  }
  if (error && typeof error === 'object') {
    const body = error as { error?: string; detail?: string }
    const detail = body.error || body.detail
    if (detail && detail.trim()) return `${fallback}: ${detail}`
  }
  return fallback
}
