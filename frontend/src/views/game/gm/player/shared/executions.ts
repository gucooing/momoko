// GM 记录查询参数（ListGmExecutionsRequest 的字段都要给，空值即不筛选）。
import type { ListGmExecutionsRequest } from '@/types/v1/game_gm'

export const gmExecutionsQuery = (serverId: string, q: Partial<ListGmExecutionsRequest> = {}): ListGmExecutionsRequest => ({
  serverId,
  actorKind: '',
  status: '',
  risk: '',
  command: '',
  pid: '',
  q: '',
  since: undefined,
  until: undefined,
  includeDryRun: false,
  page: 1,
  pageSize: 20,
  actorId: '',
  ...q,
})

/** GM 记录状态 → StatusPill 样式。 */
export const executionVariant = (status: string) => {
  switch (status) {
    case 'success':
      return 'success'
    case 'partial':
      return 'warning'
    case 'failed':
    case 'rejected':
      return 'error'
    case 'pending':
    case 'running':
      return 'primary'
    default:
      return 'neutral'
  }
}

/** 操作 → 图标（列表行内按钮与批量条用；没有图标的操作只显示标题）。 */
export const OPERATION_ICONS: Record<string, string> = {
  'gift.send': 'HOutline:GiftIcon',
  'player.kick': 'HOutline:ArrowLeftStartOnRectangleIcon',
  'player.ban': 'HOutline:NoSymbolIcon',
  'player.unban': 'HOutline:CheckCircleIcon',
  'archive.export': 'HOutline:ArchiveBoxArrowDownIcon',
}
