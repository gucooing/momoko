// 学マス 的游戏空间（GakumasPassportService，api/proto/v1/gakumas/passport.proto）：我绑定的游戏账号的数据、目录与自助操作，
// 每个函数对应一个接口；请求与响应只用通行证自己的类型和游戏数据类型，不引用管理端（gm.ts）。
import { http } from './request'
import type {
  MyArchiveExportResponse,
  MyBanResponse,
  MyCatalogRequest,
  MyChangesResponse,
  MyCostumeCatalogRequest,
  MyGashaDrawsRequest,
  MyGiftRevokeResponse,
  MyGiftSendResponse,
  MyGiftsRequest,
  MyIdolCardGrantResponse,
  MyIdolCardSkinCatalogRequest,
  MyIdolCardsRequest,
  MyItemCatalogRequest,
  MyKickResponse,
  MyMemoriesRequest,
  MyMemoryPhotoRequest,
  MyMemoryRequest,
  MyPasswordResponse,
  MyProduceCatalogRequest,
  MyRenameResponse,
  MyRentalMemoryRequest,
  MyRentalSupportCardRequest,
  MySeminarClearResponse,
  MySeminarResetResponse,
  MySupportCardCatalogRequest,
  MySupportCardGrantResponse,
  MySupportCardsRequest,
  PlayerGame,
  SelfServiceBanRequest,
  SelfServiceExpRequest,
  SelfServiceGiftRequest,
  SelfServiceGiftRevokeRequest,
  SelfServiceGrantRequest,
  SelfServiceItemRequest,
  SelfServiceJewelRequest,
  SelfServicePasswordRequest,
  SelfServiceRateRequest,
  SelfServiceRenameRequest,
  SelfServiceRequest,
} from '@/types/v1/gakumas/passport'
import type {
  CharacterPage,
  CostumePage,
  CurrencyPage,
  GashaDrawPage,
  GashaEntryPage,
  GiftPage,
  IdolCardPage,
  IdolCardSkinPage,
  ItemEntryPage,
  MemoryAbilityPage,
  MemoryDetail,
  MemoryPage,
  MeishiImage,
  MemoryPhoto,
  MusicPage,
  PlayerDetail,
  PlayerPvp,
  PlayerSeminar,
  ProduceCardEntryPage,
  ProduceEntryPage,
  ProduceItemEntryPage,
  RentalChange,
  SeminarEntryPage,
  SupportCardEntryPage,
  SupportCardPage,
} from '@/types/v1/gakumas/types'

const base = (serverId: string) => `/gakumas/games/${encodeURIComponent(serverId)}`

/** GET 参数：去掉 serverId 与空值。 */
const query = (req: object) => {
  const out: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(req)) {
    if (k === 'serverId' || v === undefined || v === null || v === '' || v === 0 || v === false) continue
    if (Array.isArray(v) && v.length === 0) continue
    out[k] = v
  }
  return out
}

const get = <T>(req: { serverId: string }, path: string) => http.get<T>(`${base(req.serverId)}${path}`, query(req))
const post = <T>(req: { serverId: string }, path: string) => http.post<T>(`${base(req.serverId)}${path}`, req)

// ---------------- 我的数据 ----------------

export const getGame = (serverId: string) => http.get<PlayerGame>(`${base(serverId)}/game`)
export const getMyPlayer = (serverId: string) => http.get<PlayerDetail>(`${base(serverId)}/player`)
export const listMyIdolCards = (req: MyIdolCardsRequest) => get<IdolCardPage>(req, '/idol-cards')
export const listMySupportCards = (req: MySupportCardsRequest) => get<SupportCardPage>(req, '/support-cards')
export const listMyGifts = (req: MyGiftsRequest) => get<GiftPage>(req, '/gifts')
export const listMyGashaDraws = (req: MyGashaDrawsRequest) => get<GashaDrawPage>(req, '/gasha-draws')
export const listMyMemories = (req: MyMemoriesRequest) => get<MemoryPage>(req, '/memories')
export const getMyMemory = (req: MyMemoryRequest) => http.get<MemoryDetail>(`${base(req.serverId)}/memories/${encodeURIComponent(req.memoryId)}`)
export const getMemoryPhoto = (req: MyMemoryPhotoRequest) => get<MemoryPhoto>(req, '/memory-photo')
export const setMyRentalMemory = (req: MyRentalMemoryRequest) => post<RentalChange>(req, '/rental-memory')
export const setMyRentalSupportCard = (req: MyRentalSupportCardRequest) => post<RentalChange>(req, '/rental-support-card')
export const getMyMeishiImage = (serverId: string) => http.get<MeishiImage>(`${base(serverId)}/meishi-image`)
export const getMyPvp = (serverId: string) => http.get<PlayerPvp>(`${base(serverId)}/pvp`)
export const getMySeminar = (serverId: string) => http.get<PlayerSeminar>(`${base(serverId)}/seminar`)
export const listOngoingGashas = (serverId: string) => http.get<GashaEntryPage>(`${base(serverId)}/gashas/ongoing`)

