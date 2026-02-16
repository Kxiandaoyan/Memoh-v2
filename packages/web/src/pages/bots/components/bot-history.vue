<template>
  <div class="max-w-4xl mx-auto space-y-5">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('bots.history.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('bots.history.subtitle') }}
        </p>
      </div>
      <div class="flex gap-2 shrink-0">
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadMessages(true)"
        >
          <Spinner
            v-if="loading"
            class="mr-1.5"
          />
          {{ $t('common.refresh') }}
        </Button>
        <ConfirmPopover
          v-if="messages.length > 0"
          :message="$t('bots.history.clearConfirm')"
          :loading="clearing"
          @confirm="handleClear"
        >
          <template #trigger>
            <Button
              variant="destructive"
              size="sm"
              :disabled="clearing"
            >
              {{ $t('bots.history.clear') }}
            </Button>
          </template>
        </ConfirmPopover>
      </div>
    </div>

    <!-- Loading -->
    <div
      v-if="loading && messages.length === 0"
      class="flex items-center gap-2 text-sm text-muted-foreground"
    >
      <Spinner />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty -->
    <div
      v-else-if="messages.length === 0"
      class="rounded-md border p-6 text-center text-sm text-muted-foreground"
    >
      {{ $t('bots.history.empty') }}
    </div>

    <!-- Message list -->
    <div
      v-else
      class="space-y-2"
    >
      <!-- Load more -->
      <div
        v-if="hasMore"
        class="text-center"
      >
        <Button
          variant="ghost"
          size="sm"
          :disabled="loading"
          @click="loadOlder"
        >
          <Spinner
            v-if="loading"
            class="mr-1.5"
          />
          {{ $t('bots.history.loadMore') }}
        </Button>
      </div>

      <div
        v-for="msg in messages"
        :key="msg.id"
        class="rounded-md border p-3 space-y-1"
        :class="msg.role === 'assistant' ? 'bg-muted/30' : ''"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2 min-w-0">
            <Badge
              :variant="roleBadgeVariant(msg.role)"
              class="text-xs shrink-0"
            >
              {{ msg.role }}
            </Badge>
            <span
              v-if="msg.sender_display_name"
              class="text-xs text-muted-foreground truncate"
            >
              {{ msg.sender_display_name }}
            </span>
            <Badge
              v-if="msg.platform"
              variant="outline"
              class="text-[10px] shrink-0"
            >
              {{ msg.platform }}
            </Badge>
          </div>
          <span class="text-xs text-muted-foreground shrink-0">
            {{ formatDate(msg.created_at) }}
          </span>
        </div>
        <div class="text-sm whitespace-pre-wrap break-words">
          {{ extractText(msg.content) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Badge,
  Button,
  Spinner,
} from '@memoh/ui'
import { ref, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'
import ConfirmPopover from '@/components/confirm-popover/index.vue'

interface HistoryMessage {
  id: string
  bot_id: string
  role: string
  content: unknown
  sender_display_name?: string
  platform?: string
  created_at: string
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const messages = ref<HistoryMessage[]>([])
const loading = ref(false)
const clearing = ref(false)
const hasMore = ref(false)
const PAGE_SIZE = 50

watch(() => props.botId, () => {
  messages.value = []
  loadMessages(true)
}, { immediate: true })

async function loadMessages(reset: boolean) {
  loading.value = true
  try {
    const params: Record<string, string> = { limit: String(PAGE_SIZE) }
    if (!reset && messages.length > 0) {
      params.before = messages.value[0].created_at
    }
    const { data } = await client.get({
      url: '/bots/{bot_id}/messages',
      path: { bot_id: props.botId },
      query: params,
    }) as { data: { items: HistoryMessage[] } }
    const fetched = data.items ?? []
    hasMore.value = fetched.length >= PAGE_SIZE
    if (reset) {
      messages.value = fetched
    } else {
      messages.value = [...fetched, ...messages.value]
    }
  } catch {
    toast.error(t('bots.history.loadFailed'))
  } finally {
    loading.value = false
  }
}

function loadOlder() {
  loadMessages(false)
}

async function handleClear() {
  clearing.value = true
  try {
    await client.delete({
      url: '/bots/{bot_id}/messages',
      path: { bot_id: props.botId },
    })
    messages.value = []
    hasMore.value = false
    toast.success(t('bots.history.clearSuccess'))
  } catch {
    toast.error(t('bots.history.clearFailed'))
  } finally {
    clearing.value = false
  }
}

function extractText(content: unknown): string {
  if (typeof content === 'string') return content
  if (content && typeof content === 'object') {
    if ('text' in (content as Record<string, unknown>)) {
      return String((content as Record<string, unknown>).text ?? '')
    }
    try {
      return JSON.stringify(content, null, 2)
    } catch {
      return String(content)
    }
  }
  return String(content ?? '')
}

function roleBadgeVariant(role: string): 'default' | 'secondary' | 'destructive' {
  if (role === 'assistant') return 'default'
  if (role === 'user') return 'secondary'
  return 'secondary'
}

function formatDate(value: string): string {
  if (!value) return '-'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? '-' : d.toLocaleString()
}
</script>
