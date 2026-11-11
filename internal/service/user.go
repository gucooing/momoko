package service

import (
	"context"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type UserService struct {
	v1.UnimplementedUserServiceServer

	uc  *biz.UserUsecase
	log *biz.OperationLogUsecase
}

func NewUserService(uc *biz.UserUsecase, operationLog *biz.OperationLogUsecase) *UserService {
	return &UserService{
		uc:  uc,
		log: operationLog,
	}
}

func (u *UserService) MeInfo(ctx context.Context, req *v1.MeInfoRequest) (*v1.MeInfoResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	userInfo, err := u.uc.UserInfo(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	appearance, currentTheme, siteTheme, err := u.uc.MeAppearance(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	rsp := &v1.MeInfoResponse{
		User:         userInfo,
		Appearance:   appearance,
		CurrentTheme: currentTheme,
		SiteTheme:    siteTheme,
	}

	return rsp, nil
}

func (u *UserService) UpdateMe(ctx context.Context, req *v1.UpdateMeRequest) (*v1.UpdateMeResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := u.uc.UpdateMe(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateMeResponse{User: info}, nil
}

func (u *UserService) UpdateMyAppearance(ctx context.Context, req *v1.UpdateMyAppearanceRequest) (*v1.UpdateMyAppearanceResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	saved, err := u.uc.UpdateMyAppearance(ctx, authCtx.UserID, req.GetAppearance())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateMyAppearanceResponse{Appearance: saved}, nil
}

func (u *UserService) MyLoginLogs(ctx context.Context, req *v1.MyLoginLogsRequest) (*v1.MyLoginLogsResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	logs, err := u.log.MyLoginLogs(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (u *UserService) ListUser(ctx context.Context, req *v1.ListUserRequest) (*v1.ListUserResponse, error) {
	users, total, counts, err := u.uc.ListUsers(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ListUserResponse{
		Users:           users,
		Page:            req.Page,
		PageSize:        req.PageSize,
		Total:           total,
		ActiveCount:     counts.Active,
		InactiveCount:   counts.Inactive,
		TodayLoginCount: counts.TodayLogin,
	}, nil
}

func (u *UserService) UserInfo(ctx context.Context, req *v1.UserInfoRequest) (*v1.UserInfoResponse, error) {
	userInfo, err := u.uc.UserInfo(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &v1.UserInfoResponse{User: userInfo}, nil
}

func (u *UserService) AddUser(ctx context.Context, req *v1.AddUserRequest) (*v1.AddUserResponse, error) {
	userInfo, err := u.uc.AddUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.AddUserResponse{User: userInfo}, nil
}

func (u *UserService) EditUser(ctx context.Context, req *v1.EditUserRequest) (*v1.EditUserResponse, error) {
	userInfo, err := u.uc.EditUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.EditUserResponse{User: userInfo}, nil
}

func (u *UserService) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	err := u.uc.DeleteUser(ctx, req.UserIds)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteUserResponse{}, nil
}

func (u *UserService) BatchUpdateUser(ctx context.Context, req *v1.BatchUpdateUserRequest) (*v1.BatchUpdateUserResponse, error) {
	updated, skipped, err := u.uc.BatchUpdateUsers(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.BatchUpdateUserResponse{Updated: updated, Skipped: skipped}, nil
}

func (u *UserService) ResetUserPassword(ctx context.Context, req *v1.ResetUserPasswordRequest) (*v1.ResetUserPasswordResponse, error) {
	if err := u.uc.ResetPassword(ctx, req.UserId, req.Password); err != nil {
		return nil, err
	}
	return &v1.ResetUserPasswordResponse{}, nil
}
