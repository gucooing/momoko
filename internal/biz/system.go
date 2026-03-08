package biz

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/menu"
	"momoko/pkg/auth"
	"momoko/pkg/cache"
	"momoko/pkg/constant"
)

type SystemUsecase struct {
	sys      SystemRepo
	userRepo UserRepo

	cache cache.Cache[string, *RoleOjb]
}

// 权限缓存
type RoleOjb struct {
	Menus       []*ent.Menu                       // 原始数据
	Permissions map[constant.Permissions]struct{} // 权限快速获取数据
}

func (s *SystemUsecase) Check(ctx context.Context, permissions constant.Permissions) error {
	refreshAuth, ok := auth.FromContext(ctx)
	if !ok {
		return ErrNoPermission
	}
	r, err := s.GetRoleOjbByUserID(ctx, refreshAuth.UserID)
	if err != nil {
		return ErrNoPermission
	}
	if r.Permissions == nil {
		return ErrNoPermission
	}
	_, ok = r.Permissions[permissions]
	if !ok {
		return ErrNoPermission
	}
	return nil
}

type SystemRepo interface {
	GetMenusByRoleId(ctx context.Context, roleId string) ([]*ent.Menu, error)
	GetMenus(ctx context.Context) ([]*ent.Menu, error)
	GetMenu(ctx context.Context, menuId string) (*ent.Menu, error)
	CreateMenu(ctx context.Context, menu *ent.Menu) (*ent.Menu, error)
	UpdateMenu(ctx context.Context, menuInfo *ent.Menu) (*ent.Menu, error)
	DeleteMenu(ctx context.Context, menuId string) error
}

func NewSystemUsecase(sys SystemRepo, userRepo UserRepo) *SystemUsecase {
	return &SystemUsecase{
		sys:      sys,
		userRepo: userRepo,
	}
}

