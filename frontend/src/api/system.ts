import request from '@/utils/request'
import type {
  EmailConfigRequest,
  EmailConfigResponse,
  ListOperationLogsRequest,
  ListOperationLogsResponse,
  SystemOverviewResponse,
  SystemStatusRequest,
  SystemStatusResponse,
  TestEmailConfigRequest,
  TestEmailConfigResponse,
  UpdateEmailConfigRequest,
  UpdateEmailConfigResponse,
} from '@/types/v1/system'

export const getSystemOverview = () => {
  return request.get<SystemOverviewResponse>('/system/overview')
}

export const getSystemStatus = (params?: SystemStatusRequest) => {
  return request.get<SystemStatusResponse>('/system/status', { params })
}

export const listOperationLogs = (params: ListOperationLogsRequest) => {
  return request.get<ListOperationLogsResponse>('/system/operation-logs', { params })
}

export const getEmailConfig = (params: EmailConfigRequest = {}) => {
  return request.get<EmailConfigResponse>('/system/email-config', { params })
}

export const updateEmailConfig = (params: UpdateEmailConfigRequest) => {
  return request.put<UpdateEmailConfigResponse>('/system/email-config', params)
}

export const testEmailConfig = (params: TestEmailConfigRequest) => {
  return request.post<TestEmailConfigResponse>('/system/email-config/test', params)
}
