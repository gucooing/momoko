// Agent（Code 与 Agent 管理）接口。会话里的对话事件走 WebSocket，见 views/agent/code/useAgentSocket.ts。
import request from '@/utils/request'
import type {
  AgentAttachment,
  AgentMcpServer,
  AgentNotifyEvents,
  AgentProvider,
  AgentRunSettings,
  ArchiveAgentSessionsResponse,
  BrowseAgentWorkspaceResponse,
  CreateAgentMcpServerResponse,
  CreateAgentProviderResponse,
  CreateAgentRuntimeResponse,
  CreateAgentSessionRequest,
  CreateAgentSessionResponse,
  CreateAgentSkillRequest,
  CreateAgentSkillResponse,
  CreateAgentWorkspaceRequest,
  CreateAgentWorkspaceResponse,
  DeleteAgentSessionsResponse,
  DetectAgentRuntimeResponse,
  ExportAgentSessionsResponse,
  FetchAgentProviderModelsRequest,
  FetchAgentProviderModelsResponse,
  ForkAgentSessionResponse,
  GetAgentDocsResponse,
  GetAgentNotifyResponse,
  GetAgentOptionsResponse,
  GetAgentOverviewResponse,
  GetAgentSessionChangesResponse,
  GetAgentSessionProcessResponse,
  GetAgentSessionResponse,
  GetAgentSidebarResponse,
  GetAgentSkillResponse,
  ImportAgentMcpServersResponse,
  ImportAgentSkillRequest,
  ImportAgentSkillResponse,
  ListAgentDirsResponse,
  ListAgentMcpServersResponse,
  ListAgentProvidersResponse,
  ListAgentRuntimesResponse,
  ListAgentSessionFilesResponse,
  ListAgentSessionsResponse,
  ListAgentSkillsResponse,
  ListAgentWorkspacesResponse,
  PromptAgentSessionRequest,
  PublicAgentApprovalResponse,
  PurgeAgentSessionsResponse,
  ReadAgentSessionFileResponse,
  GetAgentSessionUploadResponse,
  SaveAgentNotifyChannelResponse,
  SearchAgentFilesResponse,
  SetAgentRuntimeModeResponse,
  SetAgentSessionOptionResponse,
  TestAgentMcpServerResponse,
  TestAgentProviderResponse,
  UpdateAgentMcpServerResponse,
  UpdateAgentProviderResponse,
  UpdateAgentRunSettingsResponse,
  UpdateAgentRuntimeResponse,
  UpdateAgentSkillRequest,
  UpdateAgentSkillResponse,
  UpdateAgentWorkspaceRequest,
  UpdateAgentWorkspaceResponse,
} from '@/types/v1/agent'

/** 会话事件的 WebSocket 地址（相对路径，交给 buildBackendWebSocketUrl）。 */
export const AGENT_WS_PATH = '/api/v1/agent/ws'

// 一轮可能要先启动 Agent 进程、恢复上下文，放宽超时。
const SLOW = { timeout: 120000 }

/** 随消息上传的附件：图片和文件的内容是 base64（protobuf bytes 在 JSON 里的写法） */
export type AgentAttachmentInput = Omit<Partial<AgentAttachment>, 'data'> & { data?: string }

// —— 选择器 / 概览 ——
export const getAgentOptions = () => request.get<GetAgentOptionsResponse>('/agent/options')
export const getAgentOverview = () => request.get<GetAgentOverviewResponse>('/agent/overview')

// —— 会话 ——
export const getAgentSidebar = (keyword?: string) =>
  request.get<GetAgentSidebarResponse>('/agent/sidebar', { params: { keyword } })
export const listAgentSessions = (params: { status?: string; keyword?: string; page?: number; pageSize?: number }) =>
  request.get<ListAgentSessionsResponse>('/agent/sessions', { params })
export const createAgentSession = (
  data: Omit<Partial<CreateAgentSessionRequest>, 'attachments'> & { attachments?: AgentAttachmentInput[] },
) =>
  request.post<CreateAgentSessionResponse>('/agent/sessions', data, SLOW)
export const getAgentSession = (id: string) => request.get<GetAgentSessionResponse>(`/agent/sessions/${id}`)
/** 会话的 Agent 进程（命令、进程号、内存、已运行）：内存要遍历进程树，展开看时才读 */
export const getAgentSessionProcess = (id: string) => request.get<GetAgentSessionProcessResponse>(`/agent/sessions/${id}/process`)
export const promptAgentSession = (
  id: string,
  data: Omit<Partial<PromptAgentSessionRequest>, 'attachments'> & { attachments?: AgentAttachmentInput[] },
) =>
  request.post(`/agent/sessions/${id}/prompt`, data, SLOW)
