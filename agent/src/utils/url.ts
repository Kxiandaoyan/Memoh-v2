/**
 * Remove trailing slash from a URL string.
 */
export function normalizeBaseUrl(url: string): string {
  return url.replace(/\/$/, '')
}
