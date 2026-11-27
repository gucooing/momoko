// 顶栏「终端」悬浮窗的状态（模块级，挂在外壳上，切换页面不丢）：
// 窗口开着 / 最小化、终端标签（本机 shell 或 SSH 主机）、当前标签、窗口位置与大小、外观（这两样记在 localStorage）。
// 关闭窗口 = 清空全部标签，终端组件卸载即断开连接，和刷新网页一样。
import { onActivated, onBeforeUnmount, onDeactivated, reactive, watch } from 'vue'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import { FONT_SIZE_MAX, FONT_SIZE_MIN, sharedTerminalFontSize } from '@/components/terminal/useTerminalX'

/** 本机终端、SSH，或者 Agent 官方账户的登录终端（在服务器上跑那一系的登录命令） */
export type DockTabKind = 'local' | 'ssh' | 'login'

export interface DockTab {
  key: string
  kind: DockTabKind
  /** SSH 主机 id（本机终端为空） */
  hostId: string
  /** 登录终端：Agent 运行时 id */
  runtimeId: string
  /** 本机终端的序号（同时开多个时区分；指定了目录的不编号） */
  seq: number
  /** SSH：user@主机名，连上后按主机详情更新 */
  label: string
  /** 本机终端的起始目录（绝对路径）；空 = 主目录 */
  cwd: string
  status: ConsoleSocketStatus
}

