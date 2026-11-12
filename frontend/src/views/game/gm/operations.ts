// 学マス 的 GM 操作（管理端）：菜单信息、参数表单与对应的接口（src/api/gakumas.ts）。风险与目标与服务端的操作权限一致。
import * as api from '@/api/gakumas'
import type { GmExecution } from '@/types/v1/game_gm'
import type { OperationTarget } from '@/types/v1/gakumas/gm'
import type { Change, JSONSchema, OperationDef, OperationOutcome, OperationOptions, OperationRow } from '@/games/mgp'

const object = (properties: Record<string, JSONSchema> = {}, required: string[] = []): JSONSchema => ({
  type: 'object',
  properties,
  required,
  additionalProperties: false,
})

const messageField: JSONSchema = { type: 'string', title: '礼物箱文案', maxLength: 64 }

/** 把一个操作的响应转为结果：逐个对象的状态，data 中的 changes 单列，其余为 detail（detail 返回 undefined 时不显示）。 */
export const outcome = (
  execution: GmExecution | undefined,
  results: { target?: OperationTarget; data?: object }[],
  dryRun: boolean,
  detail: (data: object) => unknown = (data) => data,
): OperationOutcome => ({
  executionId: execution?.id || '',
  status: execution?.status || '',
  errorCode: execution?.errorCode,
  errorMessage: execution?.errorMessage,
  dryRun,
  backupRef: execution?.backupRef || undefined,
  results: results.map((r): OperationRow => {
    const row: OperationRow = {
      pid: r.target?.pid || '',
      name: r.target?.name || '',
      status: r.target?.status || '',
      code: r.target?.code,
      message: r.target?.message,
    }
    if (r.data) {
      if ('changes' in r.data) row.changes = (r.data as { changes: Change[] }).changes
      else {
        const d = detail(r.data)
        if (d !== undefined) row.detail = d
      }
    }
    return row
  }),
})

const str = (v: unknown) => (typeof v === 'string' ? v.trim() : '')
const num = (v: unknown) => Number(v) || 0
const one = (pids: string[]) => pids[0] || ''

/** 导出的存档：detail.export 为解析后的 JSON（提供下载）。 */
const exportDetail = (data: object) => {
  const raw = (data as { archive?: string }).archive || ''
  try {
    return { export: JSON.parse(raw) }
  } catch {
    return { export: raw }
  }
}

