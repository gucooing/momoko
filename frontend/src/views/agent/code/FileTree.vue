<!-- 资源管理器：按目录懒加载的文件树，用起来像 VS Code 左边那棵。
     点文件打开（open），点文件夹展开收起；右键菜单里新建文件 / 文件夹、改名、删除、复制路径、添加到对话；
     拖到别的文件夹上是挪位置，拖到输入框上是引用；键盘上下左右选、回车打开、F2 改名、Delete 删除。
     有未提交改动的带 M / A。会话里操作会话目录（scope.sessionId），新会话页按工作区（scope.workspaceId）。 -->
<template>
  <div class="agc-ex">
    <div class="agc-ex__bar">
      <span class="ag-lbl ag-ellipsis agc-flex" :title="cwd">{{ rootName }}</span>
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.files.newFile')" :title="t('agent.code.files.newFile')" @click="startNew(false)">
        <UIcon name="i-lucide-file-plus" />
      </button>
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.files.newDir')" :title="t('agent.code.files.newDir')" @click="startNew(true)">
        <UIcon name="i-lucide-folder-plus" />
      </button>
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.files.refresh')" :title="t('agent.code.files.refresh')" @click="refresh">
        <UIcon name="i-lucide-refresh-cw" :class="{ 'ag-spin': refreshing }" />
      </button>
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.files.collapse')" :title="t('agent.code.files.collapse')" :disabled="!expanded.size" @click="collapseAll">
        <UIcon name="i-lucide-copy-minus" />
      </button>
    </div>
    <div
      ref="listEl"
      class="agc-ex__list ag-noscroll"
      :class="{ 'is-drop': dropDir === '' }"
      role="tree"
      tabindex="0"
      :aria-label="rootName"
      @keydown="onKey"
      @contextmenu.prevent="openMenu($event, null)"
      @dragover="onDragOver($event, null)"
      @dragleave="onDragLeave"
      @drop="onDrop"
    >
      <template v-for="n in rows" :key="n.key">
        <div v-if="n.input" class="agc-ex__row is-editing">
          <span v-for="g in n.depth" :key="g" class="agc-ex__guide" />
          <span class="agc-ex__chev" />
          <UIcon :name="edit.isDir ? 'i-lucide-folder' : 'i-lucide-file'" :class="edit.isDir ? 'agc-dir' : 'ag-muted'" />
          <input
            :ref="setInput"
            v-model="edit.name"
            class="agc-ex__input ag-mono"
            :placeholder="edit.isDir ? t('agent.code.files.newDirHint') : t('agent.code.files.newFileHint')"
            spellcheck="false"
            @keydown.stop="onInputKey"
            @blur="commitEdit"
          />
        </div>
        <div
          v-else
          class="agc-ex__row"
          :class="{
            'is-sel': n.path === selected,
            'is-active': n.path === active,
            'is-drop': dropDir && isUnder(n.path, dropDir),
            'is-dragging': n.path === drag,
          }"
         
          role="treeitem"
          :aria-level="n.depth + 1"
          :aria-expanded="n.isDir ? expanded.has(n.path) : undefined"
          :aria-selected="n.path === selected"
          :data-path="n.path"
          :title="tip ? tip(n.path) : n.path"
          :draggable="!(edit.mode === 'rename' && edit.path === n.path)"
          @click="onRow(n)"
          @contextmenu.prevent.stop="openMenu($event, n)"
          @dragstart="onDragStart($event, n)"
          @dragover.stop="onDragOver($event, n)"
          @dragend="endDrag"
        >
          <span v-for="g in n.depth" :key="g" class="agc-ex__guide" />
          <span class="agc-ex__chev">
            <UIcon v-if="n.isDir" :name="expanded.has(n.path) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" />
          </span>
          <UIcon :name="iconOf(n)" :class="n.isDir ? 'agc-dir' : 'ag-muted'" />
          <input
            v-if="edit.mode === 'rename' && edit.path === n.path"
            :ref="setInput"
            v-model="edit.name"
            class="agc-ex__input ag-mono"
            spellcheck="false"
            @click.stop
            @keydown.stop="onInputKey"
            @blur="commitEdit"
          />
          <template v-else>
            <span class="ag-mono ag-ellipsis agc-flex agc-ex__name" :class="n.change ? (n.change === 'A' ? 'agc-ok' : 'agc-warn') : ''">{{ n.name }}</span>
            <span v-if="n.change" class="ag-mono agc-fr__k" :class="n.change === 'A' ? 'agc-ok' : 'agc-warn'">{{ n.change }}</span>
          </template>
        </div>
      </template>
      <div v-if="loaded && !rows.length" class="agc-right__note">{{ empty }}</div>
    </div>

    <Teleport to="body">
      <div
        v-if="menu"
        ref="menuEl"
        class="ag ag-menu agc-ctx"
        role="menu"
        :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
        @contextmenu.prevent
        @keydown.esc="closeMenu"
      >
        <template v-if="menu.row && !menu.row.isDir">
          <button type="button" class="ag-mi" role="menuitem" @click="act((r) => emit('open', r!.path))">
            <UIcon name="i-lucide-file-symlink" />{{ t('agent.code.files.open') }}
          </button>
          <div class="ag-mi-sep" />
        </template>
        <button type="button" class="ag-mi" role="menuitem" @click="startNew(false, menuDir)">
          <UIcon name="i-lucide-file-plus" />{{ t('agent.code.files.newFile') }}
        </button>
        <button type="button" class="ag-mi" role="menuitem" @click="startNew(true, menuDir)">
          <UIcon name="i-lucide-folder-plus" />{{ t('agent.code.files.newDir') }}
        </button>
        <template v-if="menu.row">
          <div class="ag-mi-sep" />
          <button v-if="canRef" type="button" class="ag-mi" role="menuitem" @click="act((r) => emit('reference', r!.path))">
            <UIcon name="i-lucide-at-sign" />{{ t('agent.code.files.reference') }}
          </button>
          <button type="button" class="ag-mi" role="menuitem" @click="act((r) => copy(absPath(r!.path)))">
            <UIcon name="i-lucide-copy" />{{ t('agent.code.files.copyPath') }}
          </button>
          <button type="button" class="ag-mi" role="menuitem" @click="act((r) => copy(r!.path))">
            <UIcon name="i-lucide-clipboard-copy" />{{ t('agent.code.files.copyRel') }}
          </button>
          <button v-if="menu.row.isDir" type="button" class="ag-mi" role="menuitem" @click="act((r) => openInFiles(r!.path))">
            <UIcon name="i-lucide-folder-open" />{{ t('agent.code.files.openInManager') }}
          </button>
          <div class="ag-mi-sep" />
          <button type="button" class="ag-mi" role="menuitem" @click="act((r) => startRename(r!))">
            <UIcon name="i-lucide-pencil" /><span class="agc-flex">{{ t('agent.code.files.rename') }}</span><span class="agc-ctx__k">F2</span>
          </button>
          <button type="button" class="ag-mi is-bad" role="menuitem" @click="act((r) => askDelete(r!))">
            <UIcon name="i-lucide-trash-2" /><span class="agc-flex">{{ t('agent.code.files.delete') }}</span><span class="agc-ctx__k">Delete</span>
          </button>
        </template>
        <template v-else>
          <div class="ag-mi-sep" />
          <button type="button" class="ag-mi" role="menuitem" @click="act(refresh)">
            <UIcon name="i-lucide-refresh-cw" />{{ t('agent.code.files.refresh') }}
          </button>
          <button type="button" class="ag-mi" role="menuitem" :disabled="!expanded.size" @click="act(collapseAll)">
            <UIcon name="i-lucide-copy-minus" />{{ t('agent.code.files.collapse') }}
          </button>
          <button type="button" class="ag-mi" role="menuitem" @click="act(() => copy(cwd))">
            <UIcon name="i-lucide-copy" />{{ t('agent.code.files.copyPath') }}
          </button>
        </template>
      </div>
    </Teleport>

    <AgConfirm
      v-model="delOpen"
      :title="t('agent.code.files.deleteTitle', { name: delRow?.name || '' })"
      :message="delRow?.isDir ? t('agent.code.files.deleteDirMsg', { path: delRow?.path || '' }) : t('agent.code.files.deleteMsg', { path: delRow?.path || '' })"
      :confirm-text="t('agent.code.files.delete')"
      danger
      :on-confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useEventListener } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { createAgentDir, deleteAgentFiles, listAgentFiles, moveAgentFile, saveAgentFile, type AgentFileScope } from '@/api/agent'
