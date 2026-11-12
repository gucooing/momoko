// GM 运营的页面上下文：当前服务器、manifest、实时 status 与目录查询。
// GM 页面（/game/gm/:serverId）提供，各标签与玩家详情用 useGmContext() 取用。
// 页面里可用的标签、指令、配置域、报表、玩法全部按服务端 manifest 判断，不按游戏写死。
import { computed, inject, provide, reactive, ref, shallowRef, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import { getGmCatalog, getGmContext, mgpQuery } from '@/api/gameGm'
import { getCurrentLocale, i18n } from '@/locales'
import { localizeEntries, lookupGameName, searchGameNames } from '@/games/names'
import { fuzzyMatch } from '@/games/search'
import type { GameServerBrief } from '@/types/v1/game_common'
import type { CatalogEntry, ListPage, Manifest, ManifestCommand, Status } from '@/games/mgp'

export interface GmCatalogQuery {
  q?: string
  characterId?: string
  rarity?: string | number
  limit?: number
  cursor?: string
}

export interface GmContext {
  serverId: Ref<string>
  server: Ref<GameServerBrief | undefined>
  manifest: Ref<Manifest | undefined>
  status: Ref<Status | undefined>
  /** status 拉取失败的原因（服务端连不上等） */
  statusError: Ref<string>
  /** 服务器时区（IANA） */
  timezone: Ref<string>
  pendingApprovals: Ref<number>
  selfServiceEnabled: Ref<number>
  linkedAccounts: Ref<number>
  loading: Ref<boolean>
  /** 加载失败（服务器不存在、没有接入 MGP 等） */
  error: Ref<string>
  /** 游戏 id（manifest.game.id，没有 manifest 时用服务器的适配器 id）；只用于判断与调接口，不显示 */
  game: ComputedRef<string>
  /** 游戏名称（显示用） */
  gameName: ComputedRef<string>
  refresh: () => Promise<void>
  command: (id: string) => ManifestCommand | undefined
  hasCommand: (id: string) => boolean
  hasConfig: (id: string) => boolean
  hasReport: (id: string) => boolean
  hasView: (id: string) => boolean
  /** 目录查询（同样的查询 60 秒内复用结果；只按需请求，不一次性拉全量） */
  catalog: (kind: string, query?: GmCatalogQuery) => Promise<ListPage<CatalogEntry>>
  /**
   * 游戏里的东西（角色、卡、道具、卡池、玩法……）的名称（响应式），从不返回 ID：
   * 前端名称表（当前语言的译名 → 游戏原文名）→ 服务端目录里的名称 → fallback。界面上只显示名称，ID 只用来调接口。
   */
  name: (kind: string, id: string | null | undefined, fallback?: string) => string
  /** 同 name，查不到名称时显示「未命名」。 */
  label: (kind: string, id: string | null | undefined) => string
  /** 指令结果里一项变化（kind + resourceId）的名称：道具等显示条目名称，货币与等级等显示对应文案。 */
  changeLabel: (kind: string, resourceId?: string) => string
}

const GM_CONTEXT: InjectionKey<GmContext> = Symbol('gm-context')

const CATALOG_TTL = 60_000
const CATALOG_CACHE_MAX = 100

/** 创建并提供 GM 上下文（GM 页面与玩家详情页调用）。 */
export function provideGmContext(serverId: Ref<string>): GmContext {
  const server = ref<GameServerBrief>()
  const manifest = shallowRef<Manifest>()
  const status = shallowRef<Status>()
  const statusError = ref('')
  const timezone = ref('')
  const pendingApprovals = ref(0)
  const selfServiceEnabled = ref(0)
  const linkedAccounts = ref(0)
  const loading = ref(false)
  const error = ref('')
  const catalogCache = new Map<string, { at: number; value: Promise<ListPage<CatalogEntry>> }>()
  // 服务端目录里的名称（前端名称表里没有的类型，如卡池、研修，才会用到；每类整表拉一次）
  const serverNames = reactive(new Map<string, Record<string, string>>())
  const serverNamesPending = new Set<string>()
  let loadedFor = ''

  const refresh = async () => {
    const id = serverId.value
    if (!id) return
    loading.value = true
    if (loadedFor !== id) {
      catalogCache.clear()
      fullCatalogs.clear()
      serverNames.clear()
      manifest.value = undefined
      status.value = undefined
    }
    try {
      const { data } = await getGmContext(id)
      if (serverId.value !== id) return
      loadedFor = id
      server.value = data.server
      manifest.value = (data.manifest as Manifest | undefined) || undefined
      status.value = (data.status as Status | undefined) || undefined
      statusError.value = data.statusError || ''
      timezone.value = data.timezone || ''
      pendingApprovals.value = data.pendingApprovals || 0
      selfServiceEnabled.value = data.selfServiceEnabled || 0
      linkedAccounts.value = data.linkedAccounts || 0
      error.value = ''
    } catch (e) {
      if (serverId.value !== id) return
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      if (serverId.value === id) loading.value = false
    }
  }

  const command = (id: string) => manifest.value?.commands?.find((c) => c.id === id)
  const has = <T extends { id: string }>(list: T[] | null | undefined, id: string) => !!list?.some((x) => x.id === id)

  const game = computed(() => manifest.value?.game?.id || server.value?.game || '')
  const gameName = computed(() => {
    const key = `game.adapters.${game.value}`
    return manifest.value?.game?.name || (game.value && i18n.global.te(key) ? i18n.global.t(key) : i18n.global.t('game.unnamed'))
  })

  // 按译名搜索时用到的完整目录（每类最多 1000 条，各类实际只有几百条；只在按译名搜索时才拉取）
  const fullCatalogs = new Map<string, Promise<CatalogEntry[]>>()
  const fullCatalog = (kind: string) => {
    const key = `${serverId.value}|${kind}`
    let p = fullCatalogs.get(key)
    if (!p) {
      p = getGmCatalog<ListPage<CatalogEntry>>(serverId.value, kind, mgpQuery({ limit: 1000 })).then((page) => page.items || [])
      p.catch(() => fullCatalogs.delete(key))
      fullCatalogs.set(key, p)
    }
    return p
  }

  // 目录条目的名称换成当前语言的译名；关键字命中译名时（服务端只认原文与 ID）在本地筛选分页。
  const fetchCatalog = async (kind: string, query: GmCatalogQuery, locale: string): Promise<ListPage<CatalogEntry>> => {
    const q = query.q?.trim() || ''
    if (q) {
      const ids = await searchGameNames(game.value, locale, kind, q)
      if (ids.size) {
        // e.id === q：按 ID 查条目（lookup）时也走这里，ID 恰好模糊命中某些名称时不能把它自己漏掉
        let list = (await fullCatalog(kind)).filter((e) => e.id === q || ids.has(e.id) || fuzzyMatch(q, [e.name]))
        if (query.characterId) list = list.filter((e) => e.characterId === query.characterId)
        if (query.rarity !== undefined && query.rarity !== '') list = list.filter((e) => String(e.rarity) === String(query.rarity))
        const offset = Number(query.cursor) || 0
        const limit = query.limit || 50
        const items = await localizeEntries(game.value, locale, kind, list.slice(offset, offset + limit))
        return { items, nextCursor: offset + limit < list.length ? String(offset + limit) : '', total: list.length }
      }
    }
    const page = await getGmCatalog<ListPage<CatalogEntry>>(serverId.value, kind, mgpQuery({ ...query }))
    return { ...page, items: await localizeEntries(game.value, locale, kind, page.items || []) }
  }

  const serverName = (kind: string, id: string): string => {
    const key = `${serverId.value}|${kind}`
    const table = serverNames.get(key)
    if (table) return table[id] || ''
    if (!serverNamesPending.has(key) && has(manifest.value?.catalogs, kind)) {
      serverNamesPending.add(key)
      fullCatalog(kind)
        .then((items) => {
          const names: Record<string, string> = {}
          for (const e of items) if (e.name) names[e.id] = e.name
          serverNames.set(key, names)
        })
        .catch(() => serverNames.set(key, {}))
        .finally(() => serverNamesPending.delete(key))
    }
    return ''
  }

  const name = (kind: string, id: string | null | undefined, fallback = ''): string => {
    if (!id || !kind) return fallback
    const local = lookupGameName(game.value, getCurrentLocale(), kind, id)
    if (local) return local
    if (local === undefined) return fallback
    return serverName(kind, id) || fallback
  }

  const label = (kind: string, id: string | null | undefined) => name(kind, id) || i18n.global.t('game.unnamed')

  const changeLabel = (kind: string, resourceId?: string) => {
    if (resourceId) return label(kind, resourceId)
    const key = `gm.change.${kind}`
    if (i18n.global.te(key)) return i18n.global.t(key)
    return name('currency', kind) || kind
  }

  const catalog = (kind: string, query: GmCatalogQuery = {}) => {
    const locale = getCurrentLocale()
    const key = `${serverId.value}|${locale}|${kind}|${mgpQuery({ ...query })}`
    const hit = catalogCache.get(key)
    if (hit && Date.now() - hit.at < CATALOG_TTL) return hit.value
    const value = fetchCatalog(kind, query, locale).catch((e) => {
      catalogCache.delete(key)
      throw e
    })
    catalogCache.set(key, { at: Date.now(), value })
    if (catalogCache.size > CATALOG_CACHE_MAX) {
      const oldest = catalogCache.keys().next().value
      if (oldest !== undefined) catalogCache.delete(oldest)
    }
    return value
  }

  const ctx: GmContext = {
    serverId,
    server,
    manifest,
    status,
    statusError,
    timezone,
    pendingApprovals,
    selfServiceEnabled,
    linkedAccounts,
    loading,
    error,
    game,
    gameName,
    refresh,
    command,
    hasCommand: (id) => !!command(id),
    hasConfig: (id) => has(manifest.value?.configs, id),
    hasReport: (id) => has(manifest.value?.reports, id),
    hasView: (id) => has(manifest.value?.views, id),
    catalog,
    name,
    label,
    changeLabel,
  }
  provide(GM_CONTEXT, ctx)
  return ctx
}

/** 取 GM 上下文（只能在 GM 页面或玩家详情页的子组件里用）。 */
export function useGmContext(): GmContext {
  const ctx = inject(GM_CONTEXT)
  if (!ctx) throw new Error('useGmContext() must be used inside the GM page')
  return ctx
}
