// Process Logs API helper
// Note: These functions use the SDK's client directly

import { client } from '@memoh/sdk/client'

export interface ProcessLog {
  id: string
  bot_id: string
  chat_id: string
  trace_id: string
  user_id?: string
  channel?: string
  step: string
  level: string
  message?: string
  data?: Record<string, any>
  duration_ms?: number
  created_at: string
}

export interface ProcessLogStats {
  step: string
  count: number
  avgDurationMs: number
}

export async function getProcessLogs(botId: string, limit = 100): Promise<ProcessLog[]> {
  const response = await client.get('/logs/recent', {
    params: { botId, limit },
  })
  return response.json()
}

export async function getProcessLogByTrace(traceId: string): Promise<ProcessLog[]> {
  const response = await client.get(`/logs/trace/${traceId}`)
  return response.json()
}

export async function getProcessLogsByChat(botId: string, chatId: string, limit = 100): Promise<ProcessLog[]> {
  const response = await client.get(`/logs/chat/${chatId}`, {
    params: { botId, limit },
  })
  return response.json()
}

export async function getProcessLogStats(botId: string): Promise<ProcessLogStats[]> {
  const response = await client.get('/logs/stats', {
    params: { botId },
  })
  return response.json()
}

export interface TraceExport {
  version: string
  exported_at: string
  trace_id: string
  bot_id: string
  chat_id: string
  channel?: string
  time_range: { start: string; end: string }
  total_duration_ms: number
  summary: {
    user_query: string
    assistant_response?: string
    model?: string
    provider?: string
    token_usage?: Record<string, any>
    steps_count: number
    errors?: string[]
    warnings?: string[]
  }
  steps: Array<{
    step: string
    level: string
    message?: string
    data?: Record<string, any>
    duration_ms?: number
    created_at: string
  }>
}

export async function exportTrace(traceId: string): Promise<TraceExport> {
  const response = await client.get({ url: `/logs/trace/${traceId}/export`, throwOnError: true }) as any
  if (response?.json) return response.json()
  return response?.data ?? response
}

export interface ChatExport {
  version: string
  exported_at: string
  bot_id: string
  chat_id: string
  channel?: string
  total_rounds: number
  time_range: { start: string; end: string }
  total_duration_ms: number
  rounds: TraceExport[]
}

export async function exportChat(botId: string, chatId: string, limit = 2000): Promise<ChatExport> {
  const response = await client.get({ url: `/logs/chat/${chatId}/export`, query: { botId, limit: String(limit) }, throwOnError: true }) as any
  if (response?.json) return response.json()
  return response?.data ?? response
}