export const cancelAgentSession = (id: string) => request.post(`/agent/sessions/${id}/cancel`, {})
export const resolveAgentPermission = (id: string, requestId: string, optionId: string) =>
  request.post(`/agent/sessions/${id}/permission`, { requestId, optionId })
export const setAgentSessionOption = (
  id: string,
  data: { kind: 'mode' | 'model' | 'effort'; value: string; providerId?: string },
) => request.post<SetAgentSessionOptionResponse>(`/agent/sessions/${id}/option`, data, SLOW)
export const renameAgentSession = (id: string, title: string) =>
  request.post(`/agent/sessions/${id}/rename`, { title })
export const forkAgentSession = (id: string, worktree: boolean) =>
  request.post<ForkAgentSessionResponse>(`/agent/sessions/${id}/fork`, { worktree }, SLOW)
export const detachAgentSession = (id: string) => request.post(`/agent/sessions/${id}/detach`, {})
export const archiveAgentSessions = (ids: string[], archived: boolean) =>
  request.post<ArchiveAgentSessionsResponse>('/agent/sessions/archive', { ids, archived })
export const deleteAgentSessions = (ids: string[], withForks: boolean) =>
  request.post<DeleteAgentSessionsResponse>('/agent/sessions/delete', { ids, withForks })
export const purgeAgentSessions = () => request.post<PurgeAgentSessionsResponse>('/agent/sessions/purge', {})
export const exportAgentSessions = (ids: string[]) =>
  request.post<ExportAgentSessionsResponse>('/agent/sessions/export', { ids })
export const getAgentSessionChanges = (id: string) =>
  request.get<GetAgentSessionChangesResponse>(`/agent/sessions/${id}/changes`)
export const revertAgentSessionFile = (id: string, path: string) =>
  request.post(`/agent/sessions/${id}/revert`, { path })
export const listAgentSessionFiles = (id: string, dir: string) =>
  request.get<ListAgentSessionFilesResponse>(`/agent/sessions/${id}/files`, { params: { dir } })
export const readAgentSessionFile = (id: string, path: string) =>
  request.get<ReadAgentSessionFileResponse>(`/agent/sessions/${id}/file`, { params: { path } })
/** 消息里上传过的附件；data 在 JSON 里是 base64 字符串。 */
export const getAgentSessionUpload = (id: string, file: string) =>
  request.get<Omit<GetAgentSessionUploadResponse, 'data'> & { data: string }>(`/agent/sessions/${id}/upload`, { params: { file } })
export const searchAgentFiles = (params: { sessionId?: string; workspaceId?: string; keyword?: string }) =>
  request.get<SearchAgentFilesResponse>('/agent/files/search', { params })

// —— 工作区 ——
export const listAgentWorkspaces = () => request.get<ListAgentWorkspacesResponse>('/agent/workspaces')
export const createAgentWorkspace = (data: Partial<CreateAgentWorkspaceRequest>) =>
  request.post<CreateAgentWorkspaceResponse>('/agent/workspaces', data)
export const updateAgentWorkspace = (id: string, data: Partial<UpdateAgentWorkspaceRequest>) =>
  request.put<UpdateAgentWorkspaceResponse>(`/agent/workspaces/${id}`, data)
export const deleteAgentWorkspace = (id: string) => request.delete(`/agent/workspaces/${id}`)
export const listAgentDirs = (path: string) => request.get<ListAgentDirsResponse>('/agent/dirs', { params: { path } })
export const browseAgentWorkspace = (id: string, dir: string) =>
  request.get<BrowseAgentWorkspaceResponse>(`/agent/workspaces/${id}/browse`, { params: { dir } })
export const getAgentDocs = (path: string) => request.get<GetAgentDocsResponse>('/agent/docs', { params: { path } })

// —— 模型供应商 ——
export const listAgentProviders = () => request.get<ListAgentProvidersResponse>('/agent/providers')
export const createAgentProvider = (provider: Partial<AgentProvider>) =>
  request.post<CreateAgentProviderResponse>('/agent/providers', { provider })
export const updateAgentProvider = (id: string, provider: Partial<AgentProvider>) =>
  request.put<UpdateAgentProviderResponse>(`/agent/providers/${id}`, { provider })
export const deleteAgentProvider = (id: string) => request.delete(`/agent/providers/${id}`)
export const testAgentProvider = (data: { id?: string; draft?: Partial<AgentProvider> }) =>
  request.post<TestAgentProviderResponse>('/agent/providers/test', data, SLOW)
