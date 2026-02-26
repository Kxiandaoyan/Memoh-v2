<template>
  <div class="flex-1 flex flex-col h-full min-w-0">
    <!-- No bot selected -->
    <div
      v-if="!currentBotId"
      class="flex-1 flex flex-col items-center justify-center text-muted-foreground gap-4"
    >
      <!-- Mobile: show bot list button -->
      <Button
        variant="outline"
        class="md:hidden gap-2"
        @click="emit('toggle-bot-list')"
      >
        <FontAwesomeIcon :icon="['fas', 'bars']" class="size-3.5" />
        {{ $t('chat.selectBot') }}
      </Button>
      <div class="text-center hidden md:block">
        <p class="text-lg">{{ $t('chat.selectBot') }}</p>
        <p class="text-sm mt-1">{{ $t('chat.selectBotHint') }}</p>
      </div>
    </div>

    <template v-else>
      <!-- Bot info header -->
      <div
        v-if="currentBot"
        class="flex items-center gap-3 px-4 py-2.5 border-b"
      >
        <!-- Mobile: toggle bot list -->
        <Button
          variant="ghost"
          size="sm"
          class="md:hidden -ml-1 shrink-0 size-8 p-0"
          @click="emit('toggle-bot-list')"
        >
          <FontAwesomeIcon :icon="['fas', 'bars']" class="size-3.5" />
        </Button>
        <Avatar class="size-8 shrink-0">
          <AvatarImage
            v-if="currentBot.avatar_url"
            :src="currentBot.avatar_url"
            :alt="currentBot.display_name"
          />
          <AvatarFallback class="text-xs">
            {{ (currentBot.display_name || currentBot.id || '').slice(0, 2).toUpperCase() }}
          </AvatarFallback>
        </Avatar>
        <div class="min-w-0">
          <span class="font-medium text-sm truncate">
            {{ currentBot.display_name || currentBot.id }}
          </span>
        </div>
        <Badge
          v-if="activeChatReadOnly"
          variant="secondary"
          class="ml-auto text-xs"
        >
          {{ $t('chat.readonly') }}
        </Badge>
      </div>

      <!-- Messages -->
      <div
        ref="scrollContainer"
        role="log"
        aria-live="polite"
        aria-relevant="additions text"
        class="flex-1 overflow-y-auto relative"
        @scroll="handleScroll"
      >
        <div class="max-w-3xl mx-auto px-4 py-6 space-y-6">
          <!-- Load older indicator -->
          <div
            v-if="loadingOlder"
            class="flex justify-center py-2"
          >
            <FontAwesomeIcon
              :icon="['fas', 'spinner']"
              class="size-3.5 animate-spin text-muted-foreground"
            />
          </div>

          <!-- Empty state -->
          <div
            v-if="messages.length === 0 && !loadingChats"
            class="flex items-center justify-center min-h-[300px]"
          >
            <p class="text-muted-foreground text-lg">
              {{ $t('chat.greeting') }}
            </p>
          </div>

          <!-- Message list -->
          <MessageItem
            v-for="msg in messages"
            :key="msg.id"
            :message="msg"
          />
        </div>

        <!-- Scroll to bottom button -->
        <ScrollToBottom
          :visible="showScrollBtn"
          :unread-count="unreadCount"
          @click="scrollToBottomClicked"
        />
      </div>

      <!-- Waiting for response banner -->
      <div
        v-if="waitingForResponse"
        class="flex items-center gap-2 px-4 py-2 bg-amber-50 dark:bg-amber-950/30 border-t border-amber-200 dark:border-amber-800 text-amber-700 dark:text-amber-300 text-xs"
      >
        <FontAwesomeIcon :icon="['fas', 'spinner']" class="size-3 animate-spin shrink-0" />
        <span>{{ $t('chat.errors.waitingForResponse') }}</span>
      </div>

      <!-- Input -->
      <div class="border-t p-4">
        <div class="max-w-3xl mx-auto space-y-2">
          <!-- Upload progress -->
          <div v-if="uploadingFiles.length" class="space-y-1">
            <div v-for="(f, i) in uploadingFiles" :key="i" class="flex items-center gap-2 text-xs text-muted-foreground">
              <span class="truncate max-w-[200px]">{{ f.name }}</span>
              <Progress :model-value="f.progress" class="flex-1 h-1.5" />
              <span>{{ f.progress }}%</span>
            </div>
          </div>
          <!-- Attached files -->
          <div v-if="attachedFiles.length" class="flex flex-wrap gap-1.5">
            <div
              v-for="(f, i) in attachedFiles"
              :key="i"
              class="flex items-center gap-1.5 bg-muted rounded-md px-2 py-1 text-xs"
            >
              <FontAwesomeIcon :icon="['fas', 'file']" class="size-3 text-muted-foreground" />
              <span class="truncate max-w-[150px]">{{ f.name }}</span>
              <span class="text-muted-foreground">({{ formatFileSize(f.size) }})</span>
              <button type="button" class="text-muted-foreground hover:text-destructive" @click="removeAttachment(i)">
                <FontAwesomeIcon :icon="['fas', 'xmark']" class="size-3" />
              </button>
            </div>
          </div>
          <!-- Textarea + buttons -->
          <div class="relative">
            <Textarea
              v-model="inputText"
              class="pr-16 pl-10 min-h-[60px] max-h-[200px] resize-none"
              :aria-label="$t('chat.inputPlaceholder')"
              :placeholder="activeChatReadOnly ? $t('chat.readonlyHint') : $t('chat.inputPlaceholder')"
              :disabled="!currentBotId || activeChatReadOnly"
              @keydown.enter.exact="handleKeydown"
            />
            <button
              type="button"
              class="absolute left-2.5 bottom-2.5 text-muted-foreground hover:text-foreground transition-colors"
              :disabled="!currentBotId || activeChatReadOnly"
              @click="triggerFileInput"
            >
              <FontAwesomeIcon :icon="['fas', 'paperclip']" class="size-4" />
            </button>
            <input
              ref="fileInputRef"
              type="file"
              multiple
              class="hidden"
              accept=".doc,.docx,.pdf,.ppt,.pptx,.xls,.xlsx,.csv,.txt,.md,.json,.yaml,.yml,.xml,.html,.htm,.zip,.tar,.gz,.png,.jpg,.jpeg,.gif,.webp,.svg,.mp3,.wav,.mp4"
              @change="handleFileSelect"
            >
            <div class="absolute right-2 bottom-2">
              <Button
                v-if="!streaming"
                size="sm"
                :disabled="!inputText.trim() || !currentBotId || activeChatReadOnly"
                @click="handleSend"
              >
                <FontAwesomeIcon :icon="['fas', 'paper-plane']" class="size-3.5" />
              </Button>
              <Button
                v-else
                size="sm"
                variant="destructive"
                @click="chatStore.abort()"
              >
                <FontAwesomeIcon :icon="['fas', 'spinner']" class="size-3.5 animate-spin" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { Textarea, Button, Avatar, AvatarImage, AvatarFallback, Badge, Progress } from '@memoh/ui'
