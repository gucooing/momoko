import request from '@/utils/request'
import type {
  ClearMonitorHistoryResponse,
  ExportMonitorDataRequest,
  ExportMonitorDataResponse,
  GetMonitorDataRequest,
  GetMonitorDataResponse,
  GetMonitorSettingsResponse,
  UpdateMonitorSettingsRequest,
  UpdateMonitorSettingsResponse,
} from '@/types/v1/monitor'

// 持久性能监控（只对当前节点生效；远程节点由主控转发）

/** 查询时段内的曲线、汇总、磁盘容量趋势与应用占用排行（unix 秒） */
export const getMonitorData = (params: GetMonitorDataRequest) => {
  return request.get<GetMonitorDataResponse>('/monitor/data', { params })
}

export const getMonitorSettings = () => {
  return request.get<GetMonitorSettingsResponse>('/monitor/settings')
}

export const updateMonitorSettings = (params: UpdateMonitorSettingsRequest) => {
  return request.put<UpdateMonitorSettingsResponse>('/monitor/settings', params)
}

/** 导出所选时段的 CSV 文本 */
export const exportMonitorData = (params: ExportMonitorDataRequest) => {
  return request.get<ExportMonitorDataResponse>('/monitor/export', { params })
}

export const clearMonitorHistory = () => {
  return request.delete<ClearMonitorHistoryResponse>('/monitor/history')
}
