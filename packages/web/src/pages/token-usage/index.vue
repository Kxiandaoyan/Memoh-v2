<template>
  <div class="p-6 space-y-6 max-w-6xl mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold">{{ $t('tokenUsage.title') }}</h1>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('tokenUsage.subtitle') }}</p>
      </div>
      <div class="flex gap-2">
        <button
          v-for="opt in rangeOptions"
          :key="opt.days"
          class="px-3 py-1.5 text-xs rounded-md border transition-colors"
          :class="days === opt.days
            ? 'bg-primary text-primary-foreground border-primary'
            : 'bg-background border-border hover:bg-accent'"
          @click="days = opt.days"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <!-- Summary cards -->
    <div class="grid grid-cols-3 gap-4">
      <div
        v-for="card in summaryCards"
        :key="card.label"
        class="rounded-xl border bg-card p-4"
      >
        <p class="text-xs text-muted-foreground">{{ card.label }}</p>
        <p class="text-2xl font-bold mt-1">{{ formatNumber(card.value) }}</p>
      </div>
    </div>

    <!-- Chart area -->
    <div class="rounded-xl border bg-card p-5">
      <h2 class="text-base font-semibold mb-4">{{ $t('tokenUsage.botComparison') }}</h2>

      <div
        v-if="loading"
        class="flex items-center justify-center h-64 text-muted-foreground"
      >
        <FontAwesomeIcon
          :icon="['fas', 'spinner']"
          class="animate-spin mr-2"
        />
        Loading...
      </div>

      <div
        v-else-if="dailyRows.length === 0"
        class="flex items-center justify-center h-64 text-muted-foreground"
      >
        {{ $t('tokenUsage.noData') }}
      </div>

      <svg
        v-else
        ref="chartEl"
        :viewBox="`0 0 ${chartW} ${chartH}`"
        class="w-full"
        preserveAspectRatio="xMidYMid meet"
      >
        <!-- Grid lines -->
        <line
          v-for="y in gridLines"
          :key="y.val"
          :x1="pad.left"
          :y1="y.y"
          :x2="chartW - pad.right"
          :y2="y.y"
          stroke="currentColor"
          stroke-opacity="0.06"
        />
        <!-- Y-axis labels -->
        <text
          v-for="y in gridLines"
          :key="'lbl-' + y.val"
          :x="pad.left - 8"
          :y="y.y + 3"
          text-anchor="end"
          class="fill-muted-foreground"
          font-size="10"
        >
          {{ formatAxisLabel(y.val) }}
        </text>

        <!-- X-axis labels -->
        <text
          v-for="(lbl, idx) in xLabels"
          :key="'x-' + idx"
          :x="lbl.x"
          :y="chartH - pad.bottom + 14"
          text-anchor="middle"
          class="fill-muted-foreground"
          font-size="9"
        >
          {{ lbl.text }}
        </text>

        <!-- Lines per bot -->
        <g
          v-for="(series, idx) in chartSeries"
          :key="series.botId"
        >
          <path
            :d="series.path"
            fill="none"
            :stroke="palette[idx % palette.length]"
            stroke-width="2"
            stroke-linejoin="round"
          />
          <circle
            v-for="(pt, pi) in series.points"
            :key="pi"
            :cx="pt.x"
            :cy="pt.y"
            r="3"
            :fill="palette[idx % palette.length]"
            class="opacity-70"
          >
            <title>{{ series.botName }}: {{ formatNumber(pt.val) }} tokens ({{ pt.day }})</title>
          </circle>
        </g>

        <!-- Legend -->
        <g
          v-for="(series, idx) in chartSeries"
          :key="'leg-' + series.botId"
          :transform="`translate(${pad.left + idx * 140}, 16)`"
        >
          <rect
            width="10"
            height="10"
            rx="2"
            :fill="palette[idx % palette.length]"
          />
          <text
            x="14"
            y="9"
            font-size="11"
            class="fill-foreground"
          >
            {{ series.botName }}
          </text>
        </g>
      </svg>
    </div>

    <!-- Per-bot table -->
    <div class="rounded-xl border bg-card p-5">
      <h2 class="text-base font-semibold mb-3">{{ $t('tokenUsage.title') }}</h2>
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b text-muted-foreground text-left">
            <th class="pb-2 font-medium">Bot</th>
            <th class="pb-2 font-medium text-right">{{ $t('tokenUsage.promptTokens') }}</th>
            <th class="pb-2 font-medium text-right">{{ $t('tokenUsage.completionTokens') }}</th>
            <th class="pb-2 font-medium text-right">{{ $t('tokenUsage.totalTokens') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in totalsRows"
            :key="row.bot_id"
            class="border-b last:border-0"
          >
            <td class="py-2">{{ getBotName(row.bot_id) }}</td>
            <td class="py-2 text-right tabular-nums">{{ formatNumber(row.prompt_tokens) }}</td>
            <td class="py-2 text-right tabular-nums">{{ formatNumber(row.completion_tokens) }}</td>
            <td class="py-2 text-right tabular-nums font-medium">{{ formatNumber(row.total_tokens) }}</td>
          </tr>
          <tr v-if="totalsRows.length === 0">
            <td
              colspan="4"
              class="py-6 text-center text-muted-foreground"
            >
              {{ $t('tokenUsage.noData') }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

const { t } = useI18n()

interface BotTotal {
  bot_id: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

interface DailyRow {
  bot_id: string
  day: string
  total_tokens: number
}

interface BotInfo {
  id: string
  display_name?: string
  name?: string
}

const days = ref(30)
const loading = ref(false)
const totalsRows = ref<BotTotal[]>([])
const dailyRows = ref<DailyRow[]>([])
const botList = ref<BotInfo[]>([])

const rangeOptions = computed(() => [
  { days: 7, label: t('tokenUsage.last7days') },
  { days: 30, label: t('tokenUsage.last30days') },
  { days: 90, label: t('tokenUsage.last90days') },
])

const summaryCards = computed(() => {
  const total = totalsRows.value.reduce((s, r) => s + r.total_tokens, 0)
  const prompt = totalsRows.value.reduce((s, r) => s + r.prompt_tokens, 0)
  const completion = totalsRows.value.reduce((s, r) => s + r.completion_tokens, 0)
  return [
    { label: t('tokenUsage.totalTokens'), value: total },
    { label: t('tokenUsage.promptTokens'), value: prompt },
    { label: t('tokenUsage.completionTokens'), value: completion },
  ]
})

const palette = [
  '#6366f1', '#f59e0b', '#10b981', '#ef4444', '#8b5cf6',
  '#ec4899', '#14b8a6', '#f97316', '#3b82f6', '#84cc16',
]

const chartW = 800
const chartH = 320
const pad = { top: 30, right: 20, bottom: 30, left: 55 }

const chartSeries = computed(() => {
  if (dailyRows.value.length === 0) return []

  const byBot = new Map<string, Map<string, number>>()
  for (const r of dailyRows.value) {
    if (!byBot.has(r.bot_id)) byBot.set(r.bot_id, new Map())
    byBot.get(r.bot_id)!.set(r.day, r.total_tokens)
  }

  const allDays = [...new Set(dailyRows.value.map((r) => r.day))].sort()
  const maxVal = Math.max(1, ...dailyRows.value.map((r) => r.total_tokens))

  const plotW = chartW - pad.left - pad.right
  const plotH = chartH - pad.top - pad.bottom

  return [...byBot.entries()].map(([botId, dayMap]) => {
    const points = allDays.map((day, i) => {
      const val = dayMap.get(day) ?? 0
      return {
        x: pad.left + (allDays.length > 1 ? (i / (allDays.length - 1)) * plotW : plotW / 2),
        y: pad.top + plotH - (val / maxVal) * plotH,
        val,
        day,
      }
    })
    const path = points
      .map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`)
      .join(' ')
    return { botId, botName: getBotName(botId), points, path }
  })
})

const gridLines = computed(() => {
  const maxVal = Math.max(1, ...dailyRows.value.map((r) => r.total_tokens))
  const plotH = chartH - pad.top - pad.bottom
  const steps = 5
  return Array.from({ length: steps + 1 }, (_, i) => {
    const val = Math.round((maxVal / steps) * (steps - i))
    const y = pad.top + (i / steps) * plotH
    return { val, y }
  })
})

const xLabels = computed(() => {
  const allDays = [...new Set(dailyRows.value.map((r) => r.day))].sort()
  if (allDays.length === 0) return []
  const plotW = chartW - pad.left - pad.right
  const step = Math.max(1, Math.ceil(allDays.length / 8))
  return allDays
    .filter((_, i) => i % step === 0 || i === allDays.length - 1)
    .map((day) => {
      const i = allDays.indexOf(day)
      return {
        x: pad.left + (allDays.length > 1 ? (i / (allDays.length - 1)) * plotW : plotW / 2),
        text: day.slice(5),
      }
    })
})

function getBotName(id: string): string {
  const bot = botList.value.find((b) => b.id === id)
  return bot?.display_name || bot?.name || id.slice(0, 8)
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`
  return n.toLocaleString()
}

function formatAxisLabel(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}k`
  return String(n)
}

async function loadBots() {
  try {
    const { data } = await client.get({ url: '/bots' }) as { data: { items: BotInfo[] } }
    botList.value = data?.items ?? data ?? []
  } catch {
    botList.value = []
  }
}

async function loadData() {
  loading.value = true
  try {
    const [totalsRes, dailyRes] = await Promise.all([
      client.get({ url: '/token-usage/all' }) as Promise<{ data: { items: BotTotal[] } }>,
      client.get({ url: '/token-usage/daily', query: { days: days.value } }) as Promise<{ data: { items: DailyRow[] } }>,
    ])
    totalsRows.value = totalsRes.data?.items ?? []
    dailyRows.value = dailyRes.data?.items ?? []
  } catch {
    totalsRows.value = []
    dailyRows.value = []
  } finally {
    loading.value = false
  }
}

watch(days, () => loadData())

onMounted(async () => {
  await loadBots()
  await loadData()
})
</script>
