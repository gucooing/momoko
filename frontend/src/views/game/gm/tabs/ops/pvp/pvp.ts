// 竞赛配置（data/PvpRate.json）的本地检查：与服务端加载时的检查相同（赛季不重叠、规则与段位奖励有效），
// 赛季引用的客户端主数据（PvpRateConfig）由服务端校验。
import type { PvpRateConfig, PvpRateGrade, PvpRateSeason } from '@/games/gakumas/types'
import { nextId } from '../shared/entry'
import { rewardValid } from '../shared/rewards'

export type SeasonStatus = 'preparing' | 'open' | 'upcoming' | 'ended'

export function seasonStatus(s: PvpRateSeason, now: number): SeasonStatus {
  if (now >= s.endTime) return 'ended'
  if (now >= s.startTime) return 'open'
  if (now >= s.prepareStartTime) return 'preparing'
  return 'upcoming'
}

export const seasonVariant = (s: SeasonStatus) =>
  ({ open: 'success', preparing: 'info', upcoming: 'warning', ended: 'neutral' })[s] as 'success' | 'info' | 'warning' | 'neutral'

export interface Problem {
  key: string
  params?: Record<string, string | number>
}

/** 按开赛时间排好的赛季（不改配置里的顺序，服务端加载时也会排序）。 */
export const sortedSeasons = (c: PvpRateConfig) => [...c.seasons].sort((a, b) => a.startTime - b.startTime)

export function seasonProblems(c: PvpRateConfig, s: PvpRateSeason): Problem[] {
  const out: Problem[] = []
  if (!s.pvpRateConfigId) out.push({ key: 'seasonId' })
  else if (c.seasons.filter((x) => x.pvpRateConfigId === s.pvpRateConfigId).length > 1) out.push({ key: 'seasonDuplicate' })
  if (!s.prepareStartTime || !s.startTime || !s.endTime) out.push({ key: 'seasonTimes' })
  else if (s.prepareStartTime > s.startTime || s.startTime >= s.endTime) out.push({ key: 'seasonOrder' })
  const list = sortedSeasons(c)
  const i = list.indexOf(s)
  const prev = i > 0 ? list[i - 1] : undefined
  if (prev && prev.endTime > s.prepareStartTime) out.push({ key: 'seasonOverlap', params: { id: prev.pvpRateConfigId } })
  return out
}

export function gradeProblems(g: PvpRateGrade): Problem[] {
  const out: Problem[] = []
  if (g.gradeKeepRate < 0 || (g.gradeUpRate !== 0 && g.gradeUpRate < g.gradeKeepRate)) out.push({ key: 'gradeRates' })
  if (!rewardValid(g.playReward)) out.push({ key: 'gradePlayReward' })
  if (g.winFanCount < 0 || g.loseFanCount < 0) out.push({ key: 'gradeFans' })
  const rates = [...g.rateRewards].map((r) => r.rate)
  if (g.rateRewards.some((r) => !(r.rate > 0) || !rewardValid(r.reward)) || rates.some((r, i) => i > 0 && rates[i - 1]! >= r)) {
    out.push({ key: 'gradeRateRewards' })
  }
  let next = 1
  for (const [j, r] of g.rankRewards.entries()) {
    const last = j === g.rankRewards.length - 1
    if (r.upperRank !== next || (r.lowerRank !== 0 && r.lowerRank < r.upperRank) || (r.lowerRank === 0 && !last) || !r.rewards.length || !r.rewards.every(rewardValid)) {
      out.push({ key: 'gradeRankRewards' })
      break
    }
    next = r.lowerRank + 1
  }
  return out
}

export function ruleProblems(c: PvpRateConfig): Problem[] {
  const out: Problem[] = []
  if (!(c.maxDailyPlayCount > 0)) out.push({ key: 'maxDailyPlayCount' })
  if (!(c.historyLimit > 0)) out.push({ key: 'historyLimit' })
  if (!c.winRates.length || c.winRates.some((r) => ![1, 2, 3].includes(r.rivalType) || !(r.rate > 0) || r.maxRate < r.rate)) out.push({ key: 'winRates' })
  if (!c.loseRates.length) out.push({ key: 'loseRates' })
  return out
}

/** 赛季的显示名：赛季内容（竞赛配置）在游戏数据里的名称，查不到时按 ID 末尾的期数显示「第 N 期」。 */
export function seasonLabel(
  t: (key: string, params?: Record<string, unknown>) => string,
  s: PvpRateSeason,
  name: (kind: string, id: string) => string = () => '',
): string {
  const named = s.pvpRateConfigId ? name('pvpRateConfig', s.pvpRateConfigId) : ''
  if (named) return named
  const m = /(\d+)$/.exec(s.pvpRateConfigId)
  return m ? t('gmConfig.pvp.seasonNo', { n: Number(m[1]) }) : s.pvpRateConfigId || t('game.unnamed')
}

/** 新赛季接在最后一个赛季后面：准备期与赛季长度沿用最后一个赛季，配置 ID 末尾的期数加一。加进配置并返回 ID。 */
export function addSeason(c: PvpRateConfig, now: number): string {
  const last = sortedSeasons(c).pop()
  const id = last ? nextId(last.pvpRateConfigId, (v) => c.seasons.some((s) => s.pvpRateConfigId === v)) : ''
  const prepare = last ? last.endTime + 1000 : now
  const lead = last ? last.startTime - last.prepareStartTime : 0
  const length = last ? last.endTime - last.startTime : 14 * 86_400_000
  c.seasons.push({ pvpRateConfigId: id, prepareStartTime: prepare, startTime: prepare + lead, endTime: prepare + lead + length })
  return id
}
