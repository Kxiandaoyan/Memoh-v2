<template>
  <section class="p-6 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <h3 class="text-xl font-semibold tracking-tight">
        {{ $t('bots.schedules.title') }}
      </h3>
    </div>

    <!-- Loading -->
    <div
      v-if="isLoading"
      class="flex items-center justify-center py-20"
    >
      <Spinner class="size-6 text-muted-foreground" />
      <span class="ml-2 text-muted-foreground">{{ $t('bots.schedules.loading') }}</span>
    </div>

    <!-- Content: grouped by bot -->
    <div
      v-else-if="groupedSchedules.length > 0"
      class="space-y-4"
    >
      <Collapsible
        v-for="group in groupedSchedules"
        :key="group.bot.id"
        :default-open="true"
        class="rounded-lg border bg-card"
      >
        <CollapsibleTrigger class="flex w-full items-center gap-3 px-4 py-3 hover:bg-accent/50 transition-colors rounded-t-lg">
          <FontAwesomeIcon
            :icon="['fas', 'chevron-right']"
            class="size-3 text-muted-foreground transition-transform [[data-state=open]>&]:rotate-90"
          />
          <Avatar class="size-7 shrink-0">
            <AvatarImage
              v-if="group.bot.avatar_url"
              :src="group.bot.avatar_url"
              :alt="group.bot.display_name"
            />
            <AvatarFallback class="text-[10px]">
              {{ (group.bot.display_name || 'B').slice(0, 2).toUpperCase() }}
            </AvatarFallback>
          </Avatar>
          <span class="font-medium text-sm">{{ group.bot.display_name || group.bot.id }}</span>
          <Badge
            variant="secondary"
            class="ml-auto"
          >
            {{ group.schedules.length }}
          </Badge>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div class="border-t">
            <div
              v-for="schedule in group.schedules"
              :key="schedule.id"
              class="flex items-center gap-4 px-4 py-3 border-b last:border-b-0 hover:bg-accent/30 transition-colors"
            >
              <!-- Enable/Disable switch -->
              <Switch
                :checked="schedule.enabled"
                @update:checked="(val: boolean) => toggleSchedule(group.bot.id!, schedule, val)"
              />

              <!-- Info -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-sm truncate">{{ schedule.name }}</span>
                  <Badge :variant="schedule.enabled ? 'default' : 'outline'">
                    {{ schedule.enabled ? $t('bots.schedules.enabled') : $t('bots.schedules.disabled') }}
                  </Badge>
                </div>
                <p
                  v-if="schedule.description"
                  class="text-xs text-muted-foreground mt-0.5 truncate"
                >
                  {{ schedule.description }}
                </p>
                <div class="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
                  <span class="flex items-center gap-1">
                    <FontAwesomeIcon
                      :icon="['fas', 'clock']"
                      class="size-3"
                    />
                    {{ schedule.pattern }}
                  </span>
                  <span class="flex items-center gap-1">
                    <FontAwesomeIcon
                      :icon="['fas', 'terminal']"
                      class="size-3"
                    />
                    <code class="bg-muted px-1 rounded text-[11px]">{{ schedule.command }}</code>
                  </span>
                  <span class="flex items-center gap-1">
                    <FontAwesomeIcon
                      :icon="['fas', 'rotate']"
                      class="size-3"
                    />
                    {{ formatCalls(schedule) }}
                  </span>
                </div>
              </div>

              <!-- Delete button -->
              <Button
                variant="ghost"
                size="sm"
                class="text-muted-foreground hover:text-destructive shrink-0"
                @click="confirmDelete(group.bot.id!, schedule)"
              >
                <FontAwesomeIcon
                  :icon="['far', 'trash-can']"
                  class="size-3.5"
                />
              </Button>
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>

    <!-- Empty state -->
    <Empty
      v-else
      class="mt-20 flex flex-col items-center justify-center"
    >
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <FontAwesomeIcon :icon="['fas', 'clock']" />
        </EmptyMedia>
      </EmptyHeader>
      <EmptyTitle>{{ $t('bots.schedules.emptyTitle') }}</EmptyTitle>
      <EmptyDescription>{{ $t('bots.schedules.emptyDescription') }}</EmptyDescription>
      <EmptyContent />
    </Empty>
  </section>
</template>

<script setup lang="ts">
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  Badge,
  Button,
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  Spinner,
  Switch,
} from '@memoh/ui'
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { getBots, getBotsByBotIdSchedule, deleteBotsByBotIdScheduleById, putBotsByBotIdScheduleById } from '@memoh/sdk'
import type { BotsBot, ScheduleSchedule } from '@memoh/sdk/types.gen'

const { t } = useI18n()

interface BotScheduleGroup {
  bot: BotsBot
  schedules: ScheduleSchedule[]
}

const allGroups = ref<BotScheduleGroup[]>([])
const isLoading = ref(true)

const groupedSchedules = computed(() =>
  allGroups.value.filter(g => g.schedules.length > 0),
)

async function loadData() {
  isLoading.value = true
  try {
    const { data: botData } = await getBots({ throwOnError: true })
    const bots = botData?.items ?? []

    const groups: BotScheduleGroup[] = []
    const results = await Promise.allSettled(
      bots.map(async (bot) => {
        const { data } = await getBotsByBotIdSchedule({
          path: { bot_id: bot.id! },
          throwOnError: true,
        })
        return { bot, schedules: data?.items ?? [] }
      }),
    )

    for (const result of results) {
      if (result.status === 'fulfilled') {
        groups.push(result.value)
      }
    }

    allGroups.value = groups
  } catch {
    toast.error(t('common.loadFailed'))
  } finally {
    isLoading.value = false
  }
}

function formatCalls(schedule: ScheduleSchedule) {
  const current = schedule.current_calls ?? 0
  if (schedule.max_calls != null) {
    return t('bots.schedules.callsWithMax', { current, max: schedule.max_calls })
  }
  return t('bots.schedules.callsUnlimited', { current })
}

async function toggleSchedule(botId: string, schedule: ScheduleSchedule, enabled: boolean) {
  try {
    await putBotsByBotIdScheduleById({
      path: { bot_id: botId, id: schedule.id! },
      body: { enabled },
      throwOnError: true,
    })
    schedule.enabled = enabled
    toast.success(enabled ? t('bots.schedules.enableSuccess') : t('bots.schedules.disableSuccess'))
  } catch {
    toast.error(t('bots.schedules.toggleFailed'))
  }
}

async function confirmDelete(botId: string, schedule: ScheduleSchedule) {
  if (!confirm(t('bots.schedules.deleteConfirm'))) return
  try {
    await deleteBotsByBotIdScheduleById({
      path: { bot_id: botId, id: schedule.id! },
      throwOnError: true,
    })
    const group = allGroups.value.find(g => g.bot.id === botId)
    if (group) {
      group.schedules = group.schedules.filter(s => s.id !== schedule.id)
    }
    toast.success(t('bots.schedules.deleteSuccess'))
  } catch {
    toast.error(t('bots.schedules.deleteFailed'))
  }
}

onMounted(loadData)
</script>
