/**
 * 按钮权限标识统一管理
 *
 * 命名规则：模块名:操作  (与后端权限标识保持一致)
 */

export const PERM = {
  // ---- 用户管理 ----
  USER_ADD: 'user:add',
  USER_EDIT: 'user:edit',
  USER_DELETE: 'user:delete',

  // ---- 角色管理 ----
  ROLE_ADD: 'role:add',
  ROLE_EDIT: 'role:edit',
  ROLE_DELETE: 'role:delete',

  // ---- 菜单管理 ----
  MENU_ADD: 'menu:add',
  MENU_EDIT: 'menu:edit',
  MENU_DELETE: 'menu:delete',

  // ---- 系统配置 ----
  SYSTEM_CONFIG_VIEW: 'system_config:view',
  SYSTEM_CONFIG_EDIT: 'system_config:edit',
  OIDC_EDIT: 'oidc:edit',

  // ---- Docker 管理 ----
  DOCKER_VIEW: 'docker:view',
  DOCKER_CONFIG_EDIT: 'docker_config:edit',
  DOCKER_CONTAINER_MANAGE: 'docker_container:manage',
  DOCKER_IMAGE_MANAGE: 'docker_image:manage',
  DOCKER_NETWORK_MANAGE: 'docker_network:manage',

	// ---- 工具 ----
	NETWORK_MANAGE: 'network:manage',
} as const

export type PermissionKey = (typeof PERM)[keyof typeof PERM]
