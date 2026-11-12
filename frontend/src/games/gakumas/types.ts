/**
 * 学マス（gakumas）MGP 数据形状：玩家列表与详情、各玩家视图、报表、结构化的指令参数与结果、
 * 事件 data，以及可表单编辑的配置域（data/Gasha.json、data/Notice.json）。
 * 字段名与类型以 gakumas 服务端 mgp 包为准；时间均为 Unix 毫秒，pid 为十进制字符串。
 */
import type { ListPage, MaintenanceStatus } from '../mgp'

// ---------------------------------------------------------------------------
// 枚举值（协议数值）
// ---------------------------------------------------------------------------

/**
 * 奖励资源类型（协议 ResourceType）：2 道具、4 支援卡、5 偶像卡皮肤、8 AP、9 制作人经验、
 * 12 乐曲、13 发型、14 服装、1100 钻石（免费）、1101 付费钻石。
 */
export type ResourceType = 2 | 4 | 5 | 8 | 9 | 12 | 13 | 14 | 1100 | 1101

/** 礼物来源：0 游戏内、1 管理指令、2 邮件、3 兑换码。 */
export type GiftSourceKind = 0 | 1 | 2 | 3

/** 礼物状态。 */
export type GiftState = 'unreceived' | 'received' | 'expired'

/** 指令参数中的礼物来源名。 */
export type GiftSourceName = 'gm' | 'mail' | 'code'

/** 存档备份种类：1 手动、2 指令前自动、3 回档前自动。 */
export type BackupKind = 1 | 2 | 3

/** 稀有度：1 R、2 SR、3 SSR。 */
export type Rarity = 1 | 2 | 3

/** 培育评价（ResultGrade）：1 F … 8 A、10 S、12 SS、14 SSS、16 SSSS、18 SSSSS，奇数为对应的 +。 */
export type ResultGrade = number

/** 卡池类型（GashaType）：1 普通、2 阶梯、3 自选 UP、4 连抽、5 阶梯自选 UP。 */
export type GashaType = 1 | 2 | 3 | 4 | 5

/** 公告分类：1 资讯、2 不具合、3 PR。 */
export type NoticeCategory = 1 | 2 | 3

/** 公告类型：1 更新、2 卡池、3 活动宣传、4 活动、5 重要、6 资讯、7 维护、8 不具合。 */
export type NoticeType = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8

// ---------------------------------------------------------------------------
// 玩家列表与详情
// ---------------------------------------------------------------------------

/** 玩家列表筛选。 */
export type PlayerFilter = 'all' | 'online' | 'activeToday' | 'banned'

/** 玩家列表条目（也是 /accounts/verify 的返回）。 */
export interface PlayerListItem {
  pid: string
  /** 游戏内显示的 User ID（9 位数字）。 */
  displayId: string
  /** 登录账号名。 */
  account: string
  name: string
  level: number
  jewelFree: number
  /** 付费与网页商店付费钻石之和。 */
  jewelPaid: number
  createdAt: number
  /** 最后登录游戏的时间，从未登录为 0。 */
  lastLoginAt: number
  online: boolean
  /** 最近一次登录的时间，没有登录记录为 0。 */
  sessionLoginAt: number
  platform: string
  device: string
  remoteAddr: string
  /** 出借记忆的偶像卡 ID，没有出借记忆时为空串。 */
  avatarIdolCardId: string
  /** 封禁到期时间，未封禁为 0；永久封禁为 9223372036854775807（超出 JS 安全整数，按 ≥ Number.MAX_SAFE_INTEGER 判断）。 */
  bannedUntil: number
  banReason: string
}

/** 玩家列表 summary：四个筛选各自的人数。 */
export interface PlayerListSummary {
  all: number
  online: number
  activeToday: number
  banned: number
}

export type PlayerListPage = ListPage<PlayerListItem, PlayerListSummary>

export interface PlayerListQuery {
  /** 匹配账号名、User ID 或昵称的一部分。 */
  q?: string
  filter?: PlayerFilter
  cursor?: string
  /** 默认 20，最大 200。 */
  limit?: number
}

