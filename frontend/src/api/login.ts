import request from '@/utils/request'
import { sortMenuTreeByOrder } from '@/utils/menu'
import type {
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

export const myLoginLogs = (params: MyLoginLogsRequest) => {
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

export const refreshTokenRequest = (params: RefreshRequest) => {
  return request.post<RefreshResponse>('/auth/refresh', params)
}

export const logoutRequest = (params: LogoutRequest = {}) => {
  return request.post<LogoutResponse>('/auth/logout', params)
}

export const devicesRequest = (params: DevicesRequest = {}) => {
  return request.get<DevicesResponse>('/auth/devices', { params })
}

/** 下线指定会话：走 logout，body 带 sessionId（后端已移除 DELETE /auth/devices/{id}）。 */
export const deleteLoginDeviceRequest = (params: { sessionId: string }) => {
  return logoutRequest({ sessionId: params.sessionId })
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
