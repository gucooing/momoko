/**
 * 学マス（gakumas）的前端类型：协议枚举值、API 类型的别名（src/types/v1/gakumas，由 api/proto/v1/gakumas 生成），
 * 以及可表单编辑的配置文档（data/Gasha.json、data/Notice.json）的结构。时间均为 Unix 毫秒，pid 为十进制字符串。
 */

// ---------------------------------------------------------------------------
// 枚举值（协议数值）
// ---------------------------------------------------------------------------

/**
 * 奖励资源类型（协议 ResourceType）：2 道具、4 支援卡、5 偶像卡皮肤、8 AP、9 制作人经验、
 * 12 乐曲、13 发型、14 服装、1100 钻石（免费）、1101 付费钻石。
 */
export type ResourceType = 2 | 4 | 5 | 8 | 9 | 12 | 13 | 14 | 1100 | 1101

/** 礼物来源：0 游戏内、1 管理操作、2 邮件、3 兑换码。 */
export type GiftSourceKind = 0 | 1 | 2 | 3

/** 礼物状态。 */
export type GiftState = 'unreceived' | 'received' | 'expired'

/** 礼物来源名。 */
export type GiftSourceName = 'gm' | 'mail' | 'code'

/** 存档备份种类：1 手动、2 操作前自动、3 回档前自动。 */
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
// API 类型
// ---------------------------------------------------------------------------

export type {
  Player,
  PlayerCounts,
  Overview,
  ProduceRun,
  GiftSummary,
  SeminarProgress,
  LoginCounts,
  Session,
  IdolCard,
  SupportCard,
  Item,
  Gift,
  GashaDrawCard,
  GashaDraw,
  DrawTotals,
  Memory,
  PvpSlot,
  PvpStage,
  PvpBattle,
  PlayerPvp,
  Backup,
  Change,
  GiftPreview,
  GiftSendResult,
  GiftRevokeResult,
  DaySeries,
  GashaStat,
  PvpSeason,
  PvpEntry,
  ProduceResult,
  SeminarGroupStat,
  TowerRank,
  Dashboard,
  Manifest,
  Modes,
  Status,
  Maintenance,
  CurrencyEntry,
  ItemEntry,
  IdolCardSkinEntry,
  SupportCardEntry,
  CharacterEntry,
  CostumeEntry,
  MusicEntry,
  GashaEntry,
  ProduceGroupEntry,
  ProduceEntry,
  SeminarEntry,
  TowerEntry,
  SessionPage,
  IdolCardPage,
  SupportCardPage,
  ItemPage,
  GiftPage,
  GashaDrawPage,
  MemoryPage,
  PlayerSeminar,
  BackupList,
  GashaStatList,
  ProducePage,
  PvpRankingPage,
  SeminarStatList,
  TowerRankingPage,
  ProduceCounts,
} from '@/types/v1/gakumas/types'


/** 玩家详情：玩家条目加概览与登录次数。 */
export type PlayerDetail = import('@/types/v1/gakumas/types').Player & {
  overview?: import('@/types/v1/gakumas/types').Overview
  logins?: import('@/types/v1/gakumas/types').LoginCounts
}

/** 玩家列表筛选（unlinked 为未绑定通行证）。 */
export type PlayerFilter = 'all' | 'online' | 'activeToday' | 'banned' | 'unlinked'

/** 礼物箱里的一件奖励（兑换码、邮件、发礼物共用）。 */
export interface RewardParam {
  resourceType: number
  resourceId: string
  quantity: number
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
