import request from '@/utils/request'
import { sortMenuTreeByOrder } from '@/utils/menu'
import type {
  DeleteMessagesRequest,
  DeleteMessagesResponse,
  ListMessagesRequest,
  ListMessagesResponse,
  PublishMessageRequest,
  PublishMessageResponse,
  ReadMessagesRequest,
  ReadMessagesResponse,
  DevicesRequest,
  DevicesResponse,
  LoginRequest,
  LoginResponse,
  LogoutRequest,
  LogoutResponse,
  RefreshRequest,
  RefreshResponse,
  RegisterRequest,
  RegisterResponse,
  ResetPasswordRequest,
  ResetPasswordResponse,
  SendResetPasswordEmailCodeRequest,
  SendResetPasswordEmailCodeResponse,
  SendLoginEmailCodeRequest,
  SendLoginEmailCodeResponse,
  SendRegisterEmailCodeRequest,
  SendRegisterEmailCodeResponse,
} from '@/types/v1/auth'
import type {
  MeInfoRequest,
  MeInfoResponse,
  MyLoginLogsRequest,
  MyLoginLogsResponse,
} from '@/types/v1/user'

export const myLoginLogs = (params: Partial<MyLoginLogsRequest>) => {
  return request.get<MyLoginLogsResponse>('/user/me/login-logs', { params })
}
import type {
  LoginConfigRequest,
  LoginConfigResponse,
  MePermissionsRequest,
  MePermissionsResponse,
  CheckUpdateRequest,
  CheckUpdateResponse,
  UpdateLoginConfigRequest,
  UpdateLoginConfigResponse,
} from '@/types/v1/system'

export const getLoginConfig = (params: LoginConfigRequest = {}) => {
  return request.get<LoginConfigResponse>('/system/login-config', { params })
}

export const updateLoginConfig = (params: UpdateLoginConfigRequest) => {
  return request.put<UpdateLoginConfigResponse>('/system/login-config', params)
}

export const login = (params: LoginRequest) => {
  return request.post<LoginResponse>('/auth/login', params)
}

export const sendLoginEmailCode = (params: SendLoginEmailCodeRequest) => {
  return request.post<SendLoginEmailCodeResponse>('/auth/login/email-code', params)
}

export const sendRegisterEmailCode = (params: SendRegisterEmailCodeRequest) => {
  return request.post<SendRegisterEmailCodeResponse>('/auth/register/email-code', params)
}

export const register = (params: RegisterRequest) => {
  return request.post<RegisterResponse>('/auth/register', params)
}

// 找回密码：向注册邮箱发送重置验证码
export const sendResetPasswordEmailCode = (params: SendResetPasswordEmailCodeRequest) => {
  return request.post<SendResetPasswordEmailCodeResponse>('/auth/password/reset/email-code', params)
}

// 找回密码：邮箱验证码 + 新密码重置（成功后该账号所有会话失效）
export const resetPassword = (params: ResetPasswordRequest) => {
  return request.post<ResetPasswordResponse>('/auth/password/reset', params)
}

export const refreshTokenRequest = (params: RefreshRequest) => {
  return request.post<RefreshResponse>('/auth/refresh', params)
}

export const logoutRequest = (params: Partial<LogoutRequest> = {}) => {
  return request.post<LogoutResponse>('/auth/logout', params)
}

export const devicesRequest = (params: DevicesRequest = {}) => {
  return request.get<DevicesResponse>('/auth/devices', { params })
}

/** 下线指定会话：走 logout，body 带 sessionId（后端已移除 DELETE /auth/devices/{id}）。 */
export const deleteLoginDeviceRequest = (params: { sessionId: string }) => {
  return logoutRequest({ sessionId: params.sessionId })
}

/** 下线除当前会话外的所有登录设备。 */
export const logoutOtherDevicesRequest = () => {
  return logoutRequest({ others: true })
}

export const userMeInfoRequest = (params: MeInfoRequest = {}) => {
  return request.get<MeInfoResponse>('/user/me', { params })
}

export const mePermissionsRequest = (params: MePermissionsRequest = {}) => {
  return request.get<MePermissionsResponse>('/permissions/me', { params }).then((response) => {
    return {
      ...response,
      data: {
        ...response.data,
        menus: sortMenuTreeByOrder(response.data?.menus || []),
      },
    }
  })
}

// 检查更新（需 system:update 权限）：查询远程最新发行版并与当前版本比较。
export const checkUpdateRequest = (params: CheckUpdateRequest = {}) => {
  return request.get<CheckUpdateResponse>('/system/update/check', { params })
}

// 站内消息：我的消息列表（含全部数 / 未读数）
export const listMessagesRequest = (params: Partial<ListMessagesRequest> = {}) => {
  return request.get<ListMessagesResponse>('/auth/messages', { params })
}

// 标记已读（all=true 时全部）
export const readMessagesRequest = (params: Partial<ReadMessagesRequest>) => {
  return request.post<ReadMessagesResponse>('/auth/messages/read', params)
}

// 删除消息（all=true 时全部）
export const deleteMessagesRequest = (params: Partial<DeleteMessagesRequest>) => {
  return request.post<DeleteMessagesResponse>('/auth/messages/delete', params)
}

// 发布通知给全部启用的用户（需 system_config:edit）
export const publishMessageRequest = (params: PublishMessageRequest) => {
  return request.post<PublishMessageResponse>('/auth/messages/publish', params)
}
