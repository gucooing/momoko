package data

import (
	"context"
	"fmt"

	"momoko/internal/data/ent"
	entmenu "momoko/internal/data/ent/menu"
	entrole "momoko/internal/data/ent/role"
	entuser "momoko/internal/data/ent/user"
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
		newDefaultMenu("menu_1", entmenu.TypeDirectory, "", "主页", "HOutline:HomeIcon", nil, 0, ""),
		newDefaultMenu("menu_1_1", entmenu.TypeMenu, "/dashboard/home", "工作台", "HOutline:ComputerDesktopIcon", ptr("menu_1"), 0, ""),
		newDefaultMenu("menu_1_2", entmenu.TypeMenu, "/dashboard/analysis", "分析页", "HOutline:ChartBarIcon", ptr("menu_1"), 1, ""),
		newDefaultMenu("menu_1_3", entmenu.TypeMenu, "/dashboard/monitor", "监控页", "HOutline:EyeIcon", ptr("menu_1"), 2, ""),

		// 实例
		newDefaultMenu("menu_2", entmenu.TypeDirectory, "", "应用管理", "HOutline:ServerStackIcon", nil, 1, ""),
		newDefaultMenu("menu_2_1", entmenu.TypeMenu, "/instance/list", "应用列表", "HOutline:CubeIcon", ptr("menu_2"), 0, ""),
		newDefaultMenu("menu_2_2", entmenu.TypeMenu, "/instance/type", "应用类型", "Element:MessageBox", ptr("menu_2"), 1, ""),
		newDefaultMenu("menu_2_3", entmenu.TypeMenu, "/instance/terminal", "终端", "HOutline:CommandLineIcon", ptr("menu_2"), 2, ""),
		newDefaultMenu("menu_2_4", entmenu.TypeMenu, "/instance/files", "文件管理", "Element:Folder", ptr("menu_2"), 3, constant.Terminal),

		newDefaultMenu("menu_12", entmenu.TypeDirectory, "", "系统管理", "HOutline:Cog6ToothIcon", nil, 10, ""),
		newDefaultMenu("menu_12_1", entmenu.TypeMenu, "/system/user", "用户管理", "HOutline:UserGroupIcon", ptr("menu_12"), 0, ""),
		newDefaultMenu("menu_12_1_button_0", entmenu.TypeButton, "", "添加用户", "", ptr("menu_12_1"), 0, constant.UserAdd),
		newDefaultMenu("menu_12_1_button_1", entmenu.TypeButton, "", "编辑用户", "", ptr("menu_12_1"), 1, constant.UserEdit),
		newDefaultMenu("menu_12_1_button_2", entmenu.TypeButton, "", "删除用户", "", ptr("menu_12_1"), 2, constant.UserDelete),
		newDefaultMenu("menu_12_1_button_3", entmenu.TypeButton, "", "查看用户", "", ptr("menu_12_1"), 3, constant.UserView),
		newDefaultMenu("menu_12_2", entmenu.TypeMenu, "/system/role", "角色管理", "HOutline:IdentificationIcon", ptr("menu_12"), 1, ""),
		newDefaultMenu("menu_12_2_button_0", entmenu.TypeButton, "", "添加角色", "", ptr("menu_12_2"), 0, constant.RoleAdd),
		newDefaultMenu("menu_12_2_button_1", entmenu.TypeButton, "", "编辑角色", "", ptr("menu_12_2"), 1, constant.RoleEdit),
		newDefaultMenu("menu_12_2_button_2", entmenu.TypeButton, "", "删除角色", "", ptr("menu_12_2"), 2, constant.RoleDelete),
		newDefaultMenu("menu_12_2_button_3", entmenu.TypeButton, "", "查看角色", "", ptr("menu_12_2"), 3, constant.RoleView),
		newDefaultMenu("menu_12_3", entmenu.TypeMenu, "/system/menu", "菜单管理", "HOutline:ListBulletIcon", ptr("menu_12"), 2, ""),
		newDefaultMenu("menu_12_3_button_0", entmenu.TypeButton, "", "添加菜单", "", ptr("menu_12_3"), 0, constant.MenuAdd),
		newDefaultMenu("menu_12_3_button_1", entmenu.TypeButton, "", "编辑菜单", "", ptr("menu_12_3"), 1, constant.MenuEdit),
		newDefaultMenu("menu_12_3_button_2", entmenu.TypeButton, "", "删除菜单", "", ptr("menu_12_3"), 2, constant.MenuDelete),
		newDefaultMenu("menu_12_3_button_3", entmenu.TypeButton, "", "查看菜单", "", ptr("menu_12_3"), 3, constant.MenuView),

		newDefaultMenu("menu_13", entmenu.TypeDirectory, "", "扩展组件", "HOutline:PuzzlePieceIcon", nil, 11, ""),
		newDefaultMenu("menu_13_1", entmenu.TypeMenu, "/extended/button", "按钮", "HOutline:HandRaisedIcon", ptr("menu_13"), 0, ""),
		newDefaultMenu("menu_13_2", entmenu.TypeMenu, "/extended/dialog", "对话框", "HOutline:WindowIcon", ptr("menu_13"), 0, ""),
		newDefaultMenu("menu_13_3", entmenu.TypeMenu, "/extended/iconSelector", "图标选择器", "HOutline:SwatchIcon", ptr("menu_13"), 1, ""),
		newDefaultMenu("menu_13_4", entmenu.TypeMenu, "/extended/textEllipsis", "文本省略器", "HOutline:EllipsisHorizontalIcon", ptr("menu_13"), 2, ""),
		newDefaultMenu("menu_13_5", entmenu.TypeMenu, "/extended/hoverAnimation", "Hover动画组件", "HOutline:CursorArrowRaysIcon", ptr("menu_13"), 3, ""),
		newDefaultMenu("menu_13_6", entmenu.TypeMenu, "/extended/transitionAnimation", "Transition内置动画", "HOutline:SparklesIcon", ptr("menu_13"), 3, ""),

		newDefaultMenu("menu_14", entmenu.TypeDirectory, "", "功能演示", "HOutline:BeakerIcon", nil, 12, ""),
		newDefaultMenu("menu_14_1", entmenu.TypeMenu, "/demo/vxeTable", "VXE Table", "HOutline:TableCellsIcon", ptr("menu_14"), 0, ""),

		newDefaultMenu("menu_15", entmenu.TypeDirectory, "", "异常页面", "HOutline:ExclamationTriangleIcon", nil, 13, ""),
		newDefaultMenu("menu_15_1", entmenu.TypeMenu, "/exception/403", "403页面", "HOutline:NoSymbolIcon", ptr("menu_15"), 0, ""),
		newDefaultMenu("menu_15_2", entmenu.TypeMenu, "/exception/404", "404页面", "HOutline:QuestionMarkCircleIcon", ptr("menu_15"), 1, ""),

		newDefaultMenu("menu_16", entmenu.TypeDirectory, "", "一级菜单", "HOutline:FolderIcon", nil, 14, ""),
		newDefaultMenu("menu_16_1", entmenu.TypeDirectory, "", "二级菜单", "HOutline:FolderOpenIcon", ptr("menu_16"), 0, ""),
		newDefaultMenu("menu_16_1_1", entmenu.TypeMenu, "/aaa/bbb/ccc", "三级菜单", "HOutline:DocumentTextIcon", ptr("menu_16_1"), 0, ""),
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
	defaultUsers = []*defaultUser{
		{
			ID:       "admin_1",
			Username: "admin",
			Password: "admin",
			Email:    "admin@alsl.xyz",
			Status:   entuser.StatusActive,
			Avatar:   "",
			Bio:      "",
			Name:     "超级管理员",
			Tags:     "",
			RoleID:   adminPermissionRoleID,
		},
	}
)

