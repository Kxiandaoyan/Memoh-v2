<template>
  <div class="p-6 space-y-6 max-w-7xl mx-auto">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold">{{ $t('logs.title') }}</h1>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('logs.subtitle') }}</p>
      </div>
      <div class="flex gap-2 items-center">
        <select
          v-model="selectedBotId"
          class="px-3 py-1.5 text-sm rounded-md border bg-background"
          @change="loadLogs"
        >
          <option value="">{{ $t('logs.allBots') }}</option>
          <option
            v-for="bot in botList"
            :key="bot.id"
            :value="bot.id"
          >
            {{ bot.display_name || bot.name || bot.id.slice(0, 8) }}
          </option>
        </select>
        <button
          class="px-3 py-1.5 text-xs rounded-md border transition-colors hover:bg-accent"
          @click="refresh"
        >
          <FontAwesomeIcon :icon="['fas', 'rotate']" class="mr-1" />
          {{ $t('common.refresh') }}
        </button>
      </div>
    </div>

    <!-- Stats Bar -->
    <div class="flex gap-4 text-sm">
      <div class="flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-blue-500" />
        <span class="text-muted-foreground">{{ $t('logs.traces') }}:</span>
        <span class="font-medium">{{ traceGroups.length }}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-red-500" />
        <span class="text-muted-foreground">{{ $t('logs.errors') }}:</span>
        <span class="font-medium text-red-500">{{ errorTraceCount }}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-yellow-500" />
        <span class="text-muted-foreground">{{ $t('logs.warnings') }}:</span>
        <span class="font-medium text-yellow-500">{{ warnTraceCount }}</span>
      </div>
    </div>

    <!-- Loading -->
    <div
      v-if="loading"
      class="flex items-center justify-center h-64 text-muted-foreground rounded-xl border bg-card"
    >
      <FontAwesomeIcon :icon="['fas', 'spinner']" class="animate-spin mr-2" />
      {{ $t('common.loading') }}
    </div>

    <!-- Empty -->
    <div
      v-else-if="traceGroups.length === 0"
      class="flex flex-col items-center justify-center h-64 text-muted-foreground rounded-xl border bg-card"
    >
      <FontAwesomeIcon :icon="['fas', 'list-check']" class="text-4xl mb-4 opacity-50" />
      {{ $t('logs.noLogs') }}
    </div>

    <!-- Trace List -->
    <div v-else class="space-y-2">
      <div
        v-for="group in traceGroups"
        :key="group.traceId"
        class="rounded-xl border bg-card overflow-hidden relative"
      >
        <!-- Trace Header (Level 1) -->
        <button
          class="w-full flex items-center gap-3 pl-4 pr-10 py-3 text-left hover:bg-accent/50 transition-colors"
          @click="toggleTrace(group.traceId)"
        >
          <FontAwesomeIcon
            :icon="['fas', 'chevron-right']"
            class="text-xs text-muted-foreground transition-transform shrink-0"
            :class="{ 'rotate-90': expandedTraces.has(group.traceId) }"
          />
          <!-- Status indicator -->
          <span
            class="w-2 h-2 rounded-full shrink-0"
            :class="traceStatusColor(group)"
          />
          <!-- Time -->
          <span class="text-xs text-muted-foreground w-16 shrink-0">
            {{ formatTime(group.startTime) }}
          </span>
          <!-- Bot name -->
          <span
            v-if="!selectedBotId"
            class="text-xs px-1.5 py-0.5 rounded bg-accent text-muted-foreground shrink-0 max-w-24 truncate"
          >
            {{ getBotName(group.botId) }}
          </span>
          <!-- Query preview -->
          <span class="text-sm truncate flex-1 min-w-0">
            {{ group.query || group.traceId.slice(0, 8) }}
          </span>
          <!-- Step count -->
          <span class="text-xs text-muted-foreground shrink-0">
            {{ group.steps.length }} {{ $t('logs.stepsLabel') }}
          </span>
          <!-- Total duration -->
          <span class="text-xs font-mono shrink-0 w-20 text-right" :class="durationColor(group.totalDuration)">
            {{ group.totalDuration > 0 ? formatDuration(group.totalDuration) : '-' }}
          </span>
        </button>
        <!-- Export button -->
        <button
          class="absolute right-2 top-2 p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/60 transition-colors z-10"
          :class="{ 'text-green-500': exportedTraceId === group.traceId }"
          :title="$t('logs.exportTrace')"
          @click.stop="exportTraceToClipboard(group.traceId)"
        >
          <FontAwesomeIcon
            :icon="['fas', exportedTraceId === group.traceId ? 'check' : 'clipboard']"
            class="text-xs"
          />
        </button>

        <!-- Step Details (Level 2) -->
        <div
          v-if="expandedTraces.has(group.traceId)"
          class="border-t"
        >
          <div class="divide-y">
            <div
              v-for="(step, idx) in group.steps"
              :key="step.id"
              class="flex items-start gap-3 px-4 py-2.5 text-sm hover:bg-accent/30 transition-colors"
            >
              <!-- Timeline connector -->
              <div class="flex flex-col items-center pt-1 shrink-0 w-4">
                <span
                  class="w-2 h-2 rounded-full"
                  :class="levelDotClass(step.level)"
                />
                <span
                  v-if="idx < group.steps.length - 1"
                  class="w-px flex-1 bg-border mt-1"
                />
              </div>
              <!-- Time -->
              <span class="text-xs text-muted-foreground w-20 shrink-0 pt-0.5">
                {{ formatTimeDetailed(step.created_at) }}
              </span>
              <!-- Level badge -->
              <span
                class="px-1.5 py-0.5 rounded text-[10px] font-medium shrink-0 w-12 text-center"
                :class="levelClass(step.level)"
              >
                {{ step.level.toUpperCase() }}
              </span>
              <!-- Step label -->
              <span class="text-xs font-medium w-32 shrink-0 pt-0.5 truncate" :title="step.step">
                {{ stepLabels[step.step] || step.step }}
              </span>
              <!-- Message + Data -->
              <div class="flex-1 min-w-0">
                <p
                  v-if="step.message"
                  class="text-sm text-foreground/80 truncate"
                  :class="{ 'text-red-500': step.level === 'error', 'text-yellow-600 dark:text-yellow-400': step.level === 'warn' }"
                  :title="step.message"
                >
                  {{ step.message }}
                </p>
                <div v-if="step.data && Object.keys(step.data).length > 0">
                  <button
                    class="text-[11px] text-muted-foreground hover:text-foreground mt-0.5"
                    @click.stop="toggleData(step.id)"
                  >
                    {{ expandedData.has(step.id) ? $t('logs.hideData') : $t('logs.showData') }}
                  </button>
                  <pre
                    v-if="expandedData.has(step.id)"
                    class="mt-1 p-2 bg-muted rounded text-xs overflow-auto max-h-40 text-foreground/70"
                  >{{ JSON.stringify(step.data, null, 2) }}</pre>
                </div>
              </div>
              <!-- Duration -->
              <span class="text-xs font-mono text-muted-foreground w-16 text-right shrink-0 pt-0.5">
                {{ step.duration_ms ? `${step.duration_ms}ms` : '-' }}
              </span>
            </div>
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
import { exportTrace } from '@/lib/api-logs'

