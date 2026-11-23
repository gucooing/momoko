<!-- 标签页里打开的文件：用项目里的 Monaco 编辑，Ctrl+S（或头部的按钮）保存。
     Markdown / HTML / SVG 可以在「源码 / 并排 / 预览」之间一键切换，预览跟着没保存的内容实时变；图片直接显示。
     Markdown 里相对路径的图片从这个目录里取，链接到别的文件在标签页里打开。
     HTML 按目录当静态站点加载（后端的预览站点，见 preview.ts）：页面互相跳、样式表、脚本、图片、fetch 都按真实路径取；
     编辑中没保存时放进 srcdoc，开头插 <base> 指向同一个地方。头部「新窗口打开」全屏看。
     预览放在隔离的 iframe 里：Markdown 不能跑脚本，HTML 能跑脚本但碰不到 Momoko 页面。
     这个文件有未提交改动时，头部带 +/− 和「查看改动」。会话里按会话目录读写；新会话页还没有会话，按工作区读写（给 workspaceId）。
     没保存的改动切走标签时存在 fileState 里，切回来接着改；保存时发现文件在打开后被别人（Agent）改过，先问是覆盖还是重新读。
     选中一段后右键（或 Ctrl+L、头部的按钮）「添加到对话」：这段连同路径和行号加进输入框的附件。 -->
