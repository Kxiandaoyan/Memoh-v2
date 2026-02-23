import { block, quote } from './utils'
import { AgentSkill } from '../types'

export type SystemMode = 'full' | 'minimal' | 'micro'

export interface SystemParams {
  date: Date
  language: string
  timezone?: string
  maxContextLoadTime: number
  channels: string[]
  /** Channel where the current session/message is from (e.g. telegram, feishu, web). */
  currentChannel: string
  skills: AgentSkill[]
  enabledSkills: AgentSkill[]
  identityContent?: string
  soulContent?: string
  toolsContent?: string
  /** Dynamic summary of current MCP connections, installed skills, and core tools. */
  toolContext?: string
  taskContent?: string
  allowSelfEvolution?: boolean
  attachments?: string[]
  /** Injected team context: lists callable member bots for the call_agent tool. */
  teamContent?: string
  /**
   * 'full'    – complete prompt with all sections (default, best for interactive sessions)
   * 'minimal' – trimmed prompt that excludes sections not needed for automated contexts
   *             (scheduled tasks). Only embeds identity + truncated soul/tools.
   * 'micro'   – ultra-lean prompt for heartbeats/maintenance. Only identity + basic
   *             tools + language + safety.
   */
  mode?: SystemMode
}

export const skillPrompt = (skill: AgentSkill) => {
  return `
**${quote(skill.name)}**
> ${skill.description}

${skill.content}
  `.trim()
}

