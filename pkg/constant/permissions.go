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
	// ApiKeyManage 允许查看全部用户的 API Key，并吊销 / 延期 / 删除他人的密钥（不能复制或刷新他人的密钥）。
	ApiKeyManage Permissions = "api_key:manage"

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

	// Instance 允许管理应用（实例），含实例增删改、终端与实例文件操作。
	Instance Permissions = "instance:manage"
	// Network 允许管理网络（端口转发等）。
	Network Permissions = "network:manage"
	// Tunnel 允许管理内网穿透（frp 隧道）。
	Tunnel Permissions = "tunnel:manage"
	// FileShare 允许管理文件分享。
	FileShare Permissions = "file:share"
	// FileSource 允许管理文件来源（OSS/FTP/WebDAV 等网络来源）。
	FileSource Permissions = "file:source"
	// TaskManage 允许查看与操作任务管理器中的全部任务。
	TaskManage Permissions = "task:manage"
	// SystemUpdate 允许执行更新检查（查询远程发行版并比较版本）。
	SystemUpdate Permissions = "system:update"
	// OIDCView 允许查看 OIDC 服务端配置与客户端。
	OIDCView Permissions = "oidc:view"
	// OIDCEdit 允许编辑 OIDC 服务端配置与客户端。
	OIDCEdit Permissions = "oidc:edit"
	// MonitorView 允许查看性能监控（历史曲线、磁盘趋势、应用占用排行）。
	MonitorView Permissions = "monitor:view"
	// MonitorEdit 允许修改性能监控设置与清空历史数据。
	MonitorEdit Permissions = "monitor:edit"
)

const (
	// NodeView 允许查看节点列表并切换到远程节点。
	NodeView Permissions = "node:view"
	// NodeManage 允许添加、编辑、删除和测试远程节点。
	NodeManage Permissions = "node:manage"

	// 以下两项本地仅需登录、不进 RBAC 菜单，只作为 API Key 的授权范围：
	// 远程主控经 API Key 调用对应功能时，必须显式授予。

	// SystemStatus 允许远程查看机器状态（CPU / 内存 / 磁盘 / 网络）。
	SystemStatus Permissions = "system:status"
	// SSHManage 允许远程管理 SSH 主机与使用 Web SSH 终端。
	SSHManage Permissions = "ssh:manage"
)

const (
	// GameServer 允许管理游戏服务器（接入、托管、专用设置、更新与备份）。
	GameServer Permissions = "game:server"
	// GameGM 允许进入 GM 运营：能进的管理员可以对任何服务器执行全部 GM 指令，并处理玩家自助申请。
	GameGM Permissions = "game:gm"
	// GamePassport 允许管理用户通行证（玩家账号独立于 Momoko 用户）：查看与管理玩家、绑定、邀请与通行证设置。
	GamePassport Permissions = "game:passport"
)

const (
	// IQCheckView 允许查看智商检测的提示词与站点配置。
	IQCheckView Permissions = "iqcheck:view"
	// IQCheckEdit 允许编辑智商检测的提示词、站点配置与设置，并手动触发检测。
	IQCheckEdit Permissions = "iqcheck:edit"
)
