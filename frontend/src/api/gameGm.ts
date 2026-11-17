import request from '@/utils/request'
import type { SearchGameServersRequest, SearchGameServersResponse } from '@/types/v1/game_common'
import type {
  ApproveGmExecutionRequest,
  DeleteGmClientLogResponse,
  GetGmContextResponse,
  GmClientLog,
  GmExecution,
  GmSelfServicePolicy,
  ListGmClientLogsRequest,
  ListGmClientLogsResponse,
  ListGmExecutionsRequest,
  ListGmExecutionsResponse,
  ListGmSelfServiceResponse,
  RejectGmExecutionRequest,
  UpdateGmSelfServiceRequest,
} from '@/types/v1/game_gm'

// GM 运营的通用接口（权限 game:gm）：服务器上下文、GM 记录与审批、玩家自助设置、玩家上传的客户端日志。
// 各游戏的查询与操作见 src/api/<游戏>.ts（学マス：src/api/gakumas.ts）。

const base = (serverId: string) => `/game/gm/${encodeURIComponent(serverId)}`
const seg = (value: string) => encodeURIComponent(value)

export const searchGmServers = (params: Partial<SearchGameServersRequest>) =>
  request.get<SearchGameServersResponse>('/game/gm/servers', { params })

export const getGmContext = (serverId: string) => request.get<GetGmContextResponse>(`${base(serverId)}/context`)

export const listGmExecutions = (params: ListGmExecutionsRequest) => {
  const { serverId, ...rest } = params
  return request.get<ListGmExecutionsResponse>(`${base(serverId)}/executions`, { params: rest })
}

export const getGmExecution = (serverId: string, id: string) =>
  request.get<GmExecution>(`${base(serverId)}/executions/${seg(id)}`)

export const approveGmExecution = (data: ApproveGmExecutionRequest) =>
  request.post<GmExecution>(`${base(data.serverId)}/executions/${seg(data.id)}/approve`, data)

export const rejectGmExecution = (data: RejectGmExecutionRequest) =>
  request.post<GmExecution>(`${base(data.serverId)}/executions/${seg(data.id)}/reject`, data)

export const listGmSelfService = (serverId: string) =>
  request.get<ListGmSelfServiceResponse>(`${base(serverId)}/self-service`)

export const updateGmSelfService = (data: UpdateGmSelfServiceRequest) =>
  request.put<GmSelfServicePolicy>(`${base(data.serverId)}/self-service/${seg(data.command)}`, data)

/** 客户端日志（全部服务器，从新到旧，分页，不含日志原文）；GM 玩家详情按服务器与玩家筛选。 */
export const listGmClientLogs = (params: Partial<ListGmClientLogsRequest>) =>
  request.get<ListGmClientLogsResponse>('/game/client-logs', { params })

/** 一份客户端日志（含日志原文）。 */
export const getGmClientLog = (id: string) => request.get<GmClientLog>(`/game/client-logs/${seg(id)}`)

export const deleteGmClientLog = (id: string) => request.delete<DeleteGmClientLogResponse>(`/game/client-logs/${seg(id)}`)

/** GM 实时动态（SSE）地址：用 fetch 带 Authorization 头读取（EventSource 不能带头）。 */
export const gmEventsPath = (serverId: string) => `/game/gm-events/${seg(serverId)}`
