<template>
  <div class="skill-manager max-w-4xl mx-auto space-y-5">
    <!-- Header -->
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('skillManager.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('skillManager.subtitle') }}
        </p>
      </div>
      <div class="flex gap-2 shrink-0">
        <Button
          size="sm"
          @click="showCreateDialog = true"
        >
          {{ $t('common.add') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="loading || skills.length === 0"
          @click="enableAll"
        >
          {{ $t('skillManager.enableAll') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="loading || skills.length === 0"
          @click="disableAll"
        >
          {{ $t('skillManager.disableAll') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="syncing"
          @click="syncDefaultSkills"
        >
          <Spinner v-if="syncing" class="mr-1.5" />
          {{ $t('skillManager.syncNewSkills') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadSkills"
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

    <!-- Loading State -->
    <div
      v-if="loading && skills.length === 0"
      class="flex items-center gap-2 text-sm text-muted-foreground"
    >
      <Spinner />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty State -->
    <div
      v-else-if="skills.length === 0"
      class="rounded-md border p-6 text-center text-sm text-muted-foreground"
    >
      {{ $t('skillManager.empty') }}
    </div>

    <!-- Skills List -->
    <div v-else class="space-y-3">
      <div
        v-for="(skill, index) in skills"
        :key="skill.name"
        class="rounded-md border bg-card transition-all"
        :class="{
          'opacity-50': !skill.enabled,
          'ring-2 ring-primary': dragOverIndex === index,
          'cursor-move': !skill.dragging,
          'opacity-60 scale-95': skill.dragging
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
              {{ index + 1 }}
            </Badge>
          </div>

          <!-- Skill Info -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
              <p class="font-mono text-sm font-medium truncate">
                {{ skill.name }}
              </p>
              <Badge v-if="skill.metadata?.verified" variant="default" class="text-[10px]">
                {{ $t('skillManager.verified') }}
              </Badge>
            </div>
            <p v-if="skill.description" class="text-sm text-muted-foreground truncate mt-0.5">
              {{ skill.description }}
            </p>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-3 shrink-0">
            <Button
              v-if="skill.content"
              variant="ghost"
              size="sm"
              class="h-7 px-2 text-xs"
              @click.stop="expandedSkill = expandedSkill === skill.name ? null : skill.name"
            >
              {{ expandedSkill === skill.name ? '▲' : '▼' }}
              SKILL.md
            </Button>
            <div class="flex items-center gap-2">
              <Label :for="`skill-${index}`" class="text-xs text-muted-foreground cursor-pointer">
                {{ skill.enabled ? $t('skillManager.enabled') : $t('skillManager.disabled') }}
              </Label>
              <Switch
                :id="`skill-${index}`"
                :checked="skill.enabled"
                @update:checked="toggleSkill(index)"
              />
            </div>
          </div>
        </div>
        <!-- Expanded SKILL.md Content -->
        <div v-if="expandedSkill === skill.name && skill.content" class="border-t px-4 py-3 bg-muted/30">
          <pre class="text-xs font-mono whitespace-pre-wrap break-words max-h-80 overflow-y-auto">{{ skill.content }}</pre>
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
        <span>{{ $t('skillManager.unsavedChanges') }}</span>
      </div>
    </div>

    <!-- Create Skill Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t('skillManager.createSkill') }}</DialogTitle>
          <DialogDescription>{{ $t('skillManager.createSkillDesc') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div class="space-y-2">
            <Label>{{ $t('skillManager.skillName') }}</Label>
            <Input v-model="newSkill.name" :placeholder="$t('skillManager.skillNamePlaceholder')" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('skillManager.skillDescription') }}</Label>
            <Input v-model="newSkill.description" :placeholder="$t('skillManager.skillDescPlaceholder')" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('skillManager.skillContent') }}</Label>
            <Textarea
              v-model="newSkill.content"
              :placeholder="$t('skillManager.skillContentPlaceholder')"
              rows="6"
              class="font-mono text-sm"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showCreateDialog = false">
            {{ $t('common.cancel') }}
          </Button>
          <Button :disabled="!newSkill.name.trim() || creating" @click="createSkill">
            <Spinner v-if="creating" class="mr-1.5" />
            {{ $t('common.create') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Spinner,
  Switch,
  Textarea,
} from '@memoh/ui'
import { ref, computed, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

interface SkillItem {
  name: string
  description: string
  content: string
  enabled: boolean
  order: number
  dragging?: boolean
  metadata?: {
    verified?: boolean
    [key: string]: unknown
  }
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const skills = ref<SkillItem[]>([])
const originalSkills = ref<SkillItem[]>([])
const loading = ref(false)
const saving = ref(false)
const syncing = ref(false)
const draggedIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)
const expandedSkill = ref<string | null>(null)
const showCreateDialog = ref(false)
const creating = ref(false)
const newSkill = ref({ name: '', description: '', content: '' })

const hasChanges = computed(() => {
  if (skills.value.length !== originalSkills.value.length) return true
  return skills.value.some((skill, index) => {
    const original = originalSkills.value[index]
    return (
      skill.name !== original?.name ||
      skill.enabled !== original?.enabled ||
      skill.order !== original?.order
    )
  })
})

watch(() => props.botId, () => {
  loadSkills()
}, { immediate: true })

async function loadSkills() {
  loading.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/container/skills',
      path: { bot_id: props.botId },
    }) as { data: { skills: Array<Partial<SkillItem>> } }

    const loadedSkills = (data.skills ?? []).map((skill, index) => ({
      ...skill,
      enabled: skill.enabled ?? true,
      order: skill.order ?? index,
      dragging: false,
    }))

    skills.value = loadedSkills
    originalSkills.value = JSON.parse(JSON.stringify(loadedSkills))
  } catch (error) {
    toast.error(t('skillManager.loadFailed'))
    console.error('Failed to load skills:', error)
  } finally {
    loading.value = false
  }
}

function toggleSkill(index: number) {
  skills.value[index].enabled = !skills.value[index].enabled
}

function enableAll() {
  skills.value.forEach((skill) => { skill.enabled = true })
}

function disableAll() {
  skills.value.forEach((skill) => { skill.enabled = false })
}

function handleDragStart(index: number, event: DragEvent) {
  draggedIndex.value = index
  skills.value[index].dragging = true
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/html', index.toString())
  }
}

function handleDragEnd() {
  if (draggedIndex.value !== null) {
    skills.value[draggedIndex.value].dragging = false
  }
  draggedIndex.value = null
  dragOverIndex.value = null
}

function handleDragOver(index: number) {
  if (draggedIndex.value === null || draggedIndex.value === index) return
  dragOverIndex.value = index
}

function handleDragLeave() {
  dragOverIndex.value = null
}

function handleDrop(targetIndex: number) {
  if (draggedIndex.value === null || draggedIndex.value === targetIndex) {
    dragOverIndex.value = null
    return
  }
  const draggedSkill = skills.value[draggedIndex.value]
  const newSkills = [...skills.value]
  newSkills.splice(draggedIndex.value, 1)
  newSkills.splice(targetIndex, 0, draggedSkill)
  newSkills.forEach((skill, index) => {
    skill.order = index
    skill.dragging = false
  })
  skills.value = newSkills
  draggedIndex.value = null
  dragOverIndex.value = null
}

async function saveConfiguration() {
  saving.value = true
  try {
    const togglePromises = skills.value
      .filter((skill, index) => skill.enabled !== originalSkills.value[index]?.enabled)
      .map(skill =>
        client.patch({
          url: '/bots/{bot_id}/container/skills/{name}',
          path: { bot_id: props.botId, name: skill.name },
          body: { enabled: skill.enabled },
        }),
      )

    const orderChanged = skills.value.some(
      (skill, index) => skill.name !== originalSkills.value[index]?.name || index !== originalSkills.value[index]?.order,
    )

    const orderPromise = orderChanged
      ? client.put({
          url: '/bots/{bot_id}/container/skills/order',
          path: { bot_id: props.botId },
          body: {
            skills: skills.value.map((skill, index) => ({
              name: skill.name,
              order: index,
            })),
          },
        })
      : Promise.resolve()

    await Promise.all([...togglePromises, orderPromise])
    toast.success(t('skillManager.saveSuccess'))
    originalSkills.value = JSON.parse(JSON.stringify(skills.value))
  } catch (error) {
    toast.error(t('skillManager.saveFailed'))
    console.error('Failed to save skills configuration:', error)
  } finally {
    saving.value = false
  }
}

async function syncDefaultSkills() {
  syncing.value = true
  try {
    const { data } = await client.post({
      url: '/bots/{bot_id}/container/skills/sync',
      path: { bot_id: props.botId },
    }) as { data: { count: number } }

    toast.success(`Synced ${data.count} new skills`)
    await loadSkills()
  } catch (error) {
    toast.error('Failed to sync skills')
    console.error(error)
  } finally {
    syncing.value = false
  }
}

async function createSkill() {
  const name = newSkill.value.name.trim()
  if (!name) return
  creating.value = true
  try {
    await client.post({
      url: '/bots/{bot_id}/container/skills',
      path: { bot_id: props.botId },
      body: {
        skills: [{
          name,
          description: newSkill.value.description.trim(),
          content: newSkill.value.content.trim(),
        }],
      },
    })
    toast.success(t('skillManager.createSuccess'))
    showCreateDialog.value = false
    newSkill.value = { name: '', description: '', content: '' }
    await loadSkills()
  } catch (error) {
    toast.error(t('skillManager.createFailed'))
    console.error(error)
  } finally {
    creating.value = false
  }
}
</script>

<style scoped>
.skill-manager {
  user-select: none;
}
</style>
