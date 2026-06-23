import axios, { type AxiosInstance, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import { imagineContext } from '@/api/imagine-context'
import { translate as t } from '@/locales'
import type {
  ListImageGenApiKeysResponse,
  ListImageGenModelsRequest,
  ListImageGenModelsResponse,
  ListImageGenGenerationsRequest,
  ListImageGenGenerationsResponse,
  CreateImageGenGenerationRequest,
  ImageGenGenerationResponse,
} from '@/types/v1/sub2api_imagine'

const baseURL = import.meta.env.VITE_API_BASE_URL as string

// 独立 axios 实例：不携带 momoko 鉴权头、不做 401 刷新；改用 X-Sub2API-Token 鉴权。
const instance: AxiosInstance = axios.create({
  baseURL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json;charset=UTF-8' },
})

instance.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (imagineContext.token) config.headers.set('X-Sub2API-Token', imagineContext.token)
  if (imagineContext.srcHost) config.headers.set('X-Sub2API-Host', imagineContext.srcHost)
  return config
})

// 解包 {code,message,data} 信封；code !== 200 转为拒绝并带 message。
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

// 公开图片 URL（同源，供 <img> 与 sub2api 回拉使用）。
export const imageGenImageUrl = (id: string) => `${baseURL}/public/sub2api/imagine/images/${id}`

// apikey / models
export const listImageGenApiKeys = () =>
  unwrap<ListImageGenApiKeysResponse>(instance.get('/public/sub2api/imagine/apikeys'))

export const listImageGenModels = (params: ListImageGenModelsRequest) =>
  unwrap<ListImageGenModelsResponse>(instance.get('/public/sub2api/imagine/models', { params }))

// generations
export const listImageGenGenerations = (params: ListImageGenGenerationsRequest = { limit: 0 }) =>
  unwrap<ListImageGenGenerationsResponse>(instance.get('/public/sub2api/imagine/generations', { params }))

export const createImageGenGeneration = (data: CreateImageGenGenerationRequest) =>
  unwrap<ImageGenGenerationResponse>(instance.post('/public/sub2api/imagine/generations', data))

export const getImageGenGeneration = (id: string) =>
  unwrap<ImageGenGenerationResponse>(instance.get(`/public/sub2api/imagine/generations/${id}`))

export const deleteImageGenGeneration = (id: string) =>
  unwrap(instance.delete(`/public/sub2api/imagine/generations/${id}`))
