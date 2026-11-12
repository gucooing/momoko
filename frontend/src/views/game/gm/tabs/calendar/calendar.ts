// 运营日历的数据：只从这台服务器 manifest 里有的配置域读取（卡池、竞赛赛季、活动、登录奖励、公告），
// 每个配置域在一次访问里只读一次。这里把配置内容换算成时间线上的条目，并从数据里算出冲突与提醒。
import { gakumas } from '@/games'
import { i18n } from '@/locales'
import type { GashaConfig, NoticeConfig } from '@/games/gakumas/types'
import { bannerImage } from '../gasha/shared/assets'
import { isPlainObject } from '../gasha/shared/diff'

export type LaneKey = 'gasha' | 'pvp' | 'event' | 'login' | 'notice'

/** 泳道与它读取的配置域（manifest 里有其中任何一个才显示这条泳道）。 */
export const LANES: { key: LaneKey; domains: string[] }[] = [
  { key: 'gasha', domains: ['gasha'] },
  { key: 'pvp', domains: ['pvpRate'] },
  { key: 'event', domains: ['home', 'research', 'storyEvent'] },
  { key: 'login', domains: ['loginBonus'] },
  { key: 'notice', domains: ['notice'] },
]

/** 结束时间晚于这个时刻视为常驻（Home.json 里常驻的任务活动写的是 2100 年）。 */
const FAR_FUTURE = Date.UTC(2090, 0, 1)

export interface CalItem {
  key: string
  lane: LaneKey
  id: string
  title: string
  start: number
  /** 0 = 不结束 */
  end: number
  image: string
  /** 常驻或没有期限 */
  permanent: boolean
  /** 草稿里的状态：new 新增 / changed 时间改了（草稿里的样子）/ removed 草稿里删掉了 */
  draft?: 'new' | 'changed' | 'removed'
  /** 说明（已翻译） */
  sub: string
  /** 点开去哪个标签 */
  link?: { tab: string; query: Record<string, string> }
  source: string
}

type T = (key: string, params?: Record<string, unknown>) => string

interface RawItem {
  id: string
  title: string
  start: number
  end: number
  image: string
  sub: string
  link?: CalItem['link']
}

const num = (v: unknown) => (typeof v === 'number' && Number.isFinite(v) ? v : 0)
const str = (v: unknown) => (typeof v === 'string' ? v : '')
const list = (v: unknown): Record<string, unknown>[] => (Array.isArray(v) ? v.filter(isPlainObject) : [])

// ---------------------------------------------------------------------------
// 各配置域 → 条目
// ---------------------------------------------------------------------------

function gashaItems(content: unknown, t: T): RawItem[] {
  const c = content as Partial<GashaConfig>
  return (c?.gashas || []).map((g) => ({
    id: g.id,
    title: g.name || t('gmConfig.gasha.unnamed'),
    start: g.startTime,
    end: g.endTime,
    image: g.bannerAssetId ? gakumas.gashaBanner(g.bannerAssetId) : '',
    sub: [g.pickupIdolCardSkinIds?.length || g.pickupSupportCardIds?.length ? t('gmConfig.calendar.pickup') : '', g.gashaPointId ? t('gmConfig.calendar.withPoint') : '']
      .filter(Boolean)
      .join(' · '),
    link: { tab: 'gasha', query: { gasha: g.id } },
  }))
}

function pvpItems(content: unknown, t: T, fmt: (ms: number) => string): RawItem[] {
  const c = isPlainObject(content) ? content : {}
  return list(c.seasons).map((s) => ({
    id: str(s.pvpRateConfigId),
    title: t('gmConfig.calendar.pvpSeason'),
    start: num(s.startTime),
    end: num(s.endTime),
    image: '',
    sub: num(s.prepareStartTime) ? t('gmConfig.calendar.pvpPrepare', { time: fmt(num(s.prepareStartTime)) }) : '',
  }))
}

function eventItems(home: unknown, research: unknown, t: T, fmt: (ms: number) => string): RawItem[] {
  const out: RawItem[] = []
  const seen = new Set<string>()
  const h = isPlainObject(home) ? home : {}
  for (const e of list(h.events)) {
    const id = str(e.eventId)
    seen.add(id)
    const type = num(e.eventType)
    const close = num(e.closeTime)
    const end = num(e.endTime)
    out.push({
      id,
      title: str(e.eventName) || t('game.unnamed'),
      start: num(e.startTime),
      end,
      image: bannerImage(str(e.bannerAssetId)),
      sub: [type ? t(`gmConfig.calendar.eventTypes.${type}`) : '', close > end && end < FAR_FUTURE ? t('gmConfig.calendar.closeAt', { time: fmt(close) }) : '']
        .filter(Boolean)
        .join(' · '),
    })
  }
  const r = isPlainObject(research) ? research : {}
  for (const e of list(r.researches)) {
    const id = str(e.id)
    if (seen.has(id)) continue
    out.push({
      id,
      title: t('gmConfig.calendar.eventTypes.12'),
      start: num(e.startTime),
      end: num(e.endTime),
      image: bannerImage(str(e.titleAssetId)),
      sub: '',
    })
  }
  return out
}

