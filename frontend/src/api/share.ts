import request from '@/utils/request'
import type {
  CreateShareRequest,
  CreateShareResponse,
  ListSharesRequest,
  ListSharesResponse,
  UpdateShareRequest,
  UpdateShareResponse,
  DeleteShareResponse,
  GetShareMetaResponse,
  ListShareDirRequest,
  ListShareDirResponse,
} from '@/types/v1/file'

// 文件分享 HTTP 封装：创建者管理（需登录） + 公开访问（按 token/提取码）。

// 创建分享
export const createShareRequest = (data: CreateShareRequest) =>
  request.post<CreateShareResponse>('/file/share/create', data)

// 分享列表
export const listSharesRequest = (params: ListSharesRequest) =>
  request.get<ListSharesResponse>('/file/share/list', { params })

// 更新分享（全量覆盖可编辑字段）
export const updateShareRequest = (data: UpdateShareRequest) =>
  request.post<UpdateShareResponse>('/file/share/update', data)

// 删除分享
export const deleteShareRequest = (id: string) =>
  request.post<DeleteShareResponse>('/file/share/delete', { id })

// 公开：获取分享元信息（无需提取码）
export const getShareMetaRequest = (token: string) =>
  request.get<GetShareMetaResponse>('/public/share/meta', { params: { token } })

// 公开：浏览分享目录（按需提取码）
export const listShareDirRequest = (data: ListShareDirRequest) =>
  request.post<ListShareDirResponse>('/public/share/list', data)

// 由 token 生成公开访问链接。
export const buildShareLink = (token: string): string => {
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  return `${origin}/public/share/${token}`
}

// 公开下载/预览链接：分享 token + 相对子路径（用于公开页下载/预览按钮）。
// inline=true 走预览（不计下载次数）；否则附件下载。后端路由：/api/v1/public/share/download
export const buildShareDownloadUrl = (
  token: string,
  options: { path?: string; code?: string; inline?: boolean } = {},
): string => {
  const base = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
  const params = new URLSearchParams({ token })
  if (options.path) params.set('path', options.path)
  if (options.code) params.set('code', options.code)
  if (options.inline) params.set('inline', '1')
  return `${base}/public/share/download?${params.toString()}`
}
