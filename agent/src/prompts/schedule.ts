import { Schedule } from '../types'

export interface ScheduleParams {
  schedule: Schedule
  date: Date
  timezone?: string
}

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
  return `
** This is a scheduled task automatically send to you by the system **
---
${Bun.YAML.stringify(headers)}
---

${params.schedule.command}
  `.trim()
}