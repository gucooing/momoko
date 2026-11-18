// 活动配置的本地检查（与服务端加载时的检查相同；任务组、道具、剧情等引用由服务端核对）与活动类型。
import type { EventPointReward, HomeConfig, HomeEvent, ResearchConfig, ResearchEvent, StoryEvent, StoryEventConfig } from '@/games/gakumas/types'
import { rewardValid } from '../shared/rewards'

/** 活动类型（EventType）：1 每日开放任务 … 15 单元剧情活动。 */
export const EVENT_TYPES = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]

const STORY_CONTENT = ['storyGroupId', 'itemId', 'missionGroupId', 'dailyMissionGroupId', 'missionDailyReleaseGroupId']

/** 每种活动类型在活动日程里要设置的内容（任务、剧情、道具）；研究、试炼、记忆重制的内容在各自的配置里。 */
export const EVENT_CONTENT_FIELDS: Record<number, string[]> = {
  1: ['missionDailyReleaseGroupId'],
  2: ['missionPanelSheetGroupId'],
  3: ['missionGroupId', 'dailyMissionGroupId', 'missionDailyReleaseGroupId'],
  4: ['storyGroupId', 'characterIds'],
  5: STORY_CONTENT,
  6: STORY_CONTENT,
  7: STORY_CONTENT,
  8: STORY_CONTENT,
  9: ['storyGroupId', 'missionGroupId'],
  10: ['dearnessMissionPanelSheetGroupIds', 'characterIds'],
  11: ['playItemId', 'missionGroupId'],
  15: STORY_CONTENT,
}

/** 内容在 StoryEvent.json 里的活动类型。 */
export const STORY_EVENT_TYPES = [5, 6, 7, 8, 15]
/** 巡演（Tour）：客户端要求有游玩道具。 */
const TOUR = 11
/** 研究活动。 */
export const RESEARCH_TYPE = 12

export interface Problem {
  level: 'bad' | 'warn'
  key: string
  params?: Record<string, string | number>
}

export function homeEventProblems(c: HomeConfig, e: HomeEvent): Problem[] {
  const out: Problem[] = []
  if (!e.eventId) out.push({ level: 'bad', key: 'idMissing' })
  else if (c.events.filter((x) => x.eventId === e.eventId).length > 1) out.push({ level: 'bad', key: 'idDuplicate' })
  if (!(e.endTime > e.startTime)) out.push({ level: 'bad', key: 'period' })
  if (e.eventType === TOUR && !e.playItemId) out.push({ level: 'bad', key: 'tourItem' })
  if (!e.eventName) out.push({ level: 'warn', key: 'name' })
  if (e.closeTime && e.closeTime < e.endTime) out.push({ level: 'warn', key: 'closeBeforeEnd' })
  return out
}

/** 积分奖励：积分从小到大，每档都有奖励；只有最后一档可以重复且要有重复间隔。minPoint 是第一档的最小积分。 */
export function pointRewardProblems(list: EventPointReward[], minPoint: number): Problem[] {
  let last = minPoint - 1
  for (const [i, r] of list.entries()) {
    if (r.point <= last || !rewardValid(r.reward)) return [{ level: 'bad', key: 'points', params: { n: i + 1 } }]
    if (r.repeat !== !!(r.repeatPoint && r.repeatPoint > 0) || (r.repeat && i !== list.length - 1)) return [{ level: 'bad', key: 'pointsRepeat' }]
    last = r.point
  }
  return []
}

export function researchProblems(c: ResearchConfig, r: ResearchEvent): Problem[] {
  const out: Problem[] = []
  if (!r.id) out.push({ level: 'bad', key: 'idMissing' })
  else if (c.researches.filter((x) => x.id === r.id).length > 1) out.push({ level: 'bad', key: 'idDuplicate' })
  if (!(r.startTime > 0) || !(r.endTime > r.startTime) || r.closeTime < r.endTime) out.push({ level: 'bad', key: 'researchPeriod' })
  if (!r.eventBonus) out.push({ level: 'bad', key: 'eventBonus' })
  out.push(...pointRewardProblems(r.pointRewards || [], 1))
  return out
}

export function storyProblems(c: StoryEventConfig, s: StoryEvent, home: HomeEvent[] | undefined): Problem[] {
  const out: Problem[] = []
  if (!s.storyEventId) out.push({ level: 'bad', key: 'idMissing' })
  else if (c.storyEvents.filter((x) => x.storyEventId === s.storyEventId).length > 1) out.push({ level: 'bad', key: 'idDuplicate' })
  out.push(...pointRewardProblems(s.pointRewards || [], 0))
  if (home && s.storyEventId && !home.some((e) => e.eventId === s.storyEventId)) out.push({ level: 'warn', key: 'storyNoSchedule' })
  return out
}
