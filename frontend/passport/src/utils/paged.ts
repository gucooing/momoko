// 按页读取的列表（页码分页，配 PPager）：每次只读当前页，换页、筛选变化时重新读；
// 先后发出的请求只采用最后一次的结果。
import { reactive } from 'vue'
import { errorText } from './format'

export interface PagedResult<T, S> {
  items: T[]
  total: number
  summary?: S
}

export interface PagedList<T, S = undefined> {
  items: T[]
  total: number
  /** 当前页（从 1 开始） */
  page: number
  pageSize: number
  summary: S | undefined
  loading: boolean
  /** 读过至少一次 */
  loaded: boolean
  error: string
}

/** MGP 列表的游标是偏移量：第 page 页的游标。 */
export const offsetCursor = (page: number, pageSize: number) => (page > 1 ? String((page - 1) * pageSize) : '')

export function usePagedList<T, S = undefined>(fetch: (page: number, pageSize: number) => Promise<PagedResult<T, S>>, pageSize: number) {
  const list = reactive({
    items: [],
    total: 0,
    page: 1,
    pageSize,
    summary: undefined,
    loading: false,
    loaded: false,
    error: '',
  }) as PagedList<T, S>
  let seq = 0

  /** 读第 page 页（默认当前页）。 */
  async function load(page = list.page) {
    const id = ++seq
    list.page = Math.max(1, page)
    list.loading = true
    list.error = ''
    try {
      const res = await fetch(list.page, pageSize)
      if (id !== seq) return
      list.items = res.items ?? []
      list.total = res.total ?? 0
      list.summary = res.summary
      list.loaded = true
    } catch (e) {
      if (id !== seq) return
      list.error = errorText(e)
      list.items = []
      list.loaded = true
    } finally {
      if (id === seq) list.loading = false
    }
  }

  /** 回到第一页重新读（筛选变化时）。 */
  const reload = () => load(1)

  return { list, load, reload }
}
