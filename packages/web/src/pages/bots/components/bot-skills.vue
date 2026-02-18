<template>
  <div class="max-w-4xl mx-auto space-y-5">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('bots.skills.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('bots.skills.subtitle') }}
        </p>
      </div>
    </div>

    <!-- Tabs -->
    <div class="flex border-b">
      <button
        class="px-4 py-2 text-sm font-medium border-b-2 transition-colors"
        :class="activeTab === 'installed' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'installed'"
      >
        {{ $t('bots.skills.installedTab') }}
      </button>
      <button
        class="px-4 py-2 text-sm font-medium border-b-2 transition-colors"
        :class="activeTab === 'clawhub' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'clawhub'"
      >
        {{ $t('bots.skills.clawHubTab') }}
      </button>
    </div>

    <!-- Installed Skills Tab -->
    <div v-if="activeTab === 'installed'">
      <div class="flex justify-end gap-2 mb-4">
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadSkills"
        >
          <Spinner v-if="loading" class="mr-1.5" />
          {{ $t('common.refresh') }}
        </Button>
        <Button size="sm" @click="openCreateDialog">
          {{ $t('common.add') }}
        </Button>
      </div>

      <div
        v-if="loading && skills.length === 0"
        class="flex items-center gap-2 text-sm text-muted-foreground"
      >
        <Spinner />
        <span>{{ $t('common.loading') }}</span>
      </div>

      <div
        v-else-if="skills.length === 0"
        class="rounded-md border p-6 text-center text-sm text-muted-foreground"
      >
        {{ $t('bots.skills.empty') }}
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="skill in skills"
          :key="skill.name"
          class="rounded-md border p-4 space-y-2"
        >
          <div class="flex items-center justify-between gap-2">
            <div class="min-w-0">
              <p class="font-mono text-sm font-medium truncate">{{ skill.name }}</p>
              <p v-if="skill.description" class="text-sm text-muted-foreground truncate">
                {{ skill.description }}
              </p>
            </div>
            <div class="flex gap-2 shrink-0">
              <Button variant="outline" size="sm" @click="viewSkill(skill)">
                {{ $t('bots.skills.view') }}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                :disabled="deleting === skill.name"
                @click="handleDelete(skill.name)"
              >
                <Spinner v-if="deleting === skill.name" class="mr-1.5" />
                {{ $t('common.delete') }}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ClawHub Marketplace Tab -->
    <div v-if="activeTab === 'clawhub'">
      <div class="flex gap-2 mb-4">
        <Input
          v-model="clawHubQuery"
          :placeholder="$t('bots.skills.clawHubSearchPlaceholder')"
          class="flex-1"
          @keydown.enter="searchClawHub"
        />
        <Button
          size="sm"
          :disabled="clawHubSearching"
          @click="searchClawHub"
        >
          <Spinner v-if="clawHubSearching" class="mr-1.5" />
          {{ $t('bots.skills.clawHubSearch') }}
        </Button>
      </div>

      <div
        v-if="clawHubSearching"
        class="flex items-center gap-2 text-sm text-muted-foreground"
      >
        <Spinner />
        <span>{{ $t('common.loading') }}</span>
      </div>

      <div
        v-else-if="!clawHubSearched"
        class="rounded-md border p-6 text-center text-sm text-muted-foreground"
      >
        {{ $t('bots.skills.clawHubEmpty') }}
      </div>

      <div
        v-else-if="clawHubResults.length === 0"
        class="rounded-md border p-6 text-center text-sm text-muted-foreground"
      >
        {{ $t('bots.skills.clawHubNoResults') }}
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="item in clawHubResults"
          :key="item.slug"
          class="rounded-md border p-4"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <p class="font-mono text-sm font-medium truncate">{{ item.name }}</p>
                <span v-if="item.version" class="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                  v{{ item.version }}
                </span>
              </div>
              <p v-if="item.description" class="text-sm text-muted-foreground mt-0.5 truncate">
                {{ item.description }}
              </p>
              <p v-if="item.author" class="text-xs text-muted-foreground/70 mt-1">
                by {{ item.author }}
              </p>
            </div>
            <Button
              size="sm"
              :disabled="clawHubInstalling === item.slug"
              @click="installFromClawHub(item.slug)"
            >
              <Spinner v-if="clawHubInstalling === item.slug" class="mr-1.5" />
              {{ clawHubInstalling === item.slug ? $t('bots.skills.clawHubInstalling') : $t('bots.skills.clawHubInstall') }}
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- View/Edit Dialog -->
    <Dialog v-model:open="viewDialogOpen">
      <DialogContent class="sm:max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle class="font-mono">{{ viewingSkill?.name }}</DialogTitle>
          <DialogDescription>{{ viewingSkill?.description || $t('bots.skills.noDescription') }}</DialogDescription>
        </DialogHeader>
        <div class="flex-1 overflow-auto mt-4">
          <pre class="text-sm whitespace-pre-wrap font-mono bg-muted/50 rounded-md p-4">{{ viewingSkill?.content }}</pre>
        </div>
        <DialogFooter class="mt-4">
          <DialogClose as-child>
            <Button variant="outline">{{ $t('common.cancel') }}</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Create Dialog -->
    <Dialog v-model:open="createDialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t('bots.skills.createTitle') }}</DialogTitle>
          <DialogDescription>{{ $t('bots.skills.createDescription') }}</DialogDescription>
        </DialogHeader>
        <div class="mt-4 space-y-4">
          <div class="space-y-2">
            <Label>{{ $t('bots.skills.nameLabel') }}</Label>
            <p class="text-xs text-muted-foreground">{{ $t('bots.skills.nameHint') }}</p>
            <Input
              v-model="createForm.name"
              :placeholder="$t('bots.skills.namePlaceholder')"
              :disabled="creating"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.skills.descriptionLabel') }}</Label>
            <p class="text-xs text-muted-foreground">{{ $t('bots.skills.descriptionHint') }}</p>
            <Input
              v-model="createForm.description"
              :placeholder="$t('bots.skills.descriptionPlaceholder')"
              :disabled="creating"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.skills.contentLabel') }}</Label>
            <p class="text-xs text-muted-foreground">{{ $t('bots.skills.contentHint') }}</p>
            <Textarea
              v-model="createForm.content"
              :placeholder="$t('bots.skills.contentPlaceholder')"
              rows="8"
              class="font-mono text-sm"
              :disabled="creating"
            />
          </div>
        </div>
        <DialogFooter class="mt-6">
          <DialogClose as-child>
            <Button variant="outline" :disabled="creating">
              {{ $t('common.cancel') }}
            </Button>
          </DialogClose>
          <Button
            :disabled="creating || !createForm.name.trim()"
            @click="handleCreate"
          >
            <Spinner v-if="creating" class="mr-1.5" />
            {{ $t('common.add') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import {
  Button,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Spinner,
  Textarea,
} from '@memoh/ui'
import { ref, reactive, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

interface SkillItem {
  name: string
  description: string
  content: string
  metadata?: Record<string, unknown>
}

interface ClawHubResult {
  name: string
  slug: string
  description: string
  author: string
  version?: string
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const activeTab = ref<'installed' | 'clawhub'>('installed')
const skills = ref<SkillItem[]>([])
const loading = ref(false)
const deleting = ref('')
const creating = ref(false)
const viewDialogOpen = ref(false)
const createDialogOpen = ref(false)
const viewingSkill = ref<SkillItem | null>(null)

const createForm = reactive({
  name: '',
  description: '',
  content: '',
})

// ClawHub state
const clawHubQuery = ref('')
const clawHubSearching = ref(false)
const clawHubSearched = ref(false)
const clawHubResults = ref<ClawHubResult[]>([])
const clawHubInstalling = ref('')

watch(() => props.botId, () => {
  loadSkills()
}, { immediate: true })

async function loadSkills() {
  loading.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/container/skills',
      path: { bot_id: props.botId },
    }) as { data: { skills: SkillItem[] } }
    skills.value = data.skills ?? []
  } catch {
    toast.error(t('bots.skills.loadFailed'))
  } finally {
    loading.value = false
  }
}

function viewSkill(skill: SkillItem) {
  viewingSkill.value = skill
  viewDialogOpen.value = true
}

function openCreateDialog() {
  createForm.name = ''
  createForm.description = ''
  createForm.content = ''
  createDialogOpen.value = true
}

async function handleCreate() {
  if (!createForm.name.trim()) return
  creating.value = true
  try {
    await client.post({
      url: '/bots/{bot_id}/container/skills',
      path: { bot_id: props.botId },
      body: {
        skills: [{
          name: createForm.name.trim(),
          description: createForm.description.trim(),
          content: createForm.content.trim(),
        }],
      },
    })
    createDialogOpen.value = false
    toast.success(t('bots.skills.createSuccess'))
    await loadSkills()
  } catch {
    toast.error(t('bots.skills.createFailed'))
  } finally {
    creating.value = false
  }
}

async function handleDelete(name: string) {
  deleting.value = name
  try {
    await client.delete({
      url: '/bots/{bot_id}/container/skills',
      path: { bot_id: props.botId },
      body: { names: [name] },
    })
    toast.success(t('bots.skills.deleteSuccess'))
    await loadSkills()
  } catch {
    toast.error(t('bots.skills.deleteFailed'))
  } finally {
    deleting.value = ''
  }
}

async function searchClawHub() {
  const q = clawHubQuery.value.trim()
  if (!q) return
  clawHubSearching.value = true
  clawHubSearched.value = false
  try {
    const { data } = await client.post({
      url: '/bots/{bot_id}/container/clawhub/search',
      path: { bot_id: props.botId },
      body: { query: q },
    }) as { data: { results: ClawHubResult[] } }
    clawHubResults.value = data.results ?? []
    clawHubSearched.value = true
  } catch {
    toast.error(t('bots.skills.clawHubSearchFailed'))
  } finally {
    clawHubSearching.value = false
  }
}

async function installFromClawHub(slug: string) {
  clawHubInstalling.value = slug
  try {
    await client.post({
      url: '/bots/{bot_id}/container/clawhub/install',
      path: { bot_id: props.botId },
      body: { slug },
    })
    toast.success(t('bots.skills.clawHubInstallSuccess'))
    await loadSkills()
  } catch {
    toast.error(t('bots.skills.clawHubInstallFailed'))
  } finally {
    clawHubInstalling.value = ''
  }
}
</script>
