<template>
  <div class="max-w-4xl mx-auto space-y-6">
    <!-- Team membership cards -->
    <div v-if="loading" class="flex items-center gap-2 text-sm text-muted-foreground">
      <FontAwesomeIcon :icon="['fas', 'spinner']" class="animate-spin size-3.5" />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <div v-else-if="botTeams.length === 0" class="text-sm text-muted-foreground">
      {{ $t('teams.notInTeam') }}
    </div>

    <div v-for="entry in botTeams" :key="entry.team.id" class="rounded-md border p-4 space-y-4">
      <!-- Team header -->
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <FontAwesomeIcon :icon="['fas', 'users']" class="text-primary size-4" />
          <span class="font-semibold">{{ entry.team.name }}</span>
          <Badge v-if="entry.isManager" variant="default" class="text-xs">{{ $t('teams.manager') }}</Badge>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-xs text-muted-foreground">{{ formatDate(entry.team.created_at) }}</span>
          <Button
            v-if="entry.isManager"
            variant="destructive"
            size="sm"
            :disabled="deleting"
            @click="handleDeleteTeam(entry.team.id)"
          >
            <FontAwesomeIcon v-if="deleting" :icon="['fas', 'spinner']" class="mr-1.5 animate-spin" />
            {{ $t('teams.dissolve') }}
          </Button>
        </div>
      </div>

      <!-- Members -->
      <div>
        <p class="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wide">{{ $t('teams.members') }}</p>
        <div class="space-y-1.5">
          <div
            v-for="member in entry.members"
            :key="member.id"
            class="flex items-center gap-3 text-sm"
          >
            <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
            <span class="font-medium">{{ getBotName(member.target_bot_id) }}</span>
            <span v-if="member.role_description" class="text-muted-foreground">— {{ member.role_description }}</span>
            <Badge v-if="member.target_bot_id === entry.team.manager_bot_id" variant="secondary" class="text-[10px]">
              {{ $t('teams.manager') }}
            </Badge>
          </div>
          <div v-if="entry.members.length === 0" class="text-xs text-muted-foreground">
            {{ $t('teams.noMembers') }}
          </div>
        </div>
      </div>
    </div>

    <!-- Call logs -->
    <div class="rounded-md border p-4 space-y-3">
      <div class="flex items-center justify-between">
        <p class="text-sm font-medium">
          <FontAwesomeIcon :icon="['fas', 'arrow-right-arrow-left']" class="mr-1.5 text-blue-500" />
          {{ $t('teams.callLogs') }}
        </p>
        <Button variant="outline" size="sm" :disabled="callLogsLoading" @click="loadCallLogs">
          <FontAwesomeIcon v-if="callLogsLoading" :icon="['fas', 'spinner']" class="mr-1.5 animate-spin" />
          {{ $t('common.refresh') }}
        </Button>
      </div>

      <div v-if="callLogsLoading && callLogs.length === 0" class="text-sm text-muted-foreground flex items-center gap-2">
        <FontAwesomeIcon :icon="['fas', 'spinner']" class="animate-spin size-3" />
        {{ $t('common.loading') }}
      </div>

      <div v-else-if="callLogs.length === 0" class="text-sm text-muted-foreground">
        {{ $t('teams.noCallLogs') }}
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="log in callLogs"
          :key="log.id"
          class="border rounded p-3 text-sm space-y-1"
        >
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <span class="font-mono">{{ getBotName(log.caller_bot_id) }}</span>
              <FontAwesomeIcon :icon="['fas', 'arrow-right']" class="size-3" />
              <span class="font-mono">{{ getBotName(log.target_bot_id) }}</span>
            </div>
            <Badge
              :variant="callLogStatusVariant(log.status)"
              class="text-[10px] shrink-0"
            >{{ log.status }}</Badge>
          </div>
          <p class="text-xs line-clamp-2 text-muted-foreground">{{ log.message }}</p>
          <p v-if="log.result" class="text-xs line-clamp-2">{{ log.result }}</p>
          <p class="text-xs text-muted-foreground">{{ formatDate(log.created_at) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Badge, Button } from '@memoh/ui'
import { useQuery, useQueryCache } from '@pinia/colada'
import { getBotsQuery, getBotsQueryKey } from '@memoh/sdk/colada'
import { useTeams } from '@/composables/use-teams'
import { listTeamMembers, listBotCallLogs, deleteTeam, type TeamMember, type CallLog } from '@/lib/api-teams'

const props = defineProps<{ botId: string }>()
const { t } = useI18n()

const { teams, loading, refreshTeams } = useTeams()
const queryCache = useQueryCache()
const { data: botData } = useQuery(getBotsQuery())
const allBots = computed(() => botData.value?.items ?? [])

function getBotName(id: string) {
  return allBots.value.find(b => b.id === id)?.display_name || id
}

interface TeamEntry {
  team: { id: string; name: string; manager_bot_id: string; created_at: string }
  isManager: boolean
  members: TeamMember[]
}

const botTeams = ref<TeamEntry[]>([])

async function loadBotTeams() {
  const myTeams = teams.value.filter(t =>
    t.manager_bot_id === props.botId || (t.memberBotIds ?? []).includes(props.botId)
  )
  const entries = await Promise.all(myTeams.map(async (team) => {
    const members = await listTeamMembers(team.id)
    return {
      team,
      isManager: team.manager_bot_id === props.botId,
      members,
    } as TeamEntry
  }))
  botTeams.value = entries
}

const deleting = ref(false)

async function handleDeleteTeam(teamId: string) {
  if (!confirm(t('teams.dissolveConfirm'))) return
  deleting.value = true
  try {
    await deleteTeam(teamId)
    await refreshTeams()
    queryCache.invalidateQueries({ key: getBotsQueryKey() })
    await loadBotTeams()
  } finally {
    deleting.value = false
  }
}

const callLogs = ref<CallLog[]>([])
const callLogsLoading = ref(false)

async function loadCallLogs() {
  callLogsLoading.value = true
  try {
    callLogs.value = await listBotCallLogs(props.botId)
  } catch {
    callLogs.value = []
  } finally {
    callLogsLoading.value = false
  }
}

function callLogStatusVariant(status: string): 'default' | 'secondary' | 'destructive' {
  if (status === 'completed') return 'default'
  if (status === 'failed' || status === 'timeout') return 'destructive'
  return 'secondary'
}

function formatDate(value: string | undefined): string {
  if (!value) return '-'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '-'
  return parsed.toLocaleString()
}

// Wait for useTeams() to finish loading before fetching bot teams.
// Call logs are independent and can load in parallel.
watch(loading, (isLoading) => {
  if (!isLoading) {
    loadBotTeams()
    loadCallLogs()
  }
}, { immediate: true })
</script>
