import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { useUserStore } from '@/stores/user'
import type { RefreshResponse } from '@/types/v1/auth'

interface IAuthAxiosRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
  skipAuthRefresh?: boolean
}

type WrappedResponse = { code?: number; message?: string; data?: unknown }

interface AppRequestError extends Error {
  isRequestError: true
  handled: boolean
  code?: number
  status?: number
}

const ACCESS_TOKEN_KEY = 'accessToken'
const REFRESH_TOKEN_KEY = 'refreshToken'
const AUTH_LOGIN_PATH = '/auth/login'
const AUTH_REFRESH_PATH = '/auth/refresh'

let refreshTokenPromise: Promise<string> | null = null
let isHandlingAuthExpired = false

const BEARER_PREFIX_REGEXP = /^Bearer\s+/i

export const normalizeAuthToken = (token: string) => {
  return token.replace(BEARER_PREFIX_REGEXP, '').trim()
}

const getAccessToken = () => {
  return normalizeAuthToken(localStorage.getItem(ACCESS_TOKEN_KEY) || '')
}

const getRefreshToken = () => {
  return normalizeAuthToken(localStorage.getItem(REFRESH_TOKEN_KEY) || '')
}

export const toBearerAuthHeader = (token: string) => `Bearer ${normalizeAuthToken(token)}`

const createRequestError = (
  message: string,
  options: Partial<Pick<AppRequestError, 'handled' | 'code' | 'status'>> = {},
): AppRequestError =>
  Object.assign(new Error(message), {
    name: 'AppRequestError',
    isRequestError: true as const,
    handled: false,
    ...options,
  })

const isRequestError = (error: unknown): error is AppRequestError => {
  return !!error && typeof error === 'object' && 'isRequestError' in error
}

export const isRequestErrorHandled = (error: unknown) => {
  return isRequestError(error) && error.handled
}

export const getRequestErrorMessage = (error: unknown, fallback = '请求失败') => {
  if (isRequestError(error) && error.message) {
    return error.message
  }

  return fallback
}

export const showRequestError = (error: unknown, fallback = '请求失败') => {
  if (isRequestErrorHandled(error)) return
  ElMessage.error(getRequestErrorMessage(error, fallback))
}

const handleAuthExpired = (message = '登录状态已过期，请重新登录') => {
  if (isHandlingAuthExpired) return

  isHandlingAuthExpired = true
  ElMessage.error(message)
  useUserStore().logoutLocal({
    forceReload: true,
    redirectPath: router.currentRoute.value.fullPath,
  })
}

const isAuthPath = (url?: string) => {
  if (!url) return false
  return url.includes(AUTH_LOGIN_PATH) || url.includes(AUTH_REFRESH_PATH)
}

const isWrappedResponse = (data: unknown): data is WrappedResponse => {
  return !!data && typeof data === 'object' && 'code' in data
}

// 创建 axios 实例
const service: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 30000,
  paramsSerializer: {
    indexes: null,
  },
  headers: {
    'Content-Type': 'application/json;charset=UTF-8',
  },
})

const refreshAccessToken = async () => {
  const refreshToken = getRefreshToken()
  if (!refreshToken) {
    throw new Error('refresh token is empty')
  }

  const { data } = await service.post<RefreshResponse>(AUTH_REFRESH_PATH, { refreshToken }, {
    skipAuthRefresh: true,
  } as AxiosRequestConfig)

  const refreshRes = data

  if (!refreshRes?.accessToken) {
    throw new Error('refresh access token failed')
  }

  localStorage.setItem(ACCESS_TOKEN_KEY, normalizeAuthToken(refreshRes.accessToken))

  if (refreshRes.refreshToken) {
    localStorage.setItem(REFRESH_TOKEN_KEY, normalizeAuthToken(refreshRes.refreshToken))
  } else {
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  }

  return normalizeAuthToken(refreshRes.accessToken)
}

const tryRefreshAndRetry = async (requestConfig?: IAuthAxiosRequestConfig) => {
  if (!requestConfig || requestConfig._retry || requestConfig.skipAuthRefresh) {
    return null
  }

  if (isAuthPath(requestConfig.url) || !getRefreshToken()) {
    return null
  }

  requestConfig._retry = true

  if (!refreshTokenPromise) {
    refreshTokenPromise = refreshAccessToken().finally(() => {
      refreshTokenPromise = null
    })
  }

  const nextAccessToken = await refreshTokenPromise

  requestConfig.headers = requestConfig.headers || {}
  requestConfig.headers.Authorization = toBearerAuthHeader(nextAccessToken)

  return service(requestConfig)
}

