import { updateMyAppearanceRequest } from '@/api/user'
import type { UserAppearance } from '@/types/v1/user'
import type { SiteTheme, Theme } from '@/types/v1/theme'
import { useThemeStore, type AppearancePrefs } from '@/stores/theme'
import { IS_THEME_PREVIEW_FRAME } from '@/theme/frame'

// 外观偏好跟随账号同步：登录后用账号里保存的偏好（含当前主题、未保存的背景修改）覆盖本机设置；
// 之后在本机改动会自动保存回账号，换设备登录也是同一套外观。
// 账号从没保存过外观时：有全站默认主题就跟随它（不主动保存，直到用户自己改了什么），没有就把本机设置存为初始值。
// 编辑器的预览框里只套用草稿，不读写账号外观。

const SAVE_DELAY_MS = 600

/** 最近一次与账号一致的偏好（JSON），用于避免把刚同步下来的值又写回去 */
let lastSynced = ''
/** 已拿到账号偏好、可以开始自动保存 */
let ready = false
let saveTimer: ReturnType<typeof setTimeout> | null = null

const toRemote = (p: AppearancePrefs): UserAppearance => ({
  theme: p.theme,
  mode: p.mode,
  compact: p.compact,
  reduceMotion: p.reduceMotion,
  radius: p.radius,
  showTabs: p.showTabs,
  background: p.background ?? undefined,
})

const save = async (prefs: AppearancePrefs) => {
  const json = JSON.stringify(prefs)
  try {
    await updateMyAppearanceRequest({ appearance: toRemote(prefs) })
    lastSynced = json
  } catch {
    // 保存失败不打扰用户：本机仍然生效，下次改动会再试
  }
}

const scheduleSave = (prefs: AppearancePrefs) => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    saveTimer = null
    void save(prefs)
  }, SAVE_DELAY_MS)
}

/** 套用账号里保存的外观偏好（获取个人信息后调用）。 */
export const applyRemoteAppearance = (
  remote?: UserAppearance | null,
  currentTheme?: Theme | null,
  siteTheme?: SiteTheme | null,
) => {
  if (IS_THEME_PREVIEW_FRAME) return
  const theme = useThemeStore()
  theme.setSiteThemeData(siteTheme)
  if (remote) {
    theme.applyAppearance(
      {
        theme: remote.theme,
        mode: remote.mode as AppearancePrefs['mode'],
        compact: remote.compact,
        reduceMotion: remote.reduceMotion,
        radius: remote.radius as AppearancePrefs['radius'],
        showTabs: remote.showTabs,
        background: remote.background ?? null,
      },
      currentTheme,
    )
    lastSynced = JSON.stringify(theme.appearance)
  } else if (siteTheme?.content) {
    // 从没保存过外观、有全站默认：跟随全站默认，不主动保存
    theme.setFollowSite(true)
    lastSynced = JSON.stringify(theme.appearance)
  } else {
    // 从没保存过、也没有全站默认：把本机当前外观作为初始值存上去
    theme.setFollowSite(false)
    lastSynced = ''
    scheduleSave({ ...theme.appearance })
  }
  ready = true
}

/** 退出登录后停止自动保存，直到下次拿到账号偏好。 */
export const stopAppearanceSync = () => {
  ready = false
  lastSynced = ''
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
  }
}

/** 在根组件里挂载一次：已登录时本机外观变化自动保存到账号。 */
export const useAppearanceSync = () => {
  if (IS_THEME_PREVIEW_FRAME) return
  const theme = useThemeStore()
  const userStore = useUserStore()

  watch(
    () => theme.appearance,
    (prefs) => {
      if (!ready || !userStore.userInfo) return
      if (JSON.stringify(prefs) === lastSynced) return
      scheduleSave({ ...prefs })
    },
    { deep: true },
  )
  watch(
    () => userStore.userInfo,
    (info) => {
      if (!info) stopAppearanceSync()
    },
  )
}
