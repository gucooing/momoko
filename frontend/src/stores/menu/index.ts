import { defineStore } from 'pinia'
import { iconComponents } from '@/config/iconRegistry'
import { useWindowSize } from '@vueuse/core'
import { mePermissionsRequest } from '@/api/login'
import { MenuStatus, MenuType, type MenuInfo } from '@/types/v1/system'

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

  const collectButtonPermissions = (menus: MenuInfo[], acc: Set<string> = new Set()) => {
    menus.forEach((menu) => {
      if (menu.type === MenuType.MenuType_Button && menu.permissions) {
        acc.add(menu.permissions)
      }
      if (menu.children?.length) {
        collectButtonPermissions(menu.children, acc)
      }
    })
    return acc
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
    const activeButtonPermissionSet = collectButtonPermissions(activeMenus)

    allMenuList.value = allMenus
    allButtonPermissions.value = res.permissions
    menuList.value = activeMenus
    buttonPermissions.value =
      activeButtonPermissionSet.size > 0
        ? res.permissions.filter((permission) => activeButtonPermissionSet.has(permission))
        : res.permissions
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
