<template>
  <div class="max-w-4xl mx-auto space-y-5">
    <!-- Status Panel -->
    <div class="rounded-md border p-4">
      <div class="flex items-center justify-between gap-2">
        <div class="space-y-1">
          <h3 class="text-lg font-semibold">
            <FontAwesomeIcon :icon="['fas', 'flask']" class="mr-1.5 text-purple-500" />
            {{ $t('bots.evolution.title') }}
          </h3>
          <p class="text-sm text-muted-foreground">
            {{ $t('bots.evolution.subtitle') }}
          </p>
        </div>
        <div class="flex items-center gap-3 shrink-0">
          <Badge
            :variant="evolutionEnabled ? 'default' : 'secondary'"
            class="text-xs"
          >
            {{ evolutionEnabled ? $t('common.enabled') : $t('common.disabled') }}
          </Badge>
          <Switch
            :checked="evolutionEnabled"
            @update:checked="handleToggleEvolution"
          />
        </div>
      </div>

      <div
        v-if="evolutionEnabled"
        class="mt-4 flex flex-wrap items-center gap-3"
      >
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <FontAwesomeIcon :icon="['fas', 'clock']" class="text-xs" />
          <span>{{ $t('bots.evolution.reflectionInterval') }}:</span>
          <Badge variant="outline" class="text-xs">
            {{ formatInterval(evolutionInterval) }}
          </Badge>
        </div>
        <Button
          size="sm"
          variant="outline"
          :disabled="triggering || !evolutionHeartbeatId"
          @click="handleTriggerNow"
        >
          <Spinner v-if="triggering" class="mr-1.5" />
          <FontAwesomeIcon
            v-else
            :icon="['fas', 'rotate']"
            class="mr-1.5"
          />
          {{ $t('bots.evolution.triggerNow') }}
        </Button>
      </div>
    </div>

    <!-- Experiments Timeline -->
    <div class="rounded-md border p-4">
      <h4 class="text-sm font-semibold mb-3">
        <FontAwesomeIcon :icon="['fas', 'chart-line']" class="mr-1.5 text-blue-500" />
        {{ $t('bots.evolution.experimentsTitle') }}
      </h4>

      <div
        v-if="loadingExperiments"
        class="flex items-center gap-2 text-sm text-muted-foreground"
      >
        <Spinner />
        <span>{{ $t('common.loading') }}</span>
      </div>

      <div
        v-else-if="experiments.length === 0"
        class="text-sm text-muted-foreground py-4 text-center"
      >
        {{ $t('bots.evolution.noExperiments') }}
      </div>

      <div
        v-else
        class="space-y-3"
      >
        <div
          v-for="(exp, idx) in experiments"
          :key="idx"
          class="border-l-2 pl-4 py-2"
          :class="exp.resultClass"
        >
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium">{{ exp.title }}</span>
            <Badge
              v-if="exp.result"
              :variant="exp.resultVariant"
              class="text-xs"
            >
              {{ exp.result }}
            </Badge>
          </div>
          <div
            v-if="exp.goal"
            class="text-xs text-muted-foreground mt-1"
          >
            <strong>{{ $t('bots.evolution.goal') }}:</strong> {{ exp.goal }}
          </div>
          <div
            v-if="exp.takeaway"
            class="text-xs text-muted-foreground mt-0.5"
          >
            <strong>{{ $t('bots.evolution.takeaway') }}:</strong> {{ exp.takeaway }}
          </div>
        </div>
      </div>
    </div>

    <!-- Persona File Viewer -->
    <div class="rounded-md border p-4">
      <h4 class="text-sm font-semibold mb-3">
        <FontAwesomeIcon :icon="['fas', 'file-lines']" class="mr-1.5 text-green-500" />
        {{ $t('bots.evolution.personaFiles') }}
      </h4>

      <div class="flex gap-2 mb-3 flex-wrap">
        <Button
          v-for="file in personaFiles"
          :key="file"
          size="sm"
          :variant="activeFile === file ? 'default' : 'outline'"
          @click="selectFile(file)"
        >
          {{ file }}
        </Button>
      </div>

      <div
        v-if="loadingFile"
        class="flex items-center gap-2 text-sm text-muted-foreground"
      >
        <Spinner />
        <span>{{ $t('common.loading') }}</span>
      </div>

      <div
        v-else-if="activeFileContent !== null"
        class="rounded-md bg-muted/50 p-3 max-h-80 overflow-auto"
      >
        <pre class="text-xs whitespace-pre-wrap font-mono">{{ activeFileContent }}</pre>
      </div>

      <div
        v-else
        class="text-sm text-muted-foreground text-center py-4"
      >
        {{ $t('bots.evolution.selectFile') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Badge, Button, Spinner, Switch } from '@memoh/ui'
import { client } from '@memoh/sdk/client'
import { toast } from 'vue-sonner'

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const evolutionEnabled = ref(false)
const evolutionInterval = ref(86400)
const evolutionHeartbeatId = ref<string | null>(null)
const triggering = ref(false)
const loadingExperiments = ref(false)

interface Experiment {
  title: string
  goal: string
  result: string
  resultVariant: 'default' | 'secondary' | 'destructive'
  resultClass: string
  takeaway: string
}

const experiments = ref<Experiment[]>([])
const personaFiles = ['IDENTITY.md', 'SOUL.md', 'TOOLS.md', 'EXPERIMENTS.md', 'NOTES.md']
const activeFile = ref<string | null>(null)
const activeFileContent = ref<string | null>(null)
const loadingFile = ref(false)

async function loadEvolutionStatus() {
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/prompts',
      path: { bot_id: props.botId },
    }) as { data: { allow_self_evolution?: boolean } }
    evolutionEnabled.value = data.allow_self_evolution ?? false
  } catch {
    evolutionEnabled.value = false
  }

  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/heartbeat',
      path: { bot_id: props.botId },
    }) as { data: { items?: { id: string; prompt: string; interval_seconds: number; enabled: boolean }[] } }
    const items = data.items ?? []
    const evoConfig = items.find(h => h.prompt.includes('[evolution-reflection]'))
    if (evoConfig) {
      evolutionHeartbeatId.value = evoConfig.id
      evolutionInterval.value = evoConfig.interval_seconds
    }
  } catch {
    // ignore
  }
}

