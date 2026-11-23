// Agent 管理各标签上的数量：七个标签页共用一份，谁改了数据谁调用 refresh()。
// 全员 Agent 页（useAgentAll）另有一份：会话、工作区、MCP、技能数的是所有人的。
import { getAgentOverview } from '@/api/agent'
import type { GetAgentOverviewResponse } from '@/types/v1/agent'
import { useAgentAll } from './useAgentAll'

interface OverviewStore {
  overview: Ref<GetAgentOverviewResponse | null>
  loading: Promise<void> | null
}

const stores: Record<'mine' | 'all', OverviewStore> = {
  mine: { overview: ref(null), loading: null },
  all: { overview: ref(null), loading: null },
}

/** 要在组件的 setup 里调用：在不在全员 Agent 页里决定用哪一份 */
export const useAgentOverview = () => {
  const all = useAgentAll()
  const store = all ? stores.all : stores.mine
  const refresh = () => {
    if (!store.loading) {
      store.loading = getAgentOverview(all)
        .then(({ data }) => {
          store.overview.value = data
        })
        .catch(() => {})
        .finally(() => {
          store.loading = null
        })
    }
    return store.loading
  }
  if (!store.overview.value) void refresh()
  return { overview: store.overview, refresh }
}
