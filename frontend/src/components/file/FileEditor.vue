<template>
  <teleport to="body">
    <transition name="fe-fade">
      <div v-if="modelValue" class="fe-overlay">
        <div
          class="file-module fe-window"
          :class="{ 'is-dark': resolvedDark, 'is-fullscreen': fullscreen, 'is-minimized': minimized }"
          :style="windowStyle"
        >
          <!-- 标题栏（可拖拽） -->
          <header class="fe-titlebar" @mousedown="startDrag" @dblclick="toggleFullscreen">
            <span class="fe-title">{{ t('fileManager.editTitle') }} - {{ activeName || '—' }}</span>
            <div class="fe-title-actions" @mousedown.stop>
              <button type="button" class="fm-icon-btn" :title="t('fileManager.minimize')" @click="toggleMinimize">
                <el-icon><IconMinimize /></el-icon>
              </button>
              <button type="button" class="fm-icon-btn" :title="t('common.close')" @click="close">
                <el-icon><IconClose /></el-icon>
              </button>
            </div>
          </header>

          <template v-if="!minimized">
            <!-- 工具栏 -->
            <div class="fe-toolbar">
              <button
                type="button"
                class="fm-btn fm-btn--primary"
                :disabled="!dirty || saving || isBinary"
                @click="save"
              >
                <el-icon :class="{ 'is-spinning': saving }"><IconSave /></el-icon>{{ t('common.save') }}
              </button>
              <button type="button" class="fm-btn" :disabled="!activePath" @click="refresh">
                <el-icon><IconRefresh /></el-icon>{{ t('fileManager.refresh') }}
              </button>
              <button type="button" class="fm-btn" :disabled="!activePath" @click="download">
                <el-icon><IconDownload /></el-icon>{{ t('fileManager.download') }}
              </button>
              <button type="button" class="fm-btn" :disabled="!activePath" @click="openRename">
                <el-icon><IconRename /></el-icon>{{ t('fileManager.rename') }}
              </button>
              <button
                type="button"
                class="fm-btn fm-btn--danger"
                :disabled="!activePath"
                @click="remove"
              >
                <el-icon><IconDelete /></el-icon>{{ t('fileManager.delete') }}
              </button>

              <div class="fe-toolbar-spacer"></div>

              <FileMenu :items="themeItems" :dark="resolvedDark" @select="setThemeMode">
                <button type="button" class="fm-btn">
                  <el-icon><component :is="themeIcon" /></el-icon>{{ t('fileManager.theme') }}
                  <el-icon><IconChevronDown /></el-icon>
                </button>
              </FileMenu>
              <button
                type="button"
                class="fm-icon-btn"
                :title="fullscreen ? t('fileManager.exitFullscreen') : t('fileManager.fullscreen')"
                @click="toggleFullscreen"
              >
                <el-icon><component :is="fullscreen ? IconExitFullscreen : IconFullscreen" /></el-icon>
              </button>
            </div>

            <!-- 主体：树 | 分隔条 | 编辑区 -->
            <div class="fe-body">
              <aside class="fe-tree" :style="{ width: `${treeWidth}px` }">
                <div class="fe-tree-head">{{ t('fileManager.fileTree') }}</div>
                <div class="fe-tree-scroll">
                  <FileTree
                    ref="treeRef"
                    :client="client"
                    :root-path="rootPath"
                    :active-path="activePath"
                    @select="onTreeSelect"
                  />
                </div>
              </aside>

              <div class="fe-splitter" @mousedown="startTreeResize"></div>

              <div class="fe-main">
                <div ref="monacoEl" class="fe-monaco"></div>
                <div v-if="maskVisible" class="fe-mask">
                  <el-icon class="fe-mask-icon"><IconWarning /></el-icon>
                  <p>{{ activePath ? t('fileManager.cannotEditBinary') : t('fileManager.fileTree') }}</p>
                </div>
              </div>
            </div>

            <!-- 状态栏 -->
            <footer class="fe-statusbar">
              <span class="fe-status-path" :title="activePath">
                {{ t('fileManager.filePath') }}: {{ activePath || '—' }}
              </span>
              <span class="fe-status-spacer"></span>
              <span class="fe-status-lang">{{ languageLabel }}</span>
              <span class="fe-status-sep">{{ t('fileManager.encoding') }}: UTF-8</span>
              <button type="button" class="fe-status-btn" @click="toggleEol">
                {{ t('fileManager.eol') }}: {{ eol }}
              </button>
              <span class="fe-status-sep">{{ t('fileManager.lineCol', { line: cursor.line, col: cursor.col }) }}</span>
              <span>{{ t('fileManager.totalLines', { n: totalLines }) }}</span>
            </footer>
          </template>

          <!-- 右下角缩放手柄 -->
          <div
            v-if="!fullscreen && !minimized"
            class="fe-resize"
            :title="t('fileManager.resizeWindow')"
            @mousedown="startResize"
          ></div>
        </div>
      </div>
    </transition>

    <FilePromptDialog
      v-model="renamePrompt.open"
      :title="t('fileManager.renameTitle')"
      :description="t('fileManager.renameTarget', { name: activeName })"
      :placeholder="t('fileManager.renamePlaceholder')"
      :initial-value="activeName"
      :confirming="renamePrompt.confirming"
      @confirm="onRenameConfirm"
    />
  </teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { showRequestError } from '@/utils/request'
