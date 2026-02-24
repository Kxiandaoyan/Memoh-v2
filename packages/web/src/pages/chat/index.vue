<template>
  <div class="flex h-[calc(100vh-calc(var(--spacing)*20))]">
    <!-- Desktop: fixed bot sidebar -->
    <div class="hidden md:flex w-56 shrink-0 border-r flex-col">
      <BotSidebar />
    </div>

    <!-- Mobile: bot sidebar in a Sheet drawer -->
    <Sheet v-model:open="mobileSidebarOpen">
      <SheetContent side="left" class="p-0 w-64">
        <BotSidebar />
      </SheetContent>
    </Sheet>

    <!-- Chat area (takes remaining width) -->
    <ChatArea @toggle-bot-list="mobileSidebarOpen = !mobileSidebarOpen" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Sheet, SheetContent } from '@memoh/ui'
import { storeToRefs } from 'pinia'
import { useChatStore } from '@/store/chat-list'
import BotSidebar from './components/bot-sidebar.vue'
import ChatArea from './components/chat-area.vue'

const chatStore = useChatStore()
const { currentBotId } = storeToRefs(chatStore)

const mobileSidebarOpen = ref(false)

// Close the mobile drawer after the user picks a bot
watch(currentBotId, () => {
  mobileSidebarOpen.value = false
})
</script>
