import { updateMyAppearanceRequest } from '@/api/user'
import type { UserAppearance } from '@/types/v1/user'
import { useThemeStore, type AppearancePrefs } from '@/stores/theme'

// 外观偏好跟随账号同步：登录后用账号里保存的偏好覆盖本机设置；之后在本机改动会自动保存回账号，
// 换设备登录也是同一套外观。从未保存过时，把本机当前设置作为初始值存上去。

const SAVE_DELAY_MS = 600

/** 最近一次与账号一致的偏好（JSON），用于避免把刚同步下来的值又写回去 */
let lastSynced = ''
/** 已拿到账号偏好、可以开始自动保存 */
let ready = false
let saveTimer: ReturnType<typeof setTimeout> | null = null

const toRemote = (p: AppearancePrefs): UserAppearance => ({
  palette: p.palette,
  mode: p.mode,
  compact: p.compact,
  reduceMotion: p.reduceMotion,
  radius: p.radius,
  showTabs: p.showTabs,
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
export const applyRemoteAppearance = (remote?: UserAppearance | null) => {
  const theme = useThemeStore()
  if (remote) {
    theme.applyAppearance({
      palette: remote.palette as AppearancePrefs['palette'],
      mode: remote.mode as AppearancePrefs['mode'],
      compact: remote.compact,
      reduceMotion: remote.reduceMotion,
      radius: remote.radius as AppearancePrefs['radius'],
      showTabs: remote.showTabs,
    })
    lastSynced = JSON.stringify(theme.appearance)
  } else {
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

/** 在已登录的外壳里挂载：本机外观变化时自动保存到账号。 */
export const useAppearanceSync = () => {
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
