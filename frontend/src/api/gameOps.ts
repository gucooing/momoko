import request from '@/utils/request'
import type {
  CreateGiftCodeBatchRequest,
  DeleteGameMailResponse,
  DryRunGameMailResponse,
  GameMail,
  GetGiftCodeBatchResponse,
  GiftCodeBatch,
  ListGameMailsRequest,
  ListGameMailsResponse,
  ListGiftCodeBatchesRequest,
  ListGiftCodeBatchesResponse,
  ListGiftCodeRedeemsRequest,
  ListGiftCodeRedeemsResponse,
  ListGiftCodesRequest,
  ListGiftCodesResponse,
  RevokeGameMailRequest,
  SaveGameMailRequest,
  SendGameMailRequest,
  UpdateGiftCodeBatchRequest,
} from '@/types/v1/game_ops'

// GM 运营 · 邮件补偿与兑换码（权限 game:gm）。

const base = (serverId: string) => `/game/gm/${encodeURIComponent(serverId)}`
const seg = (value: string) => encodeURIComponent(value)

// ---------------- 邮件补偿 ----------------

export const listGameMails = (params: ListGameMailsRequest) => {
  const { serverId, ...rest } = params
  return request.get<ListGameMailsResponse>(`${base(serverId)}/mails`, { params: rest })
}

export const getGameMail = (serverId: string, id: string) => request.get<GameMail>(`${base(serverId)}/mails/${seg(id)}`)

export const createGameMail = (data: SaveGameMailRequest) => request.post<GameMail>(`${base(data.serverId)}/mails`, data)

export const updateGameMail = (data: SaveGameMailRequest) =>
  request.put<GameMail>(`${base(data.serverId)}/mails/${seg(data.id)}`, data)

export const deleteGameMail = (serverId: string, id: string) =>
  request.delete<DeleteGameMailResponse>(`${base(serverId)}/mails/${seg(id)}`)

export const duplicateGameMail = (serverId: string, id: string) =>
  request.post<GameMail>(`${base(serverId)}/mails/${seg(id)}/duplicate`, { serverId, id })

export const dryRunGameMail = (serverId: string, id: string) =>
  request.post<DryRunGameMailResponse>(`${base(serverId)}/mails/${seg(id)}/dry-run`, { serverId, id })

export const sendGameMail = (data: SendGameMailRequest) =>
  request.post<GameMail>(`${base(data.serverId)}/mails/${seg(data.id)}/send`, data)

export const cancelGameMail = (serverId: string, id: string) =>
  request.post<GameMail>(`${base(serverId)}/mails/${seg(id)}/cancel`, { serverId, id })

export const revokeGameMail = (data: RevokeGameMailRequest) =>
  request.post<GameMail>(`${base(data.serverId)}/mails/${seg(data.id)}/revoke`, data)

// ---------------- 兑换码 ----------------

export const listGiftCodeBatches = (params: ListGiftCodeBatchesRequest) => {
  const { serverId, ...rest } = params
  return request.get<ListGiftCodeBatchesResponse>(`${base(serverId)}/code-batches`, { params: rest })
}

export const getGiftCodeBatch = (serverId: string, id: string) =>
  request.get<GetGiftCodeBatchResponse>(`${base(serverId)}/code-batches/${seg(id)}`)

export const createGiftCodeBatch = (data: CreateGiftCodeBatchRequest) =>
  request.post<GiftCodeBatch>(`${base(data.serverId)}/code-batches`, data)

export const updateGiftCodeBatch = (data: UpdateGiftCodeBatchRequest) =>
  request.put<GiftCodeBatch>(`${base(data.serverId)}/code-batches/${seg(data.id)}`, data)

export const disableGiftCodeBatch = (serverId: string, id: string) =>
  request.post<GiftCodeBatch>(`${base(serverId)}/code-batches/${seg(id)}/disable`, { serverId, id })

export const listGiftCodes = (params: ListGiftCodesRequest) => {
  const { serverId, batchId, ...rest } = params
  return request.get<ListGiftCodesResponse>(`${base(serverId)}/code-batches/${seg(batchId)}/codes`, { params: rest })
}

export const listGiftCodeRedeems = (params: ListGiftCodeRedeemsRequest) => {
  const { serverId, ...rest } = params
  return request.get<ListGiftCodeRedeemsResponse>(`${base(serverId)}/code-redeems`, { params: rest })
}