import type { AgentFileEntry } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import { AGENT_PATH_MIME } from '@/views/agent/utils/format'
import { isUnder, movedPath } from './fileState'

const props = defineProps<{
  /** 会话目录（sessionId）还是工作区（workspaceId） */
  scope: AgentFileScope
  /** 根目录的绝对路径：复制路径、在文件管理中打开用 */
  cwd: string
  empty: string
  tip?: (path: string) => string
  /** 正在标签页里看的文件，高亮出来 */
  active?: string
  /** 能不能「添加到对话」 */
  canRef?: boolean
}>()
const emit = defineEmits<{
  open: [path: string]
  /** 改名或挪了位置（文件夹的话下面的文件一起变了） */
  moved: [from: string, to: string]
  deleted: [paths: string[]]
  /** 右键「添加到对话」 */
  reference: [path: string]
}>()
const { t } = useI18n()
const router = useRouter()

const parentOf = (p: string) => (p.includes('/') ? p.slice(0, p.lastIndexOf('/')) : '')
const baseOf = (p: string) => p.slice(p.lastIndexOf('/') + 1)
const join = (dir: string, name: string) => (dir ? `${dir}/${name}` : name)
const rootName = computed(() => baseOf(props.cwd.replace(/[\\/]+$/, '').replace(/\\/g, '/')) || props.cwd)
/** 根目录下某个路径的绝对路径，分隔符跟根目录一样 */
const absPath = (rel: string) => {
  const sep = props.cwd.includes('\\') ? '\\' : '/'
  const root = props.cwd.replace(/[\\/]+$/, '')
  return rel ? `${root}${sep}${rel.split('/').join(sep)}` : props.cwd
}

