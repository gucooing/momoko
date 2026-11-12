// 卡池配置（data/Gasha.json）的计算：状态、抽选表概率、UP、发布前检查与模拟抽卡。
// 规则与学マス 服务端 gdconf/data.Gasha.go 的加载校验一致：抽选表各槽位合计 100000（十万分率），
// UP 从同资源类型同稀有度的槽位里拿出自己的概率，槽位里其余的卡平分剩下的概率。
import type {
  ConfigReward,
  GashaConfig,
  GashaDef,
  GashaLottery,
  GashaPoint,
  GashaPool,
  GashaSlot,
} from '@/games/gakumas/types'

export const PROBABILITY_TOTAL = 100000
export const RESOURCE_SUPPORT = 4
export const RESOURCE_IDOL = 5

export type GashaStatus = 'open' | 'permanent' | 'upcoming' | 'ended'
export const GASHA_STATUSES: GashaStatus[] = ['open', 'permanent', 'upcoming', 'ended']

/** 卡池状态：按开始 / 结束时间与服务器时间判断；endTime 为 0 的是常驻。 */
export function gashaStatus(g: { startTime: number; endTime: number }, now: number): GashaStatus {
  if (g.startTime > now) return 'upcoming'
  if (!g.endTime) return 'permanent'
  if (g.endTime <= now) return 'ended'
  return 'open'
}

export const statusVariant = (s: GashaStatus) =>
  ({ open: 'success', permanent: 'info', upcoming: 'warning', ended: 'neutral' })[s] as
    | 'success'
    | 'info'
    | 'warning'
    | 'neutral'

/** 十万分率 → 百分比文字（3 位小数）。 */
export const percent = (p: number) => (p / 1000).toFixed(3)

export const catalogKindOf = (resourceType: number) => (resourceType === RESOURCE_IDOL ? 'idolCardSkin' : 'supportCard')

export const rarityLabel = (r: number | undefined) => (r === 3 ? 'SSR' : r === 2 ? 'SR' : r === 1 ? 'R' : '?')

// ---------------------------------------------------------------------------
// 抽选表
// ---------------------------------------------------------------------------

export interface SlotInfo {
  index: number
  slot: GashaSlot
  pools: GashaPool[]
  /** 引用了不存在的卡组 */
  missingPools: string[]
  /** 卡组的资源类型与稀有度不一致 */
  mixed: boolean
  resourceType?: number
  rarity?: number
  /** 槽位里的卡（去重） */
  cardIds: string[]
}

export function slotInfo(cfg: GashaConfig, slot: GashaSlot, index: number): SlotInfo {
  const pools: GashaPool[] = []
  const missingPools: string[] = []
  let mixed = false
  let resourceType: number | undefined
  let rarity: number | undefined
  const cards = new Set<string>()
  for (const id of slot.poolIds || []) {
    const p = cfg.pools.find((x) => x.id === id)
    if (!p) {
      missingPools.push(id)
      continue
    }
    if (resourceType === undefined) {
      resourceType = p.resourceType
      rarity = p.rarity
    } else if (p.resourceType !== resourceType || p.rarity !== rarity) {
      mixed = true
    }
    pools.push(p)
    for (const c of p.ids || []) cards.add(c)
  }
  return { index, slot, pools, missingPools, mixed, resourceType, rarity, cardIds: [...cards] }
}

export const lotteryById = (cfg: GashaConfig, id: string) => cfg.lotteries.find((l) => l.id === id)

export const lotteryTotal = (l: GashaLottery) => (l.slots || []).reduce((sum, s) => sum + (s.probability || 0), 0)

/** 卡池用到的抽选表 ID（卡池本身、按钮 / 阶梯的覆盖、连抽保底），去重，保持出现顺序。 */
export function lotteriesOf(g: GashaDef): string[] {
  const out: string[] = []
  const add = (id: string | undefined) => {
    if (id && !out.includes(id)) out.push(id)
  }
  add(g.lotteryId)
  add(g.fixLotteryId)
  for (const b of g.buttons || []) {
    add(b.lotteryId)
    add(b.fixLotteryId)
  }
  for (const s of [...(g.stepUp?.steps || []), ...(g.stepUpSelectPickup?.steps || [])]) {
    add(s.lotteryId)
    add(s.fixLotteryId)
  }
  add(g.continuous?.guaranteeLotteryId)
  return out
}

