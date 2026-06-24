import { defineStore } from 'pinia'
import { iconComponents } from '@/config/iconRegistry'
import { useWindowSize } from '@vueuse/core'
import { mePermissionsRequest } from '@/api/login'
import { MenuStatus, type MenuInfo } from '@/types/v1/system'

export const useMenuStore = defineStore('menu', () => {
  const isCollapse = ref(false)
  const toggleCollapse = () => {
    isCollapse.value = !isCollapse.value
  }

  const { width } = useWindowSize()
  const isMobile = computed(() => width.value < 992)

  watchEffect(() => {
    if (isMobile.value) isCollapse.value = false
  })

  const isMobileMenuOpen = ref(false)
  const toggleMobileMenu = () => {
    isMobileMenuOpen.value = !isMobileMenuOpen.value
  }

  const menuList = ref<MenuInfo[]>([])
  const allMenuList = ref<MenuInfo[]>([])
  const buttonPermissions = ref<string[]>([])
  const allButtonPermissions = ref<string[]>([])
  const hasLoadedPermissions = ref(false)

  const filterActiveMenus = (menus: MenuInfo[]): MenuInfo[] => {
    return menus
      .filter((menu) => menu.status === MenuStatus.MenuStatus_Active)
      .map((menu) => ({
        ...menu,
        children: menu.children?.length ? filterActiveMenus(menu.children) : menu.children,
      }))
  }

  const hasButtonPermission = (permission: string | string[]): boolean => {
    if (typeof permission === 'string') {
      return buttonPermissions.value.includes(permission)
    }

    return permission.every((item) => buttonPermissions.value.includes(item))
  }

  const getUserPermissions = async () => {
    const { data: res } = await mePermissionsRequest({})
    if (!res) return

    const allMenus = res.menus || []
    const activeMenus = filterActiveMenus(allMenus)
    const permissions = [...(res.permissions || [])]

    allMenuList.value = allMenus
    allButtonPermissions.value = permissions
    menuList.value = activeMenus
    buttonPermissions.value = permissions
    hasLoadedPermissions.value = true
  }

  const clearUserPermissions = () => {
    menuList.value = []
    allMenuList.value = []
    buttonPermissions.value = []
    allButtonPermissions.value = []
    hasLoadedPermissions.value = false
  }

  return {
    iconComponents,
    menuList,
    allMenuList,
    buttonPermissions,
    allButtonPermissions,
    hasButtonPermission,
    isCollapse,
    isMobileMenuOpen,
    hasLoadedPermissions,
    isMobile,
    toggleCollapse,
    toggleMobileMenu,
    getUserPermissions,
    clearUserPermissions,
  }
})
