import type { MenuInfo } from '@/types/v1/system'

export const sortMenuTreeByOrder = (menus: MenuInfo[]): MenuInfo[] => {
  return [...menus]
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
    .map((menu) => ({
      ...menu,
      children: menu.children?.length ? sortMenuTreeByOrder(menu.children) : menu.children,
    }))
}
