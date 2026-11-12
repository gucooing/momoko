import { createRouter, createWebHistory } from 'vue-router'
import { menuToRoute } from '@/router/menuToRoute'
import { staticRoutes } from '@/router/route'
import { useTabsStore } from '@/stores/tabs'
import { buildLoginRoute } from '@/utils/authRedirect'
import { getInitializeStatus } from '@/api/initialize'
import { ensureRouteLocales, translate } from '@/locales'
import NProgress from 'nprogress'
import { IS_THEME_PREVIEW_FRAME } from '@/theme/frame'

// 配置 NProgress
NProgress.configure({
  easing: 'ease', // 动画方式
  speed: 500, // 递增进度条的速度
  showSpinner: false, // 是否显示加载ico
  trickleSpeed: 200, // 自动递增间隔
  minimum: 0.3, // 初始化时的最小百分比
})

// 动态路由的名称列表
const dynamicRouteNames = ref<string[]>([])

const router = createRouter({
  history: createWebHistory(import.meta.env.VITE_STATIC_URL || '/'),
  routes: staticRoutes,
})

router.beforeEach(async (to) => {
  NProgress.start()

  // 页面专用文案（locales/route/<路由第一段>/）在进入页面前按需加载
  await ensureRouteLocales(to.path).catch(() => undefined)

  // 公开路由无需鉴权，直接放行
  if (to.meta?.public) return true

  const token = localStorage.getItem('accessToken')

  // 未登录：先检查初始化状态
  if (!token) {
    if (to.path === '/initialize') return true
    try {
      const { data } = await getInitializeStatus()
      if (data && !data.initialized) {
        return { path: '/initialize', replace: true }
      }
    } catch {
      // 接口不可用时忽略，走正常登录流程
    }
    if (to.path !== '/login') return buildLoginRoute(to.fullPath)
    return true
  }

  const menuStore = useMenuStore()

  // 仅要求登录的静态路由不依赖菜单权限，例如 OIDC 授权确认页。
  if (to.meta?.authOnly) return true

  const nodeStore = useNodeStore()

  // 首次加载：初始化动态路由
  if (!menuStore.hasLoadedPermissions) {
    await menuStore.getUserPermissions()
    // 远程节点上下文：拉取节点状态与生效权限，供菜单 / 按钮 / 页面按节点授权过滤。
    await nodeStore.loadContext()
    const dynamicRoutes = menuToRoute(menuStore.menuList)

    // 如果没有动态路由，则跳转到403页面
    if (!dynamicRoutes.length) return { name: '403' }

    // 添加动态路由（在 404 之前添加，这样 404 只匹配真正不存在的路由）
    dynamicRoutes.forEach((route) => {
      router.addRoute('layout', route)
      if (route.name) dynamicRouteNames.value.push(route.name as string)
    })

    // 访问根路径，重定向到第一个菜单项
    if (to.fullPath === '/') return { name: dynamicRoutes[0]?.name as string }

    // 其他情况：使用 redirect 路由作为中间层，确保动态路由加载后再跳转（暂时注释掉，因为redirect路由会导致加载缓慢）
    // return {
    //   path: `/redirect${to.fullPath}`,
    //   query: to.query,
    //   hash: to.hash,
    // }

    // 直接跳转到目标路径，确保动态路由加载后再跳转
    return {
      path: to.path,
      query: to.query,
      hash: to.hash,
      replace: true,
    }
  }

  // 已加载：正常处理
  // 已登录用户访问初始化页面，重定向到首页
  if (to.path === '/initialize') {
    return { path: '/', replace: true }
  }

  // 访问 403 / 404 等异常页时，直接放行（此时权限已加载，403 是真的没有权限）
  if (to.name === '403' || to.name === '404') {
    return true
  }

  // 远程节点未授予该页面所需权限（或页面不支持远程节点）：不进入页面，回到工作台。
  if (!nodeStore.canAccessRoute(to.path)) {
    feedback.warning(translate('node.context.routeForbidden'))
    return { path: '/dashboard/home', replace: true }
  }

  // 主题编辑器的预览框里要能看到访客页面（登录页等，路由 meta.siteTheme）
  if (IS_THEME_PREVIEW_FRAME && to.meta?.siteTheme) return true

  // 访问登录页：重定向到第一个菜单项
  if (to.path === '/login') {
    const firstRoute = menuStore.menuList?.[0]
    // 如果第一个菜单项存在，则重定向到第一个菜单项
    if (firstRoute) return firstRoute.path
    // 如果第一个菜单项不存在，则重定向到 403 页面
    return { name: '403' }
  }

  return true
})

router.afterEach((to) => {
  NProgress.done()

  // 添加标签页
  const tabsStore = useTabsStore()
  tabsStore.addTab(to)
})

// 重置路由(清除动态路由)
const resetRouter = () => {
  dynamicRouteNames.value.forEach((name) => {
    router.removeRoute(name)
  })
  dynamicRouteNames.value = []
}

export { resetRouter }

export default router
