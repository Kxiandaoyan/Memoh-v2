import {
  generateText,
  ImagePart,
  LanguageModelUsage,
  ModelMessage,
  stepCountIs,
  streamText,
  ToolSet,
  UserModelMessage,
} from 'ai'
import {
  AgentInput,
  AgentParams,
  AgentSkill,
  allActions,
  MCPConnection,
  Schedule,
  SYSTEM_SAFE_PROVIDERS,
} from './types'
import { system, schedule, heartbeat, user, subagentSystem } from './prompts'
import { AuthFetcher } from './index'
import { createModel, getProviderOptions } from './model'
import { AgentAction } from './types/action'
import { SubagentRegistry } from './registry'
import {
  extractAttachmentsFromText,
  stripAttachmentsFromMessages,
  dedupeAttachments,
  AttachmentsStreamExtractor,
} from './utils/attachments'
import type { ContainerFileAttachment } from './types/attachment'
import { withRetry, isRetryableLLMError } from './utils/retry'
import { getMCPTools } from './tools/mcp'
import { getTools } from './tools'
import { wrapToolsWithLoopDetection, clearLoopDetectionState } from './tools/loop-detection'
import { buildIdentityHeaders } from './utils/headers'
import { createTierTools, getEnabledExtendedTools } from './tools/tier'
import { normalizeBaseUrl } from './utils/url'
import {
  truncateToolResult,
  sanitizeToolChunkMetadata,
  truncateMessagesForTransport,
  stripReasoningFromMessages,
  computeMaxToolResultChars,
} from './utils/sse'
import type { SystemMode } from './prompts/system'

const HEAD_RATIO = 0.7
const TAIL_RATIO = 0.2

const CHAR_BUDGETS: Record<SystemMode, { soul: number; tools: number }> = {
  full:    { soul: 3000, tools: 3000 },
  minimal: { soul: 800,  tools: 800  },
  micro:   { soul: 0,    tools: 0    },
}

const FILE_SIZE_WARN_THRESHOLD = 8000

function truncateHeadTail(content: string, maxChars: number): string {
  if (maxChars <= 0 || !content) return ''
  if (content.length <= maxChars) return content
  const headChars = Math.floor(maxChars * HEAD_RATIO)
  const tailChars = Math.floor(maxChars * TAIL_RATIO)
  const head = content.slice(0, headChars)
  const tail = content.slice(-tailChars)
  return `${head}\n\n[...truncated — read the full file with the \`read\` tool for complete content...]\n\n${tail}`
}