import { useChatStore } from '@/store/chat-list'
import { useChat } from '@/composables/api/useChat'
import type { FileRef } from '@/composables/api/useChat'
import { storeToRefs } from 'pinia'
import MessageItem from './message-item.vue'
import ScrollToBottom from './scroll-to-bottom.vue'

const emit = defineEmits<{ (e: 'toggle-bot-list'): void }>()

const chatStore = useChatStore()
const { uploadFile } = useChat()

const fileInputRef = ref<HTMLInputElement>()
const uploadingFiles = ref<{ name: string; progress: number }[]>([])
const attachedFiles = ref<FileRef[]>([])

const {
  messages,
  streaming,
  waitingForResponse,
  currentBotId,
  bots,
  activeChatReadOnly,
  loadingOlder,
  loadingChats,
  hasMoreOlder,
} = storeToRefs(chatStore)

const inputText = ref('')
const scrollContainer = ref<HTMLElement>()
const showScrollBtn = ref(false)
const unreadCount = ref(0)

const currentBot = computed(() =>
  bots.value.find((b) => b.id === currentBotId.value) ?? null,
)

onMounted(async () => {
  await chatStore.initialize()
})

// When messages finish loading, scroll to bottom instantly.
watch(loadingChats, (loading, wasLoading) => {
  if (wasLoading && !loading && messages.value.length > 0) {
    userScrolledUp = false
    scrollToBottomAfterRender()
  }
})

