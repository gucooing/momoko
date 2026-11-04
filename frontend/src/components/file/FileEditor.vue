<!-- 文件编辑弹窗（原型：编辑改为弹窗，不挤占文件列表；遮罩点击不关闭，避免误丢未保存的修改）：
     头部（贴纸 + 文件名 + 「已修改 · 未保存」+ 路径 · 查找替换 / 文件树 / ⋯ / 关闭）+ Monaco 编辑区（可展开左侧文件树切换文件）+
     底部状态（语言 · 编码 · 行尾 · 光标）与 取消 / 保存（Ctrl S）。
     打开 / 保存 / 刷新 / 下载 / 重命名 / 删除 / 行尾切换 / 编辑器明暗 / 全屏 均保留；props/emits 契约不变。 -->
<template>
  <teleport to="body">
    <transition name="fe-fade">
      <div v-if="modelValue" class="fe-overlay" :class="{ 'is-full': fullscreen }">
        <section
          class="fe"
          :class="{ 'is-full': fullscreen, 'is-mobile': isMobile }"
          role="dialog"
          aria-modal="true"
          :aria-label="t('fileManager.editFile', { name: activeName })"
        >
          <header class="fe-head">
            <span class="fe-sticker m-sticker" :class="`m-sticker--${visual.tone}`">
              <UIcon :name="visual.icon" />
            </span>
            <span class="fe-title">
              <span class="fe-title__row">
                <span class="fe-title__name" :title="activeName">{{ activeName || '—' }}</span>
                <StatusPill v-if="dirty" variant="warning" :label="t('fileManager.modifiedUnsaved')" />
              </span>
              <span class="fe-title__path m-mono" :title="activePath">{{ activePath || '—' }}</span>
            </span>
            <UButton
              v-if="!isMobile"
              class="fe-head__find"
              color="neutral"
              variant="soft"
              size="sm"
              icon="i-lucide-search"
              :disabled="maskVisible"
              @click="openFind"
            >
              {{ t('fileManager.findReplace') }}
            </UButton>
            <AppIconButton
              :icon="treeOpen ? 'HOutline:ChevronDoubleLeftIcon' : 'HOutline:FolderIcon'"
              :label="treeOpen ? t('fileManager.hideFileTree') : t('fileManager.fileTree')"
              :active="treeOpen"
              :box="isMobile ? 44 : 36"
              :size="18"
              @click="treeOpen = !treeOpen"
            />
            <ActionMenu :items="menuItems" :label="t('fileManager.moreActions')" @select="onMenu" />
            <AppIconButton
              icon="HOutline:XMarkIcon"
              :label="t('fileManager.closeEditor')"
              :box="isMobile ? 44 : 36"
              :size="18"
              @click="close"
            />
          </header>

          <div class="fe-body">
            <aside v-if="treeOpen" class="fe-tree" :aria-label="t('fileManager.fileTree')">
              <FileTree
                ref="treeRef"
                :client="client"
                :root-path="rootPath"
                :active-path="activePath"
                @select="onTreeSelect"
              />
            </aside>
            <div class="fe-main">
              <div ref="monacoEl" class="fe-monaco" />
              <div v-if="maskVisible" class="fe-mask">
                <UIcon name="i-lucide-file-warning" class="fe-mask__ico" />
                <p>{{ activePath ? t('fileManager.cannotEditBinary') : t('fileManager.fileTree') }}</p>
              </div>
            </div>
          </div>

          <footer class="fe-foot">
            <span class="fe-status m-num">
              <span>{{ languageLabel }}</span>
              <span>· UTF-8 ·</span>
              <button type="button" class="fe-status__eol" :title="t('fileManager.eol')" @click="toggleEol">{{ eol }}</button>
              <span>· {{ t('fileManager.cursorAt', { line: cursor.line, col: cursor.col }) }}</span>
              <span v-if="!isMobile">· {{ t('fileManager.totalLines', { n: totalLines }) }}</span>
            </span>
            <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
            <UButton
              color="primary"
              icon="i-lucide-check"
              :loading="saving"
              :disabled="!dirty || saving || isBinary"
              @click="save"
            >
              {{ t('common.save') }}<span v-if="!isMobile" class="fe-kbd m-num">Ctrl S</span>
            </UButton>
          </footer>
        </section>
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

    <!-- 通用确认（丢弃未保存 / 删除） -->
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
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import { monaco } from './monaco'
import FileTree from './FileTree.vue'
import FilePromptDialog from './FilePromptDialog.vue'
import type { FileClient } from './types'

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
const isMobile = computed(() => menuStore.isMobile)

