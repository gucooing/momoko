import axios, { type AxiosInstance, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import { lotteryContext } from '@/api/lottery-context'
import { translate as t } from '@/locales'
import type {
  GetLotteryStatusResponse,
  RegisterLotteryResponse,
  ListLotteryHistoryPublicRequest,
  ListLotteryHistoryPublicResponse,
} from '@/types/v1/sub2api'

const baseURL = import.meta.env.VITE_API_BASE_URL as string

// 独立 axios：不带 momoko JWT；用 X-Sub2API-Token 鉴权（与生图同款）。
const instance: AxiosInstance = axios.create({
  baseURL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json;charset=UTF-8' },
})

instance.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (lotteryContext.token) config.headers.set('X-Sub2API-Token', lotteryContext.token)
  if (lotteryContext.srcHost) config.headers.set('X-Sub2API-Host', lotteryContext.srcHost)
  return config
})

instance.interceptors.response.use((response: AxiosResponse) => {
  const raw = response.data
  if (raw && typeof raw === 'object' && 'code' in raw) {
    const { code, message, data } = raw
    if (code !== 200) {
      return Promise.reject(new Error(message || t('sub2api.common.failedWithCode', { code })))
    }
    response.data = data
  }
  return response
})

const unwrap = <T>(p: Promise<AxiosResponse<T>>): Promise<T> => p.then((r) => r.data)

export const getLotteryStatus = () =>
  unwrap<GetLotteryStatusResponse>(instance.get('/public/sub2api/lottery/status'))

export const registerLottery = () =>
  unwrap<RegisterLotteryResponse>(instance.post('/public/sub2api/lottery/register', {}))

export const listLotteryHistoryPublic = (params: ListLotteryHistoryPublicRequest) =>
  unwrap<ListLotteryHistoryPublicResponse>(instance.get('/public/sub2api/lottery/history', { params }))
