// 按 ID 查名称与素材 ID（道具、卡池、角色、培育……）：名称先查前端名称表（当前语言的译名 → 游戏原文名），
// 没有时再查目录接口；界面上只显示名称，从不显示 ID。
// 只查页面上真正显示出来的条目：同一类目录的 ID 攒成一批（ids 筛选，并发最多 4 个），结果在页面实例里缓存；
// 不一次性拉整个目录。偶像卡皮肤可以按角色分组查（一个角色一次，见 group）。
import { reactive } from 'vue'
import type { CatalogEntry } from '@/games/mgp'
import { resourceRef, type GakumasCatalogKind } from '@/games/gakumas/catalog'
import { gameNameVariants, lookupGameName } from '@/games/names'
import { getCurrentLocale, i18n } from '@/locales'
import { useGmContext, type GmContext } from '../../context'
import { catalogImage } from './game'

export interface RewardRef {
  /** 目录种类（货币统一在 currency 目录） */
  kind: GakumasCatalogKind
  /** 目录条目 id（货币为 jewelFree 等） */
  id: string
}

export interface CatalogLookup {
  /** 目录条目：查询中为 undefined（并开始查询），查不到为 null */
  entry: (kind: string, id: string) => CatalogEntry | null | undefined
  /** 名称（从不返回 ID）：查询中为空串，查不到时为「未命名」 */
  name: (kind: string, id: string) => string
  /** 用于按名称搜索的全部名称：当前语言的译名、游戏原文名与目录里的名称 */
  names: (kind: string, id: string) => string[]
  /** 图片（目录没有图片时为空串）；道具等需要目录里的 assetId 才对得上 */
  image: (kind: string, id: string) => string
  /** 按角色分组的条目（偶像卡皮肤等）：查询中为 undefined，查不到为 null */
  group: (kind: string, characterId: string) => CatalogEntry[] | null | undefined
  /** 奖励资源（resourceType + resourceId）对应的目录条目 */
  rewardRef: (resourceType: number, resourceId: string) => RewardRef | undefined
}

const PARALLEL = 4
const BATCH = 100
const lookups = new WeakMap<GmContext, CatalogLookup>()

function createLookup(ctx: GmContext): CatalogLookup {
  const found = reactive(new Map<string, CatalogEntry | null>())
  const groups = reactive(new Map<string, CatalogEntry[] | null>())
  const queued = new Set<string>()
  const pending = new Map<string, Set<string>>()
  const waiting: Array<() => Promise<void>> = []
  let running = 0
  let scheduled = false

  const pump = () => {
    while (running < PARALLEL && waiting.length) {
      const job = waiting.shift()
      if (!job) break
      running++
      void job().finally(() => {
        running--
        pump()
      })
    }
  }

  // 同一轮里要查的 ID 按目录种类攒成批，每批一次 ids 查询
  const flush = () => {
    scheduled = false
    const serverId = ctx.serverId.value
    for (const [kind, set] of pending) {
      pending.delete(kind)
      const all = [...set]
      for (let i = 0; i < all.length; i += BATCH) {
        const ids = all.slice(i, i + BATCH)
        waiting.push(async () => {
          try {
            const page = await ctx.catalog(kind, { ids, limit: ids.length })
            const byId = new Map(page.items.map((e) => [e.id, e]))
            for (const id of ids) found.set(`${serverId}|${kind}|${id}`, byId.get(id) || null)
          } catch {
            for (const id of ids) found.set(`${serverId}|${kind}|${id}`, null)
          }
        })
      }
    }
    pump()
  }

  const entry = (kind: string, id: string): CatalogEntry | null | undefined => {
    if (!id) return null
    const key = `${ctx.serverId.value}|${kind}|${id}`
    if (found.has(key)) return found.get(key)
    if (!queued.has(key)) {
      queued.add(key)
      let set = pending.get(kind)
      if (!set) {
        set = new Set()
        pending.set(kind, set)
      }
      set.add(id)
      if (!scheduled) {
        scheduled = true
        queueMicrotask(flush)
      }
    }
    return undefined
  }

  const group = (kind: string, characterId: string): CatalogEntry[] | null | undefined => {
    if (!characterId) return null
    const key = `${ctx.serverId.value}|${kind}|@${characterId}`
    if (groups.has(key)) return groups.get(key)
    if (!queued.has(key)) {
      queued.add(key)
      waiting.push(async () => {
        try {
          const page = await ctx.catalog(kind, { characterId, limit: 500 })
          groups.set(key, page.items)
        } catch {
          groups.set(key, null)
        }
      })
      pump()
    }
    return undefined
  }

  const name = (kind: string, id: string) => {
    if (!id) return ''
    const local = lookupGameName(ctx.game.value, getCurrentLocale(), kind, id)
    if (local) return local
    if (local === undefined) return ''
    const e = entry(kind, id)
    if (e === undefined) return ''
    return e?.name || i18n.global.t('game.unnamed')
  }

  const names = (kind: string, id: string) => {
    if (!id) return []
    const out = gameNameVariants(ctx.game.value, getCurrentLocale(), kind, id)
    const e = out.length ? undefined : entry(kind, id)
    if (e?.name && !out.includes(e.name)) out.push(e.name)
    return out
  }

  const image = (kind: string, id: string) => {
    if (!id) return ''
    const e = entry(kind, id)
    return catalogImage(ctx.game.value, kind, e || { id })
  }

  return { entry, name, names, image, group, rewardRef: resourceRef }
}

/** 某个 GM 上下文的目录查询（提供上下文的页面自己用这个：同一组件里 inject 读不到自己 provide 的值）。 */
export function catalogLookupFor(ctx: GmContext): CatalogLookup {
  let l = lookups.get(ctx)
  if (!l) {
    l = createLookup(ctx)
    lookups.set(ctx, l)
  }
  return l
}

/** 当前 GM 页面（或玩家详情页）的目录查询（在页面的子组件里用）。 */
export function useCatalogLookup(): CatalogLookup {
  return catalogLookupFor(useGmContext())
}
