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

var (
	builtinDefaultRoleID = "role_1"
	builtinDefaultMenus  = []defaultMenu{
		newDefaultMenu("menu_1", entmenu.TypeDirectory, "", "主页", "HOutline:HomeIcon", nil, 0, ""),
		newDefaultMenu("menu_12", entmenu.TypeMenu, "/dashboard/home", "工作台", "HOutline:ComputerDesktopIcon", ptr("menu_1"), 0, ""),
		newDefaultMenu("menu_13", entmenu.TypeMenu, "/dashboard/analysis", "分析页", "HOutline:ChartBarIcon", ptr("menu_1"), 1, ""),
		newDefaultMenu("menu_14", entmenu.TypeMenu, "/dashboard/monitor", "监控页", "HOutline:EyeIcon", ptr("menu_1"), 2, ""),

		newDefaultMenu("menu_2", entmenu.TypeDirectory, "", "系统管理", "HOutline:Cog6ToothIcon", nil, 1, ""),
		newDefaultMenu("menu_3", entmenu.TypeMenu, "/system/user", "用户管理", "HOutline:UserGroupIcon", ptr("menu_2"), 0, ""),
		newDefaultMenu("menu_3_button_0", entmenu.TypeButton, "", "添加用户", "", ptr("menu_3"), 0, constant.UserAdd),
		newDefaultMenu("menu_3_button_1", entmenu.TypeButton, "", "编辑用户", "", ptr("menu_3"), 1, constant.UserEdit),
		newDefaultMenu("menu_3_button_2", entmenu.TypeButton, "", "删除用户", "", ptr("menu_3"), 2, constant.UserDelete),
		newDefaultMenu("menu_3_button_3", entmenu.TypeButton, "", "查看用户", "", ptr("menu_3"), 3, constant.UserView),
		newDefaultMenu("menu_4", entmenu.TypeMenu, "/system/role", "角色管理", "HOutline:IdentificationIcon", ptr("menu_2"), 1, ""),
		newDefaultMenu("menu_4_button_0", entmenu.TypeButton, "", "添加角色", "", ptr("menu_4"), 0, constant.RoleAdd),
		newDefaultMenu("menu_4_button_1", entmenu.TypeButton, "", "编辑角色", "", ptr("menu_4"), 1, constant.RoleEdit),
		newDefaultMenu("menu_4_button_2", entmenu.TypeButton, "", "删除角色", "", ptr("menu_4"), 2, constant.RoleDelete),
		newDefaultMenu("menu_4_button_3", entmenu.TypeButton, "", "查看角色", "", ptr("menu_4"), 3, constant.RoleView),
		newDefaultMenu("menu_5", entmenu.TypeMenu, "/system/menu", "菜单管理", "HOutline:ListBulletIcon", ptr("menu_2"), 2, ""),
		newDefaultMenu("menu_5_button_0", entmenu.TypeButton, "", "添加菜单", "", ptr("menu_5"), 0, constant.MenuAdd),
		newDefaultMenu("menu_5_button_1", entmenu.TypeButton, "", "编辑菜单", "", ptr("menu_5"), 1, constant.MenuEdit),
		newDefaultMenu("menu_5_button_2", entmenu.TypeButton, "", "删除菜单", "", ptr("menu_5"), 2, constant.MenuDelete),
		newDefaultMenu("menu_5_button_3", entmenu.TypeButton, "", "查看菜单", "", ptr("menu_5"), 3, constant.MenuView),

		newDefaultMenu("menu_17", entmenu.TypeDirectory, "", "扩展组件", "HOutline:PuzzlePieceIcon", nil, 2, ""),
		newDefaultMenu("menu_18", entmenu.TypeMenu, "/extended/button", "按钮", "HOutline:HandRaisedIcon", ptr("menu_17"), 0, ""),
		newDefaultMenu("menu_19", entmenu.TypeMenu, "/extended/dialog", "对话框", "HOutline:WindowIcon", ptr("menu_17"), 0, ""),
		newDefaultMenu("menu_20", entmenu.TypeMenu, "/extended/iconSelector", "图标选择器", "HOutline:SwatchIcon", ptr("menu_17"), 1, ""),
		newDefaultMenu("menu_21", entmenu.TypeMenu, "/extended/textEllipsis", "文本省略器", "HOutline:EllipsisHorizontalIcon", ptr("menu_17"), 2, ""),
		newDefaultMenu("menu_22", entmenu.TypeMenu, "/extended/hoverAnimation", "Hover动画组件", "HOutline:CursorArrowRaysIcon", ptr("menu_17"), 3, ""),
		newDefaultMenu("menu_23", entmenu.TypeMenu, "/extended/transitionAnimation", "Transition内置动画", "HOutline:SparklesIcon", ptr("menu_17"), 3, ""),

		newDefaultMenu("menu_15", entmenu.TypeDirectory, "", "功能演示", "HOutline:BeakerIcon", nil, 3, ""),
		newDefaultMenu("menu_16", entmenu.TypeMenu, "/demo/vxeTable", "VXE Table", "HOutline:TableCellsIcon", ptr("menu_15"), 0, ""),

		newDefaultMenu("menu_9", entmenu.TypeDirectory, "", "异常页面", "HOutline:ExclamationTriangleIcon", nil, 4, ""),
		newDefaultMenu("menu_10", entmenu.TypeMenu, "/exception/403", "403页面", "HOutline:NoSymbolIcon", ptr("menu_9"), 0, ""),
		newDefaultMenu("menu_11", entmenu.TypeMenu, "/exception/404", "404页面", "HOutline:QuestionMarkCircleIcon", ptr("menu_9"), 1, ""),

		newDefaultMenu("menu_6", entmenu.TypeDirectory, "", "一级菜单", "HOutline:FolderIcon", nil, 5, ""),
		newDefaultMenu("menu_7", entmenu.TypeDirectory, "", "二级菜单", "HOutline:FolderOpenIcon", ptr("menu_6"), 0, ""),
		newDefaultMenu("menu_8", entmenu.TypeMenu, "/aaa/bbb/ccc", "三级菜单", "HOutline:DocumentTextIcon", ptr("menu_7"), 0, ""),
	}
	builtinDefaultRoles = []defaultRole{
		{
			ID:          "role_1",
			Name:        "超级管理员",
			Code:        "super_admin",
			Description: "拥有系统所有权限，可管理所有功能",
			IsBuiltin:   true,
			Status:      entrole.StatusActive,
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
	for _, menu := range builtinDefaultMenus {
		if _, ok := menuIDs[menu.ID]; ok {
			return fmt.Errorf("duplicate builtin menu id: %s", menu.ID)
		}
		menuIDs[menu.ID] = struct{}{}
	}
	roleIDs := make(map[string]struct{}, len(builtinDefaultRoles))
	builtinRoleIDs := make([]string, 0, len(builtinDefaultRoles))
	for _, role := range builtinDefaultRoles {
		if _, ok := roleIDs[role.ID]; ok {
			return fmt.Errorf("duplicate builtin role id: %s", role.ID)
		}
		roleIDs[role.ID] = struct{}{}
		builtinRoleIDs = append(builtinRoleIDs, role.ID)
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

	allMenuIDs, err := tx.Menu.Query().IDs(ctx)
	if err != nil {
		return rollback(fmt.Errorf("query all menu ids failed: %w", err))
	}

	roleBuilders := make([]*ent.RoleCreate, 0, len(builtinDefaultRoles))
	for _, item := range builtinDefaultRoles {
		builder := tx.Role.Create().
			SetID(item.ID).
			SetName(item.Name).
			SetDescription(item.Description + " (code: " + item.Code + ")").
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

	if len(builtinRoleIDs) > 0 {
		roleMenuUpdate := tx.Role.Update().
			Where(entrole.IDIn(builtinRoleIDs...)).
			ClearMenus()
		if len(allMenuIDs) > 0 {
			roleMenuUpdate.AddMenuIDs(allMenuIDs...)
		}
		if err := roleMenuUpdate.Exec(ctx); err != nil {
			return rollback(fmt.Errorf("sync builtin role menus failed: %w", err))
		}
	}

	if builtinDefaultRoleID != "" {
		if _, err := tx.Role.Get(ctx, builtinDefaultRoleID); err != nil {
			return rollback(fmt.Errorf("default role %s not found: %w", builtinDefaultRoleID, err))
		}
		if err := tx.User.Update().
			Where(entuser.Not(entuser.HasRole())).
			SetRoleID(builtinDefaultRoleID).
			Exec(ctx); err != nil {
			return rollback(fmt.Errorf("bind default role failed: %w", err))
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
