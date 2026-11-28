// Agent（Code 与 Agent 设置）接口。会话里的对话事件走 WebSocket，见 views/agent/code/useAgentSocket.ts。
import request from '@/utils/request'
import type {
  AgentAttachment,
  AgentMcpServer,
  AgentRuntimeModelSetting,
  AgentProxy,
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
  GetAgentPreviewTokenResponse,
  GetAgentProviderBalanceResponse,
  ListAgentNotifyResponse,
  CommitAgentGitResponse,
  GenerateAgentGitMessageResponse,
  GetAgentGitCommitResponse,
  GetAgentGitDiffResponse,
  GetAgentGitStatusResponse,
  ListAgentGitLogResponse,
  ListAgentGitRefsResponse,
  ListAgentGitStashesResponse,
  RunAgentGitOperationResponse,
  SyncAgentGitResponse,
  UpdateAgentGitRefResponse,
  UpdateAgentGitStashResponse,
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
  ListAgentUsersResponse,
  ListAgentWorkspacesResponse,
  PromptAgentSessionRequest,
  PublicAgentApprovalResponse,
  PurgeAgentSessionsResponse,
  ReadAgentSessionFileResponse,
  GetAgentRuntimeModelLevelsResponse,
  SaveAgentFileResponse,
  SaveAgentNotifyChannelResponse,
  PreSignAgentUploadResponse,
  SearchAgentFilesResponse,
  SignAgentFilesResponse,
  SetAgentRuntimeModeResponse,
  SetAgentRuntimeModelsResponse,
  SetAgentRuntimeProxyResponse,
  TestAgentProxyRequest,
  TestAgentProxyResponse,
  UpdateAgentSandboxResponse,
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
/** 官方账户的登录终端：在服务器上跑登录命令（?runtime=运行时 id），权限 agent:manage */
export const AGENT_LOGIN_WS_PATH = '/api/v1/agent/login/ws'

// 一轮可能要先启动 Agent 进程、恢复上下文，放宽超时。
const SLOW = { timeout: 120000 }

/** 随消息发的附件：图片和文件是传好的 uploadId（preSignAgentUpload），粘贴和选中的文字在 text 里 */
export type AgentAttachmentInput = Partial<AgentAttachment>

// —— 选择器 / 概览 ——
export const getAgentOptions = () => request.get<GetAgentOptionsResponse>('/agent/options')
/** all：Agent 管理页，会话、工作区、MCP、技能数所有人的 */
export const getAgentOverview = (all = false) =>
  request.get<GetAgentOverviewResponse>('/agent/overview', { params: all ? { all } : undefined })

// —— 会话 ——
export const getAgentSidebar = (keyword?: string) =>
  request.get<GetAgentSidebarResponse>('/agent/sidebar', { params: { keyword } })
export const listAgentSessions = (params: {
  status?: string
  keyword?: string
  page?: number
  pageSize?: number
  /** 所有人的（Agent 管理页） */
  all?: boolean
}) =>
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
/** 只停一项（ACP v2）：toolId 不为空停这个工具调用（sub 是它所在的子 Agent，主会话为空），否则停 sub 这个子 Agent */
export const stopAgentSessionItem = (id: string, sub: string, toolId: string) =>
  request.post(`/agent/sessions/${id}/stop-item`, { sub, toolId })
/** 撤回一条排着队、还没发出去的消息 */
export const deleteAgentQueuedPrompt = (id: string, queueId: string) =>
  request.delete(`/agent/sessions/${id}/queue/${queueId}`)
/** 排着队的一条马上发：能插进正在跑的一轮就插进去，插不进去的停下这一轮，让它作为下一轮发出 */
export const sendAgentQueuedPrompt = (id: string, queueId: string) =>
  request.post(`/agent/sessions/${id}/queue/${queueId}/send`, {})
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
// —— Git（右侧「Git」标签）：repo 是仓库相对会话目录的路径（会话目录本身为空），文件路径都相对仓库。
//    会改东西、要联网的操作（拉取推送、提交钩子、让 Agent 写说明）可能很久，都放宽超时 ——
const gitUrl = (id: string, path: string) => `/agent/sessions/${id}/git/${path}`
export const getAgentGitStatus = (id: string) => request.get<GetAgentGitStatusResponse>(gitUrl(id, 'status'), SLOW)
export type AgentGitDiffSource = 'unstaged' | 'staged' | 'conflict' | 'commit' | 'stash' | 'head'
export const getAgentGitDiff = (
  id: string,
  params: { repo: string; source: AgentGitDiffSource; path?: string; origPath?: string; ref?: string },
) => request.get<GetAgentGitDiffResponse>(gitUrl(id, 'diff'), { params, ...SLOW })
export type AgentGitFileAction = 'stage' | 'unstage' | 'discard' | 'ours' | 'theirs' | 'resolve' | 'ignore'
export const updateAgentGitFiles = (id: string, data: { repo: string; action: AgentGitFileAction; paths?: string[]; all?: boolean }) =>
  request.post(gitUrl(id, 'files'), data, SLOW)
export type AgentGitLineAction = 'stage' | 'unstage' | 'discard'
export const applyAgentGitLines = (
  id: string,
  data: { repo: string; path: string; action: AgentGitLineAction; patch: string; lines: number[] },
) => request.post(gitUrl(id, 'lines'), data, SLOW)
export const commitAgentGit = (
  id: string,
  data: { repo: string; message: string; amend?: boolean; all?: boolean; noVerify?: boolean; signoff?: boolean },
) => request.post<CommitAgentGitResponse>(gitUrl(id, 'commit'), data, SLOW)
export const generateAgentGitMessage = (id: string, repo: string) =>
  request.post<GenerateAgentGitMessageResponse>(gitUrl(id, 'message'), { repo }, SLOW)
export const resetAgentGit = (id: string, data: { repo: string; target: string; mode: 'soft' | 'mixed' | 'hard' }) =>
  request.post(gitUrl(id, 'reset'), data, SLOW)
export const listAgentGitRefs = (id: string, repo: string) =>
  request.get<ListAgentGitRefsResponse>(gitUrl(id, 'refs'), { params: { repo }, ...SLOW })
export type AgentGitRefAction =
  | 'checkout'
  | 'create-branch'
  | 'rename-branch'
  | 'delete-branch'
  | 'delete-remote-branch'
  | 'merge'
  | 'rebase'
  | 'create-tag'
  | 'delete-tag'
  | 'push-tag'
  | 'delete-remote-tag'
export const updateAgentGitRef = (
  id: string,
  data: {
    repo: string
    action: AgentGitRefAction
    name: string
    newName?: string
    from?: string
    checkout?: boolean
    force?: boolean
    message?: string
    remote?: string
  },
) => request.post<UpdateAgentGitRefResponse>(gitUrl(id, 'ref'), data, SLOW)
export const syncAgentGit = (
  id: string,
  data: { repo: string; action: 'fetch' | 'pull' | 'push'; remote?: string; mode?: '' | 'merge' | 'rebase' | 'ff-only'; force?: boolean; tags?: boolean },
) => request.post<SyncAgentGitResponse>(gitUrl(id, 'sync'), data, SLOW)
export const updateAgentGitRemote = (
  id: string,
  data: { repo: string; action: 'add' | 'set-url' | 'rename' | 'remove'; name: string; url?: string; newName?: string },
) => request.post(gitUrl(id, 'remote'), data, SLOW)
export const listAgentGitStashes = (id: string, repo: string) =>
  request.get<ListAgentGitStashesResponse>(gitUrl(id, 'stashes'), { params: { repo }, ...SLOW })
export const updateAgentGitStash = (
  id: string,
  data: {
    repo: string
    action: 'push' | 'apply' | 'pop' | 'drop'
    hash?: string
    message?: string
    untracked?: boolean
    staged?: boolean
    index?: boolean
    paths?: string[]
  },
) => request.post<UpdateAgentGitStashResponse>(gitUrl(id, 'stash'), data, SLOW)
export const runAgentGitOperation = (
  id: string,
  data: { repo: string; action: 'cherry-pick' | 'revert' | 'continue' | 'skip' | 'abort'; commit?: string },
) => request.post<RunAgentGitOperationResponse>(gitUrl(id, 'operation'), data, SLOW)
export const listAgentGitLog = (
  id: string,
  params: { repo: string; skip?: number; limit?: number; ref?: string; all?: boolean; path?: string; keyword?: string },
) => request.get<ListAgentGitLogResponse>(gitUrl(id, 'log'), { params, ...SLOW })
export const getAgentGitCommit = (id: string, repo: string, hash: string, stash = false) =>
  request.get<GetAgentGitCommitResponse>(gitUrl(id, `commits/${hash}`), { params: { repo, stash }, ...SLOW })
export const initAgentGitRepo = (id: string, path = '') => request.post(gitUrl(id, 'init'), { path }, SLOW)
export const listAgentSessionFiles = (id: string, dir: string) =>
  request.get<ListAgentSessionFilesResponse>(`/agent/sessions/${id}/files`, { params: { dir } })
export const readAgentSessionFile = (id: string, path: string) =>
  request.get<ReadAgentSessionFileResponse>(`/agent/sessions/${id}/file`, { params: { path } })
export const searchAgentFiles = (params: { sessionId?: string; workspaceId?: string; keyword?: string }) =>
  request.get<SearchAgentFilesResponse>('/agent/files/search', { params })

// —— 文件树里的编辑：有会话给 sessionId（会话目录），新会话页还没有会话时给 workspaceId（工作区目录） ——
export interface AgentFileScope {
  sessionId?: string
  workspaceId?: string
}
/** 修改时间是 int64（纳秒），JSON 里是字符串：原样带回去，不要转成数字 */
export type AgentModTime = string | number
export const saveAgentFile = (
  scope: AgentFileScope,
  data: { path: string; content: string; create?: boolean; baseModTime?: AgentModTime; force?: boolean },
) => request.post<SaveAgentFileResponse>('/agent/files/save', { ...scope, ...data })
export const createAgentDir = (scope: AgentFileScope, path: string) => request.post('/agent/files/mkdir', { ...scope, path })
export const moveAgentFile = (scope: AgentFileScope, from: string, to: string) =>
  request.post('/agent/files/move', { ...scope, from, to })
export const deleteAgentFiles = (scope: AgentFileScope, paths: string[]) =>
  request.post('/agent/files/delete', { ...scope, paths })
/** 文件的签名下载链接（标准的 /api/v1/download/pre，原样字节）：paths 相对工作目录或者是绝对路径，
 *  uploads 时相对会话的附件目录（消息附件、工具调用产出的图片）。和 paths 一一对应，取不到的为空。 */
export const signAgentFiles = (scope: AgentFileScope, paths: string[], uploads = false) =>
  request.get<SignAgentFilesResponse>('/agent/files/sign', { params: { ...scope, paths, uploads: uploads || undefined } })
/** 上传一个文件（消息附件、技能压缩包）：标准的分片上传，传完按 uploadId 用 */
export const preSignAgentUpload = (data: { fileName: string; fileSize: number; hash: string }) =>
  request.post<PreSignAgentUploadResponse>('/agent/uploads/pre-sign', data)
/** HTML 预览的签名：iframe 按 /api/v1/public/agent/preview/{节点}/{签名}/{相对路径} 把目录当静态站点加载 */
export const getAgentPreviewToken = (scope: AgentFileScope) =>
  request.get<GetAgentPreviewTokenResponse>('/agent/files/preview-token', { params: scope })
export const listAgentFiles = (scope: AgentFileScope, dir: string) =>
  scope.sessionId ? listAgentSessionFiles(scope.sessionId, dir) : browseAgentWorkspace(scope.workspaceId || '', dir)
export const readAgentFile = (scope: AgentFileScope, path: string) =>
  scope.sessionId ? readAgentSessionFile(scope.sessionId, path) : readAgentWorkspaceFile(scope.workspaceId || '', path)

// —— 工作区 ——
/** all：所有人的（Agent 管理页） */
export const listAgentWorkspaces = (all = false) =>
  request.get<ListAgentWorkspacesResponse>('/agent/workspaces', { params: all ? { all } : undefined })
export const createAgentWorkspace = (data: Partial<CreateAgentWorkspaceRequest>) =>
  request.post<CreateAgentWorkspaceResponse>('/agent/workspaces', data)
export const updateAgentWorkspace = (id: string, data: Partial<UpdateAgentWorkspaceRequest>) =>
  request.put<UpdateAgentWorkspaceResponse>(`/agent/workspaces/${id}`, data)
export const deleteAgentWorkspace = (id: string) => request.delete(`/agent/workspaces/${id}`)
export const listAgentDirs = (path: string) => request.get<ListAgentDirsResponse>('/agent/dirs', { params: { path } })
export const browseAgentWorkspace = (id: string, dir: string) =>
  request.get<BrowseAgentWorkspaceResponse>(`/agent/workspaces/${id}/browse`, { params: { dir } })
/** 新会话页里只读地打开工作区的文件 */
export const readAgentWorkspaceFile = (id: string, path: string) =>
  request.get<ReadAgentSessionFileResponse>(`/agent/workspaces/${id}/file`, { params: { path } })
export const getAgentDocs = (path: string) => request.get<GetAgentDocsResponse>('/agent/docs', { params: { path } })

// —— 模型供应商（第三方接口）——
/** all：所有人的（Agent 管理页）；默认是自己的，加上共享给自己的 */
export const listAgentProviders = (all = false) =>
  request.get<ListAgentProvidersResponse>('/agent/providers', { params: all ? { all } : undefined })
export const createAgentProvider = (provider: Partial<AgentProvider>) =>
  request.post<CreateAgentProviderResponse>('/agent/providers', { provider })
export const updateAgentProvider = (id: string, provider: Partial<AgentProvider>) =>
  request.put<UpdateAgentProviderResponse>(`/agent/providers/${id}`, { provider })
export const deleteAgentProvider = (id: string) => request.delete(`/agent/providers/${id}`)
export const testAgentProvider = (data: { id?: string; draft?: Partial<AgentProvider>; message?: string }) =>
  request.post<TestAgentProviderResponse>('/agent/providers/test', data, SLOW)
export const fetchAgentProviderModels = (data: Partial<FetchAgentProviderModelsRequest>) =>
  request.post<FetchAgentProviderModelsResponse>('/agent/providers/models', data, SLOW)
export const getAgentProviderBalance = (id: string) =>
  request.get<GetAgentProviderBalanceResponse>(`/agent/providers/${id}/balance`)

// —— MCP ——
/** all：所有人的（Agent 管理页） */
export const listAgentMcpServers = (all = false) =>
  request.get<ListAgentMcpServersResponse>('/agent/mcp', { params: all ? { all } : undefined })
export const createAgentMcpServer = (server: Partial<AgentMcpServer>) =>
  request.post<CreateAgentMcpServerResponse>('/agent/mcp', { server })
export const updateAgentMcpServer = (id: string, server: Partial<AgentMcpServer>) =>
  request.put<UpdateAgentMcpServerResponse>(`/agent/mcp/${id}`, { server })
export const deleteAgentMcpServer = (id: string) => request.delete(`/agent/mcp/${id}`)
export const setAgentMcpDefault = (id: string, defaultOn: boolean) =>
  request.post(`/agent/mcp/${id}/default`, { defaultOn })
export const testAgentMcpServer = (data: { id?: string; draft?: Partial<AgentMcpServer> }) =>
  request.post<TestAgentMcpServerResponse>('/agent/mcp/test', data, SLOW)
/** share：导入成共享的（Agent 管理页）；不给是导入成自己的 */
export const importAgentMcpServers = (json: string, share?: AgentShare) =>
  request.post<ImportAgentMcpServersResponse>('/agent/mcp/import', { json, ...share })

// —— 技能 ——
export const listAgentSkills = (params?: { enabled?: boolean; all?: boolean }) =>
  request.get<ListAgentSkillsResponse>('/agent/skills', { params })
export const getAgentSkill = (id: string) => request.get<GetAgentSkillResponse>(`/agent/skills/${id}`)
export const createAgentSkill = (data: Partial<CreateAgentSkillRequest>) =>
  request.post<CreateAgentSkillResponse>('/agent/skills', data)
export const updateAgentSkill = (id: string, data: Partial<UpdateAgentSkillRequest>) =>
  request.put<UpdateAgentSkillResponse>(`/agent/skills/${id}`, data)
export const deleteAgentSkill = (id: string) => request.delete(`/agent/skills/${id}`)
export const setAgentSkillEnabled = (id: string, enabled: boolean) =>
  request.post(`/agent/skills/${id}/enabled`, { enabled })
/** zip 时 uploadId 是传好的压缩包（preSignAgentUpload） */
export const importAgentSkill = (data: Partial<ImportAgentSkillRequest>) =>
  request.post<ImportAgentSkillResponse>('/agent/skills/import', data, SLOW)

/** 共享的第三方接口、MCP 服务器、技能的主人（不属于哪个人）；和后端 agentSharedOwner 一致 */
export const AGENT_SHARED_OWNER = '@shared'
/** 共享的第三方接口、MCP 服务器、技能给谁用：all 所有人；users 指定的人 */
export interface AgentShare {
  share: 'all' | 'users'
  shareUserIds: string[]
}
/** Agent 管理页给共享的第三方接口、MCP、技能选人：所有用户（要有 agent:admin） */
export const listAgentUsers = () => request.get<ListAgentUsersResponse>('/agent/users')

// —— 运行时 ——
export const listAgentRuntimes = () => request.get<ListAgentRuntimesResponse>('/agent/runtimes')
export const setAgentRuntimeMode = (id: string, managed: boolean) =>
  request.post<SetAgentRuntimeModeResponse>(`/agent/runtimes/${id}/mode`, { managed }, { timeout: 600000 })
/** 服务器上的登录：开一个会话切到这个模型，读 Agent 给它的推理等级（不保存）。要启动一次 Agent，几秒到十几秒。 */
export const getAgentRuntimeModelLevels = (id: string, model: string) =>
  request.get<GetAgentRuntimeModelLevelsResponse>(`/agent/runtimes/${id}/levels`, { params: { model }, timeout: 180_000 })
export const setAgentRuntimeModels = (id: string, models: AgentRuntimeModelSetting[]) =>
  request.post<SetAgentRuntimeModelsResponse>(`/agent/runtimes/${id}/models`, { models })
export const detectAgentRuntime = (id: string) =>
  request.post<DetectAgentRuntimeResponse>(`/agent/runtimes/${id}/detect`, {}, SLOW)
export const updateAgentRuntime = (id: string) =>
  request.post<UpdateAgentRuntimeResponse>(`/agent/runtimes/${id}/update`, {}, { timeout: 600000 })
export const createAgentRuntime = (name: string, command: string) =>
  request.post<CreateAgentRuntimeResponse>('/agent/runtimes', { name, command }, SLOW)
export const deleteAgentRuntime = (id: string) => request.delete(`/agent/runtimes/${id}`)
export const updateAgentRunSettings = (settings: AgentRunSettings) =>
  request.post<UpdateAgentRunSettingsResponse>('/agent/runtimes/settings', { settings })
/** 官方账户走的代理（只有 Linux 能配）；type 为空是不走代理 */
export const setAgentRuntimeProxy = (id: string, proxy: AgentProxy) =>
  request.post<SetAgentRuntimeProxyResponse>(`/agent/runtimes/${id}/proxy`, { proxy })
/** 试一下代理：经它连到要用的那一家（官方账户给 runtimeId，第三方接口给 providerId 或类型和接口地址） */
export const testAgentProxy = (data: Partial<TestAgentProxyRequest>) =>
  request.post<TestAgentProxyResponse>('/agent/proxy/test', data, SLOW)
/** 沙箱用的 sing-box：没下载时下载，下载过时更新到最新版本（只有 Linux） */
export const updateAgentSandbox = () =>
  request.post<UpdateAgentSandboxResponse>('/agent/sandbox/update', {}, { timeout: 600000 })

// —— 通知：每个人自己的 ——
export const getAgentNotify = () => request.get<GetAgentNotifyResponse>('/agent/notify')
export const updateAgentNotifyEvents = (events: AgentNotifyEvents) =>
  request.post('/agent/notify/events', { events })
export const saveAgentNotifyChannel = (kind: string, fields: Record<string, string>) =>
  request.post<SaveAgentNotifyChannelResponse>('/agent/notify/channel', { kind, fields })
/** userId：别人的（Agent 管理页）；不给是自己的 */
export const setAgentNotifyChannelEnabled = (kind: string, enabled: boolean, userId?: string) =>
  request.post('/agent/notify/channel/enabled', { kind, enabled, userId })
/** 移除一种通知方式；userId：别人的（Agent 管理页），不给是自己的 */
export const deleteAgentNotifyChannel = (kind: string, userId?: string) =>
  request.post('/agent/notify/channel/delete', { kind, userId })
/** 所有人的通知设置（Agent 管理页，要有 agent:admin）：只有设置过通知方式的人 */
export const listAgentNotify = () => request.get<ListAgentNotifyResponse>('/agent/notify/all')
export const testAgentNotifyChannel = (kind: string, fields?: Record<string, string>) =>
  request.post('/agent/notify/channel/test', { kind, fields: fields || {} }, SLOW)

// —— 公开批准页（通知消息里的链接，不需要登录）——
export const getPublicAgentApproval = (token: string) =>
  request.get<PublicAgentApprovalResponse>(`/public/agent/approval/${token}`)
export const resolvePublicAgentApproval = (token: string, optionId: string) =>
  request.post(`/public/agent/approval/${token}`, { optionId })
