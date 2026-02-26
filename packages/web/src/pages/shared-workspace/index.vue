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
      <div class="flex gap-2 shrink-0">
        <Button
          v-if="multiSelect && selectedFiles.size > 0"
          variant="destructive"
          size="sm"
          @click="handleBatchDelete"
        >
          {{ $t('sharedWorkspace.batchDelete', { count: selectedFiles.size }) }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :class="multiSelect ? 'ring-2 ring-primary' : ''"
          @click="toggleMultiSelect"
        >
          {{ multiSelect ? $t('sharedWorkspace.exitSelect') : $t('sharedWorkspace.multiSelect') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadCurrentDir"
        >
          <Spinner v-if="loading" class="mr-1.5" />
          {{ $t('common.refresh') }}
        </Button>
      </div>
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
        <div
          v-for="file in files"
          :key="file.name"
          class="flex items-center gap-1"
        >
          <Checkbox
            v-if="multiSelect"
            :model-value="selectedFiles.has(file.name)"
            class="shrink-0"
            @update:model-value="toggleSelect(file.name)"
          />
          <button
            type="button"
            class="flex-1 text-left px-3 py-2 rounded-md text-sm transition-colors"
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
          <DropdownMenu v-if="!multiSelect">
            <DropdownMenuTrigger as-child>
              <button type="button" class="p-1 rounded hover:bg-muted text-muted-foreground shrink-0">
                <FontAwesomeIcon :icon="['fas', 'ellipsis-vertical']" class="text-xs" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem @click="startRename(file)">
                {{ $t('sharedWorkspace.rename') }}
              </DropdownMenuItem>
              <DropdownMenuItem v-if="!file.is_dir" @click="handleCopy(file)">
                {{ $t('sharedWorkspace.copy') }}
              </DropdownMenuItem>
              <DropdownMenuItem @click="startMove(file)">
                {{ $t('sharedWorkspace.move') }}
              </DropdownMenuItem>
              <DropdownMenuItem class="text-destructive" @click="handleDeleteItem(file)">
                {{ $t('common.delete') }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
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
                v-if="previewType === 'html' || previewType === 'pdf'"
                variant="outline"
                size="sm"
                as="a"
                :href="buildPreviewUrl(activeFile)"
                target="_blank"
              >
                {{ $t('sharedWorkspace.openInNewTab') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                @click="handleDownload"
              >
                {{ $t('sharedWorkspace.download') }}
              </Button>
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
              <template v-if="!isPreviewable">
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
              </template>
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

          <!-- Preview: image -->
          <div
            v-else-if="previewType === 'image' && previewBlobUrl"
            class="rounded-md border p-4 flex items-center justify-center bg-muted/30 min-h-[400px]"
          >
            <img
              :src="previewBlobUrl"
              :alt="activeFileName"
              class="max-w-full max-h-[500px] object-contain rounded"
            >
          </div>

          <!-- Preview: PDF / HTML -->
          <iframe
            v-else-if="(previewType === 'pdf' || previewType === 'html') && previewBlobUrl"
            :src="previewBlobUrl"
            class="w-full min-h-[600px] rounded-md border"
            :sandbox="previewType === 'html' ? 'allow-same-origin' : undefined"
          />

          <!-- Preview: video -->
          <video
            v-else-if="previewType === 'video' && previewBlobUrl"
            :src="previewBlobUrl"
            controls
            class="w-full rounded-md border max-h-[500px]"
          />

          <!-- Preview: audio -->
          <audio
            v-else-if="previewType === 'audio' && previewBlobUrl"
            :src="previewBlobUrl"
            controls
            class="w-full mt-2"
          />

          <!-- No preview available (binary files) -->
          <div
            v-else-if="previewType === 'none'"
            class="rounded-md border p-8 text-center text-sm text-muted-foreground"
          >
            {{ $t('sharedWorkspace.noPreview') }}
          </div>

          <!-- Text editor -->
          <Textarea
            v-else-if="previewType === 'text'"
            v-model="editContent"
            class="font-mono text-sm min-h-[500px] resize-y"
            :placeholder="$t('sharedWorkspace.editorPlaceholder')"
          />
        </template>
      </div>
    </div>

    <!-- Action buttons -->
    <div class="flex gap-2 mt-4">
      <Button variant="outline" size="sm" @click="showNewFileDialog = true">
        {{ $t('sharedWorkspace.newFile') }}
      </Button>
      <Button variant="outline" size="sm" @click="showNewFolderDialog = true">
        {{ $t('sharedWorkspace.newFolder') }}
      </Button>
    </div>

    <!-- New file dialog -->
    <Dialog v-model:open="showNewFileDialog">
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

    <!-- New folder dialog -->
    <Dialog v-model:open="showNewFolderDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ $t('sharedWorkspace.newFolder') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-3 py-4">
          <Input
            v-model="newFolderName"
            :placeholder="$t('sharedWorkspace.newFolderPlaceholder')"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showNewFolderDialog = false">
            {{ $t('common.cancel') }}
          </Button>
          <Button :disabled="!newFolderName.trim()" @click="createNewFolder">
            {{ $t('common.create') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Rename dialog -->
    <Dialog v-model:open="showRenameDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ $t('sharedWorkspace.rename') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-3 py-4">
          <Input v-model="renameNewName" />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showRenameDialog = false">
            {{ $t('common.cancel') }}
          </Button>
          <Button :disabled="!renameNewName.trim()" @click="confirmRename">
            {{ $t('common.confirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Move dialog -->
    <Dialog v-model:open="showMoveDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ $t('sharedWorkspace.move') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-3 py-4">
          <div class="text-sm text-muted-foreground">
            {{ $t('sharedWorkspace.moveHint', { name: moveTarget?.name }) }}
          </div>
          <Input
            v-model="moveDestination"
            :placeholder="$t('sharedWorkspace.moveDestPlaceholder')"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showMoveDialog = false">
            {{ $t('common.cancel') }}
          </Button>
          <Button :disabled="!moveDestination.trim()" @click="confirmMove">
            {{ $t('common.confirm') }}
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
  Checkbox,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
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

// Part 4: file management state
const showNewFolderDialog = ref(false)
const newFolderName = ref('')
const showRenameDialog = ref(false)
const renameTarget = ref<SharedFileEntry | null>(null)
const renameNewName = ref('')
const showMoveDialog = ref(false)
const moveTarget = ref<SharedFileEntry | null>(null)
const moveDestination = ref('')
const multiSelect = ref(false)
const selectedFiles = ref<Set<string>>(new Set())

const pathSegments = computed(() =>
  currentPath.value ? currentPath.value.split('/').filter(Boolean) : [],
)

const activeFileName = computed(() => {
  if (!activeFile.value) return ''
  return activeFile.value.split('/').pop() || activeFile.value
})

const isDirty = computed(() => editContent.value !== originalContent.value)

const fileExt = computed(() => {
  const name = activeFileName.value.toLowerCase()
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1) : ''
})

type PreviewType = 'image' | 'pdf' | 'video' | 'audio' | 'html' | 'text' | 'none'

const previewType = computed<PreviewType>(() => {
  const ext = fileExt.value
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)) return 'image'
  if (ext === 'pdf') return 'pdf'
  if (['mp4', 'webm', 'ogg', 'mov'].includes(ext)) return 'video'
  if (['mp3', 'wav', 'flac', 'aac', 'ogg'].includes(ext)) return 'audio'
  if (['html', 'htm'].includes(ext)) return 'html'
  if (['md', 'txt', 'json', 'csv', 'xml', 'yaml', 'yml', 'toml', 'log', 'js', 'ts', 'css', 'py', 'go', 'sh', 'sql', 'env', 'ini', 'conf', 'cfg'].includes(ext)) return 'text'
  return 'none'
})

const isPreviewable = computed(() => previewType.value !== 'text' && previewType.value !== 'none')

const previewBlobUrl = ref('')

function buildFileUrl(filePath: string): string {
  const baseUrl = client.getConfig().baseUrl || '/api'
  return `${baseUrl}/shared/files/download/${encodeURIComponent(filePath)}`
}

async function fetchBlob(filePath: string): Promise<Blob> {
  const token = localStorage.getItem('token') || ''
  const resp = await fetch(buildFileUrl(filePath), {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
  return resp.blob()
}

async function handleDownload() {
  if (!activeFile.value) return
  try {
    const blob = await fetchBlob(activeFile.value)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = activeFileName.value
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch {
    toast.error(t('sharedWorkspace.downloadFailed'))
  }
}

function joinPath(base: string, name: string): string {
  return base ? `${base}/${name}` : name
}

function revokePreview() {
  if (previewBlobUrl.value) {
    URL.revokeObjectURL(previewBlobUrl.value)
    previewBlobUrl.value = ''
  }
}

function navigateTo(path: string) {
  revokePreview()
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
  revokePreview()
  activeFile.value = filePath
  editContent.value = ''
  originalContent.value = ''
  fileLoading.value = true
  try {
    // Determine preview type from extension before loading
    const ext = filePath.split('.').pop()?.toLowerCase() || ''
    const binaryExts = ['png','jpg','jpeg','gif','webp','svg','bmp','ico','pdf','mp4','webm','ogg','mov','mp3','wav','flac','aac','html','htm']
    if (binaryExts.includes(ext)) {
      const blob = await fetchBlob(filePath)
      previewBlobUrl.value = URL.createObjectURL(blob)
    } else {
      const { data } = await client.get({
        url: `/shared/files/${filePath}`,
      }) as { data: { content: string } }
      editContent.value = data.content ?? ''
      originalContent.value = editContent.value
    }
  } catch {
    toast.error(t('sharedWorkspace.readFailed'))
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
    revokePreview()
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

async function handleDeleteItem(file: SharedFileEntry) {
  const filePath = joinPath(currentPath.value, file.name)
  try {
    await client.delete({ url: `/shared/files/${filePath}` })
    toast.success(t('sharedWorkspace.deleteSuccess'))
    if (activeFile.value === filePath) {
      revokePreview()
      activeFile.value = ''
      editContent.value = ''
      originalContent.value = ''
    }
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.deleteFailed'))
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

// ── Part 4: File management operations ──

async function createNewFolder() {
  const name = newFolderName.value.trim()
  if (!name) return
  const folderPath = joinPath(currentPath.value, name)
  try {
    await client.post({ url: '/shared/files/mkdir', body: { path: folderPath } })
    toast.success(t('sharedWorkspace.mkdirSuccess'))
    showNewFolderDialog.value = false
    newFolderName.value = ''
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.mkdirFailed'))
  }
}

function startRename(file: SharedFileEntry) {
  renameTarget.value = file
  renameNewName.value = file.name
  showRenameDialog.value = true
}

async function confirmRename() {
  if (!renameTarget.value || !renameNewName.value.trim()) return
  const from = joinPath(currentPath.value, renameTarget.value.name)
  const to = joinPath(currentPath.value, renameNewName.value.trim())
  if (from === to) { showRenameDialog.value = false; return }
  try {
    await client.post({ url: '/shared/files/rename', body: { from, to } })
    toast.success(t('sharedWorkspace.renameSuccess'))
    showRenameDialog.value = false
    renameTarget.value = null
    if (activeFile.value === from) activeFile.value = to
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.renameFailed'))
  }
}

function startMove(file: SharedFileEntry) {
  moveTarget.value = file
  moveDestination.value = ''
  showMoveDialog.value = true
}

async function confirmMove() {
  if (!moveTarget.value || !moveDestination.value.trim()) return
  const from = joinPath(currentPath.value, moveTarget.value.name)
  const to = joinPath(moveDestination.value.trim(), moveTarget.value.name)
  try {
    await client.post({ url: '/shared/files/move', body: { from, to } })
    toast.success(t('sharedWorkspace.moveSuccess'))
    showMoveDialog.value = false
    moveTarget.value = null
    if (activeFile.value === from) { activeFile.value = ''; revokePreview() }
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.moveFailed'))
  }
}

async function handleCopy(file: SharedFileEntry) {
  const from = joinPath(currentPath.value, file.name)
  const ext = file.name.lastIndexOf('.')
  const copyName = ext > 0
    ? `${file.name.slice(0, ext)}_copy${file.name.slice(ext)}`
    : `${file.name}_copy`
  const to = joinPath(currentPath.value, copyName)
  try {
    await client.post({ url: '/shared/files/copy', body: { from, to } })
    toast.success(t('sharedWorkspace.copySuccess'))
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.copyFailed'))
  }
}

function toggleSelect(fileName: string) {
  const s = new Set(selectedFiles.value)
  if (s.has(fileName)) s.delete(fileName)
  else s.add(fileName)
  selectedFiles.value = s
}

function toggleMultiSelect() {
  multiSelect.value = !multiSelect.value
  if (!multiSelect.value) selectedFiles.value = new Set()
}

async function handleBatchDelete() {
  if (selectedFiles.value.size === 0) return
  const paths = [...selectedFiles.value].map(name => joinPath(currentPath.value, name))
  try {
    await client.post({ url: '/shared/files/batch-delete', body: { paths } })
    toast.success(t('sharedWorkspace.batchDeleteSuccess', { count: paths.length }))
    selectedFiles.value = new Set()
    multiSelect.value = false
    revokePreview()
    activeFile.value = ''
    void loadCurrentDir()
  } catch {
    toast.error(t('sharedWorkspace.batchDeleteFailed'))
  }
}

function buildPreviewUrl(filePath: string): string {
  const baseUrl = client.getConfig().baseUrl || '/api'
  const token = localStorage.getItem('token') || ''
  return `${baseUrl}/shared/files/preview/${encodeURIComponent(filePath)}?token=${encodeURIComponent(token)}`
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>
