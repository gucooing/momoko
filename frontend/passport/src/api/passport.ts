// 用户通行证接口（只在通行证 server 上）。游戏数据是 MGP 原样透传的 JSON，形状见 src/games/<game>/types.ts。
import { http } from './request'
import type {
  BindPassportAccountRequest,
  ChangePassportEmailRequest,
  ExecutePassportSelfServiceRequest,
  ExecutePassportSelfServiceResponse,
  GetPassportConfigResponse,
  ListPassportActivityResponse,
  ListPassportAppsResponse,
  ListPassportDevicesResponse,
  ListPassportGameRecordsResponse,
  ListPassportLinksResponse,
  ListPassportMessagesResponse,
  ListPassportSelfServiceResponse,
  PassportDataResponse,
  PassportGameResponse,
  PassportLink,
  PassportLoginRequest,
  PassportLoginResponse,
  PassportMe,
  PassportOidcAuthorizationInfo,
  PassportOidcAuthorizeRequest,
  PassportOidcAuthorizeResponse,
  PassportRecoveryCodes,
  PassportRegisterRequest,
  PresignPassportAvatarResponse,
  ReadPassportMessagesResponse,
  RedeemPassportGiftCodeResponse,
  SearchPassportBindServersResponse,
  SendPassportEmailCodeResponse,
  SetupPassportTotpResponse,
  UpdatePassportProfileRequest,
} from '@/types/v1/passport'

const seg = encodeURIComponent

type QueryValue = string | number | boolean | undefined | null

// ---------- 公开 ----------
export const getConfig = () => http.get<GetPassportConfigResponse>('/config')
export const sendRegisterCode = (email: string) =>
  http.post<SendPassportEmailCodeResponse>('/auth/register/email-code', { email })
export const register = (req: Partial<PassportRegisterRequest>) => http.post<PassportLoginResponse>('/auth/register', req)
export const sendLoginCode = (email: string) => http.post<SendPassportEmailCodeResponse>('/auth/login/email-code', { email })
export const login = (req: Partial<PassportLoginRequest>) => http.post<PassportLoginResponse>('/auth/login', req)
export const loginMfa = (mfaToken: string, code: string, recoveryCode = '') =>
  http.post<PassportLoginResponse>('/auth/login/mfa', { mfaToken, code, recoveryCode })
export const sendResetCode = (email: string) =>
  http.post<SendPassportEmailCodeResponse>('/auth/password/reset/email-code', { email })
export const resetPassword = (email: string, code: string, password: string) =>
  http.post<unknown>('/auth/password/reset', { email, code, password })

// ---------- 账号 ----------
export const logout = () => http.post<unknown>('/auth/logout')
export const getMe = () => http.get<PassportMe>('/me')
export const updateProfile = (req: Partial<UpdatePassportProfileRequest>) => http.put<PassportMe>('/me/profile', req)
export const presignAvatar = (contentType: string, size: number) =>
  http.post<PresignPassportAvatarResponse>('/me/avatar/presign', { contentType, size })
export const completeAvatar = (token: string) => http.post<PassportMe>('/me/avatar/complete', { token })
export const changePassword = (oldPassword: string, newPassword: string) =>
  http.put<unknown>('/me/password', { oldPassword, newPassword })
export const sendEmailChangeCode = (email: string) => http.post<SendPassportEmailCodeResponse>('/me/email/code', { email })
export const changeEmail = (req: ChangePassportEmailRequest) => http.put<PassportMe>('/me/email', req)
export const setEmailLogin = (enabled: boolean) => http.put<PassportMe>('/me/email-login', { enabled })
export const setupTotp = () => http.post<SetupPassportTotpResponse>('/me/totp/setup')
export const enableTotp = (code: string) => http.post<PassportRecoveryCodes>('/me/totp/enable', { code })
export const disableTotp = (password: string, code: string) => http.post<PassportMe>('/me/totp/disable', { password, code })
export const regenerateRecoveryCodes = (code: string) => http.post<PassportRecoveryCodes>('/me/totp/recovery-codes', { code })
export const listDevices = () => http.get<ListPassportDevicesResponse>('/me/devices')
export const revokeDevice = (sessionId: string) => http.del<unknown>(`/me/devices/${seg(sessionId)}`)
export const logoutOthers = () => http.post<unknown>('/me/devices/logout-others')
export const listApps = () => http.get<ListPassportAppsResponse>('/me/apps')
export const revokeApp = (id: string) => http.del<unknown>(`/me/apps/${seg(id)}`)
export const requestDeletion = (password: string) => http.post<PassportMe>('/me/deletion', { password })
export const cancelDeletion = () => http.del<PassportMe>('/me/deletion')
export const listMessages = (page = 1, pageSize = 20, unreadOnly = false) =>
  http.get<ListPassportMessagesResponse>('/me/messages', { page, pageSize, unreadOnly })
