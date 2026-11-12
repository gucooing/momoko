// 游戏服务器页面共用：状态色、时长、检查项与变更记录的本地化。
import dayjs from 'dayjs'
import { formatFileSize } from '@/utils/file'
import type { GameServerCheck, GameServerHistoryEntry, GameServerInfo } from '@/types/v1/game_server'

type Translate = (key: string, params?: Record<string, unknown>) => string
type Te = (key: string) => boolean

export type PillVariant = 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'primary'

export const stateVariant = (state: string): PillVariant => {
  switch (state) {
    case 'running':
      return 'success'
    case 'maintenance':
    case 'starting':
    case 'stopping':
    case 'restarting':
      return 'warning'
    case 'unreachable':
    case 'missing':
      return 'error'
    default:
      return 'neutral'
  }
}

export const checkVariant = (state: string): PillVariant => {
  switch (state) {
    case 'ok':
      return 'success'
    case 'warn':
      return 'warning'
    case 'fail':
      return 'error'
    default:
      return 'neutral'
  }
}

export const toMs = (v: unknown): number => {
  if (!v) return 0
  const d = dayjs(v as string)
  return d.isValid() ? d.valueOf() : 0
}

export const fmtTime = (v: unknown, withYear = false): string => {
  const ms = toMs(v)
  if (!ms) return '—'
  return dayjs(ms).format(withYear ? 'YYYY-MM-DD HH:mm' : 'MM-DD HH:mm')
}

export const fmtDuration = (t: Translate, ms: number): string => {
  const m = Math.floor(Math.max(0, ms) / 60000)
  const d = Math.floor(m / 1440)
  const h = Math.floor((m % 1440) / 60)
  if (d > 0) return t('gameServers.common.durDays', { d, h })
  if (h > 0) return t('gameServers.common.durHours', { h, m: m % 60 })
  return t('gameServers.common.durMinutes', { m })
}

export const fmtAgo = (t: Translate, v: unknown, now = Date.now()): string => {
  const ms = toMs(v)
  if (!ms) return '—'
  const diff = Math.max(0, now - ms)
  const m = Math.floor(diff / 60000)
  if (m < 1) return t('gameServers.common.justNow')
  if (m < 60) return t('gameServers.common.minutesAgo', { n: m })
  const h = Math.floor(m / 60)
  if (h < 24) return t('gameServers.common.hoursAgo', { n: h })
  return t('gameServers.common.daysAgo', { n: Math.floor(h / 24) })
}

export const fmtBytes = (n: number | string | undefined | null): string => formatFileSize(Number(n) || 0)

export const fmtNum = (n: number | string | undefined | null): string => {
  const v = Number(n)
  return Number.isFinite(v) ? v.toLocaleString() : '—'
}

/** MGP status 原样（Value）里的对象。 */
export type StatusDoc = {
  online?: number
  onlinePeak?: number
  registered?: number
  newToday?: number
  uptimeSeconds?: number
  version?: string
  resource?: { app?: string; master?: string; octo?: number }
  maintenance?: { enabled?: boolean }
  saves?: { waiting?: number; capacity?: number }
  database?: { migration?: number }
}

export const statusOf = (info: GameServerInfo | undefined | null): StatusDoc | null => {
  const s = info?.status as unknown
  return s && typeof s === 'object' ? (s as StatusDoc) : null
}

/** 资源版本的短文本：Octo 705100 · r83 由服务端 status.resource.octo 给出（没有 revision 时只有 octo）。 */
export const resourceText = (st: StatusDoc | null): string => {
  const r = st?.resource
  if (!r) return '—'
  const parts: string[] = []
  if (r.octo) parts.push(`Octo ${r.octo}`)
  if (r.master) parts.push(r.master.length > 10 ? `${r.master.slice(0, 8)}…` : r.master)
  return parts.join(' · ') || '—'
}

const p = (c: GameServerCheck, k: string) => c.params?.[k] ?? ''

