// 游戏空间的共享上下文：服务器、绑定、manifest、玩家摘要，以及按需、带缓存的 MGP 目录查询。
// 游戏数据都只针对自己绑定的游戏账号（后端强制）。
import { computed, inject, provide, reactive, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import { getGameCatalog } from '@passport/api/passport'
import { gakumas, gameCatalogImage, localizeEntries, lookupGameName } from '@/games'
import { i18n } from '@passport/locales'
import type { CatalogEntry, ListPage, ManifestEntry, ManifestView } from '@/games/mgp'
import type { Overview, PlayerDetail } from '@/games/gakumas/types'
import type { PassportGameResponse } from '@/types/v1/passport'

/** 玩家端拿到的 manifest 子集（不含指令）。 */
export interface PlayerManifest {
  protocol: number
  game: { id: string; name: string; theme: string; icon: string }
  capabilities: string[] | null
  views: ManifestView[] | null
  modes: ManifestEntry[] | null
  reports: ManifestEntry[] | null
  catalogs: ManifestEntry[] | null
  resourceTypes: Record<string, number> | null
}

export type GameTab = 'home' | 'cards' | 'gifts' | 'records' | 'self'

export interface GameContext {
  serverId: ComputedRef<string>
  game: ComputedRef<string>
  data: Ref<PassportGameResponse | null>
  manifest: ComputedRef<PlayerManifest | null>
  player: ComputedRef<PlayerDetail | null>
  overview: ComputedRef<Overview | null>
  timezone: ComputedRef<string>
  hasView: (id: string) => boolean
  hasMode: (id: string) => boolean
  viewTitle: (id: string) => string
  modeTitle: (id: string) => string
  /** 目录条目（响应式：没查过时返回 undefined 并开始查询；查不到为 null）。 */
  entry: (kind: string, id: string) => CatalogEntry | null | undefined
  /** 名称（从不返回 ID）：前端名称表（当前语言的译名 → 游戏原文名）→ 服务端目录的名称；都没有时为空串。 */
  nameOf: (kind: string, id: string) => string
  /** 一类目录的前 50 条（角色等小目录用）。 */
  list: (kind: string) => CatalogEntry[] | undefined
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

/** 创建目录缓存与资源换算（GameSpace 调用一次）。 */
export function createCatalogs(serverId: ComputedRef<string>, game: ComputedRef<string>, manifest: ComputedRef<PlayerManifest | null>) {
  const entries = reactive(new Map<string, CatalogEntry | null>())
  const lists = reactive(new Map<string, CatalogEntry[]>())
  const pending = new Set<string>()

  const hasCatalog = (kind: string) => !!manifest.value?.catalogs?.some((c) => c.id === kind)

  // 名称换成当前语言的译名（src/games/names.ts；没有译名的保留服务端原文）
  const locale = () => String(i18n.global.locale.value)

  const entry = (kind: string, id: string): CatalogEntry | null | undefined => {
    if (!id || !kind) return null
    const lc = locale()
    const key = `${serverId.value}|${lc}|${kind}|${id}`
    if (entries.has(key)) return entries.get(key)
    if (!pending.has(key) && hasCatalog(kind)) {
      pending.add(key)
      getGameCatalog<ListPage<CatalogEntry>>(serverId.value, kind, { q: id, limit: 10 })
        .then((page) => localizeEntries(game.value, lc, kind, page?.items ?? []))
        .then((items) => {
          entries.set(key, items.find((x) => x.id === id) ?? null)
        })
        .catch(() => {
          entries.set(key, null)
        })
        .finally(() => pending.delete(key))
    }
    return undefined
  }

  const list = (kind: string): CatalogEntry[] | undefined => {
    const lc = locale()
    const key = `${serverId.value}|${lc}|${kind}`
    if (lists.has(key)) return lists.get(key)
    if (!pending.has(key) && hasCatalog(kind)) {
      pending.add(key)
      getGameCatalog<ListPage<CatalogEntry>>(serverId.value, kind, { limit: 50 })
        .then((page) => localizeEntries(game.value, lc, kind, page?.items ?? []))
        .then((items) => {
          lists.set(key, items)
          for (const it of items) entries.set(`${serverId.value}|${lc}|${kind}|${it.id}`, it)
        })
        .catch(() => lists.set(key, []))
        .finally(() => pending.delete(key))
    }
    return undefined
  }

  const nameOf = (kind: string, id: string): string => {
    if (!id || !kind) return ''
    const local = lookupGameName(game.value, locale(), kind, id)
    if (local) return local
    if (local === undefined) return ''
    return entry(kind, id)?.name || ''
  }

  /** resourceTypes 的反查：协议数值 → 资源类型名（jewelFree、item、idolCardSkin …）。 */
  const typeNames = computed(() => {
    const out = new Map<number, string>()
    for (const [name, value] of Object.entries(manifest.value?.resourceTypes ?? {})) out.set(value, name)
    return out
  })

  const resourceRef = (resourceType: number, resourceId: string) => {
    const name = typeNames.value.get(resourceType) || ''
    // 货币（钻石、AP、经验）作为奖励时 resourceId 为空串，对应 currency 目录里的同名条目。
    if (!resourceId) return { kind: 'currency', id: name }
    return { kind: name, id: resourceId }
  }

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

  return { entry, list, nameOf, resourceRef, resourceImage, resourceName }
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
export const idolCardArt = (game: string, id: string, variant: 'sq' | 'land' = 'land') =>
  game === 'gakumas' && id ? gakumas.idolCardImage(id, variant) : ''
export const supportCardArt = (game: string, id: string, variant: 'sq' | 'land' = 'sq') =>
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

/** 毫秒时间戳 → Date（0 为空）。 */
export const ms = (v: number | null | undefined) => (v ? new Date(v) : null)
