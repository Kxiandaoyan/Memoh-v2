/**
 * Search result types
 */

export interface SearchResult {
  /** Result title */
  title: string;
  /** Result URL */
  url: string;
  /** Text snippet/description */
  snippet: string;
  /** Source engine */
  source: 'bing' | 'google';
  /** Position in results (1-based) */
  position: number;
}

export interface SearchResponse {
  /** Search query */
  query: string;
  /** Engine used for this response */
  engine: 'bing' | 'google';
  /** Search results */
  results: SearchResult[];
  /** Total results found */
  totalResults: number;
  /** Search timestamp */
  timestamp: number;
  /** Time taken in milliseconds */
  duration: number;
}
