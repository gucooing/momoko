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
  RefreshAPIKeyRequest,
  RefreshAPIKeyResponse,
  DeleteAPIKeyRequest,
  DeleteAPIKeyResponse,
  ListNodesRequest,
  ListNodesResponse,
  CreateNodeRequest,
  CreateNodeResponse,
  UpdateNodeRequest,
  UpdateNodeResponse,
  DeleteNodeRequest,
  DeleteNodeResponse,
  TestNodeRequest,
  TestNodeResponse,
  GetNodeContextRequest,
  GetNodeContextResponse,
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

export const refreshAPIKey = (data: RefreshAPIKeyRequest) => {
  return request.post<RefreshAPIKeyResponse>(`/node/api-key/${data.id}/refresh`, data)
}

export const deleteAPIKey = (data: DeleteAPIKeyRequest) => {
  return request.delete<DeleteAPIKeyResponse>(`/node/api-key/${data.id}`)
}

export const listNodes = (params: ListNodesRequest = {}) => {
  return request.get<ListNodesResponse>('/node/nodes', { params })
}

export const createNode = (data: CreateNodeRequest) => {
  return request.post<CreateNodeResponse>('/node/nodes', data)
}

export const updateNode = (data: UpdateNodeRequest) => {
  return request.put<UpdateNodeResponse>(`/node/nodes/${data.id}`, data)
}

export const deleteNode = (data: DeleteNodeRequest) => {
  return request.delete<DeleteNodeResponse>(`/node/nodes/${data.id}`)
}

export const testNode = (data: TestNodeRequest) => {
  return request.post<TestNodeResponse>('/node/nodes/test', data)
}

export const getNodeContext = (data: GetNodeContextRequest) => {
  return request.get<GetNodeContextResponse>(`/node/nodes/${data.id}/context`)
}
