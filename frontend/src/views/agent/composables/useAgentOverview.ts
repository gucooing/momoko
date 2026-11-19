// Agent 管理各标签上的数量：七个标签页共用一份，谁改了数据谁调用 refresh()。
import { getAgentOverview } from '@/api/agent'
import type { GetAgentOverviewResponse } from '@/types/v1/agent'

const overview = ref<GetAgentOverviewResponse | null>(null)
let loading: Promise<void> | null = null

const refresh = () => {
  if (!loading) {
    loading = getAgentOverview()
      .then(({ data }) => {
        overview.value = data
      })
      .catch(() => {})
      .finally(() => {
        loading = null
      })
  }
  return loading
}

export const useAgentOverview = () => {
  if (!overview.value) void refresh()
  return { overview, refresh }
}
