<template>
  <section class="h-full max-w-7xl mx-auto p-6">
    <div class="max-w-3xl mx-auto space-y-8">
      <div>
        <h1 class="text-2xl font-bold">{{ $t('systemSettings.title') }}</h1>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('systemSettings.subtitle') }}</p>
      </div>

      <!-- Display Settings -->
      <section>
        <h6 class="mb-2 flex items-center">
          <FontAwesomeIcon
            :icon="['fas', 'sliders']"
            class="mr-2"
          />
          {{ $t('systemSettings.display') }}
        </h6>
        <Separator />
        <div class="mt-4 space-y-4">
          <div class="flex items-center justify-between">
            <Label>{{ $t('settings.language') }}</Label>
            <Select
              :model-value="language"
              @update:model-value="(v) => v && setLanguage(v as Locale)"
            >
              <SelectTrigger class="w-40">
                <SelectValue :placeholder="$t('settings.languagePlaceholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="zh">
                    {{ $t('settings.langZh') }}
                  </SelectItem>
                  <SelectItem value="en">
                    {{ $t('settings.langEn') }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
          <Separator />
          <div class="flex items-center justify-between">
            <Label>{{ $t('settings.theme') }}</Label>
            <Select
              :model-value="theme"
              @update:model-value="(v) => v && setTheme(v as 'light' | 'dark')"
            >
              <SelectTrigger class="w-40">
                <SelectValue :placeholder="$t('settings.themePlaceholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="light">
                    {{ $t('settings.themeLight') }}
                  </SelectItem>
                  <SelectItem value="dark">
                    {{ $t('settings.themeDark') }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </div>
      </section>

      <!-- Timezone -->
      <section>
        <h6 class="mb-2 flex items-center">
          <FontAwesomeIcon
            :icon="['fas', 'clock']"
            class="mr-2"
          />
          {{ $t('settings.timezone') }}
        </h6>
        <Separator />
        <div class="mt-4">
          <div class="flex items-center justify-between">
            <div>
              <Label>{{ $t('systemSettings.timezoneLabel') }}</Label>
              <p class="text-xs text-muted-foreground mt-0.5">{{ $t('systemSettings.timezoneHint') }}</p>
            </div>
            <Select
              :model-value="serverTimezone"
              @update:model-value="(v) => v && onTimezoneChange(v)"
            >
              <SelectTrigger class="w-56">
                <SelectValue :placeholder="$t('settings.timezonePlaceholder')" />
              </SelectTrigger>
              <SelectContent class="max-h-60">
                <SelectGroup>
                  <SelectItem
                    v-for="tz in timezoneOptions"
                    :key="tz.value"
                    :value="tz.value"
                  >
                    {{ tz.label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </div>
      </section>

      <!-- System Diagnostics -->
      <section>
        <h6 class="mb-2 flex items-center">
          <FontAwesomeIcon
            :icon="['fas', 'stethoscope']"
            class="mr-2"
          />
          {{ $t('settings.diagnostics') }}
        </h6>
        <Separator />
        <div class="mt-4 space-y-4">
          <div class="flex items-center gap-3">
            <Button
              :disabled="runningDiagnostics"
              @click="onRunDiagnostics"
            >
              <Spinner v-if="runningDiagnostics" />
              {{ $t('settings.runDiagnostics') }}
            </Button>
            <span
              v-if="diagnosticsResult"
              class="text-sm font-medium"
              :class="{
                'text-green-600': diagnosticsResult.overall === 'healthy',
                'text-yellow-600': diagnosticsResult.overall === 'degraded',
                'text-red-600': diagnosticsResult.overall === 'unhealthy',
              }"
            >
              {{ diagnosticsResult.overall === 'healthy' ? $t('settings.diagHealthy') : diagnosticsResult.overall === 'degraded' ? $t('settings.diagDegraded') : $t('settings.diagUnhealthy') }}
            </span>
          </div>
          <div
            v-if="diagnosticsResult"
            class="space-y-2"
          >
            <div
              v-for="check in diagnosticsResult.checks"
              :key="check.name"
              class="flex items-center justify-between border rounded-md p-3"
            >
              <div class="flex items-center gap-2">
                <div
                  class="size-2.5 rounded-full shrink-0"
                  :class="{
                    'bg-green-500': check.status === 'ok',
                    'bg-yellow-500': check.status === 'warn',
                    'bg-red-500': check.status === 'error',
                  }"
                />
                <span class="font-medium text-sm">{{ check.name }}</span>
              </div>
              <div class="flex items-center gap-3 text-sm text-muted-foreground">
                <span class="truncate max-w-xs">{{ check.message }}</span>
                <span class="text-xs whitespace-nowrap">{{ check.latency_ms }}ms</span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  Button,
  Label,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Separator,
  Spinner,
} from '@memoh/ui'
import { onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { client } from '@memoh/sdk/client'
import { resolveErrorMessage } from '@/utils/error'
import { useSettingsStore } from '@/store/settings'
import type { Locale } from '@/i18n'

const { t } = useI18n()

const settingsStore = useSettingsStore()
const { language, theme } = storeToRefs(settingsStore)
const { setLanguage, setTheme } = settingsStore

const serverTimezone = ref('')

const timezoneOptions = [
  { value: 'UTC', label: 'UTC' },
  { value: 'Asia/Shanghai', label: 'Asia/Shanghai (CST, UTC+8)' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (JST, UTC+9)' },
  { value: 'Asia/Seoul', label: 'Asia/Seoul (KST, UTC+9)' },
  { value: 'Asia/Singapore', label: 'Asia/Singapore (SGT, UTC+8)' },
  { value: 'Asia/Hong_Kong', label: 'Asia/Hong_Kong (HKT, UTC+8)' },
  { value: 'Asia/Taipei', label: 'Asia/Taipei (CST, UTC+8)' },
  { value: 'Asia/Kolkata', label: 'Asia/Kolkata (IST, UTC+5:30)' },
  { value: 'Asia/Dubai', label: 'Asia/Dubai (GST, UTC+4)' },
  { value: 'Europe/London', label: 'Europe/London (GMT, UTC+0)' },
  { value: 'Europe/Paris', label: 'Europe/Paris (CET, UTC+1)' },
  { value: 'Europe/Berlin', label: 'Europe/Berlin (CET, UTC+1)' },
  { value: 'Europe/Moscow', label: 'Europe/Moscow (MSK, UTC+3)' },
  { value: 'America/New_York', label: 'America/New_York (EST, UTC-5)' },
  { value: 'America/Chicago', label: 'America/Chicago (CST, UTC-6)' },
  { value: 'America/Denver', label: 'America/Denver (MST, UTC-7)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (PST, UTC-8)' },
  { value: 'America/Sao_Paulo', label: 'America/Sao_Paulo (BRT, UTC-3)' },
  { value: 'Australia/Sydney', label: 'Australia/Sydney (AEST, UTC+10)' },
  { value: 'Pacific/Auckland', label: 'Pacific/Auckland (NZST, UTC+12)' },
]

interface DiagnosticCheck {
  name: string
  status: 'ok' | 'warn' | 'error'
  message: string
  latency_ms: number
}

interface DiagnosticsResult {
  checks: DiagnosticCheck[]
  overall: 'healthy' | 'degraded' | 'unhealthy'
  timestamp: string
}

const runningDiagnostics = ref(false)
const diagnosticsResult = ref<DiagnosticsResult | null>(null)

async function loadGlobalSettings() {
  try {
    const { data } = await client.get({ url: '/settings/global', throwOnError: true }) as { data: { timezone: string } }
    serverTimezone.value = data.timezone || 'UTC'
  } catch {
    serverTimezone.value = 'UTC'
  }
}

async function onTimezoneChange(tz: string) {
  try {
    await client.put({
      url: '/settings/global',
      body: { timezone: tz },
      throwOnError: true,
    })
    serverTimezone.value = tz
    toast.success(t('settings.timezoneSaved'))
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('settings.timezoneSaveFailed')))
  }
}

async function onRunDiagnostics() {
  runningDiagnostics.value = true
  try {
    const { data } = await client.get({ url: '/diagnostics', throwOnError: true }) as { data: DiagnosticsResult }
    diagnosticsResult.value = data
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('settings.diagFailed')))
  } finally {
    runningDiagnostics.value = false
  }
}

onMounted(() => {
  void loadGlobalSettings()
})
</script>
