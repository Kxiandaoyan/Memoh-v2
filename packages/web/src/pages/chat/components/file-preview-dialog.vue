<template>
  <Dialog v-model:open="open">
    <DialogScrollContent class="max-w-4xl max-h-[80vh]">
      <DialogHeader class="flex-row items-center justify-between gap-2 pr-8">
        <DialogTitle class="truncate text-sm font-medium">{{ fileName }}</DialogTitle>
        <button
          class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs hover:bg-muted transition-colors"
          @click="$emit('download')"
        >
          <FontAwesomeIcon :icon="['fas', 'download']" class="size-3" />
          下载
        </button>
      </DialogHeader>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <FontAwesomeIcon :icon="['fas', 'spinner']" class="size-5 animate-spin text-muted-foreground" />
      </div>

      <div v-else-if="error" class="text-sm text-destructive py-8 text-center">{{ error }}</div>

      <!-- Markdown -->
      <div v-else-if="previewType === 'markdown'" class="prose dark:prose-invert max-w-none overflow-auto">
        <MarkdownRender :content="textContent" :final="true" custom-id="file-preview" />
      </div>

      <!-- HTML -->
      <iframe
        v-else-if="previewType === 'html'"
        sandbox="allow-same-origin"
        :srcdoc="textContent"
        class="w-full h-[70vh] border rounded"
      />

      <!-- PDF -->
      <iframe
        v-else-if="previewType === 'pdf'"
        :src="authedUrl"
        class="w-full h-[70vh] border rounded"
      />

      <!-- Image -->
      <img
        v-else-if="previewType === 'image'"
        :src="authedUrl"
        :alt="fileName"
        class="max-w-full h-auto rounded mx-auto"
      >

      <!-- Code / Text -->
      <div v-else-if="previewType === 'code'" class="overflow-auto">
        <MarkdownRender :content="codeBlock" :final="true" custom-id="file-preview" />
      </div>
    </DialogScrollContent>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Dialog, DialogScrollContent, DialogHeader, DialogTitle } from '@memoh/ui'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import MarkdownRender from 'markstream-vue'

const props = defineProps<{
  modelValue: boolean
  fileName: string
  fileUrl: string
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  download: []
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const textContent = ref('')
const loading = ref(false)
const error = ref('')

const ext = computed(() => {
  const dot = props.fileName.lastIndexOf('.')
  return dot > 0 ? props.fileName.slice(dot).toLowerCase() : ''
})

const previewType = computed(() => {
  const e = ext.value
  if (e === '.md') return 'markdown'
  if (e === '.html' || e === '.htm') return 'html'
  if (e === '.pdf') return 'pdf'
  if (/^\.(png|jpe?g|gif|webp|svg|bmp|ico)$/.test(e)) return 'image'
  if (/^\.(js|ts|jsx|tsx|py|go|rs|java|c|cpp|h|rb|php|sh|sql|css|json|yaml|yml|xml|toml|txt|csv|log|env|ini|conf|vue|svelte)$/.test(e)) return 'code'
  return null
})

const codeLang = computed(() => {
  const map: Record<string, string> = {
    '.js': 'javascript', '.ts': 'typescript', '.jsx': 'jsx', '.tsx': 'tsx',
    '.py': 'python', '.go': 'go', '.rs': 'rust', '.java': 'java',
    '.c': 'c', '.cpp': 'cpp', '.h': 'c', '.rb': 'ruby', '.php': 'php',
    '.sh': 'bash', '.sql': 'sql', '.css': 'css', '.json': 'json',
    '.yaml': 'yaml', '.yml': 'yaml', '.xml': 'xml', '.toml': 'toml',
    '.html': 'html', '.vue': 'vue', '.svelte': 'svelte',
  }
  return map[ext.value] || ''
})

const codeBlock = computed(() => '```' + codeLang.value + '\n' + textContent.value + '\n```')

const authedUrl = computed(() => {
  const token = localStorage.getItem('token')
  if (!token || !props.fileUrl) return props.fileUrl
  const sep = props.fileUrl.includes('?') ? '&' : '?'
  return `${props.fileUrl}${sep}token=${encodeURIComponent(token)}`
})

const needsFetch = computed(() => previewType.value === 'markdown' || previewType.value === 'html' || previewType.value === 'code')

watch(() => props.modelValue, async (isOpen) => {
  if (!isOpen || !needsFetch.value) return
  loading.value = true
  error.value = ''
  textContent.value = ''
  const token = localStorage.getItem('token')
  try {
    const resp = await fetch(props.fileUrl, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    textContent.value = await resp.text()
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
})
</script>
