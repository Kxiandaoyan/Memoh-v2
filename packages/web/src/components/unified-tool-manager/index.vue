<template>
  <div class="unified-tool-manager max-w-4xl mx-auto space-y-5">
    <!-- Header -->
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('toolManager.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('toolManager.subtitle') }}
        </p>
      </div>
      <div class="flex gap-2 shrink-0">
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="resetToDefaults"
        >
          {{ $t('toolManager.resetToDefaults') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadTools"
        >
          <Spinner v-if="loading" class="mr-1.5" />
          {{ $t('common.refresh') }}
        </Button>
        <Button
          size="sm"
          :disabled="!hasChanges || saving"
          @click="saveConfiguration"
        >
          <Spinner v-if="saving" class="mr-1.5" />
          {{ $t('common.save') }}
        </Button>
      </div>
    </div>

    <!-- Category Filters -->
    <div class="flex flex-wrap gap-2">
      <Button
        v-for="category in categories"
        :key="category.value"
        variant="outline"
        size="sm"
        :class="{
          'border-primary bg-primary/10': selectedCategory === category.value
        }"
        @click="filterByCategory(category.value)"
      >
        <span class="mr-1.5">{{ category.icon }}</span>
        {{ category.label }}
      </Button>
    </div>

    <!-- Loading State -->
    <div
      v-if="loading && tools.length === 0"
      class="flex items-center gap-2 text-sm text-muted-foreground"
    >
      <Spinner />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty State -->
    <div
      v-else-if="filteredTools.length === 0"
      class="rounded-md border p-6 text-center text-sm text-muted-foreground"
    >
      {{ selectedCategory === '' ? $t('toolManager.empty') : $t('toolManager.noToolsInCategory') }}
    </div>

    <!-- Tools List -->
    <div v-else class="space-y-3">
      <div
        v-for="(tool, index) in filteredTools"
        :key="tool.name"
        class="rounded-md border bg-card transition-all"
        :class="{
          'opacity-50': !tool.enabled,
          'ring-2 ring-primary': dragOverIndex === index,
          'cursor-move': !tool.dragging,
          'opacity-60 scale-95': tool.dragging
        }"
        draggable="true"
        @dragstart="handleDragStart(index, $event)"
        @dragend="handleDragEnd"
        @dragover.prevent="handleDragOver(index)"
        @drop.prevent="handleDrop(index)"
        @dragleave="handleDragLeave"
      >
        <div class="p-4 flex items-center gap-3">
          <!-- Drag Handle -->
          <div class="flex items-center gap-2 shrink-0">
            <div class="cursor-move text-muted-foreground hover:text-foreground transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="size-4">
                <line x1="3" y1="12" x2="21" y2="12" />
                <line x1="3" y1="6" x2="21" y2="6" />
                <line x1="3" y1="18" x2="21" y2="18" />
              </svg>
            </div>
            <Badge variant="outline" class="font-mono text-[10px] shrink-0">
              {{ getOriginalIndex(tool) + 1 }}
            </Badge>
          </div>

          <!-- Tool Icon & Info -->
          <div class="flex-1 min-w-0 flex items-center gap-3">
            <div class="text-2xl shrink-0" :title="tool.category">
              {{ getCategoryIcon(tool.category) }}
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="font-mono text-sm font-medium truncate">
                  {{ tool.name }}
                </p>
                <Badge
                  v-if="tool.type === 'mcp'"
                  variant="secondary"
                  class="text-[10px]"
                >
                  {{ $t('toolManager.external') }}
                </Badge>
                <Badge
                  v-else
                  variant="outline"
                  class="text-[10px]"
                >
                  {{ $t('toolManager.builtin') }}
                </Badge>
              </div>
              <p class="text-xs text-muted-foreground mt-0.5">
                {{ tool.category }}
              </p>
            </div>
          </div>

          <!-- Enable/Disable Switch -->
          <div class="flex items-center gap-3 shrink-0">
            <div class="flex items-center gap-2">
              <Label
                :for="`tool-${index}`"
                class="text-xs text-muted-foreground cursor-pointer"
              >
                {{ tool.enabled ? $t('toolManager.enabled') : $t('toolManager.disabled') }}
              </Label>
              <Switch
                :id="`tool-${index}`"
                :checked="tool.enabled"
                @update:checked="toggleTool(index)"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Change Indicator -->
    <div
      v-if="hasChanges"
      class="rounded-md border border-orange-500/50 bg-orange-500/10 p-3 text-sm text-orange-600 dark:text-orange-400"
    >
      <div class="flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="size-4 shrink-0">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        <span>{{ $t('toolManager.unsavedChanges') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Badge,
  Button,
  Label,
  Spinner,
  Switch,
} from '@memoh/ui'
import { ref, computed, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

interface ToolItem {
  name: string
  category: string
  type: 'builtin' | 'mcp'
  enabled: boolean
  order: number
  dragging?: boolean
  mcpConnectionName?: string
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const tools = ref<ToolItem[]>([])
const originalTools = ref<ToolItem[]>([])
const loading = ref(false)
const saving = ref(false)
const draggedIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)
const selectedCategory = ref<string>('')

const categories = [
  { value: '', label: 'All', icon: '📋' },
  { value: 'file', label: 'File', icon: '📁' },
  { value: 'message', label: 'Message', icon: '💬' },
  { value: 'memory', label: 'Memory', icon: '🧠' },
  { value: 'web', label: 'Web', icon: '🌐' },
  { value: 'schedule', label: 'Schedule', icon: '⏰' },
  { value: 'directory', label: 'Directory', icon: '📖' },
  { value: 'shell', label: 'Shell', icon: '⚡' },
  { value: 'image', label: 'Image', icon: '🖼️' },
  { value: 'skills', label: 'Skills', icon: '🎯' },
  { value: 'team', label: 'Team', icon: '👥' },
  { value: 'subagent', label: 'Subagent', icon: '🤖' },
  { value: 'openviking', label: 'OpenViking', icon: '🗄️' },
  { value: 'admin', label: 'Admin', icon: '⚙️' },
  { value: 'mcp', label: 'MCP', icon: '🔌' },
]

const toolCategoryMap: Record<string, string> = {
  read: 'file', write: 'file', list: 'file', edit: 'file',
  exec: 'shell',
  web_search: 'web', web_fetch: 'web',
  search_memory: 'memory', query_history: 'memory',
  send: 'message', react: 'message',
  lookup_channel_user: 'directory',
  generate_image: 'image',
  create_schedule: 'schedule', list_schedule: 'schedule', get_schedule: 'schedule', update_schedule: 'schedule', delete_schedule: 'schedule',
  use_skill: 'skills', discover_skills: 'skills', fork_skill: 'skills',
  call_agent: 'team',
  list_subagents: 'subagent', create_subagent: 'subagent', delete_subagent: 'subagent', query_subagent: 'subagent', spawn_subagent: 'subagent', check_subagent_run: 'subagent', kill_subagent_run: 'subagent', steer_subagent: 'subagent', list_subagent_runs: 'subagent',
  ov_initialize: 'openviking', ov_find: 'openviking', ov_search: 'openviking', ov_read: 'openviking', ov_abstract: 'openviking', ov_overview: 'openviking', ov_ls: 'openviking', ov_tree: 'openviking', ov_add_resource: 'openviking', ov_rm: 'openviking', ov_session_commit: 'openviking',
  admin_list_bots: 'admin', admin_create_bot: 'admin', admin_delete_bot: 'admin', admin_list_models: 'admin', admin_create_model: 'admin', admin_delete_model: 'admin', admin_list_providers: 'admin', admin_create_provider: 'admin', admin_update_provider: 'admin',
}

const filteredTools = computed(() => {
  if (selectedCategory.value === '') return tools.value
  return tools.value.filter(tool => tool.category === selectedCategory.value)
})

const hasChanges = computed(() => {
  if (tools.value.length !== originalTools.value.length) return true
  return tools.value.some((tool, index) => {
    const original = originalTools.value[index]
    return (
      tool.name !== original?.name ||
      tool.enabled !== original?.enabled ||
      tool.order !== original?.order
    )
  })
})

watch(() => props.botId, () => { loadTools() }, { immediate: true })

function getCategoryIcon(category: string): string {
  return categories.find(c => c.value === category)?.icon || '🔧'
}

function getOriginalIndex(tool: ToolItem): number {
  return tools.value.indexOf(tool)
}

function filterByCategory(category: string) {
  selectedCategory.value = category
}

async function loadTools() {
  loading.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/tools',
      path: { bot_id: props.botId },
    }) as { data: { tools: Array<Partial<ToolItem>> } }

    const loadedTools = (data.tools ?? []).map((tool, index) => {
      const toolCategory = tool.type === 'mcp' ? 'mcp' : (toolCategoryMap[tool.name || ''] || 'other')
      return {
        name: tool.name || '',
        category: toolCategory,
        type: (tool.type as 'builtin' | 'mcp') || 'builtin',
        enabled: tool.enabled ?? true,
        order: tool.order ?? index,
        dragging: false,
        mcpConnectionName: tool.mcpConnectionName,
      }
    })

    tools.value = loadedTools
    originalTools.value = JSON.parse(JSON.stringify(loadedTools))
  } catch (error) {
    toast.error(t('toolManager.loadFailed'))
    console.error('Failed to load tools:', error)
  } finally {
    loading.value = false
  }
}

