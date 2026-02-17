<template>
  <section class="ml-auto">
    <Dialog v-model:open="open">
      <DialogTrigger as-child>
        <Button variant="default">
          {{ $t('models.addModel') }}
        </Button>
      </DialogTrigger>
      <DialogContent class="sm:max-w-106.25">
        <form @submit="addModel">
          <DialogHeader>
            <DialogTitle>
              {{ title === 'edit' ? $t('models.editModel') : $t('models.addModel') }}
            </DialogTitle>
            <DialogDescription class="mb-4">
              <Separator class="my-4" />
            </DialogDescription>
          </DialogHeader>
          <div class="flex flex-col gap-3">
            <!-- Type -->
            <FormField
              v-slot="{ componentField }"
              name="type"
            >
              <FormItem>
                <Label class="mb-2">
                  {{ $t('common.type') }}
                </Label>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger class="w-full">
                      <SelectValue :placeholder="$t('common.typePlaceholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="chat">
                          Chat
                        </SelectItem>
                        <SelectItem value="embedding">
                          Embedding
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Model -->
            <FormField
              v-slot="{ componentField }"
              name="model_id"
            >
              <FormItem>
                <Label class="mb-2">
                  {{ $t('models.model') }}
                </Label>
                <p class="text-xs text-muted-foreground">{{ $t('models.modelHint') }}</p>
                <FormControl>
                  <Input
                    type="text"
                    :placeholder="$t('models.modelPlaceholder')"
                    v-bind="componentField"
                  />
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Display Name -->
            <FormField
              name="name"
            >
              <FormItem>
                <Label class="mb-2">
                  {{ $t('models.displayName') }}
                  <span class="text-muted-foreground text-xs ml-1">({{ $t('common.optional') }})</span>
                </Label>
                <p class="text-xs text-muted-foreground">{{ $t('models.displayNameHint') }}</p>
                <FormControl>
                  <Input
                    type="text"
                    :placeholder="$t('models.displayNamePlaceholder')"
                    :model-value="form.values.name ?? ''"
                    @input="onNameInput"
                  />
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Dimensions (embedding only) -->
            <FormField
              v-if="selectedType === 'embedding'"
              v-slot="{ componentField }"
              name="dimensions"
            >
              <FormItem>
                <Label class="mb-2">
                  {{ $t('models.dimensions') }}
                </Label>
                <p class="text-xs text-muted-foreground">{{ $t('models.dimensionsHint') }}</p>
                <FormControl>
                  <Input
                    type="number"
                    :placeholder="$t('models.dimensionsPlaceholder')"
                    v-bind="componentField"
                  />
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Context Window (chat only) -->
            <FormField
              v-if="selectedType === 'chat'"
              v-slot="{ componentField }"
              name="context_window"
            >
              <FormItem>
                <Label class="mb-2">
                  Context Window
                  <span class="text-muted-foreground text-xs ml-1">(tokens)</span>
                </Label>
                <FormControl>
                  <Input
                    type="number"
                    placeholder="128000"
                    v-bind="componentField"
                  />
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Fallback Model (chat only) -->
            <FormField
              v-if="selectedType === 'chat'"
              name="fallback_model_id"
            >
              <FormItem>
                <Label class="mb-2">
                  {{ $t('models.fallbackModel') }}
                  <span class="text-muted-foreground text-xs ml-1">({{ $t('common.optional') }})</span>
                </Label>
                <p class="text-xs text-muted-foreground">{{ $t('models.fallbackModelHint') }}</p>
                <FormControl>
                  <Select
                    :model-value="form.values.fallback_model_id || ''"
                    @update:model-value="(v: string) => form.setFieldValue('fallback_model_id', v === '__none__' ? '' : v)"
                  >
                    <SelectTrigger class="w-full">
                      <SelectValue :placeholder="$t('models.fallbackModelPlaceholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="__none__">
                          {{ $t('models.noFallback') }}
                        </SelectItem>
                        <SelectItem
                          v-for="m in availableFallbackModels"
                          :key="m.model_id"
                          :value="m.model_id"
                        >
                          {{ m.name || m.model_id }}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Multimodal (chat only) -->
            <FormField
              v-if="selectedType === 'chat'"
              v-slot="{ componentField }"
              name="is_multimodal"
            >
              <FormItem class="flex items-center justify-between">
                <div>
                  <Label>
                    {{ $t('models.multimodal') }}
                  </Label>
                  <p class="text-xs text-muted-foreground">{{ $t('models.multimodalHint') }}</p>
                </div>
                <Switch
                  v-model="componentField.modelValue"
                  @update:model-value="componentField['onUpdate:modelValue']"
                />
              </FormItem>
            </FormField>
          </div>
          <DialogFooter class="mt-4">
            <DialogClose as-child>
              <Button variant="outline">
                {{ $t('common.cancel') }}
              </Button>
            </DialogClose>
            <Button
              type="submit"
              :disabled="!canSubmit"
            >
              <Spinner v-if="isLoading" />
              {{ title === 'edit' ? $t('common.save') : $t('models.addModel') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </section>
</template>

<script setup lang="ts">
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  Input,
  Button,
  FormField,
  FormControl,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  FormItem,
  Switch,
  Separator,
  Label,
  Spinner,
} from '@memoh/ui'
import { useForm } from 'vee-validate'
import { inject, computed, watch, nextTick, type Ref, ref } from 'vue'
import { toTypedSchema } from '@vee-validate/zod'
import z from 'zod'
import { useMutation, useQueryCache } from '@pinia/colada'
import { postModels, putModelsModelByModelId, getModels } from '@memoh/sdk'
import type { ModelsGetResponse } from '@memoh/sdk'

const formSchema = toTypedSchema(z.object({
  type: z.string().min(1),
  model_id: z.string().min(1),
  name: z.string().optional(),
  dimensions: z.coerce.number().min(1).optional(),
  is_multimodal: z.coerce.boolean().optional(),
  context_window: z.coerce.number().min(1).optional(),
  fallback_model_id: z.string().optional(),
}))

const form = useForm({
  validationSchema: formSchema,
})

const selectedType = computed(() => form.values.type || editInfo?.value?.type)

const open = inject<Ref<boolean>>('openModel', ref(false))
const title = inject<Ref<'edit' | 'title'>>('openModelTitle', ref('title'))
const editInfo = inject<Ref<ModelsGetResponse | null>>('openModelState', ref(null))

// 保存按钮：编辑模式直接可提交（表单已预填充，handleSubmit 内部会校验）
// 新建模式需要必填字段有值
const canSubmit = computed(() => {
  if (title.value === 'edit') return true
  const { type, model_id } = form.values
  return !!type && !!model_id
})

// 新建时的空值
const emptyValues = {
  type: '' as string,
  model_id: '' as string,
  name: '' as string,
  dimensions: undefined as number | undefined,
  is_multimodal: undefined as boolean | undefined,
  context_window: 128000 as number | undefined,
  fallback_model_id: '' as string | undefined,
}

// Display Name 自动跟随 Model ID，除非用户主动修改过
const userEditedName = ref(false)

// Available chat models for fallback selection
const allChatModels = ref<Array<{ model_id: string; name: string }>>([])

const availableFallbackModels = computed(() => {
  const currentModelId = form.values.model_id || editInfo?.value?.model_id
  return allChatModels.value.filter(m => m.model_id !== currentModelId)
})

watch(
  () => form.values.model_id,
  (newModelId) => {
    if (!userEditedName.value && newModelId !== undefined) {
      form.setFieldValue('name', newModelId)
    }
  },
)

function onNameInput(e: Event) {
  userEditedName.value = true
  form.setFieldValue('name', (e.target as HTMLInputElement).value)
}

const { id } = defineProps<{ id: string }>()

const queryCache = useQueryCache()
const { mutateAsync: createModel, isLoading: createLoading } = useMutation({
  mutation: async (data: Record<string, unknown>) => {
    const { data: result } = await postModels({ body: data as any, throwOnError: true })
    return result
  },
  onSettled: () => queryCache.invalidateQueries({ key: ['provider-models'] }),
})
const { mutateAsync: updateModel, isLoading: updateLoading } = useMutation({
  mutation: async ({ modelId, data }: { modelId: string; data: Record<string, unknown> }) => {
    const { data: result } = await putModelsModelByModelId({
      path: { modelId },
      body: data as any,
      throwOnError: true,
    })
    return result
  },
  onSettled: () => queryCache.invalidateQueries({ key: ['provider-models'] }),
})
const isLoading = computed(() => createLoading.value || updateLoading.value)

async function addModel(e: Event) {
  e.preventDefault()

  const isEdit = title.value === 'edit' && !!editInfo?.value
  const fallback = editInfo?.value

  // 从 form.values 读取，编辑模式用 editInfo 兜底
  // （Dialog 异步渲染可能导致 vee-validate 内部状态未同步）
  const type = form.values.type || (isEdit ? fallback!.type : '')
  const model_id = form.values.model_id || (isEdit ? fallback!.model_id : '')
  const name = form.values.name ?? (isEdit ? fallback!.name : '')
  const dimensions = form.values.dimensions ?? (isEdit ? fallback!.dimensions : undefined)
  const is_multimodal = form.values.is_multimodal ?? (isEdit ? fallback!.is_multimodal : undefined)
  const context_window = form.values.context_window ?? (isEdit ? (fallback as any)!.context_window : 128000)

  if (!type || !model_id) return

  try {
    const payload: Record<string, unknown> = {
      type,
      model_id,
      llm_provider_id: id,
    }

    if (name) {
      payload.name = name
    }

    if (type === 'embedding' && dimensions) {
      payload.dimensions = dimensions
    }

    if (type === 'chat') {
      payload.is_multimodal = is_multimodal ?? false
      payload.context_window = context_window || 128000
      const fallback_model_id = form.values.fallback_model_id ?? (isEdit ? (fallback as any)?.fallback_model_id : '')
      if (fallback_model_id) {
        payload.fallback_model_id = fallback_model_id
      }
    }

    if (isEdit) {
      await updateModel({ modelId: fallback!.model_id, data: payload as any })
    } else {
      await createModel(payload as any)
    }
    open.value = false
  } catch {
    return
  }
}

watch(open, async () => {
  if (!open.value) {
    title.value = 'title'
    editInfo.value = null
    return
  }

  // 等待 Dialog 内容和 FormField 组件挂载完成
  await nextTick()

  // Load available chat models for fallback selection
  getModels({ query: { type: 'chat' }, throwOnError: false }).then((res) => {
    allChatModels.value = (res.data as any[]) || []
  }).catch(() => {})

  if (editInfo?.value) {
    const { type, model_id, name, dimensions, is_multimodal } = editInfo.value
    const context_window = (editInfo.value as any).context_window || 128000
    const fallback_model_id = (editInfo.value as any).fallback_model_id || ''
    form.resetForm({ values: { type, model_id, name, dimensions, is_multimodal, context_window, fallback_model_id } })
    // 编辑时，如果已有 name 且与 model_id 不同，视为用户自定义
    userEditedName.value = !!(name && name !== model_id)
  } else {
    // 新建模式：显式传空值，避免复用上次编辑数据
    form.resetForm({ values: { ...emptyValues } })
    userEditedName.value = false
  }
}, {
  immediate: true,
})
</script>
