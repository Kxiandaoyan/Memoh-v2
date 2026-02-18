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
          <!-- Total (solid) -->
          <path
            :d="series.totalPath"
            fill="none"
            :stroke="palette[idx % palette.length]"
            stroke-width="2"
            stroke-linejoin="round"
          />
          <!-- Prompt / Input (dashed) -->
          <path
            :d="series.promptPath"
            fill="none"
            :stroke="palette[idx % palette.length]"
            stroke-width="1.2"
            stroke-dasharray="4 3"
            stroke-linejoin="round"
            opacity="0.6"
          />
          <!-- Completion / Output (dotted) -->
          <path
            :d="series.completionPath"
            fill="none"
            :stroke="palette[idx % palette.length]"
            stroke-width="1.2"
            stroke-dasharray="1.5 3"
            stroke-linejoin="round"
            opacity="0.6"
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
            <title>{{ series.botName }} ({{ pt.day }})
{{ $t('tokenUsage.totalTokens') }}: {{ formatNumber(pt.total) }}
{{ $t('tokenUsage.promptTokens') }}: {{ formatNumber(pt.prompt) }}
{{ $t('tokenUsage.completionTokens') }}: {{ formatNumber(pt.completion) }}</title>
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

        <!-- Line style legend -->
        <g :transform="`translate(${chartW - pad.right - 260}, ${chartH - 12})`">
          <line x1="0" y1="0" x2="16" y2="0" stroke="currentColor" stroke-width="2" opacity="0.5" />
          <text x="20" y="3" font-size="9" class="fill-muted-foreground">{{ $t('tokenUsage.totalTokens') }}</text>
          <line x1="90" y1="0" x2="106" y2="0" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" opacity="0.5" />
          <text x="110" y="3" font-size="9" class="fill-muted-foreground">{{ $t('tokenUsage.promptTokens') }}</text>
          <line x1="185" y1="0" x2="201" y2="0" stroke="currentColor" stroke-width="1.2" stroke-dasharray="1.5 3" opacity="0.5" />
          <text x="205" y="3" font-size="9" class="fill-muted-foreground">{{ $t('tokenUsage.completionTokens') }}</text>
        </g>
      </svg>
    </div>

    <!-- Model distribution pie chart -->
    <div class="rounded-xl border bg-card p-5">
      <h2 class="text-base font-semibold mb-4">{{ $t('tokenUsage.modelDistribution') }}</h2>
      <div
        v-if="modelRows.length === 0"
        class="flex items-center justify-center h-48 text-muted-foreground"
      >
        {{ $t('tokenUsage.noData') }}
      </div>
      <div v-else class="flex flex-col md:flex-row items-center gap-8">
        <!-- Pie chart SVG -->
        <svg viewBox="0 0 260 260" class="w-56 h-56 shrink-0">
          <g transform="translate(130,130)">
            <path
              v-for="(slice, idx) in pieSlices"
              :key="idx"
              :d="slice.path"
              :fill="palette[idx % palette.length]"
              stroke="var(--card)"
              stroke-width="2"
              class="transition-opacity hover:opacity-80"
            >
              <title>{{ slice.model }}: {{ formatNumber(slice.total) }} ({{ slice.pct }}%)</title>
            </path>
            <!-- Center label -->
            <text text-anchor="middle" dominant-baseline="central" class="fill-foreground font-bold" font-size="14">
              {{ formatNumber(modelGrandTotal) }}
            </text>
          </g>
        </svg>
        <!-- Legend table -->
        <table class="text-sm flex-1 w-full">
          <thead>
            <tr class="border-b text-muted-foreground text-left">
              <th class="pb-2 font-medium">{{ $t('tokenUsage.model') }}</th>
              <th class="pb-2 font-medium text-right">{{ $t('tokenUsage.promptTokens') }}</th>
              <th class="pb-2 font-medium text-right">{{ $t('tokenUsage.completionTokens') }}</th>
              <th class="pb-2 font-medium text-right">{{ $t('tokenUsage.totalTokens') }}</th>
              <th class="pb-2 font-medium text-right">%</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(row, idx) in modelRows"
              :key="row.model"
              class="border-b last:border-0"
            >
              <td class="py-2 flex items-center gap-2">
                <span class="inline-block w-3 h-3 rounded-sm shrink-0" :style="{ background: palette[idx % palette.length] }" />
                {{ row.model || 'unknown' }}
              </td>
              <td class="py-2 text-right tabular-nums">{{ formatNumber(row.prompt_tokens) }}</td>
              <td class="py-2 text-right tabular-nums">{{ formatNumber(row.completion_tokens) }}</td>
              <td class="py-2 text-right tabular-nums font-medium">{{ formatNumber(row.total_tokens) }}</td>
              <td class="py-2 text-right tabular-nums">{{ modelGrandTotal > 0 ? ((row.total_tokens / modelGrandTotal) * 100).toFixed(1) : 0 }}%</td>
            </tr>
          </tbody>
        </table>
      </div>
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
  prompt_tokens: number
  completion_tokens: number
}