<template>
  <div ref="rootEl" class="agc-file" @keydown="onKey">
    <div class="agc-file__head">
      <!-- HTML 预览像个小浏览器：点链接跳到别的页面后标签跟着换，可以后退、前进、刷新 -->
      <template v-if="kind === 'html' && showPreview">
        <button type="button" class="ag-ib ag-ib--sm" :disabled="histPos <= 0" :aria-label="t('agent.code.file.back')" :title="t('agent.code.file.back')" @click="goHist(-1)">
          <UIcon name="i-lucide-arrow-left" />
        </button>
        <button
          type="button"
          class="ag-ib ag-ib--sm"
          :disabled="histPos >= hist.length - 1"
          :aria-label="t('agent.code.file.forward')"
          :title="t('agent.code.file.forward')"
          @click="goHist(1)"
        >
          <UIcon name="i-lucide-arrow-right" />
        </button>
        <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.file.reload')" :title="t('agent.code.file.reload')" @click="reloadFrame">
          <UIcon name="i-lucide-rotate-cw" />
        </button>
      </template>
      <span class="ag-mono ag-ellipsis ag-muted agc-flex" :title="fullPath">{{ fullPath }}</span>
      <button v-if="selLines && showEditor" type="button" class="ag-btn ag-btn--sm" :title="t('agent.code.file.attachTip')" @click="attach">
        <UIcon name="i-lucide-message-square-plus" />{{ t('agent.code.file.attachLines', { lines: selLines }) }}
      </button>
      <template v-if="change">
        <span class="agc-plus ag-num">+{{ change.added }}</span>
        <span class="agc-minus ag-num">−{{ change.deleted }}</span>
        <button type="button" class="ag-btn ag-btn--sm" @click="emit('showDiff', cur)">{{ t('agent.code.file.showDiff') }}</button>
      </template>
      <button
        v-if="editable"
        type="button"
        class="ag-btn ag-btn--sm agc-file__save"
        :class="{ 'is-dirty': dirty }"
        :disabled="!dirty || saving"
        :title="t('agent.code.file.saveTip')"
        @click="save()"
      >
        <UIcon :name="saving ? 'i-lucide-loader-circle' : dirty ? 'i-lucide-save' : 'i-lucide-check'" :class="{ 'ag-spin': saving }" />
        <span class="agc-file__lbl">{{ dirty ? t('agent.code.file.save') : t('agent.code.file.saved') }}</span>
      </button>
      <button
        v-if="kind === 'html' && editable"
        type="button"
        class="ag-btn ag-btn--sm"
        :disabled="!fileUrl"
        :title="dirty ? t('agent.code.file.openWindowDirty') : t('agent.code.file.openWindowTip')"
        @click="openWindow"
      >
        <UIcon name="i-lucide-external-link" /><span class="agc-file__lbl">{{ t('agent.code.file.openWindow') }}</span>
      </button>
      <div v-if="renderable && editable" class="ag-seg agc-file__modes" role="radiogroup" :aria-label="t('agent.code.file.modeLabel')">
        <button
          v-for="m in MODES"
          :key="m.key"
          type="button"
          role="radio"
          :aria-checked="mode === m.key"
          :class="{ 'is-on': mode === m.key }"
          :title="t(`agent.code.file.modeTip.${m.key}`)"
          @click="mode = m.key"
        >
          <UIcon :name="m.icon" /><span class="agc-file__lbl">{{ t(`agent.code.file.mode.${m.key}`) }}</span>
        </button>
      </div>
      <span class="ag-tag">{{ langLabel }}</span>
    </div>

    <div v-if="conflict" class="agc-file__bar" role="alert">
      <UIcon name="i-lucide-triangle-alert" />
      <span class="agc-flex">{{ t('agent.code.file.conflict') }}</span>
      <button type="button" class="ag-btn ag-btn--sm" @click="reloadFromDisk()">{{ t('agent.code.file.conflictReload') }}</button>
      <button type="button" class="ag-btn ag-btn--sm ag-btn--stop" @click="save(true)">{{ t('agent.code.file.conflictOverwrite') }}</button>
    </div>

    <div v-if="note" class="agc-file__note">{{ note }}</div>
    <div v-else-if="kind === 'image'" class="agc-file__img">
      <img v-if="imgUrl" :src="imgUrl" :alt="cur" @load="onImgLoad" />
      <span class="ag-muted agc-small ag-num">{{ imgInfo }}</span>
    </div>
    <div v-show="!note && kind !== 'image'" class="agc-file__body" :class="`is-${renderable ? mode : 'source'}`">
      <div v-show="showEditor" ref="host" class="agc-file__editor" :aria-label="t('agent.code.file.content', { path: cur })" />
      <div v-if="showPreview" class="agc-file__preview">
        <div v-if="kind === 'svg'" class="agc-file__img"><img :src="svgUrl" :alt="cur" /></div>
        <iframe
          v-else-if="kind === 'md'"
          ref="mdFrame"
          :title="t('agent.code.file.preview')"
          sandbox="allow-same-origin allow-popups allow-popups-to-escape-sandbox"
          :srcdoc="MD_SHELL"
          @load="paintMd"
        />
        <!-- 没改动：直接加载预览站点里的页面；换 src 不重建 iframe，页面里存的东西（window.name）跳来跳去都还在 -->
        <iframe v-else-if="!dirty && frameSrc" ref="htmlFrame" :title="t('agent.code.file.preview')" :sandbox="HTML_SANDBOX" :src="frameSrc" />
        <iframe v-else ref="htmlFrame" :title="t('agent.code.file.preview')" :sandbox="HTML_SANDBOX" :srcdoc="htmlDoc" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn, useEventListener } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { getAgentFileRaw, readAgentFile, saveAgentFile, type AgentFileScope, type AgentModTime } from '@/api/agent'
import { monaco } from '@/components/file/monaco'
import type { AgentChangedFile } from '@/types/v1/agent'
import { useThemeStore } from '@/stores/theme'
import { detectMonacoLanguage } from '@/utils/file'
import { fmtBytes } from '@/views/agent/utils/format'
import type { DraftSnippet } from './Composer.vue'
import { fileKey, fileScope, fileState, type FileMode } from './fileState'
import { encodePath, previewRoot, withBase } from './preview'

