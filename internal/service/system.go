package service

import (
	"context"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
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
