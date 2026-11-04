// 节点组页面共用的时间格式：相对时间（跟随界面语言）与短日期。
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/zh-hk'
import { useI18n } from 'vue-i18n'

dayjs.extend(relativeTime)

type TimeInput = Date | string | number | null | undefined

export const toDayjs = (value: TimeInput) => {
  if (value === null || value === undefined || value === '') return null
  const d = dayjs(value)
  return d.isValid() ? d : null
}

export const useNodeTime = () => {
  const { locale } = useI18n()
  const dayjsLocale = computed(() =>
    locale.value === 'en-US' ? 'en' : locale.value === 'zh-HK' ? 'zh-hk' : 'zh-cn',
  )

  /** 相对时间：「3 分钟前」「in 17 days」 */
  const ago = (value: TimeInput) => {
    const d = toDayjs(value)
    return d ? d.locale(dayjsLocale.value).fromNow() : '—'
  }
  /** 距今时长（不带「前」）：「3 天」「a few seconds」 */
  const since = (value: TimeInput) => {
    const d = toDayjs(value)
    return d ? d.locale(dayjsLocale.value).fromNow(true) : '—'
  }
  const date = (value: TimeInput) => toDayjs(value)?.format('YYYY-MM-DD') ?? '—'
  const dateTime = (value: TimeInput) => toDayjs(value)?.format('YYYY-MM-DD HH:mm') ?? '—'
  const shortTime = (value: TimeInput) => toDayjs(value)?.format('MM-DD HH:mm') ?? '—'
  /** 距今天数（向上取整，已过去为负数） */
  const daysUntil = (value: TimeInput) => {
    const d = toDayjs(value)
    if (!d) return null
    return Math.ceil(d.diff(dayjs(), 'hour') / 24)
  }

  return { ago, since, date, dateTime, shortTime, daysUntil }
}