function loginItems(content: unknown, t: T): RawItem[] {
  const c = isPlainObject(content) ? content : {}
  const out: RawItem[] = []
  const normal = isPlainObject(c.normal) ? c.normal : undefined
  if (normal) {
    out.push({
      id: 'normal',
      title: str(normal.name) || t('gmConfig.calendar.loginNormal'),
      start: 0,
      end: 0,
      image: bannerImage(str(normal.titleAssetId)),
      sub: t('gmConfig.calendar.loginCycle', { n: list(normal.rewards).length }),
    })
  }
  const total = isPlainObject(c.total) ? c.total : undefined
  if (total) {
    const days = list(total.rewards).map((r) => num(r.days))
    out.push({
      id: 'total',
      title: str(total.name) || t('gmConfig.calendar.loginTotalName'),
      start: 0,
      end: 0,
      image: '',
      sub: t('gmConfig.calendar.loginTotal', {
        days: days.join(', ') || '—',
        from: num(total.repeatFromDays),
        every: num(total.repeatIntervalDays),
      }),
    })
  }
  for (const e of [...list(c.events), ...list(c.selects)]) {
    out.push({
      id: str(e.id),
      title: str(e.name) || t('game.unnamed'),
      start: num(e.startTime),
      end: num(e.endTime),
      image: bannerImage(str(e.titleAssetId)),
      sub: Array.isArray(e.days) ? t('gmConfig.calendar.loginSelect', { n: (e.days as unknown[]).length }) : t('gmConfig.calendar.loginEvent', { n: list(e.rewards).length }),
    })
  }
  return out
}

function noticeItems(content: unknown, t: T): RawItem[] {
  const c = content as Partial<NoticeConfig>
  return (c?.notices || []).map((n) => ({
    id: n.id,
    title: n.title || t('gmConfig.notice.untitled'),
    start: n.startTime,
    end: n.endTime,
    image: bannerImage(n.bannerAssetId),
    sub: t(`gmConfig.notice.types.${n.type}`),
    link: { tab: 'notices', query: { notice: n.id } },
  }))
}

// ---------------------------------------------------------------------------
// 合并已发布与草稿
// ---------------------------------------------------------------------------

export interface DomainContent {
  domain: string
  title: string
  published: unknown
  /** 草稿内容（没有草稿时为 undefined） */
  draft?: unknown
  draftAuthor: string
  draftTime: number
}

export interface BuildInput {
  domains: Map<string, DomainContent>
  t: T
  fmt: (ms: number) => string
}

const content = (d: DomainContent | undefined, draft: boolean) => (d ? (draft ? d.draft : d.published) : undefined)

function rawOf(lane: LaneKey, input: BuildInput, draft: boolean): RawItem[] {
  const { domains, t, fmt } = input
  const pick = (id: string) => {
    const d = domains.get(id)
    // 没有草稿的配置域在「草稿」视角下就是已发布内容
    return content(d, draft && d?.draft !== undefined)
  }
  switch (lane) {
    case 'gasha':
      return domains.has('gasha') ? gashaItems(pick('gasha'), t) : []
    case 'pvp':
      return domains.has('pvpRate') ? pvpItems(pick('pvpRate'), t, fmt) : []
    case 'event':
      return eventItems(pick('home'), pick('research'), t, fmt)
    case 'login':
      return domains.has('loginBonus') ? loginItems(pick('loginBonus'), t) : []
    case 'notice':
      return domains.has('notice') ? noticeItems(pick('notice'), t) : []
  }
}

const laneSource: Record<LaneKey, string> = { gasha: 'gasha', pvp: 'pvpRate', event: 'home', login: 'loginBonus', notice: 'notice' }

const isPermanent = (x: RawItem) => !x.end || x.end >= FAR_FUTURE

/** 一条泳道的条目：已发布的，加上草稿里新增 / 改了时间的（虚线），草稿里删掉的标出来。 */
export function laneItems(lane: LaneKey, input: BuildInput): CalItem[] {
  const published = rawOf(lane, input, false)
  const hasDraft = LANES.find((l) => l.key === lane)!.domains.some((d) => input.domains.get(d)?.draft !== undefined)
  const drafted = hasDraft ? rawOf(lane, input, true) : published
  const out: CalItem[] = []
  const toItem = (x: RawItem, draft?: CalItem['draft']): CalItem => ({
    ...x,
    key: `${lane}:${x.id}:${draft || ''}`,
    lane,
    permanent: isPermanent(x),
    draft,
    source: laneSource[lane],
  })
  for (const x of published) {
    const d = drafted.find((y) => y.id === x.id)
    out.push(toItem(x, hasDraft && !d ? 'removed' : undefined))
    if (d && (d.start !== x.start || d.end !== x.end || d.title !== x.title)) out.push(toItem(d, 'changed'))
  }
  if (hasDraft) for (const d of drafted) if (!published.some((x) => x.id === d.id)) out.push(toItem(d, 'new'))
  return out
}

