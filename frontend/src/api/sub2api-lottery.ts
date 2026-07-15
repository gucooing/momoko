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

export const distributeLotteryRound = (data: DistributeLotteryRoundRequest) =>
  request.post<DistributeLotteryRoundResponse>(`/sub2api/lottery/rounds/${data.id}/distribute`, data)

export const triggerLotterySettle = (data: TriggerLotterySettleRequest = { date: '' }) =>
  request.post<TriggerLotterySettleResponse>('/sub2api/lottery/settle', data)

export const triggerLotteryDraw = (data: TriggerLotteryDrawRequest = { date: '' }) =>
  request.post<TriggerLotteryDrawResponse>('/sub2api/lottery/draw', data)
