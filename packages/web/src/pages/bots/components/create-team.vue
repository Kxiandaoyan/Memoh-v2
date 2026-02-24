<template>
  <Sheet v-model:open="open">
    <SheetContent class="w-full sm:max-w-lg overflow-y-auto">
      <SheetHeader>
        <SheetTitle>{{ $t('teams.createTeam') }}</SheetTitle>
        <SheetDescription>{{ $t('teams.createTeamDescription') }}</SheetDescription>
      </SheetHeader>

      <!-- Step indicators -->
      <div class="flex items-center gap-2 mt-4 mb-6">
        <div
          v-for="(stepLabel, i) in steps"
          :key="i"
          class="flex items-center gap-1.5"
        >
          <div
            class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium"
            :class="i === currentStep
              ? 'bg-primary text-primary-foreground'
              : i < currentStep
                ? 'bg-primary/20 text-primary'
                : 'bg-muted text-muted-foreground'"
          >{{ i + 1 }}</div>
          <span class="text-xs" :class="i === currentStep ? 'font-medium' : 'text-muted-foreground'">{{ stepLabel }}</span>
          <div v-if="i < steps.length - 1" class="w-6 h-px bg-border" />
        </div>
      </div>

      <!-- Step 1: Select members -->
      <div v-if="currentStep === 0" class="space-y-4">
        <div>
          <Label class="text-sm font-medium mb-2 block">{{ $t('teams.selectMembers') }}</Label>
          <p class="text-xs text-muted-foreground mb-3">{{ $t('teams.selectMembersHint') }}</p>
          <div class="space-y-2 max-h-72 overflow-y-auto">
            <div
              v-for="bot in eligibleBots"
              :key="bot.id"
              class="flex items-center gap-3 p-2.5 border rounded-lg cursor-pointer transition-colors"
              :class="isSelected(bot.id) ? 'border-primary bg-primary/5' : 'border-border hover:border-muted-foreground'"
              @click="toggleBot(bot.id)"
            >
              <Checkbox :model-value="isSelected(bot.id)" @update:model-value="toggleBot(bot.id)" />
              <div class="flex-1 min-w-0">
                <div class="font-medium text-sm truncate">{{ bot.display_name || bot.id }}</div>
                <div class="text-xs text-muted-foreground">{{ bot.id }}</div>
              </div>
            </div>
          </div>
          <p v-if="eligibleBots.length === 0" class="text-sm text-muted-foreground text-center py-4">
            {{ $t('teams.noBotsAvailable') }}
          </p>
        </div>

        <!-- Role descriptions for selected bots -->
        <div v-if="selectedBots.length > 0" class="space-y-3">
          <Label class="text-sm font-medium">{{ $t('teams.memberRoles') }}</Label>
          <div
            v-for="sel in selectedBots"
            :key="sel.botId"
            class="space-y-1"
          >
            <Label class="text-xs text-muted-foreground">{{ getBotName(sel.botId) }}</Label>
            <Input
              v-model="sel.roleDescription"
              :placeholder="$t('teams.roleDescriptionPlaceholder')"
              class="text-sm"
            />
          </div>
        </div>
      </div>

      <!-- Step 2: Team name -->
      <div v-if="currentStep === 1" class="space-y-4">
        <div class="space-y-1.5">
          <Label>{{ $t('teams.teamName') }}</Label>
          <Input v-model="teamName" :placeholder="$t('teams.teamNamePlaceholder')" />
        </div>
        <div class="space-y-1.5">
          <Label>{{ $t('teams.heartbeatPrompt') }}</Label>
          <Textarea
            v-model="heartbeatPrompt"
            :placeholder="$t('teams.heartbeatPromptPlaceholder')"
            :rows="3"
          />
          <p class="text-xs text-muted-foreground">{{ $t('teams.heartbeatPromptHint') }}</p>
        </div>
      </div>

      <!-- Step 3: Confirm -->
      <div v-if="currentStep === 2" class="space-y-4">
        <div class="rounded-lg border p-4 space-y-3">
          <div class="flex items-center gap-2">
            <FontAwesomeIcon :icon="['fas', 'users']" class="text-primary" />
            <span class="font-semibold">{{ teamName }}</span>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-1.5">{{ $t('teams.memberCount', { count: selectedBots.length }) }}</p>
            <ul class="space-y-1">
              <li
                v-for="sel in selectedBots"
                :key="sel.botId"
                class="text-sm flex items-center gap-2"
              >
                <span class="font-medium">{{ getBotName(sel.botId) }}</span>
                <span v-if="sel.roleDescription" class="text-muted-foreground">— {{ sel.roleDescription }}</span>
              </li>
            </ul>
          </div>
          <p class="text-xs text-muted-foreground">{{ $t('teams.confirmNote') }}</p>
        </div>
      </div>

      <!-- Footer actions -->
      <div class="flex items-center justify-between mt-6 pt-4 border-t">
        <Button v-if="currentStep > 0" variant="outline" @click="currentStep--">
          {{ $t('common.back') }}
        </Button>
        <div v-else />

        <div class="flex gap-2">
          <Button variant="outline" @click="handleClose">{{ $t('common.cancel') }}</Button>
          <Button
            v-if="currentStep < steps.length - 1"
            :disabled="!canProceed"
            @click="currentStep++"
          >
            {{ $t('teams.next') }}
          </Button>
          <Button
            v-else
            :disabled="creating || !canSubmit"
            @click="handleCreate"
          >
            <FontAwesomeIcon v-if="creating" :icon="['fas', 'spinner']" class="mr-1.5 animate-spin" />
            {{ $t('teams.create') }}
          </Button>
        </div>
      </div>
    </SheetContent>
  </Sheet>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  Button,
  Input,
  Label,
  Checkbox,
  Textarea,
} from '@memoh/ui'
import { toast } from 'vue-sonner'
import { useQuery } from '@pinia/colada'
import { getBotsQuery } from '@memoh/sdk/colada'
import { createTeam } from '../../../lib/api-teams'
import { useTeams } from '../../../composables/use-teams'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [boolean]
  created: []
}>()

