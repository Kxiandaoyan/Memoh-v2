<template>
  <div class="max-w-6xl mx-auto px-4 py-6">
    <div class="flex items-start justify-between gap-3 mb-6">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('sharedWorkspace.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('sharedWorkspace.subtitle') }}
        </p>
      </div>
      <Button
        variant="outline"
        size="sm"
        :disabled="loading"
        @click="loadCurrentDir"
      >
        <Spinner
          v-if="loading"
          class="mr-1.5"
        />
        {{ $t('common.refresh') }}
      </Button>
    </div>

    <!-- Breadcrumb -->
    <div class="flex items-center gap-1 text-sm mb-4 flex-wrap">
      <button
        type="button"
        class="text-primary hover:underline font-medium"
        @click="navigateTo('')"
      >
        /shared
      </button>
      <template
        v-for="(segment, idx) in pathSegments"
        :key="idx"
      >
        <span class="text-muted-foreground">/</span>
        <button
          type="button"
          class="text-primary hover:underline"
          @click="navigateTo(pathSegments.slice(0, idx + 1).join('/'))"
        >
          {{ segment }}
        </button>
      </template>
    </div>

    <!-- Loading state -->
    <div
      v-if="loading && files.length === 0"
      class="flex items-center gap-2 text-sm text-muted-foreground"
    >
      <Spinner />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="files.length === 0 && !activeFile"
      class="rounded-md border p-6 text-center text-sm text-muted-foreground"
    >
      {{ $t('sharedWorkspace.empty') }}
    </div>

    <!-- File browser + Editor -->
    <div
      v-else
      class="flex gap-4"
    >
      <!-- Left: file/directory list -->
      <div class="w-56 shrink-0 space-y-1 border rounded-md p-2 max-h-[600px] overflow-y-auto">
        <button
          v-if="currentPath"
          type="button"
          class="w-full text-left px-3 py-2 rounded-md text-sm hover:bg-muted text-muted-foreground flex items-center gap-2"
          @click="navigateUp"
        >
          <FontAwesomeIcon :icon="['fas', 'arrow-left']" class="text-xs" />
          ..
        </button>
        <button
          v-for="file in files"
          :key="file.name"
          type="button"
          class="w-full text-left px-3 py-2 rounded-md text-sm transition-colors"
          :class="[
            !file.is_dir && activeFile === joinPath(currentPath, file.name)
              ? 'bg-accent text-accent-foreground font-medium'
              : 'hover:bg-muted text-muted-foreground'
          ]"
          @click="handleItemClick(file)"
        >
          <span class="flex items-center gap-2">
            <FontAwesomeIcon
              :icon="file.is_dir ? ['fas', 'folder'] : ['fas', 'file-lines']"
              :class="file.is_dir ? 'text-yellow-500' : 'text-blue-400'"
              class="text-xs"
            />
            <span class="truncate">{{ file.name }}</span>
          </span>
          <span
            v-if="!file.is_dir"
            class="text-xs opacity-60 ml-5"
          >{{ formatBytes(file.size) }}</span>
        </button>
      </div>

      <!-- Right: editor area -->
      <div class="flex-1 min-w-0 space-y-3">
        <div
          v-if="!activeFile"
          class="rounded-md border p-8 text-center text-sm text-muted-foreground"
        >
          {{ $t('sharedWorkspace.selectHint') }}
        </div>

        <template v-else>
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <Badge variant="outline" class="text-xs font-mono shrink-0">
                {{ activeFileName }}
              </Badge>
              <span
                v-if="isDirty"
                class="text-xs text-yellow-600 dark:text-yellow-400"
              >
                {{ $t('bots.files.unsaved') }}
              </span>
            </div>
            <div class="flex gap-2 shrink-0">
              <Button
                variant="destructive"
                size="sm"
                :disabled="deleting"
                @click="handleDelete"
              >
                <Spinner
                  v-if="deleting"
                  class="mr-1.5"
                />
                {{ $t('common.delete') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                :disabled="!isDirty"
                @click="handleDiscard"
              >
                {{ $t('bots.files.discard') }}
              </Button>
              <Button
                size="sm"
                :disabled="!isDirty || saving"
                @click="handleSave"
              >
                <Spinner
                  v-if="saving"
                  class="mr-1.5"
                />
                {{ $t('common.save') }}
              </Button>
            </div>
          </div>

          <!-- Loading file content -->
          <div
            v-if="fileLoading"
            class="flex items-center gap-2 text-sm text-muted-foreground py-8"
          >
            <Spinner />
            <span>{{ $t('common.loading') }}</span>
          </div>

          <!-- Textarea editor -->
          <Textarea
            v-else
            v-model="editContent"
            class="font-mono text-sm min-h-[500px] resize-y"
            :placeholder="$t('sharedWorkspace.editorPlaceholder')"
          />
        </template>
      </div>
    </div>

    <!-- New file dialog -->
    <Dialog v-model:open="showNewFileDialog">
      <DialogTrigger as-child>
        <Button
          variant="outline"
          size="sm"
          class="mt-4"
        >
          {{ $t('sharedWorkspace.newFile') }}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ $t('sharedWorkspace.newFile') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-3 py-4">
          <div class="text-sm text-muted-foreground">
            {{ $t('sharedWorkspace.newFileHint', { path: currentPath ? `/shared/${currentPath}/` : '/shared/' }) }}
          </div>
          <Input
            v-model="newFileName"
            :placeholder="$t('sharedWorkspace.newFilePlaceholder')"
          />
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            @click="showNewFileDialog = false"
          >
            {{ $t('common.cancel') }}
          </Button>
          <Button
            :disabled="!newFileName.trim()"
            @click="createNewFile"
          >
            {{ $t('common.create') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  Input,
  Spinner,
  Textarea,
} from '@memoh/ui'
import { ref, computed, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

interface SharedFileEntry {
  name: string
  size: number
  is_dir: boolean
  updated_at: string
}

const { t } = useI18n()

const files = ref<SharedFileEntry[]>([])
const loading = ref(false)
const currentPath = ref('')
const activeFile = ref('')
const fileLoading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const editContent = ref('')
const originalContent = ref('')
const showNewFileDialog = ref(false)
const newFileName = ref('')

const pathSegments = computed(() =>
  currentPath.value ? currentPath.value.split('/').filter(Boolean) : [],
)

const activeFileName = computed(() => {
  if (!activeFile.value) return ''
  return activeFile.value.split('/').pop() || activeFile.value
})

const isDirty = computed(() => editContent.value !== originalContent.value)

function joinPath(base: string, name: string): string {
  return base ? `${base}/${name}` : name
}

function navigateTo(path: string) {
  currentPath.value = path
  activeFile.value = ''
  editContent.value = ''
  originalContent.value = ''
  loadCurrentDir()
}

function navigateUp() {
  const segments = [...pathSegments.value]
  segments.pop()
  navigateTo(segments.join('/'))
}

function handleItemClick(file: SharedFileEntry) {
  if (file.is_dir) {
    navigateTo(joinPath(currentPath.value, file.name))
  } else {
    selectFile(joinPath(currentPath.value, file.name))
  }
}

onMounted(() => {
  loadCurrentDir()
})

async function loadCurrentDir() {
  loading.value = true
  try {
    const params = currentPath.value ? `?path=${encodeURIComponent(currentPath.value)}` : ''
    const { data } = await client.get({
      url: `/shared/files${params}`,
    }) as { data: { files: SharedFileEntry[] } }
    files.value = data.files ?? []
  } catch {
    toast.error(t('sharedWorkspace.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function selectFile(filePath: string) {
  if (filePath === activeFile.value) return
  activeFile.value = filePath
  fileLoading.value = true
  try {
    const { data } = await client.get({
      url: `/shared/files/${filePath}`,
    }) as { data: { content: string } }
    editContent.value = data.content ?? ''
    originalContent.value = editContent.value
  } catch {
    toast.error(t('sharedWorkspace.readFailed'))
    editContent.value = ''
    originalContent.value = ''
  } finally {
    fileLoading.value = false
  }
}

function handleDiscard() {
  editContent.value = originalContent.value
}

async function handleSave() {
  if (!activeFile.value || !isDirty.value) return
  saving.value = true
  try {
    await client.put({
      url: `/shared/files/${activeFile.value}`,
      body: { content: editContent.value },
    })
    originalContent.value = editContent.value
    toast.success(t('sharedWorkspace.saveSuccess'))
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  if (!activeFile.value) return
  deleting.value = true
  try {
    await client.delete({
      url: `/shared/files/${activeFile.value}`,
    })
    toast.success(t('sharedWorkspace.deleteSuccess'))
    activeFile.value = ''
    editContent.value = ''
    originalContent.value = ''
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.deleteFailed'))
  } finally {
    deleting.value = false
  }
}

async function createNewFile() {
  const name = newFileName.value.trim()
  if (!name) return
  const filePath = joinPath(currentPath.value, name)
  try {
    await client.put({
      url: `/shared/files/${filePath}`,
      body: { content: '' },
    })
    toast.success(t('sharedWorkspace.createSuccess'))
    showNewFileDialog.value = false
    newFileName.value = ''
    void loadCurrentDir()
    selectFile(filePath)
  } catch {
    toast.error(t('sharedWorkspace.createFailed'))
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>
