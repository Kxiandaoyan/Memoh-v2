import { defineStore } from 'pinia'
import { computed, reactive, ref, watch } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import { useUserStore } from '@/store/user'
import {
  createChat,
  deleteChat as requestDeleteChat,
  type Bot,
  type ChatSummary,
  type Message,
  type StreamEvent,
  fetchBots,
  fetchMessages,
  fetchChats,
  extractMessageText,
  extractToolCalls,
  extractAllToolResults,
  streamMessage,
  streamMessageEvents,
} from '@/composables/api/useChat'
import i18n from '@/i18n'

const BILLING_ERRORS: Record<string, string> = {
  daily_limit_exceeded: 'chat.errors.dailyLimitExceeded',
  insufficient_balance: 'chat.errors.insufficientBalance',
}

function resolveBillingError(msg: string): string {
  const key = BILLING_ERRORS[msg]
  if (key) return i18n.global.t(key) as string
  return msg
}

// ---- Message model (blocks-based, aligned with main branch) ----

export interface TextBlock {
  type: 'text'
  content: string
}

export interface ThinkingBlock {
  type: 'thinking'
  content: string
  done: boolean
}

export interface ToolCallBlock {
  type: 'tool_call'
  toolName: string
  input: unknown
  result: unknown | null
  done: boolean
}

export interface ImageBlock {
  type: 'image'
  src: string
}

export interface AttachmentBlock {
  type: 'attachment'
  attachments: unknown[]
}

export interface SubagentProgressBlock {
  type: 'subagent_progress'
  runs: Array<{
    runId: string
    name: string
    task: string
    status: string
    elapsed_ms: number
    delta?: string
    completedAt?: number
  }>
}

export type ContentBlock = TextBlock | ThinkingBlock | ToolCallBlock | ImageBlock | AttachmentBlock | SubagentProgressBlock

export interface TokenUsage {
  promptTokens: number
  completionTokens: number
  totalTokens: number
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  blocks: ContentBlock[]
  timestamp: Date
  streaming: boolean
  isWaiting?: boolean
  platform?: string
  senderDisplayName?: string
  senderAvatarUrl?: string
  isSelf?: boolean
  tokenUsage?: TokenUsage
}

// ---- Store ----

