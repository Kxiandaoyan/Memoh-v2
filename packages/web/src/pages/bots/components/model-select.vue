<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        variant="outline"
        role="combobox"
        :aria-expanded="open"
        class="w-full justify-between font-normal"
      >
        <span
          v-if="displayLabel"
          class="flex items-center gap-2 truncate"
        >
          <Badge
            v-if="displayProvider"
            variant="secondary"
            class="shrink-0 text-[10px] px-1.5 py-0"
          >
            {{ displayProvider }}
          </Badge>
          <span class="truncate">{{ displayLabel }}</span>
        </span>
        <span
          v-else
          class="text-muted-foreground"
        >{{ placeholder }}</span>
        <FontAwesomeIcon
          :icon="['fas', 'chevron-down']"
          class="ml-2 size-3 shrink-0 text-muted-foreground transition-transform"
          :class="{ 'rotate-180': open }"
        />
      </Button>
    </PopoverTrigger>
    <PopoverContent
      class="w-[--reka-popover-trigger-width] p-0"
      align="start"
    >
      <!-- Search input -->
      <div class="flex items-center border-b px-3">
        <FontAwesomeIcon
          :icon="['fas', 'magnifying-glass']"
          class="mr-2 size-3.5 shrink-0 text-muted-foreground"
        />
        <input
          ref="searchInput"
          v-model="searchTerm"
          :placeholder="$t('bots.settings.searchModel')"
          class="flex h-10 w-full bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground"
          @keydown.down.prevent="focusNext"
          @keydown.up.prevent="focusPrev"
          @keydown.enter.prevent="selectFocused"
          @keydown.escape="open = false"
        >
        <span
          v-if="searchTerm"
          class="text-xs text-muted-foreground shrink-0 tabular-nums"
        >{{ totalFilteredCount }}</span>
      </div>

      <!-- Model list -->
      <ScrollArea class="max-h-[min(400px,60vh)]">
        <!-- Clear option (for optional fields) -->
        <div
          v-if="selected && !required"
          class="p-1 border-b"
        >
          <button
            class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            :class="{ 'ring-1 ring-accent': focusedIndex === -1 }"
            @click="clearSelection"
          >
            <FontAwesomeIcon
              :icon="['fas', 'xmark']"
              class="size-3.5"
            />
            <span>{{ $t('bots.settings.clearModel') }}</span>
          </button>
        </div>

        <div
          v-if="filteredGroups.length === 0"
          class="py-6 text-center text-sm text-muted-foreground"
        >
          {{ $t('bots.settings.noModel') }}
        </div>

        <div
          v-for="group in filteredGroups"
          :key="group.providerName"
          class="p-1"
        >
          <div class="flex items-center gap-2 px-2 py-1.5 text-xs font-medium text-muted-foreground select-none">
            <span>{{ group.providerName }}</span>
            <span class="text-[10px] tabular-nums">({{ group.models.length }})</span>
          </div>
          <button
            v-for="model in group.models"
            :key="model.model_id"
            :ref="(el) => setItemRef(model.model_id, el as HTMLElement)"
            class="relative flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground transition-colors"
            :class="{
              'bg-accent/60 font-medium': selected === model.model_id,
              'ring-1 ring-primary/50': flatIndex(model.model_id) === focusedIndex,
            }"
            @click="selectModel(model.model_id)"
          >
            <FontAwesomeIcon
              v-if="selected === model.model_id"
              :icon="['fas', 'check']"
              class="size-3.5 text-primary shrink-0"
            />
            <span
              v-else
              class="size-3.5 shrink-0"
            />
            <span class="truncate">{{ model.name || model.model_id }}</span>
            <span
              v-if="model.name"
              class="ml-auto text-[11px] text-muted-foreground truncate max-w-[40%]"
            >
              {{ model.model_id }}
            </span>
          </button>
        </div>
      </ScrollArea>
    </PopoverContent>
  </Popover>
</template>

