import { MenuType, type MenuInfo } from '@/types/v1/system'

/**
 * 当前用户在当前节点上可进入的菜单路径集合（已按菜单启用状态与节点授权过滤）。
 * 用于底部导航、快捷入口等「只展示能去的地方」。
 */
export const useAccessibleRoutes = () => {
  const menuStore = useMenuStore()
  return computed(() => {
    const set = new Set<string>()
    const walk = (list: MenuInfo[]) => {
      for (const m of list) {
        if (m.type === MenuType.MenuType_Menu && m.path) set.add(m.path)
        if (m.children?.length) walk(m.children)
      }
    }
    walk(menuStore.navMenuList)
    return set
  })
}