/** GET /players/{pid}。 */
export interface PlayerDetail extends PlayerListItem {
  overview: Overview
  /** 累计与今天的登录次数。 */
  logins: { total: number; today: number }
}

// ---------------------------------------------------------------------------
// 玩家视图
// ---------------------------------------------------------------------------

/** 进行中的培育。 */
export interface ProduceRun {
  produceId: string
  idolCardId: string
  characterId: string
  stepNumber: number
  startedAt: number
}

/** 礼物箱计数。 */
export interface GiftSummary {
  /** 可领取（未领取且未过期）的件数。 */
  unreceived: number
  /** 一天内到期的件数。 */
  expiring: number
  /** 管理端（指令、邮件、兑换码）发出的件数。 */
  fromAdmin: number
}

/**
 * 研修组进度。注意：服务端该结构没有 json 标签，字段名为大写的 GroupID / Total / Cleared。
 */
export interface SeminarProgress {
  GroupID: string
  /** 试题数。 */
  Total: number
  /** 已通关的试题数。 */
  Cleared: number
}

/** overview 视图。 */
export interface Overview {
  name: string
  comment: string
  level: number
  exp: number
  /** 当前等级内已得的经验。 */
  levelExp: number
  /** 升到下一级所需的经验，满级为 0。 */
  nextLevelExp: number
  jewelFree: number
  jewelPaid: number
  jewelWebStorePaid: number
  /** 金币（item-money）数量。 */
  money: number
  actionPoint: number
  actionPointMax: number
  totalFanCount: number
  homeCharacterId: string
  loginCount: number
  tutorialCleared: boolean
  idolCardCount: number
  /** 稀有度（JSON 对象键为字符串 "1"/"2"/"3"）→ 张数。 */
  idolCardRarities: Record<string, number>
  supportCardCount: number
  supportCardRarities: Record<string, number>
  /** 持有中（未过期且数量大于 0）的道具种类数。 */
  itemCount: number
  memoryCount: number
  /** 培育结果记忆数。 */
  produceCount: number
  /** 最好的一次培育（评价高、评价值高者），没有时为 null。 */
  bestProduce: Memory | null
  produce: ProduceRun | null
  gifts: GiftSummary
  seminar: SeminarProgress[]
}

/** idolCards 视图条目，按获得时间从新到旧。 */
export interface IdolCard {
  idolCardId: string
  /** 当前使用的皮肤。 */
  skinId: string
  /** 持有的该卡全部皮肤；没有时为 null。 */
  skinIds: string[] | null
  characterId: string
  rarity: Rarity
  /** 上限解放等级。 */
  levelLimitRank: number
  /** 才能开花等级（0–4）。 */
  potentialRank: number
  primaStella: boolean
  /** 持有的该卡碎片数。 */
  pieces: number
  createdTime: number
}

/** supportCards 视图条目，按获得时间从新到旧。 */
export interface SupportCard {
  supportCardId: string
  rarity: Rarity
  level: number
  levelLimitRank: number
  /** 未使用的重复张数。 */
  stockQuantity: number
  /** 是否为资料出借的支援卡。 */
  lent: boolean
  createdTime: number
}

/** items 视图条目：未过期且数量大于 0 的道具栈，按道具 ID、期限排序。 */
export interface Item {
  itemId: string
  /** 道具类型（ItemType）。 */
  type: number
  quantity: number
  /** 期限，0 为不过期。 */
  expiryTime: number
}

/** giftBox 视图条目，按发放时间从新到旧。 */
export interface Gift {
  giftId: string
  resourceType: ResourceType | number
  /** 钻石、AP、经验为空串。 */
  resourceId: string
  quantity: number
  message: string
  /** 领取期限，0 为不过期。 */
  limitTime: number
  createdAt: number
  /** 领取时间，未领取为 0。 */
  receivedAt: number
  sourceKind: GiftSourceKind
  /** 来源 ID：指令为 requestId（或指定的 sourceId）、邮件为 mailId、兑换码为码；游戏内为空串。 */
  sourceId: string
  state: GiftState
}

