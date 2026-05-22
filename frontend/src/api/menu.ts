import request from '@/utils/request'
import { sortMenuTreeByOrder } from '@/utils/menu'
import type {
  AdminDeletePermissionsRequest,
  AdminPermissionsResponse,
  AdminPermissionsRequest,
  AdminPermissionsInfoRequest,
  AdminPermissionsInfoResponse,
  AdminAddPermissionsRequest,
  AdminAddPermissionsResponse,
  AdminEditPermissionsRequest,
  AdminEditPermissionsResponse,
  AdminDeletePermissionsResponse,
} from '@/types/v1/system'

export const adminPermissionsList = async (params: AdminPermissionsRequest = {}) => {
  const response = await request.get<AdminPermissionsResponse>('/permissions/admin', { params })
  return {
    ...response,
    data: {
      ...response.data,
      menus: sortMenuTreeByOrder(response.data?.menus || []),
    },
  }
}

export const adminPermissionsInfo = (params: AdminPermissionsInfoRequest) => {
  return request.get<AdminPermissionsInfoResponse>(`/permissions/admin/${params.menuId}`)
}

export const adminAddPermissions = (data: AdminAddPermissionsRequest) => {
  return request.post<AdminAddPermissionsResponse>('/permissions/admin/add', data)
}

export const adminEditPermissions = (data: AdminEditPermissionsRequest) => {
  return request.post<AdminEditPermissionsResponse>(`/permissions/admin/${data.menuId}`, data)
}

export const adminDeletePermissions = (params: AdminDeletePermissionsRequest) => {
  return request.delete<AdminDeletePermissionsResponse>(`/permissions/admin/${params.menuId}`)
}
