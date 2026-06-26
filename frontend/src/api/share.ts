import request from '@/utils/request'
import type {
  CreateShareRequest,
  CreateShareResponse,
  DeleteShareRequest,
  DeleteShareResponse,
  GetShareMetaResponse,
  ListShareDirRequest,
  ListShareDirResponse,
  ListSharesRequest,
  ListSharesResponse,
  UpdateShareRequest,
  UpdateShareResponse,
} from '@/types/v1/file'

// —— 管理（需登录 + file:share 权限）——
export const createShareRequest = (data: CreateShareRequest) =>
  request.post<CreateShareResponse>('/file/share/create', data)

export const listSharesRequest = (params: ListSharesRequest) =>
  request.get<ListSharesResponse>('/file/share/list', { params })

export const updateShareRequest = (data: UpdateShareRequest) =>
  request.post<UpdateShareResponse>('/file/share/update', data)

export const deleteShareRequest = (data: DeleteShareRequest) =>
  request.post<DeleteShareResponse>('/file/share/delete', data)

// —— 公开（免登录）——
export const getShareMetaRequest = (token: string) =>
  request.get<GetShareMetaResponse>('/public/share/meta', { params: { token } })

export const listShareDirRequest = (data: ListShareDirRequest) =>
  request.post<ListShareDirResponse>('/public/share/list', data)

// 分享落地页链接（前端公开路由，与 /public/* 约定一致）
export const buildShareLink = (token: string) => `${window.location.origin}/public/share/${token}`

// 公开下载/预览直链（浏览器直接访问，免登录；提取码/子路径按需带上）
// inline=true 用于预览（内联返回，且不计入下载次数）。
export const buildShareDownloadUrl = (token: string, code = '', relPath = '', inline = false) => {
  const base = import.meta.env.VITE_API_BASE_URL || '/api/v1'
  const params = new URLSearchParams({ token })
  if (code) params.set('code', code)
  if (relPath) params.set('path', relPath)
  if (inline) params.set('inline', '1')
  return `${base}/public/share/download?${params.toString()}`
}
