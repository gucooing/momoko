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
  GetTerminalInfoRequest,
  GetTerminalInfoResponse,
  GetInstanceTypesResponse,
  RestartInstanceRequest,
  RestartInstanceResponse,
  RestartTerminalRequest,
  RestartTerminalResponse,
  StartInstanceRequest,
  StartInstanceResponse,
  StartTerminalRequest,
  StartTerminalResponse,
  StopInstanceRequest,
  StopInstanceResponse,
  StopTerminalRequest,
  StopTerminalResponse,
  UpdateInstanceRequest,
  UpdateInstanceResponse,
  UpdateInstanceTypeRequest,
  UpdateInstanceTypeResponse,
  GetInstanceFileListRequest,
  GetInstanceFileListResponse,
  CreateInstanceFileRequest,
  CreateInstanceFileResponse,
  BatchDeleteInstanceFileRequest,
  BatchDeleteInstanceFileResponse,
  BatchCompressInstanceFileRequest,
  BatchCompressInstanceFileResponse,
  UnzipInstanceFileRequest,
  UnzipInstanceFileResponse,
  OpenInstanceFileRequest,
  OpenInstanceFileResponse,
  EditInstanceFileRequest,
  EditInstanceFileResponse,
  RenameInstanceFileRequest,
  RenameInstanceFileResponse,
  InstanceFilePreSignRequest,
  InstanceFilePreSignResponse,
  InstanceFilePreSignUploadRequest,
  InstanceFilePreSignUploadResponse,
  CopyInstanceFileRequest,
  CopyInstanceFileResponse,
  CutInstanceFileRequest,
  CutInstanceFileResponse,
} from '@/types/v1/instance'
import { encodeBytesToBase64 } from '@/utils/filePreview'

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

export const createInstanceType = (
  data: CreateInstanceTypeRequest & { isEnable?: boolean },
) => {
  return request.post<CreateInstanceTypeResponse>('/instance/type', data)
}

export const updateInstanceType = (data: UpdateInstanceTypeRequest) => {
  return request.put<UpdateInstanceTypeResponse>(`/instance/type/${data.id}`, data)
}

export const deleteInstanceType = (params: DelInstanceTypeRequest) => {
  return request.delete<DelInstanceTypeResponse>(`/instance/type/${params.id}`)
}

export const getTerminalInfoRequest = (params: GetTerminalInfoRequest = {}) => {
  return request.get<GetTerminalInfoResponse>('/instance/terminal', { params })
}

export const startTerminalRequest = (data: StartTerminalRequest = {}) => {
  return request.post<StartTerminalResponse>('/instance/terminal/start', data)
}

export const stopTerminalRequest = (data: StopTerminalRequest = {}) => {
  return request.post<StopTerminalResponse>('/instance/terminal/stop', data)
}

export const restartTerminalRequest = (data: RestartTerminalRequest = {}) => {
  return request.post<RestartTerminalResponse>('/instance/terminal/restart', data)
}

export const getInstanceFileListRequest = (params: GetInstanceFileListRequest) => {
  return request.get<GetInstanceFileListResponse>(`/instance/file/${params.id}`, { params })
}

export const createInstanceFileRequest = (data: CreateInstanceFileRequest) => {
  return request.post<CreateInstanceFileResponse>(`/instance/file/create/${data.id}`, data)
}

export const batchDeleteInstanceFileRequest = (data: BatchDeleteInstanceFileRequest) => {
  return request.post<BatchDeleteInstanceFileResponse>(`/instance/file/deletes/${data.id}`, data)
}

export const openInstanceFileRequest = (data: OpenInstanceFileRequest) => {
  return request.post<OpenInstanceFileResponse>(`/instance/file/open/${data.id}`, data)
}

export const getInstanceFilePreSignRequest = (params: InstanceFilePreSignRequest) => {
  return request.get<InstanceFilePreSignResponse>(`/instance/file/pre-sign/${params.id}`, {
    params,
  })
}

export const getInstanceFilePreSignUploadRequest = (
  params: InstanceFilePreSignUploadRequest,
) => {
  return request.get<InstanceFilePreSignUploadResponse>(
    `/instance/file/upload/pre-sign/${params.id}`,
    { params },
  )
}

export const batchCompressInstanceFileRequest = (data: BatchCompressInstanceFileRequest) => {
  return request.post<BatchCompressInstanceFileResponse>(
    `/instance/file/compress/${data.id}`,
    data,
  )
}

export const renameInstanceFileRequest = (data: RenameInstanceFileRequest) => {
  return request.post<RenameInstanceFileResponse>(`/instance/file/rename/${data.id}`, data)
}

export const unzipInstanceFileRequest = (data: UnzipInstanceFileRequest) => {
  return request.post<UnzipInstanceFileResponse>(`/instance/file/unzip/${data.id}`, data)
}

export const copyInstanceFileRequest = (data: CopyInstanceFileRequest) => {
  return request.post<CopyInstanceFileResponse>(`/instance/file/copy/${data.id}`, data)
}


export const editInstanceFileRequest = (data: EditInstanceFileRequest) => {
  const payload = {
    id: data.id,
    path: data.path,
    content: encodeBytesToBase64(data.content),
  }

  return request.post<EditInstanceFileResponse>(`/instance/file/edit/${data.id}`, payload)
}

export const cutInstanceFileRequest = (data: CutInstanceFileRequest) => {
  return request.post<CutInstanceFileResponse>(`/instance/file/cut/${data.id}`, data)
}
