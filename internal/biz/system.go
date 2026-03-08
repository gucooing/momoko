package biz

import (
	"context"

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

func toMenuInfos(menus []*ent.Menu) ([]*v1.MenuInfo, []string) {
	menuInfos := make([]*v1.MenuInfo, 0)
	permissions := make([]string, 0)

	menuInfoMap := make(map[string]*v1.MenuInfo, len(menus))
	permissionSet := make(map[string]struct{}, len(menus))

	for _, item := range menus {
		if item.Permission != "" {
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
		if item.Permission == "" {
			continue
		}
		permissions[constant.Permissions(item.Permission)] = struct{}{}
	}
	return permissions
}

func toMenuInfo(data *ent.Menu) *v1.MenuInfo {
	return &v1.MenuInfo{
		Id:         data.ID,
		Icon:       data.Icon,
		IsSystem:   data.IsSystem,
		Order:      int32(data.Order),
		ParentId:   data.ParentID,
		Path:       data.Path,
		Status:     toMenuStatus(data.Status),
		Title:      data.Title,
		Type:       toMenuType(data.Type),
		CreateTime: timestamppb.New(data.CreateTime),
		UpdateTime: timestamppb.New(data.UpdateTime),
		Children:   make([]*v1.MenuInfo, 0),
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
