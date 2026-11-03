/**
 * 节点权限目录：可授予 API Key 的权限（与后端 internal/biz/node_permission.go 一致），
 * 以及“按节点区分”的页面在远程节点上所需的权限。
 */

export interface NodePermissionItem {
  code: string
  /** i18n key */
  labelKey: string
  /** 本地仅需登录、不走 RBAC 的权限：本机上下文恒为允许 */
  loginOnly?: boolean
}

export interface NodePermissionGroup {
  key: string
  labelKey: string
  items: NodePermissionItem[]
}

export const NODE_PERMISSION_GROUPS: NodePermissionGroup[] = [
  {
    key: 'status',
    labelKey: 'node.permission.groups.status',
    items: [
      { code: 'system:status', labelKey: 'node.permission.items.systemStatus', loginOnly: true },
    ],
  },
  {
    key: 'app',
    labelKey: 'node.permission.groups.app',
    items: [{ code: 'instance:manage', labelKey: 'node.permission.items.instance' }],
  },
  {
    key: 'file',
    labelKey: 'node.permission.groups.file',
    items: [
      { code: 'terminal', labelKey: 'node.permission.items.file' },
      { code: 'file:source', labelKey: 'node.permission.items.fileSource' },
    ],
  },
  {
    key: 'docker',
    labelKey: 'node.permission.groups.docker',
    items: [
      { code: 'docker:view', labelKey: 'node.permission.items.dockerView' },
      { code: 'docker_container:manage', labelKey: 'node.permission.items.dockerContainer' },
      { code: 'docker_image:manage', labelKey: 'node.permission.items.dockerImage' },
      { code: 'docker_network:manage', labelKey: 'node.permission.items.dockerNetwork' },
      { code: 'docker_config:edit', labelKey: 'node.permission.items.dockerConfig' },
    ],
  },
  {
    key: 'tools',
    labelKey: 'node.permission.groups.tools',
    items: [
      { code: 'network:manage', labelKey: 'node.permission.items.portForward' },
      { code: 'tunnel:manage', labelKey: 'node.permission.items.tunnel' },
      { code: 'ssh:manage', labelKey: 'node.permission.items.ssh', loginOnly: true },
    ],
  },
  {
    key: 'task',
    labelKey: 'node.permission.groups.task',
    items: [{ code: 'task:manage', labelKey: 'node.permission.items.task' }],
  },
]

export const NODE_PERMISSION_ITEMS = NODE_PERMISSION_GROUPS.flatMap((group) => group.items)

export const NODE_PERMISSION_CODES = new Set(NODE_PERMISSION_ITEMS.map((item) => item.code))

export const LOGIN_ONLY_NODE_PERMISSIONS = new Set(
  NODE_PERMISSION_ITEMS.filter((item) => item.loginOnly).map((item) => item.code),
)

export const nodePermissionLabelKey = (code: string) =>
  NODE_PERMISSION_ITEMS.find((item) => item.code === code)?.labelKey

/**
 * 按节点区分的页面（路径前缀） → 远程节点上所需权限：
 * - ''   页面始终可见，内容在页面内按权限分块显示（工作台）；
 * - null 页面依赖主控本机上下文，远程节点上不可用（分享链接是主控域名下的公开地址）。
 * 不在表中的页面是全局页面（系统、用户、节点管理等），始终操作主控本机。
 */
const NODE_ROUTE_PERMISSIONS: Array<[string, string | null]> = [
  ['/dashboard/home', ''],
  ['/instance/list', 'instance:manage'],
  ['/instance/type', 'instance:manage'],
  ['/instance/app', 'instance:manage'],
  ['/file/index', 'terminal'],
  ['/file/source', 'file:source'],
  ['/file/share', null],
  ['/docker', 'docker:view'],
  ['/tools/port-forward', 'network:manage'],
  ['/tools/tunnel', 'tunnel:manage'],
  ['/openssh', 'ssh:manage'],
  ['/system/logs/task', 'task:manage'],
]

/** 返回页面在远程节点上所需权限；undefined 表示全局页面。 */
export const resolveNodeRoutePermission = (path: string): string | null | undefined => {
  for (const [prefix, perm] of NODE_ROUTE_PERMISSIONS) {
    if (path === prefix || path.startsWith(`${prefix}/`)) return perm
  }
  return undefined
}