// —— 目录 ——
const dirs = reactive(new Map<string, AgentFileEntry[]>())
const expanded = reactive(new Set<string>())
const loaded = ref(false)
const selected = ref('')
const listEl = ref<HTMLElement | null>(null)

interface Row {
  key: string
  path: string
  name: string
  isDir: boolean
  change: string
  depth: number
  /** 新建时插在文件夹最前面的输入行 */
  input?: boolean
}
const rows = computed(() => {
  const out: Row[] = []
  const walk = (dir: string, depth: number) => {
    if (edit.mode === 'new' && edit.dir === dir) out.push({ key: '\n', path: '', name: '', isDir: edit.isDir, change: '', depth, input: true })
    for (const e of dirs.get(dir) ?? []) {
      out.push({ key: e.path, path: e.path, name: e.name, isDir: e.isDir, change: e.change, depth })
      if (e.isDir && expanded.has(e.path)) walk(e.path, depth + 1)
    }
  }
  walk('', 0)
  return out
})

const IMAGE = /\.(png|jpe?g|gif|webp|bmp|ico|svg|avif)$/i
const CODE = /\.(ts|tsx|js|jsx|mjs|cjs|vue|go|py|rs|java|kt|c|h|cc|cpp|hpp|cs|rb|php|swift|sh|bash|ps1|bat|cmd|lua|sql|scss|sass|less|css|html?|xml|proto)$/i
const iconOf = (n: Row) => {
  if (n.isDir) return expanded.has(n.path) ? 'i-lucide-folder-open' : 'i-lucide-folder'
  if (IMAGE.test(n.name)) return 'i-lucide-file-image'
  if (/\.(md|markdown|mdx|txt|rst|log)$/i.test(n.name)) return 'i-lucide-file-text'
  if (/\.(json|jsonc|ya?ml|toml|ini|env|conf)$/i.test(n.name)) return 'i-lucide-file-cog'
  if (/\.(zip|tar|gz|tgz|7z|rar|xz)$/i.test(n.name)) return 'i-lucide-file-archive'
  if (CODE.test(n.name)) return 'i-lucide-file-code'
  return 'i-lucide-file'
}

