// 运营日历的数据：把 7 份配置（按正在编辑的内容，也就是发布后的样子）换算成时间线上的条目，
// 和服务端当前内容比出草稿里新增 / 改过 / 删掉的条目，并从数据里算出冲突与提醒。
import { gakumas } from '@/games'
import type { GakumasConfigDomain } from '@/api/gakumas'
import type { GashaConfig, HomeConfig, LoginBonusConfig, NoticeConfig, PvpRateConfig, ResearchConfig } from '@/games/gakumas/types'
import { bannerImage } from './shared/assets'
import { deepEqual } from './shared/diff'
import { FAR_FUTURE } from './shared/entry'

export type LaneKey = 'gasha' | 'pvp' | 'event' | 'login' | 'notice'

/** 泳道与它读取的配置。 */
export const LANES: { key: LaneKey; domains: GakumasConfigDomain[] }[] = [
  { key: 'gasha', domains: ['gasha'] },
  { key: 'event', domains: ['home', 'research', 'storyEvent'] },
  { key: 'pvp', domains: ['pvpRate'] },
  { key: 'login', domains: ['loginBonus'] },
  { key: 'notice', domains: ['notice'] },
]

/** 详情抽屉打开哪一项。login 的 id 是 normal（每日）、total（累计）或限时 / 自选登录的 ID。 */
export type ItemKind = 'gasha' | 'notice' | 'pvp' | 'event' | 'research' | 'login'
export interface ItemRef {
  kind: ItemKind
  id: string
}
export const refKey = (r: ItemRef) => `${r.kind}:${r.id}`
export function parseRef(v: unknown): ItemRef | undefined {
  if (typeof v !== 'string') return undefined
  const i = v.indexOf(':')
  const kind = v.slice(0, i) as ItemKind
  return i > 0 && ['gasha', 'notice', 'pvp', 'event', 'research', 'login'].includes(kind) ? { kind, id: v.slice(i + 1) } : undefined
}

export interface CalItem {
  key: string
  lane: LaneKey
  ref: ItemRef
  title: string
  start: number
  /** 0 = 不结束 */
  end: number
  image: string
  /** 常驻或没有期限 */
  permanent: boolean
  /** 和服务端当前内容比：new 草稿里新增 / changed 改过 / removed 草稿里删掉了 */
  draft?: 'new' | 'changed' | 'removed'
  /** 说明（已翻译） */
  sub: string
}

type T = (key: string, params?: Record<string, unknown>) => string

interface RawItem {
  ref: ItemRef
  title: string
  start: number
  end: number
  image: string
  sub: string
  /** 配置里的原对象（比较草稿改动用） */
  src: unknown
}

/** 一份配置：服务端当前内容与正在编辑的内容（没有改动时相同）。 */
export interface DomainContent {
  published: unknown
  working: unknown
}
export type Contents = Partial<Record<GakumasConfigDomain, DomainContent>>

// ---------------------------------------------------------------------------
// 各配置 → 条目
// ---------------------------------------------------------------------------

function gashaItems(c: GashaConfig | undefined, t: T): RawItem[] {
  return (c?.gashas || []).map((g) => ({
    ref: { kind: 'gasha', id: g.id },
    title: g.name || t('gmConfig.gasha.unnamed'),
    start: g.startTime,
    end: g.endTime,
    image: g.bannerAssetId ? gakumas.gashaBanner(g.bannerAssetId) : '',
    sub: [g.pickupIdolCardSkinIds?.length || g.pickupSupportCardIds?.length ? t('gmConfig.calendar.pickup') : '', g.gashaPointId ? t('gmConfig.calendar.withPoint') : '']
      .filter(Boolean)
      .join(' · '),
    src: g,
  }))
}

function pvpItems(c: PvpRateConfig | undefined, t: T, label: (id: string) => string): RawItem[] {
  return (c?.seasons || []).map((s) => ({
    ref: { kind: 'pvp', id: s.pvpRateConfigId },
    title: label(s.pvpRateConfigId),
    start: s.prepareStartTime || s.startTime,
    end: s.endTime,
    image: '',
    sub: t('gmConfig.calendar.pvpSeason'),
    src: s,
  }))
}

