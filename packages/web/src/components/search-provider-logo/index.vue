<template>
  <FontAwesomeIcon
    :icon="icon"
    class="shrink-0"
    :class="sizeClass"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getSearchProviderIcon } from './icons'

const props = withDefaults(defineProps<{
  /** The provider key, e.g. 'brave' */
  provider: string
  /** Size preset */
  size?: 'xs' | 'sm' | 'md' | 'lg'
}>(), {
  size: 'sm',
})

const sizeClass = computed(() => {
  switch (props.size) {
    case 'xs': return 'size-3'
    case 'sm': return 'size-3.5'
    case 'md': return 'size-4'
    case 'lg': return 'size-5'
    default: return 'size-3.5'
  }
})

const icon = computed(() => getSearchProviderIcon(props.provider))
</script>