let gen = 0
const loadDir = async (dir: string) => {
  const g = gen
  try {
    const { data } = await listAgentFiles(props.scope, dir)
    if (g === gen) dirs.set(dir, data.entries ?? [])
  } catch {
    // 目录没了（被删、被挪走）：收起来
    if (g === gen && dir) {
      expanded.delete(dir)
      dirs.delete(dir)
    }
  } finally {
    if (!dir && g === gen) loaded.value = true
  }
}
/** 根目录和展开着、看得见（上级都展开着）的目录重读一遍 */
const reload = () => {
  const shown = [...expanded].filter((d) => {
    for (let p = parentOf(d); p; p = parentOf(p)) if (!expanded.has(p)) return false
    return true
  })
  return Promise.all(['', ...shown].map((d) => loadDir(d)))
}
const refreshing = ref(false)
const refresh = async () => {
  refreshing.value = true
  try {
    await reload()
  } finally {
    refreshing.value = false
  }
}
const toggleDir = (path: string, open = !expanded.has(path)) => {
  if (!open) {
    expanded.delete(path)
    return
  }
  expanded.add(path)
  // 先显示上次读到的，同时重读一遍
  void loadDir(path)
}
const collapseAll = () => expanded.clear()

const select = (path: string) => {
  selected.value = path
  void nextTick(() => listEl.value?.querySelector(`[data-path="${CSS.escape(path)}"]`)?.scrollIntoView({ block: 'nearest' }))
}
const onRow = (n: Row) => {
  selected.value = n.path
  listEl.value?.focus({ preventScroll: true })
  if (n.isDir) toggleDir(n.path)
  else emit('open', n.path)
}

// —— 键盘 ——
const onKey = (e: KeyboardEvent) => {
  if (edit.mode) return
  const list = rows.value.filter((r) => !r.input)
  if (!list.length) return
  const i = list.findIndex((r) => r.path === selected.value)
  const cur = list[i]
  switch (e.key) {
    case 'ArrowDown':
      select((list[i + 1] ?? list[i] ?? list[0])!.path)
      break
    case 'ArrowUp':
      select((i > 0 ? list[i - 1] : list[0])!.path)
      break
    case 'ArrowRight':
      if (!cur?.isDir) return
      if (!expanded.has(cur.path)) toggleDir(cur.path, true)
      else if (list[i + 1] && list[i + 1]!.depth > cur.depth) select(list[i + 1]!.path)
      break
    case 'ArrowLeft':
      if (!cur) return
      if (cur.isDir && expanded.has(cur.path)) toggleDir(cur.path, false)
      else if (parentOf(cur.path)) select(parentOf(cur.path))
      break
    case 'Enter':
      if (!cur) return
      if (cur.isDir) toggleDir(cur.path)
      else emit('open', cur.path)
      break
    case 'F2':
      if (cur) void startRename(cur)
      break
    case 'Delete':
      if (cur) askDelete(cur)
      break
    default:
      return
  }
  e.preventDefault()
}

