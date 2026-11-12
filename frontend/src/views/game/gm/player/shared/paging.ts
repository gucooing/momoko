// MGP 列表的页码分页读取：cursor 为偏移量（mgp.ts ListPage），每次只读当前页。
import { ref, shallowRef } from 'vue'
import { getRequestErrorMessage } from '@/utils/request'
import type { ListPage } from '@/games/mgp'

export interface PageQuery {
  cursor: string
  limit: number
}

export function usePagedList<T, S = unknown>(
  fetcher: (q: PageQuery) => Promise<ListPage<T, S> | null | undefined>,
  options: { pageSize: number },
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

  const run = async (cursor: string) => {
    const id = ++seq
    loading.value = true
    error.value = ''
    try {
      const data = await fetcher({ cursor, limit: pageSize.value })
      if (id !== seq) return
      const list = data?.items || []
      items.value = list
      total.value = data?.total || 0
      summary.value = data?.summary
      nextCursor.value = data?.nextCursor || ''
      loaded.value = true
    } catch (e) {
      if (id !== seq) return
      error.value = getRequestErrorMessage(e)
      items.value = []
    } finally {
      if (id === seq) loading.value = false
    }
  }

  /** 读当前页。 */
  const load = () => run(page.value > 1 ? String((page.value - 1) * pageSize.value) : '')

  /** 回到第一页重新读（筛选变化时用）。 */
  const reload = () => {
    page.value = 1
    return load()
  }

  return { items, total, summary, nextCursor, page, pageSize, loading, loaded, error, load, reload }
}
