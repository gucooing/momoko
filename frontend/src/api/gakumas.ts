// 学マス GM 运营（GakumasGMService，权限 game:gm）：每个函数对应一个接口，筛选与分页由服务端完成。
import request from '@/utils/request'
import type {
  GetGmConfigResponse,
  GmConfigDraft,
  GmConfigRevision,
  ListGmConfigRevisionsResponse,
  PublishGmConfigResponse,
  ValidateGmConfigResponse,
  DiscardGmConfigDraftResponse,
} from '@/types/v1/game_gm'
import type {
  AddExpRequest,
  AdjustItemRequest,
  AdjustJewelRequest,
  ArchiveExportResponse,
  BackupArchiveRequest,
  BackupResponse,
  BanPlayersRequest,
  BanResponse,
  ChangesResponse,
  GetDashboardRequest,
  GetPlayerResponse,
  GiftRevokeResponse,
  GiftSendResponse,
  GrantAllRequest,
  IdolCardGrantResponse,
  KickResponse,
  ListGashaDrawsRequest,
  ListGiftsRequest,
  ListIdolCardsRequest,
  ListItemsRequest,
  ListMemoriesRequest,
  ListConfigDraftsResponse,
  ListPlayersRequest,
  ListPlayersResponse,
  ListProducesRequest,
  ListSupportCardsRequest,
  MaintenanceResponse,
  PageRequest,
  PasswordResponse,
  MemoryPhotoRequest,
  PlayerMemoryRequest,
  PlayersOperationRequest,
  RankingRequest,
  RenamePlayerRequest,
  RenameResponse,
  RentalResponse,
  ResetPasswordRequest,
  RestoreArchiveRequest,
  RestoreResponse,
  RevokeGiftsRequest,
  SeminarClearResponse,
  SeminarResetResponse,
  SendGiftRequest,
  SetMaintenanceRequest,
  SetPvpRateRequest,
  SetRentalMemoryRequest,
  SetRentalSupportCardRequest,
  SupportCardGrantResponse,
  TowerRankingRequest,
} from '@/types/v1/gakumas/gm'
import type {
  BackupList,
  Dashboard,
  GashaDrawPage,
  GashaStatList,
  GiftPage,
  IdolCardPage,
  ItemPage,
  Manifest,
  MemoryDetail,
  MemoryPage,
  MeishiImage,
  MemoryPhoto,
  PlayerPvp,
  PlayerSeminar,
  ProducePage,
  PvpRankingPage,
  SeminarStatList,
  SessionPage,
  Status,
  SupportCardPage,
  TowerRankingPage,
} from '@/types/v1/gakumas/types'

const seg = (value: string) => encodeURIComponent(value)
const base = (serverId: string) => `/gakumas/gm/${seg(serverId)}`
const player = (serverId: string, pid: string) => `${base(serverId)}/players/${seg(pid)}`

/** GET 参数：去掉路径里的字段与空值，列表参数重复给出。 */
const query = <T extends object>(req: T, ...path: (keyof T)[]) => {
  const out: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(req)) {
    if (path.includes(k as keyof T) || v === undefined || v === null || v === '' || v === 0 || v === false) continue
    if (Array.isArray(v) && v.length === 0) continue
    out[k] = v
  }
  return out
}

const get = async <T>(url: string, params?: Record<string, unknown>) => (await request.get<T>(url, { params, paramsSerializer: { indexes: null } })).data
const post = async <T>(url: string, body: unknown) => (await request.post<T>(url, body)).data
const put = async <T>(url: string, body: unknown) => (await request.put<T>(url, body)).data

// ---------------- 服务器 ----------------

export const getGakumasManifest = (serverId: string) => get<Manifest>(`${base(serverId)}/manifest`)
export const getGakumasStatus = (serverId: string) => get<Status>(`${base(serverId)}/status`)
export const getDashboard = (req: GetDashboardRequest) => get<Dashboard>(`${base(req.serverId)}/dashboard`, query(req, 'serverId'))
export const setMaintenance = (req: SetMaintenanceRequest) => put<MaintenanceResponse>(`${base(req.serverId)}/maintenance`, req)

// ---------------- 玩家 ----------------

export const listPlayers = (req: ListPlayersRequest) => get<ListPlayersResponse>(`${base(req.serverId)}/players`, query(req, 'serverId'))
export const getPlayer = (serverId: string, pid: string) => get<GetPlayerResponse>(player(serverId, pid))
export const listPlayerSessions = (req: PageRequest) => get<SessionPage>(`${player(req.serverId, req.pid)}/sessions`, query(req, 'serverId', 'pid'))
export const listPlayerIdolCards = (req: ListIdolCardsRequest) =>
  get<IdolCardPage>(`${player(req.serverId, req.pid)}/idol-cards`, query(req, 'serverId', 'pid'))
