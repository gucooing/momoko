import { getSystemOverview } from '@/api/system'

// 外壳里展示「本机 · 主机名」：整页只请求一次系统概览（切换节点会整页重载，自然刷新）。
const hostname = ref('')
let requested = false

/** 当前操作对象的主机名：本机取系统概览，远程节点取节点自报信息（离线时用最近一次在线信息）。 */
export const useCurrentHostname = () => {
  const nodeStore = useNodeStore()

  if (!requested && !nodeStore.isRemote) {
    requested = true
    getSystemOverview()
      .then(({ data }) => {
        hostname.value = data?.version?.hostname || ''
      })
      .catch(() => {
        requested = false
      })
  }

  return computed(() => {
    if (!nodeStore.isRemote) return hostname.value
    const st = nodeStore.contextInfo?.status
    return st?.remote?.hostname || st?.lastRemote?.hostname || ''
  })
}
