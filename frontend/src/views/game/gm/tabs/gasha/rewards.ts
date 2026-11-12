// 配置里的奖励（common.Reward，零值字段可省略）与 GM 奖励选择器之间的换算，以及奖励的显示名称与图片。
import { gakumas } from '@/games'
import { gameName } from '@/games/names'
import { getCurrentLocale, i18n } from '@/locales'
import type { CurrencyKind } from '@/games/gakumas/assets'
import type { ConfigReward } from '@/games/gakumas/types'
import type { GmReward } from '../../components/types'
import type { CatalogLookup } from './shared/catalogLookup'

export const toGmRewards = (list: ConfigReward[] | undefined): GmReward[] =>
  (list || []).filter(Boolean).map((r) => ({ resourceType: r.resourceType || 0, resourceId: r.resourceId || '', quantity: r.quantity || 0 }))

/** 选择器结果写回配置：没变的奖励保留原对象（键与零值写法不变），新加的只写非零字段。 */
export function mergeRewards(original: ConfigReward[] | undefined, next: GmReward[]): ConfigReward[] {
  const pool = [...(original || [])]
  return next.map((r) => {
    const i = pool.findIndex((o) => (o.resourceType || 0) === r.resourceType && (o.resourceId || '') === r.resourceId)
    if (i >= 0) {
      const o = pool.splice(i, 1)[0]!
      if ((o.quantity || 0) !== r.quantity) o.quantity = r.quantity
      return o
    }
    return r.resourceId
      ? { resourceType: r.resourceType, resourceId: r.resourceId, quantity: r.quantity }
      : { resourceType: r.resourceType, quantity: r.quantity }
  })
}

const CURRENCY_KINDS = ['jewelFree', 'jewelPaid', 'actionPoint', 'exp']

export interface RewardView {
  name: string
  image: string
  quantity: number
}

/** 奖励的名称与图片：类型按 manifest.resourceTypes 找目录；名称先查前端名称表，再查目录，都没有时为「未命名」（不显示 ID）。 */
export function rewardView(r: ConfigReward, resourceTypes: Record<string, number>, lookup: CatalogLookup): RewardView {
  const type = r.resourceType || 0
  const id = r.resourceId || ''
  const key = Object.keys(resourceTypes).find((k) => resourceTypes[k] === type) || ''
  const quantity = r.quantity || 0
  const unnamed = i18n.global.t('game.unnamed')
  const nameOf = (kind: string, entryId: string, fallback?: string) =>
    gameName('gakumas', getCurrentLocale(), kind, entryId, fallback) || unnamed
  if (CURRENCY_KINDS.includes(key)) {
    const entry = lookup.lookup('currency', key)
    return { name: nameOf('currency', key, entry?.name), image: gakumas.currencyImage(key as CurrencyKind), quantity }
  }
  if (!id) return { name: unnamed, image: '', quantity }
  const entry = key ? lookup.lookup(key, id) : undefined
  let image = ''
  if (key === 'idolCardSkin') image = gakumas.idolCardImage(id)
  else if (key === 'supportCard') image = gakumas.supportCardImage(id)
  else if (key === 'item') image = gakumas.itemImage(entry?.assetId || id)
  return { name: key ? nameOf(key, id, entry?.name) : unnamed, image, quantity }
}
