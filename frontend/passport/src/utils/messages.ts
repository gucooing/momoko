// 消息与最近活动的文案：后端只给种类与参数，这里本地化。
import { t, te } from '@passport/locales'
import { fmtDate } from '@passport/utils/format'
import type { GameServerBrief } from '@/types/v1/game_common'
import type { PassportActivity, PassportMessage } from '@/types/v1/passport'

export function serverLabel(s: GameServerBrief | undefined): string {
  if (!s) return ''
  return s.gameName ? `${s.gameName} · ${s.name}` : s.name
}

function label(prefix: string, value: string | undefined): string {
  if (!value) return ''
  const key = `${prefix}.${value}`
  return te(key) ? t(key) : value
}

export function messageText(m: PassportMessage): string {
  const p = m.params || {}
  const key = `home.msg.${m.kind}`
  if (!te(key)) return m.kind
  return t(key, {
    ...p,
    server: m.server?.name || '',
    device: p.device || p.platform || label('common.device', p.device_type),
    date: p.at ? fmtDate(p.at) : '',
    title: p.title || p.command || '',
    note: p.note || '',
  })
}

/** 消息来源：游戏 · 服务器，或「通行证」。 */
export function messageSource(m: PassportMessage): string {
  return m.server ? serverLabel(m.server) : t('nav.brand')
}

/** 安全类消息（用通行证图标），其余是游戏相关。 */
export function messageIsSecurity(m: PassportMessage): boolean {
  return !m.server
}

export function activityText(a: PassportActivity): string {
  const p = a.params || {}
  const server = a.server?.name || ''
  switch (a.kind) {
    case 'login':
      return t('home.act.login', {
        device: p.device || p.platform || label('common.device', p.device_type),
        ip: p.ip || '',
        method: label('common.method', p.method),
      })
    case 'bind':
      return t('home.act.bind', { server, nickname: p.nickname || '', method: label('common.bindMethod', p.method) })
    case 'unbind':
      return t(p.by === 'admin' ? 'home.act.unbindAdmin' : 'home.act.unbind', { server, nickname: p.nickname || '' })
    case 'self_service':
      return t('home.act.selfService', { server, title: p.title || p.command || '', status: label('home.status', p.status) })
    case 'redeem':
      return t('home.act.redeem', { server, code: p.code || '', result: label('home.redeem', p.result) })
    default:
      return a.kind
  }
}

/** 活动所在：游戏名或「通行证」。 */
export function activityWhere(a: PassportActivity): string {
  return a.server ? a.server.gameName || a.server.game : t('nav.brand')
}
