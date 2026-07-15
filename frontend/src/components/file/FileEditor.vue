<!-- 文件编辑器（重写 · 新方案）：浮窗 Monaco 编辑器，自包含独立明暗主题（--fe-*，手动 auto/亮/暗，
     不依赖全局 app 主题，也不用旧 file-module/fm-* 框架）。图标走 menuStore heroicons + 本地保存 SVG；
     主题用分段切换（替代旧 FileMenu 下拉）；提示/确认走 useFeedback + 令牌 FormDialog（不用 EP ElMessage/ElMessageBox）。
     Monaco 集成、窗口几何（拖拽/缩放/全屏/最小化）、打开/保存/重命名/删除/下载/EOL、目录树、移动端全部逻辑保留。 -->
<template>
  <teleport to="body">
    <transition name="fe-fade">
      <div v-if="modelValue" class="fe-overlay">
        <div
          class="fe-window"
          :class="{
            'is-dark': resolvedDark,
            'is-fullscreen': fullscreen,
            'is-minimized': minimized,
          }"
          :style="windowStyle"
        >
          <!-- 标题栏（可拖拽） -->
          <header class="fe-titlebar" @mousedown="startDrag" @dblclick="toggleFullscreen">
            <span class="fe-title">{{ t('fileManager.editTitle') }} - {{ activeName || '—' }}</span>
            <div class="fe-titlebar__actions" @mousedown.stop>
              <button type="button" class="fe-iconbtn" :title="t('fileManager.minimize')" @click="toggleMinimize">
                <component :is="ic('HOutline:MinusIcon')" />
              </button>
              <button type="button" class="fe-iconbtn" :title="t('common.close')" @click="close">
                <component :is="ic('HOutline:XMarkIcon')" />
              </button>
            </div>
          </header>

          <template v-if="!minimized">
            <!-- 工具栏 -->
            <div class="fe-toolbar">
              <button type="button" class="fe-btn fe-btn--primary" :disabled="!dirty || saving || isBinary" @click="save">
                <component :is="IconSave" :class="{ 'is-spin': saving }" />{{ t('common.save') }}
              </button>
              <button type="button" class="fe-btn" :disabled="!activePath" @click="refresh">
                <component :is="ic('HOutline:ArrowPathIcon')" />{{ t('fileManager.refresh') }}
              </button>
              <button type="button" class="fe-btn" :disabled="!activePath" @click="download">
                <component :is="ic('HOutline:ArrowDownTrayIcon')" />{{ t('fileManager.download') }}
              </button>
              <button type="button" class="fe-btn" :disabled="!activePath" @click="openRename">
                <component :is="ic('HOutline:PencilSquareIcon')" />{{ t('fileManager.rename') }}
              </button>
              <button type="button" class="fe-btn fe-btn--danger" :disabled="!activePath" @click="remove">
                <component :is="ic('HOutline:TrashIcon')" />{{ t('fileManager.delete') }}
              </button>

              <div class="fe-toolbar__spacer"></div>

              <!-- 主题分段（自动/亮/暗），替代旧 FileMenu 下拉 -->
              <div class="fe-seg" role="group" :aria-label="t('fileManager.theme')">
                <button
                  v-for="mode in themeModes"
                  :key="mode.key"
                  type="button"
                  class="fe-seg__btn"
                  :class="{ 'is-active': themeMode === mode.key }"
                  :title="mode.label"
                  @click="setThemeMode(mode.key)"
                >
                  <component :is="ic(mode.icon)" />
                </button>
              </div>
              <button
                type="button"
                class="fe-iconbtn"
                :title="fullscreen ? t('fileManager.exitFullscreen') : t('fileManager.fullscreen')"
                @click="toggleFullscreen"
              >
                <component :is="ic(fullscreen ? 'HOutline:ArrowsPointingInIcon' : 'HOutline:ArrowsPointingOutIcon')" />
              </button>
            </div>

            <!-- 主体：树 | 分隔条 | 编辑区 -->
            <div class="fe-body">
              <button type="button" class="fe-tree-toggle" @click="mobileTreeOpen = true">
                {{ t('fileManager.fileTree') }}
              </button>
              <div v-if="mobileTreeOpen" class="fe-tree-backdrop" @click="mobileTreeOpen = false"></div>

              <aside class="fe-tree" :class="{ 'is-mobile-open': mobileTreeOpen }" :style="{ width: `${treeWidth}px` }">
                <div class="fe-tree__head">
                  <span>{{ t('fileManager.fileTree') }}</span>
                  <button type="button" class="fe-tree__close" @click="mobileTreeOpen = false">
                    <component :is="ic('HOutline:XMarkIcon')" />
                  </button>
                </div>
                <div class="fe-tree__scroll">
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
                  <component :is="ic('HOutline:ExclamationTriangleIcon')" class="fe-mask__ico" />
                  <p>{{ activePath ? t('fileManager.cannotEditBinary') : t('fileManager.fileTree') }}</p>
                </div>
              </div>
            </div>

            <!-- 状态栏 -->
            <footer class="fe-statusbar">
              <span class="fe-statusbar__path" :title="activePath">
                {{ t('fileManager.filePath') }}: {{ activePath || '—' }}
              </span>
              <span class="fe-statusbar__spacer"></span>
              <span class="fe-statusbar__lang">{{ languageLabel }}</span>
              <span class="fe-statusbar__sep">{{ t('fileManager.encoding') }}: UTF-8</span>
              <button type="button" class="fe-statusbar__btn" @click="toggleEol">
                {{ t('fileManager.eol') }}: {{ eol }}
              </button>
              <span class="fe-statusbar__sep">{{ t('fileManager.lineCol', { line: cursor.line, col: cursor.col }) }}</span>
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

    <!-- 通用确认（丢弃未保存 / 删除），令牌 FormDialog，替代 EP ElMessageBox -->
    <FormDialog
      v-model="confirmState.open"
      :title="confirmState.title"
      :width="420"
      @close="resolveConfirm(false)"
    >
      <p class="fe-confirm">{{ confirmState.message }}</p>
      <template #footer>
        <UButton color="neutral" variant="soft" @click="resolveConfirm(false)">
          {{ t('system.common.cancel') }}
        </UButton>
        <UButton :color="confirmState.danger ? 'error' : 'primary'" @click="resolveConfirm(true)">
          {{ t('system.common.confirm') }}
        </UButton>
      </template>
    </FormDialog>
  </teleport>