export interface GiftBoxQuery {
  state?: GiftState
  cursor?: string
  /** 默认 50，最大 500。 */
  limit?: number
}

/** 抽到的一张卡。 */
export interface GashaDrawnCard {
  resourceType: ResourceType | number
  resourceId: string
  rarity: Rarity
  isNew: boolean
  isPickup: boolean
  /** 重复时转换成的资源（例如偶像卡碎片），没有转换为 0 / 空串。 */
  providedType: number
  providedId: string
  providedQuantity: number
}

/** gashaHistory 视图条目：一次抽卡请求，从新到旧。 */
export interface GashaDraw {
  id: number
  gashaId: string
  buttonId: string
  drawCount: number
  /** 消耗的资源：1100 钻石、1101 付费钻石、2 道具（抽卡券）。 */
  costResourceType: number
  costResourceId: string
  costQuantity: number
  /** 实际扣除的免费 / 付费钻石。 */
  jewelFree: number
  jewelPaid: number
  ssrCount: number
  pickupCount: number
  cards: GashaDrawnCard[]
  createdAt: number
}

/** 抽卡合计。 */
export interface DrawTotals {
  /** 请求次数。 */
  requests: number
  /** 抽数。 */
  draws: number
  ssr: number
  pickup: number
  jewelFree: number
  jewelPaid: number
}

/** gashaHistory：total 为玩家全部请求数（不随 gashaId 筛选变化），summary 为全部抽卡的合计。 */
export type GashaHistoryPage = ListPage<GashaDraw, DrawTotals>

export interface GashaHistoryQuery {
  gashaId?: string
  cursor?: string
  /** 默认 20，最大 200。 */
  limit?: number
}

/** 记忆（memories 视图条目；produceHistory 只含 produceId 非空的培育结果）。 */
export interface Memory {
  memoryId: string
  idolCardId: string
  characterId: string
  /** 培育 ID，非培育记忆为空串。 */
  produceId: string
  /** 研究活动记忆的活动 ID。 */
  researchId: string
  grade: ResultGrade
  /** 评价值。 */
  score: number
  power: number
  vocal: number
  dance: number
  visual: number
  stamina: number
  /** 是否为出借记忆。 */
  lent: boolean
  protected: boolean
  createdTime: number
}

export interface PvpSlot {
  main: Memory | null
  sub: Memory | null
}

/** 编成的一个阶段：段位允许的槽位与编成力。 */
export interface PvpStage {
  /** 阶段 1–3。 */
  stage: number
  power: number
  slots: PvpSlot[]
}

/** pvp-unit 视图：当前赛季的条目与当前配置的编成。 */
export interface PvpUnit {
  /** 当前赛季 ID，没有赛季时为空串。 */
  seasonId: string
  /** 是否已进入当前赛季。 */
  entered: boolean
  /** 段位 1–8。 */
  grade: number
  bestGrade: number
  rate: number
  playCount: number
  winCount: number
  /** 今日剩余对战次数。 */
  remainingPlays: number
  maxDailyPlays: number
  unitPower: number
  unitUpdatedTime: number
  stages: PvpStage[]
}

/** 保留的一场对战。 */
export interface PvpBattle {
  playTime: number
  seasonId: string
  grade: number
  /** 对战后的 Rate。 */
  rate: number
  /** Rate 增减。 */
  earnedRate: number
  isWin: boolean
  /** 对手 pid；NPC 为空串。 */
  rivalPid: string
  rivalDisplayId: string
  rivalName: string
}

/** pvp 视图。 */
export interface PvpView extends PvpUnit {
  /** 赛季名次，未参赛为 0。 */
  rank: number
  battles: PvpBattle[]
}

/** sessions 视图条目：登录记录，从新到旧。 */
export interface Session {
  id: number
  loginAt: number
  lastSeenAt: number
  /** 玩家在线时最新一条为 true。 */
  live: boolean
  platform: string
  device: string
  remoteAddr: string
}

