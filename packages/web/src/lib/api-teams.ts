import { client } from '@memoh/sdk/client'

export interface Team {
  id: string
  owner_user_id: string
  name: string
  manager_bot_id: string
  created_at: string
  updated_at: string
  /** Populated client-side from team members list. */
  memberBotIds?: string[]
}

export interface TeamMember {
  id: string
  team_id: string
  source_bot_id: string
  target_bot_id: string
  role_description: string
  target_display_name: string
  created_at: string
}

export interface CreateTeamInput {
  name: string
  members: Array<{ bot_id: string; role_description: string }>
  heartbeat_prompt?: string
}

export interface CallLog {
  id: string
  caller_bot_id: string
  target_bot_id: string
  message: string
  result: string
  status: 'pending' | 'completed' | 'failed' | 'timeout'
  call_depth: number
  created_at: string
  completed_at?: string
}

export async function listTeams(): Promise<Team[]> {
  const res = await client.get({ url: '/teams' }) as { data?: { items?: Team[] } }
  return res.data?.items ?? []
}

export async function createTeam(input: CreateTeamInput): Promise<Team> {
  const res = await client.post({ url: '/teams', body: input as any, throwOnError: true }) as { data: Team }
  return res.data
}

export async function deleteTeam(teamId: string): Promise<void> {
  await client.delete({ url: `/teams/${teamId}` as any, throwOnError: true })
}

export async function listTeamMembers(teamId: string): Promise<TeamMember[]> {
  const res = await client.get({ url: `/teams/${teamId}/members` as any }) as { data?: { items?: TeamMember[] } }
  return res.data?.items ?? []
}

export async function listBotCallLogs(botId: string): Promise<CallLog[]> {
  const res = await client.get({ url: `/bots/${botId}/call-logs` as any }) as { data?: { items?: CallLog[] } }
  return res.data?.items ?? []
}