async function loadExperiments() {
  loadingExperiments.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/files/{filename}',
      path: { bot_id: props.botId, filename: 'EXPERIMENTS.md' },
    }) as { data: { content: string } }
    const content = data.content ?? ''
    experiments.value = parseExperiments(content)
  } catch {
    experiments.value = []
  } finally {
    loadingExperiments.value = false
  }
}

function parseExperiments(content: string): Experiment[] {
  const sections = content.split(/^###\s+/gm).filter(s => s.trim())
  return sections.map(section => {
    const lines = section.split('\n')
    const title = lines[0]?.trim() || 'Untitled'
    const goal = extractField(section, 'Goal')
    const method = extractField(section, 'Method')
    const result = extractField(section, 'Result')
    const takeaway = extractField(section, 'Takeaway')

    let resultVariant: 'default' | 'secondary' | 'destructive' = 'secondary'
    let resultClass = 'border-muted-foreground/30'
    if (result.includes('✅') || result.toLowerCase().includes('worked') || result.toLowerCase().includes('success')) {
      resultVariant = 'default'
      resultClass = 'border-green-500/50'
    } else if (result.includes('❌') || result.toLowerCase().includes('fail')) {
      resultVariant = 'destructive'
      resultClass = 'border-red-500/50'
    } else if (result.includes('⚠️') || result.toLowerCase().includes('partial')) {
      resultVariant = 'secondary'
      resultClass = 'border-yellow-500/50'
    }

    return { title, goal, result, resultVariant, resultClass, takeaway }
  }).slice(0, 50)
}

function extractField(text: string, field: string): string {
  const regex = new RegExp(`\\*\\*${field}\\*\\*:\\s*(.+)`, 'i')
  const match = text.match(regex)
  return match ? match[1].trim() : ''
}

async function handleToggleEvolution(checked: boolean) {
  try {
    await client.put({
      url: '/bots/{bot_id}/prompts',
      path: { bot_id: props.botId },
      body: { allow_self_evolution: checked },
    })
    evolutionEnabled.value = checked
    toast.success(checked ? t('bots.evolution.enabled') : t('bots.evolution.disabled'))
    await loadEvolutionStatus()
  } catch {
    toast.error(t('common.error'))
  }
}

async function handleTriggerNow() {
  if (!evolutionHeartbeatId.value) return
  triggering.value = true
  try {
    await client.post({
      url: '/bots/{bot_id}/heartbeat/{id}/trigger',
      path: { bot_id: props.botId, id: evolutionHeartbeatId.value },
    })
    toast.success(t('bots.evolution.triggered'))
  } catch {
    toast.error(t('common.error'))
  } finally {
    triggering.value = false
  }
}

async function selectFile(filename: string) {
  activeFile.value = filename
  loadingFile.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/files/{filename}',
      path: { bot_id: props.botId, filename },
    }) as { data: { content: string } }
    activeFileContent.value = data.content ?? ''
  } catch {
    activeFileContent.value = t('bots.evolution.fileNotFound')
  } finally {
    loadingFile.value = false
  }
}

function formatInterval(seconds: number): string {
  if (seconds >= 86400) return `${Math.round(seconds / 86400)}d`
  if (seconds >= 3600) return `${Math.round(seconds / 3600)}h`
  if (seconds >= 60) return `${Math.round(seconds / 60)}m`
  return `${seconds}s`
}

onMounted(() => {
  loadEvolutionStatus()
  loadExperiments()
})
</script>