const { t } = useI18n()

interface BotInfo {
  id: string
  display_name?: string
  name?: string
}

interface TraceGroup {
  traceId: string
  botId: string
  startTime: string
  query: string
  steps: ProcessLog[]
  totalDuration: number
  hasError: boolean
  hasWarn: boolean
}

const loading = ref(false)
const logs = ref<ProcessLog[]>([])
const botList = ref<BotInfo[]>([])
const selectedBotId = ref('')
const expandedTraces = ref<Set<string>>(new Set())
const expandedData = ref<Set<string>>(new Set())
const exportedTraceId = ref<string | null>(null)

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
  memory_extract_started: t('logs.steps.memory_extract_started'),
  memory_extract_completed: t('logs.steps.memory_extract_completed'),
  memory_extract_failed: t('logs.steps.memory_extract_failed'),
  token_trimmed: t('logs.steps.token_trimmed'),
  summary_loaded: t('logs.steps.summary_loaded'),
  summary_requested: t('logs.steps.summary_requested'),
  skills_loaded: t('logs.steps.skills_loaded'),
  openviking_context: t('logs.steps.openviking_context'),
  openviking_session: t('logs.steps.openviking_session'),
  evolution_started: t('logs.steps.evolution_started'),
  evolution_completed: t('logs.steps.evolution_completed'),
  evolution_failed: t('logs.steps.evolution_failed'),
}

