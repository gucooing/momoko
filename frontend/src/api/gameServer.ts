import request from '@/utils/request'
import type {
  CreateGameServerRequest,
  DeleteGameServerResponse,
  DetectGameServerDirRequest,
  DetectGameServerDirResponse,
  DownloadGameServerBackupResponse,
  GameServerBackupPolicy,
  GameServerBuild,
  GameServerBuildInfo,
  GameServerInfo,
  GameServerOverview,
  GameServerResource,
  GameServerSettings,
  GameServerTask,
  GameServerTaskResponse,
  ListGameAdaptersResponse,
  ListGameServerBackupsResponse,
  ListGameServerHistoryResponse,
  ListGameServerLogsResponse,
  ListGameServersRequest,
  ListGameServersResponse,
  RotateGameServerKeyResponse,
  SaveGameServerSettingsResponse,
  SwitchGameServerModeRequest,
  TailGameServerLogResponse,
  TestGameServerRequest,
  TestGameServerResponse,
  UpdateGameServerHostingRequest,
  UpdateGameServerRequest,
} from '@/types/v1/game_server'
import type {
  BatchCompressInstanceFileResponse,
  BatchDeleteInstanceFileResponse,
  CopyInstanceFileResponse,
  CreateInstanceFileResponse,
  CutInstanceFileResponse,
  EditInstanceFileResponse,
  GetInstanceFileListRequest,
  GetInstanceFileListResponse,
  GetInstanceFileTreeResponse,
  InstanceFilePreSignResponse,
  InstanceFilePreSignUploadResponse,
  RenameInstanceFileResponse,
  UnzipInstanceFileResponse,
} from '@/types/v1/instance'

// 游戏服务器管理（权限 game:server）。

const seg = (value: string) => encodeURIComponent(value)
const base = (id: string) => `/game/servers/${seg(id)}`

export const listGameAdapters = () => request.get<ListGameAdaptersResponse>('/game/adapters')

export const listGameServers = (params: Partial<ListGameServersRequest>) =>
  request.get<ListGameServersResponse>('/game/servers', { params })

export const getGameServer = (id: string) => request.get<GameServerInfo>(base(id))

export const createGameServer = (data: Partial<CreateGameServerRequest>) => request.post<GameServerInfo>('/game/servers', data)

export const updateGameServer = (data: UpdateGameServerRequest) => request.put<GameServerInfo>(base(data.id), data)

export const deleteGameServer = (id: string, deleteInstances = false) =>
  request.delete<DeleteGameServerResponse>(base(id), { params: { deleteInstances } })

export const testGameServer = (data: Partial<TestGameServerRequest>) =>
  request.post<TestGameServerResponse>('/game/servers/test', data)

export const refreshGameServer = (id: string) => request.post<GameServerInfo>(`${base(id)}/refresh`, {})

// ---- 托管 ----

export const detectGameServerDir = (data: Partial<DetectGameServerDirRequest>) =>
  request.post<DetectGameServerDirResponse>('/game/servers/detect', data)

export const switchGameServerMode = (data: Partial<SwitchGameServerModeRequest> & { id: string }) =>
  request.post<GameServerInfo>(`${base(data.id)}/mode`, data)

export const rotateGameServerKey = (id: string, restart: boolean, key: { keyId: string; secret: string }) =>
  request.post<RotateGameServerKeyResponse>(`${base(id)}/rotate-key`, { id, restart, keyId: key.keyId, secret: key.secret })

export type GameServerAction = 'start' | 'stop' | 'restart'

export const controlGameServer = (id: string, action: GameServerAction, role = '', force = false) =>
  request.post<GameServerInfo>(`${base(id)}/control`, { id, action, role, force })

export const updateGameServerHosting = (data: UpdateGameServerHostingRequest) =>
  request.put<GameServerInfo>(`${base(data.id)}/hosting`, data)

export const getGameServerOverview = (id: string) => request.get<GameServerOverview>(`${base(id)}/overview`)

// ---- 专用设置 ----

export const getGameServerSettings = (id: string) => request.get<GameServerSettings>(`${base(id)}/settings`)

export const saveGameServerSettings = (data: {
  id: string
  revision: string
  values?: Record<string, unknown>
  raw?: string
  restart: boolean
}) => request.put<SaveGameServerSettingsResponse>(`${base(data.id)}/settings`, data)

// ---- 更新 ----

export const getGameServerBuild = (id: string) => request.get<GameServerBuild>(`${base(id)}/build`)

export const checkGameServerBuild = (id: string, path: string) =>
  request.post<GameServerBuildInfo>(`${base(id)}/build/check`, { id, path })

export const updateGameServerBuild = (id: string, path: string, healthTimeoutSeconds = 0) =>
  request.post<GameServerTaskResponse>(`${base(id)}/build/update`, { id, path, healthTimeoutSeconds })

export const getGameServerResource = (id: string) => request.get<GameServerResource>(`${base(id)}/resource`)