/** 还用到这个抽选表的其他卡池。 */
export function otherLotteryUsers(cfg: GashaConfig, lotteryId: string, self: string): GashaDef[] {
  return cfg.gashas.filter((g) => g.id !== self && lotteriesOf(g).includes(lotteryId))
}

/** 还用到这个交换点的其他卡池。 */
export function otherPointUsers(cfg: GashaConfig, pointId: string, self: string): GashaDef[] {
  return cfg.gashas.filter((g) => g.id !== self && g.gashaPointId === pointId)
}

/** 卡在卡组里的稀有度（不在任何卡组里返回 undefined）。 */
export function rarityInPools(cfg: GashaConfig, resourceType: number, id: string): number | undefined {
  for (const p of cfg.pools) if (p.resourceType === resourceType && (p.ids || []).includes(id)) return p.rarity
  return undefined
}

/** 不重复的新 ID：base、base-2、base-3… */
export function freeId(taken: (id: string) => boolean, base: string): string {
  if (!taken(base)) return base
  for (let i = 2; ; i++) if (!taken(`${base}-${i}`)) return `${base}-${i}`
}

// ---------------------------------------------------------------------------
// 抽选表展开（与服务端 WithPickups 相同）
// ---------------------------------------------------------------------------

export interface Bucket {
  resourceType: number
  rarity: number
  /** 十万分率（可能是小数：槽位剩余概率平分给其余的卡） */
  probability: number
  /** UP 卡 ID；槽位里其余的卡合成一项时为空 */
  pickupId?: string
  /** 这一项有几张卡 */
  cards: number
}

/**
 * 一条检查结果。params 里的引用在显示时换成名称（labels.ts 的 problem）：
 * lottery = 抽选表 ID，card + resourceType = 卡，button = 按钮 ID；界面上不显示这些 ID。
 */
export interface Problem {
  key: string
  params?: Record<string, string | number>
}

export interface Prepared {
  buckets: Bucket[]
  problems: Problem[]
}

/** 展开抽选表；rarityOf 给出 UP 卡的稀有度（查不到时返回 undefined）。 */
export function prepareLottery(
  cfg: GashaConfig,
  lottery: GashaLottery,
  rarityOf: (resourceType: number, id: string) => number | undefined,
): Prepared {
  const problems: Problem[] = []
  const buckets: Bucket[] = []
  const slots = (lottery.slots || []).map((s, i) => slotInfo(cfg, s, i))
  const seen = new Set<string>()
  for (const s of slots) {
    if (s.missingPools.length) problems.push({ key: 'poolMissing', params: { lottery: lottery.id, n: s.missingPools.length } })
    if (s.mixed) problems.push({ key: 'slotMixed', params: { lottery: lottery.id, n: s.index + 1 } })
    if (!(s.slot.probability > 0)) problems.push({ key: 'slotZero', params: { lottery: lottery.id, n: s.index + 1 } })
    const sig = `${s.resourceType}:${s.rarity}`
    if (s.resourceType !== undefined && seen.has(sig)) {
      problems.push({ key: 'slotDuplicate', params: { lottery: lottery.id, rarity: rarityLabel(s.rarity) } })
    }
    seen.add(sig)
  }
  const used = new Set<number>()
  for (const s of slots) {
    if (s.resourceType === undefined) continue
    let rest = s.slot.probability || 0
    const picked = new Set<string>()
    ;(lottery.pickups || []).forEach((p, i) => {
      if (p.resourceType !== s.resourceType) return
      const r = rarityOf(p.resourceType, p.resourceId)
      if (r !== s.rarity) return
      used.add(i)
      if (picked.has(p.resourceId)) {
        problems.push({ key: 'pickupDuplicate', params: { lottery: lottery.id, card: p.resourceId, resourceType: p.resourceType } })
        return
      }
      picked.add(p.resourceId)
      rest -= p.probability || 0
      buckets.push({ resourceType: s.resourceType!, rarity: s.rarity!, probability: p.probability || 0, pickupId: p.resourceId, cards: 1 })
    })
    const others = s.cardIds.filter((id) => !picked.has(id)).length
    if (rest < 0 || (rest > 0 && others === 0)) {
      problems.push({ key: 'pickupExceed', params: { lottery: lottery.id, rarity: rarityLabel(s.rarity) } })
    }
    if (rest > 0 && others > 0) buckets.push({ resourceType: s.resourceType, rarity: s.rarity!, probability: rest, cards: others })
  }
  ;(lottery.pickups || []).forEach((p, i) => {
    if (!(p.probability > 0)) problems.push({ key: 'pickupZero', params: { lottery: lottery.id, card: p.resourceId, resourceType: p.resourceType } })
    if (used.has(i)) return
    if (rarityOf(p.resourceType, p.resourceId) === undefined) problems.push({ key: 'pickupRarityUnknown', params: { lottery: lottery.id, card: p.resourceId, resourceType: p.resourceType } })
    else problems.push({ key: 'pickupNoSlot', params: { lottery: lottery.id, card: p.resourceId, resourceType: p.resourceType } })
  })
  return { buckets, problems }
}

