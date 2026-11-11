import { defineStore } from 'pinia'
import { logoutRequest, userMeInfoRequest } from '@/api/login'
import {
  completeMyAvatarUpload,
  presignMyAvatarUpload,
  putAvatarToStorage,
  updateMeRequest,
  updatePasswordRequest,
} from '@/api/user'
import defaultAvatarSvg from '@/assets/defaultAvatar.svg'
import router, { resetRouter } from '@/router'
import { useMenuStore } from '@/stores/menu'
import { useTabsStore } from '@/stores/tabs'
import { useUserProfileStore } from '@/stores/user/profile'
import { applyRemoteAppearance } from '@/stores/theme/sync'
import type {
  UserPasswordFormValue,
  UserProfileFormValue,
} from '@/stores/user/types'
import { buildLoginRoute } from '@/utils/authRedirect'
import { resolveAvatarUrl } from '@/utils/assets'
import type { UserInfo } from '@/types/v1/user'
import { translate } from '@/locales'

export const useUserStore = defineStore('user', () => {
  const defaultAvatarImg = ref(defaultAvatarSvg)
  const userInfo = ref<UserInfo | null>(null)
  /** 能否更换头像（系统设置开启了头像储存） */
  const avatarUploadEnabled = ref(false)
  let isRedirectingToLogin = false

  const applyUserInfo = (nextUser?: UserInfo | null) => {
    if (!nextUser) {
      userInfo.value = null
      return
    }

    userInfo.value = nextUser
    userInfo.value.bio = userInfo.value.bio || translate('user.defaultBio')

    if (!userInfo.value.avatar) {
      userInfo.value.avatar = defaultAvatarImg.value
    }
  }

  const getUserInfo = async () => {
    const { data: meRes } = await userMeInfoRequest({})
    applyUserInfo(meRes?.user)
    avatarUploadEnabled.value = !!meRes?.avatarUploadEnabled
    // 外观偏好跟随账号（见 stores/theme/sync.ts）
    if (meRes?.user) applyRemoteAppearance(meRes.appearance, meRes.currentTheme, meRes.siteTheme)
  }

  // 用户可以有多个角色：以「、」连接显示
  const getUserRoleName = () => {
    const names = (userInfo.value?.roles || []).map((r) => r.name).filter(Boolean)
    return names.join('、') || translate('user.noPermission')
  }

  const resolvedUserAvatar = computed(() => {
    return resolveAvatarUrl(userInfo.value?.avatar) || defaultAvatarImg.value
  })

  const updateCurrentUser = async (payload: UserProfileFormValue) => {
    if (!userInfo.value?.userId) {
      throw new Error(translate('user.currentUserMissing'))
    }

    const { data } = await updateMeRequest(payload)

    if (data?.user) {
      applyUserInfo(data.user)
      return
    }

    await getUserInfo()
  }

  const clearSessionAndRedirect = (
    options: { forceReload?: boolean; redirectPath?: string } = {},
  ) => {
    if (isRedirectingToLogin) return
    isRedirectingToLogin = true

    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')

    const menuStore = useMenuStore()
    const tabsStore = useTabsStore()
    const userProfileStore = useUserProfileStore()

    menuStore.clearUserPermissions()
    userInfo.value = null
    tabsStore.clearTabs()
    userProfileStore.clearProfileState()
    menuStore.isMobileMenuOpen = false

    const loginRoute = buildLoginRoute(options.redirectPath)
    const loginTarget = router.resolve(loginRoute).href

    if (options.forceReload) {
      window.location.replace(loginTarget)
      return
    }

    void router.replace(loginRoute).finally(() => {
      resetRouter()
      isRedirectingToLogin = false
    })
  }

  const updateUserProfile = async (data: UserProfileFormValue) => {
    await updateCurrentUser(data)
    feedback.success(translate('user.profileUpdateSuccess'))
  }

  /** 更换头像：申请直传 → 浏览器直接 PUT 到对象存储 → 凭凭证完成上传（服务端确认对象已存在才设为头像）。 */
  const uploadAvatar = async (file: File) => {
    const { data: presign } = await presignMyAvatarUpload({ contentType: file.type, size: file.size })
    if (!presign?.uploadUrl || !presign.uploadToken) throw new Error(translate('avatar.uploadFailed'))
    await putAvatarToStorage(presign.uploadUrl, file)
    const { data } = await completeMyAvatarUpload({ uploadToken: presign.uploadToken })
    if (data?.user) applyUserInfo(data.user)
    else await getUserInfo()
    feedback.success(translate('avatar.uploadSuccess'))
  }

  const logoutLocal = (options?: { forceReload?: boolean; redirectPath?: string }) => {
    clearSessionAndRedirect(options)
  }

  const logout = async () => {
    try {
      await logoutRequest({})
    } finally {
      clearSessionAndRedirect()
    }
  }

  const updatePassword = async (data: UserPasswordFormValue) => {
    await updatePasswordRequest({
      oldPassword: data.oldPassword,
      newPassword: data.newPassword,
    })
    feedback.success(translate('user.passwordUpdateSuccessRelogin'))
    setTimeout(logoutLocal, 1000)
  }

  return {
    userInfo,
    avatarUploadEnabled,
    resolvedUserAvatar,
    getUserInfo,
    getUserRoleName,
    updateUserProfile,
    uploadAvatar,
    updatePassword,
    logoutLocal,
    logout,
  }
})