const notifyError = (error: unknown, fallback: string) => fb.error(getRequestErrorMessage(error, fallback))

// ---- 编辑器明暗（自动=跟随界面，亮/暗手动，localStorage 持久）----
type ThemeMode = 'auto' | 'light' | 'dark'
const THEME_KEY = 'fileEditorTheme'
const readThemeMode = (): ThemeMode => {
  try {
    const v = localStorage.getItem(THEME_KEY)
    return v === 'light' || v === 'dark' ? v : 'auto'
  } catch {
    return 'auto'
  }
}
const themeMode = ref<ThemeMode>(readThemeMode())
const resolvedDark = computed(() =>
  themeMode.value === 'auto' ? themeStore.isDarkTheme : themeMode.value === 'dark',
)
const setThemeMode = (mode: ThemeMode) => {
  themeMode.value = mode
  try {
    localStorage.setItem(THEME_KEY, mode)
  } catch {
    /* ignore */
  }
}
watch(resolvedDark, (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'))

const fullscreen = ref(false)
const treeOpen = ref(false)

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
const visual = computed(() => {
  const name = activeName.value
  if (/\.(js|mjs|cjs|ts|tsx|jsx|vue|go|py|rs|java|kt|c|cc|cpp|h|cs|php|rb|sh|bash|ps1|bat|json|ya?ml|toml|xml|html?|css|scss|sql|lua)$/i.test(name)) {
    return { icon: 'i-lucide-file-code', tone: 'a2' }
  }
  if (/\.log$/i.test(name)) return { icon: 'i-lucide-scroll-text', tone: 'neutral' }
  return { icon: 'i-lucide-file-text', tone: 'a2' }
})

const ensureEditor = async () => {
  if (editorInstance || !monacoEl.value) return
  editorInstance = monaco.editor.create(monacoEl.value, {
    value: '',
    language: 'plaintext',
    theme: resolvedDark.value ? 'vs-dark' : 'vs',
    automaticLayout: true,
    fontSize: 13,
    lineHeight: 22,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    tabSize: 2,
    renderWhitespace: 'selection',
    fontFamily: "'JetBrains Mono', 'Cascadia Code', Consolas, 'Courier New', monospace",
    smoothScrolling: true,
    padding: { top: 12, bottom: 12 },
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
  // Ctrl/⌘ + S 保存
  editorInstance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    void save()
  })
}

const openFind = () => {
  if (!editorInstance) return
  editorInstance.focus()
  void editorInstance.getAction('editor.action.startFindReplaceAction')?.run()
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

// ---- 通用确认（丢弃未保存 / 删除）：令牌 FormDialog + Promise 门闸 ----
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
  if (isMobile.value) treeOpen.value = false
  openPath(path, name)
}

// ---- 操作 ----
const save = async () => {
  if (!editorInstance || !dirty.value || isBinary.value || saving.value) return
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

// ---- 「⋯」菜单：刷新 / 下载 / 重命名 / 编辑器明暗 / 全屏 / 删除 ----
const themeIcon = (mode: ThemeMode) => (themeMode.value === mode ? 'i-lucide-check' : mode === 'auto' ? 'i-lucide-monitor' : mode === 'light' ? 'i-lucide-sun' : 'i-lucide-moon')
const menuItems = computed<ActionMenuItem[]>(() => [
  { key: 'refresh', label: t('fileManager.reloadFile'), icon: 'i-lucide-refresh-cw', disabled: !activePath.value },
  { key: 'download', label: t('fileManager.download'), icon: 'i-lucide-download', disabled: !activePath.value },
  { key: 'rename', label: t('fileManager.rename'), icon: 'i-lucide-pencil', disabled: !activePath.value },
  { key: 'find', label: t('fileManager.findReplace'), icon: 'i-lucide-search', hidden: !isMobile.value, disabled: maskVisible.value },
  { key: 'theme:auto', label: t('fileManager.editorThemeAuto'), icon: themeIcon('auto') },
  { key: 'theme:light', label: t('fileManager.editorThemeLight'), icon: themeIcon('light') },
  { key: 'theme:dark', label: t('fileManager.editorThemeDark'), icon: themeIcon('dark') },
  {
    key: 'fullscreen',
    label: fullscreen.value ? t('fileManager.exitFullscreen') : t('fileManager.fullscreen'),
    icon: fullscreen.value ? 'i-lucide-minimize-2' : 'i-lucide-maximize-2',
    hidden: isMobile.value,
  },
  { key: 'delete', label: t('fileManager.delete'), icon: 'i-lucide-trash-2', danger: true, disabled: !activePath.value },
])
const onMenu = (key: string) => {
  if (key === 'refresh') refresh()
  else if (key === 'download') download()
  else if (key === 'rename') openRename()
  else if (key === 'find') openFind()
  else if (key === 'fullscreen') fullscreen.value = !fullscreen.value
  else if (key === 'delete') remove()
  else if (key.startsWith('theme:')) setThemeMode(key.slice(6) as ThemeMode)
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
      fullscreen.value = false
      treeOpen.value = false
      await nextTick()
      await ensureEditor()
      if (props.path) await openPath(props.path)
    } else {
      disposeEditor()
      activePath.value = ''
      activeName.value = ''
      dirty.value = false
      isBinary.value = false
      treeOpen.value = false
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

<style scoped lang="scss">
.fe-overlay {
  position: fixed;
  inset: 0;
  z-index: 1990;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 64px 48px;
  background: color-mix(in srgb, #0b1220 45%, transparent);
}
.fe-overlay.is-full {
  padding: 0;
}
.fe {
  display: flex;
  flex-direction: column;
  width: 1080px;
  max-width: 100%;
  height: 100%;
  max-height: 1060px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-xl, 24px);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-lg);
}
.fe.is-full {
  width: 100%;
  max-height: none;
  border-radius: 0;
}

/* 头部 */
.fe-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 14px 14px 22px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.fe-sticker {
  flex: none;
  width: 40px;
  height: 40px;
  margin-right: 4px;
  border-radius: 13px;
  transform: rotate(-5deg);
}
.fe-sticker :deep(svg),
.fe-sticker :deep(.iconify) {
  width: 18px;
  height: 18px;
}
.m-sticker--neutral {
  background: var(--m-s2);
  color: var(--m-ink2);
}
.fe-title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.fe-title__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.fe-title__name {
  min-width: 0;
  font-size: 1.0625rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fe-title__path {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fe-head__find {
  border-radius: 999px;
  font-weight: 700;
}

/* 主体 */
.fe-body {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
}
.fe-tree {
  flex: 0 0 240px;
  min-height: 0;
  overflow-y: auto;
  padding: 8px;
  border-right: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  scrollbar-width: thin;
  --ft-active-bg: var(--m-pSoft);
  --ft-active-fg: var(--m-pInk);
  --ft-hover: var(--m-s2);
  --ft-fg: var(--el-text-color-secondary);
}
.fe-tree :deep(.ftn-row) {
  border-radius: 10px;
}
.fe-tree :deep(.ftn-icon.is-folder) {
  color: var(--m-a);
}
.fe-main {
  position: relative;
  flex: 1;
  min-width: 0;
  background: var(--m-s2);
}
.fe-monaco {
  position: absolute;
  inset: 0;
}
.fe-mask {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
}
.fe-mask__ico {
  width: 32px;
  height: 32px;
}

/* 底部 */
.fe-foot {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px 12px 22px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.fe-foot :deep(button) {
  min-height: 40px;
  border-radius: 999px;
  font-weight: 700;
}
.fe-status {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
}
.fe-status__eol {
  min-height: 28px !important;
  padding: 0 6px;
  border: none;
  border-radius: 8px !important;
  background: transparent;
  color: inherit;
  font: inherit;
  font-weight: 700 !important;
  cursor: pointer;
}
.fe-status__eol:hover {
  background: var(--m-s2);
  color: var(--el-text-color-primary);
}
.fe-kbd {
  margin-left: 6px;
  font-size: 0.75rem;
  opacity: 0.8;
}
.fe-confirm {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}

/* 手机：全屏弹窗；文件树覆盖在编辑区上 */
.fe.is-mobile {
  width: 100%;
  border-radius: 0;
}
.fe.is-mobile .fe-head {
  gap: 2px;
  padding: 10px 6px 10px 12px;
}
.fe.is-mobile .fe-sticker {
  width: 36px;
  height: 36px;
  border-radius: 12px;
}
.fe.is-mobile .fe-tree {
  position: absolute;
  inset: 0 auto 0 0;
  z-index: 3;
  width: min(300px, 84vw);
  box-shadow: var(--app-shadow-lg);
}
.fe.is-mobile .fe-foot {
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 12px calc(10px + env(safe-area-inset-bottom));
}
.fe.is-mobile .fe-status {
  flex-basis: 100%;
}
.fe.is-mobile .fe-foot :deep(button) {
  flex: 1;
  justify-content: center;
  min-height: 44px;
}
@media (width <= 768px) {
  .fe-overlay {
    padding: 0;
  }
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
