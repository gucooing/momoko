// 用户通行证的接口客户端：自己的 axios 实例、自己的令牌存储键，和管理后台完全分开。
// 响应信封 { code, message, data }；失败时 message 是错误代码（PASSPORT_*），由 errorText() 本地化。
import axios, { type AxiosError, type AxiosRequestConfig, type InternalAxiosRequestConfig } from 'axios'

const ACCESS_KEY = 'passport.accessToken'
const REFRESH_KEY = 'passport.refreshToken'
const DEVICE_KEY = 'passport.deviceId'

export const tokenStore = {
  access: () => localStorage.getItem(ACCESS_KEY) || '',
  refresh: () => localStorage.getItem(REFRESH_KEY) || '',
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS_KEY, access)
    localStorage.setItem(REFRESH_KEY, refresh)
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

/** 本浏览器的设备 id（登录设备列表按它区分）。 */
export function deviceId(): string {
  let id = localStorage.getItem(DEVICE_KEY)
  if (!id) {
    id = crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`
    localStorage.setItem(DEVICE_KEY, id)
  }
  return id
}

/** 设备名称：浏览器 + 系统（只用于展示）。 */
export function deviceName(): string {
  const ua = navigator.userAgent
  const os = /Android/i.test(ua)
    ? 'Android'
    : /iPhone|iPad|iPod/i.test(ua)
      ? 'iOS'
      : /Windows/i.test(ua)
        ? 'Windows'
        : /Mac OS/i.test(ua)
          ? 'macOS'
          : /Linux/i.test(ua)
            ? 'Linux'
            : ''
  const browser = /Edg\//.test(ua)
    ? 'Edge'
    : /Firefox\//.test(ua)
      ? 'Firefox'
      : /Chrome\//.test(ua)
        ? 'Chrome'
        : /Safari\//.test(ua)
          ? 'Safari'
          : ''
  return [os, browser].filter(Boolean).join(' · ')
}

/** 接口错误：status 为 HTTP 状态，code 为错误代码或后端原文。 */
export class PassportError extends Error {
  status: number
  code: string
  constructor(status: number, code: string) {
    super(code)
    this.status = status
    this.code = code
  }
}

type Envelope<T> = { code?: number; message?: string; data?: T }

const client = axios.create({
  baseURL: import.meta.env.VITE_PASSPORT_API_BASE_URL || '/api/v1',
  timeout: 60_000,
  // 列表参数重复给出（ids=a&ids=b）
  paramsSerializer: { indexes: null },
})

client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = tokenStore.access()
  if (token && !config.headers.Authorization) config.headers.Authorization = `Bearer ${token}`
  return config
})

let refreshing: Promise<boolean> | null = null
let onExpired: (() => void) | null = null

/** 登录失效（刷新也失败）时的回调：通常跳到登录页。 */
export function setExpiredHandler(fn: () => void) {
  onExpired = fn
}

async function refreshTokens(): Promise<boolean> {
  const rt = tokenStore.refresh()
  if (!rt) return false
  try {
    const res = await axios.post<Envelope<{ accessToken: string; refreshToken: string }>>(
      `${client.defaults.baseURL}/auth/refresh`,
      { refreshToken: rt },
    )
    const data = res.data?.data
    if (!data?.accessToken) return false
    tokenStore.set(data.accessToken, data.refreshToken)
    return true
  } catch {
    return false
  }
}

const NO_RETRY = ['/auth/login', '/auth/refresh', '/auth/register', '/auth/login/mfa']

async function call<T>(config: AxiosRequestConfig, retried = false): Promise<T> {
  try {
    const res = await client.request<Envelope<T>>(config)
    return res.data?.data as T
  } catch (e) {
    const err = e as AxiosError<Envelope<unknown>>
    const status = err.response?.status ?? 0
    const message = err.response?.data?.message || (status === 0 ? 'NETWORK_ERROR' : String(status))
    if (status === 401 && !retried && !NO_RETRY.some((p) => config.url?.startsWith(p))) {
      refreshing ??= refreshTokens().finally(() => {
        refreshing = null
      })
      if (await refreshing) {
        const headers = { ...(config.headers || {}) } as Record<string, string>
        delete headers.Authorization
        return call<T>({ ...config, headers }, true)
      }
      tokenStore.clear()
      onExpired?.()
    }
    throw new PassportError(status, message)
  }
}

export const http = {
  get: <T>(url: string, params?: Record<string, unknown>) => call<T>({ method: 'GET', url, params }),
  post: <T>(url: string, data?: unknown) => call<T>({ method: 'POST', url, data: data ?? {} }),
  put: <T>(url: string, data?: unknown) => call<T>({ method: 'PUT', url, data: data ?? {} }),
  del: <T>(url: string) => call<T>({ method: 'DELETE', url }),
}
