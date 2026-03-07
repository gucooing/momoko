package biz

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/menu"
)

type SystemUsecase struct {
	sys      SystemRepo
	userRepo UserRepo
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

func (s *SystemUsecase) GetMenusByUserID(ctx context.Context, userID string) ([]*v1.MenuInfo, []string, error) {
	userInfo, err := s.userRepo.FindWithRoleByID(ctx, userID)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	if userInfo.Edges.Role == nil {
		return nil, nil, ErrUserNoRole
	}
	menuInfos, err := s.sys.GetMenusByRoleId(ctx, userInfo.Edges.Role.ID)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	menus, permissions := toMenuInfos(menuInfos)
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
