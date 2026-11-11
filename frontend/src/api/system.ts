import request from '@/utils/request'
import type {
  EmailConfigRequest,
  EmailConfigResponse,
  EmailTemplateRequest,
  EmailTemplateResponse,
  ListOperationLogsRequest,
  ListOperationLogsResponse,
  ListEmailTemplatesResponse,
  SiteConfigResponse,
  StorageConfigResponse,
  UpdateStorageConfigRequest,
  UpdateStorageConfigResponse,
  UpdateSiteConfigRequest,
  UpdateSiteConfigResponse,
  OperationLogStatsResponse,
  UpdateOperationLogRetentionRequest,
  UpdateOperationLogRetentionResponse,
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

export const getStorageConfig = () => {
  return request.get<StorageConfigResponse>('/system/storage-config')
}

export const updateStorageConfig = (params: UpdateStorageConfigRequest) => {
  return request.put<UpdateStorageConfigResponse>('/system/storage-config', params)
}

export const getEmailTemplate = (params: EmailTemplateRequest) => {
  return request.get<EmailTemplateResponse>('/system/email-template', { params })
}

export const updateEmailTemplate = (params: UpdateEmailTemplateRequest) => {
  return request.put<UpdateEmailTemplateResponse>('/system/email-template', params)
}

export const listEmailTemplates = () => {
  return request.get<ListEmailTemplatesResponse>('/system/email-templates')
}

export const getOperationLogStats = () => {
  return request.get<OperationLogStatsResponse>('/system/operation-logs/stats')
}

// 操作记录保留策略（保留天数，0=永久保留；保存后立即清理一次超期记录）
export const updateOperationLogRetention = (params: UpdateOperationLogRetentionRequest) => {
  return request.put<UpdateOperationLogRetentionResponse>('/system/operation-logs/retention', params)
}

// 站点常规配置（站点名称 / 对外地址 / 默认语言 / 版本更新偏好 + 最近检查与最近修改）
export const getSiteConfig = () => {
  return request.get<SiteConfigResponse>('/system/site-config')
}

export const updateSiteConfig = (params: UpdateSiteConfigRequest) => {
  return request.put<UpdateSiteConfigResponse>('/system/site-config', params)
}
