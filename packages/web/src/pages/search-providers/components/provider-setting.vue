<template>
  <div class="p-4">
    <section class="flex justify-between items-center">
      <div class="flex items-center gap-2">
        <SearchProviderLogo
          :provider="curProvider?.provider || ''"
          size="lg"
        />
        <h4 class="scroll-m-20 tracking-tight">
          {{ curProvider?.name }}
        </h4>
      </div>
    </section>
    <Separator class="mt-4 mb-6" />

    <form @submit="editProvider">
      <div class="**:[input]:mt-3 **:[input]:mb-4">
        <section>
          <h4 class="scroll-m-20 font-semibold tracking-tight">
            {{ $t('common.name') }}
          </h4>
          <FormField
            v-slot="{ componentField }"
            name="name"
          >
            <FormItem>
              <FormControl>
                <Input
                  type="text"
                  :placeholder="$t('common.namePlaceholder')"
                  v-bind="componentField"
                />
              </FormControl>
            </FormItem>
          </FormField>
        </section>

        <section>
          <h4 class="scroll-m-20 font-semibold tracking-tight">
            {{ $t('searchProvider.provider') }}
          </h4>
          <div class="mt-3 mb-4">
            <Select
              :model-value="form.values.provider"
              @update:model-value="(val) => form.setFieldValue('provider', val)"
            >
              <SelectTrigger class="w-full">
                <SelectValue :placeholder="$t('common.typePlaceholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem
                    v-for="type in PROVIDER_TYPES"
                    :key="type"
                    :value="type"
                  >
                    {{ type }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </section>

        <Separator class="my-4" />

        <template v-if="form.values.provider === 'brave'">
          <BraveSettings v-model="configProxy" />
        </template>
        <div
          v-else-if="form.values.provider"
          class="text-sm text-muted-foreground"
        >
          {{ $t('searchProvider.unsupportedProvider') }}
        </div>
      </div>

      <section class="flex justify-end mt-4 gap-4">
        <ConfirmPopover
          :message="$t('searchProvider.deleteConfirm')"
          :loading="deleteLoading"
          @confirm="deleteProvider"
        >
          <template #trigger>
            <Button variant="outline">
              <FontAwesomeIcon :icon="['far', 'trash-can']" />
            </Button>
          </template>
        </ConfirmPopover>

        <Button
          type="submit"
          :disabled="editLoading"
        >
          <Spinner v-if="editLoading" />
          {{ $t('provider.saveChanges') }}
        </Button>
      </section>
    </form>
  </div>
</template>

<script setup lang="ts">
import {
  Input,
  Button,
  FormControl,
  FormField,
  FormItem,
  Spinner,
  Separator,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectGroup,
  SelectItem,
} from '@memoh/ui'
import ConfirmPopover from '@/components/confirm-popover/index.vue'
import BraveSettings from './brave-settings.vue'
import SearchProviderLogo from '@/components/search-provider-logo/index.vue'
import { computed, inject, ref, watch } from 'vue'
import { toTypedSchema } from '@vee-validate/zod'
import z from 'zod'
import { useForm } from 'vee-validate'
import { useMutation, useQueryCache } from '@pinia/colada'
import { putSearchProvidersById, deleteSearchProvidersById } from '@memoh/sdk'
import type { SearchprovidersGetResponse } from '@memoh/sdk'

const PROVIDER_TYPES = ['brave'] as const

const curProvider = inject('curSearchProvider', ref<SearchprovidersGetResponse>())
const curProviderId = computed(() => curProvider.value?.id)

const queryCache = useQueryCache()

// ---- form ----
const providerSchema = toTypedSchema(z.object({
  name: z.string().min(1),
  provider: z.string().min(1),
}))

const form = useForm({
  validationSchema: providerSchema,
})

// Store config separately since it varies by provider type
const configData = ref<Record<string, unknown>>({})

const configProxy = computed({
  get: () => configData.value,
  set: (val: Record<string, unknown>) => {
    configData.value = val
  },
})

watch(curProvider, (newVal) => {
  if (newVal) {
    form.setValues({
      name: newVal.name ?? '',
      provider: newVal.provider ?? '',
    })
    configData.value = { ...(newVal.config ?? {}) }
  }
}, { immediate: true })

// ---- mutations ----
const { mutate: submitUpdate, isLoading: editLoading } = useMutation({
  mutation: async (data: { name: string; provider: string; config: Record<string, unknown> }) => {
    if (!curProviderId.value) return
    const { data: result } = await putSearchProvidersById({
      path: { id: curProviderId.value },
      body: data as any,
      throwOnError: true,
    })
    return result
  },
  onSettled: () => queryCache.invalidateQueries({ key: ['search-providers'] }),
})

const { mutate: deleteProvider, isLoading: deleteLoading } = useMutation({
  mutation: async () => {
    if (!curProviderId.value) return
    await deleteSearchProvidersById({ path: { id: curProviderId.value }, throwOnError: true })
  },
  onSettled: () => queryCache.invalidateQueries({ key: ['search-providers'] }),
})

const editProvider = form.handleSubmit(async (values) => {
  submitUpdate({
    name: values.name,
    provider: values.provider,
    config: configData.value,
  })
})
</script>
