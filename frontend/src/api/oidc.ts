import request from '@/utils/request'
import type {
  CreateOIDCAuthorizationCodeRequest,
  CreateOIDCAuthorizationCodeResponse,
  CreateOIDCClientRequest,
  CreateOIDCClientResponse,
  DeleteOIDCClientRequest,
  DeleteOIDCClientResponse,
  ListOIDCClientsRequest,
  ListOIDCClientsResponse,
  OIDCAuthorizationInfoRequest,
  OIDCAuthorizationInfoResponse,
  OIDCConfigRequest,
  OIDCConfigResponse,
  RefreshOIDCClientSecretRequest,
  RefreshOIDCClientSecretResponse,
  UpdateOIDCClientRequest,
  UpdateOIDCClientResponse,
  UpdateOIDCConfigRequest,
  UpdateOIDCConfigResponse,
} from '@/types/v1/oidc'

export const getOIDCConfig = (params: OIDCConfigRequest = {}) => {
  return request.get<OIDCConfigResponse>('/oidc/config', { params })
}

export const updateOIDCConfig = (data: UpdateOIDCConfigRequest) => {
  return request.put<UpdateOIDCConfigResponse>('/oidc/config', data)
}

export const listOIDCClients = (params: ListOIDCClientsRequest) => {
  return request.get<ListOIDCClientsResponse>('/oidc/clients', { params })
}

export const createOIDCClient = (data: CreateOIDCClientRequest) => {
  return request.post<CreateOIDCClientResponse>('/oidc/clients', data)
}

export const updateOIDCClient = (data: UpdateOIDCClientRequest) => {
  return request.put<UpdateOIDCClientResponse>(`/oidc/clients/${data.id}`, data)
}

export const deleteOIDCClient = (params: DeleteOIDCClientRequest) => {
  return request.delete<DeleteOIDCClientResponse>(`/oidc/clients/${params.id}`)
}

export const refreshOIDCClientSecret = (params: RefreshOIDCClientSecretRequest) => {
  return request.post<RefreshOIDCClientSecretResponse>(`/oidc/clients/${params.id}/secret`, {})
}

export const getOIDCAuthorizationInfo = (data: OIDCAuthorizationInfoRequest) => {
  return request.post<OIDCAuthorizationInfoResponse>('/oidc/authorize-info', data)
}

export const createOIDCAuthorizationCode = (data: CreateOIDCAuthorizationCodeRequest) => {
  return request.post<CreateOIDCAuthorizationCodeResponse>('/oidc/authorize-code', data)
}
