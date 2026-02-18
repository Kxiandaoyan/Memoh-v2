<template>
  <div class="max-w-2xl mx-auto space-y-8">
    <!-- ═══ Section: Model Configuration ═══ -->
    <section class="space-y-5">
      <div>
        <h3 class="text-sm font-semibold">{{ $t('bots.settings.sectionModels') }}</h3>
        <p class="text-xs text-muted-foreground mt-0.5">{{ $t('bots.settings.sectionModelsHint') }}</p>
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.chatModel') }}</Label>
        <p class="text-xs text-muted-foreground">{{ $t('bots.settings.chatModelHint') }}</p>
        <ModelSelect
          v-model="form.chat_model_id"
          :models="models"
          :providers="providers"
          model-type="chat"
          :placeholder="$t('bots.settings.chatModel')"
        />
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.memoryModel') }}</Label>
        <p class="text-xs text-muted-foreground">{{ $t('bots.settings.memoryModelHint') }}</p>
        <ModelSelect
          v-model="form.memory_model_id"
          :models="models"
          :providers="providers"
          model-type="chat"
          :placeholder="$t('bots.settings.memoryModel')"
        />
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.embeddingModel') }}</Label>
        <p class="text-xs text-muted-foreground">{{ $t('bots.settings.embeddingModelHint') }}</p>
        <ModelSelect
          v-model="form.embedding_model_id"
          :models="models"
          :providers="providers"
          model-type="embedding"
          :placeholder="$t('bots.settings.embeddingModel')"
        />
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.vlmModel') }}</Label>
        <p class="text-xs text-muted-foreground">{{ $t('bots.settings.vlmModelHint') }}</p>
        <ModelSelect
          v-model="form.vlm_model_id"
          :models="models"
          :providers="providers"
          model-type="chat"
          :placeholder="$t('bots.settings.vlmModel')"
        />
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.searchProvider') }}</Label>
        <p class="text-xs text-muted-foreground">{{ $t('bots.settings.searchProviderHint') }}</p>
        <SearchProviderSelect
          v-model="form.search_provider_id"
          :providers="searchProviders"
          :placeholder="$t('bots.settings.searchProviderPlaceholder')"
        />
      </div>
    </section>

    <Separator />

    <!-- ═══ Section: Behavior ═══ -->
    <section class="space-y-5">
      <div>
        <h3 class="text-sm font-semibold">{{ $t('bots.settings.sectionBehavior') }}</h3>
        <p class="text-xs text-muted-foreground mt-0.5">{{ $t('bots.settings.sectionBehaviorHint') }}</p>
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.language') }}</Label>
        <p class="text-xs text-muted-foreground">{{ $t('bots.settings.languageHint') }}</p>
        <Input
          v-model="form.language"
          type="text"
        />
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.task') }}</Label>
        <p class="text-xs text-muted-foreground">
          {{ $t('bots.settings.taskDescription') }}
        </p>
        <Textarea
          v-model="promptsForm.task"
          :placeholder="$t('bots.settings.taskPlaceholder')"
          rows="4"
        />
      </div>

      <div class="space-y-2">
        <Label>{{ $t('bots.settings.maxContextLoadTime') }}</Label>
        <p class="text-xs text-muted-foreground">{{ $t('bots.settings.maxContextLoadTimeHint') }}</p>
        <Input
          v-model.number="form.max_context_load_time"
          type="number"
          :min="0"
        />
      </div>
    </section>

    <Separator />

    <!-- ═══ Section: Features ═══ -->
    <section class="space-y-4">
      <div>
        <h3 class="text-sm font-semibold">{{ $t('bots.settings.sectionFeatures') }}</h3>
        <p class="text-xs text-muted-foreground mt-0.5">{{ $t('bots.settings.sectionFeaturesHint') }}</p>
      </div>

      <div class="rounded-md border divide-y">
        <!-- Allow Self Evolution -->
        <div class="flex items-center justify-between p-4">
          <div class="space-y-0.5 pr-4">
            <Label>{{ $t('bots.settings.allowSelfEvolution') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ $t('bots.settings.allowSelfEvolutionDescription') }}
            </p>
          </div>
          <Switch
            :model-value="promptsForm.allow_self_evolution"
            @update:model-value="(val) => promptsForm.allow_self_evolution = !!val"
          />
        </div>

        <!-- Enable OpenViking -->
        <div class="flex items-center justify-between p-4">
          <div class="space-y-0.5 pr-4">
            <Label>{{ $t('bots.settings.enableOpenviking') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ $t('bots.settings.enableOpenvikingDescription') }}
            </p>
          </div>
          <Switch
            :model-value="promptsForm.enable_openviking"
            @update:model-value="(val) => promptsForm.enable_openviking = !!val"
          />
        </div>

        <!-- Privileged Bot -->
        <div class="flex items-center justify-between p-4">
          <div class="space-y-0.5 pr-4">
            <Label>{{ $t('bots.settings.isPrivileged') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ $t('bots.settings.isPrivilegedDescription') }}
            </p>
          </div>
          <Switch
            :model-value="promptsForm.is_privileged"
            @update:model-value="(val) => promptsForm.is_privileged = !!val"
          />
        </div>

        <!-- Allow Guest: only for public bot -->
        <div
          v-if="isPublicBot"
          class="flex items-center justify-between p-4"
        >
          <div class="space-y-0.5 pr-4">
            <Label>{{ $t('bots.settings.allowGuest') }}</Label>
            <p class="text-xs text-muted-foreground">{{ $t('bots.settings.allowGuestHint') }}</p>
          </div>
          <Switch
            :model-value="form.allow_guest"
            @update:model-value="(val) => form.allow_guest = !!val"
          />
        </div>

        <!-- Group Require Mention -->
        <div class="flex items-center justify-between p-4">
          <div class="space-y-0.5 pr-4">
            <Label>{{ $t('bots.settings.groupRequireMention') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ $t('bots.settings.groupRequireMentionDescription') }}
            </p>
          </div>
          <Switch
            :model-value="form.group_require_mention"
            @update:model-value="(val) => form.group_require_mention = !!val"
          />
        </div>
      </div>
    </section>

    <Separator />

    <!-- Save -->
    <div class="flex justify-end">
      <Button
        :disabled="!hasChanges || saving"
        @click="handleSave"
      >
        <Spinner v-if="saving" />
        {{ $t('bots.settings.save') }}
      </Button>
    </div>

    <Separator />

    <!-- ═══ Danger Zone ═══ -->
    <div
      class="rounded-lg border border-destructive/50 bg-destructive/5 p-4 space-y-3"
    >
      <h3 class="text-sm font-semibold text-destructive">
        {{ $t('bots.settings.dangerZone') }}
      </h3>
      <p class="text-xs text-muted-foreground">
        {{ $t('bots.settings.deleteBotDescription') }}
      </p>
      <div class="flex items-center justify-end">
        <ConfirmPopover
          :message="$t('bots.deleteConfirm')"
          :loading="deleteLoading"
          :confirm-text="$t('common.delete')"
          @confirm="handleDeleteBot"
        >
          <template #trigger>
            <Button
              variant="destructive"
              :disabled="deleteLoading"
            >
              <Spinner v-if="deleteLoading" class="mr-1.5" />
              {{ $t('bots.settings.deleteBot') }}
            </Button>
          </template>
        </ConfirmPopover>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Label,
  Input,
  Switch,
  Button,
  Separator,
  Spinner,
  Textarea,
} from '@memoh/ui'
import { reactive, computed, watch, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { resolveErrorMessage } from '@/utils/error'
import ConfirmPopover from '@/components/confirm-popover/index.vue'
import ModelSelect from './model-select.vue'
import SearchProviderSelect from './search-provider-select.vue'
import { useQuery, useMutation, useQueryCache } from '@pinia/colada'
import { getBotsByBotIdSettings, putBotsByBotIdSettings, deleteBotsById, getModels, getProviders, getSearchProviders } from '@memoh/sdk'
import { getBotsQueryKey } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'
import type { SettingsSettings } from '@memoh/sdk'
import type { Ref } from 'vue'

interface BotPrompts {
  identity: string
  soul: string
  task: string
  allow_self_evolution: boolean
  enable_openviking: boolean
  is_privileged: boolean
}

const props = defineProps<{
  botId: string
  botType?: string
}>()

const isPublicBot = computed(() => props.botType === 'public')

const { t } = useI18n()
const router = useRouter()
const saving = ref(false)

const botIdRef = computed(() => props.botId) as Ref<string>

// ---- Data: Settings ----
const queryCache = useQueryCache()

const { data: settings } = useQuery({
  key: () => ['bot-settings', botIdRef.value],
  query: async () => {
    try {
      const { data } = await getBotsByBotIdSettings({ path: { bot_id: botIdRef.value }, throwOnError: true })
      return data
    } catch (e) {
      console.error('Failed to load bot settings:', e)
      return undefined
    }
  },
  enabled: () => !!botIdRef.value,
})

// ---- Data: Prompts (task, switches) ----
const { data: prompts } = useQuery({
  key: () => ['bot-prompts', botIdRef.value],
  query: async (): Promise<BotPrompts | undefined> => {
    try {
      const response = await client.get({
        url: '/bots/{bot_id}/prompts',
        path: { bot_id: botIdRef.value },
      })
      return response.data as BotPrompts
    } catch (e) {
      console.error('Failed to load bot prompts:', e)
      return undefined
    }
  },
  enabled: () => !!botIdRef.value,
})

const { data: modelData } = useQuery({
  key: ['all-models'],
  query: async () => {
    try {
      const { data } = await getModels({ throwOnError: true })
      return data
    } catch {
      return []
    }
  },
})

const { data: providerData } = useQuery({
  key: ['all-providers'],
  query: async () => {
    try {
      const { data } = await getProviders({ throwOnError: true })
      return data
    } catch {
      return []
    }
  },
})

const { data: searchProviderData } = useQuery({
  key: ['all-search-providers'],
  query: async () => {
    try {
      const { data } = await getSearchProviders({ throwOnError: true })
      return data
    } catch {
      return []
    }
  },
})

const { mutateAsync: updateSettings, isLoading } = useMutation({
  mutation: async (body: Partial<SettingsSettings>) => {
    const { data } = await putBotsByBotIdSettings({
      path: { bot_id: botIdRef.value },
      body,
      throwOnError: true,
    })
    return data
  },
  onSettled: () => queryCache.invalidateQueries({ key: ['bot-settings', botIdRef.value] }),
})

const { mutateAsync: deleteBot, isLoading: deleteLoading } = useMutation({
  mutation: async () => {
    await deleteBotsById({ path: { id: botIdRef.value }, throwOnError: true })
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: getBotsQueryKey() })
  },
})

