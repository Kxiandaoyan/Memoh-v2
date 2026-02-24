<template>
  <Transition name="scroll-btn">
    <button
      v-if="visible"
      class="absolute bottom-20 right-5 z-10 w-9 h-9 rounded-full bg-white shadow-lg border border-gray-100 flex items-center justify-center text-gray-500 hover:text-gray-700 hover:shadow-xl hover:scale-105 active:scale-95 transition-all duration-150"
      title="回到底部"
      @click="$emit('click')"
    >
      <span
        v-if="unreadCount > 0"
        class="absolute -top-1.5 -right-1.5 min-w-[18px] h-[18px] rounded-full bg-primary text-primary-foreground text-[10px] font-semibold flex items-center justify-center px-1"
      >
        {{ unreadCount > 99 ? '99+' : unreadCount }}
      </span>
      <FontAwesomeIcon :icon="['fas', 'arrow-down']" class="size-3.5" />
    </button>
  </Transition>
</template>

<script setup lang="ts">
defineProps<{
  visible: boolean
  unreadCount: number
}>()

defineEmits<{ click: [] }>()
</script>

<style scoped>
.scroll-btn-enter-active,
.scroll-btn-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.scroll-btn-enter-from,
.scroll-btn-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
