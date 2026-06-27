import request from '@/utils/request'
import type {
  CreateInstanceRequest,
  CreateInstanceResponse,
  DelInstanceRequest,
  DelInstanceLogRequest,
  DelInstanceLogResponse,
  DelInstanceResponse,
  CreateInstanceTypeRequest,
  CreateInstanceTypeResponse,
  DelInstanceTypeRequest,
  DelInstanceTypeResponse,
  GetInstanceInfoRequest,
  GetInstanceInfoResponse,
  GetInstancesRequest,
  GetInstancesResponse,
  GetInstanceTypesRequest,
  GetInstanceTypesResponse,
  RestartInstanceRequest,
  RestartInstanceResponse,
  StartInstanceRequest,
  StartInstanceResponse,
  StopInstanceRequest,
  StopInstanceResponse,
  UpdateInstanceRequest,
  UpdateInstanceResponse,
  UpdateInstanceTypeRequest,
  UpdateInstanceTypeResponse,
} from '@/types/v1/instance'

// 实例生命周期与类型管理 API。
// 实例「文件管理」相关 HTTP 已拆分到 `@/api/instanceFile`，并由 `@/components/file/fileClient` 统一封装。

export const getInstances = (params: GetInstancesRequest) => {
  return request.get<GetInstancesResponse>('/instance', { params })
}

export const createInstanceRequest = (data: CreateInstanceRequest) => {
  return request.post<CreateInstanceResponse>('/instance', data)
}

export const getInstanceInfoRequest = (params: GetInstanceInfoRequest) => {
  return request.get<GetInstanceInfoResponse>(`/instance/${params.id}`)
}

export const updateInstanceRequest = (data: UpdateInstanceRequest) => {
  return request.put<UpdateInstanceResponse>(`/instance/instance/${data.id}`, data)
}

export const startInstanceRequest = (data: StartInstanceRequest) => {
  return request.post<StartInstanceResponse>(`/instance/instance/start/${data.id}`, data)
}

export const stopInstanceRequest = (data: StopInstanceRequest) => {
  return request.post<StopInstanceResponse>(`/instance/instance/stop/${data.id}`, data)
}

export const restartInstanceRequest = (data: RestartInstanceRequest) => {
  return request.post<RestartInstanceResponse>(`/instance/instance/restart/${data.id}`, data)
}

export const deleteInstanceRequest = (params: DelInstanceRequest) => {
  return request.delete<DelInstanceResponse>(`/instance/instance/del/${params.id}`)
}

export const deleteInstanceLogRequest = (params: DelInstanceLogRequest) => {
  return request.delete<DelInstanceLogResponse>(`/instance/instance/del/log/${params.id}`)
}

export const getInstanceTypes = (params: GetInstanceTypesRequest = {}) => {
  return request.get<GetInstanceTypesResponse>('/instance/type', { params })
}

export const createInstanceType = (data: CreateInstanceTypeRequest & { isEnable?: boolean }) => {
  return request.post<CreateInstanceTypeResponse>('/instance/type', data)
}

export const updateInstanceType = (data: UpdateInstanceTypeRequest) => {
  return request.put<UpdateInstanceTypeResponse>(`/instance/type/${data.id}`, data)
}

export const deleteInstanceType = (params: DelInstanceTypeRequest) => {
  return request.delete<DelInstanceTypeResponse>(`/instance/type/${params.id}`)
}
