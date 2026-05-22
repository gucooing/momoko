const HTTP_URL_PATTERN = /^(https?:)?\/\//i
const NON_HTTP_SCHEME_PATTERN = /^(data:|blob:)/i
const FRONTEND_ASSET_PATH_PATTERN = /^\/(?:src|assets)\//i

const getFallbackOrigin = () => {
  if (typeof window !== 'undefined' && window.location.origin) {
    return window.location.origin
  }

  return 'http://localhost'
}

const getResolvedResourceOrigin = () => {
  const currentOrigin = getFallbackOrigin()
  const configuredBaseUrl = String(import.meta.env.VITE_API_BASE_URL || '').trim()

  if (!configuredBaseUrl) {
    return currentOrigin
  }

  try {
    const resolvedApiUrl = new URL(configuredBaseUrl, currentOrigin)
    return resolvedApiUrl.origin === currentOrigin ? currentOrigin : resolvedApiUrl.origin
  } catch {
    return currentOrigin
  }
}

const normalizeResourcePath = (value: string) => {
  return value.startsWith('/') ? value : `/${value}`
}

export const resolveStaticResourceUrl = (value?: string | null) => {
  const trimmedValue = String(value || '').trim()
  if (!trimmedValue) return ''

  if (
    HTTP_URL_PATTERN.test(trimmedValue) ||
    NON_HTTP_SCHEME_PATTERN.test(trimmedValue) ||
    FRONTEND_ASSET_PATH_PATTERN.test(trimmedValue)
  ) {
    return trimmedValue
  }

  try {
    return new URL(normalizeResourcePath(trimmedValue), getResolvedResourceOrigin()).toString()
  } catch {
    return trimmedValue
  }
}

export const resolveAvatarUrl = (avatar?: string | null) => {
  return resolveStaticResourceUrl(avatar)
}