func ptr(v string) *string {
	return &v
}

func newDefaultMenu(id string, menuType entmenu.Type, path, title, icon string, parentID *string, order int, permission constant.Permissions) defaultMenu {
	return defaultMenu{
		ID:         id,
		Type:       menuType,
		Path:       path,
		Title:      title,
		Permission: permission,
		Order:      order,
		Icon:       icon,
		IsSystem:   true,
		Status:     entmenu.StatusActive,
		ParentID:   parentID,
	}
}

func syncDefaultRBAC(ctx context.Context, client *ent.Client) error {
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
	userIDs := make(map[string]struct{}, len(defaultUsers))
	for _, user := range defaultUsers {
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

	menuBuilders := make([]*ent.MenuCreate, 0, len(builtinDefaultMenus))
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
	if len(menuBuilders) > 0 {
		if err := tx.Menu.CreateBulk(menuBuilders...).
			OnConflictColumns(entmenu.FieldID).
			UpdateNewValues().
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("upsert builtin menus failed: %w", err))
		}
	}

	roleBuilders := make([]*ent.RoleCreate, 0, len(builtinDefaultRoles))
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

	userBuilders := make([]*ent.UserCreate, 0, len(defaultUsers))
	for _, item := range defaultUsers {
		builder := tx.User.Create().
			SetID(item.ID).
			SetUsername(item.Username).
			SetPassword(item.Password).
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
