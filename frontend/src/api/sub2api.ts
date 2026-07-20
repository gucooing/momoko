import request from '@/utils/request'
import type {
  PublicSub2APIHomeRequest,
  PublicSub2APIHomeResponse,
  GetSub2APIConfigRequest,
  GetSub2APIConfigResponse,
  UpdateSub2APIConfigRequest,
  UpdateSub2APIConfigResponse,
  TestSub2APIConnectionRequest,
  TestSub2APIConnectionResponse,
  SyncSub2APIUsageRequest,
  SyncSub2APIUsageResponse,
  GetSub2APISnapshotRequest,
  GetSub2APISnapshotResponse,
  GetSub2APIStatsRequest,
  GetSub2APIStatsResponse,
  GetSub2APIAdminTotalsRequest,
  GetSub2APIAdminTotalsResponse,
  GetSub2APIAdminTrendRequest,
  GetSub2APIAdminTrendResponse,
  GetSub2APIAdminTopRequest,
  GetSub2APIAdminTopResponse,
  GetSub2APIRecentRequestsRequest,
  GetSub2APIRecentRequestsResponse,
  ListSub2APIAnnouncementsResponse,
  CreateSub2APIAnnouncementRequest,
  UpdateSub2APIAnnouncementRequest,
  Sub2APIAnnouncementResponse,
  DeleteSub2APIAnnouncementResponse,
  ListSub2APITimelineResponse,
  CreateSub2APITimelineItemRequest,
  UpdateSub2APITimelineItemRequest,
  Sub2APITimelineItemResponse,
  DeleteSub2APITimelineItemResponse,
} from '@/types/v1/sub2api'

// 公开首页（无需鉴权）
export const getPublicSub2APIHome = (params: PublicSub2APIHomeRequest = {}) => {
  return request.get<PublicSub2APIHomeResponse>('/public/sub2api/home', { params })
}

// 配置
export const getSub2APIConfig = (params: GetSub2APIConfigRequest = {}) => {
  return request.get<GetSub2APIConfigResponse>('/sub2api/config', { params })
}

export const updateSub2APIConfig = (data: UpdateSub2APIConfigRequest) => {
  return request.put<UpdateSub2APIConfigResponse>('/sub2api/config', data)
}

export const testSub2APIConnection = (data: TestSub2APIConnectionRequest) => {
  return request.post<TestSub2APIConnectionResponse>('/sub2api/config/test', data)
}

// 用量
export const syncSub2APIUsage = (data: SyncSub2APIUsageRequest) => {
  return request.post<SyncSub2APIUsageResponse>('/sub2api/sync', data)
}

export const getSub2APISnapshot = (params: GetSub2APISnapshotRequest = {}) => {
  return request.get<GetSub2APISnapshotResponse>('/sub2api/snapshot', { params })
}

export const getPublicSub2APIStats = (params: GetSub2APIStatsRequest) => {
  return request.get<GetSub2APIStatsResponse>('/public/sub2api/stats', { params })
}

// 管理端用量汇总（标量指标 + 区间标签，需鉴权）
export const getSub2APIAdminTotals = (params: GetSub2APIAdminTotalsRequest) => {
  return request.get<GetSub2APIAdminTotalsResponse>('/sub2api/stats/totals', { params })
}

// 管理端用量趋势（按区间日内分桶/按天，需鉴权）
export const getSub2APIAdminTrend = (params: GetSub2APIAdminTrendRequest) => {
  return request.get<GetSub2APIAdminTrendResponse>('/sub2api/stats/trend', { params })
}

// 管理端用量排行（模型 + 分组，按 token 降序，需鉴权）
export const getSub2APIAdminTop = (params: GetSub2APIAdminTopRequest) => {
  return request.get<GetSub2APIAdminTopResponse>('/sub2api/stats/top', { params })
}

// 管理端最近请求（按时间区间分页，需鉴权）
export const getSub2APIRecentRequests = (params: GetSub2APIRecentRequestsRequest) => {
  return request.get<GetSub2APIRecentRequestsResponse>('/sub2api/recent-requests', { params })
}

// 公告
export const listSub2APIAnnouncements = () => {
  return request.get<ListSub2APIAnnouncementsResponse>('/sub2api/announcements')
}

export const createSub2APIAnnouncement = (data: CreateSub2APIAnnouncementRequest) => {
  return request.post<Sub2APIAnnouncementResponse>('/sub2api/announcements', data)
}

export const updateSub2APIAnnouncement = (data: UpdateSub2APIAnnouncementRequest) => {
  return request.put<Sub2APIAnnouncementResponse>(`/sub2api/announcements/${data.id}`, data)
}

export const deleteSub2APIAnnouncement = (id: string) => {
  return request.delete<DeleteSub2APIAnnouncementResponse>(`/sub2api/announcements/${id}`)
}

// 时间线
export const listSub2APITimeline = () => {
  return request.get<ListSub2APITimelineResponse>('/sub2api/timeline')
}

export const createSub2APITimelineItem = (data: CreateSub2APITimelineItemRequest) => {
  return request.post<Sub2APITimelineItemResponse>('/sub2api/timeline', data)
}

export const updateSub2APITimelineItem = (data: UpdateSub2APITimelineItemRequest) => {
  return request.put<Sub2APITimelineItemResponse>(`/sub2api/timeline/${data.id}`, data)
}

export const deleteSub2APITimelineItem = (id: string) => {
  return request.delete<DeleteSub2APITimelineItemResponse>(`/sub2api/timeline/${id}`)
}