</template>

<script setup lang="ts">
import { defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import { useThemeStore } from '@/stores/theme'
import { FileSortField } from '@/types/v1/file'
import {
  applyEndOfLine,
  detectEndOfLine,
  detectMonacoLanguage,
  downloadFileFromUrl,
  getBaseName,
  getParentPath,
  isMediaFile,
  isFileTooLargeForEditor,
  resolvePreSignedFileUrl,
  type EndOfLine,
} from '@/utils/file'
import { monaco } from './monaco'
import FileTree from './FileTree.vue'
import FilePromptDialog from './FilePromptDialog.vue'
import type { FileClient } from './types'
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
const menuStore = useMenuStore()
const fb = useFeedback()

const ic = (key: string) => menuStore.iconComponents[key]
const notifyError = (error: unknown, fallback: string) => fb.error(getRequestErrorMessage(error, fallback))

// 保存图标（软盘）：Heroicons 无软盘，用本地内联 SVG（非旧框架，纯 SVG 组件）。
const IconSave = defineComponent({
  name: 'IconSave',
  render: () =>
    h(
      'svg',
      {
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: 'currentColor',
        'stroke-width': 1.5,
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        'aria-hidden': 'true',
      },
      [
        h('path', {
          d: 'M4.5 5.25A1.75 1.75 0 0 1 6.25 3.5h9.19c.46 0 .91.19 1.24.51l2.81 2.82c.33.33.51.78.51 1.24v10.68a1.75 1.75 0 0 1-1.75 1.75H6.25A1.75 1.75 0 0 1 4.5 18.75z',
        }),
        h('path', { d: 'M8 3.5v4.25c0 .41.34.75.75.75h5a.75.75 0 0 0 .75-.75V3.5' }),
        h('rect', { x: 7.5, y: 12, width: 9, height: 6, rx: 0.75 }),
      ],
    ),
})

// ---- 主题（自动/亮/暗，localStorage 持久；整窗 + Monaco 一起切） ----
type ThemeMode = 'auto' | 'light' | 'dark'
const THEME_KEY = 'fileEditorTheme'
const themeMode = ref<ThemeMode>((localStorage.getItem(THEME_KEY) as ThemeMode) || 'auto')
const resolvedDark = computed(() =>
  themeMode.value === 'auto' ? themeStore.isDarkTheme : themeMode.value === 'dark',
)
const themeModes: { key: ThemeMode; label: string; icon: string }[] = [
  { key: 'auto', label: t('fileManager.editorThemeAuto'), icon: 'HOutline:ComputerDesktopIcon' },
  { key: 'light', label: t('fileManager.editorThemeLight'), icon: 'HOutline:SunIcon' },
  { key: 'dark', label: t('fileManager.editorThemeDark'), icon: 'HOutline:MoonIcon' },
]
const setThemeMode = (mode: ThemeMode) => {
  themeMode.value = mode
  localStorage.setItem(THEME_KEY, mode)
}
watch(resolvedDark, (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'))

// ---- 窗口几何（拖拽/缩放/全屏/最小化） ----
const win = reactive({ left: 0, top: 0, width: 1200, height: 760 })
const fullscreen = ref(false)
const minimized = ref(false)
const treeWidth = ref(248)
const mobileTreeOpen = ref(false)

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
  if (vw <= 767) {
    win.width = Math.max(320, vw - 16)
    win.height = Math.max(360, vh - 16)
    win.left = 8
    win.top = 8
    treeWidth.value = Math.min(300, Math.max(240, vw - 72))
    return
  }

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
  if (fullscreen.value || window.innerWidth <= 767) return
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
  if (window.innerWidth <= 767) return
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

// ---- 通用确认（丢弃未保存 / 删除）：令牌 FormDialog + Promise 门闸，替代 EP ElMessageBox ----
const confirmState = reactive({ open: false, title: '', message: '', danger: false })
let confirmResolve: ((value: boolean) => void) | null = null
const askConfirm = (title: string, message: string, danger = false) =>
  new Promise<boolean>((resolve) => {
    confirmState.title = title
    confirmState.message = message
    confirmState.danger = danger
    confirmState.open = true
    confirmResolve = resolve
  })
const resolveConfirm = (value: boolean) => {
  if (!confirmState.open && !confirmResolve) return
  confirmState.open = false
  const resolver = confirmResolve
  confirmResolve = null
  resolver?.(value)
}

// ---- 打开文件 ----
const normalizePathForCompare = (path: string) => path.replace(/\\/g, '/').replace(/\/+$/, '')
const isSamePath = (left: string, right: string) =>
  normalizePathForCompare(left).toLowerCase() === normalizePathForCompare(right).toLowerCase()
const editorTooLargeMessage = () => t('fileManager.fileTooLargeForEditor')
const warnEditorFileTooLarge = () => fb.warning(editorTooLargeMessage())

const lookupFileSize = async (path: string, name: string): Promise<number | undefined> => {
  try {
    const { items } = await props.client.list({
      path: getParentPath(path),
      page: 1,
      pageSize: 100,
      keywords: name,
      sortField: FileSortField.FILE_SORT_FIELD_NAME,
      isDesc: false,
    })
    return items.find((item) => !item.isDir && isSamePath(item.path, path))?.size
  } catch {
    return undefined
  }
}

const ensureEditorFileSizeAllowed = async (path: string, name: string): Promise<boolean> => {
  if (isMediaFile(name)) return true
  const size = await lookupFileSize(path, name)
  if (!isFileTooLargeForEditor(name, size)) return true
  warnEditorFileTooLarge()
  return false
}

const confirmDiscardIfDirty = async (): Promise<boolean> => {
  if (!dirty.value) return true
  return askConfirm(t('fileManager.unsaved'), t('fileManager.discardChangesConfirm'))
}

const openPath = async (path: string, name?: string) => {
  if (!path) return
  const nextName = name || getBaseName(path)
  if (!(await ensureEditorFileSizeAllowed(path, nextName))) return
  if (!(await confirmDiscardIfDirty())) return

  const previous = {
    path: activePath.value,
    name: activeName.value,
    language: language.value,
    isBinary: isBinary.value,
  }
  activePath.value = path
  activeName.value = nextName
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
    if (error instanceof Error && error.message === editorTooLargeMessage()) {
      activePath.value = previous.path
      activeName.value = previous.name
      language.value = previous.language
      isBinary.value = previous.isBinary
      warnEditorFileTooLarge()
      return
    }
    notifyError(error, t('fileManager.openFailed'))
  }
}

const onTreeSelect = (path: string, name: string) => {
  mobileTreeOpen.value = false
  openPath(path, name)
}

// ---- 操作 ----
const save = async () => {
  if (!editorInstance || !dirty.value || isBinary.value) return
  saving.value = true
  try {
    const value = applyEndOfLine(editorInstance.getValue(), eol.value)
    await props.client.edit(activePath.value, value)
    originalContent.value = editorInstance.getValue()
    dirty.value = false
    fb.success(t('fileManager.fileSaveSuccess'))
    emit('saved')
  } catch (error) {
    notifyError(error, t('fileManager.fileSaveFailed'))
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
    notifyError(error, t('fileManager.downloadFailed'))
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
    fb.success(t('fileManager.renameSuccess'))
    treeRef.value?.reload()
    emit('renamed')
  } catch (error) {
    notifyError(error, t('fileManager.renameFailed'))
  } finally {
    renamePrompt.confirming = false
  }
}

const remove = async () => {
  if (!activePath.value) return
  const ok = await askConfirm(
    t('fileManager.confirmDelete'),
    t('fileManager.deleteOneConfirm', { name: activeName.value }),
    true,
  )
  if (!ok) return
  try {
    await props.client.remove([activePath.value])
    fb.success(t('fileManager.deleteSuccess'))
    activePath.value = ''
    activeName.value = ''
    isBinary.value = false
    originalContent.value = ''
    dirty.value = false
    editorInstance?.setValue('')
    treeRef.value?.reload()
    emit('deleted')
  } catch (error) {
    notifyError(error, t('fileManager.deleteFailed'))
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
      mobileTreeOpen.value = false
      await nextTick()
      await ensureEditor()
      if (props.path) await openPath(props.path)
    } else {
      disposeEditor()
      activePath.value = ''
      activeName.value = ''
      dirty.value = false
      isBinary.value = false
      mobileTreeOpen.value = false
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
/* 自包含独立主题令牌（浅色默认 + .is-dark 暗色），不依赖全局 app 主题、不用旧 fm-* 框架。 */
.fe-window {
  --fe-bg: #ffffff;
  --fe-surface: #ffffff;
  --fe-chrome: #f6f8f8;
  --fe-hover: #f2f5f5;
  --fe-active-bg: #e2f7f3;
  --fe-border: #e4e8ee;
  --fe-border-strong: #d4dae3;
  --fe-fg: #1f2328;
  --fe-fg-dim: #57606a;
  --fe-fg-faint: #8c959f;
  --fe-accent: #14b8a6;
  --fe-accent-hover: #0d9488;
  --fe-danger: #e5484d;
  --fe-danger-soft: #fdeced;
  --fe-folder: #e3a008;
  --fe-radius: 12px;
  --fe-radius-sm: 8px;
  --fe-shadow: 0 12px 40px rgba(15, 23, 42, 0.18);

  /* 目录树令牌桥接：FileTreeNode 用 --ft-*，映射到编辑器独立主题，使内嵌树跟随编辑器明暗。 */
  --ft-fg: var(--fe-fg-dim);
  --ft-fg-dim: var(--fe-fg-faint);
  --ft-hover: var(--fe-hover);
  --ft-active-bg: var(--fe-active-bg);
  --ft-active-fg: var(--fe-accent);
  --ft-folder: var(--fe-folder);
}
.fe-window.is-dark {
  --fe-bg: #1e1e1e;
  --fe-surface: #252526;
  --fe-chrome: #2d2d30;
  --fe-hover: #2a2d2e;
  --fe-active-bg: #103b38;
  --fe-border: #3c3c3c;
  --fe-border-strong: #4a4a4a;
  --fe-fg: #e6e6e6;
  --fe-fg-dim: #b8bcc2;
  --fe-fg-faint: #8a9099;
  --fe-accent: #2dd4bf;
  --fe-accent-hover: #5eead4;
  --fe-danger: #f87171;
  --fe-danger-soft: #3a1d1f;
  --fe-folder: #e3a008;
  --fe-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
}

.fe-overlay {
  position: fixed;
  inset: 0;
  z-index: 1990;
  background: rgba(15, 23, 42, 0.32);
}

.fe-window {
  position: fixed;
  display: flex;
  flex-direction: column;
  min-width: 640px;
  min-height: 0;
  color: var(--fe-fg);
  background: var(--fe-bg);
  border: 1px solid var(--fe-border);
  border-radius: var(--fe-radius);
  box-shadow: var(--fe-shadow);
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

/* 图标按钮（标题栏/工具栏） */
.fe-iconbtn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  border: none;
  border-radius: var(--fe-radius-sm);
  background: transparent;
  color: var(--fe-fg-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fe-iconbtn:hover {
  background: var(--fe-hover);
  color: var(--fe-fg);
}
.fe-iconbtn :deep(svg) {
  width: 17px;
  height: 17px;
}

/* 标题栏 */
.fe-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  height: 40px;
  padding: 0 0.5rem 0 0.875rem;
  background: var(--fe-chrome);
  border-bottom: 1px solid var(--fe-border);
  cursor: move;
  user-select: none;
}
.fe-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--fe-fg);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fe-titlebar__actions {
  display: flex;
  gap: 2px;
}

/* 工具栏 */
.fe-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--fe-border);
  background: var(--fe-surface);
}
.fe-toolbar__spacer {
  flex: 1;
}

/* 令牌按钮（工具栏文字按钮） */
.fe-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  flex-shrink: 0;
  border: 1px solid var(--fe-border);
  border-radius: var(--fe-radius-sm);
  background: var(--fe-surface);
  color: var(--fe-fg-dim);
  font-size: 0.8125rem;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.fe-btn:hover:not(:disabled) {
  border-color: var(--fe-border-strong);
  color: var(--fe-fg);
  background: var(--fe-hover);
}
.fe-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.fe-btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.fe-btn--primary {
  background: var(--fe-accent);
  border-color: var(--fe-accent);
  color: #fff;
}
.fe-btn--primary:hover:not(:disabled) {
  background: var(--fe-accent-hover);
  border-color: var(--fe-accent-hover);
  color: #fff;
}
.fe-btn--danger {
  color: var(--fe-danger);
}
.fe-btn--danger:hover:not(:disabled) {
  background: var(--fe-danger-soft);
  border-color: var(--fe-danger);
  color: var(--fe-danger);
}
.is-spin {
  animation: fe-spin 0.8s linear infinite;
}
@keyframes fe-spin {
  to {
    transform: rotate(360deg);
  }
}

/* 主题分段 */
.fe-seg {
  display: inline-flex;
  flex-shrink: 0;
  border: 1px solid var(--fe-border);
  border-radius: var(--fe-radius-sm);
  overflow: hidden;
}
.fe-seg__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: var(--fe-surface);
  color: var(--fe-fg-faint);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fe-seg__btn + .fe-seg__btn {
  border-left: 1px solid var(--fe-border);
}
.fe-seg__btn:hover {
  color: var(--fe-fg);
  background: var(--fe-hover);
}
.fe-seg__btn.is-active {
  color: var(--fe-accent);
  background: var(--fe-active-bg);
}
.fe-seg__btn :deep(svg) {
  width: 16px;
  height: 16px;
}

/* 主体 */
.fe-body {
  flex: 1;
  display: flex;
  min-height: 0;
  position: relative;
}
.fe-tree-toggle,
.fe-tree__close,
.fe-tree-backdrop {
  display: none;
}
.fe-tree {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  min-width: 0;
  background: var(--fe-surface);
  border-right: 1px solid var(--fe-border);
}
.fe-tree__head {
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0.875rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--fe-fg-faint);
  border-bottom: 1px solid var(--fe-border);
}
.fe-tree__scroll {
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
  background: var(--fe-active-bg);
}
.fe-main {
  position: relative;
  flex: 1;
  min-width: 0;
  background: var(--fe-bg);
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
  gap: 8px;
  background: var(--fe-bg);
  color: var(--fe-fg-faint);
  font-size: 0.8125rem;
}
.fe-mask__ico {
  width: 32px;
  height: 32px;
  color: var(--fe-fg-faint);
}

/* 状态栏 */
.fe-statusbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  height: 28px;
  padding: 0 0.875rem;
  background: var(--fe-chrome);
  border-top: 1px solid var(--fe-border);
  font-size: 0.75rem;
  color: var(--fe-fg-dim);
  white-space: nowrap;
  overflow: hidden;
}
.fe-statusbar__path {
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 40%;
}
.fe-statusbar__spacer {
  flex: 1;
}
.fe-statusbar__lang {
  color: var(--fe-accent);
  font-weight: 600;
}
.fe-statusbar__btn {
  border: none;
  background: transparent;
  color: var(--fe-fg-dim);
  font: inherit;
  cursor: pointer;
  padding: 0;
}
.fe-statusbar__btn:hover {
  color: var(--fe-accent);
}

