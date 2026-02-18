<template>
  <Dialog v-model:open="open">
    <DialogTrigger as-child>
      <slot name="trigger">
        <Button variant="default">
          <FontAwesomeIcon
            :icon="['fas', 'plus']"
            class="mr-1.5"
          />
          {{ $t('bots.createBot') }}
        </Button>
      </slot>
    </DialogTrigger>
    <DialogContent :class="step === 0 ? 'sm:max-w-2xl' : 'sm:max-w-md'">
      <!-- Step 0: Template Selection -->
      <template v-if="step === 0">
        <DialogHeader>
          <DialogTitle>{{ $t('bots.templates.title') }}</DialogTitle>
          <DialogDescription>
            {{ $t('bots.templates.subtitle') }}
          </DialogDescription>
        </DialogHeader>

        <!-- Category Tabs -->
        <div class="flex gap-1.5 flex-wrap mt-2">
          <Button
            v-for="cat in categories"
            :key="cat"
            :variant="selectedCategory === cat ? 'default' : 'outline'"
            size="sm"
            @click="selectedCategory = cat"
          >
            {{ $t(`bots.templates.categories.${cat}`, cat) }}
          </Button>
        </div>

        <!-- Template Grid -->
        <div class="grid grid-cols-2 gap-3 mt-4 max-h-[400px] overflow-y-auto pr-1">
          <!-- Blank Bot Card -->
          <div
            class="border rounded-lg p-3 cursor-pointer transition-all hover:border-primary"
            :class="selectedTemplateId === '' ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'border-border'"
            @click="selectTemplate('')"
          >
            <div class="flex items-center gap-2 mb-1.5">
              <div class="w-8 h-8 rounded-md bg-muted flex items-center justify-center">
                <FontAwesomeIcon :icon="['fas', 'file']" class="text-muted-foreground text-sm" />
              </div>
              <span class="font-medium text-sm">{{ $t('bots.templates.blank') }}</span>
            </div>
            <p class="text-xs text-muted-foreground line-clamp-2">{{ $t('bots.templates.blankDescription') }}</p>
          </div>

          <!-- Template Cards -->
          <div
            v-for="tmpl in filteredTemplates"
            :key="tmpl.id"
            class="border rounded-lg p-3 cursor-pointer transition-all hover:border-primary"
            :class="selectedTemplateId === tmpl.id ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'border-border'"
            @click="selectTemplate(tmpl.id)"
          >
            <div class="flex items-center gap-2 mb-1.5">
              <div class="w-8 h-8 rounded-md bg-muted flex items-center justify-center">
                <FontAwesomeIcon :icon="['fas', tmpl.icon]" class="text-muted-foreground text-sm" />
              </div>
              <span class="font-medium text-sm">{{ tmpl.name }}</span>
            </div>
            <p class="text-xs text-muted-foreground line-clamp-2">{{ tmpl.description }}</p>
          </div>
        </div>

        <DialogFooter class="mt-4">
          <DialogClose as-child>
            <Button variant="outline">
              {{ $t('common.cancel') }}
            </Button>
          </DialogClose>
          <Button @click="step = 1">
            {{ $t('bots.next') }}
          </Button>
        </DialogFooter>
      </template>

      <!-- Step 1: Bot Configuration Form -->
      <template v-else>
        <form @submit="handleSubmit">
          <DialogHeader>
            <DialogTitle>{{ $t('bots.createBot') }}</DialogTitle>
            <DialogDescription>
              <div class="flex items-center gap-2 mt-1">
                <span class="text-xs text-muted-foreground">{{ $t('bots.step', { current: 2, total: 2 }) }}</span>
                <span v-if="selectedTemplateName" class="text-xs bg-primary/10 text-primary px-1.5 py-0.5 rounded">
                  {{ selectedTemplateName }}
                </span>
              </div>
              <Separator class="my-3" />
            </DialogDescription>
          </DialogHeader>

          <div class="flex flex-col gap-4">
            <!-- Display Name -->
            <FormField
              v-slot="{ componentField }"
              name="display_name"
            >
              <FormItem>
                <Label class="mb-2">{{ $t('bots.displayName') }}</Label>
                <FormControl>
                  <Input
                    type="text"
                    :placeholder="$t('bots.displayNamePlaceholder')"
                    v-bind="componentField"
                  />
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Avatar URL -->
            <FormField
              v-slot="{ componentField }"
              name="avatar_url"
            >
              <FormItem>
                <Label class="mb-2">
                  {{ $t('bots.avatarUrl') }}
                  <span class="text-muted-foreground text-xs ml-1">({{ $t('common.optional') }})</span>
                </Label>
                <FormControl>
                  <Input
                    type="text"
                    :placeholder="$t('bots.avatarUrlPlaceholder')"
                    v-bind="componentField"
                  />
                </FormControl>
              </FormItem>
            </FormField>

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
                      <SelectValue :placeholder="$t('bots.typePlaceholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="personal">
                          <div>
                            <div>{{ $t('bots.types.personal') }}</div>
                            <p class="text-xs text-muted-foreground font-normal">{{ $t('bots.typePersonalHint') }}</p>
                          </div>
                        </SelectItem>
                        <SelectItem value="public">
                          <div>
                            <div>{{ $t('bots.types.public') }}</div>
                            <p class="text-xs text-muted-foreground font-normal">{{ $t('bots.typePublicHint') }}</p>
                          </div>
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
              </FormItem>
            </FormField>
            <!-- Privileged -->
            <FormField
              v-slot="{ value, handleChange }"
              name="is_privileged"
            >
              <FormItem>
                <div class="flex items-center justify-between">
                  <div class="space-y-0.5">
                    <Label>{{ $t('bots.settings.isPrivileged') }}</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('bots.settings.isPrivilegedHint') }}</p>
                  </div>
                  <Switch
                    :checked="value"
                    @update:checked="handleChange"
                  />
                </div>
              </FormItem>
            </FormField>
          </div>

          <DialogFooter class="mt-6">
            <Button
              variant="outline"
              type="button"
              @click="step = 0"
            >
              {{ $t('common.back') }}
            </Button>
            <Button
              type="submit"
              :disabled="!form.meta.value.valid || submitLoading"
            >
              <Spinner v-if="submitLoading" />
              {{ $t('bots.createBot') }}
            </Button>
          </DialogFooter>
        </form>
      </template>
    </DialogContent>
  </Dialog>
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
  FormItem,
  Separator,
  Label,
  Spinner,
  Switch,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@memoh/ui'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import z from 'zod'
