// Docker 引擎状态（容器/镜像/网络/配置四页 + 子导航共用）：同一时刻只发一次请求，短时间内复用结果。
// 引擎未启用/未连接时，列表页据此展示空状态而不去请求列表接口（避免一进页面就弹错误 toast）。
import { getDockerStatus } from '@/api/docker'
import type { DockerStatusResponse } from '@/types/v1/docker'

const status = ref<DockerStatusResponse | null>(null)
const loading = ref(false)
const loaded = ref(false)
let inflight: Promise<DockerStatusResponse | null> | null = null
let fetchedAt = 0
let fetchedNode = ''
const STALE_MS = 8000

export function useDockerStatus() {
  const nodeStore = useNodeStore()
  const nodeKey = () => String(nodeStore.nodeId || '')

  const refresh = async (force = false): Promise<DockerStatusResponse | null> => {
    const node = nodeKey()
    if (!force && loaded.value && node === fetchedNode && Date.now() - fetchedAt < STALE_MS) return status.value
    if (inflight) return inflight
    loading.value = true
    inflight = (async () => {
      try {
        const { data } = await getDockerStatus()
        status.value = data || null
      } catch {
        status.value = null
      } finally {
        loaded.value = true
        fetchedAt = Date.now()
        fetchedNode = node
        loading.value = false
        inflight = null
      }
      return status.value
    })()
    return inflight
  }

  /** 引擎可用：已启用且已连接 */
  const ready = computed(() => !!status.value?.connected)
  /** 状态：connected / disconnected（已启用但连不上）/ disabled */
  const state = computed<'connected' | 'disconnected' | 'disabled' | 'unknown'>(() => {
    if (!loaded.value) return 'unknown'
    if (status.value?.connected) return 'connected'
    if (status.value?.enabled) return 'disconnected'
    return 'disabled'
  })

  return { status, loading, loaded, ready, state, refresh }
}