/** 路径的最后一段（标签上显示的文件夹名）；Windows 和 Unix 的分隔符都认 */
export const folderName = (path: string) => {
  const p = path.replace(/[\\/]+$/, '')
  return p.slice(Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\')) + 1) || path
}

export interface DockRect {
  x: number
  y: number
  w: number
  h: number
}

export const DOCK_MIN_W = 420
export const DOCK_MIN_H = 240
const RECT_KEY = 'momoko.termDock.rect'
const EDGE = 16

const clamp = (n: number, min: number, max: number) => Math.min(Math.max(n, min), Math.max(min, max))

/** 窗口整体留在视口内：先限制大小，再限制位置 */
export const fitRect = (r: DockRect): DockRect => {
  const vw = window.innerWidth
  const vh = window.innerHeight
  const w = clamp(Math.round(r.w), DOCK_MIN_W, vw)
  const h = clamp(Math.round(r.h), DOCK_MIN_H, vh)
  return { w, h, x: clamp(Math.round(r.x), 0, vw - w), y: clamp(Math.round(r.y), 0, vh - h) }
}

const defaultRect = (): DockRect => {
  const w = Math.min(760, window.innerWidth - EDGE * 2)
  const h = Math.min(440, window.innerHeight - EDGE * 2)
  return fitRect({ w, h, x: window.innerWidth - w - EDGE * 2, y: window.innerHeight - h - EDGE * 2 })
}

const readRect = (): DockRect => {
  try {
    const raw = JSON.parse(localStorage.getItem(RECT_KEY) || 'null') as Partial<DockRect> | null
    if (raw && [raw.x, raw.y, raw.w, raw.h].every((n) => typeof n === 'number' && Number.isFinite(n))) {
      return fitRect(raw as DockRect)
    }
  } catch {
    /* 坏数据按默认处理 */
  }
  return defaultRect()
}

export const saveRect = () => {
  try {
    localStorage.setItem(RECT_KEY, JSON.stringify(dock.rect))
  } catch {
    /* 隐私模式等写不进去就算了 */
  }
}

// —— 外观：背景不透明度、透明时模糊背后的页面、字号（只管悬浮终端，不影响其他终端页）——
export interface DockLook {
  /** 背景不透明度，LOOK_OPACITY_MIN ~ 1；文字始终不透明 */
  opacity: number
  /** 背景透明时把后面的页面模糊掉 */
  blur: boolean
  fontSize: number
}

export const LOOK_OPACITY_MIN = 0.3
const LOOK_KEY = 'momoko.termDock.look'

// 字号从其他终端页现在用的开始，原来调过字号的人打开悬浮终端不会变
const defaultLook = (): DockLook => ({ opacity: 1, blur: true, fontSize: sharedTerminalFontSize() })

const fitLook = (l: DockLook): DockLook => ({
  opacity: clamp(Math.round(l.opacity * 100) / 100, LOOK_OPACITY_MIN, 1),
  blur: l.blur,
  fontSize: clamp(Math.round(l.fontSize), FONT_SIZE_MIN, FONT_SIZE_MAX),
})

const readLook = (): DockLook => {
  const def = defaultLook()
  try {
    const raw = JSON.parse(localStorage.getItem(LOOK_KEY) || 'null') as Partial<DockLook> | null
    if (raw && typeof raw === 'object') {
      const num = (n: unknown, d: number) => (typeof n === 'number' && Number.isFinite(n) ? n : d)
      return fitLook({
        opacity: num(raw.opacity, def.opacity),
        blur: typeof raw.blur === 'boolean' ? raw.blur : def.blur,
        fontSize: num(raw.fontSize, def.fontSize),
      })
    }
  } catch {
    /* 坏数据按默认处理 */
  }
  return def
}

let keySeq = 0
let localSeq = 0

export const dock = reactive({
  open: false,
  minimized: false,
  tabs: [] as DockTab[],
  active: '',
  rect: readRect(),
  look: readLook(),
})

/** 改外观：就地改（终端拿着 dock.look.fontSize 的引用），收在范围内并记下来 */
export const setLook = (patch: Partial<DockLook>) => {
  Object.assign(dock.look, fitLook({ ...dock.look, ...patch }))
  try {
    localStorage.setItem(LOOK_KEY, JSON.stringify(dock.look))
  } catch {
    /* 隐私模式等写不进去就算了 */
  }
}

export const resetLook = () => {
  try {
    localStorage.removeItem(LOOK_KEY)
  } catch {
    /* 同上 */
  }
  Object.assign(dock.look, defaultLook())
}

export const showDock = () => {
  if (!dock.open) dock.rect = fitRect(dock.rect)
  dock.open = true
  dock.minimized = false
}

/** 顶栏按钮：没开就打开；开着就在最小化 / 还原之间切换 */
export const toggleDock = () => {
  if (!dock.open) showDock()
  else dock.minimized = !dock.minimized
}

export const minimizeDock = () => {
  dock.minimized = true
}

/** 关闭：断开全部终端 */
export const closeDock = () => {
  dock.tabs = []
  dock.active = ''
  dock.open = false
  dock.minimized = false
  localSeq = 0
}

/** 新开一个标签：SSH 给 hostId、label；本机终端可给 cwd（在那个目录打开）；登录终端给 runtimeId、label */
export const addTab = (
  kind: DockTabKind,
  opts: { hostId?: string; label?: string; cwd?: string; runtimeId?: string } = {},
) => {
  const cwd = kind === 'local' ? opts.cwd || '' : ''
  const tab: DockTab = {
    key: `t${keySeq++}`,
    kind,
    hostId: opts.hostId || '',
    runtimeId: opts.runtimeId || '',
    seq: kind === 'local' && !cwd ? ++localSeq : 0,
    label: opts.label || '',
    cwd,
    status: 'connecting',
  }
  dock.tabs.push(tab)
  dock.active = tab.key
  showDock()
}

export const updateTab = (key: string, patch: Partial<Pick<DockTab, 'label' | 'status'>>) => {
  const tab = dock.tabs.find((t) => t.key === key)
  if (tab) Object.assign(tab, patch)
}

export const removeTab = (key: string) => {
  const idx = dock.tabs.findIndex((t) => t.key === key)
  if (idx < 0) return
  dock.tabs.splice(idx, 1)
  if (dock.active === key) dock.active = dock.tabs[Math.max(0, idx - 1)]?.key || ''
  if (!dock.tabs.length) localSeq = 0
}

// —— Agent Code 页当前会话的运行目录（工作区目录、工作树目录或临时目录），给「在当前会话的目录打开」用 ——
// 只在 Code 页在前台、开着会话时有值；切到别的页（KeepAlive 停用）或卸载就清掉。
export const sessionDir = reactive({ cwd: '', name: '' })

/** Code 页调用：把当前会话的目录交给悬浮终端 */
export const useDockSessionDir = (get: () => { cwd: string; workspaceName: string } | null | undefined) => {
  let shown = true
  const sync = () => {
    const s = shown ? get() : null
    sessionDir.cwd = s?.cwd || ''
    sessionDir.name = s?.workspaceName || ''
  }
  watch([() => get()?.cwd, () => get()?.workspaceName], sync, { immediate: true })
  onActivated(() => {
    shown = true
    sync()
  })
  onDeactivated(() => {
    shown = false
    sync()
  })
  onBeforeUnmount(() => {
    shown = false
    sync()
  })
}
