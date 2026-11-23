// Agent 设置各标签上的数量：各标签页共用一份，谁改了数据谁调用 refresh()。
// Agent 管理页（useAgentAll）另有一份：会话、工作区、MCP、技能数的是所有人的；
// 从远程节点过去时那边只有运行时和官方账户、也不给数所有人的，就数自己的（运行时的数量是共用的，一样）。
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

/** 要在组件的 setup 里调用：在不在 Agent 管理页里决定用哪一份 */
export const useAgentOverview = () => {
  const all = useAgentAll()
  const nodeStore = useNodeStore()
  const store = all ? stores.all : stores.mine
  const refresh = () => {
    if (!store.loading) {
      store.loading = getAgentOverview(all && !nodeStore.isRemote)
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
