// GM 运营的页面上下文：当前服务器、学マス manifest 与实时 status、操作定义与目录查询。
// GM 页面（/game/gm/:serverId）提供，各标签与玩家详情用 useGmContext() 取用。
import { computed, inject, provide, reactive, ref, shallowRef, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import { getGmContext } from '@/api/gameGm'
import * as gakumasApi from '@/api/gakumas'
import { getCurrentLocale, i18n } from '@/locales'
import { localizeEntries, lookupGameName, searchGameNames } from '@/games/names'
import { queryCatalog, type GakumasCatalogApi } from '@/games/gakumas/catalog'
import { gakumasOperation, gakumasOperations } from './operations'
import type { GameServerBrief } from '@/types/v1/game_common'
import type { Manifest, Modes, Status } from '@/types/v1/gakumas/types'
import type { CatalogEntry, CatalogQuery, ListPage, OperationDef } from '@/games/mgp'

export type GmCatalogQuery = CatalogQuery

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
  /** 游戏 id；只用于判断与调接口，不显示 */
  game: ComputedRef<string>
  /** 游戏名称（显示用） */
  gameName: ComputedRef<string>
  /** 已配置的玩法 */
  modes: ComputedRef<Modes | undefined>
  refresh: () => Promise<void>
  /** 操作定义（菜单、表单与接口） */
  operations: OperationDef[]
  operation: (id: string) => OperationDef | undefined
  /** 目录查询（同样的查询 60 秒内复用结果；只按需请求） */
  catalog: (kind: string, query?: GmCatalogQuery) => Promise<ListPage<CatalogEntry>>
  /**
   * 游戏里的东西（角色、卡、道具、卡池、玩法……）的名称（响应式），从不返回 ID：
   * 前端名称表（当前语言的译名 → 游戏原文名）→ 服务端目录里的名称 → fallback。
   */
  name: (kind: string, id: string | null | undefined, fallback?: string) => string
  /** 同 name，查不到名称时显示「未命名」。 */
  label: (kind: string, id: string | null | undefined) => string
  /** 操作结果里一项变化（kind + resourceId）的名称：道具等显示条目名称，货币与等级等显示对应文案。 */
  changeLabel: (kind: string, resourceId?: string) => string
}

const GM_CONTEXT: InjectionKey<GmContext> = Symbol('gm-context')

const CATALOG_TTL = 60_000
const CATALOG_CACHE_MAX = 100

const catalogApi: GakumasCatalogApi = {
  currencies: gakumasApi.listCurrencies,
  items: gakumasApi.listItemCatalog,
  idolCardSkins: gakumasApi.listIdolCardSkins,
  supportCards: gakumasApi.listSupportCardCatalog,
  characters: gakumasApi.listCharacters,
  costumes: gakumasApi.listCostumes,
  costumeHeads: gakumasApi.listCostumeHeads,
  music: gakumasApi.listMusic,
  gashas: gakumasApi.listGashas,
  produceGroups: gakumasApi.listProduceGroups,
  produces: gakumasApi.listProduceCatalog,
  seminars: gakumasApi.listSeminars,
  towers: gakumasApi.listTowers,
  produceCards: gakumasApi.listProduceCards,
  produceItems: gakumasApi.listProduceItems,
  memoryAbilities: gakumasApi.listMemoryAbilities,
}

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
  // 服务端目录里的名称（前端名称表里没有的种类，如卡池、研修，才会用到）：按 ID 成批查询
  const serverNames = reactive(new Map<string, string>())
  const wanted = new Map<string, Set<string>>()
  let flushing = false
  let loadedFor = ''

  const refresh = async () => {
    const id = serverId.value
    if (!id) return
    loading.value = true
    if (loadedFor !== id) {
      catalogCache.clear()
      serverNames.clear()
      manifest.value = undefined
      status.value = undefined
    }
    try {
      const { data } = await getGmContext(id)
      if (serverId.value !== id) return
      loadedFor = id
      server.value = data.server
      timezone.value = data.timezone || ''
      pendingApprovals.value = data.pendingApprovals || 0
      selfServiceEnabled.value = data.selfServiceEnabled || 0
      linkedAccounts.value = data.linkedAccounts || 0
      error.value = ''
      const [m, st] = await Promise.allSettled([gakumasApi.getGakumasManifest(id), gakumasApi.getGakumasStatus(id)])
      if (serverId.value !== id) return
      manifest.value = m.status === 'fulfilled' ? m.value : undefined
      status.value = st.status === 'fulfilled' ? st.value : undefined
      statusError.value = st.status === 'rejected' ? (st.reason instanceof Error ? st.reason.message : String(st.reason)) : ''
    } catch (e) {
      if (serverId.value !== id) return
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      if (serverId.value === id) loading.value = false
    }
  }

  const game = computed(() => manifest.value?.game?.id || server.value?.game || '')
  const gameName = computed(() => {
    const key = `game.adapters.${game.value}`
    return manifest.value?.game?.name || (game.value && i18n.global.te(key) ? i18n.global.t(key) : i18n.global.t('game.unnamed'))
  })

  // 关键字命中当前语言的译名时，把命中的 ID 交给服务端筛选（服务端只认原文与 ID）。
  const fetchCatalog = async (kind: string, query: GmCatalogQuery, locale: string): Promise<ListPage<CatalogEntry>> => {
    const q = query.q?.trim() || ''
    let req = { ...query }
    if (q) {
      const ids = await searchGameNames(game.value, locale, kind, q)
      if (ids.size) req = { ...query, q: '', ids: [...new Set([...ids, ...(query.ids || [])])] }
    }
    const page = await queryCatalog(catalogApi, serverId.value, kind, req)
    return { ...page, items: await localizeEntries(game.value, locale, kind, page.items) }
  }

  const flush = () => {
    flushing = false
    for (const [kind, ids] of wanted) {
      wanted.delete(kind)
      const list = [...ids]
      queryCatalog(catalogApi, serverId.value, kind, { ids: list, limit: list.length })
        .then((page) => {
          for (const id of list) serverNames.set(`${serverId.value}|${kind}|${id}`, '')
          for (const e of page.items) serverNames.set(`${serverId.value}|${kind}|${e.id}`, e.name)
        })
        .catch(() => {
          for (const id of list) serverNames.set(`${serverId.value}|${kind}|${id}`, '')
        })
    }
  }

  const serverName = (kind: string, id: string): string => {
    const key = `${serverId.value}|${kind}|${id}`
    if (serverNames.has(key)) return serverNames.get(key) || ''
    let ids = wanted.get(kind)
    if (!ids) {
      ids = new Set()
      wanted.set(kind, ids)
    }
    ids.add(id)
    if (!flushing) {
      flushing = true
      queueMicrotask(flush)
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
    const key = `${serverId.value}|${locale}|${kind}|${JSON.stringify(query)}`
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
    modes: computed(() => manifest.value?.modes),
    refresh,
    operations: gakumasOperations,
    operation: gakumasOperation,
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