function eventItems(home: HomeConfig | undefined, research: ResearchConfig | undefined, t: T): RawItem[] {
  const out: RawItem[] = []
  const seen = new Set<string>()
  for (const e of home?.events || []) {
    seen.add(e.eventId)
    out.push({
      ref: { kind: 'event', id: e.eventId },
      title: e.eventName || t('game.unnamed'),
      start: e.startTime,
      end: e.endTime,
      image: bannerImage(e.bannerAssetId || ''),
      sub: e.eventType ? t(`gmConfig.calendar.eventTypes.${e.eventType}`) : '',
      src: e,
    })
  }
  for (const r of research?.researches || []) {
    if (seen.has(r.id)) continue
    out.push({
      ref: { kind: 'research', id: r.id },
      title: t('gmConfig.events.researchName'),
      start: r.startTime,
      end: r.endTime,
      image: bannerImage(r.titleAssetId || ''),
      sub: '',
      src: r,
    })
  }
  return out
}

function loginItems(c: LoginBonusConfig | undefined, t: T): RawItem[] {
  if (!c) return []
  const out: RawItem[] = []
  if (c.normal) {
    out.push({
      ref: { kind: 'login', id: 'normal' },
      title: c.normal.name || t('gmConfig.login.normal'),
      start: 0,
      end: 0,
      image: bannerImage(c.normal.titleAssetId || ''),
      sub: t('gmConfig.calendar.loginCycle', { n: c.normal.rewards.length }),
      src: c.normal,
    })
  }
  if (c.total) {
    out.push({
      ref: { kind: 'login', id: 'total' },
      title: c.total.name || t('gmConfig.login.total'),
      start: 0,
      end: 0,
      image: '',
      sub: t('gmConfig.login.total'),
      src: c.total,
    })
  }
  for (const e of c.events || []) {
    out.push({ ref: { kind: 'login', id: e.id }, title: e.name || t('game.unnamed'), start: e.startTime, end: e.endTime, image: bannerImage(e.titleAssetId), sub: t('gmConfig.calendar.loginEvent', { n: e.rewards.length }), src: e })
  }
  for (const s of c.selects || []) {
    out.push({ ref: { kind: 'login', id: s.id }, title: s.name || t('game.unnamed'), start: s.startTime, end: s.endTime, image: bannerImage(s.titleAssetId), sub: t('gmConfig.calendar.loginSelect', { n: s.days.length }), src: s })
  }
  return out
}

function noticeItems(c: NoticeConfig | undefined, t: T): RawItem[] {
  return (c?.notices || []).map((n) => ({
    ref: { kind: 'notice', id: n.id },
    title: n.title || t('gmConfig.notice.untitled'),
    start: n.startTime,
    end: n.endTime,
    image: bannerImage(n.bannerAssetId),
    sub: t(`gmConfig.notice.types.${n.type}`),
    src: n,
  }))
}

function rawOf(lane: LaneKey, pick: <C>(d: GakumasConfigDomain) => C | undefined, t: T, seasonName: (id: string) => string): RawItem[] {
  switch (lane) {
    case 'gasha':
      return gashaItems(pick<GashaConfig>('gasha'), t)
    case 'pvp':
      return pvpItems(pick<PvpRateConfig>('pvpRate'), t, seasonName)
    case 'event':
      return eventItems(pick<HomeConfig>('home'), pick<ResearchConfig>('research'), t)
    case 'login':
      return loginItems(pick<LoginBonusConfig>('loginBonus'), t)
    case 'notice':
      return noticeItems(pick<NoticeConfig>('notice'), t)
  }
}

const isPermanent = (x: RawItem) => !x.end || x.end >= FAR_FUTURE

/** 一条泳道的条目：正在编辑的内容（新增、改过的标出来），加上草稿里删掉的（删除线）。 */
export function laneItems(lane: LaneKey, contents: Contents, t: T, seasonName: (id: string) => string): CalItem[] {
  const working = rawOf(lane, (d) => (contents[d]?.working ?? contents[d]?.published) as never, t, seasonName)
  const published = rawOf(lane, (d) => contents[d]?.published as never, t, seasonName)
  const toItem = (x: RawItem, draft?: CalItem['draft']): CalItem => ({
    key: `${x.ref.kind}:${x.ref.id}:${draft === 'removed' ? 'removed' : ''}`,
    lane,
    ref: x.ref,
    title: x.title,
    start: x.start,
    end: x.end,
    image: x.image,
    sub: x.sub,
    permanent: isPermanent(x),
    draft,
  })
  const out: CalItem[] = []
  for (const x of working) {
    const before = published.find((y) => y.ref.kind === x.ref.kind && y.ref.id === x.ref.id)
    out.push(toItem(x, !before ? 'new' : deepEqual(before.src, x.src) ? undefined : 'changed'))
  }
  for (const y of published) if (!working.some((x) => x.ref.kind === y.ref.kind && x.ref.id === y.ref.id)) out.push(toItem(y, 'removed'))
  return out
}

