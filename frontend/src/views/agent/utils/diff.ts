// 对话里改动的增删行数：ACP v2 给的 patch 直接数，v1 给的新旧全文按行比。
import type { AgentDiff } from '@/views/agent/code/types'

const toLines = (text: string) => {
  if (!text) return []
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  if (lines[lines.length - 1] === '') lines.pop()
  return lines
}

/** 两段文本按行比，增删了多少行：去掉头尾相同的行后按最长公共子序列算；太大时只去头尾，剩下的都算改动 */
export const lineDelta = (before: string, after: string): { added: number; deleted: number } => {
  const x = toLines(before)
  const y = toLines(after)
  let s = 0
  while (s < x.length && s < y.length && x[s] === y[s]) s++
  let ex = x.length
  let ey = y.length
  while (ex > s && ey > s && x[ex - 1] === y[ey - 1]) {
    ex--
    ey--
  }
  const n = ex - s
  const m = ey - s
  if (!n || !m || n * m > 4_000_000) return { added: m, deleted: n }
  let prev = new Uint32Array(m + 1)
  let cur = new Uint32Array(m + 1)
  for (let i = 1; i <= n; i++) {
    const xi = x[s + i - 1]
    for (let j = 1; j <= m; j++) cur[j] = xi === y[s + j - 1] ? prev[j - 1]! + 1 : Math.max(prev[j]!, cur[j - 1]!)
    ;[prev, cur] = [cur, prev]
  }
  const same = prev[m]!
  return { added: m - same, deleted: n - same }
}

/** git patch 从第一个 @@ 段开始的部分：前面的 diff --git、index、--- / +++ 是文件头 */
export const patchBody = (patch?: string) => {
  const p = patch || ''
  if (p.startsWith('@@')) return p
  const at = p.indexOf('\n@@')
  return at >= 0 ? p.slice(at + 1) : ''
}

/** patch 里增删的行数 */
export const patchDelta = (patch?: string) => {
  let added = 0
  let deleted = 0
  for (const line of patchBody(patch).split('\n')) {
    if (line.startsWith('+')) added++
    else if (line.startsWith('-')) deleted++
  }
  return { added, deleted }
}

const cache = new WeakMap<AgentDiff, { added: number; deleted: number }>()
/** 一次改动增删的行数（同一个对象只算一次） */
export const diffDelta = (d: AgentDiff) => {
  let v = cache.get(d)
  if (!v) {
    v = d.op ? patchDelta(d.patch) : lineDelta(d.oldText ?? '', d.newText ?? '')
    cache.set(d, v)
  }
  return v
}

/** 这次改动对文件做了什么：add / delete / modify / move / copy */
export const diffOp = (d: AgentDiff) => d.op || (d.oldText == null ? 'add' : 'modify')
