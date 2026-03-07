package data

import (
	"context"
	"fmt"
	"strings"

	"momoko/internal/data/ent"
	entmenu "momoko/internal/data/ent/menu"
	entrole "momoko/internal/data/ent/role"
	entuser "momoko/internal/data/ent/user"
)

type defaultMenu struct {
	ID         string
	Type       entmenu.Type
	Path       string
	Title      string
	Permission string
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
		newDefaultMenu("menu_1", entmenu.TypeDirectory, "", "Dashboard", "HOutline:HomeIcon", nil, 0, ""),
		newDefaultMenu("menu_12", entmenu.TypeMenu, "/dashboard/home", "工作台", "HOutline:ComputerDesktopIcon", ptr("menu_1"), 0, ""),
		newDefaultMenu("menu_13", entmenu.TypeMenu, "/dashboard/analysis", "分析页", "HOutline:ChartBarIcon", ptr("menu_1"), 1, ""),
		newDefaultMenu("menu_14", entmenu.TypeMenu, "/dashboard/monitor", "监控页", "HOutline:EyeIcon", ptr("menu_1"), 2, ""),

		newDefaultMenu("menu_2", entmenu.TypeDirectory, "", "系统管理", "HOutline:Cog6ToothIcon", nil, 1, ""),
		newDefaultMenu("menu_3", entmenu.TypeMenu, "/system/user", "用户管理", "HOutline:UserGroupIcon", ptr("menu_2"), 0, ""),
		newDefaultMenu("menu_3_button_0", entmenu.TypeButton, "", "添加用户", "", ptr("menu_3"), 0, "user:add"),
		newDefaultMenu("menu_3_button_1", entmenu.TypeButton, "", "编辑用户", "", ptr("menu_3"), 1, "user:edit"),
		newDefaultMenu("menu_3_button_2", entmenu.TypeButton, "", "删除用户", "", ptr("menu_3"), 2, "user:delete"),
		newDefaultMenu("menu_3_button_3", entmenu.TypeButton, "", "查看用户", "", ptr("menu_3"), 3, "user:view"),
		newDefaultMenu("menu_4", entmenu.TypeMenu, "/system/role", "角色管理", "HOutline:IdentificationIcon", ptr("menu_2"), 1, ""),
		newDefaultMenu("menu_4_button_0", entmenu.TypeButton, "", "添加角色", "", ptr("menu_4"), 0, "role:add"),
		newDefaultMenu("menu_4_button_1", entmenu.TypeButton, "", "编辑角色", "", ptr("menu_4"), 1, "role:edit"),
		newDefaultMenu("menu_4_button_2", entmenu.TypeButton, "", "删除角色", "", ptr("menu_4"), 2, "role:delete"),
		newDefaultMenu("menu_4_button_3", entmenu.TypeButton, "", "查看角色", "", ptr("menu_4"), 3, "role:view"),
		newDefaultMenu("menu_5", entmenu.TypeMenu, "/system/menu", "菜单管理", "HOutline:ListBulletIcon", ptr("menu_2"), 2, ""),
		newDefaultMenu("menu_5_button_0", entmenu.TypeButton, "", "添加菜单", "", ptr("menu_5"), 0, "menu:add"),
		newDefaultMenu("menu_5_button_1", entmenu.TypeButton, "", "编辑菜单", "", ptr("menu_5"), 1, "menu:edit"),
		newDefaultMenu("menu_5_button_2", entmenu.TypeButton, "", "删除菜单", "", ptr("menu_5"), 2, "menu:delete"),
		newDefaultMenu("menu_5_button_3", entmenu.TypeButton, "", "查看菜单", "", ptr("menu_5"), 3, "menu:view"),

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

func newDefaultMenu(id string, menuType entmenu.Type, path, title, icon string, parentID *string, order int, permission string) defaultMenu {
	if strings.TrimSpace(path) == "" {
		path = "/builtin/" + id
	}
	if strings.TrimSpace(permission) == "" {
		permission = "menu:" + id + ":view"
	}
	if strings.TrimSpace(icon) == "" {
		icon = "HOutline:Square3Stack3DIcon"
	}
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

	for _, item := range builtinDefaultMenus {
		exists, err := tx.Menu.Get(ctx, item.ID)
		if err != nil {
			if !ent.IsNotFound(err) {
				return rollback(fmt.Errorf("query menu %s failed: %w", item.ID, err))
			}
			create := tx.Menu.Create().
				SetID(item.ID).
				SetType(item.Type).
				SetPath(item.Path).
				SetTitle(item.Title).
				SetPermission(item.Permission).
				SetOrder(item.Order).
				SetIcon(item.Icon).
				SetIsSystem(item.IsSystem).
				SetStatus(item.Status)
			if item.ParentID != nil {
				create.SetParentID(*item.ParentID)
			}
			if _, err := create.Save(ctx); err != nil {
				return rollback(fmt.Errorf("create menu %s failed: %w", item.ID, err))
			}
			continue
		}

		update := tx.Menu.UpdateOneID(exists.ID).
			SetType(item.Type).
			SetPath(item.Path).
			SetTitle(item.Title).
			SetPermission(item.Permission).
			SetOrder(item.Order).
			SetIcon(item.Icon).
			SetIsSystem(item.IsSystem).
			SetStatus(item.Status).
			SetNillableParentID(item.ParentID)
		if err := update.Exec(ctx); err != nil {
			return rollback(fmt.Errorf("update menu %s failed: %w", item.ID, err))
		}
	}

	menuRows, err := tx.Menu.Query().All(ctx)
	if err != nil {
		return rollback(fmt.Errorf("query all menus failed: %w", err))
	}
	allMenuIDs := make([]string, 0, len(menuRows))
	for _, item := range menuRows {
		allMenuIDs = append(allMenuIDs, item.ID)
	}

	for _, item := range builtinDefaultRoles {
		exists, err := tx.Role.Get(ctx, item.ID)
		if err != nil {
			if !ent.IsNotFound(err) {
				return rollback(fmt.Errorf("query role %s failed: %w", item.ID, err))
			}
			create := tx.Role.Create().
				SetID(item.ID).
				SetName(item.Name).
				SetDescription(item.Description + " (code: " + item.Code + ")").
				SetIsBuiltin(item.IsBuiltin).
				SetStatus(item.Status)
			menuIDs := allMenuIDs
			if len(menuIDs) > 0 {
				create.AddMenuIDs(menuIDs...)
			}
			if _, err := create.Save(ctx); err != nil {
				return rollback(fmt.Errorf("create role %s failed: %w", item.ID, err))
			}
			continue
		}

		update := tx.Role.UpdateOneID(exists.ID).
			SetName(item.Name).
			SetDescription(item.Description + " (code: " + item.Code + ")").
			SetIsBuiltin(item.IsBuiltin).
			SetStatus(item.Status).
			ClearMenus()
		menuIDs := allMenuIDs
		if len(menuIDs) > 0 {
			update.AddMenuIDs(menuIDs...)
		}
		if err := update.Exec(ctx); err != nil {
			return rollback(fmt.Errorf("update role %s failed: %w", item.ID, err))
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
