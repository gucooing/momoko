package data

import (
	"context"
	"fmt"

	"momoko/internal/data/ent/gen"
	entmenu "momoko/internal/data/ent/gen/menu"
	entrole "momoko/internal/data/ent/gen/role"
	entuser "momoko/internal/data/ent/gen/user"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
)

type defaultMenu struct {
	ID         string
	Type       entmenu.Type
	Path       string
	Title      string
	Permission constant.Permissions
	Order      int
	Icon       string
	IsSystem   bool
	Status     entmenu.Status
	ParentID   *string
}

// builtinMenuGrant 内置菜单结构调整时的角色权限迁移项。
type builtinMenuGrant struct {
	Target           string
	Source           string
	LegacyPath       string
	LegacyPermission string // 可选：路径未变时用旧权限码识别旧结构
}

type defaultRole struct {
	ID          string
	Name        string
	Code        string
	Description string
	IsBuiltin   bool
	Status      entrole.Status
}

type defaultUser struct {
	ID       string
	Name     string
	Email    string
	Password string
	Username string
	Status   entuser.Status
	Bio      string
	Tags     string
	RoleID   string
}

var (
	builtinAdminUserID          = "admin_1"
	adminPermissionRoleID       = "role_1"
	noPermissionRoleID          = "role_2"
	builtinRoleWithMenuPermsIDs = []string{"role_1"}
	// 内置菜单与原型导航一一对应：同一页面里的标签（如 Docker 的容器/镜像/网络/连接配置）是这一个菜单下的
	// 「按钮」权限项，页面用 ?tab= 切换；按钮的权限码同时决定标签是否可见与对应接口能否调用。
	// 内置菜单由代码定义，启动时同步结构、标题、图标与排序（保留启用状态）。
	builtinDefaultMenus = []defaultMenu{
		newDefaultMenu("menu_1", entmenu.TypeDirectory, "", "主页", "HOutline:HomeIcon", nil, 0, "", entmenu.StatusActive),
		newDefaultMenu("menu_1_1", entmenu.TypeMenu, "/dashboard/home", "工作台", "HOutline:ComputerDesktopIcon", new("menu_1"), 0, "", entmenu.StatusActive),
		// 性能监控：持久化的历史曲线、磁盘趋势与应用占用排行；按钮项控制「监控设置」与「清空历史」
		newDefaultMenu("menu_1_2", entmenu.TypeMenu, "/monitor", "性能监控", "HOutline:PresentationChartLineIcon", new("menu_1"), 1, constant.MonitorView, entmenu.StatusActive),
		newDefaultMenu("menu_1_2_button_0", entmenu.TypeButton, "", "修改监控设置", "", new("menu_1_2"), 0, constant.MonitorEdit, entmenu.StatusActive),

		newDefaultMenu("menu_2", entmenu.TypeDirectory, "", "应用", "HOutline:ServerStackIcon", nil, 1, "", entmenu.StatusActive),
		newDefaultMenu("menu_2_1", entmenu.TypeMenu, "/instance/list", "应用列表", "HOutline:Squares2X2Icon", new("menu_2"), 0, constant.Instance, entmenu.StatusActive),
		newDefaultMenu("menu_2_2", entmenu.TypeMenu, "/instance/type", "应用类型", "HOutline:RectangleStackIcon", new("menu_2"), 1, "", entmenu.StatusActive),

		// 游戏：游戏服务器（托管 / 仅 GM）、GM 运营（服务器切换 + 页内标签）、用户通行证（玩家账号独立于 Momoko 用户）
		newDefaultMenu("menu_13", entmenu.TypeDirectory, "", "游戏", "HOutline:PuzzlePieceIcon", nil, 2, "", entmenu.StatusActive),
		newDefaultMenu("menu_13_1", entmenu.TypeMenu, "/game/servers", "游戏服务器", "HOutline:ServerStackIcon", new("menu_13"), 0, constant.GameServer, entmenu.StatusActive),
		newDefaultMenu("menu_13_2", entmenu.TypeMenu, "/game/gm", "GM 运营", "HOutline:SparklesIcon", new("menu_13"), 1, constant.GameGM, entmenu.StatusActive),
		newDefaultMenu("menu_13_3", entmenu.TypeMenu, "/game/passport", "用户通行证", "HOutline:IdentificationIcon", new("menu_13"), 2, constant.GamePassport, entmenu.StatusActive),

		// 文件：一个页面，标签 文件管理 / 分享 / 文件源（PC 侧边栏把标签展开成三个入口）
		newDefaultMenu("menu_3", entmenu.TypeDirectory, "", "文件", "HOutline:InboxStackIcon", nil, 3, "", entmenu.StatusActive),
		newDefaultMenu("menu_3_1", entmenu.TypeMenu, "/file", "文件", "HOutline:FolderIcon", new("menu_3"), 0, "", entmenu.StatusActive),
		newDefaultMenu("menu_3_1_button_0", entmenu.TypeButton, "", "文件管理", "", new("menu_3_1"), 0, constant.Terminal, entmenu.StatusActive),
		newDefaultMenu("menu_3_2", entmenu.TypeButton, "", "分享", "", new("menu_3_1"), 1, constant.FileShare, entmenu.StatusActive),
		newDefaultMenu("menu_3_3", entmenu.TypeButton, "", "文件源", "", new("menu_3_1"), 2, constant.FileSource, entmenu.StatusActive),

		// Docker：一个页面，标签 容器 / 镜像 / 网络 / 连接配置
		newDefaultMenu("menu_5", entmenu.TypeDirectory, "", "Docker", "HOutline:CubeIcon", nil, 4, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_1", entmenu.TypeMenu, "/docker", "容器 · 镜像 · 网络", "HOutline:CubeIcon", new("menu_5"), 0, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_1_button_0", entmenu.TypeButton, "", "管理容器", "", new("menu_5_1"), 0, constant.DockerContainerManage, entmenu.StatusActive),
		newDefaultMenu("menu_5_2_button_0", entmenu.TypeButton, "", "管理镜像", "", new("menu_5_1"), 1, constant.DockerImageManage, entmenu.StatusActive),
		newDefaultMenu("menu_5_3_button_0", entmenu.TypeButton, "", "管理网络", "", new("menu_5_1"), 2, constant.DockerNetworkManage, entmenu.StatusActive),
		newDefaultMenu("menu_5_5_button_0", entmenu.TypeButton, "", "编辑Docker配置", "", new("menu_5_1"), 3, constant.DockerConfigEdit, entmenu.StatusActive),

		// 工具
		newDefaultMenu("menu_6", entmenu.TypeDirectory, "", "工具", "HOutline:WrenchScrewdriverIcon", nil, 5, "", entmenu.StatusActive),
		newDefaultMenu("menu_6_1", entmenu.TypeMenu, "/tools/port-forward", "端口转发", "HOutline:PaperAirplaneIcon", new("menu_6"), 0, constant.Network, entmenu.StatusActive),
		newDefaultMenu("menu_6_2", entmenu.TypeMenu, "/tools/tunnel", "内网穿透", "HOutline:GlobeAltIcon", new("menu_6"), 1, constant.Tunnel, entmenu.StatusActive),
		newDefaultMenu("menu_6_3", entmenu.TypeMenu, "/openssh/management", "SSH 终端", "HOutline:CommandLineIcon", new("menu_6"), 2, "", entmenu.StatusActive),
		// 智商检测：一个页面，标签 检测站点 / 提示词
		newDefaultMenu("menu_8_1", entmenu.TypeMenu, "/iqcheck", "智商检测", "HOutline:SparklesIcon", new("menu_6"), 3, constant.IQCheckView, entmenu.StatusActive),
		newDefaultMenu("menu_8_2_button_0", entmenu.TypeButton, "", "编辑检测配置", "", new("menu_8_1"), 0, constant.IQCheckEdit, entmenu.StatusActive),
		newDefaultMenu("menu_8_1_button_0", entmenu.TypeButton, "", "编辑提示词", "", new("menu_8_1"), 1, constant.IQCheckEdit, entmenu.StatusActive),

		// 管理
		newDefaultMenu("menu_12", entmenu.TypeDirectory, "", "管理", "HOutline:Cog6ToothIcon", nil, 6, "", entmenu.StatusActive),
		// 节点与 API Key：标签 节点 / API Key
		newDefaultMenu("menu_4_2", entmenu.TypeMenu, "/node", "节点与 API Key", "HOutline:ServerIcon", new("menu_12"), 0, "", entmenu.StatusActive),
		newDefaultMenu("menu_4_2_button_1", entmenu.TypeButton, "", "查看节点", "", new("menu_4_2"), 0, constant.NodeView, entmenu.StatusActive),
		newDefaultMenu("menu_4_2_button_0", entmenu.TypeButton, "", "管理节点", "", new("menu_4_2"), 1, constant.NodeManage, entmenu.StatusActive),
		newDefaultMenu("menu_4_1", entmenu.TypeButton, "", "查看 API Key", "", new("menu_4_2"), 2, constant.ApiKeyView, entmenu.StatusActive),
		newDefaultMenu("menu_4_3", entmenu.TypeButton, "", "管理全部 API Key", "", new("menu_4_2"), 3, constant.ApiKeyManage, entmenu.StatusActive),
		// 用户与权限：标签 用户 / 角色 / 菜单
		newDefaultMenu("menu_12_1", entmenu.TypeMenu, "/system/access", "用户与权限", "HOutline:UserGroupIcon", new("menu_12"), 1, "", entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_3", entmenu.TypeButton, "", "查看用户", "", new("menu_12_1"), 0, constant.UserView, entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_0", entmenu.TypeButton, "", "添加用户", "", new("menu_12_1"), 1, constant.UserAdd, entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_1", entmenu.TypeButton, "", "编辑用户", "", new("menu_12_1"), 2, constant.UserEdit, entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_2", entmenu.TypeButton, "", "删除用户", "", new("menu_12_1"), 3, constant.UserDelete, entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_3", entmenu.TypeButton, "", "查看角色", "", new("menu_12_1"), 4, constant.RoleView, entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_0", entmenu.TypeButton, "", "添加角色", "", new("menu_12_1"), 5, constant.RoleAdd, entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_1", entmenu.TypeButton, "", "编辑角色", "", new("menu_12_1"), 6, constant.RoleEdit, entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_2", entmenu.TypeButton, "", "删除角色", "", new("menu_12_1"), 7, constant.RoleDelete, entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_3", entmenu.TypeButton, "", "查看菜单", "", new("menu_12_1"), 8, constant.MenuView, entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_0", entmenu.TypeButton, "", "添加菜单", "", new("menu_12_1"), 9, constant.MenuAdd, entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_1", entmenu.TypeButton, "", "编辑菜单", "", new("menu_12_1"), 10, constant.MenuEdit, entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_2", entmenu.TypeButton, "", "删除菜单", "", new("menu_12_1"), 11, constant.MenuDelete, entmenu.StatusActive),
		// 任务与日志：标签 后台任务 / 操作记录
		newDefaultMenu("menu_12_6", entmenu.TypeMenu, "/system/logs", "任务与日志", "HOutline:ClockIcon", new("menu_12"), 2, "", entmenu.StatusActive),
		newDefaultMenu("menu_12_6_button_0", entmenu.TypeButton, "", "后台任务", "", new("menu_12_6"), 0, constant.TaskManage, entmenu.StatusActive),
		newDefaultMenu("menu_12_4", entmenu.TypeButton, "", "查看操作记录", "", new("menu_12_6"), 1, constant.SystemConfigView, entmenu.StatusActive),
		// 系统设置：标签 常规 · 登录 · 邮件 / OIDC 身份提供方
		newDefaultMenu("menu_12_5", entmenu.TypeMenu, "/system/settings", "系统设置", "HOutline:AdjustmentsHorizontalIcon", new("menu_12"), 3, "", entmenu.StatusActive),
		newDefaultMenu("menu_12_5_button_0", entmenu.TypeButton, "", "查看系统配置", "", new("menu_12_5"), 0, constant.SystemConfigView, entmenu.StatusActive),
		newDefaultMenu("menu_12_5_button_1", entmenu.TypeButton, "", "编辑系统配置", "", new("menu_12_5"), 1, constant.SystemConfigEdit, entmenu.StatusActive),
		newDefaultMenu("menu_12_5_button_2", entmenu.TypeButton, "", "更新检查", "", new("menu_12_5"), 2, constant.SystemUpdate, entmenu.StatusActive),
		newDefaultMenu("menu_12_7", entmenu.TypeButton, "", "查看 OIDC", "", new("menu_12_5"), 3, constant.OIDCView, entmenu.StatusActive),
		newDefaultMenu("menu_12_7_button_1", entmenu.TypeButton, "", "编辑 OIDC", "", new("menu_12_5"), 4, constant.OIDCEdit, entmenu.StatusActive),
	}

	// 旧版菜单（每个子页一个菜单）合并成单页后，把原先拥有子页菜单的自定义角色迁移到新结构：
	// 拥有 source 的角色补上 target。只在 source 仍是旧结构（legacyPath 菜单）时执行一次。
	builtinMenuGrants = []builtinMenuGrant{
		{Target: "menu_3_1_button_0", Source: "menu_3_1", LegacyPath: "/file/index"},
		{Target: "menu_3_1", Source: "menu_3_2", LegacyPath: "/file/share"},
		{Target: "menu_3_1", Source: "menu_3_3", LegacyPath: "/file/source"},
		{Target: "menu_5_1", Source: "menu_5_2", LegacyPath: "/docker/image"},
		{Target: "menu_5_1", Source: "menu_5_3", LegacyPath: "/docker/network"},
		{Target: "menu_5_1", Source: "menu_5_5", LegacyPath: "/docker/config"},
		{Target: "menu_6", Source: "menu_8_1", LegacyPath: "/iqcheck/prompt"},
		{Target: "menu_8_1", Source: "menu_8_2", LegacyPath: "/iqcheck/site"},
		{Target: "menu_6", Source: "menu_8_2", LegacyPath: "/iqcheck/site"},
		{Target: "menu_12", Source: "menu_4_2", LegacyPath: "/node/list"},
		{Target: "menu_4_2_button_1", Source: "menu_4_2", LegacyPath: "/node/list"},
		{Target: "menu_12", Source: "menu_4_1", LegacyPath: "/node/key"},
		{Target: "menu_4_2", Source: "menu_4_1", LegacyPath: "/node/key"},
		{Target: "menu_12_1", Source: "menu_12_2", LegacyPath: "/system/role"},
		{Target: "menu_12_1", Source: "menu_12_3", LegacyPath: "/system/menu"},
		{Target: "menu_12_6_button_0", Source: "menu_12_6", LegacyPath: "/system/task"},
		{Target: "menu_12_6", Source: "menu_12_4", LegacyPath: "/system/operation"},
		{Target: "menu_12_5_button_0", Source: "menu_12_5", LegacyPath: "/system/settings", LegacyPermission: string(constant.SystemConfigView)},
		{Target: "menu_12_5", Source: "menu_12_7", LegacyPath: "/system/oidc"},
	}
	builtinDefaultRoles = []defaultRole{
		{
			ID:          adminPermissionRoleID,
			Name:        "超级管理员",
			Code:        "super_admin",
			Description: "拥有系统所有权限，可管理所有功能",
			IsBuiltin:   true,
			Status:      entrole.StatusActive,
		},
		{
			ID:          noPermissionRoleID,
			Name:        "无权限",
			Code:        "no_permission",
			Description: "无权限用户",
			IsBuiltin:   true,
			Status:      entrole.StatusActive,
		},
	}
	defaultUsers = []*defaultUser{}
)