interface BotInfo {
  id: string
  display_name?: string
  name?: string
}

interface ModelTotal {
  model: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

const days = ref(30)
const loading = ref(false)
const totalsRows = ref<BotTotal[]>([])
const dailyRows = ref<DailyRow[]>([])
const botList = ref<BotInfo[]>([])
const modelRows = ref<ModelTotal[]>([])

const modelGrandTotal = computed(() =>
  modelRows.value.reduce((s, r) => s + r.total_tokens, 0),
)

const pieSlices = computed(() => {
  const total = modelGrandTotal.value
  if (total === 0) return []
  const r = 110
  let cumAngle = -Math.PI / 2
  return modelRows.value.map((row) => {
    const frac = row.total_tokens / total
    const startAngle = cumAngle
    const endAngle = cumAngle + frac * 2 * Math.PI
    cumAngle = endAngle
    const largeArc = frac > 0.5 ? 1 : 0
    const x1 = r * Math.cos(startAngle)
    const y1 = r * Math.sin(startAngle)
    const x2 = r * Math.cos(endAngle)
    const y2 = r * Math.sin(endAngle)
    const path = modelRows.value.length === 1
      ? `M ${r} 0 A ${r} ${r} 0 1 1 ${r - 0.001} 0 A ${r} ${r} 0 1 1 ${r} 0 Z`
      : `M 0 0 L ${x1.toFixed(2)} ${y1.toFixed(2)} A ${r} ${r} 0 ${largeArc} 1 ${x2.toFixed(2)} ${y2.toFixed(2)} Z`
    return {
      model: row.model || 'unknown',
      total: row.total_tokens,
      pct: (frac * 100).toFixed(1),
      path,
    }
  })
})

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

interface DayData {
  total: number
  prompt: number
  completion: number
}

const chartSeries = computed(() => {
  if (dailyRows.value.length === 0) return []

  const byBot = new Map<string, Map<string, DayData>>()
  for (const r of dailyRows.value) {
    if (!byBot.has(r.bot_id)) byBot.set(r.bot_id, new Map())
    byBot.get(r.bot_id)!.set(r.day, {
      total: r.total_tokens,
      prompt: r.prompt_tokens,
      completion: r.completion_tokens,
    })
  }

  const allDays = [...new Set(dailyRows.value.map((r) => r.day))].sort()
  const maxVal = Math.max(1, ...dailyRows.value.map((r) => r.total_tokens))

  const plotW = chartW - pad.left - pad.right
  const plotH = chartH - pad.top - pad.bottom

  return [...byBot.entries()].map(([botId, dayMap]) => {
    const points = allDays.map((day, i) => {
      const data = dayMap.get(day) ?? { total: 0, prompt: 0, completion: 0 }
      const x = pad.left + (allDays.length > 1 ? (i / (allDays.length - 1)) * plotW : plotW / 2)
      return {
        x,
        y: pad.top + plotH - (data.total / maxVal) * plotH,
        yPrompt: pad.top + plotH - (data.prompt / maxVal) * plotH,
        yCompletion: pad.top + plotH - (data.completion / maxVal) * plotH,
        total: data.total,
        prompt: data.prompt,
        completion: data.completion,
        day,
      }
    })
    const totalPath = points
      .map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`)
      .join(' ')
    const promptPath = points
      .map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)} ${p.yPrompt.toFixed(1)}`)
      .join(' ')
    const completionPath = points
      .map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)} ${p.yCompletion.toFixed(1)}`)
      .join(' ')
    return { botId, botName: getBotName(botId), points, totalPath, promptPath, completionPath }
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
    const [totalsRes, dailyRes, modelRes] = await Promise.all([
      client.get({ url: '/token-usage/all' }) as Promise<{ data: { items: BotTotal[] } }>,
      client.get({ url: '/token-usage/daily', query: { days: days.value } }) as Promise<{ data: { items: DailyRow[] } }>,
      client.get({ url: '/token-usage/by-model' }) as Promise<{ data: { items: ModelTotal[] } }>,
    ])
    totalsRows.value = totalsRes.data?.items ?? []
    dailyRows.value = dailyRes.data?.items ?? []
    modelRows.value = modelRes.data?.items ?? []
  } catch {
    totalsRows.value = []
    dailyRows.value = []
    modelRows.value = []
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