export const listPlayerSupportCards = (req: ListSupportCardsRequest) =>
  get<SupportCardPage>(`${player(req.serverId, req.pid)}/support-cards`, query(req, 'serverId', 'pid'))
export const listPlayerItems = (req: ListItemsRequest) => get<ItemPage>(`${player(req.serverId, req.pid)}/items`, query(req, 'serverId', 'pid'))
export const listPlayerGifts = (req: ListGiftsRequest) => get<GiftPage>(`${player(req.serverId, req.pid)}/gifts`, query(req, 'serverId', 'pid'))
export const listPlayerGashaDraws = (req: ListGashaDrawsRequest) =>
  get<GashaDrawPage>(`${player(req.serverId, req.pid)}/gasha-draws`, query(req, 'serverId', 'pid'))
export const listPlayerMemories = (req: ListMemoriesRequest) =>
  get<MemoryPage>(`${player(req.serverId, req.pid)}/memories`, query(req, 'serverId', 'pid'))
export const getMemoryPhoto = (req: MemoryPhotoRequest) => get<MemoryPhoto>(`${base(req.serverId)}/memory-photo`, query(req, 'serverId'))
export const getPlayerMemory = (req: PlayerMemoryRequest) =>
  get<MemoryDetail>(`${player(req.serverId, req.pid)}/memories/${seg(req.memoryId)}`)
export const getPlayerMeishiImage = (serverId: string, pid: string) => get<MeishiImage>(`${player(serverId, pid)}/meishi-image`)
export const getPlayerPvp = (serverId: string, pid: string) => get<PlayerPvp>(`${player(serverId, pid)}/pvp`)
export const getPlayerSeminar = (serverId: string, pid: string) => get<PlayerSeminar>(`${player(serverId, pid)}/seminar`)
export const listPlayerBackups = (serverId: string, pid: string) => get<BackupList>(`${player(serverId, pid)}/backups`)

// ---------------- 全服 ----------------

export const getGashaStats = (serverId: string) => get<GashaStatList>(`${base(serverId)}/gashas/stats`)
export const listProduces = (req: ListProducesRequest) => get<ProducePage>(`${base(req.serverId)}/produces`, query(req, 'serverId'))
export const getPvpRanking = (req: RankingRequest) => get<PvpRankingPage>(`${base(req.serverId)}/pvp/ranking`, query(req, 'serverId'))
export const getSeminarStats = (serverId: string) => get<SeminarStatList>(`${base(serverId)}/seminar/stats`)
export const getTowerRanking = (req: TowerRankingRequest) =>
  get<TowerRankingPage>(`${base(req.serverId)}/towers/${seg(req.towerId)}/ranking`, query(req, 'serverId', 'towerId'))

// ---------------- 操作 ----------------

export const sendGift = (req: SendGiftRequest) => post<GiftSendResponse>(`${base(req.serverId)}/gifts`, req)
export const revokeGifts = (req: RevokeGiftsRequest) => post<GiftRevokeResponse>(`${base(req.serverId)}/players/gifts/revoke`, req)
export const grantAllIdolCards = (req: GrantAllRequest) =>
  post<IdolCardGrantResponse>(`${base(req.serverId)}/players/idol-cards/grant-all`, req)
export const grantAllSupportCards = (req: GrantAllRequest) =>
  post<SupportCardGrantResponse>(`${base(req.serverId)}/players/support-cards/grant-all`, req)
export const adjustItem = (req: AdjustItemRequest) => post<ChangesResponse>(`${base(req.serverId)}/players/items/adjust`, req)
export const adjustJewel = (req: AdjustJewelRequest) => post<ChangesResponse>(`${base(req.serverId)}/players/jewels/adjust`, req)
export const refillActionPoint = (req: PlayersOperationRequest) =>
  post<ChangesResponse>(`${base(req.serverId)}/players/action-point/refill`, req)
