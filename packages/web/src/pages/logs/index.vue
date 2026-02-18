<template>
  <div class="p-6 space-y-6 max-w-7xl mx-auto">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold">{{ $t('logs.title') }}</h1>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('logs.subtitle') }}</p>
      </div>
      <div class="flex gap-2">
        <button
          class="px-3 py-1.5 text-xs rounded-md border transition-colors hover:bg-accent"
          @click="refresh"
        >
          <FontAwesomeIcon :icon="['fas', 'rotate']" class="mr-1" />
          {{ $t('common.refresh') }}
        </button>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-4 gap-4">
      <div class="lg:col-span-1">
        <div class="rounded-xl border bg-card p-4">
          <div class="space-y-4">
            <div>
              <label class="text-sm font-medium">{{ $t('logs.selectBot') }}</label>
              <select
                v-model="selectedBotId"
                class="w-full mt-1 px-3 py-2 rounded-md border bg-background"
                @change="loadLogs"
              >
                <option value="">{{ $t('common.none') }}</option>
                <option
                  v-for="bot in botList"
                  :key="bot.id"
                  :value="bot.id"
                >
                  {{ bot.display_name || bot.name || bot.id.slice(0, 8) }}
                </option>
              </select>
            </div>

            <div>
              <label class="text-sm font-medium">{{ $t('logs.allSteps') }}</label>
              <select
                v-model="selectedStep"
                class="w-full mt-1 px-3 py-2 rounded-md border bg-background"
                @change="loadLogs"
              >
                <option value="">{{ $t('logs.allSteps') }}</option>
                <option
                  v-for="(label, step) in stepLabels"
                  :key="step"
                  :value="step"
                >
                  {{ label }}
                </option>
              </select>
            </div>

            <div>
              <label class="text-sm font-medium">{{ $t('common.search') }}</label>
              <input
                v-model="searchQuery"
                :placeholder="$t('logs.searchPlaceholder')"
                class="w-full mt-1 px-3 py-2 rounded-md border bg-background"
                @input="debounceSearch"
              >
            </div>
          </div>
        </div>

        <div class="mt-4 rounded-xl border bg-card p-4">
          <h3 class="font-semibold mb-3">{{ $t('logs.title') }}</h3>
          <div class="space-y-2">
            <div class="flex justify-between text-sm">
              <span class="text-muted-foreground">{{ $t('common.loading') }}</span>
              <span>{{ logs.length }}</span>
            </div>
            <div class="flex justify-between text-sm">
              <span class="text-muted-foreground">Error</span>
              <span class="text-red-500">{{ errorCount }}</span>
            </div>
            <div class="flex justify-between text-sm">
              <span class="text-muted-foreground">Warn</span>
              <span class="text-yellow-500">{{ warnCount }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="lg:col-span-3">
        <div class="rounded-xl border bg-card">
          <div
            v-if="loading"
            class="flex items-center justify-center h-96 text-muted-foreground"
          >
            <FontAwesomeIcon
              :icon="['fas', 'spinner']"
              class="animate-spin mr-2"
            />
            {{ $t('common.loading') }}
          </div>

          <div
            v-else-if="filteredLogs.length === 0"
            class="flex flex-col items-center justify-center h-96 text-muted-foreground"
          >
            <FontAwesomeIcon
              :icon="['fas', 'list-check']"
              class="text-4xl mb-4 opacity-50"
            />
            {{ $t('logs.noLogs') }}
          </div>

          <div
            v-else
            class="overflow-auto"
            style="max-height: calc(100vh - 200px)"
          >
            <table class="w-full text-sm">
              <thead class="sticky top-0 bg-card border-b">
                <tr class="text-muted-foreground text-left">
                  <th class="px-4 py-3 font-medium w-32">{{ $t('common.createdAt') }}</th>
                  <th class="px-4 py-3 font-medium w-24">Level</th>
                  <th class="px-4 py-3 font-medium w-40">Step</th>
                  <th class="px-4 py-3 font-medium">Message</th>
                  <th class="px-4 py-3 font-medium w-24 text-right">Duration</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="log in filteredLogs"
                  :key="log.id"
                  class="border-b hover:bg-accent/50 transition-colors"
                >
                  <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                    {{ formatTime(log.created_at) }}
                  </td>
                  <td class="px-4 py-3">
                    <span
                      class="px-2 py-1 rounded text-xs font-medium"
                      :class="levelClass(log.level)"
                    >
                      {{ log.level.toUpperCase() }}
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <span class="truncate" :title="log.step">
                      {{ stepLabels[log.step] || log.step }}
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="max-w-md">
                      <p class="truncate" :title="log.message">
                        {{ log.message }}
                      </p>
                      <div
                        v-if="log.data"
                        class="mt-1"
                      >
                        <button
                          class="text-xs text-muted-foreground hover:text-foreground"
                          @click="toggleData(log.id)"
                        >
                          {{ expandedData.has(log.id) ? '隐藏数据' : '显示数据' }}
                        </button>
                        <pre
                          v-if="expandedData.has(log.id)"
                          class="mt-1 p-2 bg-muted rounded text-xs overflow-auto max-h-40"
                        >{{ JSON.stringify(log.data, null, 2) }}</pre>
                      </div>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-right text-muted-foreground">
                    {{ log.duration_ms ? `${log.duration_ms}ms` : '-' }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import type { ProcessLog } from '@/lib/api-logs'

const { t } = useI18n()

interface BotInfo {
  id: string
  display_name?: string
  name?: string
}

const loading = ref(false)
const logs = ref<ProcessLog[]>([])
const botList = ref<BotInfo[]>([])
const selectedBotId = ref('')
const selectedStep = ref('')
const searchQuery = ref('')
const expandedData = ref<Set<string>>(new Set())

let searchTimeout: ReturnType<typeof setTimeout> | null = null

const stepLabels: Record<string, string> = {
  user_message_received: t('logs.steps.user_message_received'),
  history_loaded: t('logs.steps.history_loaded'),
  memory_searched: t('logs.steps.memory_searched'),
  memory_loaded: t('logs.steps.memory_loaded'),
  prompt_built: t('logs.steps.prompt_built'),
  llm_request_sent: t('logs.steps.llm_request_sent'),
  llm_response_received: t('logs.steps.llm_response_received'),
  tool_call_started: t('logs.steps.tool_call_started'),
  tool_call_completed: t('logs.steps.tool_call_completed'),
  response_sent: t('logs.steps.response_sent'),
  memory_stored: t('logs.steps.memory_stored'),
  stream_started: t('logs.steps.stream_started'),
  stream_completed: t('logs.steps.stream_completed'),
  stream_error: t('logs.steps.stream_error'),
}

const filteredLogs = computed(() => {
  let result = logs.value

  if (selectedStep.value) {
    result = result.filter(log => log.step === selectedStep.value)
  }

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(log =>
      log.message?.toLowerCase().includes(q) ||
      log.step.toLowerCase().includes(q) ||
      log.trace_id?.toLowerCase().includes(q)
    )
  }

  return result
})

const errorCount = computed(() => logs.value.filter(l => l.level === 'error').length)
const warnCount = computed(() => logs.value.filter(l => l.level === 'warn').length)

function levelClass(level: string) {
  switch (level.toLowerCase()) {
    case 'error':
      return 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300'
    case 'warn':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-300'
    case 'info':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
  }
}

function formatTime(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleTimeString()
}

async function loadBots() {
  try {
    const { data } = await client.get({ url: '/bots' }) as { data: { items: BotInfo[] } }
    botList.value = data?.items ?? data ?? []
  } catch {
    botList.value = []
  }
}

async function loadLogs() {
  loading.value = true
  try {
    let url = '/logs/recent'
    const params: Record<string, string> = {}
    if (selectedBotId.value) {
      params.botId = selectedBotId.value
    }
    params.limit = '500'
    const response = await client.get({ url, query: params }) as Promise<{ data: ProcessLog[] }>
    logs.value = (response as any).json ? await (response as any).json() : (response as any).data || []
  } catch (e) {
    console.error('Failed to load logs', e)
    logs.value = []
  } finally {
    loading.value = false
  }
}

function refresh() {
  loadLogs()
}

function toggleData(logId: string) {
  if (expandedData.value.has(logId)) {
    expandedData.value.delete(logId)
  } else {
    expandedData.value.add(logId)
  }
  expandedData.value = new Set(expandedData.value)
}

function debounceSearch() {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  searchTimeout = setTimeout(() => {
  }, 300)
}

onMounted(async () => {
  await loadBots()
  await loadLogs()
})
</script>
