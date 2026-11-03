import request from '@/utils/request'
import type {
  ListTunnelsRequest,
  ListTunnelsResponse,
  CreateTunnelRequest,
  CreateTunnelResponse,
  GetTunnelRequest,
  GetTunnelResponse,
  UpdateTunnelRequest,
  UpdateTunnelResponse,
  DeleteTunnelRequest,
  DeleteTunnelResponse,
  GetTunnelStatsRequest,
  GetTunnelStatsResponse,
  GetFrpsConfigResponse,
  UpdateFrpsConfigRequest,
  UpdateFrpsConfigResponse,
  RestartFrpsResponse,
  ResetTunnelCredentialRequest,
  ResetTunnelCredentialResponse,
  GetFrpsLogsRequest,
  GetFrpsLogsResponse,
} from '@/types/v1/tunnel'

export const listTunnels = (params: ListTunnelsRequest) => {
  return request.get<ListTunnelsResponse>('/tunnel/tunnels', { params })
}

export const createTunnel = (data: CreateTunnelRequest) => {
  return request.post<CreateTunnelResponse>('/tunnel/tunnels', data)
}

export const getTunnel = (params: GetTunnelRequest) => {
  return request.get<GetTunnelResponse>(`/tunnel/tunnels/${params.id}`)
}

export const updateTunnel = (data: UpdateTunnelRequest) => {
  return request.put<UpdateTunnelResponse>(`/tunnel/tunnels/${data.id}`, data)
}

export const deleteTunnel = (params: DeleteTunnelRequest) => {
  return request.delete<DeleteTunnelResponse>(`/tunnel/tunnels/${params.id}`)
}

export const getTunnelStats = (params: GetTunnelStatsRequest) => {
  return request.get<GetTunnelStatsResponse>(`/tunnel/tunnels/${params.id}/stats`, {
    params: { startTimeMs: params.startTimeMs, endTimeMs: params.endTimeMs },
  })
}

export const getFrpsConfig = () => {
  return request.get<GetFrpsConfigResponse>('/tunnel/frps-config')
}

export const updateFrpsConfig = (data: UpdateFrpsConfigRequest) => {
  return request.put<UpdateFrpsConfigResponse>('/tunnel/frps-config', data)
}

export const restartFrps = () => {
  return request.post<RestartFrpsResponse>('/tunnel/frps-restart', {})
}

export const resetTunnelCredential = (data: ResetTunnelCredentialRequest) => {
  return request.post<ResetTunnelCredentialResponse>(`/tunnel/tunnels/${data.id}/credential/reset`, data)
}

export const getFrpsLogs = (params: GetFrpsLogsRequest) => {
  return request.get<GetFrpsLogsResponse>('/tunnel/frps-logs', { params })
}
