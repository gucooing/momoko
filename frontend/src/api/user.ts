import request from '@/utils/request'
import type { UpdatePasswordRequest, UpdatePasswordResponse } from '@/types/v1/auth'
import type {
  AddUserRequest,
  AddUserResponse,
  BatchUpdateUserRequest,
  BatchUpdateUserResponse,
  DeleteUserRequest,
  DeleteUserResponse,
  EditUserRequest,
  EditUserResponse,
  ListUserRequest,
  ListUserResponse,
  ResetUserPasswordRequest,
  ResetUserPasswordResponse,
  UserInfoRequest,
  UpdateMeRequest,
  UpdateMeResponse,
  UpdateMyAppearanceRequest,
  UpdateMyAppearanceResponse,
  UserInfoResponse,
} from '@/types/v1/user'

export const userPage = (params: ListUserRequest) => {
  return request.get<ListUserResponse>('/user/list', { params })
}

export const createUser = (data: AddUserRequest) => {
  return request.post<AddUserResponse>('/user/add', data)
}

export const userInfo = (params: UserInfoRequest) => {
  return request.get<UserInfoResponse>(`/user/${params.userId}`)
}

export const updateUser = (data: EditUserRequest) => {
  return request.put<EditUserResponse>(`/user/${data.userId}`, data)
}

export const deleteUser = (params: DeleteUserRequest) => {
  return request.delete<DeleteUserResponse>('/user/del', {
    params,
  })
}

export const batchUpdateUser = (data: BatchUpdateUserRequest) => {
  return request.post<BatchUpdateUserResponse>('/user/batch', data)
}

export const resetUserPassword = (data: ResetUserPasswordRequest) => {
  return request.put<ResetUserPasswordResponse>(`/user/${data.userId}/password`, data)
}

export const updateMeRequest = (data: UpdateMeRequest) => {
  return request.put<UpdateMeResponse>('/auth/me', data)
}

// 外观偏好跟随账号同步（不记操作日志）
export const updateMyAppearanceRequest = (data: UpdateMyAppearanceRequest) => {
  return request.put<UpdateMyAppearanceResponse>('/user/me/appearance', data)
}

export const updatePasswordRequest = (data: UpdatePasswordRequest) => {
  return request.put<UpdatePasswordResponse>('/auth/password', data)
}
