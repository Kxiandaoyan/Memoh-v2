export interface SubagentParams {
  date: Date
  name: string
  description?: string
  timezone?: string
  toolContext?: string
  skills?: { name: string; description: string }[]
  identityContent?: string
  soulContent?: string
  taskContent?: string
}

export const subagentSystem = ({
  date, name, description, timezone,
  toolContext, identityContent, soulContent, taskContent,
}: SubagentParams) => {
  const tz = timezone || 'UTC'
  const headers = {
    'name': name,
    'timezone': tz,
    'time-now': date.toLocaleString('sv-SE', { timeZone: tz }).replace(' ', 'T'),
  }

  const sections: string[] = []

  sections.push(`---\n${Bun.YAML.stringify(headers)}---`)

  if (identityContent?.trim()) {
    sections.push(`## Your Role\n\n${identityContent}`)
  } else {
    sections.push(
      `You are **${name}**` + (description ? ` — ${description}` : '') + '.\n\n' +
      'You are a subagent working under the main agent\'s coordination.',
    )
  }

  if (soulContent?.trim()) {
    sections.push(`## Your Principles\n\n${soulContent}`)
  }

  if (taskContent?.trim()) {
    sections.push(`## Your Workflow\n\n${taskContent}`)
  }

  if (toolContext?.trim()) {
    sections.push(`## Tools & Capabilities\n\n${toolContext}`)
  }

  sections.push(
    '## Instructions\n\n' +
    '`/data` is the private HOME — do NOT save task output here. ' +
    '`/shared` is the shared workspace; always save generated files under `/shared/`.\n\n' +
    'Complete your assigned task thoroughly. When finished, provide a clear summary of what you accomplished and any results or output file paths.',
  )

  return sections.join('\n\n')
}