export const gakumasOperations: OperationDef[] = [
  {
    id: 'gift.send',
    title: '发送到礼物箱',
    group: '发放',
    risk: 'medium',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(
      {
        rewards: { type: 'array', title: '奖励', 'x-picker': 'reward', minItems: 1, maxItems: 20 },
        message: messageField,
        expireDays: { type: 'integer', title: '领取期限（天，0 不过期）', minimum: 0, maximum: 365 },
      },
      ['rewards'],
    ),
    run: async (serverId, pids, p, opt) => {
      const r = await api.sendGift({
        serverId,
        pids,
        rewards: (p.rewards as { resourceType: number; resourceId: string; quantity: number }[]) || [],
        message: str(p.message),
        expireDays: num(p.expireDays),
        dryRun: opt.dryRun,
        summary: opt.summary,
      })
      return outcome(r.execution, r.results, opt.dryRun, () => undefined)
    },
  },
  {
    id: 'gift.revoke',
    title: '撤回未领取的礼物',
    group: '发放',
    risk: 'medium',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(
      {
        sourceKind: { type: 'string', title: '来源', enum: ['gm', 'mail', 'code'] },
        sourceId: { type: 'string', title: '来源 ID' },
        giftIds: { type: 'array', title: '礼物 ID', items: { type: 'string', title: '礼物 ID' } },
      },
      ['sourceKind'],
    ),
    run: async (serverId, pids, p, opt) => {
      const r = await api.revokeGifts({
        serverId,
        pids,
        sourceKind: str(p.sourceKind),
        sourceId: str(p.sourceId),
        giftIds: (p.giftIds as string[]) || [],
        dryRun: opt.dryRun,
      })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'idolCard.grantAll',
    title: '全部偶像卡（含满潜能碎片）',
    group: '发放',
    risk: 'danger',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object({ message: messageField }),
    run: async (serverId, pids, p, opt) => {
      const r = await api.grantAllIdolCards({
        serverId,
        pids,
        message: str(p.message),
        dryRun: opt.dryRun,
      })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'supportCard.grantAll',
    title: '全部支援卡（满突破）',
    group: '发放',
    risk: 'danger',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object({ message: messageField }),
    run: async (serverId, pids, p, opt) => {
      const r = await api.grantAllSupportCards({
        serverId,
        pids,
        message: str(p.message),
        dryRun: opt.dryRun,
      })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'jewel.adjust',
    title: '调整钻石（免费 / 付费）',
    group: '资源',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object({
      free: { type: 'integer', title: '免费钻石增减' },
      paid: { type: 'integer', title: '付费钻石增减' },
    }),
    run: async (serverId, pids, p, opt) => {
      const r = await api.adjustJewel({ serverId, pids, free: num(p.free), paid: num(p.paid), dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'item.adjust',
    title: '调整金币与道具',
    group: '资源',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(
      {
        itemId: { type: 'string', title: '道具', 'x-picker': 'item' },
        quantity: { type: 'integer', title: '数量增减' },
      },
      ['itemId', 'quantity'],
    ),
    run: async (serverId, pids, p, opt) => {
      const r = await api.adjustItem({ serverId, pids, itemId: str(p.itemId), quantity: num(p.quantity), dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'actionPoint.refill',
    title: '回满体力',
    group: '资源',
    risk: 'low',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(),
    run: async (serverId, pids, _p, opt) => {
      const r = await api.refillActionPoint({ serverId, pids, dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'exp.add',
    title: '制作人经验 / 等级',
    group: '养成',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object({ exp: { type: 'integer', title: '增加的经验', minimum: 1 } }, ['exp']),
    run: async (serverId, pids, p, opt) => {
      const r = await api.addExp({ serverId, pids, exp: num(p.exp), dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'player.kick',
    title: '强制下线',
    group: '账号',
    risk: 'low',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(),
    run: async (serverId, pids, _p, opt) => {
      const r = await api.kickPlayers({ serverId, pids, dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'player.ban',
    title: '封禁',
    group: '账号',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object({
      until: { type: 'integer', title: '封禁到（时间）' },
      permanent: { type: 'boolean', title: '永久封禁' },
      reason: { type: 'string', title: '封禁原因', maxLength: 256 },
    }),
    run: async (serverId, pids, p, opt) => {
      const r = await api.banPlayers({
        serverId,
        pids,
        until: num(p.until),
        permanent: !!p.permanent,
        reason: str(p.reason),
        dryRun: opt.dryRun,
      })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'player.unban',
    title: '解封',
    group: '账号',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(),
    run: async (serverId, pids, _p, opt) => {
      const r = await api.unbanPlayers({ serverId, pids, dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'player.password',
    title: '重置登录密码',
    group: '账号',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: false,
    params: object({ password: { type: 'string', title: '新密码', format: 'password' } }, ['password']),
    run: async (serverId, pids, p) => {
      const r = await api.resetPassword({ serverId, pids, password: String(p.password ?? '') })
      return outcome(r.execution, r.results, false)
    },
  },
  {
    id: 'player.rename',
    title: '改昵称',
    group: '账号',
    risk: 'low',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object({ name: { type: 'string', title: '新昵称' } }, ['name']),
    run: async (serverId, pids, p, opt) => {
      const r = await api.renamePlayer({ serverId, pids, name: str(p.name), dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'supportCard.setRental',
    title: '设为援助支援卡',
    group: '账号',
    risk: 'low',
    target: 'player',
    selfService: false,
    dryRun: true,
    params: object({ supportCardId: { type: 'string', title: '支援卡 ID' } }, ['supportCardId']),
    run: async (serverId, pids, p, opt) => {
      const r = await api.setRentalSupportCard({ serverId, pid: one(pids), supportCardId: str(p.supportCardId), dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'memory.setRental',
    title: '设为援助回忆',
    group: '账号',
    risk: 'low',
    target: 'player',
    selfService: false,
    dryRun: true,
    params: object({ memoryId: { type: 'string', title: '回忆 ID' } }, ['memoryId']),
    run: async (serverId, pids, p, opt) => {
      const r = await api.setRentalMemory({ serverId, pid: one(pids), memoryId: str(p.memoryId), dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'pvp.setRate',
    title: '竞赛：调整 Rate',
    group: '玩法',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object({ rate: { type: 'integer', title: 'Rate', minimum: 0 } }, ['rate']),
    run: async (serverId, pids, p, opt) => {
      const r = await api.setPvpRate({ serverId, pids, rate: num(p.rate), dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'pvp.resetDaily',
    title: '竞赛：重置今日次数',
    group: '玩法',
    risk: 'high',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(),
    run: async (serverId, pids, _p, opt) => {
      const r = await api.resetPvpDaily({ serverId, pids, dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'seminar.clear',
    title: '研修：标记全部通关并发放奖励',
    group: '玩法',
    risk: 'medium',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(),
    run: async (serverId, pids, _p, opt) => {
      const r = await api.clearSeminar({ serverId, pids, dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'seminar.reset',
    title: '研修：重置进度',
    group: '玩法',
    risk: 'medium',
    target: 'players',
    selfService: true,
    dryRun: true,
    params: object(),
    run: async (serverId, pids, _p, opt) => {
      const r = await api.resetSeminar({ serverId, pids, dryRun: opt.dryRun })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'archive.export',
    title: '导出存档（JSON）',
    group: '存档',
    risk: 'medium',
    target: 'players',
    selfService: true,
    dryRun: false,
    params: object(),
    run: async (serverId, pids) => {
      const r = await api.exportArchive({ serverId, pids, dryRun: false })
      return outcome(r.execution, r.results, false, exportDetail)
    },
  },
  {
    id: 'archive.backup',
    title: '备份当前存档',
    group: '存档',
    risk: 'low',
    target: 'player',
    selfService: false,
    dryRun: false,
    params: object({ note: { type: 'string', title: '备注', maxLength: 256 } }),
    run: async (serverId, pids, p) => {
      const r = await api.backupArchive({ serverId, pid: one(pids), note: str(p.note) })
      return outcome(r.execution, r.results, false)
    },
  },
  {
    id: 'archive.restore',
    title: '回档到备份',
    group: '存档',
    risk: 'danger',
    target: 'player',
    selfService: false,
    dryRun: true,
    params: object({ backupId: { type: 'integer', title: '备份', 'x-picker': 'backup' } }, ['backupId']),
    run: async (serverId, pids, p, opt) => {
      const r = await api.restoreArchive({
        serverId,
        pid: one(pids),
        backupId: num(p.backupId),
        dryRun: opt.dryRun,
      })
      return outcome(r.execution, r.results, opt.dryRun)
    },
  },
  {
    id: 'maintenance.set',
    title: '维护模式 / 登录白名单',
    group: '运营',
    risk: 'high',
    target: 'server',
    selfService: false,
    dryRun: true,
    params: object(
      {
        enabled: { type: 'boolean', title: '开启维护' },
        whitelist: { type: 'array', title: '维护期间可登录的玩家', items: { type: 'string', title: '玩家 ID' } },
      },
      ['enabled'],
    ),
    run: async (serverId, _pids, p, opt) => {
      const r = await api.setMaintenance({
        serverId,
        enabled: !!p.enabled,
        whitelist: (p.whitelist as string[]) || [],
        dryRun: opt.dryRun,
      })
      return { ...outcome(r.execution, [], opt.dryRun), summary: r.data }
    },
  },
]

/** 只在玩家详情页「更多」里执行的操作（存档备份与回档）：GM 操作页不列出。 */
export const PLAYER_PAGE_OPERATIONS = ['archive.backup', 'archive.restore']

/** 在支援卡、回忆详情里对具体的一张卡或一条回忆执行的操作：不放进操作菜单，GM 操作页也不列出。 */
export const DETAIL_OPERATIONS = ['supportCard.setRental', 'memory.setRental']

/** 玩家列表勾选后在浮动工具条里直接执行的操作；其余操作带着勾选的玩家去 GM 操作页。 */
export const PLAYER_LIST_BATCH = ['gift.send', 'player.kick', 'player.ban', 'player.unban']

/** 按 id 查找操作。 */
export const gakumasOperation = (id: string) => gakumasOperations.find((op) => op.id === id)

const parseData = (raw: string): object | undefined => {
  if (!raw) return undefined
  try {
    const v: unknown = JSON.parse(raw)
    return v && typeof v === 'object' ? v : undefined
  } catch {
    return undefined
  }
}

/** GM 记录详情 → 结果（逐个对象的 data 为游戏服务端结果的 JSON 文本；服务器操作的结果在 summary）。 */
export const executionOutcome = (e: GmExecution): OperationOutcome => {
  const detail = e.command === 'archive.export' ? exportDetail : undefined
  if (gakumasOperation(e.command)?.target === 'server') {
    return { ...outcome(e, [], e.dryRun), summary: parseData(e.results[0]?.data || '') }
  }
  return outcome(
    e,
    e.results.map((r) => ({
      target: { pid: r.pid, name: r.name, status: r.status, code: r.code, message: r.message },
      data: parseData(r.data),
    })),
    e.dryRun,
    detail,
  )
}

export type { OperationOptions }