export const readMessages = (ids: string[], all = false) =>
  http.post<ReadPassportMessagesResponse>('/me/messages/read', { ids, all })
export const listActivity = (limit = 10) => http.get<ListPassportActivityResponse>('/me/activity', { limit })

// ---------- 游戏账号 ----------
export const listLinks = () => http.get<ListPassportLinksResponse>('/links')
export const setPrimaryLink = (id: string) => http.post<unknown>(`/links/${seg(id)}/primary`)
export const unbindLink = (id: string) => http.post<unknown>(`/links/${seg(id)}/unbind`)
export const searchBindServers = (q: string, game = '') =>
  http.get<SearchPassportBindServersResponse>('/bind/servers', { q, game })
export const bindAccount = (req: BindPassportAccountRequest) => http.post<PassportLink>('/bind', req)

// ---------- 游戏空间 ----------
const gbase = (serverId: string) => `/games/${seg(serverId)}`

/** 把查询参数对象转成透传给 MGP 的查询串（空值省略）。 */
export const mgpQuery = (params: Record<string, QueryValue>) => {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '') continue
    q.set(k, String(v))
  }
  return q.toString()
}

export const getGame = (serverId: string) => http.get<PassportGameResponse>(gbase(serverId))
export const getGameBrief = (serverId: string) => http.get<PassportGameResponse>(`${gbase(serverId)}/brief`)

/** 自己的游戏视图（MGP /players/{我的 pid}/views/{view}），返回 data 原样。 */
export const getGameView = async <T>(serverId: string, view: string, query: Record<string, QueryValue> = {}) =>
  (await http.get<PassportDataResponse>(`${gbase(serverId)}/views/${seg(view)}`, { query: mgpQuery(query) })).data as T

/** 目录（名称与素材 ID，每次最多 50 条）。 */
export const getGameCatalog = async <T>(serverId: string, kind: string, query: Record<string, QueryValue> = {}) =>
  (await http.get<PassportDataResponse>(`${gbase(serverId)}/catalogs/${seg(kind)}`, { query: mgpQuery(query) })).data as T

/** 正在进行的卡池（id、名称、横幅素材与起止时间）。 */
export const listOngoing = async <T>(serverId: string) =>
  (await http.get<PassportDataResponse>(`${gbase(serverId)}/ongoing`)).data as T

export const listSelfService = (serverId: string) =>
  http.get<ListPassportSelfServiceResponse>(`${gbase(serverId)}/self-service`)

export const executeSelfService = (
  req: Partial<ExecutePassportSelfServiceRequest> & { serverId: string; command: string },
) =>
  http.post<ExecutePassportSelfServiceResponse>(`${gbase(req.serverId)}/self-service/${seg(req.command)}`, {
    params: req.params ?? {},
    reason: req.reason ?? '',
    totpCode: req.totpCode ?? '',
  })

export const listGameRequests = (serverId: string, page = 1, pageSize = 10) =>
  http.get<ListPassportGameRecordsResponse>(`${gbase(serverId)}/requests`, { page, pageSize })
export const listGameRecords = (serverId: string, page = 1, pageSize = 10) =>
  http.get<ListPassportGameRecordsResponse>(`${gbase(serverId)}/records`, { page, pageSize })
export const redeemGiftCode = (serverId: string, code: string) =>
  http.post<RedeemPassportGiftCodeResponse>(`${gbase(serverId)}/redeem`, { code })

// ---------- OIDC 授权确认 ----------
export const getOidcAuthorization = (req: Partial<PassportOidcAuthorizeRequest>) =>
  http.get<PassportOidcAuthorizationInfo>('/oidc/authorize', req as Record<string, unknown>)
export const approveOidcAuthorization = (req: Partial<PassportOidcAuthorizeRequest>) =>
  http.post<PassportOidcAuthorizeResponse>('/oidc/authorize', req)
