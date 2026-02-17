<template>
  <div class="max-w-4xl mx-auto space-y-5">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('bots.heartbeat.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('bots.heartbeat.subtitle') }}
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
      {{ $t('bots.heartbeat.empty') }}
    </div>

    <!-- List -->
    <div
      v-else
      class="space-y-3"
    >
      <div
        v-for="item in items"
        :key="item.id"
        class="rounded-md border p-4 space-y-3"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2 min-w-0">
            <Badge
              :variant="item.enabled ? 'default' : 'secondary'"
              class="text-xs"
            >
              {{ item.enabled ? $t('bots.heartbeat.enabled') : $t('bots.heartbeat.disabled') }}
            </Badge>
            <span class="text-sm text-muted-foreground">
              {{ $t('bots.heartbeat.interval', { seconds: item.interval_seconds }) }}
            </span>
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
        <div class="text-sm">
          <p class="font-medium text-xs text-muted-foreground mb-1">{{ $t('bots.heartbeat.prompt') }}</p>
          <p class="whitespace-pre-wrap">{{ item.prompt || '-' }}</p>
        </div>
        <div
          v-if="item.event_triggers && item.event_triggers.length > 0"
          class="flex flex-wrap gap-1.5"
        >
          <Badge
            v-for="trigger in item.event_triggers"
            :key="trigger"
            variant="outline"
            class="text-xs"
          >
            {{ trigger }}
          </Badge>
        </div>
      </div>
    </div>

    <!-- Create/Edit Dialog -->
    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editingId ? $t('bots.heartbeat.editTitle') : $t('bots.heartbeat.createTitle') }}</DialogTitle>
        </DialogHeader>
        <div class="mt-4 space-y-4">
          <div class="flex items-center justify-between">
            <Label>{{ $t('common.enable') }}</Label>
            <Switch v-model="form.enabled" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.heartbeat.intervalLabel') }}</Label>
            <Input
              v-model.number="form.interval_seconds"
              type="number"
              min="10"
              :placeholder="$t('bots.heartbeat.intervalPlaceholder')"
              :disabled="saving"
            />
            <p class="text-xs text-muted-foreground">{{ $t('bots.heartbeat.intervalHint') }}</p>
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.heartbeat.promptLabel') }}</Label>
            <p class="text-xs text-muted-foreground">{{ $t('bots.heartbeat.promptHint') }}</p>
            <Textarea
              v-model="form.prompt"
              :placeholder="$t('bots.heartbeat.promptPlaceholder')"
              rows="4"
              :disabled="saving"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.heartbeat.eventTriggersLabel') }}</Label>
            <p class="text-xs text-muted-foreground">{{ $t('bots.heartbeat.eventTriggersHint') }}</p>
            <div class="space-y-2">
              <div
                v-for="trigger in allTriggers"
                :key="trigger"
                class="flex items-center gap-2"
              >
                <Checkbox
                  :id="`trigger-${trigger}`"
                  :checked="form.event_triggers.includes(trigger)"
                  @update:checked="toggleTrigger(trigger, $event)"
                />
                <Label
                  :for="`trigger-${trigger}`"
                  class="font-normal text-sm"
                >
                  {{ trigger }}
                </Label>
              </div>
            </div>
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
            :disabled="saving"
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
  Checkbox,
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Spinner,
  Switch,
  Textarea,
} from '@memoh/ui'
import { ref, reactive, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

interface HeartbeatConfig {
  id: string
  bot_id: string
  enabled: boolean
  interval_seconds: number
  prompt: string
  event_triggers: string[]
  created_at: string
  updated_at: string
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const allTriggers = ['message_created', 'schedule_completed']
const items = ref<HeartbeatConfig[]>([])
const loading = ref(false)
const saving = ref(false)
const deletingId = ref('')
const dialogOpen = ref(false)
const editingId = ref('')

const form = reactive({
  enabled: true,
  interval_seconds: 3600,
  prompt: '',
  event_triggers: [] as string[],
})

watch(() => props.botId, () => {
  loadList()
}, { immediate: true })

async function loadList() {
  loading.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/heartbeat',
      path: { bot_id: props.botId },
    }) as { data: { items: HeartbeatConfig[] } }
    items.value = data.items ?? []
  } catch {
    toast.error(t('bots.heartbeat.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingId.value = ''
  form.enabled = true
  form.interval_seconds = 3600
  form.prompt = ''
  form.event_triggers = []
  dialogOpen.value = true
}

function openEditDialog(item: HeartbeatConfig) {
  editingId.value = item.id
  form.enabled = item.enabled
  form.interval_seconds = item.interval_seconds
  form.prompt = item.prompt
  form.event_triggers = [...(item.event_triggers ?? [])]
  dialogOpen.value = true
}

function toggleTrigger(trigger: string, checked: boolean | string) {
  if (checked === true || checked === 'true') {
    if (!form.event_triggers.includes(trigger)) {
      form.event_triggers.push(trigger)
    }
  } else {
    form.event_triggers = form.event_triggers.filter((t) => t !== trigger)
  }
}

async function handleSave() {
  saving.value = true
  try {
    const body = {
      enabled: form.enabled,
      interval_seconds: form.interval_seconds,
      prompt: form.prompt,
      event_triggers: form.event_triggers,
    }
    if (editingId.value) {
      await client.put({
        url: '/bots/{bot_id}/heartbeat/{id}',
        path: { bot_id: props.botId, id: editingId.value },
        body,
      })
    } else {
      await client.post({
        url: '/bots/{bot_id}/heartbeat',
        path: { bot_id: props.botId },
        body,
      })
    }
    dialogOpen.value = false
    toast.success(t('bots.heartbeat.saveSuccess'))
    await loadList()
  } catch {
    toast.error(t('bots.heartbeat.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: string) {
  deletingId.value = id
  try {
    await client.delete({
      url: '/bots/{bot_id}/heartbeat/{id}',
      path: { bot_id: props.botId, id },
    })
    toast.success(t('bots.heartbeat.deleteSuccess'))
    await loadList()
  } catch {
    toast.error(t('bots.heartbeat.deleteFailed'))
  } finally {
    deletingId.value = ''
  }
}
</script>
