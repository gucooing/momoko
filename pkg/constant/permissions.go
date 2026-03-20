package constant

type Permissions string

const (
	// UserAdd 允许添加用户。
	UserAdd Permissions = "user:add"
	// UserEdit 允许编辑用户。
	UserEdit Permissions = "user:edit"
	// UserDelete 允许删除用户。
	UserDelete Permissions = "user:delete"
	// UserView 允许查看用户。
	UserView Permissions = "user:view"

	// RoleAdd 允许添加角色。
	RoleAdd Permissions = "role:add"
	// RoleEdit 允许编辑角色。
	RoleEdit Permissions = "role:edit"
	// RoleDelete 允许删除角色。
	RoleDelete Permissions = "role:delete"
	// RoleView 允许查看角色。
	RoleView Permissions = "role:view"

	// MenuAdd 允许添加菜单。
	MenuAdd Permissions = "menu:add"
	// MenuEdit 允许编辑菜单。
	MenuEdit Permissions = "menu:edit"
	// MenuDelete 允许删除菜单。
	MenuDelete Permissions = "menu:delete"
	// MenuView 允许查看菜单。
	MenuView Permissions = "menu:view"

	// Terminal 终端页面
	Terminal Permissions = "terminal"
)
