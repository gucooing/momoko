<script setup lang="ts">
import { Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { usePreferredDark } from '@vueuse/core'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api.js'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'

defineOptions({ name: 'FileEditorDialogContent' })

type EditorThemeMode = 'light' | 'dark' | 'auto'

interface PointerSnapshot {
  startX: number
  startY: number
  startLeft: number
  startTop: number
  startWidth: number
  startHeight: number
}

const EDITOR_THEME_STORAGE_KEY = 'file-editor-theme-mode'
const DEFAULT_DIALOG_MARGIN = 12

const props = defineProps<{
  fileName: string
  filePath: string
  content: string
  onClose: () => void
  onSave: (content: string) => Promise<void>
}>()

const editorHostRef = useTemplateRef<HTMLDivElement>('editorHost')
const editorRef = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
const modelRef = shallowRef<monaco.editor.ITextModel | null>(null)
const dialogElementRef = shallowRef<HTMLElement | null>(null)
const preferredDark = usePreferredDark()
const draftContent = ref(props.content)
const originalContent = ref(props.content)
const saveLoading = ref(false)
const themeMode = ref<EditorThemeMode>('auto')

let dragSnapshot: PointerSnapshot | null = null
let resizeSnapshot: PointerSnapshot | null = null
let removeDragListeners: (() => void) | null = null
let removeResizeListeners: (() => void) | null = null

const themeModeOptions: Array<{ label: string; value: EditorThemeMode }> = [
  { label: '自动', value: 'auto' },
  { label: '亮', value: 'light' },
  { label: '暗', value: 'dark' },
]

const languageByExtension: Record<string, string> = {
  '.bat': 'bat',
  '.c': 'c',
  '.cc': 'cpp',
  '.cpp': 'cpp',
  '.css': 'css',
  '.go': 'go',
  '.h': 'cpp',
  '.hpp': 'cpp',
  '.html': 'html',
  '.ini': 'ini',
  '.java': 'java',
  '.js': 'javascript',
  '.json': 'json',
  '.less': 'less',
  '.lua': 'lua',
  '.md': 'markdown',
  '.php': 'php',
  '.ps1': 'powershell',
  '.py': 'python',
  '.rb': 'ruby',
  '.rs': 'rust',
  '.sass': 'scss',
  '.scss': 'scss',
  '.sh': 'shell',
  '.sql': 'sql',
  '.ts': 'typescript',
  '.tsx': 'typescript',
  '.txt': 'plaintext',
  '.vue': 'html',
  '.xml': 'xml',
  '.yml': 'yaml',
}

const isDirty = computed(() => draftContent.value !== originalContent.value)
const detectedLanguage = computed(() => {
  const normalizedPath = props.filePath.toLowerCase()
  const extension = normalizedPath.match(/\.[^./\\]+$/)?.[0] || ''
  return languageByExtension[extension] || 'plaintext'
})
const effectiveThemeMode = computed<'light' | 'dark'>(() => {
  if (themeMode.value === 'auto') {
    return preferredDark.value ? 'dark' : 'light'
  }

  return themeMode.value
})
const monacoTheme = computed(() => (effectiveThemeMode.value === 'dark' ? 'vs-dark' : 'vs'))

const ensureMonacoEnvironment = () => {
  const target = globalThis as typeof globalThis & {
    MonacoEnvironment?: {
      getWorker: (_moduleId: string, label: string) => Worker
    }
  }

  if (target.MonacoEnvironment) {
    return
  }

  target.MonacoEnvironment = {
    getWorker: (_moduleId: string, label: string) => {
      if (label === 'json') return new jsonWorker()
      if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
      if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
      if (label === 'typescript' || label === 'javascript') return new tsWorker()
      return new editorWorker()
    },
  }
}

const loadStoredThemeMode = () => {
  const storedThemeMode = localStorage.getItem(EDITOR_THEME_STORAGE_KEY)

  if (storedThemeMode === 'light' || storedThemeMode === 'dark' || storedThemeMode === 'auto') {
    themeMode.value = storedThemeMode
  }
}

const createEditorModel = () => {
  modelRef.value?.dispose()
  modelRef.value = monaco.editor.createModel(draftContent.value, detectedLanguage.value)
}

const layoutEditor = () => {
  editorRef.value?.layout()
}

const resolveDialogElement = (fallbackElement?: HTMLElement | null) => {
  const nextDialogElement =
    dialogElementRef.value ||
    (fallbackElement?.closest('.el-dialog') as HTMLElement | null) ||
    (editorHostRef.value?.closest('.el-dialog') as HTMLElement | null)

  if (nextDialogElement) {
    dialogElementRef.value = nextDialogElement
  }

  return nextDialogElement
}

const getMinimumWidth = () => Math.min(680, window.innerWidth - DEFAULT_DIALOG_MARGIN * 2)
const getMinimumHeight = () => Math.min(420, window.innerHeight - DEFAULT_DIALOG_MARGIN * 2)

const applyDialogFrame = (frame: { left: number; top: number; width: number; height: number }) => {
  const dialogElement = resolveDialogElement()
  if (!dialogElement) {
    return
  }

  const minWidth = getMinimumWidth()
  const minHeight = getMinimumHeight()
  const maxWidth = Math.max(minWidth, window.innerWidth - DEFAULT_DIALOG_MARGIN * 2)
  const maxHeight = Math.max(minHeight, window.innerHeight - DEFAULT_DIALOG_MARGIN * 2)
  const width = Math.min(Math.max(frame.width, minWidth), maxWidth)
  const height = Math.min(Math.max(frame.height, minHeight), maxHeight)
  const left = Math.min(
    Math.max(frame.left, DEFAULT_DIALOG_MARGIN),
    Math.max(DEFAULT_DIALOG_MARGIN, window.innerWidth - width - DEFAULT_DIALOG_MARGIN),
  )
  const top = Math.min(
    Math.max(frame.top, DEFAULT_DIALOG_MARGIN),
    Math.max(DEFAULT_DIALOG_MARGIN, window.innerHeight - height - DEFAULT_DIALOG_MARGIN),
  )

  dialogElement.style.position = 'fixed'
  dialogElement.style.margin = '0'
  dialogElement.style.left = `${left}px`
  dialogElement.style.top = `${top}px`
  dialogElement.style.width = `${width}px`
  dialogElement.style.height = `${height}px`
  dialogElement.style.maxWidth = 'none'
  dialogElement.style.maxHeight = 'none'

  layoutEditor()
}

const initializeDialogFrame = () => {
  const dialogElement = resolveDialogElement()
  if (!dialogElement) {
    return
  }

  const rect = dialogElement.getBoundingClientRect()
  applyDialogFrame({
    left: rect.left,
    top: rect.top,
    width: rect.width,
    height: rect.height,
  })
}

const stopDragging = () => {
  removeDragListeners?.()
  removeDragListeners = null
  dragSnapshot = null
}

const stopResizing = () => {
  removeResizeListeners?.()
  removeResizeListeners = null
  resizeSnapshot = null
}

const handleDragMove = (event: MouseEvent) => {
  if (!dragSnapshot) {
    return
  }

  applyDialogFrame({
    left: dragSnapshot.startLeft + event.clientX - dragSnapshot.startX,
    top: dragSnapshot.startTop + event.clientY - dragSnapshot.startY,
    width: dragSnapshot.startWidth,
    height: dragSnapshot.startHeight,
  })
}

const handleResizeMove = (event: MouseEvent) => {
  if (!resizeSnapshot) {
    return
  }

  applyDialogFrame({
    left: resizeSnapshot.startLeft,
    top: resizeSnapshot.startTop,
    width: resizeSnapshot.startWidth + event.clientX - resizeSnapshot.startX,
    height: resizeSnapshot.startHeight + event.clientY - resizeSnapshot.startY,
  })
}

const startListening = (moveHandler: (event: MouseEvent) => void, stopHandler: () => void) => {
  const onMouseMove = (event: MouseEvent) => moveHandler(event)
  const onMouseUp = () => stopHandler()

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp, { once: true })

  return () => {
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
  }
}

