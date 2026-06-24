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
	Avatar   string
	Bio      string
	Tags     string
	RoleID   string
}

var (
	adminPermissionRoleID       = "role_1"
	noPermissionRoleID          = "role_2"
	builtinRoleWithMenuPermsIDs = []string{"role_1"}
	builtinDefaultMenus         = []defaultMenu{
		newDefaultMenu("menu_1", entmenu.TypeDirectory, "", "主页", "HOutline:HomeIcon", nil, 0, "", entmenu.StatusActive),
		newDefaultMenu("menu_1_1", entmenu.TypeMenu, "/dashboard/home", "工作台", "HOutline:ComputerDesktopIcon", new("menu_1"), 0, "", entmenu.StatusActive),
		newDefaultMenu("menu_1_2", entmenu.TypeMenu, "/dashboard/analysis", "分析页", "HOutline:ChartBarIcon", new("menu_1"), 1, "", entmenu.StatusInactive),
		newDefaultMenu("menu_1_3", entmenu.TypeMenu, "/dashboard/monitor", "监控页", "HOutline:EyeIcon", new("menu_1"), 2, "", entmenu.StatusInactive),

		// 实例
		newDefaultMenu("menu_2", entmenu.TypeDirectory, "", "应用", "HOutline:ServerStackIcon", nil, 1, "", entmenu.StatusActive),
		newDefaultMenu("menu_2_1", entmenu.TypeMenu, "/instance/list", "应用列表", "HOutline:CubeIcon", new("menu_2"), 0, constant.Instance, entmenu.StatusActive),
		newDefaultMenu("menu_2_2", entmenu.TypeMenu, "/instance/type", "应用类型", "Element:MessageBox", new("menu_2"), 1, "", entmenu.StatusActive),
		newDefaultMenu("menu_2_3", entmenu.TypeMenu, "/instance/terminal", "终端", "HOutline:CommandLineIcon", new("menu_2"), 2, constant.Instance, entmenu.StatusActive),
		newDefaultMenu("menu_2_4", entmenu.TypeMenu, "/openssh/management", "SSH管理", "HOutline:SwatchIcon", new("menu_2"), 3, "", entmenu.StatusActive),

		newDefaultMenu("menu_3", entmenu.TypeDirectory, "", "文件", "HOutline:InboxStackIcon", nil, 2, "", entmenu.StatusActive),
		newDefaultMenu("menu_3_1", entmenu.TypeMenu, "/file/index", "文件管理", "Element:Folder", new("menu_3"), 0, constant.Terminal, entmenu.StatusActive),

		newDefaultMenu("menu_4", entmenu.TypeDirectory, "", "节点管理", "HOutline:SignalIcon", nil, 3, "", entmenu.StatusActive),
		newDefaultMenu("menu_4_1", entmenu.TypeMenu, "/node/key", "ApiKey", "HOutline:KeyIcon", new("menu_4"), 1, constant.ApiKeyView, entmenu.StatusActive),

		newDefaultMenu("menu_5", entmenu.TypeDirectory, "", "Docker", "HOutline:CircleStackIcon", nil, 4, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_1", entmenu.TypeMenu, "/docker/container", "容器", "HOutline:CubeIcon", new("menu_5"), 0, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_1_button_0", entmenu.TypeButton, "", "管理容器", "", new("menu_5_1"), 0, constant.DockerContainerManage, entmenu.StatusActive),
		newDefaultMenu("menu_5_2", entmenu.TypeMenu, "/docker/image", "镜像", "HOutline:PhotoIcon", new("menu_5"), 1, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_2_button_0", entmenu.TypeButton, "", "管理镜像", "", new("menu_5_2"), 0, constant.DockerImageManage, entmenu.StatusActive),
		newDefaultMenu("menu_5_3", entmenu.TypeMenu, "/docker/network", "网络", "HOutline:GlobeAltIcon", new("menu_5"), 2, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_3_button_0", entmenu.TypeButton, "", "管理网络", "", new("menu_5_3"), 0, constant.DockerNetworkManage, entmenu.StatusActive),
		newDefaultMenu("menu_5_4", entmenu.TypeMenu, "/docker/volume", "储存卷", "HOutline:ArchiveBoxIcon", new("menu_5"), 3, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_4_button_0", entmenu.TypeButton, "", "管理储存卷", "", new("menu_5_4"), 0, constant.DockerVolumeManage, entmenu.StatusActive),
		newDefaultMenu("menu_5_5", entmenu.TypeMenu, "/docker/config", "配置", "HOutline:Cog6ToothIcon", new("menu_5"), 4, constant.DockerView, entmenu.StatusActive),
		newDefaultMenu("menu_5_5_button_0", entmenu.TypeButton, "", "编辑Docker配置", "", new("menu_5_5"), 0, constant.DockerConfigEdit, entmenu.StatusActive),

		newDefaultMenu("menu_6", entmenu.TypeDirectory, "", "工具", "Element:Box", nil, 5, "", entmenu.StatusActive),
		newDefaultMenu("menu_6_1", entmenu.TypeMenu, "/tools/port-forward", "端口转发", "HOutline:PaperAirplaneIcon", new("menu_6"), 1, constant.Network, entmenu.StatusActive),
		newDefaultMenu("menu_6_2", entmenu.TypeMenu, "/tools/sub2api", "Sub2API", "HOutline:CloudIcon", new("menu_6"), 2, constant.Sub2APIView, entmenu.StatusActive),
		newDefaultMenu("menu_6_2_button_0", entmenu.TypeButton, "", "编辑Sub2API", "", new("menu_6_2"), 0, constant.Sub2APIEdit, entmenu.StatusActive),

		newDefaultMenu("menu_12", entmenu.TypeDirectory, "", "系统", "HOutline:Cog6ToothIcon", nil, 10, "", entmenu.StatusActive),
		newDefaultMenu("menu_12_1", entmenu.TypeMenu, "/system/user", "用户管理", "HOutline:UserGroupIcon", new("menu_12"), 0, "", entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_0", entmenu.TypeButton, "", "添加用户", "", new("menu_12_1"), 0, constant.UserAdd, entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_1", entmenu.TypeButton, "", "编辑用户", "", new("menu_12_1"), 1, constant.UserEdit, entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_2", entmenu.TypeButton, "", "删除用户", "", new("menu_12_1"), 2, constant.UserDelete, entmenu.StatusActive),
		newDefaultMenu("menu_12_1_button_3", entmenu.TypeButton, "", "查看用户", "", new("menu_12_1"), 3, constant.UserView, entmenu.StatusActive),
		newDefaultMenu("menu_12_2", entmenu.TypeMenu, "/system/role", "角色管理", "HOutline:IdentificationIcon", new("menu_12"), 1, "", entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_0", entmenu.TypeButton, "", "添加角色", "", new("menu_12_2"), 0, constant.RoleAdd, entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_1", entmenu.TypeButton, "", "编辑角色", "", new("menu_12_2"), 1, constant.RoleEdit, entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_2", entmenu.TypeButton, "", "删除角色", "", new("menu_12_2"), 2, constant.RoleDelete, entmenu.StatusActive),
		newDefaultMenu("menu_12_2_button_3", entmenu.TypeButton, "", "查看角色", "", new("menu_12_2"), 3, constant.RoleView, entmenu.StatusActive),
		newDefaultMenu("menu_12_3", entmenu.TypeMenu, "/system/menu", "菜单管理", "HOutline:ListBulletIcon", new("menu_12"), 2, "", entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_0", entmenu.TypeButton, "", "添加菜单", "", new("menu_12_3"), 0, constant.MenuAdd, entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_1", entmenu.TypeButton, "", "编辑菜单", "", new("menu_12_3"), 1, constant.MenuEdit, entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_2", entmenu.TypeButton, "", "删除菜单", "", new("menu_12_3"), 2, constant.MenuDelete, entmenu.StatusActive),
		newDefaultMenu("menu_12_3_button_3", entmenu.TypeButton, "", "查看菜单", "", new("menu_12_3"), 3, constant.MenuView, entmenu.StatusActive),
		newDefaultMenu("menu_12_4", entmenu.TypeMenu, "/system/operation", "操作记录", "HOutline:ChartBarSquareIcon", new("menu_12"), 3, constant.SystemConfigView, entmenu.StatusActive),
		newDefaultMenu("menu_12_5", entmenu.TypeMenu, "/system/settings", "系统配置", "HOutline:CogIcon", new("menu_12"), 4, constant.SystemConfigView, entmenu.StatusActive),
		newDefaultMenu("menu_12_5_button_1", entmenu.TypeButton, "", "编辑系统配置", "", new("menu_12_5"), 0, constant.SystemConfigEdit, entmenu.StatusActive),

		newDefaultMenu("menu_13", entmenu.TypeDirectory, "", "扩展组件", "HOutline:PuzzlePieceIcon", nil, 11, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_1", entmenu.TypeMenu, "/extended/button", "按钮", "HOutline:HandRaisedIcon", new("menu_13"), 0, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_2", entmenu.TypeMenu, "/extended/dialog", "对话框", "HOutline:WindowIcon", new("menu_13"), 1, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_3", entmenu.TypeMenu, "/extended/iconSelector", "图标选择器", "HOutline:SwatchIcon", new("menu_13"), 2, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_4", entmenu.TypeMenu, "/extended/textEllipsis", "文本省略器", "HOutline:EllipsisHorizontalIcon", new("menu_13"), 3, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_5", entmenu.TypeMenu, "/extended/hoverAnimation", "Hover动画组件", "HOutline:CursorArrowRaysIcon", new("menu_13"), 4, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_6", entmenu.TypeMenu, "/extended/transitionAnimation", "Transition内置动画", "HOutline:SparklesIcon", new("menu_13"), 5, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_7", entmenu.TypeMenu, "/demo/vxeTable", "VXE Table", "HOutline:TableCellsIcon", new("menu_13"), 6, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_8", entmenu.TypeMenu, "/exception/403", "403页面", "HOutline:NoSymbolIcon", new("menu_15"), 7, constant.Dev, entmenu.StatusInactive),
		newDefaultMenu("menu_13_9", entmenu.TypeMenu, "/exception/404", "404页面", "HOutline:QuestionMarkCircleIcon", new("menu_15"), 8, constant.Dev, entmenu.StatusInactive),
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

	menuBuilders := make([]*gen.MenuCreate, 0, len(builtinDefaultMenus))
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
		menuBuilders = append(menuBuilders, builder)
	}
	obsoleteBuiltinMenus := tx.Menu.Delete().
		Where(entmenu.IsSystemEQ(true))
	if len(builtinMenuIDs) > 0 {
		obsoleteBuiltinMenus = obsoleteBuiltinMenus.Where(entmenu.IDNotIn(builtinMenuIDs...))
	}
	if _, err := obsoleteBuiltinMenus.Exec(ctx); err != nil {
		return rollback(fmt.Errorf("delete obsolete builtin menus failed: %w", err))
	}
	if len(menuBuilders) > 0 {
		if err := tx.Menu.CreateBulk(menuBuilders...).
			OnConflictColumns(entmenu.FieldID).
			DoNothing().
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("insert builtin menus failed: %w", err))
		}
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
			SetAvatar(item.Avatar).
			SetBio(item.Bio).
			SetName(item.Name).
			SetTags(item.Tags)
		if item.RoleID != "" {
			builder.SetRoleID(item.RoleID)
		}
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

	if adminPermissionRoleID != "" {
		if _, err := tx.Role.Get(ctx, adminPermissionRoleID); err != nil {
			return rollback(fmt.Errorf("default role %s not found: %w", adminPermissionRoleID, err))
		}
		if err := tx.User.Update().
			Where(entuser.Not(entuser.HasRole())).
			SetRoleID(adminPermissionRoleID).
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("bind default role failed: %w", err))
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
