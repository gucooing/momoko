// 公告配置（data/Notice.json）的计算：状态与本地检查，规则与学マス 服务端 gdconf/data.Notice.go 的加载校验一致。
import type { NoticeConfig, NoticeEntry } from '@/games/gakumas/types'

/** 公告分类与类型（客户端协议枚举 NoticeCategory / NoticeType 的取值）。 */
export const NOTICE_CATEGORIES = [1, 2, 3] as const
export const NOTICE_TYPES = [1, 2, 3, 4, 5, 6, 7, 8] as const

export type NoticeStatus = 'showing' | 'upcoming' | 'ended'
export const NOTICE_STATUSES: NoticeStatus[] = ['showing', 'upcoming', 'ended']

/** 展示中：startTime ≤ 现在，且不结束或还没到 endTime（服务端 NoticeEntry.Active）。 */
export function noticeStatus(n: NoticeEntry, now: number): NoticeStatus {
  if (n.startTime > now) return 'upcoming'
  if (n.endTime && n.endTime <= now) return 'ended'
  return 'showing'
}

export const noticeStatusVariant = (s: NoticeStatus) =>
  ({ showing: 'success', upcoming: 'warning', ended: 'neutral' })[s] as 'success' | 'warning' | 'neutral'

export interface NoticeProblem {
  level: 'bad' | 'warn'
  key: string
  params?: Record<string, string | number>
}

/** 一条公告的问题（bad 会被服务端拒绝；warn 只是提醒）。 */
export function noticeProblems(cfg: NoticeConfig, n: NoticeEntry): NoticeProblem[] {
  const out: NoticeProblem[] = []
  if (!n.id) out.push({ level: 'bad', key: 'idMissing' })
  else if (cfg.notices.filter((x) => x.id === n.id).length > 1) out.push({ level: 'bad', key: 'idDuplicate' })
  if (!(NOTICE_CATEGORIES as readonly number[]).includes(n.category)) out.push({ level: 'bad', key: 'category' })
  if (!(NOTICE_TYPES as readonly number[]).includes(n.type)) out.push({ level: 'bad', key: 'type' })
  if (n.endTime !== 0 && n.endTime <= n.startTime) out.push({ level: 'bad', key: 'period' })
  if (n.lastUpdateTime < n.startTime) out.push({ level: 'bad', key: 'lastUpdate' })
  if (!n.title) out.push({ level: 'warn', key: 'title' })
  if (!n.detailUrl) out.push({ level: 'warn', key: 'detailUrl' })
  else if (!/^https?:\/\//i.test(n.detailUrl)) out.push({ level: 'warn', key: 'detailUrlScheme' })
  return out
}

/** 整份配置的问题（每页条数）。 */
export function configProblems(cfg: NoticeConfig): NoticeProblem[] {
  return cfg.pageSize > 0 ? [] : [{ level: 'bad', key: 'pageSize' }]
}

/**
 * 新公告（复制时沿用模板的全部字段）：ID 自动生成（gm-notice-<时间>，不会和服务端给卡池生成的 notice-<卡池> 重复），界面上不出现；
 * 从现在（到分钟）开始展示。加进配置并返回。
 */
export function addNotice(cfg: NoticeConfig, now: number, title: string, from?: NoticeEntry): NoticeEntry {
  const d = new Date(now)
  const pad = (n: number) => String(n).padStart(2, '0')
  const base = `gm-notice-${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
  let id = base
  for (let i = 2; cfg.notices.some((n) => n.id === id); i++) id = `${base}-${i}`
  const start = Math.floor(now / 60_000) * 60_000
  const entry: NoticeEntry = from
    ? { ...(JSON.parse(JSON.stringify(from)) as NoticeEntry), id, title, startTime: start, lastUpdateTime: start, endTime: from.endTime ? start + (from.endTime - from.startTime) : 0 }
    : { id, title, category: 1, type: 6, startTime: start, endTime: 0, lastUpdateTime: start, bannerAssetId: '', detailUrl: '' }
  cfg.notices.push(entry)
  return entry
}
