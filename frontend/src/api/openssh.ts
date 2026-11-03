import request from '@/utils/request'
import type {
  CreateSSHHostRequest,
  CreateSSHHostResponse,
  DeleteSSHHostRequest,
  DeleteSSHHostResponse,
  GetSSHHostInfoRequest,
  GetSSHHostInfoResponse,
  GetSSHHostsRequest,
  GetSSHHostsResponse,
  ShareSSHHostRequest,
  ShareSSHHostResponse,
  TestSSHHostRequest,
  TestSSHHostResponse,
  TestSSHHostsRequest,
  TestSSHHostsResponse,
  UpdateSSHHostRequest,
  UpdateSSHHostResponse,
} from '@/types/v1/openssh'

export const getSshHosts = (params: GetSSHHostsRequest) => {
  return request.get<GetSSHHostsResponse>('/openssh/host', { params })
}

export const createSshHost = (data: CreateSSHHostRequest) => {
  return request.post<CreateSSHHostResponse>('/openssh/host', data)
}

export const getSshHostInfo = (params: GetSSHHostInfoRequest) => {
  return request.get<GetSSHHostInfoResponse>(`/openssh/host/${params.id}`)
}

export const updateSshHost = (data: UpdateSSHHostRequest) => {
  return request.put<UpdateSSHHostResponse>(`/openssh/host/${data.id}`, data)
}

export const deleteSshHost = (params: DeleteSSHHostRequest) => {
  return request.delete<DeleteSSHHostResponse>(`/openssh/host/${params.id}`)
}

export const shareSshHost = (data: ShareSSHHostRequest) => {
  return request.post<ShareSSHHostResponse>(`/openssh/host/${data.id}/share`, data)
}

// 测试当前表单草稿连通性（不写库状态）
export const testSshHost = (data: TestSSHHostRequest) => {
  return request.post<TestSSHHostResponse>('/openssh/host/test', data)
}

// 批量测试已保存主机并记录结果；ids 为空表示全部可访问主机
export const testSshHosts = (data: TestSSHHostsRequest) => {
  return request.post<TestSSHHostsResponse>('/openssh/host/batch-test', data, { timeout: 120000 })
}
