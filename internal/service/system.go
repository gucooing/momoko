package service

import (
	"context"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
)

type SystemService struct {
	v1.UnimplementedSystemServer

	uc *biz.SystemUsecase
}

func NewSystemService(uc *biz.SystemUsecase) *SystemService {
	return &SystemService{
		uc: uc,
	}
}

func (s *SystemService) MePermissions(ctx context.Context, req *v1.MePermissionsRequest) (*v1.MePermissionsResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	menus, permissions, err := s.uc.GetMenusByUserID(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}

	return &v1.MePermissionsResponse{
		Permissions: permissions,
		Menus:       menus,
	}, nil
}

func (s *SystemService) AdminPermissions(ctx context.Context, req *v1.AdminPermissionsRequest) (*v1.AdminPermissionsResponse, error) {
	if err := s.uc.Check(ctx, constant.MenuView); err != nil {
		return nil, err
	}
	menus, _, err := s.uc.GetAllMenus(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.AdminPermissionsResponse{Menus: menus}, nil
}

func (s *SystemService) AdminPermissionsInfo(ctx context.Context, req *v1.AdminPermissionsInfoRequest) (*v1.AdminPermissionsInfoResponse, error) {
	if err := s.uc.Check(ctx, constant.MenuView); err != nil {
		return nil, err
	}
	menu, err := s.uc.GetMenu(ctx, req.MenuId)
	if err != nil {
		return nil, err
	}
	return &v1.AdminPermissionsInfoResponse{Menu: menu}, nil
}

func (s *SystemService) AdminAddPermissions(ctx context.Context, req *v1.AdminAddPermissionsRequest) (*v1.AdminAddPermissionsResponse, error) {
	if err := s.uc.Check(ctx, constant.MenuAdd); err != nil {
		return nil, err
	}
	err := s.uc.AddMenu(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.AdminAddPermissionsResponse{}, nil
}

func (s *SystemService) AdminEditPermissions(ctx context.Context, req *v1.AdminEditPermissionsRequest) (*v1.AdminEditPermissionsResponse, error) {
	if err := s.uc.Check(ctx, constant.MenuEdit); err != nil {
		return nil, err
	}
	err := s.uc.UpdateMenu(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.AdminEditPermissionsResponse{}, nil
}

func (s *SystemService) AdminDeletePermissions(ctx context.Context, req *v1.AdminDeletePermissionsRequest) (*v1.AdminDeletePermissionsResponse, error) {
	if err := s.uc.Check(ctx, constant.MenuDelete); err != nil {
		return nil, err
	}
	err := s.uc.DeleteMenu(ctx, req.MenuId)
	if err != nil {
		return nil, err
	}
	return &v1.AdminDeletePermissionsResponse{}, nil
}
