import { defineStore } from 'pinia'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import { MenuType, type MenuInfo } from '@/types/v1/system'
import { translate as t } from '@/locales'

export interface TabItem {
  path: string
  fullPath: string
  title: string
  icon?: string
  closable: boolean
  name?: string | symbol
}

/** 不算换页的地址参数 */
const IN_PAGE_PARAMS = ['tab', 'view', 'session', 'workspace']

export const useTabsStore = defineStore('tabs', () => {
  const tabs = ref<TabItem[]>([])
  const activePath = ref<string>('')
  const routeRefreshKeys = ref<Record<string, number>>({})

  // 页内标签参数 ?tab= 与标签里的子页参数 ?view=（如 GM 运营「操作与日志」的执行操作 / GM 记录 / 客户端日志）不算换页：
  // 同一页面切换标签共用一个页面实例，不重建整页、不重新拉数据（见 config/pageGroups.ts）。
  // Code 页的 ?session=（当前会话）和 ?workspace=（新会话预选的工作区）同理：换会话是页面自己的事。
  const pageKeyOf = (fullPath: string) => {
    const [pathAndQuery = '', hash = ''] = fullPath.split('#')
    const [path = '', query = ''] = pathAndQuery.split('?')
    const params = new URLSearchParams(query)
    for (const key of IN_PAGE_PARAMS) params.delete(key)
    const rest = params.toString()
    return `${path}${rest ? `?${rest}` : ''}${hash ? `#${hash}` : ''}`
  }

  const getRouteRefreshKey = (fullPath: string) => routeRefreshKeys.value[pageKeyOf(fullPath)] || 0

  const getRouteRenderKey = (fullPath: string) => `${pageKeyOf(fullPath)}::${getRouteRefreshKey(fullPath)}`

  const refreshTab = (fullPath: string) => {
    if (!fullPath) return
    routeRefreshKeys.value[pageKeyOf(fullPath)] = getRouteRefreshKey(fullPath) + 1
  }

  const clearClosedTabRefreshKeys = () => {
    const openedPaths = new Set(tabs.value.map((tab) => pageKeyOf(tab.fullPath)))
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

    // 同一页面不同 ?tab= 视为同一个标签，只更新地址
    const existTab = tabs.value.find((tab) => pageKeyOf(tab.fullPath) === pageKeyOf(route.fullPath))
    if (existTab) {
      existTab.fullPath = route.fullPath
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
      title: queryTitle || (route.meta?.title as string) || route.name?.toString() || t('utils.tabs.unnamed'),
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