func (s *SystemUsecase) GetRoleOjbByUserID(ctx context.Context, userID string) (*RoleOjb, error) {
	userInfo, err := s.userRepo.FindWithRoleByID(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if userInfo.Edges.Role == nil {
		return nil, ErrUserNoRole
	}
	add := func() (*RoleOjb, error) {
		menuInfos, err := s.sys.GetMenusByRoleId(ctx, userInfo.Edges.Role.ID)
		if err != nil {
			return nil, ErrSystem(err)
		}
		return &RoleOjb{
			Menus:       menuInfos,
			Permissions: toPermissions(menuInfos),
		}, nil
	}
	ojb, ok := s.cache.GetByAdd(userInfo.Edges.Role.ID, add)
	if !ok {
		return nil, ErrUserNoRole
	}
	return ojb, nil
}

func (s *SystemUsecase) GetMenusByUserID(ctx context.Context, userID string) ([]*v1.MenuInfo, []string, error) {
	ojb, err := s.GetRoleOjbByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	menus, permissions := toMenuInfos(ojb.Menus)
	return menus, permissions, nil
}

func (s *SystemUsecase) GetAllMenus(ctx context.Context) ([]*v1.MenuInfo, []string, error) {
	menuInfos, err := s.sys.GetMenus(ctx)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	menus, permissions := toMenuInfos(menuInfos)
	return menus, permissions, nil
}

func (s *SystemUsecase) GetMenu(ctx context.Context, menuId string) (*v1.MenuInfo, error) {
	menuInfo, err := s.sys.GetMenu(ctx, menuId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toMenuInfo(menuInfo), nil
}

func (s *SystemUsecase) AddMenu(ctx context.Context, menu *v1.AdminAddPermissionsRequest) error {
	_, err := s.sys.CreateMenu(ctx, &ent.Menu{
		ID:         uuid.NewString(),
		Type:       toEntMenuType(menu.Type),
		Path:       menu.Path,
		Title:      menu.Title,
		Permission: menu.Permissions,
		Order:      int(menu.Order),
		Icon:       menu.Icon,
		IsSystem:   false,
		Status:     toEntMenuStatus(menu.Status),
		ParentID:   menu.ParentId,
	})
	if err != nil {
		return ErrSystem(err)
	}
	s.cache.Clear()
	return nil
}

func (s *SystemUsecase) UpdateMenu(ctx context.Context, menu *v1.AdminEditPermissionsRequest) error {
	_, err := s.sys.UpdateMenu(ctx, &ent.Menu{
		ID:         menu.MenuId,
		Path:       menu.Path,
		Title:      menu.Title,
		Permission: menu.Permissions,
		Order:      int(menu.Order),
		Icon:       menu.Icon,
		Status:     toEntMenuStatus(menu.Status),
	})
	if err != nil {
		return ErrSystem(err)
	}
	s.cache.Clear()
	return nil
}

func (s *SystemUsecase) DeleteMenu(ctx context.Context, menuId string) error {
	err := s.sys.DeleteMenu(ctx, menuId)
	if err != nil {
		return ErrSystem(err)
	}
	s.cache.Clear()
	return nil
}

func toMenuInfos(menus []*ent.Menu) ([]*v1.MenuInfo, []string) {
	menuInfos := make([]*v1.MenuInfo, 0)
	permissions := make([]string, 0)

	menuInfoMap := make(map[string]*v1.MenuInfo, len(menus))
	permissionSet := make(map[string]struct{}, len(menus))

	for _, item := range menus {
		if item.Permission != "" &&
			item.Status == menu.StatusActive {
			if _, ok := permissionSet[item.Permission]; !ok {
				permissionSet[item.Permission] = struct{}{}
				permissions = append(permissions, item.Permission)
			}
		}
		menuInfoMap[item.ID] = toMenuInfo(item)
	}

	for _, item := range menuInfoMap {
		if item.ParentId == "" { // 没有上级表示顶层
			menuInfos = append(menuInfos, item)
			continue
		}
		parent, ok := menuInfoMap[item.ParentId]
		if !ok {
			continue
		}
		parent.Children = append(parent.Children, item)
	}

	return menuInfos, permissions
}

func toPermissions(menus []*ent.Menu) map[constant.Permissions]struct{} {
	permissions := make(map[constant.Permissions]struct{})
	for _, item := range menus {
		if item.Permission == "" ||
			item.Status != menu.StatusActive {
			continue
		}
		permissions[constant.Permissions(item.Permission)] = struct{}{}
	}
	return permissions
}

func toMenuInfo(data *ent.Menu) *v1.MenuInfo {
	return &v1.MenuInfo{
		Id:          data.ID,
		Icon:        data.Icon,
		IsSystem:    data.IsSystem,
		Order:       int32(data.Order),
		ParentId:    data.ParentID,
		Path:        data.Path,
		Status:      toMenuStatus(data.Status),
		Title:       data.Title,
		Type:        toMenuType(data.Type),
		Permissions: data.Permission,
		CreateTime:  timestamppb.New(data.CreateTime),
		UpdateTime:  timestamppb.New(data.UpdateTime),
		Children:    make([]*v1.MenuInfo, 0),
	}
}

func toMenuStatus(data menu.Status) v1.MenuStatus {
	switch data {
	case menu.StatusActive:
		return v1.MenuStatus_MenuStatus_Active
	case menu.StatusInactive:
		return v1.MenuStatus_MenuStatus_InActive
	default:
		return v1.MenuStatus_MenuStatus_InActive
	}
}

func toEntMenuStatus(data v1.MenuStatus) menu.Status {
	switch data {
	case v1.MenuStatus_MenuStatus_Active:
		return menu.StatusActive
	case v1.MenuStatus_MenuStatus_InActive:
		return menu.StatusInactive
	default:
		return menu.StatusInactive
	}
}

func toMenuType(data menu.Type) v1.MenuType {
	switch data {
	case menu.TypeDirectory:
		return v1.MenuType_MenuType_Directory
	case menu.TypeMenu:
		return v1.MenuType_MenuType_Menu
	case menu.TypeButton:
		return v1.MenuType_MenuType_Button
	default:
		return v1.MenuType_MenuType_Menu
	}
}

func toEntMenuType(data v1.MenuType) menu.Type {
	switch data {
	case v1.MenuType_MenuType_Directory:
		return menu.TypeDirectory
	case v1.MenuType_MenuType_Menu:
		return menu.TypeMenu
	case v1.MenuType_MenuType_Button:
		return menu.TypeButton
	default:
		return menu.TypeMenu
	}
}