// ---------------------------------------------------------------------------
// 发布前检查
// ---------------------------------------------------------------------------

export type CheckLevel = 'ok' | 'warn' | 'bad' | 'pending'

export interface Check extends Problem {
  level: CheckLevel
}

export interface CheckDeps {
  rarityOf: (resourceType: number, id: string) => number | undefined
  /** 目录核对状态；不传时跳过目录核对（列表上的「需检查」只看本地规则） */
  cardStatus?: (resourceType: number, id: string) => 'ok' | 'missing' | 'pending' | 'error'
}

/** 卡池引用的 UP 卡（展示用的 UP 与抽选表里的 UP）。 */
export function upCardsOf(cfg: GashaConfig, g: GashaDef): { resourceType: number; id: string }[] {
  const out: { resourceType: number; id: string }[] = []
  const add = (resourceType: number, id: string) => {
    if (id && !out.some((x) => x.resourceType === resourceType && x.id === id)) out.push({ resourceType, id })
  }
  for (const id of g.pickupIdolCardSkinIds || []) add(RESOURCE_IDOL, id)
  for (const id of g.pickupSupportCardIds || []) add(RESOURCE_SUPPORT, id)
  for (const lid of lotteriesOf(g)) for (const p of lotteryById(cfg, lid)?.pickups || []) add(p.resourceType, p.resourceId)
  return out
}

/** 卡池类型需要的形态字段（1 普通卡池没有）。 */
const SHAPE_OF_TYPE: Record<number, 'stepUp' | 'selectPickup' | 'continuous' | 'stepUpSelectPickup' | null> = {
  1: null,
  2: 'stepUp',
  3: 'selectPickup',
  4: 'continuous',
  5: 'stepUpSelectPickup',
}

const rewardsValid = (list: ConfigReward[] | undefined) => !!list?.length && list.every((r) => !!r && (r.resourceType || 0) > 0)

export function pointProblems(p: GashaPoint): Problem[] {
  const out: Problem[] = []
  if (!p.name) out.push({ key: 'pointName' })
  if (!(p.startTime > 0) || !(p.endTime > p.startTime)) out.push({ key: 'pointPeriod' })
  if (!p.expireReward || !((p.expireReward.resourceType || 0) > 0)) out.push({ key: 'pointExpire' })
  if (!p.exchanges?.length) out.push({ key: 'pointNoExchange' })
  ;(p.exchanges || []).forEach((x, i) => {
    if (x.number !== i + 1 || !(x.requiredQuantity > 0) || !rewardsValid(x.rewards) || (x.endTime !== 0 && x.endTime <= x.startTime)) {
      out.push({ key: 'pointExchange', params: { n: i + 1 } })
    }
  })
  return out
}

