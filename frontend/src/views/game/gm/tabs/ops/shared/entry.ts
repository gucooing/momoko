// 运营配置条目的公共计算（卡池、活动、竞赛、登录奖励、公告共用，只有这一份）：
// 期间状态与显示文字、新条目的 ID、复制、和服务端当前内容比出的草稿状态、数字输入。
import { computed, type ComputedRef } from 'vue'
import { deepEqual } from './diff'
import { formatShort } from './time'

export type PeriodStatus = 'open' | 'permanent' | 'upcoming' | 'ended'

/** 结束时间晚于这个时刻视为常驻（Home.json 里常驻的任务活动写的是 2100 年）。 */
export const FAR_FUTURE = Date.UTC(2090, 0, 1)

/** 常驻活动写进配置的结束时间（2099-12-31 23:59:59 UTC+9，和官方数据相同）。 */
export const PERMANENT_END = 4102412399000

export function periodStatus(start: number, end: number, now: number): PeriodStatus {
  if (start > now) return 'upcoming'
  if (!end || end >= FAR_FUTURE) return 'permanent'
  if (end <= now) return 'ended'
  return 'open'
}

export const periodVariant = (s: PeriodStatus) =>
  ({ open: 'success', permanent: 'info', upcoming: 'warning', ended: 'neutral' })[s] as 'success' | 'info' | 'warning' | 'neutral'

/** 「开始 → 结束」；常驻时只写开始。 */
export function periodText(start: number, end: number, zone: string, permanentLabel: (from: string) => string): string {
  const from = start ? formatShort(start, zone) : '—'
  if (!end || end >= FAR_FUTURE) return permanentLabel(from)
  return `${from} → ${formatShort(end, zone)}`
}

/** 下一个整点（新条目从这里开始）。 */
export const nextHour = (now: number) => Math.ceil(now / 3_600_000) * 3_600_000

/** 新结束时间写回一项配置：排名确定、关闭时间与结束时间的间隔保持不变。 */
export function moveEnd(item: { endTime: number; fixRankTime?: number; closeTime?: number }, end: number) {
  const prev = item.endTime
  item.endTime = end
  if (item.fixRankTime) item.fixRankTime = end + (item.fixRankTime - prev)
  if (item.closeTime) item.closeTime = end + Math.max(0, item.closeTime - prev)
}

/** 不重复的新 ID：base、base-2、base-3… */
export function freeId(taken: (id: string) => boolean, base: string): string {
  if (!taken(base)) return base
  for (let i = 2; ; i++) if (!taken(`${base}-${i}`)) return `${base}-${i}`
}

/** ID 末尾的数字加一（pvp_rate_config-56 → pvp_rate_config-57）；没有数字时同 freeId。 */
export function nextId(id: string, taken: (id: string) => boolean): string {
  const m = /^(.*?)(\d+)$/.exec(id)
  if (!m) return freeId(taken, id)
  let n = Number(m[2])
  for (;;) {
    const v = `${m[1]}${String(++n).padStart(m[2]!.length, '0')}`
    if (!taken(v)) return v
  }
}

/** 深拷贝一项配置（JSON 往返）。 */
export const cloneJson = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T

export type DraftState = 'new' | 'changed' | undefined

/** 一项配置在草稿里的状态：服务端没有为新增，内容不同为已改。 */
export const draftOf = (published: unknown, working: unknown): DraftState =>
  working === undefined ? undefined : published === undefined ? 'new' : deepEqual(published, working) ? undefined : 'changed'

/** 数字输入框的值：非负整数。 */
export function intOf(e: Event): number {
  const v = (e.target as HTMLInputElement).valueAsNumber
  return Number.isFinite(v) ? Math.max(0, Math.floor(v)) : 0
}

/** 表单给的新内容写回草稿里的同一个对象（别处的引用不变）。 */
export function assignInPlace(target: object, next: Record<string, unknown>) {
  const t = target as Record<string, unknown>
  for (const k of Object.keys(t)) if (!(k in next)) delete t[k]
  Object.assign(t, next)
}

/**
 * 窗口里正在编辑的那一项：按打开时的 ID 找到后跟着这个对象（表单里改了 ID 也不丢）；
 * 配置重新读取（发布、放弃草稿）后对象换了，再按它最后的 ID 找。
 */
export function useTracked<T>(list: () => T[] | undefined, idOf: (x: T) => string, id: string): ComputedRef<T | undefined> {
  let current: T | undefined
  let lastId = id
  return computed(() => {
    const items = list() || []
    if (current && items.includes(current)) {
      lastId = idOf(current)
      return current
    }
    current = items.find((x) => idOf(x) === lastId)
    return current
  })
}