<script setup lang="ts">
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
  Button,
  ScrollArea,
  Badge,
} from '@memoh/ui'
import { computed, ref, watch, nextTick } from 'vue'
import type { ModelsGetResponse, ProvidersGetResponse } from '@memoh/sdk'

const props = withDefaults(defineProps<{
  models: ModelsGetResponse[]
  providers: ProvidersGetResponse[]
  modelType: 'chat' | 'embedding'
  placeholder?: string
  required?: boolean
}>(), {
  required: false,
})

const selected = defineModel<string>({ default: '' })
const searchTerm = ref('')
const open = ref(false)
const searchInput = ref<HTMLInputElement | null>(null)
const focusedIndex = ref(-2)
const itemRefs = new Map<string, HTMLElement>()

function setItemRef(id: string, el: HTMLElement | null) {
  if (el) itemRefs.set(id, el)
  else itemRefs.delete(id)
}

watch(open, (val) => {
  if (val) {
    searchTerm.value = ''
    focusedIndex.value = -2
    nextTick(() => searchInput.value?.focus())
  } else {
    itemRefs.clear()
  }
})

const typeFilteredModels = computed(() =>
  props.models.filter((m) => m.type === props.modelType),
)

const providerMap = computed(() => {
  const map = new Map<string, string>()
  for (const p of props.providers) {
    map.set(p.id, p.name ?? p.id)
  }
  return map
})

const filteredGroups = computed(() => {
  const keyword = searchTerm.value.trim().toLowerCase()
  const models = keyword
    ? typeFilteredModels.value.filter((m) => {
      const providerName = providerMap.value.get(m.llm_provider_id) ?? ''
      return (
        m.model_id.toLowerCase().includes(keyword)
        || (m.name?.toLowerCase().includes(keyword) ?? false)
        || providerName.toLowerCase().includes(keyword)
      )
    })
    : typeFilteredModels.value

  const groups = new Map<string, { providerName: string; models: ModelsGetResponse[] }>()
  for (const model of models) {
    const pid = model.llm_provider_id
    const providerName = providerMap.value.get(pid) ?? pid
    if (!groups.has(pid)) {
      groups.set(pid, { providerName, models: [] })
    }
    groups.get(pid)!.models.push(model)
  }
  return Array.from(groups.values())
})

const flatModelIds = computed(() => {
  const ids: string[] = []
  for (const g of filteredGroups.value) {
    for (const m of g.models) {
      ids.push(m.model_id)
    }
  }
  return ids
})

const totalFilteredCount = computed(() => flatModelIds.value.length)

function flatIndex(modelId: string): number {
  return flatModelIds.value.indexOf(modelId)
}

function focusNext() {
  const max = flatModelIds.value.length - 1
  if (max < 0) return
  focusedIndex.value = Math.min(focusedIndex.value + 1, max)
  scrollToFocused()
}

function focusPrev() {
  const min = selected.value && !props.required ? -1 : 0
  focusedIndex.value = Math.max(focusedIndex.value - 1, min)
  scrollToFocused()
}

function scrollToFocused() {
  nextTick(() => {
    const id = flatModelIds.value[focusedIndex.value]
    if (!id) return
    const el = itemRefs.get(id)
    el?.scrollIntoView({ block: 'nearest' })
  })
}

function selectFocused() {
  if (focusedIndex.value === -1 && !props.required) {
    clearSelection()
    return
  }
  const id = flatModelIds.value[focusedIndex.value]
  if (id) selectModel(id)
}

const displayLabel = computed(() => {
  if (!selected.value) return ''
  const model = props.models.find((m) => m.model_id === selected.value)
  return model?.name || model?.model_id || selected.value
})

const displayProvider = computed(() => {
  if (!selected.value) return ''
  const model = props.models.find((m) => m.model_id === selected.value)
  if (!model) return ''
  return providerMap.value.get(model.llm_provider_id) ?? ''
})

function selectModel(modelId: string) {
  selected.value = modelId
  open.value = false
}

function clearSelection() {
  selected.value = ''
  open.value = false
}
</script>
