import request from '@/utils/request'
import { isRemoteNode } from '@/utils/nodeContext'
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
  CreateShareSessionResponse,
  GetShareQRCodeResponse,
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

// 分享链接二维码：后端按对外地址（没设置时按本机当前站点）拼出公开链接并编码为 PNG（JSON 里 bytes 为 base64），
// 返回可直接用作 <img src> 的 data URL。远程节点不带当前站点：主控的地址不是节点的分享地址。
export const getShareQRCodeRequest = async (id: string): Promise<{ src: string; link: string }> => {
  const origin = typeof window !== 'undefined' && !isRemoteNode() ? window.location.origin : ''
  const { data } = await request.get<GetShareQRCodeResponse>('/file/share/qrcode', { params: { id, origin } })
  const png = data.png as unknown as string | Uint8Array
  const b64 = typeof png === 'string' ? png : btoa(String.fromCharCode(...png))
  return { src: `data:image/png;base64,${b64}`, link: data.link }
}

// 公开：获取分享元信息（引导用，不含目录/文件数据）
export const getShareMetaRequest = (token: string) =>
  request.get<GetShareMetaResponse>('/public/share/meta', { params: { token } })

// 公开：用 token(+提取码) 换取一次性会话签名；后续请求只带签名。
export const createShareSessionRequest = (token: string, code = '') =>
  request.post<CreateShareSessionResponse>('/public/share/session', { token, code })

// 公开：浏览分享目录（需会话签名）
export const listShareDirRequest = (data: ListShareDirRequest) =>
  request.post<ListShareDirResponse>('/public/share/list', data)

// 分享的公开访问链接：后端按「对外地址」拼好的 link 优先；本机没设置时按当前站点拼。
// 远程节点没设置对外地址时返回空（主控的地址不是节点的分享地址）。
export const shareLinkOf = (share: { token: string; link?: string }): string => {
  if (share.link) return share.link
  if (typeof window === 'undefined' || isRemoteNode()) return ''
  return `${window.location.origin}/public/share/${share.token}`
}

// 公开下载/预览链接：分享 token + 会话签名 + 相对子路径。
// inline=true 走预览（不计下载次数）；否则附件下载。后端路由：/api/v1/public/share/download
// 所有访问都强制携带会话签名（避免无签名直链导致的越权）。
export const buildShareDownloadUrl = (
  token: string,
  sign: string,
  options: { path?: string; paths?: string[]; inline?: boolean } = {},
): string => {
  const base = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
  const params = new URLSearchParams({ token, sign })
  if (options.path) params.set('path', options.path)
  // 多选下载：path 重复出现，后端把这些条目一起打包为 zip。
  for (const p of options.paths || []) if (p) params.append('path', p)
  if (options.inline) params.set('inline', '1')
  return `${base}/public/share/download?${params.toString()}`
}
