/**
 * name: route name should stay unique.
 * meta.keepAlive: whether the route should be cached.
 * meta.title / meta.titleKey: page name (titleKey is an i18n key and wins); also shown in the theme editor.
 * meta.theme: false hides the page from the theme editor (error pages, the editor itself, setup wizard).
 * meta.siteTheme: true marks a visitor page that uses the site default theme (login).
 * Every other page (static or from menus) is listed in the theme editor automatically, see theme/pages.ts.
 */
import type { RouteComponent } from 'vue-router'

// 游戏页面的子路由（服务器详情、玩家详情…）：视图文件按需加载，文件不存在时落到 404 视图（同 menuToRoute）。
const gameViews = import.meta.glob('@/views/game/**/index.vue')
const gameView = (path: string) =>
  (gameViews[`/src/views/game/${path}/index.vue`] || (() => import('@/views/exception/404/index.vue'))) as RouteComponent
export const staticRoutes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/index.vue'),
    meta: { keepAlive: false, titleKey: 'theme.routes.login', siteTheme: true },
  },
  {
    path: '/initialize',
    name: 'initialize',
    component: () => import('@/views/initialize/index.vue'),
    meta: { keepAlive: false, theme: false },
  },
  {
    path: '/public/share/:token',
    name: 'PublicShare',
    component: () => import('@/views/public/share/index.vue'),
    meta: { keepAlive: false, public: true, titleKey: 'theme.routes.publicShare' },
  },
  {
    path: '/public/iqcheck/:slug',
    name: 'PublicIQCheck',
    component: () => import('@/views/public/iqcheck/index.vue'),
    meta: { keepAlive: false, public: true, titleKey: 'theme.routes.publicIqcheck' },
  },
  {
    path: '/oidc/authorize',
    name: 'OIDCAuthorize',
    component: () => import('@/views/public/oidc/authorize/index.vue'),
    meta: { keepAlive: false, authOnly: true, titleKey: 'theme.routes.oidcAuthorize' },
  },
  {
    // 在线主题编辑器：独立全屏页（无外壳），id 为「我的主题」id 或 new（从当前外观新建）
    path: '/profile/theme/editor/:id',
    name: 'ThemeEditor',
    component: () => import('@/views/theme-editor/index.vue'),
    meta: { title: '主题编辑器', keepAlive: false, theme: false },
  },
  {
    path: '/',
    name: 'layout',
    component: () => import('@/layouts/index.vue'),
    children: [
      {
        path: '/profile',
        name: 'ProfileView',
        component: () => import('@/views/profile/index.vue'),
        meta: { title: '个人中心', titleKey: 'theme.routes.profile', icon: 'HOutline:UserCircleIcon', keepAlive: true },
      },
      {
        path: '/openssh/management',
        name: 'OpensshManagementView',
        component: () => import('@/views/openssh/management/index.vue'),
        meta: { title: 'SSH 管理', titleKey: 'theme.routes.sshManage', icon: 'HOutline:ServerIcon', keepAlive: true },
      },
      {
        path: '/openssh/terminal',
        name: 'OpensshTerminalView',
        component: () => import('@/views/openssh/terminal/index.vue'),
        // hideTabbar：手机端整屏页面（原型无底部导航），见 AppShell / AppContent
        meta: { title: 'SSH 终端', titleKey: 'theme.routes.sshTerminal', icon: 'HOutline:CommandLineIcon', keepAlive: true, fullBleed: true, hideTabbar: true },
      },
      {
        // 接入游戏服务器（选择游戏与接入方式）
        path: '/game/servers/new',
        name: 'GameServerNewView',
        component: gameView('servers/new'),
        meta: { title: '接入游戏服务器', titleKey: 'theme.routes.gameServerNew', icon: 'HOutline:ServerStackIcon', keepAlive: false },
      },
      {
        // 游戏服务器详情：概览 / 控制台 / 专用设置 / 更新 / 备份 / 文件 / 日志，?tab= 切换
        path: '/game/servers/:id',
        name: 'GameServerDetailView',
        component: gameView('servers/detail'),
        meta: { title: '游戏服务器', titleKey: 'theme.routes.gameServerDetail', icon: 'HOutline:ServerStackIcon', keepAlive: true },
      },
      {
        // GM 运营（带服务器）：与菜单 /game/gm 是同一个页面
        path: '/game/gm/:serverId',
        name: 'GameGmServerView',
        component: gameView('gm'),
        meta: { title: 'GM 运营', titleKey: 'theme.routes.gameGm', icon: 'HOutline:SparklesIcon', keepAlive: true },
      },
      {
        // 玩家详情：/game/gm/:serverId/players/:pid?tab=…
        path: '/game/gm/:serverId/players/:pid',
        name: 'GameGmPlayerView',
        component: gameView('gm/player'),
        meta: { title: '玩家详情', titleKey: 'theme.routes.gameGmPlayer', icon: 'HOutline:UserIcon', keepAlive: true },
      },
      {
        // 应用详情：控制台 / 文件 / 配置 在同一页，?tab= 切换
        path: '/instance/app/:instanceId',
        name: 'InstanceAppView',
        component: () => import('@/views/instance/console/index.vue'),
        meta: { title: '应用详情', titleKey: 'theme.routes.appDetail', icon: 'HOutline:CommandLineIcon', keepAlive: true, fullBleed: true, hideTabbar: true },
      },
    ],
  },
  // 403 / 404 是无外壳的独立错误页（菜单指向的视图文件缺失时，menuToRoute 仍会把 404 视图放进外壳里兜底）
  {
    path: '/exception/403',
    name: '403',
    component: () => import('@/views/exception/403/index.vue'),
    meta: { title: '403', keepAlive: false, theme: false },
  },
  {
    path: '/:pathMatch(.*)*',
    name: '404',
    component: () => import('@/views/exception/404/index.vue'),
    meta: { title: '404', keepAlive: false, theme: false },
  },
]
