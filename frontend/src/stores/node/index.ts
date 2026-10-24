import { defineStore } from 'pinia'
import { getNodeContext, listNodes } from '@/api/node'
import router from '@/router'
import { NodeState, type NodeInfo } from '@/types/v1/node'
import { resolveNodeRoutePermission } from '@/config/nodePermission'
import { currentNodeId, storeCurrentNodeId } from '@/utils/nodeContext'
import { translate } from '@/locales'

// 切换节点后这些带资源 id 的页面在新节点上无意义，回到工作台。
const NODE_DETAIL_ROUTE_PREFIXES = ['/instance/console', '/instance/files', '/openssh/terminal']

export const useNodeStore = defineStore('node', () => {
  const nodes = ref<NodeInfo[]>([])
  const nodesLoading = ref(false)
  const contextInfo = ref<NodeInfo | null>(null)
  const permissions = ref<Set<string>>(new Set())
  const contextLoaded = ref(false)

  const isRemote = computed(() => !!currentNodeId.value)
  const nodeId = computed(() => currentNodeId.value)
  const isOnline = computed(() => contextInfo.value?.status?.state === NodeState.NodeState_Online)
  /** 远程节点上 API Key 所属用户 id（节点侧资源的 owner 比较用）。 */
  const remoteUserId = computed(() => contextInfo.value?.status?.remote?.userId || '')

  // 本机 / 远程的权限合并判断在 menu store 的 hasButtonPermission 中完成（避免两个 store 互相引用）。

  /** 页面在当前节点上下文中是否可访问（全局页面始终可访问）。 */
  const canAccessRoute = (path: string): boolean => {
    if (!isRemote.value) return true
    const perm = resolveNodeRoutePermission(path)
    if (perm === undefined || perm === '') return true
    if (perm === null) return false
    return permissions.value.has(perm)
  }

  const loadNodes = async () => {
    nodesLoading.value = true
    try {
      const { data } = await listNodes({})
      nodes.value = data?.infos || []
    } finally {
      nodesLoading.value = false
    }
  }

  /** 加载当前远程节点上下文（应用启动时一次）；节点已删除或无权访问时回落到主控本机。 */
  const loadContext = async () => {
    if (!isRemote.value) {
      contextLoaded.value = true
      return
    }
    try {
      const { data } = await getNodeContext({ id: currentNodeId.value })
      contextInfo.value = data?.info || null
      permissions.value = new Set(data?.permissions || [])
      if (contextInfo.value && !isOnline.value) {
        feedback.warning(
          translate('node.context.unavailable', {
            name: contextInfo.value.name,
            reason: contextInfo.value.status?.message || '',
          }),
        )
      }
    } catch {
      storeCurrentNodeId('')
      contextInfo.value = null
      permissions.value = new Set()
      feedback.warning(translate('node.context.fallbackLocal'))
    } finally {
      contextLoaded.value = true
    }
  }

  /** 切换节点：整页重载，确保在途上传、WebSocket、轮询与页面缓存全部重置。 */
  const switchNode = (id: string) => {
    if ((id || '') === currentNodeId.value) return
    storeCurrentNodeId(id)
    const current = router.currentRoute.value
    const stayOnPage =
      !NODE_DETAIL_ROUTE_PREFIXES.some((prefix) => current.path.startsWith(prefix)) &&
      Object.keys(current.params || {}).length === 0
    const target = stayOnPage ? current.path : '/dashboard/home'
    window.location.assign(router.resolve(target).href)
  }

  return {
    nodes,
    nodesLoading,
    contextInfo,
    permissions,
    contextLoaded,
    isRemote,
    nodeId,
    isOnline,
    remoteUserId,
    canAccessRoute,
    loadNodes,
    loadContext,
    switchNode,
  }
})