export const runGameServerResourceTask = (id: string, action: 'verify' | 'update', masterMetadata = '') =>
  request.post<GameServerTaskResponse>(`${base(id)}/resource/run`, { id, action, masterMetadata })

export const getGameServerTask = (id: string, taskId: string) =>
  request.get<GameServerTask>(`${base(id)}/tasks/${seg(taskId)}`)

export const cancelGameServerTask = (id: string, taskId: string) =>
  request.post<GameServerTask>(`${base(id)}/tasks/${seg(taskId)}/cancel`, {})

export const listGameServerHistory = (id: string, params: { kind?: string; page?: number; pageSize?: number }) =>
  request.get<ListGameServerHistoryResponse>(`${base(id)}/history`, { params })

// ---- 备份 ----

export const listGameServerBackups = (id: string, page = 1, pageSize = 20) =>
  request.get<ListGameServerBackupsResponse>(`${base(id)}/backups`, { params: { page, pageSize } })

export const createGameServerBackup = (id: string, note = '') =>
  request.post<GameServerTaskResponse>(`${base(id)}/backups`, { id, note })

export const deleteGameServerBackup = (id: string, backupId: string) =>
  request.delete<void>(`${base(id)}/backups/${seg(backupId)}`)

export const restoreGameServerBackup = (id: string, backupId: string) =>
  request.post<GameServerTaskResponse>(`${base(id)}/backups/${seg(backupId)}/restore`, {})

export const downloadGameServerBackup = (id: string, backupId: string) =>
  request.get<DownloadGameServerBackupResponse>(`${base(id)}/backups/${seg(backupId)}/download`)

export const updateGameServerBackupPolicy = (id: string, data: { enabled: boolean; intervalHours: number; keep: number }) =>
  request.put<GameServerBackupPolicy>(`${base(id)}/backup-policy`, { id, ...data })

// ---- 日志 ----

export const listGameServerLogs = (id: string) => request.get<ListGameServerLogsResponse>(`${base(id)}/logs`)

export const tailGameServerLog = (
  id: string,
  module: string,
  params: { limit?: number; level?: string; q?: string; before?: number },
) => request.get<TailGameServerLogResponse>(`${base(id)}/logs/${seg(module)}`, { params })

// ---- 文件（服务端目录；与应用实例的文件接口同构）----

export const gameServerFileList = (id: string, params: Omit<GetInstanceFileListRequest, 'id'>) =>
  request.get<GetInstanceFileListResponse>(`${base(id)}/files`, { params })

export const gameServerFileTree = (id: string, path: string) =>
  request.get<GetInstanceFileTreeResponse>(`${base(id)}/files/tree`, { params: { path } })

export const createGameServerFile = (id: string, info: { path: string; isDir: boolean; content?: string }) =>
  request.post<CreateInstanceFileResponse>(`${base(id)}/files/create`, { id, info })

export const renameGameServerFile = (id: string, path: string, newName: string) =>
  request.post<RenameInstanceFileResponse>(`${base(id)}/files/rename`, { id, path, newName })

export const deleteGameServerFiles = (id: string, paths: string[]) =>
  request.post<BatchDeleteInstanceFileResponse>(`${base(id)}/files/delete`, { id, paths })

export const copyGameServerFiles = (id: string, paths: string[], targetPath: string) =>
  request.post<CopyInstanceFileResponse>(`${base(id)}/files/copy`, { id, paths, targetPath })

export const cutGameServerFiles = (id: string, paths: string[], targetPath: string) =>
  request.post<CutInstanceFileResponse>(`${base(id)}/files/cut`, { id, paths, targetPath })

export const compressGameServerFiles = (id: string, paths: string[], targetPath?: string) =>
  request.post<BatchCompressInstanceFileResponse>(`${base(id)}/files/compress`, { id, paths, targetPath })

export const unzipGameServerFile = (id: string, path: string, targetPath?: string) =>
  request.post<UnzipInstanceFileResponse>(`${base(id)}/files/unzip`, { id, path, targetPath })

export const editGameServerFile = (id: string, path: string, content: string) =>
  request.post<EditInstanceFileResponse>(`${base(id)}/files/edit`, { id, path, content })

export const gameServerFilePreSign = (id: string, path: string) =>
  request.get<InstanceFilePreSignResponse>(`${base(id)}/files/pre-sign`, { params: { path } })

export const gameServerFilePreSignUpload = (
  id: string,
  params: { path: string; fileName: string; fileSize: number; hash: string; partSize?: number },
) => request.get<InstanceFilePreSignUploadResponse>(`${base(id)}/files/upload/pre-sign`, { params })

// ---- 原始路由 ----

/** 托管进程控制台（WebSocket）路径：?id=<服务器>&role=main|resource（token 由页面附加）。 */
export const gameServerConsolePath = '/api/v1/game/server-console/ws'

/** 服务器实时动态（SSE，转发 MGP /events）路径（相对 /api/v1）。 */
export const gameServerEventsPath = (id: string) => `/game/server-events/${seg(id)}`
