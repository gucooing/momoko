import request from '@/utils/request'
import type {
  CreateIQCheckPromptRequest,
  CreateIQCheckPromptResponse,
  CreateIQCheckSiteRequest,
  CreateIQCheckSiteResponse,
  DeleteIQCheckPromptRequest,
  DeleteIQCheckPromptResponse,
  DeleteIQCheckSiteRequest,
  DeleteIQCheckSiteResponse,
  FetchIQCheckModelsRequest,
  FetchIQCheckModelsResponse,
  ListIQCheckPromptsRequest,
  ListIQCheckPromptsResponse,
  ListIQCheckSitesRequest,
  ListIQCheckSitesResponse,
  PublicIQCheckResultsRequest,
  PublicIQCheckResultsResponse,
  PublicIQCheckSiteResponse,
  TriggerIQCheckRequest,
  TriggerIQCheckResponse,
  UpdateIQCheckPromptRequest,
  UpdateIQCheckPromptResponse,
  UpdateIQCheckSiteRequest,
  UpdateIQCheckSiteResponse,
} from '@/types/v1/iqcheck'

// —— 提示词 ——

export const listIQCheckPrompts = (params: ListIQCheckPromptsRequest) => {
  return request.get<ListIQCheckPromptsResponse>('/iqcheck/prompts', { params })
}

export const createIQCheckPrompt = (data: CreateIQCheckPromptRequest) => {
  return request.post<CreateIQCheckPromptResponse>('/iqcheck/prompts', data)
}

export const updateIQCheckPrompt = (data: UpdateIQCheckPromptRequest) => {
  return request.put<UpdateIQCheckPromptResponse>(`/iqcheck/prompts/${data.id}`, data)
}

export const deleteIQCheckPrompt = (params: DeleteIQCheckPromptRequest) => {
  return request.delete<DeleteIQCheckPromptResponse>(`/iqcheck/prompts/${params.id}`)
}

// —— 站点 ——

export const listIQCheckSites = (params: ListIQCheckSitesRequest) => {
  return request.get<ListIQCheckSitesResponse>('/iqcheck/sites', { params })
}

export const createIQCheckSite = (data: CreateIQCheckSiteRequest) => {
  return request.post<CreateIQCheckSiteResponse>('/iqcheck/sites', data)
}

export const updateIQCheckSite = (data: UpdateIQCheckSiteRequest) => {
  return request.put<UpdateIQCheckSiteResponse>(`/iqcheck/sites/${data.id}`, data)
}

export const deleteIQCheckSite = (params: DeleteIQCheckSiteRequest) => {
  return request.delete<DeleteIQCheckSiteResponse>(`/iqcheck/sites/${params.id}`)
}

export const fetchIQCheckModels = (data: FetchIQCheckModelsRequest) => {
  return request.post<FetchIQCheckModelsResponse>('/iqcheck/models', data)
}

export const triggerIQCheck = (params: TriggerIQCheckRequest) => {
  return request.post<TriggerIQCheckResponse>(`/iqcheck/sites/${params.id}/check`, {})
}

// —— 公开页（无需登录）——

export const getPublicIQCheckSite = (slug: string) => {
  return request.get<PublicIQCheckSiteResponse>(`/public/iqcheck/${encodeURIComponent(slug)}`)
}

export const listPublicIQCheckResults = (params: PublicIQCheckResultsRequest) => {
  const { slug, ...query } = params
  return request.get<PublicIQCheckResultsResponse>(`/public/iqcheck/${encodeURIComponent(slug)}/results`, {
    params: query,
  })
}
