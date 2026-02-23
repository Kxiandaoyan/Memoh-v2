<template>
  <div
    class="flex gap-3 items-start"
    :class="message.role === 'user' && isSelf ? 'justify-end' : ''"
  >
    <!-- Assistant avatar -->
    <div
      v-if="message.role === 'assistant'"
      class="relative shrink-0"
    >
      <Avatar class="size-8">
        <AvatarImage
          v-if="botAvatarUrl"
          :src="botAvatarUrl"
          :alt="botName"
        />
        <AvatarFallback class="text-xs bg-primary/10 text-primary">
          <FontAwesomeIcon
            :icon="['fas', 'robot']"
            class="size-4"
          />
        </AvatarFallback>
      </Avatar>
      <ChannelBadge
        v-if="message.platform"
        :platform="message.platform"
      />
    </div>

    <!-- User avatar (other sender, left-aligned) -->
    <div
      v-if="message.role === 'user' && !isSelf"
      class="relative shrink-0"
    >
      <Avatar class="size-8">
        <AvatarImage
          v-if="message.senderAvatarUrl"
          :src="message.senderAvatarUrl"
          :alt="message.senderDisplayName"
        />
        <AvatarFallback class="text-xs">
          {{ senderFallback }}
        </AvatarFallback>
      </Avatar>
      <ChannelBadge
        v-if="message.platform"
        :platform="message.platform"
      />
    </div>

    <!-- Content -->
    <div
      class="min-w-0"
      :class="contentClass"
    >
      <!-- Sender name for non-self user messages -->
      <p
        v-if="message.role === 'user' && !isSelf"
        class="text-xs text-muted-foreground mb-1"
      >
        {{ message.senderDisplayName || senderFallbackName }}
      </p>

      <!-- User message -->
      <div
        v-if="message.role === 'user'"
        class="rounded-2xl px-4 py-2.5 text-sm whitespace-pre-wrap"
        :class="isSelf
          ? 'rounded-tr-sm bg-primary text-primary-foreground'
          : 'rounded-tl-sm bg-accent/60 text-foreground'"
      >
        {{ (message.blocks[0] as TextBlock)?.content }}
      </div>

      <!-- Assistant message blocks -->
      <div
        v-else
        class="space-y-3"
      >
        <!-- Bot name label -->
        <p
          v-if="botName"
          class="text-xs text-muted-foreground"
        >
          {{ botName }}
        </p>

        <template
          v-for="(block, i) in message.blocks"
          :key="i"
        >
          <!-- Thinking block -->
          <ThinkingBlock
            v-if="block.type === 'thinking'"
            :block="(block as ThinkingBlockType)"
            :streaming="message.streaming && !block.done"
          />

          <!-- Tool call block -->
          <ToolCallBlock
            v-else-if="block.type === 'tool_call'"
            :block="(block as ToolCallBlockType)"
          />

          <!-- Text block -->
          <div
            v-else-if="block.type === 'text' && block.content"
            class="prose prose-sm dark:prose-invert max-w-none *:first:mt-0"
          >
            <MarkdownRender
              :content="block.content"
              custom-id="chat-msg"
            />
          </div>

          <!-- Image block -->
          <div
            v-else-if="block.type === 'image' && (block as ImageBlockType).src"
            class="max-w-sm"
          >
            <img
              :src="(block as ImageBlockType).src"
              alt="Generated image"
              class="rounded-lg border max-w-full h-auto"
            >
          </div>

          <!-- Attachment block -->
          <div
            v-else-if="block.type === 'attachment'"
            class="flex flex-wrap gap-2"
          >
            <button
              v-for="(att, ai) in (block as AttachmentBlockType).attachments"
              :key="ai"
              class="inline-flex items-center gap-2 rounded-lg border bg-muted/50 hover:bg-muted px-3 py-2 text-xs text-foreground transition-colors cursor-pointer"
              :title="getAttachmentName(att, ai)"
              @click="downloadAttachment(att, ai)"
            >
              <FontAwesomeIcon :icon="['fas', 'file-lines']" class="size-4 text-muted-foreground" />
              <span class="truncate max-w-[200px]">{{ getAttachmentName(att, ai) }}</span>
              <FontAwesomeIcon :icon="['fas', 'download']" class="size-3 text-muted-foreground ml-1" />
            </button>
          </div>
        </template>

        <!-- Streaming indicator -->
        <div
          v-if="message.streaming && message.blocks.length === 0"
          class="flex items-center gap-2 text-sm text-muted-foreground h-8"
        >
          <FontAwesomeIcon
            :icon="['fas', 'spinner']"
            class="size-3.5 animate-spin"
          />
          {{ $t('chat.thinking') }}
        </div>

        <!-- Token usage badge -->
        <span
          v-if="!message.streaming && message.tokenUsage && message.tokenUsage.totalTokens > 0"
          class="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground/60 select-none"
          :title="`Prompt: ${message.tokenUsage.promptTokens} / Completion: ${message.tokenUsage.completionTokens}`"
        >
          ⚡{{ formatTokens(message.tokenUsage.totalTokens) }}
        </span>
      </div>
    </div>

    <!-- Self user avatar (right side) -->
    <div
      v-if="message.role === 'user' && isSelf"
      class="relative shrink-0"
    >
      <Avatar class="size-8">
        <AvatarImage
          v-if="selfAvatarUrl"
          :src="selfAvatarUrl"
          alt=""
        />
        <AvatarFallback class="text-xs">
          {{ selfFallback }}
        </AvatarFallback>
      </Avatar>
      <ChannelBadge
        v-if="message.platform"
        :platform="message.platform"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Avatar, AvatarImage, AvatarFallback } from '@memoh/ui'