// ---------------- 目录 ----------------

export const listCurrencies = (req: MyCatalogRequest) => get<CurrencyPage>(req, '/catalog/currencies')
export const listItemCatalog = (req: MyItemCatalogRequest) => get<ItemEntryPage>(req, '/catalog/items')
export const listIdolCardSkins = (req: MyIdolCardSkinCatalogRequest) => get<IdolCardSkinPage>(req, '/catalog/idol-card-skins')
export const listSupportCardCatalog = (req: MySupportCardCatalogRequest) => get<SupportCardEntryPage>(req, '/catalog/support-cards')
export const listCharacters = (req: MyCatalogRequest) => get<CharacterPage>(req, '/catalog/characters')
export const listCostumes = (req: MyCostumeCatalogRequest) => get<CostumePage>(req, '/catalog/costumes')
export const listCostumeHeads = (req: MyCostumeCatalogRequest) => get<CostumePage>(req, '/catalog/costume-heads')
export const listMusic = (req: MyCatalogRequest) => get<MusicPage>(req, '/catalog/music')
export const listProduceCatalog = (req: MyProduceCatalogRequest) => get<ProduceEntryPage>(req, '/catalog/produces')
export const listSeminars = (req: MyCatalogRequest) => get<SeminarEntryPage>(req, '/catalog/seminars')
export const listProduceCards = (req: MyCatalogRequest) => get<ProduceCardEntryPage>(req, '/catalog/produce-cards')
export const listProduceItems = (req: MyCatalogRequest) => get<ProduceItemEntryPage>(req, '/catalog/produce-items')
export const listMemoryAbilities = (req: MyCatalogRequest) => get<MemoryAbilityPage>(req, '/catalog/memory-abilities')

// ---------------- 自助操作 ----------------

export const refillMyActionPoint = (req: SelfServiceRequest) => post<MyChangesResponse>(req, '/self-service/action-point-refill')
export const grantMyIdolCards = (req: SelfServiceGrantRequest) => post<MyIdolCardGrantResponse>(req, '/self-service/idol-card-grant-all')
export const grantMySupportCards = (req: SelfServiceGrantRequest) =>
  post<MySupportCardGrantResponse>(req, '/self-service/support-card-grant-all')
export const kickMe = (req: SelfServiceRequest) => post<MyKickResponse>(req, '/self-service/kick')
export const renameMe = (req: SelfServiceRenameRequest) => post<MyRenameResponse>(req, '/self-service/rename')
export const resetMyPvpDaily = (req: SelfServiceRequest) => post<MyChangesResponse>(req, '/self-service/pvp-reset-daily')
export const exportMyArchive = (req: SelfServiceRequest) => post<MyArchiveExportResponse>(req, '/self-service/archive-export')
export const sendMyGift = (req: SelfServiceGiftRequest) => post<MyGiftSendResponse>(req, '/self-service/gift-send')
export const revokeMyGifts = (req: SelfServiceGiftRevokeRequest) => post<MyGiftRevokeResponse>(req, '/self-service/gift-revoke')
export const adjustMyJewel = (req: SelfServiceJewelRequest) => post<MyChangesResponse>(req, '/self-service/jewel-adjust')
export const adjustMyItem = (req: SelfServiceItemRequest) => post<MyChangesResponse>(req, '/self-service/item-adjust')
export const addMyExp = (req: SelfServiceExpRequest) => post<MyChangesResponse>(req, '/self-service/exp-add')
export const banMe = (req: SelfServiceBanRequest) => post<MyBanResponse>(req, '/self-service/ban')
export const unbanMe = (req: SelfServiceRequest) => post<MyBanResponse>(req, '/self-service/unban')
export const setMyPassword = (req: SelfServicePasswordRequest) => post<MyPasswordResponse>(req, '/self-service/password')
export const setMyPvpRate = (req: SelfServiceRateRequest) => post<MyChangesResponse>(req, '/self-service/pvp-rate')
export const clearMySeminar = (req: SelfServiceRequest) => post<MySeminarClearResponse>(req, '/self-service/seminar-clear')
export const resetMySeminar = (req: SelfServiceRequest) => post<MySeminarResetResponse>(req, '/self-service/seminar-reset')