// ---------------------------------------------------------------------------
// 冲突与提醒（按正在编辑的内容）
// ---------------------------------------------------------------------------

export interface Hint {
  level: 'warn' | 'info'
  key: string
  params: Record<string, string | number>
  /** 点「查看」打开哪一项 */
  ref?: ItemRef
}

const HOUR = 3_600_000
const DAY = 86_400_000

export function computeHints(contents: Contents, now: number, monthStart: number, monthEnd: number, fmt: (ms: number) => string, t: T): Hint[] {
  const out: Hint[] = []
  const pick = <C>(d: GakumasConfigDomain) => (contents[d]?.working ?? contents[d]?.published) as C | undefined
  const gashas = pick<GashaConfig>('gasha')?.gashas || []

  // 同一小时里结束的卡池（3 个以上）
  const ends = new Map<number, string[]>()
  for (const g of gashas) {
    if (!g.endTime || g.endTime < monthStart || g.endTime >= monthEnd) continue
    const hour = Math.floor(g.endTime / HOUR) * HOUR
    ends.set(hour, [...(ends.get(hour) || []), g.name || t('gmConfig.gasha.unnamed')])
  }
  for (const [hour, names] of ends) {
    if (names.length >= 3) out.push({ level: 'warn', key: 'endCluster', params: { time: fmt(hour), n: names.length, names: names.slice(0, 4).join('、') } })
  }

  // 有 UP 的卡池最晚在什么时候结束，之后没有 UP 卡池
  const pickup = gashas.filter((g) => g.pickupIdolCardSkinIds?.length || g.pickupSupportCardIds?.length)
  if (pickup.length && !pickup.some((g) => !g.endTime)) {
    const last = Math.max(...pickup.map((g) => g.endTime))
    if (last >= now && last <= now + 30 * DAY) out.push({ level: 'info', key: 'pickupGap', params: { time: fmt(last) } })
  } else if (gashas.length && !pickup.length) {
    out.push({ level: 'info', key: 'noPickup', params: {} })
  }

  // 交换点没覆盖卡池开放时间
  const points = pick<GashaConfig>('gasha')?.points || []
  for (const g of gashas) {
    if (!g.gashaPointId) continue
    const p = points.find((x) => x.id === g.gashaPointId)
    if (p && (p.startTime > g.startTime || !g.endTime || p.endTime < g.endTime) && (!g.endTime || g.endTime >= now)) {
      out.push({ level: 'warn', key: 'pointCover', params: { name: g.name || t('gmConfig.gasha.unnamed') }, ref: { kind: 'gasha', id: g.id } })
    }
  }

  // 竞赛赛季
  const pvp = pick<PvpRateConfig>('pvpRate')
  if (pvp) {
    const seasons = [...pvp.seasons].map((s) => ({ id: s.pvpRateConfigId, prepare: s.prepareStartTime || s.startTime, start: s.startTime, end: s.endTime })).sort((a, b) => a.start - b.start)
    const current = seasons.find((s) => s.prepare <= now && now < s.end)
    const next = seasons.find((s) => s.prepare > now)
    if (!current && !next) out.push({ level: 'warn', key: 'pvpNone', params: {} })
    else if (current && !next && current.end < monthEnd) out.push({ level: 'info', key: 'pvpLast', params: { time: fmt(current.end) }, ref: { kind: 'pvp', id: current.id } })
    for (let i = 1; i < seasons.length; i++) {
      const a = seasons[i - 1]!
      const b = seasons[i]!
      if (b.prepare > a.end + 1000 && b.prepare >= monthStart && a.end < monthEnd) out.push({ level: 'info', key: 'pvpGap', params: { from: fmt(a.end), to: fmt(b.prepare) }, ref: { kind: 'pvp', id: b.id } })
    }
  }

  // 已经结束但还留在文件里的公告
  const notices = pick<NoticeConfig>('notice')?.notices || []
  const ended = notices.filter((n) => n.endTime && n.endTime <= now).length
  if (ended) out.push({ level: 'info', key: 'noticesEnded', params: { n: ended } })

  return out
}
