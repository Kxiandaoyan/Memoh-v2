<template>
  <div class="space-y-2">
    <div
      v-for="run in block.runs"
      :key="run.runId"
      class="flex items-start gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-xs"
    >
      <FontAwesomeIcon
        :icon="run.status === 'running' ? ['fas', 'spinner'] : run.status === 'completed' ? ['fas', 'check-circle'] : ['fas', 'times-circle']"
        :class="[
          'size-3.5 mt-0.5 shrink-0',
          run.status === 'running' ? 'animate-spin text-blue-500' : run.status === 'completed' ? 'text-green-500' : 'text-red-500',
        ]"
      />
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="font-medium truncate">{{ run.name }}</span>
          <span class="text-muted-foreground shrink-0">{{ formatElapsed(run.elapsed_ms) }}</span>
        </div>
        <p class="text-muted-foreground truncate">{{ run.task }}</p>
        <p
          v-if="run.delta"
          class="mt-1 text-foreground/70 line-clamp-2 whitespace-pre-wrap"
        >{{ run.delta.slice(-200) }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SubagentProgressBlock } from '@/store/chat-list'

defineProps<{ block: SubagentProgressBlock }>()

function formatElapsed(ms: number): string {
  const s = Math.round(ms / 1000)
  return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m${s % 60}s`
}
</script>