export const fetchAgentProviderModels = (data: Partial<FetchAgentProviderModelsRequest>) =>
  request.post<FetchAgentProviderModelsResponse>('/agent/providers/models', data, SLOW)

// —— MCP ——
export const listAgentMcpServers = () => request.get<ListAgentMcpServersResponse>('/agent/mcp')
export const createAgentMcpServer = (server: Partial<AgentMcpServer>) =>
  request.post<CreateAgentMcpServerResponse>('/agent/mcp', { server })
export const updateAgentMcpServer = (id: string, server: Partial<AgentMcpServer>) =>
  request.put<UpdateAgentMcpServerResponse>(`/agent/mcp/${id}`, { server })
export const deleteAgentMcpServer = (id: string) => request.delete(`/agent/mcp/${id}`)
export const setAgentMcpDefault = (id: string, defaultOn: boolean) =>
  request.post(`/agent/mcp/${id}/default`, { defaultOn })
export const testAgentMcpServer = (data: { id?: string; draft?: Partial<AgentMcpServer> }) =>
  request.post<TestAgentMcpServerResponse>('/agent/mcp/test', data, SLOW)
export const importAgentMcpServers = (json: string) =>
  request.post<ImportAgentMcpServersResponse>('/agent/mcp/import', { json })

// —— 技能 ——
export const listAgentSkills = (params?: { enabled?: boolean }) => request.get<ListAgentSkillsResponse>('/agent/skills', { params })
export const getAgentSkill = (id: string) => request.get<GetAgentSkillResponse>(`/agent/skills/${id}`)
export const createAgentSkill = (data: Partial<CreateAgentSkillRequest>) =>
  request.post<CreateAgentSkillResponse>('/agent/skills', data)
export const updateAgentSkill = (id: string, data: Partial<UpdateAgentSkillRequest>) =>
  request.put<UpdateAgentSkillResponse>(`/agent/skills/${id}`, data)
export const deleteAgentSkill = (id: string) => request.delete(`/agent/skills/${id}`)
export const setAgentSkillEnabled = (id: string, enabled: boolean) =>
  request.post(`/agent/skills/${id}/enabled`, { enabled })
/** data 是 zip 的 base64（protobuf bytes 在 JSON 里的写法）。 */
export const importAgentSkill = (data: Omit<Partial<ImportAgentSkillRequest>, 'data'> & { data?: string }) =>
  request.post<ImportAgentSkillResponse>('/agent/skills/import', data, SLOW)

// —— 运行时 ——
export const listAgentRuntimes = () => request.get<ListAgentRuntimesResponse>('/agent/runtimes')
export const setAgentRuntimeMode = (id: string, managed: boolean) =>
  request.post<SetAgentRuntimeModeResponse>(`/agent/runtimes/${id}/mode`, { managed }, { timeout: 600000 })
export const detectAgentRuntime = (id: string) =>
  request.post<DetectAgentRuntimeResponse>(`/agent/runtimes/${id}/detect`, {}, SLOW)
export const updateAgentRuntime = (id: string) =>
  request.post<UpdateAgentRuntimeResponse>(`/agent/runtimes/${id}/update`, {}, { timeout: 600000 })
export const createAgentRuntime = (name: string, command: string) =>
  request.post<CreateAgentRuntimeResponse>('/agent/runtimes', { name, command }, SLOW)
export const deleteAgentRuntime = (id: string) => request.delete(`/agent/runtimes/${id}`)
export const updateAgentRunSettings = (settings: AgentRunSettings) =>
  request.post<UpdateAgentRunSettingsResponse>('/agent/runtimes/settings', { settings })

// —— 通知 ——
export const getAgentNotify = () => request.get<GetAgentNotifyResponse>('/agent/notify')
export const updateAgentNotifyEvents = (events: AgentNotifyEvents) =>
  request.post('/agent/notify/events', { events })
export const saveAgentNotifyChannel = (kind: string, fields: Record<string, string>) =>
  request.post<SaveAgentNotifyChannelResponse>('/agent/notify/channel', { kind, fields })
export const setAgentNotifyChannelEnabled = (kind: string, enabled: boolean) =>
  request.post('/agent/notify/channel/enabled', { kind, enabled })
export const testAgentNotifyChannel = (kind: string, fields?: Record<string, string>) =>
  request.post('/agent/notify/channel/test', { kind, fields: fields || {} }, SLOW)

// —— 公开批准页（通知消息里的链接，不需要登录）——
export const getPublicAgentApproval = (token: string) =>
  request.get<PublicAgentApprovalResponse>(`/public/agent/approval/${token}`)
export const resolvePublicAgentApproval = (token: string, optionId: string) =>
  request.post(`/public/agent/approval/${token}`, { optionId })
