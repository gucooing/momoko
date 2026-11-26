// 蓝色星原的奖励种类与名称（管理端与用户通行证共用）：名称表由 scripts/games/azurpromilia/export-names.py 从游戏服务端 res 导出。
// 两端各自用 azurNames(当前语言) 取得按当前语言显示的名称函数。
import { gameNameIds, lookupGameName, searchGameNames } from '@/games/names'
import rewardTypes from './reward-types.json'

export const GAME = 'azurpromilia'

/** 奖励：type 为游戏 reward_type 的数值，id 为该种类里的 ID。 */
export interface AzurReward {
  type: number
  id: number
  num: number
}

/** 有名称表的奖励种类：type 为游戏 reward_type 的数值，kind 为它的表名（名称表的种类）。 */
export const REWARD_KINDS: { type: number; kind: string }[] = rewardTypes

const kindByType = new Map(REWARD_KINDS.map((k) => [k.type, k.kind]))

export const kindOf = (type: number) => kindByType.get(type) || ''

/** 种类表名对应的奖励种类数值（没有名称表的种类为 0）。 */
export const typeOfKind = (kind: string) => REWARD_KINDS.find((k) => k.kind === kind)?.type ?? 0

/** 按当前语言显示的名称函数；名称表里没有时显示 ID。 */
export function azurNames(locale: () => string) {
  const kindName = (kind: string, id: number | string) => (kind && lookupGameName(GAME, locale(), kind, String(id))) || `#${id}`
  const rewardTypeName = (type: number) => lookupGameName(GAME, locale(), 'reward_type', String(type)) || `#${type}`
  const itemName = (type: number, id: number | string) => kindName(kindOf(type), id)
  const rewardText = (r: AzurReward) => `${itemName(r.type, r.id)} ×${r.num}`
  const rewardsText = (rs: AzurReward[] | undefined) => (rs?.length ? rs.map(rewardText).join('、') : '—')

  /** 在一种奖励的名称表里按名称或 ID 搜索，最多 limit 条。 */
  const searchItems = async (type: number, q: string, limit = 30): Promise<{ id: number; name: string }[]> => {
    const kind = kindOf(type)
    if (!kind) return []
    const text = q.trim()
    let ids: string[]
    if (!text) ids = await gameNameIds(GAME, kind)
    else {
      const hits = await searchGameNames(GAME, locale(), kind, text)
      ids = [...hits]
      if (/^\d+$/.test(text) && !hits.has(text) && (await gameNameIds(GAME, kind)).includes(text)) ids.unshift(text)
    }
    return ids.slice(0, limit).map((id) => ({ id: Number(id), name: itemName(type, id) }))
  }

  return { kindName, rewardTypeName, itemName, rewardText, rewardsText, searchItems }
}