const handleToolbarMouseDown = (event: MouseEvent) => {
  const target = event.target instanceof HTMLElement ? event.target : null
  if (event.button !== 0 || !target || target.closest('.file-editor-actions')) {
    return
  }

  const dialogElement = resolveDialogElement(event.currentTarget as HTMLElement | null)
  if (!dialogElement) {
    return
  }

  const rect = dialogElement.getBoundingClientRect()
  stopDragging()
  dragSnapshot = {
    startX: event.clientX,
    startY: event.clientY,
    startLeft: rect.left,
    startTop: rect.top,
    startWidth: rect.width,
    startHeight: rect.height,
  }

  removeDragListeners = startListening(handleDragMove, stopDragging)
  event.preventDefault()
}

const handleResizeHandleMouseDown = (event: MouseEvent) => {
  if (event.button !== 0) {
    return
  }

  const dialogElement = resolveDialogElement(event.currentTarget as HTMLElement | null)
  if (!dialogElement) {
    return
  }

  const rect = dialogElement.getBoundingClientRect()
  stopResizing()
  resizeSnapshot = {
    startX: event.clientX,
    startY: event.clientY,
    startLeft: rect.left,
    startTop: rect.top,
    startWidth: rect.width,
    startHeight: rect.height,
  }

  removeResizeListeners = startListening(handleResizeMove, stopResizing)
  event.preventDefault()
  event.stopPropagation()
}

