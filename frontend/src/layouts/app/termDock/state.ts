// 顶栏「终端」悬浮窗的状态（模块级，挂在外壳上，切换页面不丢）：
// 窗口开着 / 最小化、终端标签（本机 shell 或 SSH 主机）、当前标签、窗口位置与大小（位置大小记在 localStorage）。
// 关闭窗口 = 清空全部标签，终端组件卸载即断开连接，和刷新网页一样。
import { reactive } from 'vue'
import type { ConsoleSocketStatus } from '@/stores/instance/types'

export type DockTabKind = 'local' | 'ssh'

export interface DockTab {
  key: string
  kind: DockTabKind
  /** SSH 主机 id（本机终端为空） */
  hostId: string
  /** 本机终端的序号（同时开多个时区分） */
  seq: number
  /** SSH：user@主机名，连上后按主机详情更新 */
  label: string
  status: ConsoleSocketStatus
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

let keySeq = 0
let localSeq = 0

export const dock = reactive({
  open: false,
  minimized: false,
  tabs: [] as DockTab[],
  active: '',
  rect: readRect(),
})

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

export const addTab = (kind: DockTabKind, hostId = '', label = '') => {
  const tab: DockTab = {
    key: `t${keySeq++}`,
    kind,
    hostId,
    seq: kind === 'local' ? ++localSeq : 0,
    label,
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