export const createAgent = (
  {
    model: modelConfig,
    backgroundModel: backgroundModelConfig,
    activeContextTime = 24 * 60,
    language = 'Same as the user input',
    timezone = 'UTC',
    allowedActions = allActions,
    channels = [],
    skills = [],
    mcpConnections = [],
    currentChannel = 'Unknown Channel',
    identity = {
      botId: '',
      containerId: '',
      channelIdentityId: '',
      displayName: '',
    },
    auth,
    botIdentity = '',
    botSoul = '',
    botTask = '',
    allowSelfEvolution = true,
    botTeam = '',
    teamMembers = [] as string[],
    callDepth = 0,
  }: AgentParams,
  fetch: AuthFetcher,
) => {
  const model = createModel(modelConfig)
  const providerOptions = getProviderOptions(modelConfig)
  const registry = new SubagentRegistry()
  const enabledSkills: AgentSkill[] = []

  const enableSkill = (skill: string): { content: string; description: string } | null => {
    const agentSkill = skills.find((s) => s.name === skill)
    if (!agentSkill) return null
    if (!enabledSkills.some((s) => s.name === skill)) {
      enabledSkills.push(agentSkill)
    }
    return { content: agentSkill.content, description: agentSkill.description }
  }

  const getEnabledSkills = () => {
    return enabledSkills.map((skill) => skill.name)
  }

  const TOOL_CATEGORIES: Array<{ label: string; tools: string[]; desc: string }> = [
    { label: 'File', tools: ['read', 'write', 'list', 'edit'], desc: '/data/ (private), /shared/ (cross-bot)' },
    { label: 'Shell', tools: ['exec'], desc: 'run commands in container' },
    { label: 'Web', tools: ['web_search', 'web_fetch'], desc: 'search & fetch web content' },
    { label: 'Memory', tools: ['search_memory', 'query_history'], desc: 'search memories & conversation history' },
    { label: 'Knowledge', tools: ['knowledge_read', 'knowledge_write'], desc: 'read & write bot knowledge base' },
    { label: 'Message', tools: ['send', 'react', 'lookup_channel_user'], desc: 'send messages, reactions & user lookup' },
    { label: 'Image', tools: ['generate_image'], desc: 'generate image from text prompt (async, auto-delivered)' },
    { label: 'Schedule', tools: ['create_schedule', 'list_schedule', 'get_schedule', 'update_schedule', 'delete_schedule'], desc: 'manage cron-based recurring tasks' },
    { label: 'Skills', tools: ['use_skill', 'discover_skills', 'fork_skill'], desc: 'activate, search & import skills' },
    { label: 'Subagent', tools: ['list_subagents', 'create_subagent', 'delete_subagent', 'query_subagent', 'spawn_subagent', 'check_subagent_run', 'kill_subagent_run', 'steer_subagent', 'list_subagent_runs'], desc: 'create & manage sub-agents. ONLY use spawn_subagent when 2+ independent long-running tasks need parallel execution. For simple questions, single-step tasks, or sequential work — do it yourself, never spawn.' },
    { label: 'OpenViking', tools: ['ov_initialize', 'ov_find', 'ov_search', 'ov_read', 'ov_abstract', 'ov_overview', 'ov_ls', 'ov_tree', 'ov_add_resource', 'ov_rm', 'ov_session_commit'], desc: 'context database (see TOOLS.md for details)' },
  ]

  const generateToolContext = (toolNames: string[] = []): string => {
    const sections: string[] = []
    const externalMcps = mcpConnections.filter(c => c.name !== 'builtin')
    if (externalMcps.length > 0) {
      sections.push(
        '### External Tools (MCP)\n' +
        externalMcps.map(c => `- **${c.name}** (${c.type})`).join('\n'),
      )
    }

    const has = (name: string) => toolNames.includes(name)
    const lines: string[] = []
    const categorized = new Set<string>()

    for (const cat of TOOL_CATEGORIES) {
      const present = cat.tools.filter(has)
      if (present.length === 0) continue
      present.forEach(t => categorized.add(t))
      lines.push(`- ${cat.label}: ${present.map(t => `\`${t}\``).join(', ')} — ${cat.desc}`)
    }

    const uncategorized = toolNames.filter(n => !categorized.has(n))
    for (const name of uncategorized) {
      lines.push(`- \`${name}\``)
    }

    if (lines.length > 0) {
      sections.push(
        '### Available Tools\n' +
        lines.join('\n') +
        '\n\nCLI tools (use via `exec`): `agent-browser`, `clawhub`, `actionbook`' +
        '\n\nFor detailed documentation, read /data/TOOLS.md',
      )
    }

    return sections.join('\n\n')
  }

  const systemFileCache: {
    key: string
    data: { identityContent: string; soulContent: string; toolsContent: string }
    expiry: number
  } | null = { key: '', data: { identityContent: '', soulContent: '', toolsContent: '' }, expiry: 0 }

  const SYSTEM_FILE_CACHE_TTL_MS = 60_000

  const loadSystemFiles = async () => {
    if (!auth?.bearer || !identity.botId) {
      return {
        identityContent: botIdentity,
        soulContent: botSoul,
        toolsContent: '',
      }
    }

    const cacheKey = `${identity.botId}:${botIdentity}:${botSoul}`
    if (systemFileCache && systemFileCache.key === cacheKey && Date.now() < systemFileCache.expiry) {
      return systemFileCache.data
    }

    const mcpHeaders = (): Record<string, string> => {
      const h: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'application/json, text/event-stream',
        Authorization: `Bearer ${auth.bearer}`,
      }
      if (identity.channelIdentityId) {
        h['X-Memoh-Channel-Identity-Id'] = identity.channelIdentityId
      }
      return h
    }

    const mcpToolsURL = `${normalizeBaseUrl(auth.baseUrl)}/bots/${identity.botId}/tools`

    const readViaMCP = async (path: string): Promise<string> => {
      const body = JSON.stringify({
        jsonrpc: '2.0',
        id: `read-${path}`,
        method: 'tools/call',
        params: { name: 'read', arguments: { path } },
      })
      const response = await fetch(mcpToolsURL, { method: 'POST', headers: mcpHeaders(), body })
      if (!response.ok) return ''
      const data = await response.json().catch(() => ({}))
      const structured =
        data?.result?.structuredContent ?? data?.result?.content?.[0]?.text
      if (typeof structured === 'string') {
        try {
          const parsed = JSON.parse(structured)
          return typeof parsed?.content === 'string' ? parsed.content : ''
        } catch {
          return structured
        }
      }
      if (typeof structured === 'object' && structured?.content) {
        return typeof structured.content === 'string' ? structured.content : ''
      }
      return ''
    }

    // Async restore: write DB persona content back to container if the file is
    // missing/empty. Fires in the background; failures are silently ignored so
    // they never block the current request.
    const restoreViaMCP = (path: string, content: string): void => {
      if (!content.trim()) return
      const body = JSON.stringify({
        jsonrpc: '2.0',
        id: `write-${path}`,
        method: 'tools/call',
        params: { name: 'write', arguments: { path, content } },
      })
      fetch(mcpToolsURL, { method: 'POST', headers: mcpHeaders(), body }).catch(() => {})
    }

    const needIdentity = !botIdentity
    const needSoul = !botSoul
    const needTools = true

    const mcpReads: Promise<string>[] = [
      needIdentity ? readViaMCP('IDENTITY.md') : Promise.resolve(''),
      needSoul ? readViaMCP('SOUL.md') : Promise.resolve(''),
      needTools ? readViaMCP('TOOLS.md') : Promise.resolve(''),
    ]
    const [mcpIdentity, mcpSoul, toolsContent] = await Promise.all(mcpReads)

    // Self-healing: if DB has persona content but container file is empty,
    // asynchronously restore the file so evolution can read it next time.
    if (botIdentity && !mcpIdentity.trim()) restoreViaMCP('IDENTITY.md', botIdentity)
    if (botSoul && !mcpSoul.trim()) restoreViaMCP('SOUL.md', botSoul)

    const result = {
      identityContent: botIdentity || mcpIdentity,
      soulContent: botSoul || mcpSoul,
      toolsContent,
    }

    if (result.soulContent.length > FILE_SIZE_WARN_THRESHOLD) {
      console.warn(`[${identity.botId}] SOUL.md is ${result.soulContent.length} chars — consider distilling to reduce token consumption`)
    }
    if (result.toolsContent.length > FILE_SIZE_WARN_THRESHOLD) {
      console.warn(`[${identity.botId}] TOOLS.md is ${result.toolsContent.length} chars — consider distilling to reduce token consumption`)
    }

    Object.assign(systemFileCache!, { key: cacheKey, data: result, expiry: Date.now() + SYSTEM_FILE_CACHE_TTL_MS })

    return result
  }

  const generateSystemPrompt = async (mode: SystemMode = 'full', toolNames: string[] = []) => {
    const { identityContent, soulContent, toolsContent } =
      await loadSystemFiles()
    const budget = CHAR_BUDGETS[mode]
    return system({
      date: new Date(),
      language,
      timezone,
      maxContextLoadTime: activeContextTime,
      channels,
      currentChannel,
      skills,
      enabledSkills,
      identityContent,
      soulContent: truncateHeadTail(soulContent, budget.soul),
      toolsContent: truncateHeadTail(toolsContent, budget.tools),
      toolContext: generateToolContext(toolNames),
      taskContent: botTask,
      allowSelfEvolution,
      teamContent: botTeam || undefined,
      mode,
    })
  }

  // Cache builtin MCP tool definitions to avoid repeated HTTP round-trips.
  const mcpToolCache: {
    tools: ToolSet
    close: () => Promise<void>
    extKey: string
    expiry: number
  } = { tools: {}, close: async () => {}, extKey: '', expiry: 0 }

  const getAgentTools = async (sessionId?: string) => {
    const baseUrl = normalizeBaseUrl(auth.baseUrl)
    const botId = identity.botId.trim()
    if (!baseUrl || !botId) {
      return {
        tools: {},
        toolNames: [] as string[],
        close: async () => {},
      }
    }
    const baseHeaders = buildIdentityHeaders(identity, auth)
    const enabledExt = getEnabledExtendedTools()
    const extKey = enabledExt.join(',')
    const builtinHeaders = enabledExt.length > 0
      ? { ...baseHeaders, 'X-Memoh-Include-Tools': extKey }
      : baseHeaders

    // Use cached builtin tools when available; always connect external MCP fresh.
    let mcpTools: ToolSet
    let closeMCP: () => Promise<void>

    const cacheHit = mcpToolCache.expiry > Date.now()
      && mcpToolCache.extKey === extKey
      && Object.keys(mcpToolCache.tools).length > 0

    if (!cacheHit) {
      await mcpToolCache.close().catch(() => {})
      const builtinConn: MCPConnection = {
        type: 'http', name: 'builtin',
        url: `${baseUrl}/bots/${botId}/tools`,
        headers: builtinHeaders,
      }
      const res = await getMCPTools([builtinConn], { auth, fetch, botId })
      mcpToolCache.tools = res.tools
      mcpToolCache.close = res.close
      mcpToolCache.extKey = extKey
      mcpToolCache.expiry = Date.now() + SYSTEM_FILE_CACHE_TTL_MS
    }

    if (mcpConnections.length > 0) {
      const ext = await getMCPTools(mcpConnections, { auth, fetch, botId })
      mcpTools = { ...mcpToolCache.tools, ...ext.tools }
      closeMCP = ext.close
    } else {
      mcpTools = mcpToolCache.tools
      closeMCP = async () => {}
    }
    const tools = getTools(allowedActions, { fetch, model: modelConfig, backgroundModel: backgroundModelConfig, identity, auth, enableSkill, mcpConnections, registry, teamMembers, callDepth })
    const tierTools = createTierTools({ auth, identity, fetch })
    const merged = { ...tierTools, ...mcpTools, ...tools } as ToolSet
    const toolNames = Object.keys(merged)
    const wrappedTools = sessionId ? wrapToolsWithLoopDetection(merged, sessionId) : merged
    return {
      tools: wrappedTools,
      toolNames,
      close: async () => {
        await closeMCP()
        if (sessionId) clearLoopDetectionState(sessionId)
      },
    }
  }

  const generateUserPrompt = (input: AgentInput) => {
    const images = input.attachments.filter(
      (attachment) => attachment.type === 'image',
    )
    const files = input.attachments.filter(
      (a): a is ContainerFileAttachment => a.type === 'file',
    )
    const text = user(input.query, {
      channelIdentityId: identity.channelIdentityId || identity.contactId || '',
      displayName: identity.displayName || identity.contactName || 'User',
      channel: currentChannel,
      conversationType: identity.conversationType || 'direct',
      date: new Date(),
      attachments: files,
    })
    const userMessage: UserModelMessage = {
      role: 'user',
      content: [
        { type: 'text', text },
        ...images.map(
          (image) => ({ type: 'image', image: image.base64 }) as ImagePart,
        ),
      ],
    }
    return userMessage
  }

  const sanitizeMessages = (messages: ModelMessage[]): ModelMessage[] => {
    const supportedRoles = new Set(['user', 'assistant', 'system', 'tool'])
    const supportedTypes = new Set(['text', 'image', 'file', 'tool-call', 'tool-result', 'reasoning'])
    return messages
      .filter((msg) => {
        // Drop messages with unsupported roles (e.g. item_reference from Responses API).
        if (!msg || typeof msg !== 'object') return false
        const role = (msg as Record<string, unknown>).role
        if (typeof role !== 'string' || !supportedRoles.has(role)) return false
        // Drop messages that have a non-standard "type" field at the top level.
        const msgType = (msg as Record<string, unknown>).type
        if (typeof msgType === 'string' && msgType !== '' && !supportedTypes.has(msgType)) return false
        return true
      })
      .map((msg) => {
        const role = (msg as Record<string, unknown>).role as string
        if (role === 'system' && !SYSTEM_SAFE_PROVIDERS.has(modelConfig.clientType)) {
          return { ...msg, role: 'user' } as ModelMessage
        }
        if (!Array.isArray(msg.content)) return msg
        const original = msg.content as Array<Record<string, unknown>>
        const filtered = original.filter((part) => {
          if (!part || typeof part !== 'object') return true
          const t = part.type
          if (!t || typeof t !== 'string') return true
          return supportedTypes.has(t)
        })
        if (filtered.length === original.length) return msg
        if (filtered.length === 0) {
          return { ...msg, content: [{ type: 'text', text: '' }] } as ModelMessage
        }
        return { ...msg, content: filtered } as ModelMessage
      })
  }

  // Normalize AI SDK v6 usage fields to the legacy names expected by the
  // Go backend (gatewayUsage) and the web frontend (promptTokens, etc.).
  const normalizeUsage = (usage: LanguageModelUsage | null) => {
    if (!usage) return { promptTokens: 0, completionTokens: 0, totalTokens: 0 }
    const input = (usage as Record<string, unknown>).inputTokens as number | undefined
    const output = (usage as Record<string, unknown>).outputTokens as number | undefined
    const prompt = (usage as Record<string, unknown>).promptTokens as number | undefined
    const completion = (usage as Record<string, unknown>).completionTokens as number | undefined
    const p = prompt ?? input ?? 0
    const c = completion ?? output ?? 0
    return {
      promptTokens: p,
      completionTokens: c,
      totalTokens: usage.totalTokens ?? (p + c),
    }
  }

  const ask = async (input: AgentInput) => {
    const userPrompt = generateUserPrompt(input)
    const messages = [...sanitizeMessages(input.messages), userPrompt]
    input.skills.forEach((skill) => enableSkill(skill))
    const sessionId = `ask:${identity.botId}:${Date.now()}`
    const { tools, toolNames, close } = await getAgentTools(sessionId)
    const systemPrompt = await generateSystemPrompt('full', toolNames)
    const { response, reasoning, text, usage } = await withRetry(
      () =>
        generateText({
          model,
          messages,
          system: systemPrompt,
          stopWhen: stepCountIs(Infinity),
          onFinish: async () => {
            await close()
          },
          tools,
          providerOptions,
        }),
      isRetryableLLMError,
    )
    const { cleanedText, attachments: textAttachments } =
      extractAttachmentsFromText(text)
    const { messages: strippedMessages, attachments: messageAttachments } =
      stripAttachmentsFromMessages(response.messages)
    const cleanedMessages = stripReasoningFromMessages(
      truncateMessagesForTransport(strippedMessages),
    )
    const allAttachments = dedupeAttachments([
      ...textAttachments,
      ...messageAttachments,
    ])
    return {
      messages: cleanedMessages,
      reasoning: reasoning.map((part) => part.text),
      usage: normalizeUsage(usage),
      text: cleanedText,
      attachments: allAttachments,
      skills: getEnabledSkills(),
    }
  }

  const askAsSubagent = async (params: {
    input: string;
    name: string;
    description: string;
    messages: ModelMessage[];
    abortSignal?: AbortSignal;
  }) => {
    const userPrompt: UserModelMessage = {
      role: 'user',
      content: [{ type: 'text', text: params.input }],
    }
    const generateSubagentSystemPrompt = () => {
      return subagentSystem({
        date: new Date(),
        name: params.name,
        description: params.description,
        timezone,
      })
    }
    const messages = [...sanitizeMessages(params.messages), userPrompt]
    const sessionId = `subagent:${identity.botId}:${params.name}:${Date.now()}`
    const { tools, toolNames: _subToolNames, close } = await getAgentTools(sessionId)
    const { response, reasoning, text, usage } = await withRetry(
      () =>
        generateText({
          model,
          messages,
          system: generateSubagentSystemPrompt(),
          stopWhen: stepCountIs(Infinity),
          onFinish: async () => {
            await close()
          },
          tools,
          abortSignal: params.abortSignal,
          providerOptions,
        }),
      isRetryableLLMError,
    )
    return {
      messages: stripReasoningFromMessages(
        truncateMessagesForTransport([userPrompt, ...response.messages]),
      ),
      reasoning: reasoning.map((part) => part.text),
      usage: normalizeUsage(usage),
      text,
      skills: getEnabledSkills(),
    }
  }

  const triggerSchedule = async (params: {
    schedule: Schedule;
    messages: ModelMessage[];
    skills: string[];
  }) => {
    const isHeartbeat = params.schedule.triggerType === 'heartbeat'
    const promptText = isHeartbeat
      ? heartbeat({ schedule: params.schedule, date: new Date(), timezone })
      : schedule({ schedule: params.schedule, date: new Date(), timezone })
    const scheduleMessage: UserModelMessage = {
      role: 'user',
      content: [
        {
          type: 'text',
          text: promptText,
        },
      ],
    }
    const messages = [...sanitizeMessages(params.messages), scheduleMessage]
    params.skills.forEach((skill) => enableSkill(skill))
    const sessionId = `schedule:${identity.botId}:${params.schedule.id}:${Date.now()}`
    const { tools, toolNames: schedToolNames, close } = await getAgentTools(sessionId)
    const systemPromptText = await generateSystemPrompt(isHeartbeat ? 'micro' : 'minimal', schedToolNames)
    const { response, reasoning, text, usage } = await withRetry(
      () =>
        generateText({
          model,
          messages,
          system: systemPromptText,
          stopWhen: stepCountIs(Infinity),
          onFinish: async () => {
            await close()
          },
          tools,
          providerOptions,
        }),
      isRetryableLLMError,
    )
    return {
      messages: stripReasoningFromMessages(
        truncateMessagesForTransport([scheduleMessage, ...response.messages]),
      ),
      reasoning: reasoning.map((part) => part.text),
      usage: normalizeUsage(usage),
      text,
      skills: getEnabledSkills(),
    }
  }

  const resolveStreamErrorMessage = (raw: unknown): string => {
    if (raw instanceof Error && raw.message.trim()) {
      return raw.message
    }
    if (typeof raw === 'string' && raw.trim()) {
      return raw
    }
    if (raw && typeof raw === 'object') {
      const candidate = raw as { message?: unknown; error?: unknown }
      if (typeof candidate.message === 'string' && candidate.message.trim()) {
        return candidate.message
      }
      if (typeof candidate.error === 'string' && candidate.error.trim()) {
        return candidate.error
      }
      if (candidate.error instanceof Error && candidate.error.message.trim()) {
        return candidate.error.message
      }
    }
    return 'Model stream failed'
  }

  // -- Write-tool attachment helpers --
  const FILE_WRITE_TOOLS = new Set(['write', 'save_file', 'create_file', 'write_file'])
  const isDeliverableWrite = (p: string): boolean => {
    if (!p.startsWith('/shared/')) return false
    const dot = p.lastIndexOf('.')
    return dot !== -1 && dot > p.lastIndexOf('/')
  }

  const extractWritePath = (input: unknown): string | null => {
    if (!input || typeof input !== 'object') return null
    const p = (input as Record<string, unknown>).path
    return typeof p === 'string' && p.length > 0 ? p : null
  }

  const isWriteSuccess = (output: unknown): boolean => {
    if (output === undefined || output === null) return false
    if (typeof output === 'string') return !output.toLowerCase().includes('error')
    if (typeof output === 'object') {
      const o = output as Record<string, unknown>
      if (o.isError === true) return false
      const content = o.content
      if (Array.isArray(content)) {
        return !content.some((c: any) => c?.type === 'text' && typeof c.text === 'string' && c.text.toLowerCase().includes('error'))
      }
    }
    return true
  }

  async function* stream(input: AgentInput): AsyncGenerator<AgentAction> {
    const userPrompt = generateUserPrompt(input)
    const messages = [...sanitizeMessages(input.messages), userPrompt]
    input.skills.forEach((skill) => enableSkill(skill))
    const sessionId = `stream:${identity.botId}:${Date.now()}`
    const { tools, toolNames: streamToolNames, close } = await getAgentTools(sessionId)
    const systemPrompt = await generateSystemPrompt('full', streamToolNames)
    const attachmentsExtractor = new AttachmentsStreamExtractor()
    const result: {
      messages: ModelMessage[];
      reasoning: string[];
      usage: LanguageModelUsage | null;
    } = {
      messages: [],
      reasoning: [],
      usage: null,
    }
    let closeCalled = false
    const safeClose = async () => {
      if (!closeCalled) {
        closeCalled = true
        await close()
      }
    }
    const { fullStream } = streamText({
      model,
      messages,
      system: systemPrompt,
      stopWhen: stepCountIs(Infinity),
      tools,
      providerOptions,
      onFinish: async ({ usage, reasoning, response }) => {
        await safeClose()
        result.usage = usage as never
        result.reasoning = reasoning.map((part) => part.text)
        result.messages = response.messages
      },
    })
    yield {
      type: 'agent_start',
      input,
    }
    try {
      for await (const chunk of fullStream) {
        if (chunk.type === 'error') {
          throw new Error(
            resolveStreamErrorMessage((chunk as { error?: unknown }).error),
          )
        }
        switch (chunk.type) {
          case 'reasoning-start':
            yield {
              type: 'reasoning_start',
              metadata: chunk,
            }
            break
          case 'reasoning-delta':
            yield {
              type: 'reasoning_delta',
              delta: chunk.text,
            }
            break
          case 'reasoning-end':
            yield {
              type: 'reasoning_end',
              metadata: chunk,
            }
            break
          case 'text-start':
            yield {
              type: 'text_start',
            }
            break
          case 'text-delta': {
            const { visibleText, attachments } = attachmentsExtractor.push(
              chunk.text,
            )
            if (visibleText) {
              yield {
                type: 'text_delta',
                delta: visibleText,
              }
            }
            if (attachments.length) {
              yield {
                type: 'attachment_delta',
                attachments,
              }
            }
            break
          }
          case 'text-end': {
            const remainder = attachmentsExtractor.flushRemainder()
            if (remainder.visibleText) {
              yield {
                type: 'text_delta',
                delta: remainder.visibleText,
              }
            }
            if (remainder.attachments.length) {
              yield {
                type: 'attachment_delta',
                attachments: remainder.attachments,
              }
            }
            yield {
              type: 'text_end',
              metadata: chunk,
            }
            break
          }
          case 'tool-call':
            yield {
              type: 'tool_call_start',
              toolName: chunk.toolName,
              toolCallId: chunk.toolCallId,
              input: chunk.input,
              metadata: chunk,
            }
            break
          case 'tool-result':
            yield {
              type: 'tool_call_end',
              toolName: chunk.toolName,
              toolCallId: chunk.toolCallId,
              input: chunk.input,
              result: truncateToolResult(chunk.output),
              metadata: sanitizeToolChunkMetadata(
                chunk as unknown as Record<string, unknown>,
              ),
            }
            // Auto-emit attachment for write tool so frontend doesn't depend on LLM <attachments> tag
            if (FILE_WRITE_TOOLS.has(chunk.toolName) && isWriteSuccess(chunk.output)) {
              const writePath = extractWritePath(chunk.input)
              if (writePath && isDeliverableWrite(writePath)) {
                yield { type: 'attachment_delta', attachments: [{ type: 'file', path: writePath }] }
              }
            }
            break
          case 'file':
            yield {
              type: 'image_delta',
              image: chunk.file.base64,
              metadata: chunk,
            }
        }
      }
    } finally {
      await safeClose()
    }

    const { messages: strippedMessages } = stripAttachmentsFromMessages(
      result.messages,
    )
    const cleanedMessages = stripReasoningFromMessages(
      truncateMessagesForTransport(strippedMessages),
    ) as ModelMessage[]
    yield {
      type: 'agent_end',
      messages: cleanedMessages,
      reasoning: result.reasoning,
      usage: normalizeUsage(result.usage),
      skills: getEnabledSkills(),
    }
  }

  return {
    stream,
    ask,
    askAsSubagent,
    triggerSchedule,
  }
}