// ---- Auto-scroll ----

let userScrolledUp = false

function scrollToBottom(smooth = true) {
  nextTick(() => {
    const el = scrollContainer.value
    if (!el) return
    el.scrollTo({
      top: el.scrollHeight,
      behavior: smooth ? 'smooth' : 'instant',
    })
  })
}

// Wait for two render cycles + a rAF before scrolling so that all child
// components have had a chance to paint and scrollHeight is stable.
function scrollToBottomAfterRender() {
  nextTick(() => {
    nextTick(() => {
      requestAnimationFrame(() => {
        const el = scrollContainer.value
        if (!el) return
        el.scrollTo({ top: el.scrollHeight, behavior: 'instant' })
      })
    })
  })
}

function scrollToBottomClicked() {
  userScrolledUp = false
  unreadCount.value = 0
  showScrollBtn.value = false
  scrollToBottom(true)
}

function handleScroll() {
  const el = scrollContainer.value
  if (!el) return
  const distanceFromBottom = el.scrollHeight - el.clientHeight - el.scrollTop
  // During streaming, use a generous threshold so that Markdown layout shifts
  // don't falsely disengage auto-scroll.
  const threshold = streaming.value ? 300 : 150
  userScrolledUp = distanceFromBottom > threshold
  showScrollBtn.value = userScrolledUp
  if (!userScrolledUp) unreadCount.value = 0

  // Load older messages when scrolled near top
  if (el.scrollTop < 200 && hasMoreOlder.value && !loadingOlder.value) {
    const prevHeight = el.scrollHeight
    chatStore.loadOlderMessages().then((count) => {
      if (count > 0) {
        nextTick(() => {
          el.scrollTop = el.scrollHeight - prevHeight
        })
      }
    })
  }
}

// Stream content auto-scroll
watch(
  () => {
    const last = messages.value[messages.value.length - 1]
    return last?.blocks.reduce((acc, b) => {
      if (b.type === 'text') return acc + b.content.length
      if (b.type === 'thinking') return acc + b.content.length
      return acc + 1
    }, 0) ?? 0
  },
  () => {
    if (!userScrolledUp) scrollToBottom()
  },
)

// New message auto-scroll / unread counter
watch(
  () => messages.value.length,
  () => {
    if (userScrolledUp) {
      unreadCount.value += 1
    } else {
      unreadCount.value = 0
      userScrolledUp = false
      scrollToBottom()
    }
  },
)

// When streaming ends, always scroll to show the complete response
watch(streaming, (isStreaming, wasStreaming) => {
  if (wasStreaming && !isStreaming) {
    nextTick(() => scrollToBottom(true))
  }
})

function handleKeydown(e: KeyboardEvent) {
  if (e.isComposing) return
  e.preventDefault()
  handleSend()
}

function handleSend() {
  const text = inputText.value.trim()
  if (!text || streaming.value || activeChatReadOnly.value) return
  inputText.value = ''
  const refs = attachedFiles.value.length > 0 ? [...attachedFiles.value] : undefined
  attachedFiles.value = []
  chatStore.sendMessage(text, refs)
}

function triggerFileInput() {
  fileInputRef.value?.click()
}

async function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  const files = input.files
  if (!files?.length) return
  for (const file of Array.from(files)) {
    const entry = { name: file.name, progress: 0 }
    uploadingFiles.value.push(entry)
    try {
      const ref = await uploadFile(file, (pct) => { entry.progress = pct })
      attachedFiles.value.push(ref)
    } catch { /* ignore */ }
    uploadingFiles.value = uploadingFiles.value.filter(f => f !== entry)
  }
  input.value = ''
}

function removeAttachment(idx: number) {
  attachedFiles.value.splice(idx, 1)
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>
