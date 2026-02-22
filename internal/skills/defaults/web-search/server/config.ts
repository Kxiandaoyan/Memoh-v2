/**
 * Web Search Skill Configuration
 */

export interface BrowserConfig {
  /** Chrome executable path (auto-detected if not provided) */
  chromePath?: string;
  /** CDP debugging port */
  cdpPort: number;
  /** User data directory for browser isolation */
  userDataDir?: string;
  /** Whether to run browser headless */
  headless: boolean;
  /** Additional Chrome flags */
  chromeFlags?: string[];
}

export interface ServerConfig {
  /** Bridge server port */
  port: number;
  /** Bridge server host */
  host: string;
}

export interface SearchConfig {
  /** Default search engine */
  defaultEngine: 'auto' | 'bing' | 'google';
  /** Engine fallback order when defaultEngine is auto */
  fallbackOrder: Array<'google' | 'bing'>;
  /** Default max results per search */
  defaultMaxResults: number;
  /** Search timeout in milliseconds */
  searchTimeout: number;
  /** Navigation timeout in milliseconds */
  navigationTimeout: number;
}

export interface Config {
  browser: BrowserConfig;
  server: ServerConfig;
  search: SearchConfig;
}

/**
 * Default configuration
 */
export const defaultConfig: Config = {
  browser: {
    cdpPort: 9222,
    headless: false, // Always visible for transparency
    chromeFlags: [
      '--no-first-run',
      '--no-default-browser-check',
      '--disable-background-timer-throttling',
      '--disable-backgrounding-occluded-windows',
      '--disable-renderer-backgrounding'
    ]
  },
  server: {
    port: 8923,
    host: '127.0.0.1' // Localhost only for security
  },
  search: {
    defaultEngine: 'auto',
    fallbackOrder: ['google', 'bing'],
    defaultMaxResults: 10,
    searchTimeout: 30000, // 30 seconds
    navigationTimeout: 15000 // 15 seconds
  }
};

/**
 * Merge user config with defaults
 */
export function mergeConfig(userConfig?: Partial<Config>): Config {
  if (!userConfig) {
    return defaultConfig;
  }

  return {
    browser: { ...defaultConfig.browser, ...userConfig.browser },
    server: { ...defaultConfig.server, ...userConfig.server },
    search: { ...defaultConfig.search, ...userConfig.search }
  };
}
