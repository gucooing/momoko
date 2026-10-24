import { normalizeAuthToken, toBearerAuthHeader } from '@/utils/request'
import { applyNodeQuery } from '@/utils/nodeContext'

const AUTH_QUERY_KEYS = ['accessToken', 'token', 'authorization']

export const buildBackendWebSocketUrl = (
  wsPath: string,
  extend?: (url: URL) => void,
) => {
  const path = wsPath.trim()
  if (!path) return ''

  let url: URL
  if (/^wss?:\/\//i.test(path)) {
    url = new URL(path)
  } else if (/^https?:\/\//i.test(path)) {
    url = new URL(path)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  } else {
    const apiBaseUrl = new URL(
      import.meta.env.VITE_API_BASE_URL || window.location.origin,
      window.location.origin,
    )
    const wsOrigin = `${apiBaseUrl.protocol === 'https:' ? 'wss:' : 'ws:'}//${apiBaseUrl.host}`
    url = new URL(path.startsWith('/') ? path : `/${path}`, wsOrigin)
    // 后端返回的相对 WS 路径属于当前节点：远程节点上下文中由主控经 gRPC 隧道转发。
    applyNodeQuery(url)
  }

  const token = normalizeAuthToken(localStorage.getItem('accessToken') || '')
  const hasAuthQuery = AUTH_QUERY_KEYS.some((key) => url.searchParams.has(key))
  if (token && !hasAuthQuery) {
    url.searchParams.set('accessToken', toBearerAuthHeader(token))
  }

  extend?.(url)

  return url.toString()
}
