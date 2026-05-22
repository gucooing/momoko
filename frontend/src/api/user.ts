import request from '@/utils/request'
import type { UpdatePasswordRequest, UpdatePasswordResponse } from '@/types/v1/auth'
import type {
  AddUserRequest,
  AddUserResponse,
  DeleteUserRequest,
  DeleteUserResponse,
  EditUserRequest,
  EditUserResponse,
  ListUserRequest,
  ListUserResponse,
  UserInfoRequest,
  UpdateMeRequest,
  UpdateMeResponse,
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
  return request.post<EditUserResponse>(`/user/${data.userId}`, data)
}

export const deleteUser = (params: DeleteUserRequest) => {
  return request.delete<DeleteUserResponse>('/user/del', {
    params,
  })
}

export const updateMeRequest = (data: UpdateMeRequest) => {
  return request.put<UpdateMeResponse>('/auth/me', data)
}

export const updatePasswordRequest = (data: UpdatePasswordRequest) => {
  return request.put<UpdatePasswordResponse>('/auth/password', data)
}