const traceGroups = computed<TraceGroup[]>(() => {
  const groups = new Map<string, ProcessLog[]>()
  for (const log of logs.value) {
    const key = log.trace_id || log.id
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(log)
  }
  const result: TraceGroup[] = []
  for (const [traceId, steps] of groups) {
    steps.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
    const userMsg = steps.find(s => s.step === 'user_message_received')
    let query = ''
    if (userMsg?.data?.query) {
      query = String(userMsg.data.query)
    } else if (userMsg?.message) {
      query = userMsg.message
    }
    const totalDuration = steps.reduce((sum, s) => sum + (s.duration_ms || 0), 0)
    result.push({
      traceId,
      botId: steps[0]?.bot_id || '',
      startTime: steps[0]?.created_at || '',
      query: query.length > 100 ? query.slice(0, 100) + '...' : query,
      steps,
      totalDuration,
      hasError: steps.some(s => s.level === 'error'),
      hasWarn: steps.some(s => s.level === 'warn'),
    })
  }
  result.sort((a, b) => new Date(b.startTime).getTime() - new Date(a.startTime).getTime())
  return result
})

const errorTraceCount = computed(() => traceGroups.value.filter(g => g.hasError).length)
const warnTraceCount = computed(() => traceGroups.value.filter(g => g.hasWarn && !g.hasError).length)

function traceStatusColor(group: TraceGroup) {
  if (group.hasError) return 'bg-red-500'
  if (group.hasWarn) return 'bg-yellow-500'
  return 'bg-green-500'
}

function durationColor(ms: number) {
  if (ms > 10000) return 'text-red-500'
  if (ms > 5000) return 'text-yellow-500'
  return 'text-muted-foreground'
}

function formatDuration(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`
  return `${ms}ms`
}

function levelClass(level: string) {
  switch (level.toLowerCase()) {
    case 'error': return 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300'
    case 'warn': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-300'
    case 'info': return 'bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300'
    default: return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
  }
}

function levelDotClass(level: string) {
  switch (level.toLowerCase()) {
    case 'error': return 'bg-red-500'
    case 'warn': return 'bg-yellow-500'
    case 'info': return 'bg-blue-400'
    default: return 'bg-gray-400'
  }
}

function formatTime(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatTimeDetailed(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function getBotName(botId: string) {
  const bot = botList.value.find(b => b.id === botId)
  return bot?.display_name || bot?.name || botId.slice(0, 8)
}

function toggleTrace(traceId: string) {
  if (expandedTraces.value.has(traceId)) {
    expandedTraces.value.delete(traceId)
  } else {
    expandedTraces.value.add(traceId)
  }
  expandedTraces.value = new Set(expandedTraces.value)
}

function toggleData(logId: string) {
  if (expandedData.value.has(logId)) {
    expandedData.value.delete(logId)
  } else {
    expandedData.value.add(logId)
  }
  expandedData.value = new Set(expandedData.value)
}

async function exportTraceToClipboard(traceId: string) {
  try {
    const data = await exportTrace(traceId)
    const json = JSON.stringify(data, null, 2)
    await navigator.clipboard.writeText(json)
    exportedTraceId.value = traceId
    setTimeout(() => {
      exportedTraceId.value = null
    }, 2000)
  } catch (e) {
    console.error('Failed to export trace', e)
  }
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
    const params: Record<string, string> = { limit: '500' }
    if (selectedBotId.value) params.botId = selectedBotId.value
    const response = await client.get({ url: '/logs/recent', query: params }) as Promise<{ data: ProcessLog[] }>
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

onMounted(async () => {
  await loadBots()
  await loadLogs()
})
</script>