import { ref, computed, watch, onMounted } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { postBotsMutation, getBotsQueryKey } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'

const open = defineModel<boolean>('open', { default: false })

const step = ref(0)
const selectedTemplateId = ref('')
const selectedCategory = ref('all')

interface TemplateMeta {
  id: string
  name: string
  description: string
  icon: string
  category: string
}

const iconMap: Record<string, string> = {
  'crown': 'chess-king',
  'cpu': 'microchip',
  'cursor-click': 'hand-pointer',
  'megaphone': 'bullhorn',
  'rocket': 'rocket',
  'cube': 'cube',
  'shield-check': 'shield-halved',
  'currency-dollar': 'dollar-sign',
  'paint-brush': 'paintbrush',
  'magnifying-glass': 'magnifying-glass',
  'calendar-check': 'calendar-check',
  'book-open': 'book-open',
  'code': 'code',
}

function mapIcon(backendIcon: string): string {
  return iconMap[backendIcon] || backendIcon
}

const allTemplates = ref<TemplateMeta[]>([])
const loadingTemplates = ref(false)

async function loadTemplates() {
  loadingTemplates.value = true
  try {
    const { data } = await client.get({ url: '/templates', throwOnError: true }) as { data: { items: TemplateMeta[] } }
    const items = data?.items ?? data ?? []
    allTemplates.value = items.map((t: any) => ({
      id: t.id,
      name: t.name,
      description: t.description,
      icon: mapIcon(t.icon || 'robot'),
      category: t.category || 'productivity',
    }))
  } catch {
    allTemplates.value = []
  } finally {
    loadingTemplates.value = false
  }
}

const categories = computed(() => {
  const cats = new Set(allTemplates.value.map(t => t.category))
  return ['all', ...Array.from(cats)]
})

const filteredTemplates = computed(() => {
  if (selectedCategory.value === 'all') return allTemplates.value
  return allTemplates.value.filter(t => t.category === selectedCategory.value)
})

onMounted(() => {
  void loadTemplates()
})

const selectedTemplateName = computed(() => {
  const tmpl = allTemplates.value.find(t => t.id === selectedTemplateId.value)
  return tmpl?.name || ''
})

function selectTemplate(id: string) {
  selectedTemplateId.value = id
}

const formSchema = toTypedSchema(z.object({
  display_name: z.string().min(1),
  avatar_url: z.string().optional(),
  type: z.string(),
  is_privileged: z.boolean().optional(),
}))

const form = useForm({
  validationSchema: formSchema,
  initialValues: {
    display_name: '',
    avatar_url: '',
    type: 'personal',
    is_privileged: false,
  },
})

const queryCache = useQueryCache()
const { mutate: createBot, isLoading: submitLoading } = useMutation({
  ...postBotsMutation(),
  onSettled: () => queryCache.invalidateQueries({ key: getBotsQueryKey() }),
})

watch(open, (val) => {
  if (val) {
    step.value = 0
    selectedTemplateId.value = ''
    selectedCategory.value = 'all'
    form.resetForm({
      values: {
        display_name: '',
        avatar_url: '',
        type: 'personal',
        is_privileged: false,
      },
    })
  } else {
    form.resetForm()
  }
})

const handleSubmit = form.handleSubmit(async (values) => {
  try {
    await createBot({
      body: {
        display_name: values.display_name,
        avatar_url: values.avatar_url || undefined,
        type: values.type,
        is_active: true,
        is_privileged: values.is_privileged || false,
        template_id: selectedTemplateId.value || undefined,
      } as any,
    })
    open.value = false
  } catch {
    return
  }
})
</script>