/** 外面给每个标签一个固定的 key：会话、工作区不变；路径只在 HTML 预览里跳页面时由这里自己换（emit navigate） */
const props = defineProps<{
  sessionId?: string
  workspaceId?: string
  path: string
  cwd: string
  change?: AgentChangedFile | null
  /** 每一轮结束加一：没改过的文件跟着重读（Agent 可能改了它） */
  tick?: number
}>()
const emit = defineEmits<{
  showDiff: [path: string]
  attach: [snippet: DraftSnippet]
  open: [path: string]
  /** HTML 预览里跳到了别的页面（或者后退、前进）：这个标签换成那个文件 */
  navigate: [from: string, to: string]
}>()
const { t } = useI18n()
const themeStore = useThemeStore()

const scope: AgentFileScope = props.workspaceId ? { workspaceId: props.workspaceId } : { sessionId: props.sessionId || '' }
const scopeKey = fileScope(props.sessionId, props.workspaceId)
/** 现在显示的文件：HTML 预览里点链接跳到别的页面时跟着换（只在 HTML 之间跳，类型不变） */
const cur = ref(props.path)
let key = fileKey(scopeKey, cur.value)

const kind = kindOf(props.path)
const renderable = kind === 'md' || kind === 'html' || kind === 'svg'
const MODES: { key: FileMode; icon: string }[] = [
  { key: 'source', icon: 'i-lucide-code' },
  { key: 'split', icon: 'i-lucide-columns-2' },
  { key: 'preview', icon: 'i-lucide-eye' },
]
const mode = ref<FileMode>(fileState.mode(key) ?? (renderable ? 'preview' : 'source'))
watch(mode, (m) => {
  fileState.setMode(key, m)
  if (m !== 'source') void nextTick(refreshPreview)
})

const rootEl = ref<HTMLElement | null>(null)
const host = ref<HTMLElement | null>(null)
const note = ref('')
/** 文本读出来了，能编辑 */
const editable = ref(false)
const dirty = ref(false)
const saving = ref(false)
const conflict = ref(false)
const showEditor = computed(() => !renderable || mode.value !== 'preview')
const showPreview = computed(() => renderable && editable.value && mode.value !== 'source')
const language = detectMonacoLanguage(props.path)
const langLabel = computed(() => (language === 'plaintext' ? t('agent.code.file.plain') : language))
const fullPath = computed(() => `${props.cwd.replace(/[\\/]+$/, '')}/${cur.value}`)

let editor: monaco.editor.IStandaloneCodeEditor | null = null
let model: monaco.editor.ITextModel | null = null
/** 磁盘上的内容和修改时间（打开或上次保存时） */
let saved = ''
let mod: AgentModTime = 0
/** 和磁盘一致时编辑器的版本号：撤销回到这里也算没改过 */
let savedVer = 0

const updateDirty = () => {
  const d = !!model && model.getAlternativeVersionId() !== savedVer
  if (d === dirty.value) return
  dirty.value = d
  fileState.setDirty(key, d)
}

// 切走标签（卸下）或换到别的文件时，把没保存的改动、光标位置存起来
const register = () =>
  fileState.attach(
    key,
    () => (model && editable.value ? { content: model.getValue(), saved, mod } : null),
    () => editor?.saveViewState() ?? null,
  )
let detach = register()

// —— 选中一段添加到对话 ——
/** 选中的行（「22-26」），没选中为空 */
const selLines = ref('')
/** 选中范围的起止行：结尾停在下一行行首时不算那一行 */
const selRange = (sel: monaco.Selection) => {
  let end = sel.endLineNumber
  if (sel.endColumn === 1 && end > sel.startLineNumber) end--
  return { start: sel.startLineNumber, end }
}
const attach = () => {
  const sel = editor?.getSelection()
  if (!sel || !model || sel.isEmpty()) return
  const text = model.getValueInRange(sel)
  if (!text.trim()) return
  const { start, end } = selRange(sel)
  emit('attach', { path: cur.value, startLine: start, endLine: end, text })
}

