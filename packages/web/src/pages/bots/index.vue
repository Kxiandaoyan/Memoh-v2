<template>
  <section class="p-6 max-w-7xl mx-auto">
    <!-- Header: search + create -->
    <div class="flex items-center justify-between mb-6">
      <h3 class="text-xl font-semibold tracking-tight">
        {{ $t('bots.title') }}
      </h3>
      <div class="flex items-center gap-3">
        <div class="relative">
          <FontAwesomeIcon
            :icon="['fas', 'magnifying-glass']"
            class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground size-3.5"
          />
          <Input
            v-model="searchText"
            :placeholder="$t('bots.searchPlaceholder')"
            class="pl-9 w-64"
          />
        </div>
        <CreateBot v-model:open="dialogOpen" />
        <Button variant="outline" @click="createTeamOpen = true">
          <FontAwesomeIcon :icon="['fas', 'users']" class="mr-2 size-3.5" />
          {{ $t('teams.createTeam') }}
        </Button>
      </div>
    </div>

    <!-- Team groups -->
    <template v-if="teams.length > 0 && !searchText">
      <div v-for="team in teams" :key="team.id" class="mb-8">
        <div class="flex items-center gap-2 mb-3">
          <FontAwesomeIcon :icon="['fas', 'users']" class="text-primary size-4" />
          <h4 class="font-semibold text-base">{{ team.name }}</h4>
          <span class="text-xs text-muted-foreground">{{ $t('teams.teamGroup') }}</span>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <BotCard
            v-for="bot in getTeamBots(team)"
            :key="bot.id"
            :bot="bot"
          />
        </div>
      </div>

      <!-- Independent bots section -->
      <div v-if="independentBots.length > 0" class="mb-8">
        <div class="flex items-center gap-2 mb-3">
          <FontAwesomeIcon :icon="['fas', 'robot']" class="text-muted-foreground size-4" />
          <h4 class="font-semibold text-base text-muted-foreground">{{ $t('teams.independent') }}</h4>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <BotCard
            v-for="bot in independentBots"
            :key="bot.id"
            :bot="bot"
          />
        </div>
      </div>
    </template>

    <!-- Default flat bot grid (search results or no teams) -->
    <template v-else>
      <div
        v-if="filteredBots.length > 0"
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
      >
        <BotCard
          v-for="bot in filteredBots"
          :key="bot.id"
          :bot="bot"
        />
      </div>

      <!-- Empty state -->
      <Empty
        v-else-if="!isLoading"
        class="mt-20 flex flex-col items-center justify-center"
      >
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <FontAwesomeIcon :icon="['fas', 'robot']" />
          </EmptyMedia>
        </EmptyHeader>
        <EmptyTitle>{{ $t('bots.emptyTitle') }}</EmptyTitle>
        <EmptyDescription>{{ $t('bots.emptyDescription') }}</EmptyDescription>
        <EmptyContent />
      </Empty>
    </template>

    <!-- Create Team drawer -->
    <CreateTeam v-model:open="createTeamOpen" @created="onTeamCreated" />
  </section>
</template>

<script setup lang="ts">
import {
  Button,
  Input,
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@memoh/ui'
import { ref, computed, watch, onUnmounted } from 'vue'
import BotCard from './components/bot-card.vue'
import CreateBot from './components/create-bot.vue'
import CreateTeam from './components/create-team.vue'
import { useQuery, useQueryCache } from '@pinia/colada'
import { getBotsQuery, getBotsQueryKey } from '@memoh/sdk/colada'
import { useTeams } from '../../composables/use-teams'

const searchText = ref('')
const dialogOpen = ref(false)
const createTeamOpen = ref(false)
const queryCache = useQueryCache()

const { data: botData, status } = useQuery(getBotsQuery())
const { teams, teamBotIds, refreshTeams } = useTeams()

const isLoading = computed(() => status.value === 'loading')

const allBots = computed(() => botData.value?.items ?? [])

const filteredBots = computed(() => {
  const keyword = searchText.value.trim().toLowerCase()
  if (!keyword) return allBots.value
  return allBots.value.filter((bot) =>
    bot.display_name?.toLowerCase().includes(keyword)
    || bot.id.toLowerCase().includes(keyword)
    || bot.type?.toLowerCase().includes(keyword),
  )
})

function getTeamBots(team: { manager_bot_id: string; memberBotIds?: string[] }) {
  const ids = new Set([team.manager_bot_id, ...(team.memberBotIds ?? [])])
  return allBots.value.filter(b => ids.has(b.id))
}

const independentBots = computed(() =>
  allBots.value.filter(b => !teamBotIds.value.has(b.id))
)

const hasPendingBots = computed(() =>
  allBots.value.some((bot) => bot.status === 'creating' || bot.status === 'deleting'),
)

function onTeamCreated() {
  refreshTeams()
  queryCache.invalidateQueries({ key: getBotsQueryKey() })
}

let pollTimer: ReturnType<typeof setInterval> | null = null

watch(hasPendingBots, (pending) => {
  if (pending) {
    if (pollTimer == null) {
      pollTimer = setInterval(() => {
        queryCache.invalidateQueries({ key: getBotsQueryKey() })
      }, 2000)
    }
    return
  }
  if (pollTimer != null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}, { immediate: true })

onUnmounted(() => {
  if (pollTimer != null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>
