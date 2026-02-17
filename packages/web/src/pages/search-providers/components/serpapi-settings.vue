<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <Label>API Key</Label>
      <Input v-model="localConfig.api_key" type="password" />
    </div>
    <div class="space-y-2">
      <Label>{{ $t('searchProvider.serpapi.engine') }}</Label>
      <Select
        :model-value="localConfig.engine"
        @update:model-value="(val) => localConfig.engine = val"
      >
        <SelectTrigger class="w-full">
          <SelectValue :placeholder="$t('searchProvider.serpapi.enginePlaceholder')" />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem
              v-for="eng in ENGINES"
              :key="eng.value"
              :value="eng.value"
            >
              {{ eng.label }}
            </SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
    <div class="space-y-2">
      <Label>Base URL</Label>
      <Input v-model="localConfig.base_url" />
    </div>
    <div class="space-y-2">
      <Label>Timeout (seconds)</Label>
      <Input v-model.number="localConfig.timeout_seconds" type="number" :min="1" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import {
  Input,
  Label,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectGroup,
  SelectItem,
} from '@memoh/ui'

const ENGINES = [
  { value: 'google', label: 'Google' },
  { value: 'bing', label: 'Bing' },
  { value: 'duckduckgo', label: 'DuckDuckGo' },
  { value: 'yahoo', label: 'Yahoo' },
  { value: 'baidu', label: 'Baidu' },
  { value: 'yandex', label: 'Yandex' },
  { value: 'naver', label: 'Naver' },
] as const

const props = defineProps<{
  modelValue: Record<string, unknown>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const localConfig = reactive({
  api_key: '',
  engine: 'google',
  base_url: 'https://serpapi.com/search.json',
  timeout_seconds: 15,
})

watch(
  () => props.modelValue,
  (val) => {
    localConfig.api_key = String(val?.api_key ?? '')
    localConfig.engine = String(val?.engine ?? 'google')
    localConfig.base_url = String(val?.base_url ?? 'https://serpapi.com/search.json')
    const timeout = Number(val?.timeout_seconds ?? 15)
    localConfig.timeout_seconds = Number.isFinite(timeout) && timeout > 0 ? timeout : 15
  },
  { immediate: true, deep: true },
)

watch(localConfig, () => {
  emit('update:modelValue', {
    api_key: localConfig.api_key,
    engine: localConfig.engine,
    base_url: localConfig.base_url,
    timeout_seconds: localConfig.timeout_seconds,
  })
}, { deep: true })
</script>
