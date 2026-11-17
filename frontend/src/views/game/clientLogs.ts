// 游戏客户端错误日志的共用逻辑：「客户端日志」页（全部服务器）与 GM 玩家详情的「客户端日志」用同一套列表、查看器与操作。
import dayjs from 'dayjs'
import { getGmClientLog } from '@/api/gameGm'
import type { GmClientLog } from '@/types/v1/game_gm'
import { feedback } from '@/utils/feedback'
import { copyTextToClipboard } from '@/utils/file'

/** 上传原因：exception 异常、crash 崩溃、fatal 致命错误由客户端自动上传，manual 是玩家手动上传。 */
export const CLIENT_LOG_REASONS = ['exception', 'crash', 'fatal', 'manual'] as const

/** 每个游戏账号保留的份数（服务端规则）。 */
export const CLIENT_LOG_KEEP = 5

type Translate = (key: string, params?: Record<string, unknown>) => string

export const clientLogReasonLabel = (t: Translate, reason: string) =>
  (CLIENT_LOG_REASONS as readonly string[]).includes(reason) ? t(`clientLogs.reasons.${reason}`) : reason || '—'

export const clientLogReasonVariant = (reason: string) => {
  switch (reason) {
    case 'crash':
    case 'fatal':
      return 'error' as const
    case 'exception':
      return 'warning' as const
    case 'manual':
      return 'primary' as const
    default:
      return 'neutral' as const
  }
}

/** 绑定的通行证：昵称（用户名）；没有绑定时为空串。 */
export const clientLogPassport = (log: GmClientLog) => {
  const l = log.link
  if (!l?.passportUserId) return ''
  return l.name && l.name !== l.username ? `${l.name} (@${l.username})` : `@${l.username}`
}

/** 服务器名称与游戏名称。 */
export const clientLogServer = (log: GmClientLog) => {
  const s = log.server
  if (!s) return '—'
  return [s.name || s.id, s.gameName].filter(Boolean).join(' · ')
}

/** 读取日志原文（查看与下载共用）。 */
export const fetchClientLogContent = async (id: string) => (await getGmClientLog(id)).data.content

/** 把日志原文存成 .log 文件。 */
export const saveClientLog = (log: GmClientLog, text: string) => {
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `client-${log.playerId}-${dayjs(log.occurredAt).format('YYYYMMDD-HHmmss')}-${log.reason || 'log'}.log`
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** 复制文本并提示结果。 */
export const copyClientLogText = async (t: Translate, text: string, success: string) => {
  try {
    await copyTextToClipboard(text)
    feedback.success(success)
  } catch {
    feedback.error(t('clientLogs.copyFailed'))
  }
}
