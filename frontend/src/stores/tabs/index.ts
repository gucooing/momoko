import { defineStore } from 'pinia'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import { MenuType, type MenuInfo } from '@/types/v1/system'

export interface TabItem {
  path: string
  fullPath: string
  title: string
  icon?: string
  closable: boolean
  name?: string | symbol
}

export const useTabsStore = defineStore('tabs', () => {
  const tabs = ref<TabItem[]>([])
  const activePath = ref<string>('')
  const routeRefreshKeys = ref<Record<string, number>>({})

  const getRouteRefreshKey = (fullPath: string) => routeRefreshKeys.value[fullPath] || 0

  const getRouteRenderKey = (fullPath: string) => `${fullPath}::${getRouteRefreshKey(fullPath)}`

  const refreshTab = (fullPath: string) => {
    if (!fullPath) return
    routeRefreshKeys.value[fullPath] = getRouteRefreshKey(fullPath) + 1
  }

  const clearClosedTabRefreshKeys = () => {
    const openedPaths = new Set(tabs.value.map((tab) => tab.fullPath))
    Object.keys(routeRefreshKeys.value).forEach((fullPath) => {
      if (!openedPaths.has(fullPath)) {
        delete routeRefreshKeys.value[fullPath]
      }
    })
  }

  const moveTab = (fullPath: string, targetIndex: number) => {
    const fromIndex = tabs.value.findIndex((tab) => tab.fullPath === fullPath)
    if (fromIndex === -1) return
    if (!tabs.value[fromIndex]?.closable) return

    const nextTabs = [...tabs.value]
    const [movedTab] = nextTabs.splice(fromIndex, 1)
    if (!movedTab) return

    const fixedTabsCount = nextTabs.filter((tab) => !tab.closable).length
    const nextIndex = Math.max(fixedTabsCount, Math.min(targetIndex, nextTabs.length))
    nextTabs.splice(nextIndex, 0, movedTab)
    tabs.value = nextTabs
  }

  const findMenuByPath = (menus: MenuInfo[], path: string): MenuInfo | null => {
    for (const item of menus) {
      if (item.type === MenuType.MenuType_Menu && item.path === path) {
        return item
      }
      if (item.children?.length) {
        const child = findMenuByPath(item.children, path)
        if (child) return child
      }
    }
    return null
  }

  const addTab = (route: RouteLocationNormalizedLoaded) => {
    if (!route.meta?.keepAlive) return

    if (!tabs.value.some((tab) => tab.path === '/dashboard/home')) {
      const menuStore = useMenuStore()
      const homeMenu = findMenuByPath(menuStore.menuList, '/dashboard/home')
      if (homeMenu) {
        tabs.value.unshift({
          path: homeMenu.path,
          fullPath: homeMenu.path,
          title: homeMenu.title,
          icon: homeMenu.icon,
          closable: false,
          name: 'HomeView',
        })
      }
    }

    const existTab = tabs.value.find((tab) => tab.fullPath === route.fullPath)
    if (existTab) {
      activePath.value = route.fullPath
      return
    }

    const rawTitle = route.query?.tabTitle
    const queryTitle =
      typeof rawTitle === 'string'
        ? rawTitle.trim()
        : Array.isArray(rawTitle) && typeof rawTitle[0] === 'string'
          ? rawTitle[0].trim()
          : ''

    tabs.value.push({
      path: route.path,
      fullPath: route.fullPath,
      title: queryTitle || (route.meta?.title as string) || route.name?.toString() || '未命名',
      icon: route.meta?.icon as string | undefined,
      closable: route.path !== '/home' && route.path !== '/',
      name: route.name,
    })

    activePath.value = route.fullPath
    if (tabs.value.length === 1) {
      tabs.value[0]!.closable = false
    }
  }

  const removeTab = (fullPath: string) => {
    const index = tabs.value.findIndex((tab) => tab.fullPath === fullPath)
    if (index === -1) return

    const isActive = tabs.value[index]?.fullPath === activePath.value
    tabs.value.splice(index, 1)

    if (isActive && tabs.value.length > 0) {
      const nextTab = tabs.value[index] || tabs.value[index - 1]
      if (nextTab) {
        activePath.value = nextTab.fullPath
      }
    }

    if (tabs.value.length === 1) {
      tabs.value[0]!.closable = false
    }

    clearClosedTabRefreshKeys()
  }

  const closeOtherTabs = (fullPath: string) => {
    tabs.value = tabs.value.filter((tab) => tab.fullPath === fullPath || !tab.closable)
    activePath.value = fullPath
    clearClosedTabRefreshKeys()
  }

  const closeAllTabs = () => {
    tabs.value = tabs.value.filter((tab) => !tab.closable)
    if (tabs.value.length > 0) {
      activePath.value = tabs.value[0]?.fullPath || ''
    }
    clearClosedTabRefreshKeys()
  }

  const closeLeftTabs = (fullPath: string) => {
    const index = tabs.value.findIndex((tab) => tab.fullPath === fullPath)
    if (index === -1) return

    tabs.value = tabs.value.filter((tab, i) => i >= index || !tab.closable)
    activePath.value = fullPath
    clearClosedTabRefreshKeys()
  }

  const closeRightTabs = (fullPath: string) => {
    const index = tabs.value.findIndex((tab) => tab.fullPath === fullPath)
    if (index === -1) return

    tabs.value = tabs.value.filter((tab, i) => i <= index || !tab.closable)
    activePath.value = fullPath
    clearClosedTabRefreshKeys()
  }

  const updateTabTitle = (fullPath: string, title: string) => {
    const tab = tabs.value.find((t) => t.fullPath === fullPath)
    if (tab) {
      tab.title = title
    }
  }

  const clearTabs = () => {
    tabs.value = []
    activePath.value = ''
    routeRefreshKeys.value = {}
  }

  return {
    tabs,
    activePath,
    routeRefreshKeys,
    getRouteRefreshKey,
    getRouteRenderKey,
    refreshTab,
    addTab,
    moveTab,
    removeTab,
    closeOtherTabs,
    closeAllTabs,
    closeLeftTabs,
    closeRightTabs,
    updateTabTitle,
    clearTabs,
  }
})