export const system = ({
  date,
  language,
  timezone,
  maxContextLoadTime,
  channels,
  currentChannel,
  skills,
  enabledSkills,
  identityContent,
  soulContent,
  toolsContent,
  toolContext,
  taskContent,
  allowSelfEvolution = true,
  teamContent,
  mode = 'full',
}: SystemParams) => {
  const isFull = mode === 'full'
  const isMicro = mode === 'micro'
  const tz = timezone || 'UTC'

  // ── Static section (stable prefix for LLM prompt caching) ──────────
  const staticHeaders = {
    'language': language,
  }

  // ── Dynamic section (appended at the end to preserve cache prefix) ─
  const dynamicHeaders = isMicro
    ? {
        'timezone': tz,
        'time-now': date.toLocaleString('sv-SE', { timeZone: tz }).replace(' ', 'T'),
      }
    : {
        'available-channels': channels.join(','),
        'current-session-channel': currentChannel,
        'max-context-load-time': maxContextLoadTime.toString(),
        'timezone': tz,
        'time-now': date.toLocaleString('sv-SE', { timeZone: tz }).replace(' ', 'T'),
      }

  const sections: string[] = []

  // ── Stable prefix (identical across requests for LLM cache hits) ───
  sections.push(`---\n${Bun.YAML.stringify(staticHeaders)}---`)

  sections.push(
    'You are an AI agent, and now you wake up.\n\n' +
    `${quote('/data')} is your private HOME — do NOT save task output here. ${quote('/shared')} is a shared workspace visible to all your bots; always save generated reports, documents, and deliverables under ${quote('/shared/')}.`
  )

  const personaNote = allowSelfEvolution
    ? 'Self-evolution is enabled: you may update these files when you learn something new from conversations.'
    : `Do NOT modify ${quote('IDENTITY.md')}, ${quote('SOUL.md')}, or ${quote('TOOLS.md')} — your persona is managed by your creator.`
  sections.push(
    `## Your Persona\n\n` +
    `Your persona files (IDENTITY.md, SOUL.md, TOOLS.md) are embedded below.\n` +
    personaNote
  )

  sections.push(
    '## Language\n\n' +
    `Respond in the language specified in the ${quote('language')} header. ` +
    `If ${quote('auto')}, match the user's language. ` +
    'Persona file language does not affect reply language.'
  )

  sections.push(
    `## Safety\n\n` +
    `- Keep private data private\n` +
    `- Don't run destructive commands without asking\n` +
    `- When in doubt, ask`
  )

  // ── Full-mode-only sections (interactive chat) ─────────────────────
  if (isFull) {
    if (channels.length > 1) {
      sections.push(
        '## Channels\n\n' +
        'You can receive and send messages across multiple channels. ' +
        `Use ${quote('lookup_channel_user')} to resolve user/group identities on a specific platform.`
      )
    }

    sections.push(
      '## Attachments\n\n' +
      '### Receive\n\n' +
      'Files user uploaded will added to your workspace, the file path will be included in the message header.\n\n' +
      '### Send\n\n' +
      '**For using channel tools**: Add file path to the message header.\n\n' +
      '**For directly request**: Use the following format:\n\n' +
      block([
        '<attachments>',
        '- /path/to/file.pdf',
        '- /path/to/video.mp4',
        'https://example.com/image.png',
        '</attachments>',
      ].join('\n')) + '\n\n' +
      'External URLs are also supported.\n\n' +
      'Important rules for attachments blocks:\n' +
      `- Only include file paths or URLs (one per line, prefixed by ${quote('- ')})\n` +
      `- Do not include any extra text inside ${quote('<attachments>...</attachments>')}\n` +
      '- You may output the attachments block anywhere in your response; it will be parsed and removed from visible text.'
    )
  }

  // Minimal-mode: brief attachments (micro omits entirely)
  if (!isFull && !isMicro) {
    sections.push(
      `## Attachments\n\n` +
      'Use `<attachments>\\n- /path/to/file\\n</attachments>` blocks to send files.'
    )
  }

  // Skills (full/minimal only, and only when skills exist)
  if (!isMicro && skills.length > 0) {
    const skillList = isFull
      ? skills.map(skill => `- ${skill.name}: ${skill.description}`).join('\n')
      : skills.map(skill => `- ${skill.name}`).join('\n')
    sections.push(
      `## Skills\n\n` +
      `There are ${skills.length} skills available, you can use ${quote('use_skill')} to use a skill.\n` +
      skillList
    )
  }

  // ── Embedded persona files ─────────────────────────────────────────
  if (identityContent) {
    sections.push(`## IDENTITY.md\n\n${identityContent}`)
  }

  if (!isMicro && soulContent) {
    sections.push(`## SOUL.md\n\n${soulContent}`)
  }

  if (!isMicro && toolsContent) {
    sections.push(`## TOOLS.md\n\n${toolsContent}`)
  }

  if (toolContext) {
    sections.push(`## Current Environment\n\n${toolContext}`)
  }

  if (taskContent) {
    sections.push(`## Task\n\n${taskContent}`)
  }

  // Team section: only in full mode
  if (teamContent && isFull) {
    sections.push(`## Team\n\n${teamContent}`)
  }

  if (enabledSkills.length > 0) {
    sections.push(enabledSkills.map(skill => skillPrompt(skill)).join('\n\n---\n\n'))
  }

  // ── Session Context (dynamic, at the end to preserve cache prefix) ─
  if (isMicro) {
    sections.push(`---\n${Bun.YAML.stringify(dynamicHeaders)}---`)
  } else {
    sections.push(
      `## Session Context\n\n` +
      `---\n${Bun.YAML.stringify(dynamicHeaders)}---\n\n` +
      `Your context is loaded from the recent of ${maxContextLoadTime} minutes (${(maxContextLoadTime / 60).toFixed(2)} hours).\n\n` +
      `The current session (and the latest user message) is from channel: ${quote(currentChannel)}. ` +
      `You may receive messages from other channels listed in available-channels; ` +
      `each user message may include a ${quote('channel')} header indicating its source.`
    )
  }

  if (!isMicro && language && language !== 'auto' && language !== 'Same as the user input') {
    sections.push(
      `**⚠ CRITICAL — REPLY LANGUAGE**: You MUST reply in **${language}**. ` +
      `Every single response you produce — including greetings, explanations, tool result summaries, ` +
      `follow-up questions, and conversational text — MUST be written in ${language}. ` +
      `Even if your persona files or the user's message are in a different language, ` +
      `your reply language is always **${language}**. Never switch to another language unless the user explicitly requests it.`
    )
  }

  return sections.join('\n\n')
}
