import { ref } from 'vue'

// 与后端 internal/server/node_proxy.go 保持一致。
export const LOCAL_NODE_ID = 'local'
export const NODE_HEADER = 'X-Momoko-Node'
export const NODE_QUERY_KEY = 'momokoNode'

const STORAGE_KEY = 'momoko-node'

const readStoredNodeId = () => {
  try {
    return sessionStorage.getItem(STORAGE_KEY) || ''
  } catch {
    return ''
  }
}

/**
 * 当前浏览器标签页所操作的节点 id（空 = 主控本机）。按标签页隔离（sessionStorage），刷新后保留。
 * 切换节点会整页重载（见 stores/node 的 switchNode），因此页面生命周期内该值保持不变，
 * 上传、WebSocket、轮询等在途请求不会跨节点串用。
 */
export const currentNodeId = ref<string>(readStoredNodeId())

export const isRemoteNode = () => !!currentNodeId.value

export const storeCurrentNodeId = (id: string) => {
  const next = id && id !== LOCAL_NODE_ID ? id : ''
  currentNodeId.value = next
  try {
    if (next) sessionStorage.setItem(STORAGE_KEY, next)
    else sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    // 隐私模式等场景无法持久化，仅保留内存值
  }
}

const isRelativeResource = (raw: string) =>
  !/^[a-z][a-z0-9+.-]*:/i.test(raw) && !raw.startsWith('//')

/**
 * 浏览器直连的后端地址（WebSocket、预签名下载/上传）无法携带请求头，改用查询参数标识目标节点。
 * raw 为后端返回的原始地址：只有相对路径（momoko 自身接口）才附加；OSS 等第三方直链原样返回。
 */
export const withNodeQuery = (resolved: string, raw: string = resolved): string => {
  if (!currentNodeId.value || !resolved || !isRelativeResource(raw.trim())) return resolved
  try {
    const url = new URL(resolved, window.location.origin)
    if (!url.searchParams.has(NODE_QUERY_KEY)) {
      url.searchParams.set(NODE_QUERY_KEY, currentNodeId.value)
    }
    return url.toString()
  } catch {
    return resolved
  }
}

/** 为已构造好的 URL 对象（WebSocket）附加节点标识。 */
export const applyNodeQuery = (url: URL) => {
  if (currentNodeId.value && !url.searchParams.has(NODE_QUERY_KEY)) {
    url.searchParams.set(NODE_QUERY_KEY, currentNodeId.value)
  }
}
