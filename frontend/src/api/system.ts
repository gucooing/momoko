import request from '@/utils/request'
import type {
  SystemOverviewResponse,
  SystemStatusRequest,
  SystemStatusResponse,
} from '@/types/v1/system'

export const getSystemOverview = () => {
  return request.get<SystemOverviewResponse>('/system/overview')
}

export const getSystemStatus = (params?: SystemStatusRequest) => {
  return request.get<SystemStatusResponse>('/system/status', { params })
}