// 请求拦截器
service.interceptors.request.use(
  (config) => {
    const token = getAccessToken()
    if (token && !isAuthPath(config.url)) {
      config.headers.Authorization = toBearerAuthHeader(token)
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  },
)

// 响应拦截器
service.interceptors.response.use(
  async (response: AxiosResponse) => {
    const raw = response.data

    if (!isWrappedResponse(raw)) {
      return response
    }

    const { code, message } = raw

    if (code === 500) {
      const errorMessage = message || '服务器内部错误'
      ElMessage.error(errorMessage)
      return Promise.reject(createRequestError(errorMessage, { handled: true, code }))
    }

    if (code === 401) {
      const originalRequest = response.config as IAuthAxiosRequestConfig

      if (originalRequest.skipAuthRefresh) {
        return Promise.reject(createRequestError(message || '未授权', { code }))
      }

      if (isAuthPath(originalRequest.url)) {
        const errorMessage = message || '未授权'
        ElMessage.error(errorMessage)
        return Promise.reject(createRequestError(errorMessage, { handled: true, code }))
      }

      try {
        const retryResponse = await tryRefreshAndRetry(originalRequest)
        if (retryResponse) return retryResponse
      } catch {
        const errorMessage = message || '登录状态已过期，请重新登录'
        handleAuthExpired(errorMessage)
        return Promise.reject(createRequestError(errorMessage, { handled: true, code }))
      }

      const errorMessage = message || '登录状态已过期，请重新登录'
      handleAuthExpired(errorMessage)
      return Promise.reject(createRequestError(errorMessage, { handled: true, code }))
    }

    if (code !== undefined && code !== 200) {
      const errorMessage = message || '请求失败'
      ElMessage.error(errorMessage)
      return Promise.reject(createRequestError(errorMessage, { handled: true, code }))
    }

    response.data = raw.data
    return response
  },

  async (error: AxiosError) => {
    let errorMessage = '请求失败'

    if (error.response) {
      const responseData = error.response.data as { message?: string } | undefined
      switch (error.response.status) {
        case 401: {
          const originalRequest = error.config as IAuthAxiosRequestConfig | undefined

          if (originalRequest?.skipAuthRefresh) {
            return Promise.reject(
              createRequestError(responseData?.message || '未授权', { status: 401 }),
            )
          }

          if (!isAuthPath(originalRequest?.url)) {
            try {
              const retryResponse = await tryRefreshAndRetry(originalRequest)
              if (retryResponse) return retryResponse
            } catch {
              // 刷新失败，走统一过期处理
            }

            handleAuthExpired('未授权，请重新登录')
            return Promise.reject(
              createRequestError('未授权，请重新登录', { handled: true, status: 401 }),
            )
          }

          errorMessage = '未授权，请重新登录'
          break
        }
        case 403:
          errorMessage = '拒绝访问'
          break
        case 404:
          errorMessage = '请求地址不存在'
          break
        case 500:
          errorMessage = responseData?.message || '服务器内部错误'
          break
        default:
          errorMessage = '服务器内部错误'
      }
    } else if (error.request) {
      errorMessage = '网络连接失败，请检查网络'
    } else {
      errorMessage = error.message || '请求失败'
    }

    ElMessage.error(errorMessage)
    return Promise.reject(
      createRequestError(errorMessage, {
        handled: true,
        status: error.response?.status,
      }),
    )
  },
)

// 请求方法对象
const request = {
  /**
   * GET 请求
   * @param url 请求地址
   * @param config 请求配置（可选）
   * @returns Promise<AxiosResponse>
   */
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<AxiosResponse<T>> {
    return service.get<T>(url, config)
  },

  /**
   * POST 请求
   * @param url 请求地址
   * @param data 请求体数据（可选）
   * @param config 请求配置（可选）
   * @returns Promise<AxiosResponse>
   */
  post<T = unknown>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig,
  ): Promise<AxiosResponse<T>> {
    return service.post<T>(url, data, config)
  },

  /**
   * PUT 请求
   * @param url 请求地址
   * @param data 请求体数据（可选）
   * @param config 请求配置（可选）
   * @returns Promise<AxiosResponse>
   */
  put<T = unknown>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig,
  ): Promise<AxiosResponse<T>> {
    return service.put<T>(url, data, config)
  },

  /**
   * DELETE 请求
   * @param url 请求地址
   * @param config 请求配置（可选）
   * @returns Promise<AxiosResponse>
   */
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<AxiosResponse<T>> {
    return service.delete<T>(url, config)
  },
}

export default request
export { service }
