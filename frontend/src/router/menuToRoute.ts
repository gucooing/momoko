import { type RouteComponent, type RouteRecordRaw } from 'vue-router'
import { MenuType, type MenuInfo } from '@/types/v1/system'

const modules = import.meta.glob('@/views/**/*.vue')
const fallbackView = modules['/src/views/exception/404/index.vue'] as RouteComponent | undefined

export const menuToRoute = (menuList: MenuInfo[]) => {
  const dynamicRoute: RouteRecordRaw[] = []

  menuList.forEach((menu) => {
    if (menu.type === MenuType.MenuType_Menu) {
      const path = menu.path.replace(/^\/+/, '')
      const viewPath = `/src/views/${path}/index.vue`
      const viewComponent = modules[viewPath] as RouteComponent | undefined
      const component = viewComponent || fallbackView

      if (!component) return
      if (!viewComponent) {
        console.warn(`[menuToRoute] Missing view file: ${viewPath}. Fallback to /exception/404.`)
      }

      dynamicRoute.push({
        path,
        name: menu.id,
        component,
        meta: {
          icon: menu.icon,
          title: menu.title,
          id: menu.id,
          parentId: menu.parentId,
          keepAlive: true,
        },
      })
    }

    if (menu.type === MenuType.MenuType_Directory && menu.children?.length) {
      dynamicRoute.push(...menuToRoute(menu.children))
    }
  })

  return dynamicRoute
}
