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
} as const

export type PermissionKey = (typeof PERM)[keyof typeof PERM]
