// 卡池编辑器里的显示名：卡池、抽选表、卡组、抽卡按钮、卡在界面上都只显示名称，ID 只写进配置、只用来调接口。
// 抽选表和卡组在配置里没有名称：抽选表按「哪个卡池的哪一栏在用」称呼，卡组按「种类 + 稀有度（同类有多个时加序号）」称呼。
import { useI18n } from 'vue-i18n'
import type { GashaConfig, GashaDef, GashaPool } from '@/games/gakumas/types'
import { lookupGameName } from '@/games/names'
import { getCurrentLocale } from '@/locales'
import { useGmContext, type GmContext } from '../../context'
import { catalogKindOf, rarityLabel, RESOURCE_IDOL, RESOURCE_SUPPORT, type Problem } from './gasha'

export interface GashaLabels {
  /** 卡池名称（没有名称时为「未命名卡池」） */
  gasha: (g: GashaDef | undefined) => string
  /** 抽选表：self 是正在编辑的卡池时，它自己用的抽选表只显示用途（通常枠 / 确定枠 / 按钮…） */
  lottery: (cfg: GashaConfig, id: string, self?: GashaDef) => string
  /** 卡组 */
  pool: (cfg: GashaConfig, pool: GashaPool) => string
  /** 抽卡按钮（客户端主数据 GashaButton 的名称） */
  button: (id: string) => string
  /** 卡（偶像卡皮肤 / 支援卡） */
  card: (resourceType: number, id: string) => string
  /** 卡的种类：P 偶像卡 / 支援卡 */
  kind: (resourceType: number | undefined) => string
  /** 检查结果的文字（params 里的抽选表、卡、按钮换成名称） */
  problem: (p: Problem, cfg?: GashaConfig, self?: GashaDef) => string
  /** 配置改动里引用的名称（ConfigDiffView 的 refLabel）：依次在 cfgs（草稿、已发布）里找；认不出时为空串 */
  ref: (cfgs: (GashaConfig | undefined)[], collection: string, id: string) => string
}

// 奖励 resourceId 可能是卡、道具、服装…：ID 在这些名称表里不重复，挨个查
const RESOURCE_KINDS = ['idolCardSkin', 'supportCard', 'item', 'costume', 'costumeHead', 'music', 'character']

type Translate = ReturnType<typeof useI18n>['t']

export function createGashaLabels(t: Translate, ctx: GmContext): GashaLabels {
  const gasha = (g: GashaDef | undefined) => g?.name || t('gmConfig.gasha.unnamed')
  const button = (id: string) => ctx.label('gashaButton', id)
  const card = (resourceType: number, id: string) => ctx.label(catalogKindOf(resourceType), id)
  const kind = (resourceType: number | undefined) =>
    resourceType ? t(`gmConfig.gasha.picker.${catalogKindOf(resourceType)}`) : t('gmConfig.gasha.unknownKind')

  // 抽选表在某个卡池里的用途；没用到时为空串
  const roleIn = (g: GashaDef, id: string): string => {
    const slot = (normal: string | undefined) => (normal === id ? t('gmConfig.gasha.normalSlot') : t('gmConfig.gasha.fixSlot'))
    if (g.lotteryId === id) return t('gmConfig.gasha.normalSlot')
    if (g.fixLotteryId === id) return t('gmConfig.gasha.fixSlot')
    for (const b of g.buttons || []) {
      if (b.lotteryId === id || b.fixLotteryId === id) return `${button(b.id)} · ${slot(b.lotteryId)}`
    }
    for (const s of [...(g.stepUp?.steps || []), ...(g.stepUpSelectPickup?.steps || [])]) {
      if (s.lotteryId === id || s.fixLotteryId === id) return `STEP ${s.number} · ${slot(s.lotteryId)}`
    }
    if (g.continuous?.guaranteeLotteryId === id) return t('gmConfig.gasha.guaranteeLottery')
    return ''
  }

  const lottery = (cfg: GashaConfig, id: string, self?: GashaDef) => {
    if (self) {
      const role = roleIn(self, id)
      if (role) return role
    }
    for (const g of cfg.gashas) {
      const role = roleIn(g, id)
      if (role) return `${gasha(g)} · ${role}`
    }
    const i = cfg.lotteries.findIndex((l) => l.id === id)
    return i >= 0 ? t('gmConfig.gasha.unusedLottery', { n: i + 1 }) : t('gmConfig.gasha.missingLottery')
  }

  const pool = (cfg: GashaConfig, p: GashaPool) => {
    const same = cfg.pools.filter((x) => x.resourceType === p.resourceType && x.rarity === p.rarity)
    const base = t('gmConfig.gasha.poolName', { kind: kind(p.resourceType), rarity: rarityLabel(p.rarity) })
    return same.length > 1 ? `${base} ${same.indexOf(p) + 1}` : base
  }

  const problem = (p: Problem, cfg?: GashaConfig, self?: GashaDef) => {
    const params: Record<string, string | number> = { ...(p.params || {}) }
    if (typeof params.lottery === 'string') params.lottery = cfg ? lottery(cfg, params.lottery, self) : ''
    if (typeof params.card === 'string') params.card = card(Number(params.resourceType), params.card)
    if (typeof params.button === 'string') params.button = button(params.button)
    return t(`gmConfig.checks.${p.key}`, params)
  }

  const resource = (id: string) => {
    for (const k of RESOURCE_KINDS) {
      const name = lookupGameName(ctx.game.value, getCurrentLocale(), k, id)
      if (name) return name
    }
    return ''
  }

  const ref = (cfgs: (GashaConfig | undefined)[], collection: string, id: string) => {
    const list = cfgs.filter((c): c is GashaConfig => !!c)
    const find = <T>(pick: (c: GashaConfig) => T | undefined): [GashaConfig, T] | undefined => {
      for (const c of list) {
        const v = pick(c)
        if (v) return [c, v]
      }
      return undefined
    }
    switch (collection) {
      case 'gashas':
      case 'gashaId': {
        const hit = find((c) => c.gashas.find((g) => g.id === id))
        return hit ? gasha(hit[1]) : ''
      }
      case 'lotteries':
      case 'lotteryId':
      case 'fixLotteryId':
      case 'guaranteeLotteryId': {
        const hit = find((c) => (c.lotteries.some((l) => l.id === id) ? c : undefined))
        return hit ? lottery(hit[0], id) : ''
      }
      case 'pools':
      case 'poolIds': {
        const hit = find((c) => c.pools.find((p) => p.id === id))
        return hit ? pool(hit[0], hit[1]) : ''
      }
      case 'points':
      case 'gashaPointId':
        return find((c) => c.points.find((p) => p.id === id))?.[1].name || ''
      case 'buttons':
      case 'gashaButtonId':
        return button(id)
      case 'pickupIdolCardSkinIds':
      case 'idolCardSkinIds':
      case 'idolCardSkinId':
        return card(RESOURCE_IDOL, id)
      case 'pickupSupportCardIds':
      case 'supportCardIds':
      case 'supportCardId':
        return card(RESOURCE_SUPPORT, id)
      case 'resourceId':
        return resource(id)
      default:
        return ''
    }
  }

  return { gasha, lottery, pool, button, card, kind, problem, ref }
}

/** 卡池页子组件里用（卡池页自己用 createGashaLabels）。 */
export function useGashaLabels(): GashaLabels {
  const { t } = useI18n()
  return createGashaLabels(t, useGmContext())
}
