// 应用页面共用的小工具：贴纸取色、首字母、时长文案。
type Translate = (key: string, params?: Record<string, unknown>) => string

const TONES = ['p', 'a2', 'a', 'ok'] as const
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']

const hashOf = (s: string) => [...(s || '')].reduce((n, c) => (n * 31 + c.charCodeAt(0)) >>> 0, 7)

/** 按 id 稳定取贴纸色，同一个应用每次都是同一种颜色 */
export const toneOf = (id: string) => TONES[hashOf(id) % TONES.length]!
/** 按 id 稳定取贴纸的轻微旋转角度 */
export const rotOf = (id: string) => ROTS[hashOf(id) % ROTS.length]!
/** 名称首字母（大写） */
export const initialOf = (name: string) => (name || '?').trim().charAt(0).toUpperCase() || '?'

/** 时长：3 天 4 小时 / 2 小时 13 分 / 5 分钟 / 不到 1 分钟 */
export const fmtDuration = (t: Translate, ms: number) => {
  const m = Math.floor(Math.max(0, ms) / 60000)
  if (m < 1) return t('instance.ui.durJustNow')
  const d = Math.floor(m / 1440)
  const h = Math.floor((m % 1440) / 60)
  if (d > 0) return t('instance.ui.durDays', { d, h })
  if (h > 0) return t('instance.ui.durHours', { h, m: m % 60 })
  return t('instance.ui.durMinutes', { m })
}