/** backups 视图条目，从新到旧。 */
export interface Backup {
  id: number
  kind: BackupKind
  note: string
  /** 备份时的存档 revision。 */
  revision: number
  createdAt: number
}

/** 视图 ID → 返回的 data。 */
export interface GakumasViews {
  overview: Overview
  idolCards: ListPage<IdolCard>
  supportCards: ListPage<SupportCard>
  items: ListPage<Item>
  giftBox: ListPage<Gift>
  gashaHistory: GashaHistoryPage
  produceHistory: ListPage<Memory>
  memories: ListPage<Memory>
  pvp: PvpView
  'pvp-unit': PvpUnit
  seminar: ListPage<SeminarProgress>
  sessions: ListPage<Session>
  backups: ListPage<Backup>
}

export type GakumasViewId = keyof GakumasViews

// ---------------------------------------------------------------------------
// 报表
// ---------------------------------------------------------------------------

/** 工作台图表的一天。 */
export interface DaySeries {
  /** YYYY-MM-DD（服务器时区）。 */
  day: string
  /** 当天登录过游戏的账号数。 */
  active: number
  /** 当天抽数。 */
  draws: number
}

/** 卡池表现（gasha 报表条目）：开放中或有过抽卡的卡池。 */
export interface GashaStats {
  gashaId: string
  name: string
  type: GashaType
  bannerAssetId: string
  startTime: number
  /** 0 为不结束。 */
  endTime: number
  open: boolean
  today: DrawTotals
  total: DrawTotals
}

/** 排行中的玩家。 */
export interface RankedPlayer {
  pid: string
  displayId: string
  name: string
  avatarIdolCardId: string
}

/** 竞赛赛季（pvp 报表 summary、工作台 pvp）。 */
export interface PvpSeasonSummary {
  seasonId: string
  prepareStartTime: number
  startTime: number
  endTime: number
  /** 参赛人数。 */
  participants: number
  /** 今日对战场数。 */
  battlesToday: number
}

/** pvp 报表条目：段位高、Rate 高者在前。 */
export interface PvpEntry extends RankedPlayer {
  rank: number
  grade: number
  rate: number
  playCount: number
  winCount: number
  /** 编成力。 */
  power: number
  updatedTime: number
}

/** 没有赛季时 items 为空且没有 summary。 */
export type PvpReport = ListPage<PvpEntry, PvpSeasonSummary>

/** 全服培育列表条目（avatarIdolCardId 总是空串）。 */
export interface ProduceEntry extends RankedPlayer, Memory {
  produceGroupId: string
}

export interface ProduceReportSummary {
  today: number
  total: number
}

export type ProduceReport = ListPage<ProduceEntry, ProduceReportSummary>

export interface ProduceReportQuery {
  produceGroupId?: string
  /** 优先于 produceGroupId。 */
  produceId?: string
  characterId?: string
  /** Unix 毫秒。 */
  since?: number
  until?: number
  /** 搜索玩家。 */
  q?: string
  cursor?: string
  limit?: number
}

/** seminar 报表条目。 */
export interface SeminarGroupStats {
  groupId: string
  name: string
  order: number
  /** 试题数。 */
  exams: number
  /** 全部试题的钻石奖励合计。 */
  jewelReward: number
  /** 全部通关的玩家数。 */
  clearedPlayers: number
  /** 玩家总数。 */
  players: number
}

/** tower 报表条目：当前排行季到达的最高层。 */
export interface TowerEntry extends RankedPlayer {
  rank: number
  layer: number
  reachedAt: number
}

export interface TowerReportSummary {
  towerId: string
  /** 排行季开始时间。 */
  seasonStart: number
}

/** 首个排行季之前 items 为空且没有 summary。 */
export type TowerReport = ListPage<TowerEntry, TowerReportSummary>

export interface TowerReportQuery {
  /** 必填。 */
  towerId: string
  q?: string
  cursor?: string
  limit?: number
}

