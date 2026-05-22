import request from '@/utils/request'
import type {
  AdminAddRoleRequest,
  AdminAddRoleResponse,
  AdminDeleteRoleRequest,
  AdminDeleteRoleResponse,
  AdminEditRoleRequest,
  AdminEditRoleResponse,
  AdminRoleRequest,
  AdminRoleResponse,
  AdminRolesRequest,
  AdminRolesResponse,
} from '@/types/v1/system'

export const rolePage = (params: AdminRolesRequest) => {
  return request.get<AdminRolesResponse>('/role/admin', { params })
}

export const createRole = (data: AdminAddRoleRequest) => {
  return request.post<AdminAddRoleResponse>('/role/admin/add', data)
}

export const roleInfo = (params: AdminRoleRequest) => {
  return request.get<AdminRoleResponse>(`/role/admin/${params.roleId}`)
}

export const updateRole = (data: AdminEditRoleRequest) => {
  return request.post<AdminEditRoleResponse>(`/role/admin/${data.roleId}`, data)
}

export const deleteRole = (params: AdminDeleteRoleRequest) => {
  return request.delete<AdminDeleteRoleResponse>('/role/admin/del', {
    params,
  })
}
