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

	uc           *biz.SystemUsecase
	config       *biz.ConfigUsecase
	operationLog *biz.OperationLogUsecase
}

func NewSystemService(uc *biz.SystemUsecase, config *biz.ConfigUsecase, operationLog *biz.OperationLogUsecase) *SystemService {
	return &SystemService{
		uc:           uc,
		config:       config,
		operationLog: operationLog,
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

func (s *SystemService) AdminRoles(ctx context.Context, req *v1.AdminRolesRequest) (*v1.AdminRolesResponse, error) {
	if err := s.uc.Check(ctx, constant.RoleView); err != nil {
		return nil, err
	}
	roles, total, err := s.uc.GetAllRoles(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.AdminRolesResponse{
		Roles:    roles,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
	}, nil
}

func (s *SystemService) AdminRole(ctx context.Context, req *v1.AdminRoleRequest) (*v1.AdminRoleResponse, error) {
	if err := s.uc.Check(ctx, constant.RoleView); err != nil {
		return nil, err
	}
	roleInfo, err := s.uc.GetRole(ctx, req.RoleId)
	if err != nil {
		return nil, err
	}
	return &v1.AdminRoleResponse{Role: roleInfo}, nil
}

func (s *SystemService) AdminAddRole(ctx context.Context, req *v1.AdminAddRoleRequest) (*v1.AdminAddRoleResponse, error) {
	if err := s.uc.Check(ctx, constant.RoleAdd); err != nil {
		return nil, err
	}
	roleInfo, err := s.uc.AddRole(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.AdminAddRoleResponse{Role: roleInfo}, nil
}

func (s *SystemService) AdminEditRole(ctx context.Context, req *v1.AdminEditRoleRequest) (*v1.AdminEditRoleResponse, error) {
	if err := s.uc.Check(ctx, constant.RoleEdit); err != nil {
		return nil, err
	}
	roleInfo, err := s.uc.UpdateRole(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.AdminEditRoleResponse{Role: roleInfo}, nil
}

func (s *SystemService) AdminDeleteRole(ctx context.Context, req *v1.AdminDeleteRoleRequest) (*v1.AdminDeleteRoleResponse, error) {
	if err := s.uc.Check(ctx, constant.RoleDelete); err != nil {
		return nil, err
	}
	err := s.uc.DeleteRole(ctx, req.RoleIds)
	if err != nil {
		return nil, err
	}
	return &v1.AdminDeleteRoleResponse{}, nil
}

func (s *SystemService) LoginConfig(ctx context.Context, req *v1.LoginConfigRequest) (*v1.LoginConfigResponse, error) {
	config, err := s.config.LoginConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.LoginConfigResponse{Config: config}, nil
}

func (s *SystemService) UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.UpdateLoginConfigResponse, error) {
	if err := s.uc.Check(ctx, constant.SystemConfigEdit); err != nil {
		return nil, err
	}
	config, err := s.config.UpdateLoginConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateLoginConfigResponse{Config: config}, nil
}

func (s *SystemService) SystemOverview(ctx context.Context, req *v1.SystemOverviewRequest) (*v1.SystemOverviewResponse, error) {
	return s.uc.SystemOverview(ctx)
}

func (s *SystemService) SystemStatus(ctx context.Context, req *v1.SystemStatusRequest) (*v1.SystemStatusResponse, error) {
	return s.uc.SystemStatus(ctx, req)
}

func (s *SystemService) ListOperationLogs(ctx context.Context, req *v1.ListOperationLogsRequest) (*v1.ListOperationLogsResponse, error) {
	if err := s.uc.Check(ctx, constant.SystemConfigView); err != nil {
		return nil, err
	}
	return s.operationLog.ListOperationLogs(ctx, req)
}
