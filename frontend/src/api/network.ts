import request from '@/utils/request'
import type {
  ListPortForwardsRequest,
  ListPortForwardsResponse,
  CreatePortForwardRequest,
  CreatePortForwardResponse,
  GetPortForwardRequest,
  GetPortForwardResponse,
  UpdatePortForwardRequest,
  UpdatePortForwardResponse,
  DeletePortForwardRequest,
  DeletePortForwardResponse,
  GetPortForwardStatsRequest,
  GetPortForwardStatsResponse,
} from '@/types/v1/network'

export const listPortForwards = (params: ListPortForwardsRequest) => {
  return request.get<ListPortForwardsResponse>('/network/port-forwards', { params })
}

export const createPortForward = (data: CreatePortForwardRequest) => {
  return request.post<CreatePortForwardResponse>('/network/port-forwards', data)
}

export const getPortForward = (params: GetPortForwardRequest) => {
  return request.get<GetPortForwardResponse>(`/network/port-forwards/${params.id}`)
}

export const updatePortForward = (data: UpdatePortForwardRequest) => {
  return request.put<UpdatePortForwardResponse>(`/network/port-forwards/${data.id}`, data)
}

export const deletePortForward = (params: DeletePortForwardRequest) => {
  return request.delete<DeletePortForwardResponse>(`/network/port-forwards/${params.id}`)
}

export const getPortForwardStats = (params: GetPortForwardStatsRequest) => {
  return request.get<GetPortForwardStatsResponse>(`/network/port-forwards/${params.id}/stats`, {
    params: { startTimeMs: params.startTimeMs, endTimeMs: params.endTimeMs },
  })
}