/* 确认弹窗文案 */
.fe-confirm {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
  word-break: break-word;
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
    var(--fe-border-strong) 50% 60%,
    transparent 60% 70%,
    var(--fe-border-strong) 70% 80%,
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

@media (max-width: 767px) {
  .fe-window:not(.is-minimized) {
    left: 8px !important;
    top: 8px !important;
    width: calc(100vw - 16px) !important;
    height: calc(100dvh - 16px) !important;
    min-width: 0;
    max-width: calc(100vw - 16px);
    max-height: calc(100dvh - 16px);
  }
  .fe-window.is-minimized {
    left: 8px !important;
    top: 8px !important;
    width: calc(100vw - 16px) !important;
    min-width: 0;
  }
  .fe-titlebar {
    height: 36px;
    padding-left: 0.75rem;
    cursor: default;
  }
  .fe-toolbar {
    overflow-x: auto;
    gap: 6px;
    padding: 7px 10px;
  }
  .fe-toolbar .fe-btn,
  .fe-toolbar .fe-iconbtn {
    flex-shrink: 0;
  }
  .fe-body {
    overflow: hidden;
  }
  .fe-tree-toggle {
    position: absolute;
    left: 0;
    top: 50%;
    z-index: 3;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 84px;
    padding: 0;
    border: 1px solid var(--fe-border);
    border-left: none;
    border-radius: 0 var(--fe-radius-sm) var(--fe-radius-sm) 0;
    background: var(--fe-surface);
    color: var(--fe-fg-dim);
    font-size: 0.75rem;
    writing-mode: vertical-rl;
    transform: translateY(-50%);
  }
  .fe-tree-backdrop {
    position: absolute;
    inset: 0;
    z-index: 4;
    display: block;
    background: rgba(15, 23, 42, 0.24);
  }
  .fe-tree {
    position: absolute;
    inset: 0 auto 0 0;
    z-index: 5;
    width: min(300px, calc(100vw - 56px)) !important;
    max-width: calc(100vw - 56px);
    transform: translateX(-100%);
    transition: transform 0.18s ease;
    box-shadow: var(--fe-shadow);
  }
  .fe-tree.is-mobile-open {
    transform: translateX(0);
  }
  .fe-tree__close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: none;
    border-radius: var(--fe-radius-sm);
    background: transparent;
    color: var(--fe-fg-faint);
  }
  .fe-tree__close :deep(svg) {
    width: 16px;
    height: 16px;
  }
  .fe-splitter {
    display: none;
  }
  .fe-main {
    flex-basis: 100%;
  }
  .fe-statusbar {
    gap: 0.55rem;
    height: 26px;
    padding: 0 0.6rem;
  }
  .fe-statusbar__path {
    max-width: 62%;
  }
  .fe-statusbar__sep,
  .fe-statusbar__btn,
  .fe-statusbar > span:last-child {
    display: none;
  }
  .fe-resize {
    display: none;
  }
}
</style>