/** gifts 报表：某来源礼物的发放与领取统计。 */
export interface GiftStatsReport {
  /** 请求中的 sourceKind 原样回显（可能为空串，表示 gm）。 */
  sourceKind: GiftSourceName | ''
  sourceId: string
  gifts: number
  receivedGifts: number
  players: number
  receivedPlayers: number
}

export interface GiftStatsQuery {
  sourceKind?: GiftSourceName
  /** 必填。 */
  sourceId: string
}

/** dashboard 报表。 */
export interface DashboardReport {
  asOf: number
  online: number
  onlinePeak: number
  onlinePeakAt: number
  activeToday: number
  activeYesterday: number
  newAccountsToday: number
  newAccountPids: string[]
  registered: number
  drawsToday: DrawTotals
  drawsYesterday: DrawTotals
  /** days 天（默认 7，最大 60）的每日数据，最后一项为今天。 */
  series: DaySeries[]
  gashas: GashaStats[]
  /** 没有赛季时为 null。 */
  pvp: PvpSeasonSummary | null
  pvpTop: PvpEntry | null
  producesToday: number
  producesTotal: number
  bestProduceToday: ProduceEntry | null
  /** 今日研修试题通关次数。 */
  seminarClearsToday: number
  seminar: SeminarGroupStats[]
  uptimeSeconds: number
  version: string
  maintenanceEnabled: boolean
  /** 存档队列中等待的数量。 */
  saveWaiting: number
}

/** 报表 ID → 返回的 data。 */
export interface GakumasReports {
  dashboard: DashboardReport
  gasha: GashaStats[]
  pvp: PvpReport
  produce: ProduceReport
  seminar: SeminarGroupStats[]
  tower: TowerReport
  gifts: GiftStatsReport
}

export type GakumasReportId = keyof GakumasReports

// ---------------------------------------------------------------------------
// 指令参数与结果
// ---------------------------------------------------------------------------

/** 一项奖励参数（x-picker: reward）。 */
export interface RewardParam {
  resourceType: ResourceType | number
  /** 钻石、AP、经验为空串或省略。 */
  resourceId?: string
  quantity: number
}

/** gift.send 参数。 */
export interface GiftSendParams {
  /** 1–20 项。 */
  rewards: RewardParam[]
  /** 最多 64 字，缺省「GMプレゼント」。 */
  message?: string
  /** 0–365，0 不过期。 */
  expireDays?: number
  sourceKind?: GiftSourceName
  /** gm 缺省为 requestId；mail、code 必填。 */
  sourceId?: string
}

/** gift.send 试运行时每项奖励的预览。 */
export interface GiftPreview {
  resourceType: number
  resourceId: string
  quantity: number
  /** 是否已持有（偶像卡皮肤、支援卡）。 */
  owned: boolean
  /** 已持有的偶像卡皮肤领取时转换成的碎片道具。 */
  convertsTo?: string
  convertedQuantity?: number
}

/** gift.send 试运行的 detail。 */
export interface GiftSendPreviewDetail {
  /** 礼物箱可领取件数（发放前 / 后）。 */
  giftBoxBefore: number
  giftBoxAfter: number
  rewards: GiftPreview[]
}

/** gift.send 实际执行的 detail。 */
export interface GiftSendDetail {
  gifts: Gift[]
  sourceId: string
}

/** gift.revoke 参数：sourceId 与 giftIds 至少给一个。 */
export interface GiftRevokeParams {
  sourceKind: GiftSourceName
  sourceId?: string
  giftIds?: string[]
}

/** gift.revoke 的 detail：移除（试运行时为将移除）的礼物。 */
export interface GiftRevokeDetail {
  gifts: Gift[]
}

/** mail.broadcast 的定向条件（Unix 毫秒，0 或省略为不限）。 */
export interface MailCondition {
  createdSince?: number
  createdBefore?: number
  lastLoginSince?: number
  lastLoginBefore?: number
}

/** mail.broadcast 参数；奖励不能是付费钻石（1101）。 */
export interface MailBroadcastParams {
  mailId: string
  target: 'all' | 'condition' | 'players'
  players?: string[]
  condition?: MailCondition
  excludeBanned?: boolean
  rewards: RewardParam[]
  message?: string
  expireDays?: number
}