const models = computed(() => modelData.value ?? [])
const providers = computed(() => providerData.value ?? [])
const searchProviders = computed(() => searchProviderData.value ?? [])

// ---- Form: Settings ----
const form = reactive({
  chat_model_id: '',
  memory_model_id: '',
  embedding_model_id: '',
  vlm_model_id: '',
  search_provider_id: '',
  max_context_load_time: 0,
  language: '',
  allow_guest: false,
  group_require_mention: true,
})

watch(settings, (val) => {
  if (val) {
    form.chat_model_id = val.chat_model_id ?? ''
    form.memory_model_id = val.memory_model_id ?? ''
    form.embedding_model_id = val.embedding_model_id ?? ''
    form.vlm_model_id = val.vlm_model_id ?? ''
    form.search_provider_id = val.search_provider_id ?? ''
    form.max_context_load_time = val.max_context_load_time ?? 0
    form.language = val.language ?? ''
    form.allow_guest = val.allow_guest ?? false
    form.group_require_mention = (val as any).group_require_mention ?? true
  }
}, { immediate: true })

// ---- Form: Prompts (task, switches) ----
const promptsForm = reactive({
  task: '',
  allow_self_evolution: true,
  enable_openviking: false,
  is_privileged: false,
})

watch(prompts, (val) => {
  if (val) {
    promptsForm.task = val.task ?? ''
    promptsForm.allow_self_evolution = val.allow_self_evolution ?? true
    promptsForm.enable_openviking = val.enable_openviking ?? false
    promptsForm.is_privileged = val.is_privileged ?? false
  }
}, { immediate: true })