// —— 新建、改名（树里就地出一个输入框） ——
const edit = reactive({ mode: '' as '' | 'new' | 'rename', dir: '', path: '', isDir: false, name: '' })
let inputEl: HTMLInputElement | null = null
const setInput = (el: unknown) => {
  inputEl = el instanceof HTMLInputElement ? el : null
}
const focusInput = async (selectStem = false) => {
  await nextTick()
  if (!inputEl) return
  inputEl.focus()
  const dot = edit.name.lastIndexOf('.')
  // 改文件名时先选中扩展名前面那段
  inputEl.setSelectionRange(0, selectStem && !edit.isDir && dot > 0 ? dot : edit.name.length)
}
/** 新建在哪个文件夹：选中的是文件夹就在它里面，是文件就在它旁边 */
const targetDir = (n: Row | null | undefined) => (n ? (n.isDir ? n.path : parentOf(n.path)) : '')
const selectedRow = () => rows.value.find((r) => !r.input && r.path === selected.value)
const startNew = async (isDir: boolean, dir = targetDir(selectedRow())) => {
  closeMenu()
  if (dir) {
    expanded.add(dir)
    if (!dirs.has(dir)) await loadDir(dir)
  }
  Object.assign(edit, { mode: 'new', dir, path: '', isDir, name: '' })
  await focusInput()
  inputEl?.scrollIntoView({ block: 'nearest' })
}
const startRename = async (n: Row) => {
  closeMenu()
  selected.value = n.path
  Object.assign(edit, { mode: 'rename', dir: parentOf(n.path), path: n.path, isDir: n.isDir, name: n.name })
  await focusInput(true)
}
const cancelEdit = () => {
  const was = edit.mode
  edit.mode = ''
  if (was) listEl.value?.focus({ preventScroll: true })
}
let committing = false
const commitEdit = async () => {
  if (!edit.mode || committing) return
  const { mode, dir, path, isDir } = edit
  const name = edit.name.trim().replace(/\\/g, '/').replace(/^\/+|\/+$/g, '')
  if (!name || (mode === 'rename' && name === baseOf(path))) {
    cancelEdit()
    return
  }
  if (name.split('/').some((s) => !s.trim() || s === '.' || s === '..') || (mode === 'rename' && name.includes('/'))) {
    feedback.warning(t('agent.code.files.badName'))
    void focusInput()
    return
  }
  committing = true
  const target = join(dir, name)
  try {
    if (mode === 'new') {
      if (isDir) await createAgentDir(props.scope, target)
      else await saveAgentFile(props.scope, { path: target, content: '', create: true })
      // 名字里带 / 的话中间的文件夹也建出来了：都展开
      for (let p = parentOf(target); p && p !== dir; p = parentOf(p)) expanded.add(p)
    } else {
      await moveAgentFile(props.scope, path, target)
      afterMove(path, target)
    }
    edit.mode = ''
    await reload()
    select(target)
    listEl.value?.focus({ preventScroll: true })
    if (mode === 'new' && !isDir) emit('open', target)
  } catch {
    // 请求层已经提示过错误，输入框留着改
    void focusInput()
  } finally {
    committing = false
  }
}
const onInputKey = (e: KeyboardEvent) => {
  if (e.key === 'Enter') {
    e.preventDefault()
    void commitEdit()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    cancelEdit()
  }
}

/** 改名或挪动之后：展开状态、缓存的目录、选中的跟着换路径 */
const afterMove = (from: string, to: string) => {
  for (const d of [...expanded]) {
    if (!isUnder(d, from)) continue
    expanded.delete(d)
    expanded.add(movedPath(d, from, to))
  }
  for (const d of [...dirs.keys()]) if (isUnder(d, from)) dirs.delete(d)
  if (selected.value && isUnder(selected.value, from)) selected.value = movedPath(selected.value, from, to)
  emit('moved', from, to)
}

// —— 删除 ——
const delOpen = ref(false)
const delRow = ref<Row | null>(null)
const askDelete = (n: Row) => {
  closeMenu()
  delRow.value = n
  delOpen.value = true
}
const doDelete = async () => {
  const n = delRow.value
  if (!n) return
  await deleteAgentFiles(props.scope, [n.path])
  for (const d of [...expanded]) if (isUnder(d, n.path)) expanded.delete(d)
  for (const d of [...dirs.keys()]) if (isUnder(d, n.path)) dirs.delete(d)
  if (selected.value && isUnder(selected.value, n.path)) selected.value = parentOf(n.path)
  emit('deleted', [n.path])
  feedback.success(t('agent.code.files.deleted', { name: n.name }))
  void reload()
  listEl.value?.focus({ preventScroll: true })
}

