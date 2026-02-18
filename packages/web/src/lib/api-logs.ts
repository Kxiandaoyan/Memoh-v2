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
