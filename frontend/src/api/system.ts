import request from '@/utils/request'
import type {
  EmailConfigRequest,
  EmailConfigResponse,
  EmailTemplateRequest,
  EmailTemplateResponse,
  ListOperationLogsRequest,
  ListOperationLogsResponse,
  SystemOverviewResponse,
  SystemStatusRequest,
  SystemStatusResponse,
  TestEmailConfigRequest,
  TestEmailConfigResponse,
  UpdateEmailConfigRequest,
  UpdateEmailConfigResponse,
  UpdateEmailTemplateRequest,
  UpdateEmailTemplateResponse,
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

export const getEmailTemplate = (params: EmailTemplateRequest) => {
  return request.get<EmailTemplateResponse>('/system/email-template', { params })
}

export const updateEmailTemplate = (params: UpdateEmailTemplateRequest) => {
  return request.put<UpdateEmailTemplateResponse>('/system/email-template', params)
}
