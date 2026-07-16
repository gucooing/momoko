import request from '@/utils/request'
import type {
  GetLotteryOverviewRequest,
  GetLotteryOverviewResponse,
  UpdateLotterySettingsRequest,
  UpdateLotterySettingsResponse,
  ListLotteryRoundsRequest,
  ListLotteryRoundsResponse,
  GetLotteryRoundDetailRequest,
  GetLotteryRoundDetailResponse,
  ListLotteryRegistrantsRequest,
  ListLotteryRegistrantsResponse,
  GetSub2APIUserRequest,
  GetSub2APIUserResponse,
  DistributeLotteryRoundRequest,
  DistributeLotteryRoundResponse,
  TriggerLotterySettleRequest,
  TriggerLotterySettleResponse,
  TriggerLotteryDrawRequest,
  TriggerLotteryDrawResponse,
} from '@/types/v1/sub2api'

/** 管理端：抽奖三段概览 + 配置 */
export const getLotteryOverview = (params: GetLotteryOverviewRequest = {}) =>
  request.get<GetLotteryOverviewResponse>('/sub2api/lottery/overview', { params })

export const updateLotterySettings = (data: UpdateLotterySettingsRequest) =>
  request.put<UpdateLotterySettingsResponse>('/sub2api/lottery/settings', data)

export const listLotteryRounds = (params: ListLotteryRoundsRequest) =>
  request.get<ListLotteryRoundsResponse>('/sub2api/lottery/rounds', { params })

export const getLotteryRoundDetail = (params: GetLotteryRoundDetailRequest) =>
  request.get<GetLotteryRoundDetailResponse>(`/sub2api/lottery/rounds/${params.id}`)

/** 管理端：某轮报名者名单（仅本地快照：用户id/用户名/消费额） */
export const listLotteryRegistrants = (params: ListLotteryRegistrantsRequest) =>
  request.get<ListLotteryRegistrantsResponse>(`/sub2api/lottery/rounds/${params.id}/registrants`)

/** 管理端：单个 Sub2API 用户详情（点击报名者时实时拉取） */
export const getSub2APIUser = (params: GetSub2APIUserRequest) =>
  request.get<GetSub2APIUserResponse>(`/sub2api/users/${params.userId}`)

export const distributeLotteryRound = (data: DistributeLotteryRoundRequest) =>
  request.post<DistributeLotteryRoundResponse>(`/sub2api/lottery/rounds/${data.id}/distribute`, data)

export const triggerLotterySettle = (data: TriggerLotterySettleRequest = { date: '' }) =>
  request.post<TriggerLotterySettleResponse>('/sub2api/lottery/settle', data)

export const triggerLotteryDraw = (data: TriggerLotteryDrawRequest = { date: '' }) =>
  request.post<TriggerLotteryDrawResponse>('/sub2api/lottery/draw', data)