const createEditor = (value: string) => {
  editor = monaco.editor.create(host.value!, {
    value,
    language,
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: 12.5,
    scrollBeyondLastLine: false,
    wordWrap: kind === 'md' ? 'on' : 'off',
    theme: themeStore.isDarkTheme ? 'vs-dark' : 'vs',
  })
  model = editor.getModel()
  editor.addAction({
    id: 'momoko.agent.attachSelection',
    label: t('agent.code.file.attach'),
    contextMenuGroupId: 'navigation',
    contextMenuOrder: 0,
    precondition: 'editorHasSelection',
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyL],
    run: attach,
  })
  editor.onDidChangeCursorSelection(({ selection }) => {
    if (selection.isEmpty()) {
      selLines.value = ''
      return
    }
    const { start, end } = selRange(selection)
    selLines.value = start === end ? String(start) : `${start}-${end}`
  })
  editor.onDidChangeModelContent(() => {
    updateDirty()
    if (showPreview.value) schedulePreview()
  })
}

// —— 读 ——
let loadSeq = 0
/** 把 path 的内容放进编辑器（第一次建编辑器，之后换内容）；切走前没保存的改动从 fileState 接着用。
 *  quiet：换文件时不显示「读取中」（不然预览会被藏起来一下） */
const loadText = async (path: string, quiet = false) => {
  const seq = ++loadSeq
  const draft = fileState.draft(key)
  if (!quiet) note.value = t('agent.common.loading')
  let content = ''
  if (draft) {
    // 切走前没保存的改动：不用重读，接着改（保存时文件被改过会提示）
    saved = draft.saved
    mod = draft.mod
  } else {
    try {
      const { data } = await readAgentFile(scope, path)
      if (seq !== loadSeq) return
      if (data.binary) {
        note.value = t('agent.code.file.binary')
        return
      }
      if (data.tooLarge) {
        note.value = t('agent.code.file.tooLarge', { size: fmtBytes(data.size) })
        return
      }
      content = data.content
      saved = data.content
      mod = data.modTime
    } catch {
      if (seq === loadSeq) note.value = t('agent.code.file.failed')
      return
    }
  }
  note.value = ''
  await nextTick()
  if (!host.value || seq !== loadSeq) return
  const base = draft ? draft.saved : content
  if (!editor) createEditor(base)
  else model!.setValue(base)
  savedVer = model!.getAlternativeVersionId()
  if (draft) {
    // 作为一次编辑放上去：撤销能回到磁盘上的版本
    model!.pushEditOperations([], [{ range: model!.getFullModelRange(), text: draft.content }], () => null)
  }
  const view = fileState.view(key)
  if (view) editor!.restoreViewState(view as monaco.editor.ICodeEditorViewState)
  editable.value = true
  updateDirty()
  void refreshPreview()
}
const load = async () => {
  if (kind === 'image') return loadImage()
  if (kind === 'html') {
    previewRoot(scope).then(
      (root) => {
        previewBase.value = root
        setFrame(cur.value)
        void refreshPreview()
      },
      () => {
        /* 拿不到就只用 srcdoc 预览 */
      },
    )
  }
  await loadText(cur.value)
}

/** 换到另一个文件（HTML 预览里跳了页面、后退、前进）：存下这个文件的改动，读那个文件，标签跟着换 */
const switchTo = async (path: string) => {
  if (path === cur.value) return
  const from = cur.value
  detach()
  cur.value = path
  key = fileKey(scopeKey, path)
  detach = register()
  // dirty 直接清掉：上一个文件的「没保存」标记留在 fileState 里，不去动它
  dirty.value = false
  conflict.value = false
  selLines.value = ''
  raw.clear()
  emit('navigate', from, path)
  await loadText(path, true)
}

