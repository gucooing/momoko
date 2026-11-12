// 游戏空间的共享上下文：服务器、绑定、玩法、我的玩家摘要，以及按需、带缓存的目录查询。
// 游戏数据都只针对自己绑定的游戏账号（后端按绑定决定 pid）；学マス 的接口见 passport/src/api/gakumas.ts。
import { inject, provide, reactive, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import * as api from '@passport/api/gakumas'
import { gakumas, gameCatalogImage, localizeEntries, lookupGameName } from '@/games'
import { queryCatalog, resourceRef as gakumasResourceRef, type GakumasCatalogApi } from '@/games/gakumas/catalog'
import { gakumasModes, type GakumasMode } from '@/games/gakumas/modes'
import { i18n } from '@passport/locales'
import type { CatalogEntry, CatalogQuery, ListPage } from '@/games/mgp'
import type { Modes, Overview, PlayerDetail } from '@/games/gakumas/types'
import type { PassportGameResponse } from '@/types/v1/passport'

export type GameTab = 'home' | 'cards' | 'gifts' | 'records' | 'self'

export interface GameContext {
  serverId: ComputedRef<string>
  game: ComputedRef<string>
  data: Ref<PassportGameResponse | null>
  /** 这台服务器配置了的玩法 */
  modes: ComputedRef<Modes | undefined>
  player: ComputedRef<PlayerDetail | null>
  overview: ComputedRef<Overview | null>
  timezone: ComputedRef<string>
  hasMode: (id: GakumasMode) => boolean
  modeTitle: (id: GakumasMode) => string
  /** 目录条目（响应式：没查过时返回 undefined 并开始查询；查不到为 null）。 */
  entry: (kind: string, id: string) => CatalogEntry | null | undefined
  /** 名称（从不返回 ID）：前端名称表（当前语言的译名 → 游戏原文名）→ 服务端目录的名称；都没有时为空串。 */
  nameOf: (kind: string, id: string) => string
  /** 一类目录的前 50 条（角色等小目录用）。 */
  list: (kind: string) => CatalogEntry[] | undefined
  /** 目录查询（关键字已换成名称表命中的 ID 时由服务端按 ID 筛选）。 */
  catalog: (kind: string, query: CatalogQuery) => Promise<ListPage<CatalogEntry>>
  /** 资源（奖励、礼物、抽到的卡）的目录位置：currency / item / idolCardSkin …。 */
  resourceRef: (resourceType: number, resourceId: string) => { kind: string; id: string }
  resourceImage: (resourceType: number, resourceId: string) => string
  resourceName: (resourceType: number, resourceId: string) => string
  setTab: (tab: GameTab, extra?: Record<string, string>) => void
  reload: () => Promise<void>
}

const KEY: InjectionKey<GameContext> = Symbol('passport-game')

export function provideGameContext(ctx: GameContext) {
  provide(KEY, ctx)
}

export function useGameContext(): GameContext {
  const ctx = inject(KEY)
  if (!ctx) throw new Error('game context missing')
  return ctx
}

/** 学マス 玩法的标题。 */
export const modeTitleOf = (id: GakumasMode) => gakumasModes.find((m) => m.id === id)?.title || id

// 玩家端可查的目录（通行证上没有卡池、塔、培育组目录）
const catalogApi: GakumasCatalogApi = {
  currencies: api.listCurrencies,
  items: api.listItemCatalog,
  idolCardSkins: api.listIdolCardSkins,
  supportCards: api.listSupportCardCatalog,
  characters: api.listCharacters,
  costumes: api.listCostumes,
  costumeHeads: api.listCostumeHeads,
  music: api.listMusic,
  produces: api.listProduceCatalog,
  seminars: api.listSeminars,
  produceCards: api.listProduceCards,
  produceItems: api.listProduceItems,
  memoryAbilities: api.listMemoryAbilities,
}

const BATCH = 100

/** 创建目录缓存与资源换算（GameSpace 调用一次）。 */
export function createCatalogs(serverId: ComputedRef<string>, game: ComputedRef<string>) {
  const entries = reactive(new Map<string, CatalogEntry | null>())
  const lists = reactive(new Map<string, CatalogEntry[]>())
  const pending = new Set<string>()
  const wanted = new Map<string, Set<string>>()
  let scheduled = false

  // 名称换成当前语言的译名（src/games/names.ts；没有译名的保留服务端原文）
  const locale = () => String(i18n.global.locale.value)

  const fetch = async (kind: string, query: CatalogQuery, lc: string) => {
    const page = await queryCatalog(catalogApi, serverId.value, kind, query)
    return { ...page, items: await localizeEntries(game.value, lc, kind, page.items) }
  }

  // 同一轮里要查的 ID 按目录种类攒成批，每批一次 ids 查询
  const flush = () => {
    scheduled = false
    const lc = locale()
    const sid = serverId.value
    for (const [kind, set] of wanted) {
      wanted.delete(kind)
      const all = [...set]
      for (let i = 0; i < all.length; i += BATCH) {
        const ids = all.slice(i, i + BATCH)
        fetch(kind, { ids, limit: ids.length }, lc)
          .then((page) => {
            const byId = new Map(page.items.map((e) => [e.id, e]))
            for (const id of ids) entries.set(`${sid}|${lc}|${kind}|${id}`, byId.get(id) ?? null)
          })
          .catch(() => {
            for (const id of ids) entries.set(`${sid}|${lc}|${kind}|${id}`, null)
          })
      }
    }
  }

  const entry = (kind: string, id: string): CatalogEntry | null | undefined => {
    if (!id || !kind) return null
    const lc = locale()
    const key = `${serverId.value}|${lc}|${kind}|${id}`
    if (entries.has(key)) return entries.get(key)
    if (!pending.has(key)) {
      pending.add(key)
      let set = wanted.get(kind)
      if (!set) {
        set = new Set()
        wanted.set(kind, set)
      }
      set.add(id)
      if (!scheduled) {
        scheduled = true
        queueMicrotask(flush)
      }
    }
    return undefined
  }

  const list = (kind: string): CatalogEntry[] | undefined => {
    const lc = locale()
    const key = `${serverId.value}|${lc}|${kind}`
    if (lists.has(key)) return lists.get(key)
    if (!pending.has(key)) {
      pending.add(key)
      fetch(kind, { limit: 50 }, lc)
        .then((page) => {
          lists.set(key, page.items)
          for (const it of page.items) entries.set(`${serverId.value}|${lc}|${kind}|${it.id}`, it)
        })
        .catch(() => lists.set(key, []))
    }
    return undefined
  }

  const catalog = (kind: string, query: CatalogQuery) => fetch(kind, query, locale())

  const nameOf = (kind: string, id: string): string => {
    if (!id || !kind) return ''
    const local = lookupGameName(game.value, locale(), kind, id)
    if (local) return local
    if (local === undefined) return ''
    return entry(kind, id)?.name || ''
  }

  // 货币（钻石、AP、经验）作为奖励时 resourceId 为空串，对应 currency 目录里的同名条目
  const resourceRef = (resourceType: number, resourceId: string) => gakumasResourceRef(resourceType, resourceId) || { kind: '', id: '' }

  const resourceImage = (resourceType: number, resourceId: string) => {
    const ref = resourceRef(resourceType, resourceId)
    if (!ref.id || !ref.kind) return ''
    const e = ref.kind === 'item' ? entry('item', ref.id) : undefined
    return gameCatalogImage(game.value, ref.kind, { id: ref.id, assetId: e?.assetId })
  }

  const resourceName = (resourceType: number, resourceId: string) => {
    const ref = resourceRef(resourceType, resourceId)
    return ref.kind && ref.id ? nameOf(ref.kind, ref.id) : ''
  }

  return { entry, list, catalog, nameOf, resourceRef, resourceImage, resourceName }
}

// ---------- 协议里固定的数值标签 ----------

// ResultGrade：1 F … 8 A、10 S、12 SS、14 SSS、16 SSSS、18 SSSSS，奇数为对应的 +
const GRADES = ['', 'F', 'E', 'D', 'C', 'C+', 'B', 'B+', 'A', 'A+', 'S', 'S+', 'SS', 'SS+', 'SSS', 'SSS+', 'SSSS', 'SSSS+', 'SSSSS', 'SSSSS+']

export const gradeLabel = (grade: number | null | undefined) => (grade ? GRADES[grade] || String(grade) : '—')

// Rarity：1 R、2 SR、3 SSR
const RARITIES = ['', 'R', 'SR', 'SSR']

export const rarityLabel = (rarity: number | null | undefined) => (rarity ? RARITIES[rarity] || String(rarity) : '')

/** 游戏内显示的 ID：每 3 位一组。 */
export const spacedId = (id: string) => (id || '').replace(/(\d{3})(?=\d)/g, '$1 ')

/** 偶像卡大图 / 方图、支援卡、角色头像与半身像：只有学マス 有素材，其他游戏返回空串。 */
export const idolCardArt = (game: string, id: string, variant: 'sq' | 'land' | 'full' = 'land') =>
  game === 'gakumas' && id ? gakumas.idolCardImage(id, variant) : ''
export const supportCardArt = (game: string, id: string, variant: 'sq' | 'land' | 'full' = 'sq') =>
  game === 'gakumas' && id ? gakumas.supportCardImage(id, variant) : ''
export const charaArt = (game: string, id: string, variant: 'circle' | 'bust' = 'circle') =>
  game === 'gakumas' && id ? gakumas.charaImage(id, variant) : ''
export const currencyArt = (game: string, kind: 'jewelFree' | 'jewelPaid' | 'actionPoint' | 'money') =>
  game === 'gakumas' ? gakumas.currencyImage(kind) : ''
export const gashaArt = (game: string, idOrAsset: string) => (game === 'gakumas' && idOrAsset ? gakumas.gashaBanner(idOrAsset) : '')

/** 原样 JSON 值转短文本（变化前后）。 */
export const valueText = (v: unknown): string => {
  if (v === null || v === undefined) return '—'
  if (typeof v === 'number') return new Intl.NumberFormat().format(v)
  if (typeof v === 'string' || typeof v === 'boolean') return String(v)
  return JSON.stringify(v)
}

/** 自助操作记录的结果（JSON 文本）里改变的值；没有改变的值时为空。 */
export const recordChanges = (data: string): { before: unknown; after: unknown }[] => {
  if (!data) return []
  try {
    const v = JSON.parse(data) as { changes?: { before: unknown; after: unknown }[] }
    return Array.isArray(v?.changes) ? v.changes : []
  } catch {
    return []
  }
}

/** 毫秒时间戳 → Date（0 为空）。 */
export const ms = (v: number | null | undefined) => (v ? new Date(v) : null)
