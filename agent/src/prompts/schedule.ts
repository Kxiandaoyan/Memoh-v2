import { Schedule } from '../types'

export interface ScheduleParams {
  schedule: Schedule
  date: Date
  timezone?: string
}

// Matches: send 'text', send "text", send `text` (with optional trailing whitespace)
const SEND_PATTERN = /^send\s+(['"`])([\s\S]*?)\1\s*$/

export const schedule = (params: ScheduleParams) => {
  const tz = params.timezone || 'UTC'
  const headers = {
    'schedule-name': params.schedule.name,
    'schedule-description': params.schedule.description,
    'schedule-id': params.schedule.id,
    'max-calls': params.schedule.maxCalls ?? 'Unlimited',
    'cron-pattern': params.schedule.pattern,
    'timezone': tz,
  }

  const sendMatch = params.schedule.command.match(SEND_PATTERN)
  if (sendMatch) {
    const messageText = sendMatch[2]
    return `
** Scheduled task triggered: ${params.schedule.name} **
---
${Bun.YAML.stringify(headers)}
---

REQUIRED ACTION: You MUST call the \`send\` tool RIGHT NOW to deliver the following message to the current channel. Do NOT output any text response — only call the tool.

send tool arguments:
- text: ${JSON.stringify(messageText)}
- (platform and target default to the current session — do NOT specify them unless you have an explicit override)

Call the send tool immediately. This is your only task.
    `.trim()
  }

  return `
** This is a scheduled task automatically triggered by the system **
---
${Bun.YAML.stringify(headers)}
---

Execute the following command:

${params.schedule.command}

---
MANDATORY FOLLOW-UP: After completing the task above, you MUST call the \`send\` tool to deliver a brief result summary to the user. Use the current session platform and target (do NOT specify them explicitly unless you have an override). Do NOT skip this step even if the task produced no output — always report what happened.
  `.trim()
}