const initializeEditor = () => {
  if (!editorHostRef.value) {
    return
  }

  ensureMonacoEnvironment()
  createEditorModel()
  monaco.editor.setTheme(monacoTheme.value)

  editorRef.value = monaco.editor.create(editorHostRef.value, {
    model: modelRef.value,
    automaticLayout: true,
    minimap: { enabled: false },
    readOnly: false,
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    fontSize: 13,
    tabSize: 2,
    roundedSelection: false,
    lineNumbersMinChars: 3,
    glyphMargin: false,
    folding: true,
    theme: monacoTheme.value,
  })

  editorRef.value.onDidChangeModelContent(() => {
    draftContent.value = editorRef.value?.getValue() || ''
  })
}

const handleSave = async () => {
  saveLoading.value = true

  try {
    await props.onSave(draftContent.value)
    originalContent.value = draftContent.value
    ElMessage.success('文件保存成功')
  } catch (error: unknown) {
    ElMessage.error((error as Error)?.message || '文件保存失败')
  } finally {
    saveLoading.value = false
  }
}

watch(themeMode, (value) => {
  localStorage.setItem(EDITOR_THEME_STORAGE_KEY, value)
})

watch(monacoTheme, (value) => {
  monaco.editor.setTheme(value)
})

onMounted(() => {
  loadStoredThemeMode()
  initializeEditor()
  nextTick(() => {
    initializeDialogFrame()
  })
})

onBeforeUnmount(() => {
  stopDragging()
  stopResizing()
  editorRef.value?.dispose()
  modelRef.value?.dispose()
})
</script>

<template>
  <section class="file-editor-dialog-content" :class="[`is-${effectiveThemeMode}`]">
    <header class="file-editor-toolbar" @mousedown="handleToolbarMouseDown">
      <div class="file-editor-name">{{ props.fileName }}</div>
      <div class="file-editor-actions">
        <el-select v-model="themeMode" class="file-editor-theme-select" size="small">
          <el-option
            v-for="option in themeModeOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
        <span class="file-editor-language">{{ detectedLanguage }}</span>
        <span v-if="isDirty" class="file-editor-dirty">未保存</span>
        <el-button type="primary" :loading="saveLoading" @click="handleSave">保存</el-button>
        <el-button text class="file-editor-close" @click="props.onClose()">
          <el-icon size="18"><Close /></el-icon>
        </el-button>
      </div>
    </header>

    <div ref="editorHost" class="file-editor-host" />
    <button
      type="button"
      class="file-editor-resize-handle"
      aria-label="调整窗口大小"
      @mousedown="handleResizeHandleMouseDown"
    />
  </section>
</template>

