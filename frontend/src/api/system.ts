import request from '@/utils/request'
import type {
  ApplyUpdateResponse,
  CancelUpdateResponse,
  ClearDownloadsResponse,
  DownloadActionResponse,
  ListDownloadsResponse,
  RemoveDownloadResponse,
  StartUpdateResponse,
  UpdateStatusResponse,
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

// ==================== 在线更新 ====================

// 更新进度（silent：重启期间轮询连不上是正常的，不弹错误）
export const getUpdateStatus = (silent = false) => {
  return request.get<UpdateStatusResponse>('/system/update/status', { silent, timeout: 8000 })
}

// 从 GitHub 下载当前更新通道的最新版本，校验后替换程序并重启
export const startUpdate = () => {
  return request.post<StartUpdateResponse>('/system/update/start', {})
}

// 上传程序文件（也可以是发行版的 .zip / .tar.gz），校验通过后等确认安装
export const uploadUpdate = (file: File, onProgress?: (percent: number) => void) => {
  return request.post<UpdateStatusResponse>('/system/update/upload', file, {
    params: { name: file.name },
    headers: { 'Content-Type': 'application/octet-stream' },
    timeout: 0,
    onUploadProgress: (e) => onProgress?.(e.total ? Math.round((e.loaded / e.total) * 100) : 0),
  })
}

// 安装已上传的程序并重启
export const applyUpdate = () => {
  return request.post<ApplyUpdateResponse>('/system/update/apply', {})
}

// 取消更新：停止下载 / 丢掉已上传的文件 / 清掉失败状态
export const cancelUpdate = () => {
  return request.post<CancelUpdateResponse>('/system/update/cancel', {})
}

// ==================== 下载管理 ====================

// 服务器从外网下载的文件（系统更新、Agent 安装包、技能压缩包……），只列出有权限的
export const listDownloads = (silent = true) => {
  return request.get<ListDownloadsResponse>('/system/downloads', { silent })
}

export const pauseDownload = (id: string) => {
  return request.post<DownloadActionResponse>(`/system/downloads/${encodeURIComponent(id)}/pause`, {})
}

export const resumeDownload = (id: string) => {
  return request.post<DownloadActionResponse>(`/system/downloads/${encodeURIComponent(id)}/resume`, {})
}

export const cancelDownload = (id: string) => {
  return request.post<DownloadActionResponse>(`/system/downloads/${encodeURIComponent(id)}/cancel`, {})
}

export const removeDownload = (id: string) => {
  return request.delete<RemoveDownloadResponse>(`/system/downloads/${encodeURIComponent(id)}`)
}

export const clearDownloads = () => {
  return request.post<ClearDownloadsResponse>('/system/downloads/clear', {})
}
