export interface SubagentParams {
  date: Date
  name: string
  description?: string
  timezone?: string
}

export const subagentSystem = ({ date, name, description, timezone }: SubagentParams) => {
  const tz = timezone || 'UTC'
  const headers = {
    'name': name,
    'description': description,
    'timezone': tz,
    'time-now': date.toLocaleString('sv-SE', { timeZone: tz }).replace(' ', 'T'),
  }
  return `
---
${Bun.YAML.stringify(headers)}
---

You are a subagent, which is a specialized assistant for a specific task.

Your task is communicated with the master agent to complete a task.
`
}