// —— 预览里的历史：后退、前进 ——
const hist = ref<string[]>([props.path])
const histPos = ref(0)
/** 页面在 iframe 里自己跳到了 path（点了链接）：记进历史，标签换成它；iframe 已经在那个页面上了，不用重新加载 */
const onFrameNav = (path: string) => {
  if (path === cur.value) return
  hist.value = [...hist.value.slice(0, histPos.value + 1), path]
  histPos.value = hist.value.length - 1
  // 编辑中的页面放在 srcdoc 里：换过去之后改用直接加载
  const fromDoc = dirty.value
  void switchTo(path).then(() => {
    if (fromDoc) setFrame(path)
  })
}
const goHist = async (d: number) => {
  const path = hist.value[histPos.value + d]
  if (!path) return
  histPos.value += d
  await switchTo(path)
  setFrame(path)
}
const reloadFrame = () => setFrame(cur.value)

/** 按磁盘上的重读：冲突时选「重新读取」（丢掉自己的改动），或者一轮结束后没改过的文件（quiet） */
const reloadFromDisk = async (quiet = false) => {
  if (!model || !editor) return
  try {
    const { data } = await readAgentFile(scope, cur.value)
    if (quiet && (dirty.value || String(data.modTime) === String(mod))) return
    if (data.binary || data.tooLarge) {
      if (!quiet) feedback.warning(t('agent.code.file.notText'))
      return
    }
    saved = data.content
    mod = data.modTime
    conflict.value = false
    if (model.getValue() !== data.content) {
      const view = editor.saveViewState()
      model.setValue(data.content)
      if (view) editor.restoreViewState(view)
    }
    savedVer = model.getAlternativeVersionId()
    raw.clear()
    if (kind === 'html') setFrame(cur.value)
    updateDirty()
    void refreshPreview()
  } catch {
    /* shown */
  }
}

// —— 保存 ——
const save = async (force = false) => {
  if (!model || saving.value || (!dirty.value && !force)) return
  saving.value = true
  const content = model.getValue()
  const ver = model.getAlternativeVersionId()
  try {
    const { data } = await saveAgentFile(scope, { path: cur.value, content, baseModTime: mod, force })
    if (data.conflict) {
      conflict.value = true
      return
    }
    saved = content
    mod = data.modTime
    savedVer = ver
    conflict.value = false
    if (kind === 'html') setFrame(cur.value)
    updateDirty()
  } catch {
    /* shown */
  } finally {
    saving.value = false
  }
}
const onKey = (e: KeyboardEvent) => {
  if ((e.ctrlKey || e.metaKey) && !e.altKey && e.key.toLowerCase() === 's') {
    e.preventDefault()
    void save()
  }
}

// —— 图片 ——
const imgUrl = ref('')
const imgInfo = ref('')
let imgSize = 0
const loadImage = async () => {
  note.value = t('agent.common.loading')
  try {
    const { data } = await getAgentFileRaw(scope, cur.value)
    if (!data.data) throw new Error('missing')
    imgSize = Math.floor((data.data.length * 3) / 4)
    imgUrl.value = `data:${data.mime || 'image/png'};base64,${data.data}`
    note.value = ''
  } catch {
    note.value = t('agent.code.file.failed')
  }
}
const onImgLoad = (e: Event) => {
  const img = e.target as HTMLImageElement
  imgInfo.value = `${img.naturalWidth} × ${img.naturalHeight} · ${fmtBytes(imgSize)}`
}

