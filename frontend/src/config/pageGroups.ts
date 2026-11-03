import { computed, defineAsyncComponent, ref, watch, type Component } from 'vue'
import { useRoute } from 'vue-router'

/**
 * 「一个菜单 = 一个页面」里的标签页分组。
 *
 * 原型里 Docker、节点与 API Key、用户与权限、任务与日志、系统设置、智商检测都是单个页面，
 * 页内用下划线标签切换（地址形如 /docker?tab=image）。每个分组只有一个菜单路由，
 * 标签是这个页面里的子视图：切换标签不换路由、不重新挂载，已打开的标签保留筛选、滚动与未保存的表单。
 *
 * 标签可见性：
 * - perm：需要的权限码（来自该菜单下的按钮权限项，同时控制对应接口）；为空表示有菜单即可见；
 * - nodePath：远程节点上下文中用于判断节点授权的虚拟路径（见 config/nodePermission.ts），默认 `${path}/${key}`。
 */
export interface PageTab {
  key: string
  labelKey: string
  perm?: string
  nodePath?: string
  component: Component
}

export interface PageGroup {
  path: string
  labelKey: string
  tabs: PageTab[]
}

const lazy = (loader: () => Promise<unknown>) =>
  defineAsyncComponent(loader as () => Promise<{ default: Component }>)

export const PAGE_GROUPS = {
  docker: {
    path: '/docker',
    labelKey: 'pageGroup.docker.label',
    tabs: [
      { key: 'container', labelKey: 'pageGroup.docker.container', component: lazy(() => import('@/views/docker/container/index.vue')) },
      { key: 'image', labelKey: 'pageGroup.docker.image', component: lazy(() => import('@/views/docker/image/index.vue')) },
      { key: 'network', labelKey: 'pageGroup.docker.network', component: lazy(() => import('@/views/docker/network/index.vue')) },
      { key: 'config', labelKey: 'pageGroup.docker.config', component: lazy(() => import('@/views/docker/config/index.vue')) },
    ],
  },
  node: {
    path: '/node',
    labelKey: 'pageGroup.node.label',
    tabs: [
      { key: 'list', labelKey: 'pageGroup.node.list', perm: 'node:view', component: lazy(() => import('@/views/node/list/index.vue')) },
      { key: 'key', labelKey: 'pageGroup.node.key', perm: 'api_key:view', component: lazy(() => import('@/views/node/key/index.vue')) },
    ],
  },
  access: {
    path: '/system/access',
    labelKey: 'pageGroup.access.label',
    tabs: [
      { key: 'user', labelKey: 'pageGroup.access.user', perm: 'user:view', component: lazy(() => import('@/views/system/user/index.vue')) },
      { key: 'role', labelKey: 'pageGroup.access.role', perm: 'role:view', component: lazy(() => import('@/views/system/role/index.vue')) },
      { key: 'menu', labelKey: 'pageGroup.access.menu', perm: 'menu:view', component: lazy(() => import('@/views/system/menu/index.vue')) },
    ],
  },
  logs: {
    path: '/system/logs',
    labelKey: 'pageGroup.logs.label',
    tabs: [
      { key: 'task', labelKey: 'pageGroup.logs.task', perm: 'task:manage', component: lazy(() => import('@/views/system/task/index.vue')) },
      { key: 'operation', labelKey: 'pageGroup.logs.operation', perm: 'system_config:view', component: lazy(() => import('@/views/system/operation/index.vue')) },
    ],
  },
  settings: {
    path: '/system/settings',
    labelKey: 'pageGroup.settings.label',
    tabs: [
      { key: 'general', labelKey: 'pageGroup.settings.general', perm: 'system_config:view', component: lazy(() => import('@/views/system/settings/general/index.vue')) },
      { key: 'oidc', labelKey: 'pageGroup.settings.oidc', perm: 'oidc:view', component: lazy(() => import('@/views/system/oidc/index.vue')) },
    ],
  },
  iqcheck: {
    path: '/iqcheck',
    labelKey: 'pageGroup.iqcheck.label',
    tabs: [
      { key: 'site', labelKey: 'pageGroup.iqcheck.site', component: lazy(() => import('@/views/iqcheck/site/index.vue')) },
      { key: 'prompt', labelKey: 'pageGroup.iqcheck.prompt', component: lazy(() => import('@/views/iqcheck/prompt/index.vue')) },
    ],
  },
} satisfies Record<string, PageGroup>

export type PageGroupKey = keyof typeof PAGE_GROUPS

/** 生成某个分组标签的地址（带上其它额外查询参数时传 extra）。 */
export const pageTabTo = (group: PageGroupKey, tab: string, extra?: Record<string, string>) => ({
  path: PAGE_GROUPS[group].path,
  query: { tab, ...(extra || {}) },
})

/** 当前分组里可见的标签与当前标签（无效或无权限的 ?tab= 回落到第一个可见标签）。 */
export const usePageGroupTabs = (group: PageGroupKey) => {
  const route = useRoute()
  const menuStore = useMenuStore()
  const nodeStore = useNodeStore()
  const def: PageGroup = PAGE_GROUPS[group]

  const tabs = computed(() =>
    def.tabs.filter(
      (tab) =>
        (!tab.perm || menuStore.hasButtonPermission(tab.perm)) &&
        nodeStore.canAccessRoute(tab.nodePath || `${def.path}/${tab.key}`),
    ),
  )
  // 只在地址属于本分组时跟随 ?tab=；页面被缓存、地址已是别的页面时保持原标签，避免后台切换标签
  const wanted = ref('')
  watch(
    () => [route.path, route.query.tab] as const,
    ([path, tab]) => {
      if (path === def.path) wanted.value = typeof tab === 'string' ? tab : ''
    },
    { immediate: true },
  )
  const current = computed<PageTab | undefined>(
    () => tabs.value.find((tab) => tab.key === wanted.value) || tabs.value[0],
  )
  return { def, tabs, current }
}
