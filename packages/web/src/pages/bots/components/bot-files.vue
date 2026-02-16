<template>
  <div class="max-w-5xl mx-auto">
    <div class="flex items-start justify-between gap-3 mb-6">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('bots.files.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('bots.files.subtitle') }}
        </p>
      </div>
      <Button
        variant="outline"
        size="sm"
        :disabled="listLoading"
        @click="loadFileList"
      >
        <Spinner
          v-if="listLoading"
          class="mr-1.5"
        />
        {{ $t('common.refresh') }}
      </Button>
    </div>

    <!-- Loading state -->
    <div
      v-if="listLoading && files.length === 0"
      class="flex items-center gap-2 text-sm text-muted-foreground"
    >
      <Spinner />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="files.length === 0"
      class="rounded-md border p-6 text-center text-sm text-muted-foreground"
    >
      {{ $t('bots.files.empty') }}
    </div>

    <!-- File list + Editor -->
    <div
      v-else
      class="flex gap-4"
    >
      <!-- Sidebar file list -->
      <div class="w-48 shrink-0 space-y-1">
        <button
          v-for="file in files"
          :key="file.name"
          type="button"
          class="w-full text-left px-3 py-2 rounded-md text-sm transition-colors"
          :class="[
            activeFile === file.name
              ? 'bg-accent text-accent-foreground font-medium'
              : 'hover:bg-muted text-muted-foreground'
          ]"
          @click="selectFile(file.name)"
        >
          <span class="truncate block">{{ file.name }}</span>
          <span class="text-xs opacity-60">{{ formatBytes(file.size) }}</span>
        </button>
      </div>

      <!-- Editor area -->
      <div class="flex-1 min-w-0 space-y-3">
        <div
          v-if="!activeFile"
          class="rounded-md border p-8 text-center text-sm text-muted-foreground"
        >
          {{ $t('bots.files.selectHint') }}
        </div>

        <template v-else>
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <Badge variant="outline" class="text-xs font-mono shrink-0">
                {{ activeFile }}
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
            :placeholder="$t('bots.files.editorPlaceholder')"
          />
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Badge,
  Button,
  Spinner,
  Textarea,
} from '@memoh/ui'
import { ref, computed, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { client } from '@memoh/sdk/client'

interface FileEntry {
  name: string
  size: number
  updated_at: string
}

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const files = ref<FileEntry[]>([])
const listLoading = ref(false)
const activeFile = ref('')
const fileLoading = ref(false)
const saving = ref(false)
const editContent = ref('')
const originalContent = ref('')

const isDirty = computed(() => editContent.value !== originalContent.value)

watch(() => props.botId, () => {
  activeFile.value = ''
  editContent.value = ''
  originalContent.value = ''
  loadFileList()
}, { immediate: true })

async function loadFileList() {
  listLoading.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/files',
      path: { bot_id: props.botId },
    }) as { data: { files: FileEntry[] } }
    files.value = data.files ?? []
    if (activeFile.value && !files.value.some((f) => f.name === activeFile.value)) {
      activeFile.value = ''
      editContent.value = ''
      originalContent.value = ''
    }
  } catch {
    toast.error(t('bots.files.loadFailed'))
  } finally {
    listLoading.value = false
  }
}

async function selectFile(name: string) {
  if (name === activeFile.value) return
  activeFile.value = name
  fileLoading.value = true
  try {
    const { data } = await client.get({
      url: '/bots/{bot_id}/files/{filename}',
      path: { bot_id: props.botId, filename: name },
    }) as { data: { content: string } }
    editContent.value = data.content ?? ''
    originalContent.value = editContent.value
  } catch {
    toast.error(t('bots.files.readFailed'))
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
      url: '/bots/{bot_id}/files/{filename}',
      path: { bot_id: props.botId, filename: activeFile.value },
      body: { content: editContent.value },
    })
    originalContent.value = editContent.value
    toast.success(t('bots.files.saveSuccess'))
    void loadFileList()
  } catch {
    toast.error(t('bots.files.saveFailed'))
  } finally {
    saving.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>
