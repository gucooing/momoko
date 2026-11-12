// 按 ID 查目录条目（名称、稀有度、角色），每个 ID 只请求一次（ids 筛选，不拉全量）。
// lookup() 在渲染里调用即可：还没查到时返回 undefined 并开始查询，查到后用到它的组件自动更新。
import { reactive } from 'vue'
import type { CatalogEntry } from '@/games/mgp'
import type { GmContext } from '../../../context'

export type LookupStatus = 'ok' | 'missing' | 'pending' | 'error'

export interface CatalogLookup {
  /** 条目；undefined = 还在查，null = 目录里没有 */
  lookup: (kind: string, id: string) => CatalogEntry | null | undefined
  status: (kind: string, id: string) => LookupStatus
  /** 记下已经拿到的条目（例如选择器里点选的） */
  remember: (kind: string, entry: CatalogEntry) => void
}

export function useCatalogLookup(ctx: GmContext): CatalogLookup {
  const entries = reactive(new Map<string, CatalogEntry | null>())
  const failed = reactive(new Set<string>())
  const pending = new Set<string>()
  const keyOf = (kind: string, id: string) => `${kind}\u0000${id}`

  const request = (kind: string, id: string) => {
    const key = keyOf(kind, id)
    if (!id || pending.has(key) || failed.has(key) || entries.has(key)) return
    pending.add(key)
    ctx
      .catalog(kind, { ids: [id], limit: 1 })
      .then((page) => {
        entries.set(key, (page.items || []).find((e) => e.id === id) ?? null)
      })
      .catch(() => {
        failed.add(key)
      })
      .finally(() => pending.delete(key))
  }

  const lookup = (kind: string, id: string) => {
    const key = keyOf(kind, id)
    if (entries.has(key)) return entries.get(key)
    request(kind, id)
    return undefined
  }

  const status = (kind: string, id: string): LookupStatus => {
    const key = keyOf(kind, id)
    if (failed.has(key)) return 'error'
    const v = lookup(kind, id)
    if (v === undefined) return 'pending'
    return v ? 'ok' : 'missing'
  }

  const remember = (kind: string, entry: CatalogEntry) => {
    entries.set(keyOf(kind, entry.id), entry)
  }

  return { lookup, status, remember }
}
