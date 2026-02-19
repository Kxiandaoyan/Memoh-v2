/**
 * Retry utilities for LLM API calls.
 *
 * Only sync (non-streaming) calls should be wrapped — retrying mid-stream
 * would corrupt output already sent to the client.
 */

/**
 * Calls fn up to maxAttempts times (default 3). On each failure it checks
 * isRetryable; if false the error is re-thrown immediately. Between attempts
 * it waits baseDelayMs * 2^(attempt-1) milliseconds (500 ms, 1000 ms).
 */
export async function withRetry<T>(
  fn: () => Promise<T>,
  isRetryable: (err: unknown) => boolean,
  maxAttempts = 3,
  baseDelayMs = 500,
): Promise<T> {
  let lastErr: unknown
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    if (attempt > 0) {
      const delay = baseDelayMs * Math.pow(2, attempt - 1)
      await sleep(delay)
    }
    try {
      return await fn()
    } catch (err) {
      if (!isRetryable(err)) {
        throw err
      }
      lastErr = err
    }
  }
  throw lastErr
}

/**
 * Returns true for transient failures that are safe to retry:
 * - HTTP 429 (rate limit)
 * - HTTP 5xx (server error)
 * - Network errors (fetch failed, ECONNRESET, ETIMEDOUT, etc.)
 *
 * Returns false for permanent failures:
 * - HTTP 400/401/403/422 (client error, auth, validation, context overflow)
 * - Tool execution errors
 */
export function isRetryableLLMError(err: unknown): boolean {
  if (err == null) return false

  // AI SDK APICallError — check status code.
  if (isAPICallError(err)) {
    const status = (err as { statusCode?: number }).statusCode ?? 0
    // 429 and 5xx are transient; everything else is not.
    return status === 429 || status >= 500
  }

  // Network / fetch errors — check message for known transient patterns.
  const msg = errorMessage(err).toLowerCase()
  const transientPatterns = [
    'fetch failed',
    'network error',
    'econnreset',
    'econnrefused',
    'etimedout',
    'socket hang up',
    'connection reset',
    'failed to fetch',
    'network request failed',
  ]
  for (const pattern of transientPatterns) {
    if (msg.includes(pattern)) return true
  }

  return false
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  return String(err)
}

/**
 * Rough duck-type check for AI SDK's APICallError or similar typed errors
 * that carry an HTTP status code.
 */
function isAPICallError(err: unknown): boolean {
  return (
    err != null &&
    typeof err === 'object' &&
    'statusCode' in err &&
    typeof (err as Record<string, unknown>).statusCode === 'number'
  )
}