export const addExp = (req: AddExpRequest) => post<ChangesResponse>(`${base(req.serverId)}/players/exp/add`, req)
export const kickPlayers = (req: PlayersOperationRequest) => post<KickResponse>(`${base(req.serverId)}/players/kick`, req)
export const banPlayers = (req: BanPlayersRequest) => post<BanResponse>(`${base(req.serverId)}/players/ban`, req)
export const unbanPlayers = (req: PlayersOperationRequest) => post<BanResponse>(`${base(req.serverId)}/players/unban`, req)
export const resetPassword = (req: ResetPasswordRequest) => post<PasswordResponse>(`${base(req.serverId)}/players/password`, req)
export const renamePlayer = (req: RenamePlayerRequest) => post<RenameResponse>(`${base(req.serverId)}/players/name`, req)
export const setRentalMemory = (req: SetRentalMemoryRequest) => post<RentalResponse>(`${player(req.serverId, req.pid)}/rental-memory`, req)
export const setRentalSupportCard = (req: SetRentalSupportCardRequest) =>
  post<RentalResponse>(`${player(req.serverId, req.pid)}/rental-support-card`, req)
export const setPvpRate = (req: SetPvpRateRequest) => post<ChangesResponse>(`${base(req.serverId)}/players/pvp/rate`, req)
export const resetPvpDaily = (req: PlayersOperationRequest) => post<ChangesResponse>(`${base(req.serverId)}/players/pvp/reset-daily`, req)
export const clearSeminar = (req: PlayersOperationRequest) => post<SeminarClearResponse>(`${base(req.serverId)}/players/seminar/clear`, req)
export const resetSeminar = (req: PlayersOperationRequest) => post<SeminarResetResponse>(`${base(req.serverId)}/players/seminar/reset`, req)
export const exportArchive = (req: PlayersOperationRequest) =>
  post<ArchiveExportResponse>(`${base(req.serverId)}/players/archive/export`, req)
export const backupArchive = (req: BackupArchiveRequest) => post<BackupResponse>(`${player(req.serverId, req.pid)}/backups`, req)
export const restoreArchive = (req: RestoreArchiveRequest) =>
  post<RestoreResponse>(`${player(req.serverId, req.pid)}/backups/${seg(String(req.backupId))}/restore`, req)

// ---------------- 配置文档 ----------------

/**
 * 配置文档（游戏服务端的数据文件）：gasha 卡池、notice 公告、pvpRate 竞赛、home 活动日程、research 研究活动、
 * storyEvent 剧情活动、loginBonus 登录奖励。草稿与版本由 Momoko 保存，发布写回服务端并热重载。
 */
export type GakumasConfigDomain = 'gasha' | 'notice' | 'pvpRate' | 'home' | 'research' | 'storyEvent' | 'loginBonus'

/** 一份配置文档的接口。 */
export interface ConfigApi {
  get: (serverId: string) => Promise<GetGmConfigResponse>
  saveDraft: (serverId: string, content: string, baseRevision: string, comment?: string) => Promise<GmConfigDraft>
  discardDraft: (serverId: string) => Promise<DiscardGmConfigDraftResponse>
  validate: (serverId: string, content: string) => Promise<ValidateGmConfigResponse>
  publish: (serverId: string, content: string, baseRevision: string, comment?: string) => Promise<PublishGmConfigResponse>
  revisions: (serverId: string, page: number, pageSize: number) => Promise<ListGmConfigRevisionsResponse>
  revision: (serverId: string, id: string) => Promise<GmConfigRevision>
}

/** 未发布的草稿（不含内容，最近修改的在前）。 */
export const listConfigDrafts = (serverId: string) => get<ListConfigDraftsResponse>(`${base(serverId)}/config-drafts`)

export const configApi = (domain: GakumasConfigDomain): ConfigApi => {
  const path = (serverId: string) => `${base(serverId)}/config/${domain}`
  return {
    get: (serverId) => get<GetGmConfigResponse>(path(serverId)),
    saveDraft: (serverId, content, baseRevision, comment) => put<GmConfigDraft>(`${path(serverId)}/draft`, { serverId, domain, content, baseRevision, comment }),
    discardDraft: async (serverId) => (await request.delete<DiscardGmConfigDraftResponse>(`${path(serverId)}/draft`)).data,
    validate: (serverId, content) => post<ValidateGmConfigResponse>(`${path(serverId)}/validate`, { serverId, domain, content }),
    publish: (serverId, content, baseRevision, comment) =>
      post<PublishGmConfigResponse>(`${path(serverId)}/publish`, { serverId, domain, content, baseRevision, comment }),
    revisions: (serverId, page, pageSize) => get<ListGmConfigRevisionsResponse>(`${path(serverId)}/revisions`, { page, pageSize }),
    revision: (serverId, id) => get<GmConfigRevision>(`${path(serverId)}/revisions/${seg(id)}`),
  }
}
