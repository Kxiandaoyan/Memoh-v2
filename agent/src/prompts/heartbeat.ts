import { Schedule } from '../types'

export interface HeartbeatParams {
  schedule: Schedule
  date: Date
  timezone?: string
}

export const heartbeat = (params: HeartbeatParams) => {
  const tz = params.timezone || 'UTC'
  const headers: Record<string, string | number | null | undefined> = {
    'heartbeat-id': params.schedule.id,
    'trigger-reason': params.schedule.description,
    'interval': params.schedule.pattern || 'unknown',
    'timezone': tz,
  }

  return `
** Internal heartbeat maintenance triggered **
---
${Bun.YAML.stringify(headers)}
---

Execute the following maintenance task:

${params.schedule.command}

---
IMPORTANT: This is a background maintenance task. Do NOT call the \`send\` tool to message the user — heartbeat results should be silent. Only use \`send\` if the task itself explicitly requires notifying the user about something urgent (e.g. a critical alert). Routine reports like "no pending tasks" must NOT be sent.
  `.trim()
}
