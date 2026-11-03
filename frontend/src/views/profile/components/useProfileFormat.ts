import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'

/** 个人中心用的时间 / 设备 / 登录方式格式化 */
export const useProfileFormat = () => {
  const { t } = useI18n()

  /** 相对时间：刚刚 / x 分钟前 / x 小时前 / 昨天 HH:mm / M-D HH:mm / YYYY-MM-DD */
  const relTime = (value?: unknown) => {
    if (!value) return '—'
    const d = dayjs(value instanceof Date ? value : String(value))
    if (!d.isValid()) return '—'
    const now = dayjs()
    const diffMin = now.diff(d, 'minute')
    if (diffMin < 1) return t('profile.time.justNow')
    if (diffMin < 60) return t('profile.time.minutesAgo', { n: diffMin })
    if (d.isSame(now, 'day')) return t('profile.time.today', { time: d.format('HH:mm') })
    if (d.isSame(now.subtract(1, 'day'), 'day')) return t('profile.time.yesterday', { time: d.format('HH:mm') })
    if (d.isSame(now, 'year')) return d.format('MM-DD HH:mm')
    return d.format('YYYY-MM-DD')
  }

  const daysAgo = (value?: unknown) => {
    if (!value) return null
    const d = dayjs(value instanceof Date ? value : String(value))
    return d.isValid() ? Math.max(0, dayjs().diff(d, 'day')) : null
  }

  const fullTime = (value?: unknown) => {
    if (!value) return '—'
    const d = dayjs(value instanceof Date ? value : String(value))
    return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '—'
  }

  const scopeLabel = (scope?: string) =>
    scope === 'loopback' || scope === 'lan' || scope === 'public' ? t(`profile.ipScope.${scope}`) : ''

  const methodLabel = (method?: string) =>
    method === 'email_code' ? t('profile.method.emailCode') : t('profile.method.password')

  const deviceIcon = (type?: string) => {
    if (type === 'mobile') return 'i-lucide-smartphone'
    if (type === 'tablet') return 'i-lucide-tablet'
    if (type === 'cli') return 'i-lucide-terminal'
    return 'i-lucide-monitor'
  }

  const deviceTypeLabel = (type?: string) =>
    type === 'mobile' || type === 'tablet' || type === 'cli' || type === 'desktop'
      ? t(`profile.deviceType.${type}`)
      : t('profile.device.unknown')

  /** 设备名：「Windows 10 / Chrome 130」→「Windows 10 · Chrome 130」 */
  const deviceName = (device?: string) => {
    const raw = (device || '').trim()
    if (!raw || raw === 'unknown') return t('profile.device.unknown')
    return raw
      .split('/')
      .map((s) => s.trim())
      .filter(Boolean)
      .join(' · ')
  }

  return { relTime, daysAgo, fullTime, scopeLabel, methodLabel, deviceIcon, deviceTypeLabel, deviceName }
}