/** mail.broadcast / mail.revoke 的 summary；试运行只有 matched。 */
export interface MailSummary {
  mailId: string
  matched: number
  /** 投递（撤回）的玩家数。 */
  delivered: number
  gifts: number
  failed: number
  /** 最多 50 个。 */
  failedPids: string[]
}

/** idolcard.grantAll 的 detail。 */
export interface IdolCardGrantAllDetail {
  skinGifts: number
  pieceGifts: number
  pieces: number
}

/** supportcard.grantAll 的 detail。 */
export interface SupportCardGrantAllDetail {
  gifts: number
  copies: number
}

/** jewel.adjust 参数：增减量，至少一个非 0。 */
export interface JewelAdjustParams {
  free?: number
  paid?: number
}

/** item.adjust 参数。 */
export interface ItemAdjustParams {
  itemId: string
  quantity: number
}

/** session.kick 的 detail。 */
export interface SessionKickDetail {
  online: boolean
  sessionEnded: boolean
}

/** account.ban 参数：until 与 permanent 二选一。 */
export interface AccountBanParams {
  until?: number
  permanent?: boolean
  /** 缺省取指令的 reason。 */
  reason?: string
}

/** account.rename 的 detail；失败时没有 after。 */
export interface AccountRenameDetail {
  before: string
  after?: string
}

/** seminar.clear 的 detail：本次通关的试题，没有时为 null。 */
export interface SeminarClearDetail {
  cleared: string[] | null
}

/** seminar.reset 的 detail：移除的通关记录数。 */
export interface SeminarResetDetail {
  removed: number
}

/** archive.export 导出的存档（字段名为服务端 Go 字段名，没有 json 标签）。 */
export interface PlayerExport {
  Basic: {
    AccountID: number
    Name: string
    BornMonth: number
    BornDay: number
    SupportCardID: string
    Comment: string
    Exp: number
    TotalFanCount: number
    /** RFC 3339 时间字符串，从未登录为 null。 */
    LastLoginAt: string | null
    UpdatedAt: string
    [key: string]: unknown
  }
  Currency: {
    AccountID: number
    Free: number
    Paid: number
    WebStorePaid: number
  }
  Memories: Record<string, Record<string, unknown>>
  Gifts: Record<string, Record<string, unknown>>
  /** 试题 / 研修组 ID → 通关时间。 */
  SeminarClears: Record<string, number>
  /** 完整存档数据（结构随游戏版本变化）。 */
  Data: Record<string, unknown>
}

/** archive.export 的 detail。 */
export interface ArchiveExportDetail {
  export: PlayerExport
}

/** archive.restore 实际执行的 detail；试运行的 detail 为来源备份（Backup）。 */
export interface ArchiveRestoreDetail {
  restoredBackupId: number
  /** 回档前自动保存的备份。 */
  keptBackup: Backup
}

/** server.maintenance 参数。 */
export interface MaintenanceParams {
  enabled: boolean
  whitelist?: string[]
}

/** 指令 ID → { params, detail（玩家指令）或 summary（服务器指令）}。detail 为试运行与实际执行的联合。 */
export interface GakumasCommands {
  'gift.send': { params: GiftSendParams; detail: GiftSendPreviewDetail | GiftSendDetail }
  'gift.revoke': { params: GiftRevokeParams; detail: GiftRevokeDetail }
  'mail.broadcast': { params: MailBroadcastParams; summary: MailSummary }
  'mail.revoke': { params: { mailId: string }; summary: MailSummary }
  'idolcard.grantAll': { params: { message?: string }; detail: IdolCardGrantAllDetail }
  'supportcard.grantAll': { params: { message?: string }; detail: SupportCardGrantAllDetail }
  'jewel.adjust': { params: JewelAdjustParams; detail: never }
  'item.adjust': { params: ItemAdjustParams; detail: never }
  'ap.refill': { params: Record<string, never>; detail: never }
  'producer.exp': { params: { exp: number }; detail: never }
  'session.kick': { params: Record<string, never>; detail: SessionKickDetail }
  'account.ban': { params: AccountBanParams; detail: never }
  'account.unban': { params: Record<string, never>; detail: never }
  'account.password': { params: { password: string }; detail: never }
  'account.rename': { params: { name: string }; detail: AccountRenameDetail }
  'pvp.setRate': { params: { rate: number }; detail: never }
  'pvp.resetDaily': { params: Record<string, never>; detail: never }
  'seminar.clear': { params: Record<string, never>; detail: SeminarClearDetail }
  'seminar.reset': { params: Record<string, never>; detail: SeminarResetDetail }
  'archive.export': { params: Record<string, never>; detail: ArchiveExportDetail }
  'archive.backup': { params: { note?: string }; detail: Backup }
  'archive.restore': { params: { backupId: number }; detail: Backup | ArchiveRestoreDetail }
  'server.maintenance': { params: MaintenanceParams; summary: MaintenanceStatus }
}