// ---------------------------------------------------------------------------
// 冲突与提醒（只用已发布的内容）
// ---------------------------------------------------------------------------

export interface Hint {
  level: 'warn' | 'info'
  key: string
  params: Record<string, string | number>
  link?: CalItem['link']
}

const HOUR = 3_600_000
const DAY = 86_400_000

export function computeHints(domains: Map<string, DomainContent>, now: number, monthStart: number, monthEnd: number, fmt: (ms: number) => string): Hint[] {
  const out: Hint[] = []
  const gasha = domains.get('gasha')?.published as Partial<GashaConfig> | undefined
  const gashas = gasha?.gashas || []

  // 同一小时里结束的卡池（3 个以上）
  const ends = new Map<number, string[]>()
  for (const g of gashas) {
    if (!g.endTime || g.endTime < monthStart || g.endTime >= monthEnd) continue
    const hour = Math.floor(g.endTime / HOUR) * HOUR
    ends.set(hour, [...(ends.get(hour) || []), g.name || i18n.global.t('gmConfig.gasha.unnamed')])
  }
  for (const [hour, names] of ends) {
    if (names.length >= 3) out.push({ level: 'warn', key: 'endCluster', params: { time: fmt(hour), n: names.length, names: names.slice(0, 4).join('、') } })
  }

  // 有 UP 的卡池最晚在什么时候结束，之后没有 UP 卡池
  const pickup = gashas.filter((g) => g.pickupIdolCardSkinIds?.length || g.pickupSupportCardIds?.length)
  if (pickup.length && !pickup.some((g) => !g.endTime)) {
    const last = Math.max(...pickup.map((g) => g.endTime))
    if (last >= now && last <= now + 30 * DAY) out.push({ level: 'info', key: 'pickupGap', params: { time: fmt(last) }, link: { tab: 'gasha', query: {} } })
  } else if (gasha && !pickup.length) {
    out.push({ level: 'info', key: 'noPickup', params: {} })
  }

  // 交换点没覆盖卡池开放时间
  for (const g of gashas) {
    if (!g.gashaPointId) continue
    const p = gasha?.points?.find((x) => x.id === g.gashaPointId)
    if (p && (p.startTime > g.startTime || !g.endTime || p.endTime < g.endTime) && (!g.endTime || g.endTime >= now)) {
      out.push({ level: 'warn', key: 'pointCover', params: { name: g.name || i18n.global.t('gmConfig.gasha.unnamed') }, link: { tab: 'gasha', query: { gasha: g.id } } })
    }
  }

  // 竞赛赛季
  const pvp = domains.get('pvpRate')?.published
  if (isPlainObject(pvp)) {
    const seasons = list(pvp.seasons)
      .map((s) => ({ id: str(s.pvpRateConfigId), prepare: num(s.prepareStartTime) || num(s.startTime), start: num(s.startTime), end: num(s.endTime) }))
      .sort((a, b) => a.start - b.start)
    const current = seasons.find((s) => s.prepare <= now && now < s.end)
    const next = seasons.find((s) => s.prepare > now)
    if (!current && !next) out.push({ level: 'warn', key: 'pvpNone', params: {} })
    else if (current && !next && current.end < monthEnd) out.push({ level: 'info', key: 'pvpLast', params: { time: fmt(current.end) } })
    for (let i = 1; i < seasons.length; i++) {
      const a = seasons[i - 1]!
      const b = seasons[i]!
      if (b.prepare > a.end && b.prepare >= monthStart && a.end < monthEnd) out.push({ level: 'info', key: 'pvpGap', params: { from: fmt(a.end), to: fmt(b.prepare) } })
    }
  }

  // 研究活动 / 剧情活动没有积分奖励
  const research = domains.get('research')?.published
  if (isPlainObject(research)) {
    for (const r of list(research.researches)) {
      const end = num(r.endTime)
      if (!list(r.pointRewards).length && end >= monthStart && num(r.startTime) < monthEnd) {
        out.push({ level: 'info', key: 'researchNoRewards', params: { time: fmt(end) } })
      }
    }
  }
  const story = domains.get('storyEvent')?.published
  const home = domains.get('home')?.published
  if (isPlainObject(story)) {
    const events = isPlainObject(home) ? list(home.events) : []
    for (const s of list(story.storyEvents)) {
      const id = str(s.storyEventId)
      const ev = events.find((e) => str(e.eventId) === id)
      if (!list(s.pointRewards).length && ev && num(ev.endTime) >= monthStart && num(ev.startTime) < monthEnd) {
        out.push({ level: 'info', key: 'storyNoRewards', params: { name: str(ev.eventName) || i18n.global.t('game.unnamed') } })
      }
    }
  }

  // 已经结束但还留在文件里的公告
  const notice = domains.get('notice')?.published as Partial<NoticeConfig> | undefined
  const ended = (notice?.notices || []).filter((n) => n.endTime && n.endTime <= now).length
  if (ended) out.push({ level: 'info', key: 'noticesEnded', params: { n: ended }, link: { tab: 'notices', query: {} } })

  return out
}
