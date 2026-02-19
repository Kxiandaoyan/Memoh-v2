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
MANDATORY FOLLOW-UP: After completing the task above, you MUST call the \`send\` tool to deliver a brief result summary to the user. Use the current session platform and target (do NOT specify them explicitly unless you have an override). Do NOT skip this step even if the task produced no output — always report what happened.
  `.trim()
}