import { useThemeStore } from '@/stores/theme'
import {
  applyEndOfLine,
  detectEndOfLine,
  detectMonacoLanguage,
  downloadFileFromUrl,
  getBaseName,
  isMediaFile,
  resolvePreSignedFileUrl,
  type EndOfLine,
} from '@/utils/file'
import { monaco } from './monaco'
import FileTree from './FileTree.vue'
import FileMenu, { type FileMenuItem } from './FileMenu.vue'
import FilePromptDialog from './FilePromptDialog.vue'
import type { FileClient } from './types'
import {
  IconSave,
  IconRefresh,
  IconDownload,
  IconRename,
  IconDelete,
  IconAuto,
  IconLight,
  IconDark,
  IconFullscreen,
  IconExitFullscreen,
  IconMinimize,
  IconClose,
  IconChevronDown,
  IconWarning,
} from './icons'
import type { CSSProperties } from 'vue'

const props = defineProps<{
  modelValue: boolean
  client: FileClient
  path: string
  rootPath: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
  renamed: []
  deleted: []
}>()

const { t } = useI18n()
const themeStore = useThemeStore()

// ---- 主题（自动/亮/暗，localStorage 持久；整窗 + Monaco 一起切） ----
type ThemeMode = 'auto' | 'light' | 'dark'
const THEME_KEY = 'fileEditorTheme'
const themeMode = ref<ThemeMode>(
  (localStorage.getItem(THEME_KEY) as ThemeMode) || 'auto',
)
const resolvedDark = computed(() =>
  themeMode.value === 'auto' ? themeStore.isDarkTheme : themeMode.value === 'dark',
)
const themeIcon = computed(() =>
  themeMode.value === 'auto' ? IconAuto : themeMode.value === 'light' ? IconLight : IconDark,
)
const themeItems = computed<FileMenuItem[]>(() => [
  { key: 'auto', label: t('fileManager.editorThemeAuto'), icon: IconAuto },
  { key: 'light', label: t('fileManager.editorThemeLight'), icon: IconLight },
  { key: 'dark', label: t('fileManager.editorThemeDark'), icon: IconDark },
])
const setThemeMode = (mode: string) => {
  themeMode.value = mode as ThemeMode
  localStorage.setItem(THEME_KEY, mode)
}
watch(resolvedDark, (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'))

// ---- 窗口几何（拖拽/缩放/全屏/最小化） ----
const win = reactive({ left: 0, top: 0, width: 1200, height: 760 })
const fullscreen = ref(false)
const minimized = ref(false)
const treeWidth = ref(248)

const windowStyle = computed<CSSProperties>(() => {
  if (fullscreen.value) return {}
  const style: CSSProperties = {
    left: `${win.left}px`,
    top: `${win.top}px`,
    width: `${win.width}px`,
  }
  if (!minimized.value) style.height = `${win.height}px`
  return style
})

const computeDefaultGeometry = () => {
  const vw = window.innerWidth
  const vh = window.innerHeight
  win.width = Math.min(1360, vw - 24)
  win.height = Math.min(880, vh - 64)
  win.left = Math.max(12, (vw - win.width) / 2)
  win.top = Math.max(12, (vh - win.height) / 2)
}

const toggleFullscreen = () => {
  if (minimized.value) minimized.value = false
  fullscreen.value = !fullscreen.value
}
const toggleMinimize = () => {
  if (fullscreen.value) fullscreen.value = false
  minimized.value = !minimized.value
}

// 拖拽移动
const startDrag = (event: MouseEvent) => {
  if (fullscreen.value) return
  const startX = event.clientX
  const startY = event.clientY
  const originLeft = win.left
  const originTop = win.top
  const onMove = (e: MouseEvent) => {
    win.left = Math.max(0, originLeft + (e.clientX - startX))
    win.top = Math.max(0, originTop + (e.clientY - startY))
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// 右下角缩放
const startResize = (event: MouseEvent) => {
  event.preventDefault()
  const startX = event.clientX
  const startY = event.clientY
  const originW = win.width
  const originH = win.height
  const onMove = (e: MouseEvent) => {
    win.width = Math.max(640, originW + (e.clientX - startX))
    win.height = Math.max(420, originH + (e.clientY - startY))
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// 树宽拖拽
const startTreeResize = (event: MouseEvent) => {
  event.preventDefault()
  const startX = event.clientX
  const originW = treeWidth.value
  const onMove = (e: MouseEvent) => {
    treeWidth.value = Math.min(480, Math.max(160, originW + (e.clientX - startX)))
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// ---- Monaco ----
const monacoEl = ref<HTMLElement | null>(null)
let editorInstance: monaco.editor.IStandaloneCodeEditor | null = null

const activePath = ref('')
const activeName = ref('')
const originalContent = ref('')
const dirty = ref(false)
const saving = ref(false)
const isBinary = ref(false)
const language = ref('plaintext')
const eol = ref<EndOfLine>('LF')
const cursor = reactive({ line: 1, col: 1 })
const totalLines = ref(1)

const maskVisible = computed(() => !activePath.value || isBinary.value)
const languageLabel = computed(() =>
  language.value === 'plaintext' ? t('fileManager.plainText') : language.value.toUpperCase(),
)

const ensureEditor = async () => {
  if (editorInstance || !monacoEl.value) return
  editorInstance = monaco.editor.create(monacoEl.value, {
    value: '',
    language: 'plaintext',
    theme: resolvedDark.value ? 'vs-dark' : 'vs',
    automaticLayout: true,
    fontSize: 13,
    lineHeight: 20,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    tabSize: 2,
    renderWhitespace: 'selection',
    fontFamily: "'JetBrains Mono', 'Cascadia Code', Consolas, 'Courier New', monospace",
    smoothScrolling: true,
  })
  editorInstance.onDidChangeModelContent(() => {
    if (!editorInstance) return
    dirty.value = editorInstance.getValue() !== originalContent.value
    totalLines.value = editorInstance.getModel()?.getLineCount() ?? 1
  })
  editorInstance.onDidChangeCursorPosition((e) => {
    cursor.line = e.position.lineNumber
    cursor.col = e.position.column
  })
}

const applyContent = (text: string) => {
  if (!editorInstance) return
  eol.value = detectEndOfLine(text)
  editorInstance.setValue(text)
  const model = editorInstance.getModel()
  if (model) {
    monaco.editor.setModelLanguage(model, language.value)
    model.setEOL(
      eol.value === 'CRLF'
        ? monaco.editor.EndOfLineSequence.CRLF
        : monaco.editor.EndOfLineSequence.LF,
    )
    totalLines.value = model.getLineCount()
  }
  originalContent.value = editorInstance.getValue()
  dirty.value = false
  cursor.line = 1
  cursor.col = 1
}

// ---- 打开文件 ----
const confirmDiscardIfDirty = async (): Promise<boolean> => {
  if (!dirty.value) return true
  try {
    await ElMessageBox.confirm(t('fileManager.discardChangesConfirm'), t('fileManager.unsaved'), {
      type: 'warning',
      confirmButtonText: t('system.common.confirm'),
      cancelButtonText: t('system.common.cancel'),
    })
    return true
  } catch {
    return false
  }
}

const openPath = async (path: string, name?: string) => {
  if (!path) return
  if (!(await confirmDiscardIfDirty())) return

  activePath.value = path
  activeName.value = name || getBaseName(path)
  language.value = detectMonacoLanguage(activeName.value)

  // 媒体在编辑区给遮罩（走列表的预览，不在编辑器打开）
  if (isMediaFile(activeName.value)) {
    isBinary.value = true
    return
  }

  try {
    const result = await props.client.open(path)
    if (result.isBinary) {
      isBinary.value = true
      return
    }
    isBinary.value = false
    await ensureEditor()
    applyContent(result.text)
  } catch (error) {
    showRequestError(error, t('fileManager.openFailed'))
  }
}

const onTreeSelect = (path: string, name: string) => openPath(path, name)

// ---- 操作 ----
const save = async () => {
  if (!editorInstance || !dirty.value || isBinary.value) return
  saving.value = true
  try {
    const value = applyEndOfLine(editorInstance.getValue(), eol.value)
    await props.client.edit(activePath.value, value)
    originalContent.value = editorInstance.getValue()
    dirty.value = false
    ElMessage.success(t('fileManager.fileSaveSuccess'))
    emit('saved')
  } catch (error) {
    showRequestError(error, t('fileManager.fileSaveFailed'))
  } finally {
    saving.value = false
  }
}

const refresh = async () => {
  if (!activePath.value) return
  await openPath(activePath.value, activeName.value)
}

const download = async () => {
  if (!activePath.value) return
  try {
    const downloadPath = await props.client.preSignDownload(activePath.value)
    downloadFileFromUrl(resolvePreSignedFileUrl(downloadPath), activeName.value)
  } catch (error) {
    showRequestError(error, t('fileManager.downloadFailed'))
  }
}

const treeRef = ref<InstanceType<typeof FileTree> | null>(null)
const renamePrompt = reactive({ open: false, confirming: false })
const openRename = () => {
  if (!activePath.value) return
  renamePrompt.open = true
}
const onRenameConfirm = async (value: string) => {
  renamePrompt.confirming = true
  try {
    const newPath = await props.client.rename(activePath.value, value)
    activePath.value = newPath || activePath.value
    activeName.value = value
    language.value = detectMonacoLanguage(value)
    if (editorInstance) {
      const model = editorInstance.getModel()
      if (model) monaco.editor.setModelLanguage(model, language.value)
    }
    renamePrompt.open = false
    ElMessage.success(t('fileManager.renameSuccess'))
    treeRef.value?.reload()
    emit('renamed')
  } catch (error) {
    showRequestError(error, t('fileManager.renameFailed'))
  } finally {
    renamePrompt.confirming = false
  }
}

const remove = async () => {
  if (!activePath.value) return
  try {
    await ElMessageBox.confirm(
      t('fileManager.deleteOneConfirm', { name: activeName.value }),
      t('fileManager.confirmDelete'),
      {
        type: 'warning',
        confirmButtonText: t('system.common.confirm'),
        cancelButtonText: t('system.common.cancel'),
      },
    )
  } catch {
    return
  }
  try {
    await props.client.remove([activePath.value])
    ElMessage.success(t('fileManager.deleteSuccess'))
    activePath.value = ''
    activeName.value = ''
    isBinary.value = false
    originalContent.value = ''
    dirty.value = false
    editorInstance?.setValue('')
    treeRef.value?.reload()
    emit('deleted')
  } catch (error) {
    showRequestError(error, t('fileManager.deleteFailed'))
  }
}

const toggleEol = () => {
  eol.value = eol.value === 'LF' ? 'CRLF' : 'LF'
  const model = editorInstance?.getModel()
  if (model) {
    model.setEOL(
      eol.value === 'CRLF'
        ? monaco.editor.EndOfLineSequence.CRLF
        : monaco.editor.EndOfLineSequence.LF,
    )
    dirty.value = editorInstance!.getValue() !== originalContent.value
  }
}

// ---- 关闭 / 生命周期 ----
const close = async () => {
  if (!(await confirmDiscardIfDirty())) return
  emit('update:modelValue', false)
}

const disposeEditor = () => {
  editorInstance?.dispose()
  editorInstance = null
}

watch(
  () => props.modelValue,
  async (open) => {
    if (open) {
      computeDefaultGeometry()
      fullscreen.value = false
      minimized.value = false
      await nextTick()
      await ensureEditor()
      if (props.path) await openPath(props.path)
    } else {
      disposeEditor()
      activePath.value = ''
      activeName.value = ''
      dirty.value = false
      isBinary.value = false
    }
  },
)

// 已打开状态下浏览器请求打开另一个文件
watch(
  () => props.path,
  (path) => {
    if (props.modelValue && path && path !== activePath.value) openPath(path)
  },
)

onBeforeUnmount(disposeEditor)
</script>

<style scoped>
.fe-overlay {
  position: fixed;
  inset: 0;
  z-index: 2100;
  background: rgba(15, 23, 42, 0.32);
}

.fe-window {
  position: fixed;
  display: flex;
  flex-direction: column;
  min-width: 640px;
  min-height: 0;
  background: var(--fm-bg);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius);
  box-shadow: var(--fm-shadow);
  overflow: hidden;
}
.fe-window.is-fullscreen {
  inset: 0;
  width: 100vw !important;
  height: 100vh !important;
  border-radius: 0;
}
.fe-window.is-minimized {
  height: auto !important;
}

/* 标题栏 */
.fe-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  height: 40px;
  padding: 0 0.5rem 0 0.875rem;
  background: var(--fm-header);
  border-bottom: 1px solid var(--fm-border);
  cursor: move;
  user-select: none;
}
.fe-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--fm-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fe-title-actions {
  display: flex;
  gap: 0.125rem;
}

/* 工具栏 */
.fe-toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid var(--fm-border);
  background: var(--fm-surface);
}
.fe-toolbar-spacer {
  flex: 1;
}
.is-spinning {
  animation: fe-spin 0.8s linear infinite;
}
@keyframes fe-spin {
  from {
    transform: perspective(120px) rotateY(0deg);
  }
  to {
    transform: perspective(120px) rotateY(360deg);
  }
}

/* 主体 */
.fe-body {
  flex: 1;
  display: flex;
  min-height: 0;
}
.fe-tree {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  min-width: 0;
  background: var(--fm-surface);
  border-right: 1px solid var(--fm-border);
}
.fe-tree-head {
  height: 34px;
  display: flex;
  align-items: center;
  padding: 0 0.875rem;
  font-size: 12px;
  font-weight: 600;
  color: var(--fm-text-3);
  border-bottom: 1px solid var(--fm-border);
}
.fe-tree-scroll {
  flex: 1;
  overflow: auto;
}
.fe-splitter {
  width: 5px;
  flex-shrink: 0;
  cursor: col-resize;
  background: transparent;
  transition: background 0.15s;
}
.fe-splitter:hover {
  background: var(--fm-accent-soft);
}
.fe-main {
  position: relative;
  flex: 1;
  min-width: 0;
  background: var(--fm-bg);
}
.fe-monaco {
  position: absolute;
  inset: 0;
}
.fe-mask {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  background: var(--fm-bg);
  color: var(--fm-text-3);
  font-size: 13px;
}
.fe-mask-icon {
  font-size: 32px;
  color: var(--fm-text-3);
}

/* 状态栏 */
.fe-statusbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  height: 28px;
  padding: 0 0.875rem;
  background: var(--fm-header);
  border-top: 1px solid var(--fm-border);
  font-size: 12px;
  color: var(--fm-text-2);
  white-space: nowrap;
  overflow: hidden;
}
.fe-status-path {
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 40%;
}
.fe-status-spacer {
  flex: 1;
}
.fe-status-lang {
  color: var(--fm-accent);
  font-weight: 600;
}
.fe-status-btn {
  border: none;
  background: transparent;
  color: var(--fm-text-2);
  font: inherit;
  cursor: pointer;
  padding: 0;
}
.fe-status-btn:hover {
  color: var(--fm-accent);
}

/* 缩放手柄 */
.fe-resize {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 16px;
  height: 16px;
  cursor: nwse-resize;
  background: linear-gradient(
    135deg,
    transparent 0 50%,
    var(--fm-border-strong) 50% 60%,
    transparent 60% 70%,
    var(--fm-border-strong) 70% 80%,
    transparent 80%
  );
}

.fe-fade-enter-active,
.fe-fade-leave-active {
  transition: opacity 0.18s ease;
}
.fe-fade-enter-from,
.fe-fade-leave-to {
  opacity: 0;
}
</style>
