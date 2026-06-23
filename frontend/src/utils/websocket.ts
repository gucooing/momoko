import { normalizeAuthToken, toBearerAuthHeader } from '@/utils/request'

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
  }

  const token = normalizeAuthToken(localStorage.getItem('accessToken') || '')
  const hasAuthQuery = AUTH_QUERY_KEYS.some((key) => url.searchParams.has(key))
  if (token && !hasAuthQuery) {
    url.searchParams.set('accessToken', toBearerAuthHeader(token))
  }

  extend?.(url)

  return url.toString()
}
