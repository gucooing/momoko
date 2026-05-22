import { defineStore } from 'pinia'
import { ElMessage } from 'element-plus'
import { logoutRequest, userMeInfoRequest } from '@/api/login'
import { updateMeRequest, updatePasswordRequest } from '@/api/user'
import defaultAvatarSvg from '@/assets/defaultAvatar.svg'
import router, { resetRouter } from '@/router'
import { useMenuStore } from '@/stores/menu'
import { useTabsStore } from '@/stores/tabs'
import { useUserProfileStore } from '@/stores/user/profile'
import type {
  UserPasswordFormValue,
  UserProfileFormValue,
} from '@/stores/user/types'
import { buildLoginRoute } from '@/utils/authRedirect'
import { resolveAvatarUrl } from '@/utils/assets'
import type { UserInfo } from '@/types/v1/user'

export const useUserStore = defineStore('user', () => {
  const defaultAvatarImg = ref(defaultAvatarSvg)
  const userInfo = ref<UserInfo | null>(null)
  let isRedirectingToLogin = false

  const applyUserInfo = (nextUser?: UserInfo | null) => {
    if (!nextUser) {
      userInfo.value = null
      return
    }

    userInfo.value = nextUser
    userInfo.value.bio = userInfo.value.bio || '这个人很懒，什么都没留下~'

    if (!userInfo.value.avatar) {
      userInfo.value.avatar = defaultAvatarImg.value
    }
  }

  const getUserInfo = async () => {
    const { data: meRes } = await userMeInfoRequest({})
    applyUserInfo(meRes?.user)
  }

  const getUserRoleName = () => {
    return userInfo.value?.roleName || '无权限'
  }

  const resolvedUserAvatar = computed(() => {
    return resolveAvatarUrl(userInfo.value?.avatar) || defaultAvatarImg.value
  })

  const updateCurrentUser = async (payload: UserProfileFormValue) => {
    if (!userInfo.value?.userId) {
      throw new Error('当前用户信息缺失')
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
    ElMessage.success('修改个人资料成功')
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
    ElMessage.success('修改密码成功,即将重新登录')
    setTimeout(logoutLocal, 1000)
  }

  return {
    userInfo,
    resolvedUserAvatar,
    getUserInfo,
    getUserRoleName,
    updateUserProfile,
    updatePassword,
    logoutLocal,
    logout,
  }
})
