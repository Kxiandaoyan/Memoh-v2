<template>
  <div class="max-w-4xl mx-auto space-y-5">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('bots.subagents.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('bots.subagents.subtitle') }}
        </p>
      </div>
      <div class="flex gap-2 shrink-0">
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadList"
        >
          <Spinner
            v-if="loading"
            class="mr-1.5"
          />
          {{ $t('common.refresh') }}
        </Button>
        <Button
          size="sm"
          @click="openCreateDialog"
        >
          {{ $t('common.add') }}
        </Button>
      </div>
    </div>

    <!-- Loading -->
    <div
      v-if="loading && items.length === 0"
      class="flex items-center gap-2 text-sm text-muted-foreground"
    >
      <Spinner />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty -->
    <div
      v-else-if="items.length === 0"
      class="rounded-md border p-6 text-center text-sm text-muted-foreground"
    >
      {{ $t('bots.subagents.empty') }}
    </div>

    <!-- List -->
    <div
      v-else
      class="space-y-3"
    >
      <div
        v-for="item in items"
        :key="item.id"
        class="rounded-md border p-4"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0 space-y-1">
            <p class="text-sm font-medium truncate">{{ item.name }}</p>
            <p
              v-if="item.description"
              class="text-sm text-muted-foreground truncate"
            >
              {{ item.description }}
            </p>
            <div class="flex flex-wrap gap-1.5 mt-1">
              <Badge
                v-for="skill in item.skills"
                :key="skill"
                variant="secondary"
                class="text-xs"
              >
                {{ skill }}
              </Badge>
            </div>
          </div>
          <div class="flex gap-2 shrink-0">
            <Button
              variant="outline"
              size="sm"
              @click="openEditDialog(item)"
            >
              {{ $t('common.edit') }}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              :disabled="deletingId === item.id"
              @click="handleDelete(item.id)"
            >
              <Spinner
                v-if="deletingId === item.id"
                class="mr-1.5"
              />
              {{ $t('common.delete') }}
            </Button>
          </div>
        </div>
        <p class="text-xs text-muted-foreground mt-2">
          {{ $t('bots.subagents.createdAt') }}: {{ formatDate(item.created_at) }}
        </p>
      </div>
    </div>

    <!-- Create/Edit Dialog -->
    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editingId ? $t('bots.subagents.editTitle') : $t('bots.subagents.createTitle') }}</DialogTitle>
        </DialogHeader>
        <div class="mt-4 space-y-4">
          <div class="space-y-2">
            <Label>{{ $t('bots.subagents.nameLabel') }}</Label>
            <Input
              v-model="form.name"
              :placeholder="$t('bots.subagents.namePlaceholder')"
              :disabled="saving"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.subagents.descriptionLabel') }}</Label>
            <Textarea
              v-model="form.description"
              :placeholder="$t('bots.subagents.descriptionPlaceholder')"
              rows="3"
              :disabled="saving"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.subagents.skillsLabel') }}</Label>
            <Input
              v-model="skillsInput"
              :placeholder="$t('bots.subagents.skillsPlaceholder')"
              :disabled="saving"
            />
            <p class="text-xs text-muted-foreground">{{ $t('bots.subagents.skillsHint') }}</p>
          </div>
        </div>
        <DialogFooter class="mt-6">
          <DialogClose as-child>
            <Button
              variant="outline"
              :disabled="saving"
            >
              {{ $t('common.cancel') }}
            </Button>
          </DialogClose>
          <Button
            :disabled="saving || !form.name.trim()"
            @click="handleSave"
          >
            <Spinner
              v-if="saving"
              class="mr-1.5"
            />
            {{ $t('common.save') }}
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
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Spinner,
  Textarea,
} from '@memoh/ui'
import { ref, reactive, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

interface SubagentItem {
  id: string
  name: string
  description: string
  bot_id: string
  skills: string[]
  messages: unknown[]
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const items = ref<SubagentItem[]>([])
const loading = ref(false)
const saving = ref(false)
const deletingId = ref('')
const dialogOpen = ref(false)
const editingId = ref('')
const skillsInput = ref('')

const form = reactive({
  name: '',
  description: '',
})

watch(() => props.botId, () => {
  loadList()
}, { immediate: true })

async function loadList() {
  loading.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/subagents',
      path: { bot_id: props.botId },
    }) as { data: { items: SubagentItem[] } }
    items.value = data.items ?? []
  } catch {
    toast.error(t('bots.subagents.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingId.value = ''
  form.name = ''
  form.description = ''
  skillsInput.value = ''
  dialogOpen.value = true
}

function openEditDialog(item: SubagentItem) {
  editingId.value = item.id
  form.name = item.name
  form.description = item.description
  skillsInput.value = (item.skills ?? []).join(', ')
  dialogOpen.value = true
}

async function handleSave() {
  if (!form.name.trim()) return
  saving.value = true
  const skills = skillsInput.value
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  try {
    if (editingId.value) {
      await client.put({
        url: '/bots/{bot_id}/subagents/{id}',
        path: { bot_id: props.botId, id: editingId.value },
        body: {
          name: form.name.trim(),
          description: form.description.trim(),
        },
      })
      if (skills.length > 0) {
        await client.put({
          url: '/bots/{bot_id}/subagents/{id}/skills',
          path: { bot_id: props.botId, id: editingId.value },
          body: { skills },
        })
      }
    } else {
      await client.post({
        url: '/bots/{bot_id}/subagents',
        path: { bot_id: props.botId },
        body: {
          name: form.name.trim(),
          description: form.description.trim(),
          skills,
        },
      })
    }
    dialogOpen.value = false
    toast.success(t('bots.subagents.saveSuccess'))
    await loadList()
  } catch {
    toast.error(t('bots.subagents.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: string) {
  deletingId.value = id
  try {
    await client.delete({
      url: '/bots/{bot_id}/subagents/{id}',
      path: { bot_id: props.botId, id },
    })
    toast.success(t('bots.subagents.deleteSuccess'))
    await loadList()
  } catch {
    toast.error(t('bots.subagents.deleteFailed'))
  } finally {
    deletingId.value = ''
  }
}

function formatDate(value: string): string {
  if (!value) return '-'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? '-' : d.toLocaleString()
}
</script>