export type GakumasCommandId = keyof GakumasCommands

// ---------------------------------------------------------------------------
// 事件 data
// ---------------------------------------------------------------------------

/** session.login 的 data。 */
export interface SessionLoginEventData {
  platform: string
  device: string
}

/** gasha.draw 的 data。 */
export interface GashaDrawEventData {
  gashaId: string
  buttonId: string
  drawCount: number
  ssrCount: number
  pickupCount: number
  cards: GashaDrawnCard[]
}

/** produce.finish 的 data。 */
export interface ProduceFinishEventData {
  produceId: string
  idolCardId: string
  grade: ResultGrade
  score: number
  cleared: boolean
  /** 是否保留了培育结果记忆。 */
  memoryKept: boolean
}

// ---------------------------------------------------------------------------
// 配置域：data/Gasha.json（gdconf.Gasha）
// ---------------------------------------------------------------------------

/** 手写数据中的奖励（protobuf common.Reward，零值字段可省略）。 */
export interface ConfigReward {
  resourceType?: ResourceType | number
  resourceId?: string
  quantity?: number
}

/** 卡组：一种资源类型（4 支援卡 / 5 偶像卡皮肤）与一种稀有度的卡。 */
export interface GashaPool {
  id: string
  resourceType: 4 | 5
  rarity: Rarity
  /** 卡 ID，不可重复，稀有度必须与 rarity 一致。 */
  ids: string[]
}

/** 抽选槽位：几个卡组的并集与它们共享的概率。 */
export interface GashaSlot {
  poolIds: string[]
  /** 十万分率；同一抽选表的槽位合计 100000。 */
  probability: number
}

/** UP 卡：从同资源类型同稀有度的槽位中拿出的独立概率。 */
export interface GashaPickup {
  resourceType: 4 | 5
  resourceId: string
  /** 十万分率。 */
  probability: number
}

/** 抽选表。 */
export interface GashaLottery {
  id: string
  slots: GashaSlot[]
  pickups: GashaPickup[]
}

/** 卡池的按钮（GashaButton 主数据 ID）；空的抽选表 ID 使用卡池的。 */
export interface GashaButtonRef {
  id: string
  lotteryId: string
  fixLotteryId: string
  /** 设置时替代 GashaButton 的 limitCount（限次按钮）。 */
  limitCount?: number
}

/** 阶梯卡池的一步（阶梯自选 UP 卡池另带本步的候选）。 */
export interface GashaStep {
  number: number
  gashaButtonId: string
  stepLabelColor: string
  appealAssetId: string
  isFeature: boolean
  bonusRewards: ConfigReward[]
  lotteryId: string
  fixLotteryId: string
  pickupCount: number
  idolCardSkinIds: string[]
  supportCardIds: string[]
  idolCardProbability: number
  supportCardProbability: number
}

export interface GashaStepUp {
  /** 最后一步之后卡池结束；否则循环。 */
  isLimited: boolean
  steps: GashaStep[]
}

export interface GashaStepUpSelectPickup {
  isLimited: boolean
  /** 各步共用一次自选。 */
  isCommon: boolean
  steps: GashaStep[]
}