/** 一个卡池的发布前检查（本地规则 + 目录核对）。服务端校验另外调用。 */
export function gashaChecks(cfg: GashaConfig, g: GashaDef, deps: CheckDeps): Check[] {
  const out: Check[] = []
  const bad = (key: string, params?: Problem['params']) => out.push({ level: 'bad', key, params })

  if (!g.id) bad('idMissing')
  else if (cfg.gashas.filter((x) => x.id === g.id).length > 1) bad('idDuplicate')
  if (!g.name) bad('nameMissing')
  if (!g.bannerAssetId) bad('bannerMissing')

  if (!(g.startTime > 0)) bad('startMissing')
  else if (g.endTime !== 0 && g.endTime <= g.startTime) bad('periodInvalid')
  else out.push({ level: 'ok', key: 'periodOk' })

  // 形态与类型
  const shapes = [g.stepUp, g.selectPickup, g.stepUpSelectPickup, g.continuous].filter(Boolean).length
  const buttons = g.buttons || []
  const shapeKey = SHAPE_OF_TYPE[g.type]
  if (!(g.type in SHAPE_OF_TYPE)) bad('typeUnknown')
  else if (shapeKey) {
    if (shapes !== 1 || !g[shapeKey]) bad('shapeMismatch')
  } else if (shapes !== 0) bad('shapeMismatch')

  // 按钮
  const stepOnly = g.type === 2 || g.type === 5
  if (stepOnly && buttons.length) bad('buttonsStepOnly')
  if (!stepOnly && !buttons.length) bad('buttonsMissing')
  const ids = new Set<string>()
  let buttonBad = false
  for (const b of buttons) {
    if (!b.id) {
      bad('buttonIdEmpty')
      buttonBad = true
    } else if (ids.has(b.id)) {
      bad('buttonDuplicate', { button: b.id })
      buttonBad = true
    }
    ids.add(b.id)
    if ((b.limitCount ?? 0) < 0) {
      bad('buttonLimit', { button: b.id })
      buttonBad = true
    }
  }
  if (!stepOnly && buttons.length && !buttonBad) out.push({ level: 'ok', key: 'buttonsOk', params: { n: buttons.length } })

  // 抽选表
  const lids = lotteriesOf(g)
  let lotteryBad = false
  const cards = new Set<string>()
  for (const id of lids) {
    const l = lotteryById(cfg, id)
    if (!l) {
      bad('lotteryMissing', { lottery: id })
      lotteryBad = true
      continue
    }
    const total = lotteryTotal(l)
    if (total !== PROBABILITY_TOTAL) {
      bad('lotterySum', { lottery: id, total: percent(total) })
      lotteryBad = true
    }
    const prepared = prepareLottery(cfg, l, deps.rarityOf)
    for (const p of prepared.problems) {
      out.push({ level: p.key === 'pickupRarityUnknown' ? 'pending' : 'bad', key: p.key, params: p.params })
      if (p.key !== 'pickupRarityUnknown') lotteryBad = true
    }
    for (const s of l.slots || []) for (const c of slotInfo(cfg, s, 0).cardIds) cards.add(c)
  }
  if (!lids.length) {
    bad('lotteryNone')
    lotteryBad = true
  }
  if (!lotteryBad) out.push({ level: 'ok', key: 'lotteriesOk', params: { n: lids.length, cards: cards.size } })

  // UP 卡：在游戏目录里存在；展示的 UP 在抽选表里有单独概率
  const ups = upCardsOf(cfg, g)
  if (ups.length && deps.cardStatus) {
    let pending = false
    let missing = false
    for (const u of ups) {
      const st = deps.cardStatus(u.resourceType, u.id)
      if (st === 'missing') {
        bad('cardMissing', { card: u.id, resourceType: u.resourceType })
        missing = true
      } else if (st === 'pending' || st === 'error') pending = true
    }
    if (pending && !missing) out.push({ level: 'pending', key: 'cardsChecking' })
    else if (!missing) out.push({ level: 'ok', key: 'cardsOk', params: { n: ups.length } })
  }
  const rated = new Set<string>()
  for (const p of lotteryById(cfg, g.lotteryId)?.pickups || []) rated.add(`${p.resourceType}:${p.resourceId}`)
  for (const id of g.pickupIdolCardSkinIds || []) if (!rated.has(`${RESOURCE_IDOL}:${id}`)) out.push({ level: 'warn', key: 'upNoRate', params: { card: id, resourceType: RESOURCE_IDOL } })
  for (const id of g.pickupSupportCardIds || []) if (!rated.has(`${RESOURCE_SUPPORT}:${id}`)) out.push({ level: 'warn', key: 'upNoRate', params: { card: id, resourceType: RESOURCE_SUPPORT } })

  // 背景
  if ((g.movieAssetId || '').endsWith('.usm')) bad('movieExtension')
  if (!g.movieAssetId && !g.screenAssetId && !g.backgroundPickupIdolCardSkins?.length && !g.backgroundPickupSupportCards?.length) {
    bad('backgroundMissing')
  }

  // 交换点
  if (g.gashaPointId) {
    const p = cfg.points.find((x) => x.id === g.gashaPointId)
    if (!p) bad('pointMissing')
    else {
      for (const x of pointProblems(p)) bad(x.key, x.params)
      const covers = p.startTime <= g.startTime && g.endTime !== 0 && p.endTime >= g.endTime
      out.push(covers ? { level: 'ok', key: 'pointCovers' } : { level: 'warn', key: 'pointNotCover' })
    }
  }
  return out
}