export const useChatStore = defineStore('chat', () => {
  const messages = reactive<ChatMessage[]>([])
  const streaming = ref(false)
  const waitingForResponse = ref(false)
  const chats = ref<ChatSummary[]>([])
  const loading = ref(false)
  const loadingChats = ref(false)
  const loadingOlder = ref(false)
  const hasMoreOlder = ref(true)
  const initializing = ref(false)
  const currentBotId = useLocalStorage<string | null>('chat-bot-id', null)
  const chatId = useLocalStorage<string | null>('chat-id', null)
  const bots = ref<Bot[]>([])

  let abortFn: (() => void) | null = null
  let waitingTimeout: ReturnType<typeof setTimeout> | null = null
  let messageEventsController: AbortController | null = null
  let messageEventsLoopVersion = 0
  let messageEventsSince = ''

  const participantChats = computed(() =>
    chats.value.filter((c) => (c.access_mode ?? 'participant') === 'participant'),
  )
  const observedChats = computed(() =>
    chats.value.filter((c) => c.access_mode === 'channel_identity_observed'),
  )
  const activeChat = computed(() =>
    chats.value.find((c) => c.id === chatId.value) ?? null,
  )
  const activeChatReadOnly = computed(() =>
    activeChat.value?.access_mode === 'channel_identity_observed',
  )

  watch(currentBotId, (newId) => {
    if (newId) {
      if (!initializing.value) {
        void initialize()
      }
    } else {
      stopMessageEvents()
      messageEventsSince = ''
      chats.value = []
      chatId.value = null
      replaceMessages([])
    }
  })

  const nextId = () => `${Date.now()}-${Math.floor(Math.random() * 1000)}`

  const isPendingBot = (bot: Bot | null | undefined) =>
    bot?.status === 'creating' || bot?.status === 'deleting'

  const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms))

  // ---- Message adapter: convert server Message to ChatMessage ----

  function extractTokenUsageFromMetadata(raw: Message): TokenUsage | undefined {
    const meta = raw.metadata
    if (!meta || typeof meta !== 'object') return undefined
    const tu = meta.token_usage
    if (!tu || typeof tu !== 'object') return undefined
    const u = tu as Record<string, unknown>
    const total = Number(u.total_tokens ?? 0)
    if (total <= 0) return undefined
    return {
      promptTokens: Number(u.prompt_tokens ?? 0),
      completionTokens: Number(u.completion_tokens ?? 0),
      totalTokens: total,
    }
  }

  function extractFileAttachments(raw: Message): AttachmentBlock | null {
    const meta = raw.metadata
    if (!meta || typeof meta !== 'object') return null
    const fa = meta.file_attachments
    if (!Array.isArray(fa) || fa.length === 0) return null
    return { type: 'attachment', attachments: fa }
  }

  function messageToChat(raw: Message): ChatMessage | null {
    if (raw.role !== 'user' && raw.role !== 'assistant') return null

    const text = extractMessageText(raw)
    if (!text) return null

    const createdAt = raw.created_at ? new Date(raw.created_at) : new Date()
    const timestamp = Number.isNaN(createdAt.getTime()) ? new Date() : createdAt
    const platform = (raw.platform ?? '').trim().toLowerCase()
    const channelTag = platform && platform !== 'web' ? platform : undefined
    const tokenUsage = raw.role === 'assistant' ? extractTokenUsageFromMetadata(raw) : undefined

    if (raw.role === 'user') {
      const isSelf = resolveIsSelf(raw)
      const senderName = (raw.sender_display_name ?? '').trim() || undefined
      const senderAvatar = (raw.sender_avatar_url ?? '').trim() || undefined
      return {
        id: raw.id || nextId(),
        role: 'user',
        blocks: [{ type: 'text', content: text }],
        timestamp,
        streaming: false,
        isSelf,
        ...(channelTag && { platform: channelTag }),
        ...(channelTag && { senderDisplayName: senderName, senderAvatarUrl: senderAvatar }),
      }
    }

    const blocks: ContentBlock[] = [{ type: 'text', content: text }]
    const attBlock = extractFileAttachments(raw)
    if (attBlock) blocks.push(attBlock)

    return {
      id: raw.id || nextId(),
      role: 'assistant',
      blocks,
      timestamp,
      streaming: false,
      ...(channelTag && { platform: channelTag }),
      ...(tokenUsage && { tokenUsage }),
    }
  }

  /**
   * Convert an ordered array of raw messages into ChatMessages,
   * merging consecutive assistant(tool_calls) + tool + assistant(text)
   * sequences into a single ChatMessage with ToolCallBlocks.
   */
  function convertMessagesToChats(rows: Message[]): ChatMessage[] {
    const result: ChatMessage[] = []
    let pendingAssistant: ChatMessage | null = null
    const pendingToolCallMap = new Map<string, ToolCallBlock>()

    function flushPending() {
      if (!pendingAssistant) return
      for (const block of pendingAssistant.blocks) {
        if (block.type === 'tool_call' && !block.done) block.done = true
      }
      result.push(pendingAssistant)
      pendingAssistant = null
      pendingToolCallMap.clear()
    }

    function makeTimestamp(raw: Message): Date {
      const d = raw.created_at ? new Date(raw.created_at) : new Date()
      return Number.isNaN(d.getTime()) ? new Date() : d
    }

    for (const raw of rows) {
      if (raw.role === 'user') {
        flushPending()
        const chat = messageToChat(raw)
        if (chat) result.push(chat)
        continue
      }

      if (raw.role === 'assistant') {
        const toolCalls = extractToolCalls(raw)
        const text = extractMessageText(raw)

        if (toolCalls.length > 0) {
          if (!pendingAssistant) {
            const platform = (raw.platform ?? '').trim().toLowerCase()
            const channelTag = platform && platform !== 'web' ? platform : undefined
            pendingAssistant = {
              id: raw.id || nextId(),
              role: 'assistant',
              blocks: [],
              timestamp: makeTimestamp(raw),
              streaming: false,
              ...(channelTag && { platform: channelTag }),
            }
          }
          if (text) {
            pendingAssistant.blocks.push({ type: 'text', content: text })
          }
          for (const tc of toolCalls) {
            const block: ToolCallBlock = {
              type: 'tool_call',
              toolName: tc.name,
              input: tc.input,
              result: null,
              done: false,
            }
            pendingAssistant.blocks.push(block)
            if (tc.id) pendingToolCallMap.set(tc.id, block)
          }
          continue
        }

        // Assistant message without tool_calls
        if (pendingAssistant && text) {
          pendingAssistant.blocks.push({ type: 'text', content: text })
          const attBlock = extractFileAttachments(raw)
          if (attBlock) pendingAssistant.blocks.push(attBlock)
          // Attach token usage from the final assistant message to the merged message.
          const tu = extractTokenUsageFromMetadata(raw)
          if (tu) pendingAssistant.tokenUsage = tu
          flushPending()
          continue
        }

        flushPending()
        const chat = messageToChat(raw)
        if (chat) result.push(chat)
        continue
      }

      if (raw.role === 'tool') {
        const results = extractAllToolResults(raw)
        for (const r of results) {
          if (r.toolCallId && pendingToolCallMap.has(r.toolCallId)) {
            const block = pendingToolCallMap.get(r.toolCallId)!
            block.result = r.output
            block.done = true
          }
        }
        continue
      }
    }

    flushPending()
    return result
  }

  function resolveIsSelf(raw: Message): boolean {
    const platform = (raw.platform ?? '').trim().toLowerCase()
    if (!platform || platform === 'web') return true
    const senderUserId = (raw.sender_user_id ?? '').trim()
    if (!senderUserId) return false
    const userStore = useUserStore()
    const currentUserId = (userStore.userInfo.id ?? '').trim()
    if (!currentUserId) return false
    return senderUserId === currentUserId
  }

  // ---- Abort ----

  function abort() {
    abortFn?.()
    abortFn = null
    if (waitingTimeout) { clearTimeout(waitingTimeout); waitingTimeout = null }
    for (const msg of messages) {
      if (msg.streaming) msg.streaming = false
      if (msg.isWaiting) msg.isWaiting = false
    }
    streaming.value = false
    waitingForResponse.value = false
  }

  function clearWaiting() {
    if (waitingTimeout) { clearTimeout(waitingTimeout); waitingTimeout = null }
    waitingForResponse.value = false
  }

  // ---- Message list management ----

  function replaceMessages(items: ChatMessage[]) {
    messages.splice(0, messages.length, ...items)
  }

  // ---- SSE real-time events ----

  function stopMessageEvents() {
    messageEventsLoopVersion += 1
    if (messageEventsController) {
      messageEventsController.abort()
      messageEventsController = null
    }
  }

  function updateSince(createdAt?: string) {
    const v = (createdAt ?? '').trim()
    if (!v) return
    if (!messageEventsSince) { messageEventsSince = v; return }
    const cur = Date.parse(messageEventsSince)
    const next = Date.parse(v)
    if (!Number.isNaN(next) && (Number.isNaN(cur) || next > cur)) {
      messageEventsSince = v
    }
  }

  function updateSinceFromRows(rows: Message[]) {
    messageEventsSince = ''
    for (const row of rows) updateSince(row.created_at)
  }

  function hasMessageWithId(id: string) {
    const tid = id.trim()
    return tid ? messages.some((m) => String(m.id).trim() === tid) : false
  }

  function appendRealtimeMessage(raw: Message) {
    updateSince(raw.created_at)
    const platform = (raw.platform ?? '').trim().toLowerCase()
    console.debug('[appendRealtimeMessage] id=%s role=%s platform=%s waitingForResponse=%s', raw.id, raw.role, platform, waitingForResponse.value)

    if (platform === 'web') {
      // Replace waiting placeholder when backend finally delivers the assistant message
      if (waitingForResponse.value && raw.role === 'assistant') {
        const idx = messages.findIndex((m) => m.isWaiting)
        if (idx >= 0) {
          const item = messageToChat(raw)
          if (item) {
            messages.splice(idx, 1, item)
            if (waitingTimeout) { clearTimeout(waitingTimeout); waitingTimeout = null }
            waitingForResponse.value = false
            touchChat(chatId.value ?? '')
            return
          }
        }
      }
      // During active streaming, skip web-platform messages to avoid duplicates
      if (streaming.value || loading.value) {
        console.debug('[appendRealtimeMessage] skipped web msg during streaming, id=%s', raw.id)
        return
      }
    }
    const mid = String(raw.id ?? '').trim()
    if (mid && hasMessageWithId(mid)) {
      console.debug('[appendRealtimeMessage] skipped duplicate, id=%s', mid)
      return
    }
    // Skip intermediate assistant messages that contain tool calls — these are
    // multi-step agent steps (e.g. "reading files...") and would appear as
    // duplicates alongside the final text response.
    if (raw.role === 'assistant' && extractToolCalls(raw).length > 0) {
      console.debug('[appendRealtimeMessage] skipped tool-call message, id=%s', raw.id)
      return
    }
    const item = messageToChat(raw)
    if (!item) return
    messages.push(item)
    messages.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
    if (chatId.value) touchChat(chatId.value)
  }

  function handleStreamEvent(targetBotId: string, event: Record<string, unknown>) {
    const eventType = String(event.type ?? '').toLowerCase()

    // Subagent completed: reset waiting timer so we don't timeout prematurely
    if (eventType === 'subagent_completed' && waitingForResponse.value && waitingTimeout) {
      clearTimeout(waitingTimeout)
      waitingTimeout = setTimeout(() => {
        const idx = messages.findIndex((m) => m.isWaiting)
        if (idx >= 0) {
          messages[idx]!.isWaiting = false
          messages[idx]!.blocks = [{ type: 'text', content: i18n.global.t('chat.errors.responseTimeout') as string }]
          waitingForResponse.value = false
        }
        waitingTimeout = null
      }, 180_000)
      return
    }

    if (eventType !== 'message_created') return
    const eBotId = String(event.bot_id ?? '').trim()
    if (eBotId && eBotId !== targetBotId) return
    const payload = event.message
    if (!payload || typeof payload !== 'object') return
    const raw = payload as Message
    const pBotId = String(raw.bot_id ?? '').trim()
    if (pBotId && pBotId !== targetBotId) return
    appendRealtimeMessage(raw)
  }

  function startMessageEvents(targetBotId: string) {
    const bid = targetBotId.trim()
    stopMessageEvents()
    if (!bid) return

    const controller = new AbortController()
    messageEventsController = controller
    const version = messageEventsLoopVersion

    const run = async () => {
      let delay = 1000
      let retries = 0
      const maxRetries = 50
      while (!controller.signal.aborted && messageEventsLoopVersion === version) {
        try {
          await streamMessageEvents(
            bid, controller.signal,
            (e) => handleStreamEvent(bid, e as unknown as Record<string, unknown>),
            messageEventsSince || undefined,
          )
          delay = 1000
          retries = 0
          if (!controller.signal.aborted && messageEventsLoopVersion === version) {
            await sleep(300)
          }
        } catch {
          if (controller.signal.aborted || messageEventsLoopVersion !== version) return
          retries++
          if (retries >= maxRetries) {
            console.warn(`[message-events] max retries (${maxRetries}) reached, stopping`)
            return
          }
          await sleep(delay)
          delay = Math.min(delay * 2, 5000)
        }
      }
    }
    void run()
  }

  // ---- Bot management ----

  async function ensureBot(): Promise<string | null> {
    try {
      const list = await fetchBots()
      bots.value = list
      if (!list.length) { currentBotId.value = null; return null }
      if (currentBotId.value) {
        const found = list.find((b) => b.id === currentBotId.value)
        if (found && !isPendingBot(found)) return currentBotId.value
      }
      const ready = list.find((b) => !isPendingBot(b))
      currentBotId.value = ready ? ready.id : list[0]!.id
      return currentBotId.value
    } catch (err) {
      console.error('Failed to fetch bots:', err)
      return currentBotId.value
    }
  }

  // ---- Pagination ----

  const PAGE_SIZE = 30

  async function loadMessages(botId: string, cid: string) {
    const rows = await fetchMessages(botId, cid, { limit: PAGE_SIZE })
    const items = convertMessagesToChats(rows)
    replaceMessages(items)
    hasMoreOlder.value = true
    updateSinceFromRows(rows)
  }

  async function loadOlderMessages(): Promise<number> {
    const bid = currentBotId.value ?? ''
    const cid = chatId.value ?? ''
    if (!bid || !cid || loadingOlder.value || !hasMoreOlder.value) return 0
    const first = messages[0]
    if (!first?.timestamp) return 0

    const before = first.timestamp.toISOString()
    loadingOlder.value = true
    try {
      const rows = await fetchMessages(bid, cid, { limit: PAGE_SIZE, before })
      const items = convertMessagesToChats(rows)
      if (rows.length < PAGE_SIZE) hasMoreOlder.value = false
      messages.unshift(...items)
      return items.length
    } finally {
      loadingOlder.value = false
    }
  }

  // ---- Chat CRUD ----

  function touchChat(targetChatId: string) {
    const idx = chats.value.findIndex((c) => c.id === targetChatId)
    if (idx < 0) return
    const [target] = chats.value.splice(idx, 1)
    if (!target) return
    target.updated_at = new Date().toISOString()
    chats.value.unshift(target)
  }

  async function ensureActiveChat() {
    if (chatId.value) return
    const bid = currentBotId.value ?? await ensureBot()
    if (!bid) throw new Error('Bot not ready')
    const created = await createChat(bid)
    chats.value = [created, ...chats.value.filter((c) => c.id !== created.id)]
    chatId.value = created.id
    replaceMessages([])
  }

  // ---- Initialize ----

  async function initialize() {
    if (initializing.value) return
    initializing.value = true
    loadingChats.value = true
    clearWaiting()
    stopMessageEvents()
    try {
      const bid = await ensureBot()
      if (!bid) {
        messageEventsSince = ''
        chats.value = []
        chatId.value = null
        replaceMessages([])
        return
      }
      const visible = await fetchChats(bid)
      chats.value = visible
      if (!visible.length) {
        messageEventsSince = ''
        chatId.value = null
        replaceMessages([])
        return
      }
      const activeChatId = chatId.value && visible.some((c) => c.id === chatId.value)
        ? chatId.value
        : visible[0]!.id
      chatId.value = activeChatId
      await loadMessages(bid, activeChatId)

      // After refresh: if last message is user with no assistant reply,
      // backend is likely still processing. Show waiting placeholder.
      const last = messages[messages.length - 1]
      if (last?.role === 'user') {
        messages.push({
          id: `wait-${Date.now()}`,
          role: 'assistant',
          blocks: [],
          timestamp: new Date(),
          streaming: false,
          isWaiting: true,
        })
        waitingForResponse.value = true
        waitingTimeout = setTimeout(() => {
          const idx = messages.findIndex((m) => m.isWaiting)
          if (idx >= 0) {
            messages[idx]!.isWaiting = false
            messages[idx]!.blocks = [{ type: 'text', content: i18n.global.t('chat.errors.responseTimeout') as string }]
            waitingForResponse.value = false
          }
          waitingTimeout = null
        }, 180_000)
      }

      startMessageEvents(bid)
    } finally {
      loadingChats.value = false
      initializing.value = false
    }
  }

  async function selectBot(targetBotId: string) {
    if (currentBotId.value === targetBotId) return
    abort()
    currentBotId.value = targetBotId
    chatId.value = null
    await initialize()
  }

  async function selectChat(targetChatId: string) {
    const cid = targetChatId.trim()
    if (!cid || cid === chatId.value) return
    clearWaiting()
    chatId.value = cid
    loadingChats.value = true
    try {
      const bid = currentBotId.value ?? ''
      if (!bid) throw new Error('Bot not selected')
      await loadMessages(bid, cid)
    } finally {
      loadingChats.value = false
    }
  }

  async function createNewChat() {
    clearWaiting()
    loadingChats.value = true
    try {
      const bid = await ensureBot()
      if (!bid) return
      const created = await createChat(bid)
      chats.value = [created, ...chats.value.filter((c) => c.id !== created.id)]
      chatId.value = created.id
      replaceMessages([])
    } finally {
      loadingChats.value = false
    }
  }

  async function removeChat(targetChatId: string) {
    const delId = targetChatId.trim()
    if (!delId) return
    clearWaiting()
    loadingChats.value = true
    try {
      const bid = currentBotId.value ?? ''
      if (!bid) throw new Error('Bot not selected')
      await requestDeleteChat(bid, delId)
      const remaining = chats.value.filter((c) => c.id !== delId)
      chats.value = remaining
      if (chatId.value !== delId) return
      if (!remaining.length) {
        chatId.value = null
        replaceMessages([])
        return
      }
      chatId.value = remaining[0]!.id
      await loadMessages(bid, remaining[0]!.id)
    } finally {
      loadingChats.value = false
    }
  }

  // ---- Send message (blocks-based streaming) ----

  async function sendMessage(text: string) {
    const trimmed = text.trim()
    if (!trimmed || streaming.value || !currentBotId.value) return

    loading.value = true
    streaming.value = true

    try {
      await ensureActiveChat()
      if (activeChatReadOnly.value) throw new Error('Chat is read-only')

      const bid = currentBotId.value!
      const cid = chatId.value!

      // Add user message
      messages.push({
        id: nextId(),
        role: 'user',
        blocks: [{ type: 'text', content: trimmed }],
        timestamp: new Date(),
        streaming: false,
      })

      // Add assistant placeholder
      messages.push({
        id: nextId(),
        role: 'assistant',
        blocks: [],
        timestamp: new Date(),
        streaming: true,
      })
      const assistantMsg = messages[messages.length - 1]!

      let textBlockIdx = -1
      let thinkingBlockIdx = -1

      function pushBlock(block: ContentBlock): number {
        assistantMsg.blocks.push(block)
        return assistantMsg.blocks.length - 1
      }

      abortFn = streamMessage(
        bid, cid, trimmed,
        (event: StreamEvent) => {
          const type = (event.type ?? '').toLowerCase()

          switch (type) {
            case 'text_start':
              textBlockIdx = pushBlock({ type: 'text', content: '' })
              break

            case 'text_delta':
              if (typeof event.delta === 'string') {
                if (textBlockIdx < 0 || assistantMsg.blocks[textBlockIdx]?.type !== 'text') {
                  textBlockIdx = pushBlock({ type: 'text', content: '' })
                }
                ;(assistantMsg.blocks[textBlockIdx] as TextBlock).content += event.delta
              }
              break

            case 'text_end':
              textBlockIdx = -1
              break

            case 'reasoning_start':
              thinkingBlockIdx = pushBlock({ type: 'thinking', content: '', done: false })
              break

            case 'reasoning_delta':
              if (typeof event.delta === 'string') {
                if (thinkingBlockIdx < 0 || assistantMsg.blocks[thinkingBlockIdx]?.type !== 'thinking') {
                  thinkingBlockIdx = pushBlock({ type: 'thinking', content: '', done: false })
                }
                ;(assistantMsg.blocks[thinkingBlockIdx] as ThinkingBlock).content += event.delta
              }
              break

            case 'reasoning_end':
              if (thinkingBlockIdx >= 0 && assistantMsg.blocks[thinkingBlockIdx]?.type === 'thinking') {
                ;(assistantMsg.blocks[thinkingBlockIdx] as ThinkingBlock).done = true
              }
              thinkingBlockIdx = -1
              break

            case 'tool_call_start':
              pushBlock({
                type: 'tool_call',
                toolName: (event.toolName as string) ?? 'unknown',
                input: event.input ?? null,
                result: null,
                done: false,
              })
              textBlockIdx = -1
              break

            case 'tool_call_end':
              for (let i = 0; i < assistantMsg.blocks.length; i++) {
                const b = assistantMsg.blocks[i]
                if (b && b.type === 'tool_call' && b.toolName === event.toolName && !b.done) {
                  b.result = event.result ?? null
                  b.done = true
                  break
                }
              }
              break

            case 'attachment_delta':
              if (Array.isArray(event.attachments) && event.attachments.length > 0) {
                pushBlock({ type: 'attachment', attachments: event.attachments })
              }
              break

            case 'image_delta':
              if (typeof event.image === 'string' && event.image) {
                pushBlock({ type: 'image', src: event.image })
                textBlockIdx = -1
              }
              break

            case 'subagent_progress': {
              const runId = String((event as any).runId ?? '')
              if (!runId) break
              let spBlock = assistantMsg.blocks.find((b): b is SubagentProgressBlock => b.type === 'subagent_progress') as SubagentProgressBlock | undefined
              if (!spBlock) {
                spBlock = { type: 'subagent_progress', runs: [] }
                pushBlock(spBlock)
              }
              const existing = spBlock.runs.find((r) => r.runId === runId)
              const info = { runId, name: String((event as any).name ?? ''), task: String((event as any).task ?? ''), status: String((event as any).status ?? 'running'), elapsed_ms: Number((event as any).elapsed_ms ?? 0) }
              if (existing) { Object.assign(existing, info) } else { spBlock.runs.push(info) }
              const now = Date.now()
              spBlock.runs = spBlock.runs.filter((r) => !r.completedAt || now - r.completedAt < 10_000)
              break
            }

            case 'subagent_delta': {
              const runId = String((event as any).runId ?? '')
              const delta = String((event as any).delta ?? '')
              if (!runId || !delta) break
              const spBlock = assistantMsg.blocks.find((b): b is SubagentProgressBlock => b.type === 'subagent_progress') as SubagentProgressBlock | undefined
              if (spBlock) {
                const run = spBlock.runs.find((r) => r.runId === runId)
                if (run) run.delta = (run.delta ?? '') + delta
              }
              break
            }

            case 'subagent_completed': {
              const runId = String((event as any).runId ?? '')
              if (!runId) break
              const spBlock = assistantMsg.blocks.find((b): b is SubagentProgressBlock => b.type === 'subagent_progress') as SubagentProgressBlock | undefined
              if (spBlock) {
                const run = spBlock.runs.find((r) => r.runId === runId)
                if (run) {
                  run.status = String((event as any).status ?? 'completed')
                  run.completedAt = Date.now()
                }
              }
              break
            }

            case 'processing_started':
              if (assistantMsg.blocks.length === 0) {
                pushBlock({ type: 'text', content: '' })
                textBlockIdx = 0
              }
              break

            case 'processing_completed':
            case 'processing_failed':
            case 'agent_start':
              break

            case 'agent_end':
              if (event.usage && typeof event.usage === 'object') {
                const u = event.usage as Record<string, number>
                assistantMsg.tokenUsage = {
                  promptTokens: u.promptTokens ?? 0,
                  completionTokens: u.completionTokens ?? 0,
                  totalTokens: u.totalTokens ?? 0,
                }
              }
              break

            case 'error': {
              const rawErr = typeof event.error === 'string' ? event.error : typeof event.message === 'string' ? event.message : 'Stream error'
              const errMsg = resolveBillingError(rawErr)
              if (textBlockIdx < 0 || assistantMsg.blocks[textBlockIdx]?.type !== 'text') {
                textBlockIdx = pushBlock({ type: 'text', content: '' })
              }
              ;(assistantMsg.blocks[textBlockIdx] as TextBlock).content += `\n\n**Error:** ${errMsg}`
              break
            }

            default: {
              const fallback = extractFallbackText(event)
              if (fallback) {
                if (textBlockIdx < 0 || assistantMsg.blocks[textBlockIdx]?.type !== 'text') {
                  textBlockIdx = pushBlock({ type: 'text', content: '' })
                }
                ;(assistantMsg.blocks[textBlockIdx] as TextBlock).content += fallback
              }
              break
            }
          }
        },
        () => {
          assistantMsg.streaming = false
          streaming.value = false
          loading.value = false
          abortFn = null
          touchChat(cid)
        },
        (err) => {
          assistantMsg.streaming = false
          loading.value = false
          abortFn = null

          if (assistantMsg.blocks.length > 0) {
            // Partial content already received — keep it and stop streaming normally.
            streaming.value = false
            return
          }

          // No content received: enter waiting state.
          assistantMsg.isWaiting = true
          waitingForResponse.value = true
          streaming.value = false

          waitingTimeout = setTimeout(() => {
            if (assistantMsg.isWaiting) {
              assistantMsg.isWaiting = false
              assistantMsg.blocks = [{
                type: 'text',
                content: i18n.global.t('chat.errors.responseTimeout') as string,
              }]
              waitingForResponse.value = false
            }
            waitingTimeout = null
          }, 180_000)
        },
      )
    } catch (err) {
      const raw = err instanceof Error ? err.message : 'Unknown error'
      const reason = resolveBillingError(raw)
      const last = messages[messages.length - 1]
      if (last?.role === 'assistant' && last.streaming) {
        last.blocks = [{ type: 'text', content: reason }]
        last.streaming = false
      } else {
        messages.push({
          id: nextId(),
          role: 'assistant',
          blocks: [{ type: 'text', content: reason }],
          timestamp: new Date(),
          streaming: false,
        })
      }
      streaming.value = false
      loading.value = false
    }
  }

  function retryLastMessage() {
    for (let i = messages.length - 1; i >= 0; i--) {
      if (messages[i].role === 'user' && messages[i].blocks[0]?.type === 'text') {
        const text = (messages[i].blocks[0] as TextBlock).content
        messages.splice(i + 1)
        streaming.value = false
        loading.value = false
        waitingForResponse.value = false
        if (waitingTimeout) { clearTimeout(waitingTimeout); waitingTimeout = null }
        void sendMessage(text)
        return
      }
    }
  }

  function clearMessages() {
    abort()
    replaceMessages([])
  }

  return {
    messages,
    streaming,
    waitingForResponse,
    chats,
    participantChats,
    observedChats,
    chatId,
    currentBotId,
    bots,
    activeChat,
    activeChatReadOnly,
    loading,
    loadingChats,
    loadingOlder,
    hasMoreOlder,
    initializing,
    initialize,
    selectBot,
    selectChat,
    createNewChat,
    removeChat,
    deleteChat: removeChat,
    sendMessage,
    retryLastMessage,
    clearMessages,
    loadOlderMessages,
    abort,
  }
})

function extractFallbackText(event: StreamEvent): string | null {
  if (typeof event.delta === 'string') return event.delta
  if (typeof (event as Record<string, unknown>).text === 'string') return (event as Record<string, unknown>).text as string
  if (typeof (event as Record<string, unknown>).content === 'string') return (event as Record<string, unknown>).content as string
  return null
}
