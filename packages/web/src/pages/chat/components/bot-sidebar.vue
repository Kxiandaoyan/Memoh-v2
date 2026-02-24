<template>
  <div class="w-56 shrink-0 border-r flex flex-col h-full">
    <div class="p-3 border-b">
      <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
        {{ $t('sidebar.bots') }}
      </h3>
    </div>

    <ScrollArea class="flex-1">
      <div class="p-1">
        <!-- Loading -->
        <div
          v-if="isLoading"
          class="flex justify-center py-4"
        >
          <FontAwesomeIcon
            :icon="['fas', 'spinner']"
            class="size-4 animate-spin text-muted-foreground"
          />
        </div>

        <!-- Team groups -->
        <template v-if="teams.length > 0">
          <div v-for="team in teams" :key="team.id" class="mb-1">
            <div class="flex items-center gap-1.5 px-3 py-1.5">
              <FontAwesomeIcon :icon="['fas', 'users']" class="size-3 text-muted-foreground" />
              <span class="text-xs font-semibold text-muted-foreground truncate uppercase tracking-wide">{{ team.name }}</span>
            </div>
            <button
              v-for="bot in getTeamBots(team)"
              :key="bot.id"
              class="flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors hover:bg-accent"
              :class="{ 'bg-accent': currentBotId === bot.id }"
              @click="handleSelect(bot)"
            >
              <Avatar class="size-8 shrink-0">
                <AvatarImage
                  v-if="bot.avatar_url"
                  :src="bot.avatar_url"
                  :alt="bot.display_name"
                />
                <AvatarFallback class="text-xs">
                  {{ (bot.display_name || bot.id).slice(0, 2).toUpperCase() }}
                </AvatarFallback>
              </Avatar>
              <div class="flex-1 text-left min-w-0">
                <div class="font-medium truncate">
                  {{ bot.display_name || bot.id }}
                  <span v-if="bot.id === team.manager_bot_id" class="text-[10px] text-primary ml-1">{{ $t('teams.manager') }}</span>
                </div>
              </div>
            </button>
          </div>

          <!-- Independent bots -->
          <template v-if="independentBots.length > 0">
            <div class="flex items-center gap-1.5 px-3 py-1.5 mt-1">
              <FontAwesomeIcon :icon="['fas', 'robot']" class="size-3 text-muted-foreground" />
              <span class="text-xs font-semibold text-muted-foreground uppercase tracking-wide">{{ $t('teams.independent') }}</span>
            </div>
            <button
              v-for="bot in independentBots"
              :key="bot.id"
              class="flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors hover:bg-accent"
              :class="{ 'bg-accent': currentBotId === bot.id }"
              @click="handleSelect(bot)"
            >
              <Avatar class="size-8 shrink-0">
                <AvatarImage
                  v-if="bot.avatar_url"
                  :src="bot.avatar_url"
                  :alt="bot.display_name"
                />
                <AvatarFallback class="text-xs">
                  {{ (bot.display_name || bot.id).slice(0, 2).toUpperCase() }}
                </AvatarFallback>
              </Avatar>
              <div class="flex-1 text-left min-w-0">
                <div class="font-medium truncate">
                  {{ bot.display_name || bot.id }}
                </div>
              </div>
            </button>
          </template>
        </template>

        <!-- No teams — flat list -->
        <template v-else>
          <button
            v-for="bot in bots"
            :key="bot.id"
            class="flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors hover:bg-accent"
            :class="{ 'bg-accent': currentBotId === bot.id }"
            @click="handleSelect(bot)"
          >
            <Avatar class="size-8 shrink-0">
              <AvatarImage
                v-if="bot.avatar_url"
                :src="bot.avatar_url"
                :alt="bot.display_name"
              />
              <AvatarFallback class="text-xs">
                {{ (bot.display_name || bot.id).slice(0, 2).toUpperCase() }}
              </AvatarFallback>
            </Avatar>
            <div class="flex-1 text-left min-w-0">
              <div class="font-medium truncate">
                {{ bot.display_name || bot.id }}
              </div>
              <div
                v-if="bot.type"
                class="text-xs text-muted-foreground truncate"
              >
                {{ botTypeLabel(bot.type) }}
              </div>
            </div>
          </button>
        </template>

        <!-- Empty -->
        <div
          v-if="!isLoading && bots.length === 0"
          class="px-3 py-6 text-center text-sm text-muted-foreground"
        >
          {{ $t('bots.emptyTitle') }}
        </div>
      </div>
    </ScrollArea>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarImage, AvatarFallback, ScrollArea } from '@memoh/ui'
import { useQuery } from '@pinia/colada'
import { getBotsQuery } from '@memoh/sdk/colada'
import type { BotsBot } from '@memoh/sdk'
import { useChatStore } from '@/store/chat-list'
import { storeToRefs } from 'pinia'
import { useTeams } from '@/composables/use-teams'

const { t } = useI18n()
const chatStore = useChatStore()
const { currentBotId } = storeToRefs(chatStore)

const { data: botData, isLoading } = useQuery(getBotsQuery())
const bots = computed<BotsBot[]>(() => botData.value?.items ?? [])

const { teams, teamBotIds } = useTeams()

function getTeamBots(team: { manager_bot_id: string; memberBotIds?: string[] }) {
  const ids = new Set([team.manager_bot_id, ...(team.memberBotIds ?? [])])
  return bots.value.filter(b => ids.has(b.id))
}

const independentBots = computed(() =>
  bots.value.filter(b => !teamBotIds.value.has(b.id))
)

function botTypeLabel(type: string): string {
  if (!type) return ''
  const key = `bots.types.${type}`
  const out = t(key)
  return out !== key ? out : type
}

function handleSelect(bot: BotsBot) {
  chatStore.selectBot(bot.id)
}
</script>
