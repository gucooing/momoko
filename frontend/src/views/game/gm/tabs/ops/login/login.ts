// 登录奖励配置（data/LoginBonus.json）的本地检查：与服务端加载时的检查相同；演出 ID、条件与奖励条目由服务端核对。
import type { LoginBonusConfig, LoginBonusEvent, LoginBonusSelect } from '@/games/gakumas/types'
import { cloneJson, freeId } from '../shared/entry'
import { rewardValid } from '../shared/rewards'

/** 每日登录奖励固定 7 天一循环。 */
export const DAILY_DAYS = 7

export interface Problem {
  key: string
  params?: Record<string, string | number>
}

export function normalProblems(c: LoginBonusConfig): Problem[] {
  const n = c.normal
  const out: Problem[] = []
  if (!n.name) out.push({ key: 'name' })
  if (n.rewards.length !== DAILY_DAYS || !n.rewards.every(rewardValid)) out.push({ key: 'normalRewards' })
  if (!n.startLoginBonusMotionId || !n.endLoginBonusMotionId) out.push({ key: 'motion' })
  return out
}

export function totalProblems(c: LoginBonusConfig): Problem[] {
  const tl = c.total
  const out: Problem[] = []
  if (!tl.name) out.push({ key: 'name' })
  if (!tl.startLoginBonusMotionId || !tl.endLoginBonusMotionId) out.push({ key: 'motion' })
  let last = 0
  for (const r of tl.rewards) {
    if (r.days <= last || !rewardValid(r.reward)) {
      out.push({ key: 'totalDays' })
      break
    }
    last = r.days
  }
  if (!(tl.repeatIntervalDays > 0) || !rewardValid(tl.repeatReward)) out.push({ key: 'totalRepeat' })
  if (tl.repeatFromDays <= last) out.push({ key: 'totalRepeatFrom', params: { days: last } })
  return out
}

export function eventProblems(c: LoginBonusConfig, e: LoginBonusEvent): Problem[] {
  const out: Problem[] = []
  if (!e.id) out.push({ key: 'id' })
  else if (c.events.filter((x) => x.id === e.id).length > 1) out.push({ key: 'idDuplicate' })
  if (!e.name) out.push({ key: 'name' })
  if (e.type !== 2 && e.type !== 3) out.push({ key: 'eventType' })
  if (!(e.startTime > 0) || e.endTime <= e.startTime) out.push({ key: 'period' })
  if (!e.rewards.length || !e.rewards.every(rewardValid)) out.push({ key: 'eventRewards' })
  return out
}

export function selectProblems(c: LoginBonusConfig, s: LoginBonusSelect): Problem[] {
  const out: Problem[] = []
  if (!s.id) out.push({ key: 'id' })
  else if (c.selects.filter((x) => x.id === s.id).length > 1) out.push({ key: 'idDuplicate' })
  if (!s.name) out.push({ key: 'name' })
  if (!(s.startTime > 0) || s.endTime <= s.startTime) out.push({ key: 'period' })
  if (!s.days.length) out.push({ key: 'selectDays' })
  for (const [i, day] of s.days.entries()) {
    const numbers = day.slots.map((x) => x.number)
    if (!day.slots.length || day.slots.some((x) => !(x.number > 0) || !x.rewards.length || !x.rewards.every(rewardValid)) || new Set(numbers).size !== numbers.length) {
      out.push({ key: 'selectDay', params: { n: i + 1 } })
    }
  }
  return out
}

const DAY = 86_400_000

/**
 * 新的限时 / 自选登录（新建与复制共用）：以 from 为模板（不给时用最近的一个；都没有时从空白开始），
 * 从 start 开始，天数与奖励沿用模板，期间长度不变。加进配置并返回 ID。
 */
export function addLoginBonus(c: LoginBonusConfig, kind: 'event' | 'select', start: number, name: (from?: string) => string, from?: LoginBonusEvent | LoginBonusSelect): string {
  const latest = <T extends { startTime: number }>(list: T[]) => [...list].sort((a, b) => b.startTime - a.startTime)[0]
  if (kind === 'event') {
    const src = (from as LoginBonusEvent | undefined) || latest(c.events)
    const id = freeId((v) => c.events.some((x) => x.id === v), src ? `${src.id}-copy` : 'event-loginbonus-new')
    const e: LoginBonusEvent = src
      ? { ...cloneJson(src), id, name: name(src.name), startTime: start, endTime: start + (src.endTime - src.startTime) }
      : { id, type: 3, name: name(), description: '', titleAssetId: '', backgroundAssetId: '', startLoginBonusMotionId: '', endLoginBonusMotionId: '', openConditionSetId: '', startTime: start, endTime: start + 7 * DAY, rewards: [] }
    c.events.push(e)
    return id
  }
  const src = (from as LoginBonusSelect | undefined) || latest(c.selects)
  const id = freeId((v) => c.selects.some((x) => x.id === v), src ? `${src.id}-copy` : 'select-loginbonus-new')
  const s: LoginBonusSelect = src
    ? { ...cloneJson(src), id, name: name(src.name), startTime: start, endTime: start + (src.endTime - src.startTime) }
    : { id, name: name(), titleAssetId: '', backgroundAssetId: '', startTime: start, endTime: start + 7 * DAY, days: [] }
  c.selects.push(s)
  return id
}
