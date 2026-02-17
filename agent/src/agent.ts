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
} from './types'
import { system, schedule, user, subagentSystem } from './prompts'
import { AuthFetcher } from './index'
import { createModel } from './model'
import { AgentAction } from './types/action'
import { SubagentRegistry } from './registry'
import {
  extractAttachmentsFromText,
  stripAttachmentsFromMessages,
  dedupeAttachments,
  AttachmentsStreamExtractor,
} from './utils/attachments'
import type { ContainerFileAttachment } from './types/attachment'
import { getMCPTools } from './tools/mcp'
import { getTools } from './tools'
import { buildIdentityHeaders } from './utils/headers'
import { normalizeBaseUrl } from './utils/url'
import {
  truncateToolResult,
  sanitizeToolChunkMetadata,
  truncateMessagesForTransport,
  stripReasoningFromMessages,
} from './utils/sse'

export const createAgent = (
  {
    model: modelConfig,
    activeContextTime = 24 * 60,
    language = 'Same as the user input',
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
  }: AgentParams,
  fetch: AuthFetcher,
) => {
  const model = createModel(modelConfig)
  const registry = new SubagentRegistry()
  const enabledSkills: AgentSkill[] = []

  const enableSkill = (skill: string) => {
    const agentSkill = skills.find((s) => s.name === skill)
    if (agentSkill) {
      enabledSkills.push(agentSkill)
    }
  }

  const getEnabledSkills = () => {
    return enabledSkills.map((skill) => skill.name)
  }

  const loadSystemFiles = async () => {
    if (!auth?.bearer || !identity.botId) {
      return {
        identityContent: botIdentity,
        soulContent: botSoul,
        toolsContent: '',
      }
    }
    const readViaMCP = async (path: string): Promise<string> => {
      const url = `${normalizeBaseUrl(auth.baseUrl)}/bots/${identity.botId}/tools`
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'application/json, text/event-stream',
        Authorization: `Bearer ${auth.bearer}`,
      }
      if (identity.channelIdentityId) {
        headers['X-Memoh-Channel-Identity-Id'] = identity.channelIdentityId
      }
      const body = JSON.stringify({
        jsonrpc: '2.0',
        id: `read-${path}`,
        method: 'tools/call',
        params: { name: 'read', arguments: { path } },
      })
      const response = await fetch(url, { method: 'POST', headers, body })
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

    // Use DB values when available, fall back to container MD files via MCP.
    const needIdentity = !botIdentity
    const needSoul = !botSoul
    const needTools = true // TOOLS.md is always read from container

    const mcpReads: Promise<string>[] = [
      needIdentity ? readViaMCP('IDENTITY.md') : Promise.resolve(''),
      needSoul ? readViaMCP('SOUL.md') : Promise.resolve(''),
      needTools ? readViaMCP('TOOLS.md') : Promise.resolve(''),
    ]
    const [mcpIdentity, mcpSoul, toolsContent] = await Promise.all(mcpReads)

    return {
      identityContent: botIdentity || mcpIdentity,
      soulContent: botSoul || mcpSoul,
      toolsContent,
    }
  }

  const generateSystemPrompt = async () => {
    const { identityContent, soulContent, toolsContent } =
      await loadSystemFiles()
    return system({
      date: new Date(),
      language,
      maxContextLoadTime: activeContextTime,
      channels,
      currentChannel,
      skills,
      enabledSkills,
      identityContent,
      soulContent,
      toolsContent,
      taskContent: botTask,
      allowSelfEvolution,
    })
  }

  const getAgentTools = async () => {
    const baseUrl = normalizeBaseUrl(auth.baseUrl)
    const botId = identity.botId.trim()
    if (!baseUrl || !botId) {
      return {
        tools: {},
        close: async () => {},
      }
    }
    const headers = buildIdentityHeaders(identity, auth)
    const builtins: MCPConnection[] = [
      {
        type: 'http',
        name: 'builtin',
        url: `${baseUrl}/bots/${botId}/tools`,
        headers,
      }
    ]
    const { tools: mcpTools, close: closeMCP } = await getMCPTools([...builtins, ...mcpConnections], {
      auth,
      fetch,
      botId,
    })
    const tools = getTools(allowedActions, { fetch, model: modelConfig, identity, auth, enableSkill, mcpConnections, registry })
    return {
      tools: { ...mcpTools, ...tools } as ToolSet,
      close: closeMCP,
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
        if (typeof msgType === 'string' && msgType !== '' && !supportedRoles.has(msgType)) return false
        return true
      })
      .map((msg) => {
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
    const systemPrompt = await generateSystemPrompt()
    const { tools, close } = await getAgentTools()
    const { response, reasoning, text, usage } = await generateText({
      model,
      messages,
      system: systemPrompt,
      stopWhen: stepCountIs(Infinity),
      onFinish: async () => {
        await close()
      },
      tools,
    })
    const { cleanedText, attachments: textAttachments } =
      extractAttachmentsFromText(text)
    const { messages: strippedMessages, attachments: messageAttachments } =
      stripAttachmentsFromMessages(response.messages)
    const cleanedMessages = stripReasoningFromMessages(strippedMessages)
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
      })
    }
    const messages = [...params.messages, userPrompt]
    const { tools, close } = await getAgentTools()
    const { response, reasoning, text, usage } = await generateText({
      model,
      messages,
      system: generateSubagentSystemPrompt(),
      stopWhen: stepCountIs(Infinity),
      onFinish: async () => {
        await close()
      },
      tools,
      abortSignal: params.abortSignal,
    })
    return {
      messages: stripReasoningFromMessages([userPrompt, ...response.messages]),
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
    const scheduleMessage: UserModelMessage = {
      role: 'user',
      content: [
        {
          type: 'text',
          text: schedule({ schedule: params.schedule, date: new Date() }),
        },
      ],
    }
    const messages = [...params.messages, scheduleMessage]
    params.skills.forEach((skill) => enableSkill(skill))
    const { tools, close } = await getAgentTools()
    const { response, reasoning, text, usage } = await generateText({
      model,
      messages,
      system: await generateSystemPrompt(),
      stopWhen: stepCountIs(Infinity),
      onFinish: async () => {
        await close()
      },
      tools,
    })
    return {
      messages: stripReasoningFromMessages([scheduleMessage, ...response.messages]),
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

  async function* stream(input: AgentInput): AsyncGenerator<AgentAction> {
    const userPrompt = generateUserPrompt(input)
    const messages = [...sanitizeMessages(input.messages), userPrompt]
    input.skills.forEach((skill) => enableSkill(skill))
    const systemPrompt = await generateSystemPrompt()
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
    const { tools, close } = await getAgentTools()
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