// —— 预览 ——
/** 目录里取过的文件（图片、样式表、脚本）：路径 → base64 和类型，取不到为 null */
const raw = new Map<string, Promise<{ mime: string; data: string } | null>>()
const fetchRaw = (path: string) => {
  let hit = raw.get(path)
  if (!hit) {
    hit = getAgentFileRaw(scope, path).then(
      ({ data }) => (data.data ? { mime: data.mime, data: data.data } : null),
      () => null,
    )
    raw.set(path, hit)
  }
  return hit
}
/** 预览里的相对地址换成目录里的路径；外部地址、页内锚点、跳出目录的返回 null */
const resolveRel = (src: string): string | null => {
  if (!src || src.startsWith('#') || src.startsWith('//') || /^[a-z][a-z0-9+.-]*:/i.test(src)) return null
  let p = src.split(/[?#]/)[0] || ''
  try {
    p = decodeURIComponent(p)
  } catch {
    /* 原样 */
  }
  const parts = p.startsWith('/') ? [] : cur.value.split('/').slice(0, -1)
  for (const seg of p.split('/')) {
    if (!seg || seg === '.') continue
    if (seg === '..') {
      if (!parts.length) return null
      parts.pop()
    } else parts.push(seg)
  }
  return parts.join('/') || null
}
const dataUrl = (r: { mime: string; data: string }) => `data:${r.mime || 'application/octet-stream'};base64,${r.data}`

const svgUrl = ref('')
const htmlDoc = ref('')
/** HTML 预览站点的根地址（见 preview.ts），拿到之前为空 */
const previewBase = ref('')
/** 这个文件在预览站点里的地址（新窗口打开用） */
const fileUrl = computed(() => (previewBase.value ? `${previewBase.value}${encodePath(cur.value)}` : ''))
/** 预览 iframe 加载的地址：只在第一次、保存、重读、后退前进、刷新时换（带个序号保证重新加载）；
 *  页面自己点链接跳走时不动它 */
const frameSrc = ref('')
let frameSeq = 0
const setFrame = (path: string) => {
  if (previewBase.value) frameSrc.value = `${previewBase.value}${encodePath(path)}?v=${++frameSeq}`
}
const htmlFrame = ref<HTMLIFrameElement | null>(null)
// 预览页面开头的脚本（后端注入）报告它现在在哪个地址：是这个目录里的另一个 HTML 就跟过去
const PREVIEW_PATH = /\/public\/agent\/preview\/[^/]+\/[^/]+\/(.*)$/
useEventListener(window, 'message', (e: MessageEvent) => {
  const data = e.data as { momokoPreview?: number; href?: string } | null
  if (kind !== 'html' || !data || typeof data !== 'object' || !data.momokoPreview) return
  if (!htmlFrame.value || e.source !== htmlFrame.value.contentWindow) return
  let url: URL
  try {
    url = new URL(String(data.href))
  } catch {
    return
  }
  const m = PREVIEW_PATH.exec(url.pathname)
  if (!m) return
  let path = m[1] || ''
  try {
    path = decodeURIComponent(path)
  } catch {
    return
  }
  if (!path || path.endsWith('/')) path += 'index.html'
  if (kindOf(path) === 'html') onFrameNav(path)
})
const mdHtml = ref('')
const mdFrame = ref<HTMLIFrameElement | null>(null)
let previewSeq = 0

const refreshPreview = async () => {
  if (!showPreview.value || !model) return
  const text = model.getValue()
  const seq = ++previewSeq
  if (kind === 'svg') {
    svgUrl.value = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(text)}`
    return
  }
  const html = kind === 'md' ? await renderMd(text) : renderHtml(text)
  if (seq !== previewSeq) return
  if (kind === 'md') {
    mdHtml.value = html
    paintMd()
  } else htmlDoc.value = html
}
const debounced = { md: useDebounceFn(refreshPreview, 200), html: useDebounceFn(refreshPreview, 600), svg: useDebounceFn(refreshPreview, 150) }
const schedulePreview = () => {
  if (kind === 'md' || kind === 'html' || kind === 'svg') void debounced[kind]()
}

/** 相对路径的图片换成取回来的内容（预览里没法按路径去取） */
const inlineImages = async (doc: Document) => {
  const jobs: Promise<void>[] = []
  for (const img of [...doc.querySelectorAll('img[src]')].slice(0, 80)) {
    const path = resolveRel(img.getAttribute('src') || '')
    if (!path) continue
    jobs.push(
      fetchRaw(path).then((r) => {
        if (r) img.setAttribute('src', dataUrl(r))
      }),
    )
  }
  await Promise.all(jobs)
}
const renderMd = async (text: string) => {
  const doc = new DOMParser().parseFromString(`<body>${md.render(text)}</body>`, 'text/html')
  await inlineImages(doc)
  return doc.body.innerHTML
}
/** 编辑中的 HTML：插一个 <base> 指向它在预览站点里的位置，相对路径的链接和资源照样能取到 */
const renderHtml = (text: string) => (previewBase.value ? withBase(text, previewBase.value, cur.value) : text)

/** 在新窗口里打开（全屏看，链接、脚本都能用）；打开的是磁盘上已保存的版本 */
const openWindow = () => {
  if (fileUrl.value) window.open(fileUrl.value, '_blank', 'noopener')
}

/** 把 Markdown 写进 iframe：只换 body，滚动位置不跳 */
const paintMd = () => {
  const doc = mdFrame.value?.contentDocument
  if (!doc?.body || !rootEl.value) return
  let style = doc.getElementById('md-style')
  if (!style) {
    style = doc.createElement('style')
    style.id = 'md-style'
    doc.head.appendChild(style)
    doc.addEventListener('click', onMdClick)
  }
  style.textContent = mdCss(rootEl.value, themeStore.isDarkTheme)
  doc.body.innerHTML = mdHtml.value
}
// 预览里点链接：页内锚点就地滚过去，目录里的文件在标签页里打开，外部地址在新窗口打开（链接本身带 target=_blank）
const onMdClick = (e: MouseEvent) => {
  const a = (e.target as Element | null)?.closest?.('a')
  const href = a?.getAttribute('href') || ''
  if (!a || !href) return
  if (href.startsWith('#')) {
    e.preventDefault()
    const id = decodeURIComponent(href.slice(1))
    mdFrame.value?.contentDocument?.getElementById(id)?.scrollIntoView({ block: 'start' })
    return
  }
  const path = resolveRel(href)
  if (path) {
    e.preventDefault()
    emit('open', path)
  }
}

watch(
  () => themeStore.isDarkTheme,
  (dark) => {
    monaco.editor.setTheme(dark ? 'vs-dark' : 'vs')
    if (kind === 'md') void nextTick(paintMd)
  },
)
watch(
  () => props.tick,
  () => {
    if (editable.value && !dirty.value && !conflict.value && !saving.value) void reloadFromDisk(true)
  },
)
onMounted(() => void load())
onBeforeUnmount(() => {
  detach()
  mdFrame.value?.contentDocument?.removeEventListener('click', onMdClick)
  editor?.getModel()?.dispose()
  editor?.dispose()
  editor = null
  model = null
})
</script>

<script lang="ts">
import MarkdownIt from 'markdown-it'

type Kind = 'md' | 'html' | 'svg' | 'image' | 'text'
function kindOf(path: string): Kind {
  const ext = /\.([^./]+)$/.exec(path)?.[1]?.toLowerCase() ?? ''
  if (['md', 'markdown', 'mdx'].includes(ext)) return 'md'
  if (['html', 'htm', 'xhtml'].includes(ext)) return 'html'
  if (ext === 'svg') return 'svg'
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'ico', 'avif'].includes(ext)) return 'image'
  return 'text'
}

// Markdown 预览：允许里面写 HTML（README 里常见），放在不能跑脚本的 iframe 里所以没关系
const md = new MarkdownIt({ html: true, linkify: true })
const defaultLink = md.renderer.rules.link_open ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const href = tokens[idx]?.attrGet('href') || ''
  if (/^(https?:|mailto:)/i.test(href)) {
    tokens[idx]?.attrSet('target', '_blank')
    tokens[idx]?.attrSet('rel', 'noopener noreferrer')
  }
  return defaultLink(tokens, idx, options, env, self)
}
// 标题带上 id（和 GitHub 一样的规则），页内锚点 #xxx 能跳过去
md.core.ruler.push('heading_ids', (state) => {
  const seen = new Map<string, number>()
  state.tokens.forEach((tok, i) => {
    if (tok.type !== 'heading_open') return
    const base =
      (state.tokens[i + 1]?.content || '')
        .trim()
        .toLowerCase()
        .replace(/[^\p{L}\p{N}\s_-]/gu, '')
        .replace(/\s/g, '-') || 'section'
    const n = seen.get(base) ?? 0
    seen.set(base, n + 1)
    tok.attrSet('id', n ? `${base}-${n}` : base)
  })
})

/** HTML 预览的沙箱：能跑脚本、提交表单、弹窗、下载，没有 allow-same-origin，碰不到 Momoko 页面 */
const HTML_SANDBOX = 'allow-scripts allow-popups allow-forms allow-modals allow-downloads'

const MD_SHELL = '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body></body></html>'

/** iframe 里 Markdown 的样式：颜色取页面当前的主题 */
function mdCss(el: HTMLElement, dark: boolean) {
  const cs = getComputedStyle(el)
  const v = (name: string, def: string) => cs.getPropertyValue(name).trim() || def
  const ink = v('--m-ink', dark ? '#eee' : '#222')
  const ink2 = v('--m-ink2', dark ? '#aaa' : '#666')
  const line = v('--m-line', dark ? '#333' : '#e5e5e5')
  const bg = v('--m-s', dark ? '#1e1e1e' : '#fff')
  const soft = v('--m-s2', dark ? '#262626' : '#f6f6f6')
  const link = v('--m-pInk', v('--m-p', '#d6245f'))
  const mono = v('--app-font-mono', "ui-monospace, Consolas, 'Courier New', monospace")
  return `
:root{color-scheme:${dark ? 'dark' : 'light'}}
html,body{margin:0;background:${bg};color:${ink}}
body{box-sizing:border-box;max-width:980px;margin:0 auto;padding:24px 32px 48px;font:14px/1.75 'Segoe UI','Microsoft YaHei UI','Microsoft YaHei','PingFang SC','Noto Sans SC',system-ui,sans-serif;overflow-wrap:break-word}
body>:first-child{margin-top:0}
h1,h2,h3,h4,h5,h6{margin:1.4em 0 .6em;line-height:1.3;font-weight:650}
h1{font-size:1.9em;padding-bottom:.3em;border-bottom:1px solid ${line}}
h2{font-size:1.45em;padding-bottom:.25em;border-bottom:1px solid ${line}}
h3{font-size:1.2em}h4{font-size:1.05em}h5,h6{font-size:.95em;color:${ink2}}
p,ul,ol,blockquote,table,pre,details{margin:0 0 1em}
ul,ol{padding-left:1.8em}li+li{margin-top:.25em}li>p{margin:0}
a{color:${link};text-decoration:none}a:hover{text-decoration:underline}
code{font-family:${mono};font-size:.88em;padding:.15em .4em;border-radius:5px;background:${soft}}
pre{padding:12px 14px;border-radius:8px;background:${soft};overflow:auto;line-height:1.55}
pre code{padding:0;background:none;font-size:.86em}
blockquote{margin-left:0;padding:.1em 1em;border-left:4px solid ${line};color:${ink2}}
table{border-collapse:collapse;display:block;width:max-content;max-width:100%;overflow:auto}
th,td{padding:6px 12px;border:1px solid ${line}}th{background:${soft};font-weight:600}
img{max-width:100%;height:auto}
hr{border:0;border-top:1px solid ${line};margin:1.6em 0}
kbd{font-family:${mono};font-size:.85em;padding:.1em .4em;border:1px solid ${line};border-bottom-width:2px;border-radius:4px}
`
}
</script>
