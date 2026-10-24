import { defineStore } from 'pinia'
import { iconComponents } from '@/config/iconRegistry'
import { useWindowSize } from '@vueuse/core'
import { mePermissionsRequest } from '@/api/login'
import { MenuStatus, MenuType, type MenuInfo } from '@/types/v1/system'
import { LOGIN_ONLY_NODE_PERMISSIONS, NODE_PERMISSION_CODES } from '@/config/nodePermission'
import { useNodeStore } from '@/stores/node'

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
  const currentVersion = ref('')

  const filterActiveMenus = (menus: MenuInfo[]): MenuInfo[] => {
    return menus
      .filter((menu) => menu.status === MenuStatus.MenuStatus_Active)
      .map((menu) => ({
        ...menu,
        children: menu.children?.length ? filterActiveMenus(menu.children) : menu.children,
      }))
  }

  /**
   * 按节点区分的权限（应用、文件、Docker 管理等）：
   * 远程节点上下文以节点上下文返回的生效权限为准（节点授予 API Key 的范围 ∩ 当前用户在主控的权限，后端已求交）；
   * 本机按主控 RBAC，本地仅需登录的项（机器状态、SSH）恒为允许。
   */
  const hasSinglePermission = (permission: string): boolean => {
    if (NODE_PERMISSION_CODES.has(permission)) {
      const nodeStore = useNodeStore()
      if (nodeStore.isRemote) return nodeStore.permissions.has(permission)
      if (LOGIN_ONLY_NODE_PERMISSIONS.has(permission)) return true
    }
    return buttonPermissions.value.includes(permission)
  }

  const hasButtonPermission = (permission: string | string[]): boolean => {
    if (typeof permission === 'string') {
      return hasSinglePermission(permission)
    }

    return permission.every((item) => hasSinglePermission(item))
  }

  // 导航用菜单：远程节点上下文中隐藏节点未授权的页面，目录下全部隐藏时目录一并隐藏。
  const filterNodeMenus = (menus: MenuInfo[]): MenuInfo[] => {
    const nodeStore = useNodeStore()
    return menus.flatMap((menu) => {
      if (menu.type === MenuType.MenuType_Menu) {
        return nodeStore.canAccessRoute(menu.path) ? [menu] : []
      }
      if (menu.type === MenuType.MenuType_Directory && menu.children?.length) {
        const children = filterNodeMenus(menu.children)
        return children.length ? [{ ...menu, children }] : []
      }
      return [menu]
    })
  }

  const navMenuList = computed(() =>
    useNodeStore().isRemote ? filterNodeMenus(menuList.value) : menuList.value,
  )

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
    currentVersion.value = res.currentVersion || ''
    hasLoadedPermissions.value = true
  }

  const clearUserPermissions = () => {
    menuList.value = []
    allMenuList.value = []
    buttonPermissions.value = []
    allButtonPermissions.value = []
    currentVersion.value = ''
    hasLoadedPermissions.value = false
  }

  return {
    iconComponents,
    menuList,
    navMenuList,
    allMenuList,
    buttonPermissions,
    allButtonPermissions,
    hasButtonPermission,
    isCollapse,
    isMobileMenuOpen,
    hasLoadedPermissions,
    currentVersion,
    isMobile,
    toggleCollapse,
    toggleMobileMenu,
    getUserPermissions,
    clearUserPermissions,
  }
})
