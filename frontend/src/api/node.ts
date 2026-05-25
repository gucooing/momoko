import request from '@/utils/request'
import type {
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  ListAPIKeysRequest,
  ListAPIKeysResponse,
  CopyAPIKeyRequest,
  CopyAPIKeyResponse,
  UpdateAPIKeyRequest,
  UpdateAPIKeyResponse,
} from '@/types/v1/node'

export const createAPIKey = (data: CreateAPIKeyRequest) => {
  return request.post<CreateAPIKeyResponse>('/node/api-key', data)
}

export const listAPIKeys = (params: ListAPIKeysRequest) => {
  return request.get<ListAPIKeysResponse>('/node/api-key', { params })
}

export const copyAPIKey = (data: CopyAPIKeyRequest) => {
  return request.post<CopyAPIKeyResponse>(`/node/api-key/${data.id}/copy`, data)
}

export const updateAPIKey = (data: UpdateAPIKeyRequest) => {
  return request.put<UpdateAPIKeyResponse>(`/node/api-key/${data.id}`, data)
}