const { t } = useI18n()

const open = computed({
  get: () => props.open,
  set: (v) => emit('update:open', v),
})

const steps = computed(() => [
  t('teams.step1'),
  t('teams.step2'),
  t('teams.step3'),
])

const currentStep = ref(0)
const teamName = ref('')
const heartbeatPrompt = ref('')
const selectedBots = ref<Array<{ botId: string; roleDescription: string }>>([])
const creating = ref(false)

const { data: botData } = useQuery(getBotsQuery())
const { teamBotIds } = useTeams()
const allBots = computed(() => botData.value?.items ?? [])
const eligibleBots = computed(() =>
  allBots.value.filter(b => b.status === 'ready' && !teamBotIds.value.has(b.id))
)

function isSelected(botId: string) {
  return selectedBots.value.some(s => s.botId === botId)
}

function toggleBot(botId: string) {
  const idx = selectedBots.value.findIndex(s => s.botId === botId)
  if (idx >= 0) {
    selectedBots.value.splice(idx, 1)
  } else {
    selectedBots.value.push({ botId, roleDescription: '' })
  }
}

function getBotName(botId: string) {
  return allBots.value.find(b => b.id === botId)?.display_name || botId
}

const canProceed = computed(() => {
  if (currentStep.value === 0) return selectedBots.value.length >= 1
  if (currentStep.value === 1) return teamName.value.trim().length > 0
  return true
})

const canSubmit = computed(() => teamName.value.trim().length > 0 && selectedBots.value.length >= 1)

async function handleCreate() {
  if (!canSubmit.value) return
  creating.value = true
  try {
    await createTeam({
      name: teamName.value.trim(),
      members: selectedBots.value.map(s => ({
        bot_id: s.botId,
        role_description: s.roleDescription,
      })),
      heartbeat_prompt: heartbeatPrompt.value.trim() || undefined,
    })
    toast.success(t('teams.createSuccess'))
    emit('created')
    handleClose()
  } catch (err: any) {
    const msg = err?.message || err?.error?.message || t('teams.createFailed')
    toast.error(msg)
  } finally {
    creating.value = false
  }
}

function handleClose() {
  open.value = false
  currentStep.value = 0
  teamName.value = ''
  heartbeatPrompt.value = ''
  selectedBots.value = []
}
</script>
