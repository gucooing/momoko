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
} from '@/types/v1/auth'
import type { MeInfoRequest, MeInfoResponse } from '@/types/v1/user'
import type { MePermissionsRequest, MePermissionsResponse } from '@/types/v1/system'

export const login = (params: LoginRequest) => {
  return request.post<LoginResponse>('/auth/login', params)
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
