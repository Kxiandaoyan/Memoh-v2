/**
 * Centralized configuration constants for the agent gateway.
 * Adjust these values without hunting through multiple files.
 */

// ── System prompt truncation ──
export const HEAD_RATIO = 0.7
export const TAIL_RATIO = 0.2
export const FILE_SIZE_WARN_THRESHOLD = 8000

export const CHAR_BUDGETS = {
  full:    { soul: 3000, tools: 3000 },
  minimal: { soul: 800,  tools: 800  },
  micro:   { soul: 0,    tools: 0    },
} as const

// ── SSE payload truncation ──
export const DEFAULT_CONTEXT_WINDOW = 128_000
export const TOOL_RESULT_CONTEXT_SHARE = 0.3
export const CHARS_PER_TOKEN = 3.5
export const SSE_HEAD_CHARS = 1500
export const SSE_TAIL_CHARS = 1500

// ── Loop detection ──
export const LOOP_WINDOW_SIZE = 40
export const LOOP_REPEAT_NO_PROGRESS = 8
export const LOOP_PING_PONG_PAIRS = 5
export const LOOP_GLOBAL_BREAKER = 25

// ── Timeouts ──
export const MCP_CONNECT_TIMEOUT_MS = 30_000
export const SUBAGENT_TIMEOUT_MS = 180_000
export const MAX_SUBAGENT_CONTEXT = 20
export const SCHEDULE_TIMEOUT_MINUTES = 10

// ── System file cache ──
export const SYSTEM_FILE_CACHE_TTL_MS = 60_000
