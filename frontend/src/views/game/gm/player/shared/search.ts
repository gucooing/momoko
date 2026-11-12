// 玩家数据页的名称搜索：关键字为空时照常分页；输入关键字后把这名玩家这一类数据全部读出来（只读一次，数据刷新后重读），
// 按名称在本地模糊筛选（games/search.ts 的 fuzzyMatch：当前语言的译名与游戏原文名都能搜，不区分全半角、大小写、平片假名）。
import { computed, ref, shallowRef, watch, type WatchSource } from 'vue'
import { fuzzyMatch } from '@/games/search'
import { getRequestErrorMessage } from '@/utils/request'

export function useNameSearch<T>(options: {
  /** 读出全部条目 */
  fetchAll: () => Promise<T[]>
  /** 一条数据可被搜索的名称（卡名、角色名、道具名……） */
  names: (item: T) => (string | undefined | null)[]
  /** 数据版本（指令执行后递增、筛选条件变化），变化时丢掉已读的全部数据 */
  version: WatchSource<unknown> | WatchSource<unknown>[]
}) {
  const keyword = ref('')
  const all = shallowRef<T[] | null>(null)
  const loading = ref(false)
  const error = ref('')
  let seq = 0

  const ensure = async () => {
    if (all.value || loading.value) return
    const id = ++seq
    loading.value = true
    error.value = ''
    try {
      const list = await options.fetchAll()
      if (id === seq) all.value = list
    } catch (e) {
      if (id === seq) error.value = getRequestErrorMessage(e)
    } finally {
      if (id === seq) loading.value = false
    }
  }

  const active = computed(() => !!keyword.value.trim())
  watch(keyword, (q) => {
    if (q.trim()) void ensure()
  })
  watch(options.version, () => {
    seq++
    all.value = null
    loading.value = false
    if (active.value) void ensure()
  })

  /** 命中的条目（没有关键字时为空数组；名称表加载完成后自动重新筛选） */
  const results = computed<T[]>(() => {
    if (!active.value || !all.value) return []
    const q = keyword.value
    return all.value.filter((item) => fuzzyMatch(q, options.names(item)))
  })

  return { keyword, active, results, loading, error }
}
