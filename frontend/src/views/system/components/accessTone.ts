// 用户与权限组的小工具：角色配色、贴纸配色/旋转、最近登录时间的友好显示。
import dayjs from 'dayjs'

export const SUPER_ADMIN_ROLE_ID = 'role_1'
export const NO_PERMISSION_ROLE_ID = 'role_2'

export type Tone = 'p' | 'a' | 'a2' | 'ok' | 'warn' | 'none'

const CYCLE: Tone[] = ['p', 'a2', 'ok']
const STICKERS: Exclude<Tone, 'none'>[] = ['a', 'p', 'a2', 'ok']
const ROTS = [-4, 3, -2, 4]

const hash = (s: string) => {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0
  return Math.abs(h)
}

/** 角色标签配色：超级管理员黄色、无权限灰色，其它按 id 稳定取色。 */
export const roleTone = (roleId?: string): Tone => {
  if (!roleId) return 'none'
  if (roleId === SUPER_ADMIN_ROLE_ID) return 'a'
  if (roleId === NO_PERMISSION_ROLE_ID) return 'none'
  return CYCLE[hash(roleId) % CYCLE.length]!
}

/** 贴纸头像配色 + 轻微旋转（按 key 稳定）。 */
export const stickerOf = (key: string, index?: number) => {
  const k = index ?? hash(key)
  return {
    cls: `m-sticker--${STICKERS[k % STICKERS.length]}`,
    rot: `rotate(${ROTS[k % ROTS.length]}deg)`,
  }
}

/** 名字首字（中文取第一个字，英文取首字母大写）。 */
export const initialOf = (name?: string) => (name || '?').trim().charAt(0).toUpperCase() || '?'

type T = (key: string, params?: Record<string, unknown>) => string

/** 最近登录：刚刚 / 今天 09:42 / 昨天 20:30 / 3 天前 / 10-28 14:02 / 2025-10-28。 */
export const formatLastLogin = (t: T, value: unknown): string => {
  if (!value) return t('accessAdmin.user.neverLogin')
  const d = dayjs(value as string)
  if (!d.isValid()) return t('accessAdmin.user.neverLogin')
  const now = dayjs()
  const mins = now.diff(d, 'minute')
  if (mins < 2) return t('accessAdmin.user.justNow')
  if (d.isSame(now, 'day')) return t('accessAdmin.user.todayAt', { time: d.format('HH:mm') })
  if (d.isSame(now.subtract(1, 'day'), 'day'))
    return t('accessAdmin.user.yesterdayAt', { time: d.format('HH:mm') })
  const days = now.startOf('day').diff(d.startOf('day'), 'day')
  if (days < 7) return t('accessAdmin.user.daysAgo', { n: days })
  return d.isSame(now, 'year') ? d.format('MM-DD HH:mm') : d.format('YYYY-MM-DD')
}

export const loginMethodLabel = (t: T, method?: string) => {
  if (!method) return ''
  if (method === 'password' || method === 'email_code') return t(`accessAdmin.user.methods.${method}`)
  return t('accessAdmin.user.methods.other')
}
