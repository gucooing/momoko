// MGP 列表的分页读取：cursor 为偏移量（mgp.ts ListPage），total 可靠时用页码分页，否则「加载更多」追加。
import { ref, shallowRef } from 'vue'
import { getRequestErrorMessage } from '@/utils/request'
import type { ListPage } from '@/games/mgp'

export interface PageQuery {
  cursor: string
  limit: number
}

export function usePagedList<T, S = unknown>(
  fetcher: (q: PageQuery) => Promise<ListPage<T, S> | null | undefined>,
  options: { pageSize: number; append?: boolean },
) {
  const items = shallowRef<T[]>([])
  const total = ref(0)
  const summary = shallowRef<S>()
  const nextCursor = ref('')
  const page = ref(1)
  const pageSize = ref(options.pageSize)
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref('')
  let seq = 0

  const run = async (cursor: string, appending: boolean) => {
    const id = ++seq
    loading.value = true
    error.value = ''
    try {
      const data = await fetcher({ cursor, limit: pageSize.value })
      if (id !== seq) return
      const list = data?.items || []
      items.value = appending ? [...items.value, ...list] : list
      total.value = data?.total || 0
      summary.value = data?.summary
      nextCursor.value = data?.nextCursor || ''
      loaded.value = true
    } catch (e) {
      if (id !== seq) return
      error.value = getRequestErrorMessage(e)
      if (!appending) items.value = []
    } finally {
      if (id === seq) loading.value = false
    }
  }

  /** 页码模式：读当前页；追加模式：从头读第一页。 */
  const load = () => {
    if (options.append) return run('', false)
    return run(page.value > 1 ? String((page.value - 1) * pageSize.value) : '', false)
  }

  /** 回到第一页重新读（筛选变化时用）。 */
  const reload = () => {
    page.value = 1
    return load()
  }

  /** 追加下一页。 */
  const more = () => (nextCursor.value && !loading.value ? run(nextCursor.value, true) : Promise.resolve())

  return { items, total, summary, nextCursor, page, pageSize, loading, loaded, error, load, reload, more }
}

/** 读出全部条目（按名称搜索时用）：每页 200 条依次读，最多 max 条。 */
export async function fetchAllPages<T>(
  fetcher: (q: PageQuery) => Promise<ListPage<T, unknown> | null | undefined>,
  max = 5000,
): Promise<T[]> {
  const out: T[] = []
  const seen = new Set<string>()
  let cursor = ''
  do {
    seen.add(cursor)
    const data = await fetcher({ cursor, limit: 200 })
    out.push(...(data?.items || []))
    cursor = data?.nextCursor || ''
  } while (cursor && !seen.has(cursor) && out.length < max)
  return out
}