func newDefaultMenu(id string, menuType entmenu.Type, path, title, icon string, parentID *string, order int, permission constant.Permissions, status entmenu.Status) defaultMenu {
	return defaultMenu{
		ID:         id,
		Type:       menuType,
		Path:       path,
		Title:      title,
		Permission: permission,
		Order:      order,
		Icon:       icon,
		IsSystem:   true,
		Status:     status,
		ParentID:   parentID,
	}
}

func syncDefaultRBAC(ctx context.Context, client *gen.Client) error {
	return syncDefaultRBACWithUsers(ctx, client, defaultUsers)
}

func syncDefaultRBACWithUsers(ctx context.Context, client *gen.Client, users []*defaultUser) error {
	menuIDs := make(map[string]struct{}, len(builtinDefaultMenus))
	builtinMenuIDs := make([]string, 0, len(builtinDefaultMenus))
	for _, menu := range builtinDefaultMenus {
		if _, ok := menuIDs[menu.ID]; ok {
			return fmt.Errorf("duplicate builtin menu id: %s", menu.ID)
		}
		menuIDs[menu.ID] = struct{}{}
		builtinMenuIDs = append(builtinMenuIDs, menu.ID)
	}
	roleIDs := make(map[string]struct{}, len(builtinDefaultRoles))
	for _, role := range builtinDefaultRoles {
		if _, ok := roleIDs[role.ID]; ok {
			return fmt.Errorf("duplicate builtin role id: %s", role.ID)
		}
		roleIDs[role.ID] = struct{}{}
	}
	for _, roleID := range builtinRoleWithMenuPermsIDs {
		if _, ok := roleIDs[roleID]; !ok {
			return fmt.Errorf("builtin role for menu perms not found: %s", roleID)
		}
	}
	userIDs := make(map[string]struct{}, len(users))
	for _, user := range users {
		if _, ok := userIDs[user.ID]; ok {
			return fmt.Errorf("duplicate default user id: %s", user.ID)
		}
		userIDs[user.ID] = struct{}{}
		if user.RoleID != "" {
			if _, ok := roleIDs[user.RoleID]; !ok {
				return fmt.Errorf("default user role id not found: %s", user.RoleID)
			}
		}
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}

	rollback := func(err error) error {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("%w: rollback failed: %v", err, rbErr)
		}
		return err
	}

	// 1. 识别需要迁移的旧结构（在菜单被改写之前判断）
	pendingGrants := make([]builtinMenuGrant, 0, len(builtinMenuGrants))
	for _, g := range builtinMenuGrants {
		src, err := tx.Menu.Get(ctx, g.Source)
		if gen.IsNotFound(err) {
			continue
		}
		if err != nil {
			return rollback(fmt.Errorf("load builtin menu %s failed: %w", g.Source, err))
		}
		if src.Path != g.LegacyPath {
			continue
		}
		if g.LegacyPermission != "" && src.Permission != g.LegacyPermission {
			continue
		}
		pendingGrants = append(pendingGrants, g)
	}

	// 2. 写入/同步内置菜单（结构、标题、图标、排序以代码为准，保留启用状态）。
	//    先写父级再写子级（列表本身按父在前排列），保证改挂父级时父级已存在。
	for _, item := range builtinDefaultMenus {
		builder := tx.Menu.Create().
			SetID(item.ID).
			SetType(item.Type).
			SetPath(item.Path).
			SetTitle(item.Title).
			SetPermission(string(item.Permission)).
			SetOrder(item.Order).
			SetIcon(item.Icon).
			SetIsSystem(item.IsSystem).
			SetStatus(item.Status)
		if item.ParentID != nil {
			builder.SetParentID(*item.ParentID)
		}
		hasParent := item.ParentID != nil
		if err := builder.
			OnConflictColumns(entmenu.FieldID).
			Update(func(u *gen.MenuUpsert) {
				u.UpdateType().
					UpdatePath().
					UpdateTitle().
					UpdatePermission().
					UpdateOrder().
					UpdateIcon().
					UpdateIsSystem()
				if hasParent {
					u.UpdateParentID()
				} else {
					u.ClearParentID()
				}
			}).
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("upsert builtin menu %s failed: %w", item.ID, err))
		}
	}

	// 3. 旧结构的自定义角色补上新结构对应的菜单
	for _, g := range pendingGrants {
		roleIDs, err := tx.Role.Query().
			Where(
				entrole.HasMenusWith(entmenu.ID(g.Source)),
				entrole.Not(entrole.HasMenusWith(entmenu.ID(g.Target))),
			).
			IDs(ctx)
		if err != nil {
			return rollback(fmt.Errorf("query roles for menu grant %s failed: %w", g.Target, err))
		}
		for _, roleID := range roleIDs {
			if err := tx.Role.UpdateOneID(roleID).AddMenuIDs(g.Target).Exec(ctx); err != nil {
				return rollback(fmt.Errorf("grant menu %s to role %s failed: %w", g.Target, roleID, err))
			}
		}
	}

	// 4. 删除已不在内置列表里的旧内置菜单（子级已在第 2 步改挂到新父级）
	obsoleteBuiltinMenus := tx.Menu.Delete().
		Where(entmenu.IsSystemEQ(true))
	if len(builtinMenuIDs) > 0 {
		obsoleteBuiltinMenus = obsoleteBuiltinMenus.Where(entmenu.IDNotIn(builtinMenuIDs...))
	}
	if _, err := obsoleteBuiltinMenus.Exec(ctx); err != nil {
		return rollback(fmt.Errorf("delete obsolete builtin menus failed: %w", err))
	}

	roleBuilders := make([]*gen.RoleCreate, 0, len(builtinDefaultRoles))
	for _, item := range builtinDefaultRoles {
		builder := tx.Role.Create().
			SetID(item.ID).
			SetName(item.Name).
			SetDescription(item.Description).
			SetIsBuiltin(item.IsBuiltin).
			SetStatus(item.Status)
		roleBuilders = append(roleBuilders, builder)
	}
	if len(roleBuilders) > 0 {
		if err := tx.Role.CreateBulk(roleBuilders...).
			OnConflictColumns(entrole.FieldID).
			UpdateNewValues().
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("upsert builtin roles failed: %w", err))
		}
	}

	if len(builtinRoleWithMenuPermsIDs) > 0 && len(builtinMenuIDs) > 0 {
		if err := tx.Role.Update().
			Where(entrole.IDIn(builtinRoleWithMenuPermsIDs...)).
			AddMenuIDs(builtinMenuIDs...).
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("sync builtin role menus failed: %w", err))
		}
	}

	userBuilders := make([]*gen.UserCreate, 0, len(users))
	for _, item := range users {
		passwordHash, err := auth.HashPassword(item.Password)
		if err != nil {
			return rollback(fmt.Errorf("hash default user password failed: %w", err))
		}
		builder := tx.User.Create().
			SetID(item.ID).
			SetUsername(item.Username).
			SetPassword(passwordHash).
			SetEmail(item.Email).
			SetStatus(item.Status).
			SetBio(item.Bio).
			SetName(item.Name).
			SetTags(item.Tags)
		userBuilders = append(userBuilders, builder)
	}
	if len(userBuilders) > 0 {
		if err := tx.User.CreateBulk(userBuilders...).
			OnConflictColumns(entuser.FieldID).
			DoNothing().
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("insert default users failed: %w", err))
		}
	}

	// 默认用户（新建时）绑定指定角色；已存在且已有角色的用户不动。
	for _, item := range users {
		if item.RoleID == "" {
			continue
		}
		if err := tx.User.Update().
			Where(entuser.IDEQ(item.ID), entuser.Not(entuser.HasRoles())).
			AddRoleIDs(item.RoleID).
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("bind default user role failed: %w", err))
		}
	}
	// 内置超级管理员始终拥有超级管理员角色；其余没有任何角色的用户给内置「无权限」角色。
	if err := tx.User.Update().
		Where(
			entuser.IDEQ(builtinAdminUserID),
			entuser.Not(entuser.HasRolesWith(entrole.IDEQ(adminPermissionRoleID))),
		).
		AddRoleIDs(adminPermissionRoleID).
		Exec(ctx); err != nil {
		return rollback(fmt.Errorf("bind builtin admin role failed: %w", err))
	}
	if err := tx.User.Update().
		Where(entuser.Not(entuser.HasRoles())).
		AddRoleIDs(noPermissionRoleID).
		Exec(ctx); err != nil {
		return rollback(fmt.Errorf("bind default role failed: %w", err))
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