// ---------------------------------------------------------------------------
// 模拟抽卡（按草稿里的抽选表在浏览器里抽，不经过服务端）
// ---------------------------------------------------------------------------

export interface SimGroup {
  resourceType: number
  rarity: number
  expected: number
  observed: number
}

export interface SimPickup {
  id: string
  resourceType: number
  expected: number
  observed: number
}

export interface SimResult {
  n: number
  groups: SimGroup[]
  pickups: SimPickup[]
  /** 两次抽到 UP 之间（含第一次）的抽数 */
  firstUp: { mean: number; median: number; hits: number } | null
  /** 理论平均：100000 / UP 概率之和 */
  expectedFirstUp: number | null
}

export function simulate(prepared: Prepared, n: number, rand: () => number = Math.random): SimResult {
  const buckets = prepared.buckets.filter((b) => b.probability > 0)
  const total = buckets.reduce((s, b) => s + b.probability, 0)
  const counts = new Array<number>(buckets.length).fill(0)
  const gaps: number[] = []
  let since = 0
  const cumulative: number[] = []
  let acc = 0
  for (const b of buckets) {
    acc += b.probability
    cumulative.push(acc)
  }
  for (let i = 0; i < n && total > 0; i++) {
    const r = rand() * total
    let lo = 0
    let hi = cumulative.length - 1
    while (lo < hi) {
      const mid = (lo + hi) >> 1
      if (r < (cumulative[mid] ?? 0)) hi = mid
      else lo = mid + 1
    }
    counts[lo] = (counts[lo] ?? 0) + 1
    since++
    if (buckets[lo]?.pickupId) {
      gaps.push(since)
      since = 0
    }
  }
  const groupMap = new Map<string, SimGroup>()
  const pickups: SimPickup[] = []
  buckets.forEach((b, i) => {
    const key = `${b.resourceType}:${b.rarity}`
    let g = groupMap.get(key)
    if (!g) {
      g = { resourceType: b.resourceType, rarity: b.rarity, expected: 0, observed: 0 }
      groupMap.set(key, g)
    }
    g.expected += b.probability / PROBABILITY_TOTAL
    g.observed += (counts[i] ?? 0) / Math.max(1, n)
    if (b.pickupId) {
      pickups.push({ id: b.pickupId, resourceType: b.resourceType, expected: b.probability / PROBABILITY_TOTAL, observed: (counts[i] ?? 0) / Math.max(1, n) })
    }
  })
  const groups = [...groupMap.values()].sort((a, b) => b.rarity - a.rarity || b.resourceType - a.resourceType)
  let firstUp: SimResult['firstUp'] = null
  if (gaps.length) {
    const sorted = [...gaps].sort((a, b) => a - b)
    const mid = sorted.length >> 1
    const median = sorted.length % 2 ? (sorted[mid] ?? 0) : ((sorted[mid - 1] ?? 0) + (sorted[mid] ?? 0)) / 2
    firstUp = { mean: gaps.reduce((s, v) => s + v, 0) / gaps.length, median, hits: gaps.length }
  }
  const upTotal = buckets.filter((b) => b.pickupId).reduce((s, b) => s + b.probability, 0)
  return { n, groups, pickups, firstUp, expectedFirstUp: upTotal > 0 ? total / upTotal : null }
}
