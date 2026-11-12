import axios, { type AxiosRequestConfig, type AxiosResponseTransformer } from 'axios'
import request from '@/utils/request'
import { i18n, translate } from '@/locales'
import type { SearchGameServersRequest, SearchGameServersResponse } from '@/types/v1/game_common'
import type {
  AdminBindPassportLinkRequest,
  CreatePassportInviteRequest,
  GetPassportSettingsResponse,
  GetPassportStatsResponse,
  GetPassportUserResponse,
  ListPassportInvitesResponse,
  ListPassportOidcClientsResponse,
  ListPassportUsersRequest,
  ListPassportUsersResponse,
  PassportInvite,
  PassportOidcClient,
  PassportSettings,
  SavePassportOidcClientRequest,
  SearchPassportGamePlayersResponse,
} from '@/types/v1/passport_admin'

// 用户通行证后台（主 server，权限 game:passport）。通行证账号独立于 Momoko 用户。
//
// 通行证接口的错误 message 是错误代码（PASSPORT_*），这里在响应进入全局拦截器之前换成当前语言的文案，
// 全局拦截器照常弹出提示；4xx 也按统一信封处理，不再被替换成笼统的「服务器错误」。

const ERROR_CODE = /^PASSPORT_[A-Z_]+$/

/** 把通行证错误代码换成当前语言的文案（没有对应文案时原样返回）。 */
export const passportAdminErrorText = (message: string) => {
  if (!ERROR_CODE.test(message)) return message
  const key = `passportAdmin.errors.${message}`
  return i18n.global.te(key) ? translate(key) : message
}

const localizeError: AxiosResponseTransformer = (data: unknown) => {
  if (data && typeof data === 'object' && 'message' in data) {
    const body = data as { code?: number; message?: unknown }
    if (body.code !== 200 && typeof body.message === 'string') {
      body.message = passportAdminErrorText(body.message)
    }
  }
  return data
}

const defaults = axios.defaults.transformResponse
const transforms: AxiosResponseTransformer[] = [
  ...(Array.isArray(defaults) ? defaults : defaults ? [defaults] : []),
  localizeError,
]

const cfg = (extra?: AxiosRequestConfig): AxiosRequestConfig => ({
  ...extra,
  // 4xx 也交给统一信封处理（code / message），由上面的转换先本地化错误代码。
  validateStatus: (status) => status < 500 || status === 502,
  transformResponse: transforms,
})

const seg = (value: string) => encodeURIComponent(value)
const base = '/game/passport'

export const getPassportStats = () => request.get<GetPassportStatsResponse>(`${base}/stats`, cfg())

export const listPassportUsers = (params: Partial<ListPassportUsersRequest>) =>
  request.get<ListPassportUsersResponse>(`${base}/users`, cfg({ params }))

export const getPassportUser = (id: string) =>
  request.get<GetPassportUserResponse>(`${base}/users/${seg(id)}`, cfg())

export const setPassportUserStatus = (id: string, disabled: boolean) =>
  request.post<GetPassportUserResponse>(`${base}/users/${seg(id)}/status`, { disabled }, cfg())

export const adminUnbindPassportLink = (userId: string, linkId: string) =>
  request.post<GetPassportUserResponse>(`${base}/users/${seg(userId)}/links/${seg(linkId)}/unbind`, {}, cfg())

export const adminBindPassportLink = (req: AdminBindPassportLinkRequest) =>
  request.post<GetPassportUserResponse>(
    `${base}/users/${seg(req.userId)}/links`,
    { serverId: req.serverId, pid: req.pid },
    cfg(),
  )

export const searchPassportServers = (params: Partial<SearchGameServersRequest>) =>
  request.get<SearchGameServersResponse>(`${base}/servers`, cfg({ params }))

export const searchPassportGamePlayers = (serverId: string, q: string) =>
  request.get<SearchPassportGamePlayersResponse>(`${base}/servers/${seg(serverId)}/players`, cfg({ params: { q } }))

export const listPassportInvites = (page: number, pageSize: number) =>
  request.get<ListPassportInvitesResponse>(`${base}/invites`, cfg({ params: { page, pageSize } }))

export const createPassportInvite = (req: CreatePassportInviteRequest) =>
  request.post<PassportInvite>(`${base}/invites`, req, cfg())

export const revokePassportInvite = (id: string) =>
  request.post<PassportInvite>(`${base}/invites/${seg(id)}/revoke`, {}, cfg())

export const getPassportSettings = () => request.get<GetPassportSettingsResponse>(`${base}/settings`, cfg())

export const updatePassportSettings = (settings: PassportSettings) =>
  request.put<GetPassportSettingsResponse>(`${base}/settings`, settings, cfg())

export const listPassportOidcClients = () =>
  request.get<ListPassportOidcClientsResponse>(`${base}/oidc-clients`, cfg())

export const createPassportOidcClient = (req: SavePassportOidcClientRequest) =>
  request.post<PassportOidcClient>(`${base}/oidc-clients`, req, cfg())

export const updatePassportOidcClient = (req: SavePassportOidcClientRequest) =>
  request.put<PassportOidcClient>(`${base}/oidc-clients/${seg(req.id)}`, req, cfg())

export const deletePassportOidcClient = (id: string) => request.delete(`${base}/oidc-clients/${seg(id)}`, cfg())

export const refreshPassportOidcClientSecret = (id: string) =>
  request.post<PassportOidcClient>(`${base}/oidc-clients/${seg(id)}/secret`, {}, cfg())