// —— 右键菜单 ——
const menu = ref<{ x: number; y: number; row: Row | null } | null>(null)
const menuEl = ref<HTMLElement | null>(null)
/** 菜单里新建在哪：右键的是文件夹就在它里面，是文件就在它旁边，空白处在根目录 */
const menuDir = computed(() => targetDir(menu.value?.row))
const openMenu = async (e: MouseEvent, row: Row | null) => {
  if (edit.mode) return
  if (row) selected.value = row.path
  menu.value = { x: e.clientX, y: e.clientY, row }
  await nextTick()
  const el = menuEl.value
  if (!el || !menu.value) return
  const { width, height } = el.getBoundingClientRect()
  menu.value.x = Math.max(8, Math.min(menu.value.x, window.innerWidth - width - 8))
  menu.value.y = menu.value.y + height > window.innerHeight - 8 ? Math.max(8, menu.value.y - height) : menu.value.y
  el.querySelector<HTMLElement>('.ag-mi')?.focus()
}
const closeMenu = () => {
  menu.value = null
}
/** 菜单项：先关菜单，再对右键的那一行（空白处为 null）做事 */
const act = (fn: (row: Row | null) => unknown) => {
  const row = menu.value?.row ?? null
  closeMenu()
  void fn(row)
}
onClickOutside(menuEl, closeMenu)
useEventListener(window, 'blur', closeMenu)
useEventListener(window, 'resize', closeMenu)
useEventListener(listEl, 'scroll', closeMenu)

const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    feedback.success(t('agent.code.files.copied'))
  } catch {
    feedback.error(t('agent.code.diff.copyFailed'))
  }
}
const openInFiles = (rel: string) => router.push({ path: '/file', query: { workdir: absPath(rel) } })

// —— 拖动：拖到文件夹上是挪位置；拖到输入框上是引用（输入框自己接） ——
const drag = ref('')
/** 放下去会进的文件夹（'' 是根目录），null 是放不下 */
const dropDir = ref<string | null>(null)
let hoverTimer: ReturnType<typeof setTimeout> | undefined
let hoverDir = ''
const onDragStart = (e: DragEvent, n: Row) => {
  if (!e.dataTransfer) return
  e.dataTransfer.setData(AGENT_PATH_MIME, n.path)
  e.dataTransfer.setData('text/plain', n.path)
  e.dataTransfer.effectAllowed = 'copyMove'
  drag.value = n.path
  selected.value = n.path
}
const stopHover = () => {
  clearTimeout(hoverTimer)
  hoverDir = ''
}
const onDragOver = (e: DragEvent, n: Row | null) => {
  // 只管树里自己拖的，外面拖进来的（电脑上的文件）不接
  const from = drag.value
  if (!from || n?.input) return
  const dir = targetDir(n)
  // 本来就在这个文件夹里、或者要挪进它自己里面：放不下
  if (parentOf(from) === dir || isUnder(dir, from)) {
    dropDir.value = null
    stopHover()
    return
  }
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  dropDir.value = dir
  // 停在收起来的文件夹上一会儿就展开
  if (n?.isDir && !expanded.has(n.path)) {
    if (hoverDir !== n.path) {
      stopHover()
      hoverDir = n.path
      hoverTimer = setTimeout(() => toggleDir(n.path, true), 600)
    }
  } else stopHover()
}
const onDragLeave = (e: DragEvent) => {
  if (!(e.relatedTarget instanceof Node) || !listEl.value?.contains(e.relatedTarget)) {
    dropDir.value = null
    stopHover()
  }
}
const endDrag = () => {
  drag.value = ''
  dropDir.value = null
  stopHover()
}
const onDrop = async (e: DragEvent) => {
  const from = drag.value
  const dir = dropDir.value
  endDrag()
  if (!from || dir == null) return
  e.preventDefault()
  const to = join(dir, baseOf(from))
  try {
    await moveAgentFile(props.scope, from, to)
    afterMove(from, to)
    if (dir) expanded.add(dir)
    await reload()
    select(to)
  } catch {
    /* shown */
  }
}

// —— 换了会话或工作区：清空重读 ——
// 分开看两个值：scope 在模板里每次都是新对象，整个比会每次渲染都当成换了
watch(
  [() => props.scope.sessionId, () => props.scope.workspaceId],
  () => {
    gen++
    dirs.clear()
    expanded.clear()
    selected.value = ''
    edit.mode = ''
    loaded.value = false
    closeMenu()
    if (props.scope.sessionId || props.scope.workspaceId) void loadDir('')
  },
  { immediate: true },
)
onBeforeUnmount(stopHover)

defineExpose({ reload })
</script>