// ---- Change detection ----
const hasSettingsChanges = computed(() => {
  if (!settings.value) return true
  const s = settings.value as any
  let changed =
    form.chat_model_id !== (s.chat_model_id ?? '')
    || form.memory_model_id !== (s.memory_model_id ?? '')
    || form.embedding_model_id !== (s.embedding_model_id ?? '')
    || form.vlm_model_id !== (s.vlm_model_id ?? '')
    || form.search_provider_id !== (s.search_provider_id ?? '')
    || form.max_context_load_time !== (s.max_context_load_time ?? 0)
    || form.language !== (s.language ?? '')
    || form.group_require_mention !== (s.group_require_mention ?? true)
  if (isPublicBot.value) {
    changed = changed || form.allow_guest !== (s.allow_guest ?? false)
  }
  return changed
})

const hasPromptsChanges = computed(() => {
  if (!prompts.value) return true
  const p = prompts.value
  return (
    promptsForm.task !== (p.task ?? '')
    || promptsForm.allow_self_evolution !== (p.allow_self_evolution ?? true)
    || promptsForm.enable_openviking !== (p.enable_openviking ?? false)
    || promptsForm.is_privileged !== (p.is_privileged ?? false)
  )
})

const hasChanges = computed(() => hasSettingsChanges.value || hasPromptsChanges.value)

// ---- Save (dual API) ----
async function handleSave() {
  saving.value = true
  try {
    const promises: Promise<unknown>[] = []
    if (hasSettingsChanges.value) {
      promises.push(updateSettings({ ...form }))
    }
    if (hasPromptsChanges.value) {
      const promptsBody: Record<string, unknown> = {}
      const p = prompts.value
      if (promptsForm.task !== (p?.task ?? '')) {
        promptsBody.task = promptsForm.task
      }
      if (promptsForm.allow_self_evolution !== (p?.allow_self_evolution ?? true)) {
        promptsBody.allow_self_evolution = promptsForm.allow_self_evolution
      }
      if (promptsForm.enable_openviking !== (p?.enable_openviking ?? false)) {
        promptsBody.enable_openviking = promptsForm.enable_openviking
      }
      if (promptsForm.is_privileged !== (p?.is_privileged ?? false)) {
        promptsBody.is_privileged = promptsForm.is_privileged
      }
      promises.push(
        client.put({
          url: '/bots/{bot_id}/prompts',
          path: { bot_id: botIdRef.value },
          body: promptsBody,
        }).then(() => {
          queryCache.invalidateQueries({ key: ['bot-prompts', botIdRef.value] })
        }),
      )
    }
    await Promise.all(promises)
    toast.success(t('bots.settings.saveSuccess'))
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('bots.settings.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function handleDeleteBot() {
  try {
    await deleteBot()
    toast.success(t('bots.deleteSuccess'))
    await router.push({ name: 'bots' })
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('bots.lifecycle.deleteFailed')))
  }
}
</script>