<style scoped lang="scss">
.file-editor-dialog-content {
  --file-editor-surface-bg: #ffffff;
  --file-editor-surface-border: #d9e0ea;
  --file-editor-toolbar-bg: #f7f9fc;
  --file-editor-toolbar-shadow: inset 0 -1px 0 rgb(15 23 42 / 6%);
  --file-editor-text-primary: #1f2937;
  --file-editor-text-secondary: #6b7280;
  --file-editor-chip-bg: rgb(148 163 184 / 14%);
  --file-editor-chip-text: #64748b;
  --file-editor-dirty-bg: rgb(245 158 11 / 16%);
  --file-editor-dirty-text: #b45309;
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: 14px;
  background: var(--file-editor-surface-bg);
}

.file-editor-dialog-content.is-dark {
  --file-editor-surface-bg: #1e1e1e;
  --file-editor-surface-border: #313131;
  --file-editor-toolbar-bg: #252526;
  --file-editor-toolbar-shadow: inset 0 -1px 0 rgb(255 255 255 / 6%);
  --file-editor-text-primary: #f3f4f6;
  --file-editor-text-secondary: #9ca3af;
  --file-editor-chip-bg: rgb(255 255 255 / 8%);
  --file-editor-chip-text: #d1d5db;
  --file-editor-dirty-bg: rgb(245 158 11 / 20%);
  --file-editor-dirty-text: #fbbf24;
}

.file-editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
  min-height: 56px;
  padding: 0.5rem 0.8rem;
  background: var(--file-editor-toolbar-bg);
  box-shadow: var(--file-editor-toolbar-shadow);
  cursor: move;
  user-select: none;
}

.file-editor-name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--file-editor-text-primary);
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-editor-actions {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  flex-shrink: 0;
  justify-content: flex-end;
}

.file-editor-theme-select {
  width: 108px;
}

.file-editor-theme-select :deep(.el-select__wrapper) {
  min-height: 32px;
  border-radius: 999px;
  background: rgb(255 255 255 / 72%);
}

.file-editor-dialog-content.is-dark .file-editor-theme-select :deep(.el-select__wrapper) {
  background: rgb(255 255 255 / 6%);
  box-shadow: 0 0 0 1px rgb(255 255 255 / 8%) inset;
}

.file-editor-language,
.file-editor-dirty {
  border-radius: 999px;
  padding: 0.24rem 0.58rem;
  font-size: 0.75rem;
  line-height: 1;
  white-space: nowrap;
}

.file-editor-language {
  background: var(--file-editor-chip-bg);
  color: var(--file-editor-chip-text);
}

.file-editor-dirty {
  background: var(--file-editor-dirty-bg);
  color: var(--file-editor-dirty-text);
}

.file-editor-close {
  padding-inline: 0.4rem;
  color: var(--file-editor-text-secondary);
}

.file-editor-close:hover {
  color: var(--file-editor-text-primary);
}

.file-editor-host {
  flex: 1;
  min-height: 0;
}

.file-editor-resize-handle {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 18px;
  height: 18px;
  border: none;
  background:
    linear-gradient(
      135deg,
      transparent 0 42%,
      rgb(255 255 255 / 0%) 42% 48%,
      currentcolor 48% 54%,
      transparent 54%
    ),
    linear-gradient(
      135deg,
      transparent 0 60%,
      rgb(255 255 255 / 0%) 60% 66%,
      currentcolor 66% 72%,
      transparent 72%
    );
  color: var(--file-editor-text-secondary);
  cursor: nwse-resize;
  opacity: 0.65;
}

.file-editor-resize-handle:hover {
  opacity: 1;
}

:global(.file-editor-dialog.el-dialog) {
  min-width: 0;
  overflow: hidden !important;
  background: transparent !important;
  box-shadow: none !important;
  border: none !important;
  resize: none !important;
}

:global(.file-editor-dialog.el-dialog .el-dialog__body) {
  padding: 0 !important;
  height: 100%;
}

@media (width <= 768px) {
  .file-editor-toolbar {
    flex-direction: column;
    align-items: stretch;
    cursor: default;
  }

  .file-editor-actions {
    width: 100%;
    justify-content: flex-end;
    flex-wrap: wrap;
  }

  .file-editor-theme-select {
    width: 96px;
  }
}
</style>
