export interface SubagentRun {
  id: string
  run_id: string
  bot_id: string
  name: string
  task: string
  status: string
  spawn_depth: number
  parent_run_id?: string
  result_summary?: string
  error_message?: string
  started_at: string
  ended_at?: string
  created_at: string
}

import { client } from '@memoh/sdk/client'

export async function listSubagentRuns(botId: string): Promise<SubagentRun[]> {
  const response = await client.get('/subagent-runs', { params: { botId } })
  const data = response.data as SubagentRun[] | null
  return Array.isArray(data) ? data : []
}

export async function deleteSubagentRun(runId: string): Promise<void> {
  await client.delete(`/subagent-runs/${encodeURIComponent(runId)}`)
}
