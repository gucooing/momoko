import request from '@/utils/request'
import { sortMenuTreeByOrder } from '@/utils/menu'
import type {
  DelLoginRequest,
  DelLoginResponse,
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

export const deleteLoginDeviceRequest = (params: DelLoginRequest) => {
  return request.delete<DelLoginResponse>(`/auth/devices/${params.id}`)
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