function toggleTool(index: number) {
  const actualIndex = tools.value.indexOf(filteredTools.value[index])
  if (actualIndex !== -1) {
    tools.value[actualIndex].enabled = !tools.value[actualIndex].enabled
  }
}

function handleDragStart(index: number, event: DragEvent) {
  const actualIndex = tools.value.indexOf(filteredTools.value[index])
  draggedIndex.value = actualIndex
  tools.value[actualIndex].dragging = true
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/html', actualIndex.toString())
  }
}

function handleDragEnd() {
  if (draggedIndex.value !== null) {
    tools.value[draggedIndex.value].dragging = false
  }
  draggedIndex.value = null
  dragOverIndex.value = null
}

function handleDragOver(index: number) {
  const actualIndex = tools.value.indexOf(filteredTools.value[index])
  if (draggedIndex.value === null || draggedIndex.value === actualIndex) return
  dragOverIndex.value = index
}

function handleDragLeave() {
  dragOverIndex.value = null
}

function handleDrop(targetIndex: number) {
  const actualTargetIndex = tools.value.indexOf(filteredTools.value[targetIndex])
  if (draggedIndex.value === null || draggedIndex.value === actualTargetIndex) {
    dragOverIndex.value = null
    return
  }
  const draggedTool = tools.value[draggedIndex.value]
  const newTools = [...tools.value]
  newTools.splice(draggedIndex.value, 1)
  newTools.splice(actualTargetIndex, 0, draggedTool)
  newTools.forEach((tool, index) => {
    tool.order = index
    tool.dragging = false
  })
  tools.value = newTools
  draggedIndex.value = null
  dragOverIndex.value = null
}

async function saveConfiguration() {
  saving.value = true
  try {
    const builtinTools = tools.value
      .filter(tool => tool.type === 'builtin')
      .map((tool, index) => ({
        tool_name: tool.name,
        category: tool.category || 'other',
        enabled: tool.enabled,
        priority: index,
      }))

    await client.put({
      url: '/bots/{bot_id}/tools/builtin',
      path: { bot_id: props.botId },
      body: { tools: builtinTools },
    })

    toast.success(t('toolManager.saveSuccess'))
    originalTools.value = JSON.parse(JSON.stringify(tools.value))
  } catch (error) {
    toast.error(t('toolManager.saveFailed'))
    console.error('Failed to save tools configuration:', error)
  } finally {
    saving.value = false
  }
}

async function resetToDefaults() {
  if (!confirm(t('toolManager.confirmReset'))) return
  loading.value = true
  try {
    await client.post({
      url: '/bots/{bot_id}/tools/reset',
      path: { bot_id: props.botId },
    })
    toast.success(t('toolManager.resetSuccess'))
    await loadTools()
  } catch (error) {
    toast.error(t('toolManager.resetFailed'))
    console.error('Failed to reset tools:', error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.unified-tool-manager {
  user-select: none;
}
</style>
