/**
 * name: route name should stay unique.
 * meta.keepAlive: whether the route should be cached.
 */
export const staticRoutes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/index.vue'),
    meta: { keepAlive: false },
  },
  {
    path: '/initialize',
    name: 'initialize',
    component: () => import('@/views/initialize/index.vue'),
    meta: { keepAlive: false },
  },
  {
    path: '/public/share/:token',
    name: 'PublicShare',
    component: () => import('@/views/public/share/index.vue'),
    meta: { keepAlive: false, public: true },
  },
  {
    path: '/public/iqcheck/:slug',
    name: 'PublicIQCheck',
    component: () => import('@/views/public/iqcheck/index.vue'),
    meta: { keepAlive: false, public: true },
  },
  {
    path: '/oidc/authorize',
    name: 'OIDCAuthorize',
    component: () => import('@/views/public/oidc/authorize/index.vue'),
    meta: { keepAlive: false, authOnly: true },
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
        meta: { title: '个人中心', icon: 'HOutline:UserCircleIcon', keepAlive: true },
      },
      {
        path: '/openssh/management',
        name: 'OpensshManagementView',
        component: () => import('@/views/openssh/management/index.vue'),
        meta: { title: 'SSH 管理', icon: 'HOutline:ServerIcon', keepAlive: true },
      },
      {
        path: '/openssh/terminal',
        name: 'OpensshTerminalView',
        component: () => import('@/views/openssh/terminal/index.vue'),
        meta: { title: 'SSH 终端', icon: 'HOutline:CommandLineIcon', keepAlive: true, fullBleed: true },
      },
      {
        // 应用详情：控制台 / 文件 / 配置 在同一页，?tab= 切换
        path: '/instance/app/:instanceId',
        name: 'InstanceAppView',
        component: () => import('@/views/instance/console/index.vue'),
        meta: { title: '应用详情', icon: 'HOutline:CommandLineIcon', keepAlive: true, fullBleed: true },
      },
      {
        path: '/exception/403',
        name: '403',
        component: () => import('@/views/exception/403/index.vue'),
        meta: { title: '403', icon: 'HOutline:NoSymbolIcon', keepAlive: true },
      },
      {
        path: '/:pathMatch(.*)*',
        name: '404',
        component: () => import('@/views/exception/404/index.vue'),
        meta: { title: '404', icon: 'HOutline:QuestionMarkCircleIcon', keepAlive: true },
      },
    ],
  },
]
