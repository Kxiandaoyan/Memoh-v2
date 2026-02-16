<template>
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
          {{ $t('provider.apiKey') }}
        </h4>
        <FormField
          v-slot="{ componentField }"
          name="api_key"
        >
          <FormItem>
            <FormControl>
              <Input
                type="password"
                :placeholder="props.provider?.api_key || $t('provider.apiKeyPlaceholder')"
                v-bind="componentField"
              />
            </FormControl>
          </FormItem>
        </FormField>
      </section>

      <section>
        <h4 class="scroll-m-20 font-semibold tracking-tight">
          {{ $t('provider.url') }}
        </h4>
        <FormField
          v-slot="{ componentField }"
          name="base_url"
        >
          <FormItem>
            <FormControl>
              <Input
                type="text"
                :placeholder="$t('provider.urlPlaceholder')"
                v-bind="componentField"
              />
            </FormControl>
          </FormItem>
        </FormField>
      </section>
    </div>

    <section class="flex justify-end mt-4 gap-4">
      <ConfirmPopover
        :message="$t('provider.deleteConfirm')"
        :loading="deleteLoading"
        @confirm="$emit('delete')"
      >
        <template #trigger>
          <Button variant="outline">
            <FontAwesomeIcon :icon="['far', 'trash-can']" />
          </Button>
        </template>
      </ConfirmPopover>

      <Button
        type="submit"
        :disabled="!hasChanges || !form.meta.value.valid"
      >
        <Spinner v-if="editLoading" />
        {{ $t('provider.saveChanges') }}
      </Button>
    </section>
  </form>
</template>

<script setup lang="ts">
import {
  Input,
  Button,
  FormControl,
  FormField,
  FormItem,
  Spinner,
} from '@memoh/ui'
import ConfirmPopover from '@/components/confirm-popover/index.vue'
import { computed, watch } from 'vue'
import { toTypedSchema } from '@vee-validate/zod'
import z from 'zod'
import { useForm } from 'vee-validate'
import type { ProvidersGetResponse } from '@memoh/sdk'

const props = defineProps<{
  provider: Partial<ProvidersGetResponse> | undefined
  editLoading: boolean
  deleteLoading: boolean
}>()

const emit = defineEmits<{
  submit: [values: Record<string, unknown>]
  delete: []
}>()

const providerSchema = toTypedSchema(z.object({
  name: z.string().min(1),
  base_url: z.string().min(1),
  client_type: z.string().min(1),
  api_key: z.string().optional(),
  metadata: z.object({
    additionalProp1: z.object({}),
  }),
}))

const form = useForm({
  validationSchema: providerSchema,
})

watch(() => props.provider, (newVal) => {
  if (newVal) {
    form.setValues({
      name: newVal.name,
      base_url: newVal.base_url,
      client_type: newVal.client_type,
      // Keep key input empty by default so masked placeholders are never submitted back.
      api_key: '',
    })
  }
}, { immediate: true })

const hasChanges = computed(() => {
  const raw = props.provider
  const baseChanged = JSON.stringify({
    name: form.values.name,
    base_url: form.values.base_url,
    client_type: form.values.client_type,
    metadata: form.values.metadata,
  }) !== JSON.stringify({
    name: raw?.name,
    base_url: raw?.base_url,
    client_type: raw?.client_type,
    metadata: { additionalProp1: {} },
  })

  const apiKeyChanged = Boolean(form.values.api_key && form.values.api_key.trim() !== '')
  return baseChanged || apiKeyChanged
})

const editProvider = form.handleSubmit(async (value) => {
  const payload: Record<string, unknown> = {
    name: value.name,
    base_url: value.base_url,
    client_type: value.client_type,
    metadata: value.metadata,
  }
  if (value.api_key && value.api_key.trim() !== '') {
    payload.api_key = value.api_key
  }
  emit('submit', payload)
})
</script>