/** 一项检查的标题与说明。 */
export const checkText = (t: Translate, te: Te, c: GameServerCheck): { title: string; desc: string } => {
  const role = p(c, 'role')
  const title = t(`gameServers.check.title.${c.key}`, { role: role ? t(`gameServers.role.${role}`) : '' })
  const params = { ...c.params } as Record<string, unknown>
  if (params.size) params.size = fmtBytes(params.size as string)
  if (params.bytes) params.bytes = fmtBytes(params.bytes as string)
  if (params.objects) params.objects = fmtNum(params.objects as string)
  let desc = ''
  switch (c.key) {
    case 'dir':
      desc = t('gameServers.check.dirFail', params)
      break
    case 'executable':
      if (c.state === 'ok') desc = t(p(c, 'version') ? 'gameServers.check.executableOk' : 'gameServers.check.executableNoVersion', params)
      else if (te(`gameServers.check.executableFail.${c.errorCode}`)) desc = t(`gameServers.check.executableFail.${c.errorCode}`, params)
      break
    case 'config':
      if (c.state !== 'ok') desc = t('gameServers.check.configFail', params)
      else desc = p(c, 'template') ? t('gameServers.check.configTemplate', params) : t('gameServers.check.configOk', params)
      break
    case 'database':
      if (p(c, 'driver') === 'sqlite') desc = t(c.state === 'ok' ? 'gameServers.check.dbSqlite' : 'gameServers.check.dbSqliteNew', params)
      else desc = t('gameServers.check.dbOther', params)
      break
    case 'resources':
      desc = t(c.state === 'ok' ? 'gameServers.check.resourcesOk' : 'gameServers.check.resourcesFail', params)
      break
    case 'data':
      desc = t(c.state === 'ok' ? 'gameServers.check.dataOk' : 'gameServers.check.dataFail', params)
      break
    case 'mgpKey':
      desc = t(c.state === 'ok' ? 'gameServers.check.mgpKeyOk' : 'gameServers.check.mgpKeyPending', params)
      break
    case 'tokenSecret':
      desc = t('gameServers.check.tokenSecret', params)
      break
    case 'resourceServer':
      desc = t('gameServers.check.resourceServerOk', params)
      break
    case 'mirror':
      if (c.state === 'ok') {
        desc = `${t('gameServers.check.mirrorOk', params)} · ${t(p(c, 'published') === 'true' ? 'gameServers.check.mirrorPublished' : 'gameServers.check.mirrorUnpublished')}`
      } else desc = t('gameServers.check.mirrorWarn')
      break
    case 'port': {
      const map: Record<string, string> = { ok: 'portOk', warn: 'portWarn', pending: 'portPending', fail: 'portFail' }
      desc = t(`gameServers.check.${map[c.state] || 'portOk'}`, params)
      break
    }
    case 'connect':
      desc = t(c.state === 'ok' ? 'gameServers.check.connectOk' : 'gameServers.check.connectFail', params)
      break
    case 'signature':
      desc = c.state === 'pending' ? t('gameServers.check.pending') : t(c.state === 'ok' ? 'gameServers.check.signatureOk' : 'gameServers.check.signatureFail', params)
      break
    case 'manifest':
      desc = c.state === 'ok' ? t('gameServers.check.manifestOk', params) : c.state === 'pending' ? t('gameServers.check.pending') : ''
      break
    case 'status':
      desc = c.state === 'ok' ? t('gameServers.check.statusOk', params) : c.state === 'pending' ? t('gameServers.check.pending') : ''
      break
  }
  if (c.state === 'fail' && c.errorMessage) desc = desc ? `${desc} · ${c.errorMessage}` : c.errorMessage
  return { title, desc }
}

/** 变更记录的说明。 */
export const historyText = (t: Translate, te: Te, h: GameServerHistoryEntry): string => {
  const params: Record<string, unknown> = { ...h.params }
  const onOff = (v: unknown) => t(v === 'true' ? 'gameServers.history.on' : 'gameServers.history.off')
  if (h.kind === 'server') {
    params.from = params.from || t('gameServers.history.versionUnknown')
    params.to = params.to || t('gameServers.history.versionUnknown')
  }
  if (h.kind === 'mode') {
    if (params.mode) params.mode = t(`gameServers.mode.${params.mode}`)
    if (params.from) params.from = t(`gameServers.mode.${params.from}`)
    if (params.to) params.to = t(`gameServers.mode.${params.to}`)
  }
  if (h.action === 'policy') params.enabled = onOff(params.enabled)
  if (h.kind === 'backup' && h.action === 'create') {
    params.trigger = t(`gameServers.backup.trigger.${params.trigger}`)
    params.size = params.size ? fmtBytes(params.size as string) : '—'
  }
  if (h.action === 'options') {
    params.autoStart = onOff(params.autoStart)
    params.autoRestart = onOff(params.autoRestart)
    params.resourceServer = onOff(params.resourceServer)
  }
  if (typeof params.keys === 'string') params.keys = (params.keys as string).split(',').join(', ')
  if (h.kind === 'backup' && h.action === 'restore') return t('gameServers.history.restore', params)
  const key = `gameServers.history.action.${h.kind}.${h.action}`
  const text = te(key) ? t(key, params) : `${h.kind}.${h.action}`
  return text
}

export const historyMessage = (t: Translate, h: GameServerHistoryEntry): string => {
  if (!h.message) return ''
  if (h.message === 'interrupted') return t('gameServers.history.interrupted')
  return h.message
}

export const historyVariant = (result: string): PillVariant => {
  switch (result) {
    case 'success':
      return 'success'
    case 'failed':
      return 'error'
    case 'rolled_back':
      return 'warning'
    default:
      return 'info'
  }
}
