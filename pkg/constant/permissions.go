package constant

type Permissions string

const (
	// Dev 开发权限
	Dev Permissions = "dev"

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

	// SystemConfigView 允许查看系统配置。
	SystemConfigView Permissions = "system_config:view"
	// SystemConfigEdit 允许编辑系统配置。
	SystemConfigEdit Permissions = "system_config:edit"

	// Terminal 终端页面
	Terminal Permissions = "terminal"

	// ApiKeyAdd api key 添加
	ApiKeyAdd    Permissions = "api_key:add"
	ApiKeyEdit   Permissions = "api_key:edit"
	ApiKeyDelete Permissions = "api_key:delete"
	ApiKeyView   Permissions = "api_key:view"

	// DockerView 允许查看 Docker 资源。
	DockerView Permissions = "docker:view"
	// DockerConfigEdit 允许编辑 Docker 配置。
	DockerConfigEdit Permissions = "docker_config:edit"
	// DockerContainerManage 允许管理 Docker 容器。
	DockerContainerManage Permissions = "docker_container:manage"
	// DockerImageManage 允许管理 Docker 镜像。
	DockerImageManage Permissions = "docker_image:manage"
	// DockerNetworkManage 允许管理 Docker 网络。
	DockerNetworkManage Permissions = "docker_network:manage"
	// DockerVolumeManage 允许管理 Docker 储存卷。
	DockerVolumeManage Permissions = "docker_volume:manage"

	// Sub2APIView 允许查看 Sub2API 用量与配置。
	Sub2APIView Permissions = "sub2api:view"
	// Sub2APIEdit 允许编辑 Sub2API 配置、公告与时间线。
	Sub2APIEdit Permissions = "sub2api:edit"

	// Instance 允许管理应用（实例），含实例增删改、终端与实例文件操作。
	Instance Permissions = "instance:manage"
	// Network 允许管理网络（端口转发等）。
	Network Permissions = "network:manage"
)