/** 自选 UP：玩家从候选中选 pickupCount 张，选中的卡按其资源类型的概率抽出。 */
export interface GashaSelectPickup {
  idolCardSkinIds: string[]
  supportCardIds: string[]
  pickupCount: number
  idolCardProbability: number
  supportCardProbability: number
}

/** 连抽卡池下一次的抽数权重；count 0 结束本轮。 */
export interface GashaDrawRewardCount {
  count: number
  weight: number
}

/** 连抽（引き放題）卡池。 */
export interface GashaContinuous {
  firstDrawCount: number
  drawRewardCounts: GashaDrawRewardCount[]
  maxContinueCount: number
  /** 一轮没有 SSR 时最后一张的保底抽选表。 */
  guaranteeLotteryId: string
  advAssetId: string
  sharedRewards: ConfigReward[]
  sharedContinuedCount: number
  sharedDrawRewardCount: number
}

/** 交换点数可兑换的一项。 */
export interface GashaPointExchange {
  /** 从 1 起连续编号。 */
  number: number
  requiredQuantity: number
  rewards: ConfigReward[]
  startTime: number
  /** 0 为不结束。 */
  endTime: number
}

/** 交换点数：每抽一张加一点，endTime 后按每点 expireReward 折算。 */
export interface GashaPoint {
  id: string
  name: string
  conditionSetId: string
  startTime: number
  endTime: number
  expireReward: ConfigReward
  exchanges: GashaPointExchange[]
}

/** 卡池背景 UP 演出（protobuf common.GashaBackgroundPickup，零值字段可省略）。 */
export interface GashaBackgroundPickup {
  idolCardSkinId?: string
  supportCardId?: string
  produceCardSkinId?: string
  movieAssetId?: string
  screenAssetId?: string
  rewards?: ConfigReward[]
}

/** 一个卡池。 */
export interface GashaDef {
  id: string
  type: GashaType
  name: string
  description: string
  bannerAssetId: string
  bannerLabelAssetId: string
  movieAssetId: string
  screenAssetId: string
  appealTextAssetId: string
  animationBackgroundAssetId: string
  bgmAssetId: string
  viewConditionSetId: string
  unlockConditionSetId: string
  startTime: number
  /** 0 为常驻。 */
  endTime: number
  /** 显示顺序，小者在前。 */
  order: number
  lotteryId: string
  fixLotteryId: string
  buttons: GashaButtonRef[]
  pickupIdolCardSkinIds: string[]
  pickupSupportCardIds: string[]
  backgroundPickupIdolCardSkins: GashaBackgroundPickup[]
  backgroundPickupSupportCards: GashaBackgroundPickup[]
  /** 交换点数 ID，没有为空串。 */
  gashaPointId: string
  bonusRewards: ConfigReward[]
  stepUp: GashaStepUp | null
  selectPickup: GashaSelectPickup | null
  stepUpSelectPickup: GashaStepUpSelectPickup | null
  continuous: GashaContinuous | null
}

/** gasha 配置域内容（data/Gasha.json）。 */
export interface GashaConfig {
  pools: GashaPool[]
  lotteries: GashaLottery[]
  points: GashaPoint[]
  gashas: GashaDef[]
}

// ---------------------------------------------------------------------------
// 配置域：data/Notice.json（gdconf.Notice）
// ---------------------------------------------------------------------------

/** 手写公告；卡池公告由服务端按卡池自动生成（ID 为 notice-<gashaId>），不在此文件中。 */
export interface NoticeEntry {
  id: string
  title: string
  category: NoticeCategory
  type: NoticeType
  startTime: number
  /** 0 为不结束，否则必须晚于 startTime。 */
  endTime: number
  /** 不早于 startTime。 */
  lastUpdateTime: number
  bannerAssetId: string
  /** 详情页 URL。 */
  detailUrl: string
}

/** notice 配置域内容（data/Notice.json）。 */
export interface NoticeConfig {
  /** 每页条数，必须为正。 */
  pageSize: number
  notices: NoticeEntry[]
}
