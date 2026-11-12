// 用户通行证后台共用的小工具：时间格式、头像、绑定方式与游戏名称。
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/zh-hk'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { gameAsset } from '@/games'
import { i18n } from '@/locales'
import { getRequestErrorMessage, isRequestErrorHandled } from '@/utils/request'
import { feedback } from '@/utils/feedback'
import type { GameServerBrief } from '@/types/v1/game_common'

dayjs.extend(relativeTime)

type TimeInput = Date | string | number | null | undefined

const toDayjs = (value: TimeInput) => {
  if (value === null || value === undefined || value === '') return null
  const d = dayjs(value)
  return d.isValid() && d.year() > 1970 ? d : null
}

export const usePassportTime = () => {
  const { locale } = useI18n()
  const dayjsLocale = computed(() =>
    locale.value === 'en-US' ? 'en' : locale.value === 'zh-HK' ? 'zh-hk' : 'zh-cn',
  )
  const ago = (value: TimeInput) => {
    const d = toDayjs(value)
    return d ? d.locale(dayjsLocale.value).fromNow() : ''
  }
  const date = (value: TimeInput) => toDayjs(value)?.format('YYYY-MM-DD') ?? ''
  const dateTime = (value: TimeInput) => toDayjs(value)?.format('YYYY-MM-DD HH:mm') ?? ''
  return { ago, date, dateTime }
}

/**
 * 通行证头像：上传的图片地址；否则游戏素材头像（<game>/<素材 key>，前端素材包解析）；都没有时返回空串（显示首字母）。
 */
export const passportAvatarSrc = (avatar?: string, avatarAsset?: string) => {
  if (avatar) return avatar
  if (avatarAsset) {
    const i = avatarAsset.indexOf('/')
    if (i > 0) return gameAsset(avatarAsset.slice(0, i), avatarAsset.slice(i + 1))
  }
  return ''
}

/** 游戏名称：manifest 给出的名称，其次适配器名称，最后是游戏 id。 */
export const useGameName = () => {
  const { t, te } = useI18n()
  return (game: string, name?: string) => {
    if (name) return name
    const key = `game.adapters.${game}`
    return te(key) ? t(key) : t('game.unnamed')
  }
}

/** 服务器名称（带游戏名称前缀不重复显示）。 */
export const serverLabel = (server?: GameServerBrief) => server?.name || i18n.global.t('game.unnamed')

/** 失败提示：全局拦截器已经提示过的不重复提示。 */
export const notifyError = (error: unknown) => {
  if (isRequestErrorHandled(error)) return
  feedback.error(getRequestErrorMessage(error))
}

/** 复制到剪贴板。 */
export const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
