import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { setExpiredHandler, tokenStore } from '@passport/api/request'
import { useSessionStore } from '@passport/stores/session'

declare module 'vue-router' {
  interface RouteMeta {
    /** 不需要登录 */
    public?: boolean
    /** 外壳导航里高亮哪一项 */
    nav?: string
    /** 手机上页面自带头部与导航（外壳不显示） */
    ownMobileChrome?: boolean
  }
}

// 通行证的页面：首页、账号与安全、绑定、消息、我的游戏与游戏空间、OIDC 授权确认；登录、注册与找回密码。
export const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('@passport/views/auth/LoginView.vue'), meta: { public: true } },
  {
    path: '/register',
    name: 'register',
    component: () => import('@passport/views/auth/RegisterView.vue'),
    meta: { public: true },
  },
  { path: '/forgot', name: 'forgot', component: () => import('@passport/views/auth/ForgotView.vue'), meta: { public: true } },
  { path: '/oidc/authorize', name: 'oidc', component: () => import('@passport/views/auth/OidcView.vue') },
  {
    path: '/',
    component: () => import('@passport/layouts/PassportShell.vue'),
    children: [
      { path: '', name: 'home', component: () => import('@passport/views/HomeView.vue'), meta: { nav: 'home' } },
      {
        path: 'account',
        name: 'account',
        component: () => import('@passport/views/account/AccountView.vue'),
        meta: { nav: 'account' },
      },
      {
        path: 'bind',
        name: 'bind',
        component: () => import('@passport/views/BindView.vue'),
        meta: { nav: 'games', ownMobileChrome: true },
      },
      { path: 'messages', name: 'messages', component: () => import('@passport/views/MessagesView.vue'), meta: { nav: 'messages' } },
      { path: 'games', name: 'games', component: () => import('@passport/views/GamesView.vue'), meta: { nav: 'games' } },
      {
        path: 'games/:serverId',
        name: 'game',
        component: () => import('@passport/views/game/GameSpace.vue'),
        meta: { nav: 'games', ownMobileChrome: true },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.VITE_STATIC_URL || '/'),
  routes,
  scrollBehavior: (to, from, saved) => {
    if (saved) return saved
    if (to.path === from.path) return false
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const session = useSessionStore()
  const authed = !!tokenStore.access()
  if (to.meta.public) {
    if (authed && (to.name === 'login' || to.name === 'register')) return { path: String(to.query.redirect || '/') }
    return true
  }
  if (!authed) return { name: 'login', query: { redirect: to.fullPath } }
  if (!session.me) {
    try {
      await session.loadMe()
    } catch {
      if (!tokenStore.access()) return { name: 'login', query: { redirect: to.fullPath } }
    }
  }
  return true
})

setExpiredHandler(() => {
  const session = useSessionStore()
  session.clear()
  const current = router.currentRoute.value
  if (!current.meta.public) void router.replace({ name: 'login', query: { redirect: current.fullPath } })
})

export default router