import MarkdownRender, { enableKatex, enableMermaid } from 'markstream-vue'
import ThinkingBlock from './thinking-block.vue'
import ToolCallBlock from './tool-call-block.vue'
import ChannelBadge from '@/components/chat-list/channel-badge/index.vue'
import { useUserStore } from '@/store/user'
import { useChatStore } from '@/store/chat-list'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import type {
  ChatMessage,
  TextBlock,
  ThinkingBlock as ThinkingBlockType,
  ToolCallBlock as ToolCallBlockType,
  ImageBlock as ImageBlockType,
  AttachmentBlock as AttachmentBlockType,
} from '@/store/chat-list'

enableKatex()
enableMermaid()

const props = defineProps<{
  message: ChatMessage
}>()

const userStore = useUserStore()
const chatStore = useChatStore()
const { currentBotId, bots } = storeToRefs(chatStore)

const isSelf = computed(() => props.message.isSelf !== false)

const currentBot = computed(() =>
  bots.value.find((b) => b.id === currentBotId.value) ?? null,
)

const botAvatarUrl = computed(() => currentBot.value?.avatar_url ?? '')
const botName = computed(() => currentBot.value?.display_name ?? '')

// For isSelf messages: prefer channel avatar/name over web platform avatar
const selfAvatarUrl = computed(() =>
  props.message.senderAvatarUrl || userStore.userInfo.avatarUrl || '',
)
const selfFallback = computed(() => {
  const name = props.message.senderDisplayName
    || userStore.userInfo.displayName
    || userStore.userInfo.username
    || ''
  return name.slice(0, 2).toUpperCase() || 'U'
})

const { t } = useI18n()

const senderFallbackName = computed(() => {
  const p = (props.message.platform ?? '').trim()
  const platformLabel = p
    ? t(`bots.channels.types.${p}`, p.charAt(0).toUpperCase() + p.slice(1))
    : ''
  return t('chat.unknownUser', { platform: platformLabel })
})

const senderFallback = computed(() => {
  const name = props.message.senderDisplayName ?? ''
  return name.slice(0, 2).toUpperCase() || '?'
})

const contentClass = computed(() => {
  if (props.message.role === 'user') {
    return isSelf.value ? 'max-w-[80%]' : 'max-w-[80%]'
  }
  return 'flex-1 max-w-full'
})

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`
  return String(n)
}

function getAttachmentName(att: unknown, index: number): string {
  if (typeof att !== 'object' || att === null) return `Attachment ${index + 1}`
  const obj = att as Record<string, unknown>
  if (typeof obj.path === 'string' && obj.path) return obj.path.split('/').pop() || `Attachment ${index + 1}`
  if (typeof obj.name === 'string' && obj.name) return obj.name
  return `Attachment ${index + 1}`
}

function getAttachmentPath(att: unknown): string {
  if (typeof att !== 'object' || att === null) return ''
  const obj = att as Record<string, unknown>
  return typeof obj.path === 'string' ? obj.path : ''
}

async function downloadAttachment(att: unknown, index: number) {
  const path = getAttachmentPath(att)
  if (!path || !currentBotId.value) return
  const apiBase = (import.meta.env.VITE_API_URL?.trim() || '/api')
  const url = path.startsWith('/shared/')
    ? `${apiBase}/shared/files/download/${path.slice('/shared/'.length)}`
    : `${apiBase}/bots/${currentBotId.value}/files/download/${path.replace(/^\/data\//, '')}`
  const token = localStorage.getItem('token')
  try {
    const resp = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!resp.ok) throw new Error(`Download failed: ${resp.status}`)
    const blob = await resp.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = getAttachmentName(att, index)
    document.body.appendChild(a)
    a.click()
    setTimeout(() => { URL.revokeObjectURL(a.href); document.body.removeChild(a) }, 100)
  } catch (e) {
    console.error('Download failed:', e)
    toast.error('Download failed')
  }
}
</script>
