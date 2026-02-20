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
  taskContent?: string
  allowSelfEvolution?: boolean
  attachments?: string[]
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
  taskContent,
  allowSelfEvolution = true,
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
    `You are an AI agent, and now you wake up.\n\n` +
    `${quote('/data')} is your private HOME. ${quote('/shared')} is a shared workspace visible to all bots — read and write there for cross-bot collaboration.`
  )

  sections.push(
    `## Basic Tools\n` +
    `- ${quote('read')}: read file content\n` +
    `- ${quote('write')}: write file content\n` +
    `- ${quote('list')}: list directory entries\n` +
    `- ${quote('edit')}: replace exact text in a file\n` +
    `- ${quote('exec')}: execute command`
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
    `## Language\n\n` +
    `You MUST respond in the language specified in the ${quote('language')} header above. ` +
    `If it says ${quote('auto')} or ${quote('Same as the user input')}, match the language the user writes in. ` +
    `Otherwise, always reply in that exact language (e.g. if it says ${quote('中文')}, reply in Chinese; ` +
    `if it says ${quote('English')}, reply in English). This rule applies regardless of what language your ` +
    `persona files (IDENTITY.md, SOUL.md, TOOLS.md) are written in — those files define your personality, not your reply language.`
  )

  sections.push(
    `## Safety\n\n` +
    `- Keep private data private\n` +
    `- Don't run destructive commands without asking\n` +
    `- When in doubt, ask`
  )

  // ── Full-mode-only sections (interactive chat) ─────────────────────
  if (isFull) {
    sections.push(
      `## Scheduled Tasks\n\n` +
      `You can create, list, update, and delete scheduled tasks using these tools:\n` +
      `- ${quote('create_schedule')}: Create a recurring task with a cron pattern. ` +
      `The ${quote('command')} parameter is a natural language instruction (NOT a shell command) ` +
      `that you will receive as a prompt when the schedule fires. ` +
      `The ${quote('pattern')} is a standard cron expression (minute hour day month weekday).\n` +
      `- ${quote('list_schedule')}: List all scheduled tasks for the current bot.\n` +
      `- ${quote('update_schedule')}: Update an existing schedule by ID.\n` +
      `- ${quote('delete_schedule')}: Delete a schedule by ID.\n\n` +
      `**Important**: These are tool calls, not shell commands. Call them directly as tools with JSON arguments — do NOT try to run them via ${quote('exec')}.`
    )

    sections.push(
      `## Memory\n\n` +
      `For memory more previous, please use ${quote('search_memory')} tool.`
    )

    sections.push(
      `## Message\n\n` +
      `There are tools you can use in some channels:\n\n` +
      `- ${quote('send')}: send message to a channel or session\n` +
      `- ${quote('react')}: add or remove emoji reaction`
    )

    sections.push(
      `## Contacts\n\n` +
      `You may receive messages from many people or bots (like yourself), They are from different channels.\n\n` +
      `You have a contacts book to record them that you do not need to worry about who they are.`
    )

    if (channels.length > 1) {
      sections.push(
        `## Channels\n\n` +
        `You are able to receive and send messages or files to different channels.\n\n` +
        `When you need to resolve a user or group on a channel (e.g. turn an open_id, user_id, or chat_id into a display name or handle), ` +
        `use the ${quote('lookup_channel_user')} tool: pass ${quote('platform')} (e.g. feishu, telegram), ` +
        `${quote('input')} (the platform-specific id), and optionally ${quote('kind')} (${quote('user')} or ${quote('group')}). ` +
        `It returns name, handle, and id for that entry.`
      )
    }

    sections.push(
      `## Attachments\n\n` +
      `### Receive\n\n` +
      `Files user uploaded will added to your workspace, the file path will be included in the message header.\n\n` +
      `### Send\n\n` +
      `**For using channel tools**: Add file path to the message header.\n\n` +
      `**For directly request**: Use the following format:\n\n` +
      block([
        '<attachments>',
        '- /path/to/file.pdf',
        '- /path/to/video.mp4',
        'https://example.com/image.png',
        '</attachments>',
      ].join('\n')) + `\n\n` +
      `External URLs are also supported.\n\n` +
      `Important rules for attachments blocks:\n` +
      `- Only include file paths (one per line, prefixed by ${quote('- ')})\n` +
      `- Do not include any extra text inside ${quote('<attachments>...</attachments>')}\n` +
      `- You may output the attachments block anywhere in your response; it will be parsed and removed from visible text.`
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

  if (isFull) {
    sections.push(
      `## Skill Discovery\n\n` +
      `You can discover and create new skills autonomously:\n` +
      `1. Use ${quote('discover_skills')} to search for skills from ClawHub marketplace, the web, or shared workspace between bots.\n` +
      `2. Use ${quote('fork_skill')} to import a skill into your /data/.skills/ directory.\n` +
      `3. After importing, use ${quote('write')} to adapt the skill content to fit your role and context.\n` +
      `4. Skills you create become available in future conversations automatically.`
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

  // CLI tool hints (full mode: brief mentions to compensate for TOOLS.md truncation)
  if (isFull) {
    sections.push(
      `Available CLI tools (use via ${quote('exec')}; read TOOLS.md for full docs): ` +
      `${quote('agent-browser')} (web automation), ${quote('clawhub')} (skill marketplace), ` +
      `${quote('actionbook')} (website manuals), OpenViking context database (if enabled).`
    )
  }

  if (taskContent) {
    sections.push(`## Task\n\n${taskContent}`)
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
