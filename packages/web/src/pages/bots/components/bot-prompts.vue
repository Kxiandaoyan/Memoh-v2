<template>
  <div class="max-w-2xl mx-auto space-y-6">
    <!-- Identity -->
    <div class="space-y-2">
      <Label>{{ $t('bots.persona.identity') }}</Label>
      <Textarea
        v-model="form.identity"
        :placeholder="$t('bots.persona.identityPlaceholder')"
        rows="5"
      />
    </div>

    <!-- Soul -->
    <div class="space-y-2">
      <Label>{{ $t('bots.persona.soul') }}</Label>
      <Textarea
        v-model="form.soul"
        :placeholder="$t('bots.persona.soulPlaceholder')"
        rows="5"
      />
    </div>

    <!-- Task -->
    <div class="space-y-2">
      <Label>{{ $t('bots.persona.task') }}</Label>
      <Textarea
        v-model="form.task"
        :placeholder="$t('bots.persona.taskPlaceholder')"
        rows="5"
      />
    </div>

    <Separator />

    <!-- Allow Self Evolution -->
    <div class="flex items-center justify-between">
      <div class="space-y-0.5">
        <Label>{{ $t('bots.persona.allowSelfEvolution') }}</Label>
        <p class="text-sm text-muted-foreground">
          {{ $t('bots.persona.allowSelfEvolutionDescription') }}
        </p>
      </div>
      <Switch
        :model-value="form.allow_self_evolution"
        @update:model-value="(val) => form.allow_self_evolution = !!val"
      />
    </div>

    <!-- Enable OpenViking -->
    <div class="flex items-center justify-between">
      <div class="space-y-0.5">
        <Label>{{ $t('bots.persona.enableOpenviking') }}</Label>
        <p class="text-sm text-muted-foreground">
          {{ $t('bots.persona.enableOpenvikingDescription') }}
        </p>
      </div>
      <Switch
        :model-value="form.enable_openviking"
        @update:model-value="(val) => form.enable_openviking = !!val"
      />
    </div>

    <Separator />

    <!-- Save -->
    <div class="flex justify-end">
      <Button
        :disabled="!hasChanges || isSaving"
        @click="handleSave"
      >
        <Spinner v-if="isSaving" />
        {{ $t('bots.persona.save') }}
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Label,
  Button,
  Switch,
  Separator,
  Spinner,
  Textarea,
} from '@memoh/ui'
import { reactive, computed, watch, ref } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { useQuery, useQueryCache } from '@pinia/colada'
import { client } from '@memoh/sdk/client'
import type { Ref } from 'vue'

interface BotPrompts {
  identity: string
  soul: string
  task: string
  allow_self_evolution: boolean
  enable_openviking: boolean
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()
const queryCache = useQueryCache()
const botIdRef = computed(() => props.botId) as Ref<string>
const isSaving = ref(false)

// ---- Data ----
const { data: prompts } = useQuery({
  key: () => ['bot-prompts', botIdRef.value],
  query: async (): Promise<BotPrompts> => {
    const response = await client.get({
      url: '/bots/{bot_id}/prompts',
      path: { bot_id: botIdRef.value },
    })
    return response.data as BotPrompts
  },
  enabled: () => !!botIdRef.value,
})

// ---- Form ----
const form = reactive<BotPrompts>({
  identity: '',
  soul: '',
  task: '',
  allow_self_evolution: true,
  enable_openviking: false,
})

watch(prompts, (val) => {
  if (val) {
    form.identity = val.identity ?? ''
    form.soul = val.soul ?? ''
    form.task = val.task ?? ''
    form.allow_self_evolution = val.allow_self_evolution ?? true
    form.enable_openviking = val.enable_openviking ?? false
  }
}, { immediate: true })

const hasChanges = computed(() => {
  if (!prompts.value) return true
  const p = prompts.value
  return (
    form.identity !== (p.identity ?? '')
    || form.soul !== (p.soul ?? '')
    || form.task !== (p.task ?? '')
    || form.allow_self_evolution !== (p.allow_self_evolution ?? true)
    || form.enable_openviking !== (p.enable_openviking ?? false)
  )
})

async function handleSave() {
  isSaving.value = true
  try {
    await client.put({
      url: '/bots/{bot_id}/prompts',
      path: { bot_id: botIdRef.value },
      body: { ...form },
    })
    queryCache.invalidateQueries({ key: ['bot-prompts', botIdRef.value] })
    toast.success(t('bots.persona.saveSuccess'))
  } catch {
    toast.error('Failed to save persona')
  } finally {
    isSaving.value = false
  }
}
</script>
