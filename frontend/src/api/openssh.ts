import request from '@/utils/request'
import type {
  BatchTestSSHHostsRequest,
  BatchTestSSHHostsResponse,
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

export const testSshHost = (params: TestSSHHostRequest) => {
  return request.post<TestSSHHostResponse>(`/openssh/host/${params.id}/test`,params)
}

export const batchTestSshHosts = (data: BatchTestSSHHostsRequest) => {
  return request.post<BatchTestSSHHostsResponse>('/openssh/host/batch-test', data)
}
