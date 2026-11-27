// 蓝色星原 GM（AzurPromiliaGMService，权限 game:gm）：每个函数对应一个接口，筛选与分页由服务端完成。
import request from '@/utils/request'
import type {
  AdjustItemsRequest,
  BanPlayersRequest,
  CodeBatchResponse,
  CreateCodesRequest,
  CreateCodesResponse,
  DailyStatsRequest,
  GetPlayerResponse,
  IDOperationRequest,
  IDPageRequest,
  ListByIDsRequest,
  ListPageRequest,
  ListPlayersRequest,
  ListPlayersResponse,
  MailPlayersRequest,
  MailResponse,
  OperationResponse,
  PageRequest,
  PlayersOperationRequest,
  RenamePlayerRequest,
  ResetPasswordRequest,
  SendMailRequest,
  SendMailResponse,
  SetCodeBatchDisabledRequest,
  SetHeroRequest,
  SetPetRequest,
} from '@/types/v1/azurpromilia/gm'
import type {
  CodeBatch,
  CodeBatchPage,
  CodePage,
  DailyStats,
  GmMail,
  GmMailPage,
  HeroPage,
  ItemPage,
  LoginPage,
  MailPage,
  Manifest,
  PetPage,
  RedemptionPage,
  Status,
} from '@/types/v1/azurpromilia/types'

const seg = (value: string) => encodeURIComponent(value)
const base = (serverId: string) => `/azurpromilia/gm/${seg(serverId)}`
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

// ---------------- 服务器 ----------------

export const getManifest = (serverId: string) => get<Manifest>(`${base(serverId)}/manifest`)
export const getStatus = (serverId: string) => get<Status>(`${base(serverId)}/status`)
export const getDailyStats = (req: DailyStatsRequest) => get<DailyStats>(`${base(req.serverId)}/stats/daily`, query(req, 'serverId'))

// ---------------- 玩家 ----------------

export const listPlayers = (req: Partial<ListPlayersRequest> & { serverId: string }) =>
  get<ListPlayersResponse>(`${base(req.serverId)}/players`, query(req, 'serverId'))
export const getPlayer = (serverId: string, pid: string) => get<GetPlayerResponse>(player(serverId, pid))
export const listPlayerLogins = (req: PageRequest) => get<LoginPage>(`${player(req.serverId, req.pid)}/logins`, query(req, 'serverId', 'pid'))
export const listPlayerItems = (req: ListByIDsRequest) => get<ItemPage>(`${player(req.serverId, req.pid)}/items`, query(req, 'serverId', 'pid'))
export const listPlayerCurrencies = (serverId: string, pid: string) => get<ItemPage>(`${player(serverId, pid)}/currencies`)
export const listPlayerHeroes = (req: ListByIDsRequest) => get<HeroPage>(`${player(req.serverId, req.pid)}/heroes`, query(req, 'serverId', 'pid'))
export const listPlayerPets = (req: ListByIDsRequest) => get<PetPage>(`${player(req.serverId, req.pid)}/pets`, query(req, 'serverId', 'pid'))
export const listPlayerMails = (req: PageRequest) => get<MailPage>(`${player(req.serverId, req.pid)}/mails`, query(req, 'serverId', 'pid'))

// ---------------- 对玩家的操作 ----------------

export const adjustItems = (req: AdjustItemsRequest) => post<OperationResponse>(`${base(req.serverId)}/players/items/adjust`, req)
export const mailPlayers = (req: MailPlayersRequest) => post<OperationResponse>(`${base(req.serverId)}/players/mail`, req)
export const kickPlayers = (req: PlayersOperationRequest) => post<OperationResponse>(`${base(req.serverId)}/players/kick`, req)
export const banPlayers = (req: BanPlayersRequest) => post<OperationResponse>(`${base(req.serverId)}/players/ban`, req)
export const unbanPlayers = (req: PlayersOperationRequest) => post<OperationResponse>(`${base(req.serverId)}/players/unban`, req)
export const resetPassword = (req: ResetPasswordRequest) => post<OperationResponse>(`${base(req.serverId)}/players/password`, req)
export const renamePlayer = (req: RenamePlayerRequest) => post<OperationResponse>(`${base(req.serverId)}/players/rename`, req)
export const setPlayerHero = (req: SetHeroRequest) => post<OperationResponse>(`${player(req.serverId, req.pid)}/heroes/${req.seq}`, req)
export const setPlayerPet = (req: SetPetRequest) => post<OperationResponse>(`${player(req.serverId, req.pid)}/pets/${req.guid}`, req)

// ---------------- 邮件补偿 ----------------

export const listMails = (req: ListPageRequest) => get<GmMailPage>(`${base(req.serverId)}/mails`, query(req, 'serverId'))
export const getMail = (serverId: string, id: string) => get<GmMail>(`${base(serverId)}/mails/${seg(id)}`)
export const sendMail = (req: SendMailRequest) => post<SendMailResponse>(`${base(req.serverId)}/mails`, req)
export const revokeMail = (req: IDOperationRequest) => post<MailResponse>(`${base(req.serverId)}/mails/${seg(req.id)}/revoke`, req)

// ---------------- 兑换码 ----------------

export const listCodeBatches = (req: ListPageRequest) => get<CodeBatchPage>(`${base(req.serverId)}/codes`, query(req, 'serverId'))
export const getCodeBatch = (serverId: string, id: string) => get<CodeBatch>(`${base(serverId)}/codes/${seg(id)}`)
export const createCodes = (req: CreateCodesRequest) => post<CreateCodesResponse>(`${base(req.serverId)}/codes`, req)
export const setCodeBatchDisabled = (req: SetCodeBatchDisabledRequest) =>
  post<CodeBatchResponse>(`${base(req.serverId)}/codes/${seg(req.id)}/disabled`, req)
export const listBatchCodes = (req: IDPageRequest) => get<CodePage>(`${base(req.serverId)}/codes/${seg(req.id)}/codes`, query(req, 'serverId', 'id'))
export const listBatchRedemptions = (req: IDPageRequest) =>
  get<RedemptionPage>(`${base(req.serverId)}/codes/${seg(req.id)}/redemptions`, query(req, 'serverId', 'id'))
