// 打开的文件标签共用的状态。文件标签切走时 FileTab 会卸下：没保存的改动、光标和滚动位置、源码 / 预览先存在这里，切回来接着用；
// 标签栏靠 dirtyFiles 显示「没保存」的圆点。文件在树里改名、挪位置、删除时跟着换键或丢掉。
// 键是「范围 + 路径」：范围是会话 id，新会话页是 ws:工作区 id。只在这个页面的内存里，刷新页面就没了。
import type { AgentModTime } from '@/api/agent'

export type FileMode = 'source' | 'split' | 'preview'

export interface FileDraft {
  /** 编辑器里的内容 */
  content: string
  /** 磁盘上的内容（打开或上次保存时） */
  saved: string
  /** 磁盘上那份的修改时间，保存时带回去 */
  mod: AgentModTime
}

/** 没保存的改动 */
const drafts = new Map<string, FileDraft>()
/** 光标、滚动位置（Monaco 的 view state） */
const views = new Map<string, unknown>()
/** 源码 / 并排 / 预览 */
const modes = new Map<string, FileMode>()
/** 正开着的文件标签：键 → 取它当前内容的函数。改名、删除时要从开着的标签里取 */
const live = new Map<string, () => FileDraft | null>()
/** 有没保存改动的文件（键） */
export const dirtyFiles = reactive(new Set<string>())

export const fileScope = (sessionId?: string, workspaceId?: string) => (workspaceId ? `ws:${workspaceId}` : sessionId || '')
export const fileKey = (scope: string, path: string) => `${scope}\n${path}`
export const isDirty = (scope: string, path: string) => dirtyFiles.has(fileKey(scope, path))

/** path 本身或在它下面（path 是文件夹时） */
const under = (p: string, path: string) => p === path || p.startsWith(`${path}/`)

export const fileState = {
  draft: (key: string) => drafts.get(key),
  view: (key: string) => views.get(key),
  mode: (key: string) => modes.get(key),
  setMode: (key: string, mode: FileMode) => modes.set(key, mode),
  setDirty(key: string, dirty: boolean) {
    if (dirty) dirtyFiles.add(key)
    else dirtyFiles.delete(key)
  },
  /** 标签开着：登记取内容的函数，返回卸下时调用的函数（标签被改名、删掉、关掉之后不再存） */
  attach(key: string, get: () => FileDraft | null, view: () => unknown) {
    live.set(key, get)
    return () => {
      if (live.get(key) !== get) return
      live.delete(key)
      const d = get()
      if (d && d.content !== d.saved) drafts.set(key, d)
      else drafts.delete(key)
      views.set(key, view())
    }
  },
}

/** 改名或挪位置：from（文件，或文件夹和下面的文件）换成 to 下面的同名路径 */
export const moveFileState = (scope: string, from: string, to: string) => {
  const prefix = `${scope}\n`
  const rekey = (key: string) => `${prefix}${to}${key.slice(prefix.length + from.length)}`
  const hit = (key: string) => key.startsWith(prefix) && under(key.slice(prefix.length), from)
  for (const [key, get] of [...live]) {
    if (!hit(key)) continue
    live.delete(key)
    const d = get()
    if (d && d.content !== d.saved) drafts.set(key, d)
  }
  for (const map of [drafts, views, modes] as Map<string, unknown>[]) {
    for (const [key, v] of [...map]) {
      if (!hit(key)) continue
      map.delete(key)
      map.set(rekey(key), v)
    }
  }
  for (const key of [...dirtyFiles]) {
    if (!hit(key)) continue
    dirtyFiles.delete(key)
    dirtyFiles.add(rekey(key))
  }
}

/** 关掉标签（不保存）或文件被删掉：丢掉它（和下面的文件）的一切 */
export const dropFileState = (scope: string, paths: string[]) => {
  const prefix = `${scope}\n`
  const hit = (key: string) => key.startsWith(prefix) && paths.some((p) => under(key.slice(prefix.length), p))
  for (const map of [live, drafts, views, modes] as Map<string, unknown>[]) {
    for (const key of [...map.keys()]) if (hit(key)) map.delete(key)
  }
  for (const key of [...dirtyFiles]) if (hit(key)) dirtyFiles.delete(key)
}

/**
 * 文件标签的 key：每次打开给一个新的，标签在 HTML 预览里跳到别的页面（换了路径）时 key 跟着走，
 * 文件页不重建（iframe 里的页面不重新加载）。没登记过的（切会话恢复的）用路径本身。
 */
export const useTabKeys = () => {
  const keys = reactive(new Map<string, string>())
  let seq = 0
  return {
    key: (path: string) => keys.get(path) ?? path,
    /** 新打开的标签（或改名、挪动之后要重建的） */
    fresh: (path: string) => keys.set(path, `\u0001${++seq}`),
    drop: (path: string) => keys.delete(path),
    /** 标签从 from 换成了 to：key 不变 */
    carry: (from: string, to: string) => {
      const k = keys.get(from) ?? from
      keys.delete(from)
      keys.set(to, k)
    },
  }
}

/** 打开的标签里，路径换成改名或挪动之后的（不受影响的原样） */
export const movedPath = (p: string, from: string, to: string) => (under(p, from) ? `${to}${p.slice(from.length)}` : p)
export const isUnder = under
