// 手机底部导航与「更多」面板的共享定义。

export interface TabbarItem {
  path: string
  label: string
  icon: string
  /** 这些路径前缀下的页面也高亮该标签（如应用详情、分享 / 文件源） */
  also?: string[]
}

/** 底部导航的一级页面（按当前用户可访问的菜单出现），其余页面都归到「更多」。 */
export const TABBAR_ITEMS: TabbarItem[] = [
  { path: '/dashboard/home', label: 'layout.tabbar.home', icon: 'HOutline:HomeIcon' },
  { path: '/instance/list', label: 'layout.tabbar.apps', icon: 'HOutline:CubeIcon', also: ['/instance/'] },
  { path: '/docker', label: 'layout.tabbar.containers', icon: 'HOutline:ServerStackIcon' },
  { path: '/file', label: 'layout.tabbar.files', icon: 'HOutline:FolderIcon' },
]

export const TAB_ROOTS = TABBAR_ITEMS.map((i) => i.path)

/** 每次请求聚焦「更多」里的搜索框时自增（顶栏搜索按钮、工作台搜索按钮）。 */
export const moreSearchFocusTick = ref(0)

/** 打开 / 关闭「更多」面板（状态复用 menuStore.isMobileMenuOpen，退出登录时会被一并复位）。 */
export const useMorePanel = () => {
  const menuStore = useMenuStore()
  return {
    open: (focusSearch = false) => {
      menuStore.isMobileMenuOpen = true
      if (focusSearch) moreSearchFocusTick.value += 1
    },
    close: () => {
      menuStore.isMobileMenuOpen = false
    },
    toggle: () => {
      menuStore.isMobileMenuOpen = !menuStore.isMobileMenuOpen
    },
  }
}
