import { MenuType, type MenuInfo } from '@/types/v1/system'
import { PAGE_GROUPS, type PageGroup } from '@/config/pageGroups'

/**
 * 当前用户在当前节点上可进入的菜单路径集合（已按菜单启用状态与节点授权过滤）。
 * 分组页（config/pageGroups.ts）里有权限的标签另以 `<分组路径>/<标签>` 虚拟路径列出（如 /file/share，
 * 与 config/nodePermission.ts 的节点授权路径一致）；跳转到标签请用 pageTabTo。
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
    for (const def of Object.values(PAGE_GROUPS) as PageGroup[]) {
      if (!set.has(def.path)) continue
      for (const tab of def.tabs) {
        if (!tab.perm || menuStore.hasButtonPermission(tab.perm)) set.add(tab.nodePath || `${def.path}/${tab.key}`)
      }
    }
    return set
  })
}
