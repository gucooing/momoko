// 蓝色星原 的游戏空间（AzurPromiliaPassportService，api/proto/v1/azurpromilia/passport.proto）：我绑定的游戏账号的数据与自助操作。
import { http } from './request'
import type {
  MySelfServiceResponse,
  SelfServiceBanRequest,
  SelfServiceItemsRequest,
  SelfServiceMailRequest,
  SelfServicePasswordRequest,
  SelfServiceRenameRequest,
  SelfServiceRequest,
} from '@/types/v1/azurpromilia/passport'
import type { HeroPage, ItemPage, LoginPage, MailPage, PetPage, PlayerDetail } from '@/types/v1/azurpromilia/types'

const base = (serverId: string) => `/azurpromilia/games/${encodeURIComponent(serverId)}`

// ids：只取这些道具 / 英雄 / 奇波 ID（为空时不筛选）
const page = (cursor: string, limit: number, ids: number[] = []) => {
  const query: Record<string, unknown> = { limit }
  if (cursor) query.cursor = cursor
  if (ids.length) query.ids = ids
  return query
}

export const getMyPlayer = (serverId: string) => http.get<PlayerDetail>(`${base(serverId)}/player`)
export const listMyLogins = (serverId: string, cursor: string, limit: number) => http.get<LoginPage>(`${base(serverId)}/logins`, page(cursor, limit))
export const listMyItems = (serverId: string, cursor: string, limit: number, ids: number[] = []) =>
  http.get<ItemPage>(`${base(serverId)}/items`, page(cursor, limit, ids))
export const listMyCurrencies = (serverId: string) => http.get<ItemPage>(`${base(serverId)}/currencies`)
export const listMyHeroes = (serverId: string, cursor: string, limit: number, ids: number[] = []) =>
  http.get<HeroPage>(`${base(serverId)}/heroes`, page(cursor, limit, ids))
export const listMyPets = (serverId: string, cursor: string, limit: number, ids: number[] = []) =>
  http.get<PetPage>(`${base(serverId)}/pets`, page(cursor, limit, ids))
export const listMyMails = (serverId: string, cursor: string, limit: number) => http.get<MailPage>(`${base(serverId)}/mails`, page(cursor, limit))

// ---------------- 自助操作 ----------------

const self = <T extends { serverId: string }>(path: string) => (req: T) => http.post<MySelfServiceResponse>(`${base(req.serverId)}/self/${path}`, req)

export const mailMe = self<SelfServiceMailRequest>('mail')
export const adjustMyItems = self<SelfServiceItemsRequest>('items')
export const kickMe = self<SelfServiceRequest>('kick')
export const banMe = self<SelfServiceBanRequest>('ban')
export const unbanMe = self<SelfServiceRequest>('unban')
export const setMyPassword = self<SelfServicePasswordRequest>('password')
export const renameMe = self<SelfServiceRenameRequest>('rename')
