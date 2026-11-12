import request from '@/utils/request'
import type { SearchGameServersRequest, SearchGameServersResponse } from '@/types/v1/game_common'
import type {
  ApproveGmExecutionRequest,
  DiscardGmConfigDraftResponse,
  ExecuteGmCommandRequest,
  ExecuteGmCommandResponse,
  GetGmConfigResponse,
  GetGmContextResponse,
  GetGmPlayerResponse,
  GmConfigDraft,
  GmConfigRevision,
  GmDataResponse,
  GmExecution,
  GmSelfServicePolicy,
  ListGmConfigRevisionsResponse,
  ListGmConfigsResponse,
  ListGmExecutionsRequest,
  ListGmExecutionsResponse,
  ListGmPlayersRequest,
  ListGmPlayersResponse,
  ListGmSelfServiceResponse,
  PublishGmConfigRequest,
  PublishGmConfigResponse,
  RejectGmExecutionRequest,
  SaveGmConfigDraftRequest,
  UpdateGmSelfServiceRequest,
  ValidateGmConfigResponse,
} from '@/types/v1/game_gm'

// GM 运营（权限 game:gm）。游戏数据是 MGP 原样透传的 JSON，类型见 src/games/mgp.ts 与 src/games/<game>/types.ts。

const base = (serverId: string) => `/game/gm/${encodeURIComponent(serverId)}`
const seg = (value: string) => encodeURIComponent(value)

/** 把查询参数对象转成透传给 MGP 的查询串（空值省略）。 */
export const mgpQuery = (params: Record<string, string | number | boolean | undefined | null>) => {
  const q = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    q.set(key, String(value))
  }
  return q.toString()
}

export const searchGmServers = (params: Partial<SearchGameServersRequest>) =>
  request.get<SearchGameServersResponse>('/game/gm/servers', { params })

export const getGmContext = (serverId: string) => request.get<GetGmContextResponse>(`${base(serverId)}/context`)

export const listGmPlayers = (params: ListGmPlayersRequest) =>
  request.get<ListGmPlayersResponse>(`${base(params.serverId)}/players`, {
    params: { q: params.q, filter: params.filter, cursor: params.cursor, limit: params.limit },
  })

export const getGmPlayer = (serverId: string, pid: string) =>
  request.get<GetGmPlayerResponse>(`${base(serverId)}/players/${seg(pid)}`)

/** 玩家视图（MGP /players/{pid}/views/{view}），返回 data 原样。 */
export const getGmPlayerView = async <T>(serverId: string, pid: string, view: string, query = '') => {
  const { data } = await request.get<GmDataResponse>(`${base(serverId)}/players/${seg(pid)}/views/${seg(view)}`, {
    params: { query },
  })
  return data.data as T
}

/** 全服报表（MGP /reports/{report}），返回 data 原样。 */
export const getGmReport = async <T>(serverId: string, report: string, query = '') => {
  const { data } = await request.get<GmDataResponse>(`${base(serverId)}/reports/${seg(report)}`, { params: { query } })
  return data.data as T
}

/** 选择器目录（MGP /catalogs/{kind}），返回 data 原样。 */
export const getGmCatalog = async <T>(serverId: string, kind: string, query = '') => {
  const { data } = await request.get<GmDataResponse>(`${base(serverId)}/catalogs/${seg(kind)}`, { params: { query } })
  return data.data as T
}

export const executeGmCommand = (data: ExecuteGmCommandRequest) =>
  request.post<ExecuteGmCommandResponse>(`${base(data.serverId)}/commands/${seg(data.command)}`, data)

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

export const listGmConfigs = (serverId: string) => request.get<ListGmConfigsResponse>(`${base(serverId)}/configs`)

export const getGmConfig = (serverId: string, domain: string) =>
  request.get<GetGmConfigResponse>(`${base(serverId)}/configs/${seg(domain)}`)

export const saveGmConfigDraft = (data: SaveGmConfigDraftRequest) =>
  request.put<GmConfigDraft>(`${base(data.serverId)}/configs/${seg(data.domain)}/draft`, data)

export const discardGmConfigDraft = (serverId: string, domain: string) =>
  request.delete<DiscardGmConfigDraftResponse>(`${base(serverId)}/configs/${seg(domain)}/draft`)

export const validateGmConfig = (serverId: string, domain: string, content: string) =>
  request.post<ValidateGmConfigResponse>(`${base(serverId)}/configs/${seg(domain)}/validate`, {
    serverId,
    domain,
    content,
  })

export const publishGmConfig = (data: PublishGmConfigRequest) =>
  request.post<PublishGmConfigResponse>(`${base(data.serverId)}/configs/${seg(data.domain)}/publish`, data)

export const listGmConfigRevisions = (serverId: string, domain: string, page = 1, pageSize = 20) =>
  request.get<ListGmConfigRevisionsResponse>(`${base(serverId)}/configs/${seg(domain)}/revisions`, {
    params: { page, pageSize },
  })

export const getGmConfigRevision = (serverId: string, domain: string, id: string) =>
  request.get<GmConfigRevision>(`${base(serverId)}/configs/${seg(domain)}/revisions/${seg(id)}`)

/** GM 实时动态（SSE）地址：用 fetch 带 Authorization 头读取（EventSource 不能带头）。 */
export const gmEventsPath = (serverId: string) => `/game/gm-events/${seg(serverId)}`
