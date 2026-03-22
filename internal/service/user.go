package service

import (
	"context"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
)

type UserService struct {
	v1.UnimplementedUserServiceServer

	uc  *biz.UserUsecase
	sys *biz.SystemUsecase
}

func NewUserService(uc *biz.UserUsecase, sys *biz.SystemUsecase) *UserService {
	return &UserService{
		uc:  uc,
		sys: sys,
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
	rsp := &v1.MeInfoResponse{
		User: userInfo,
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

func (u *UserService) ListUser(ctx context.Context, req *v1.ListUserRequest) (*v1.ListUserResponse, error) {
	if err := u.sys.Check(ctx, constant.UserView); err != nil {
		return nil, err
	}
	users, total, err := u.uc.ListUsers(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ListUserResponse{
		Users:    users,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
	}, nil
}

func (u *UserService) UserInfo(ctx context.Context, req *v1.UserInfoRequest) (*v1.UserInfoResponse, error) {
	if err := u.sys.Check(ctx, constant.UserView); err != nil {
		return nil, err
	}
	userInfo, err := u.uc.UserInfo(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &v1.UserInfoResponse{User: userInfo}, nil
}

func (u *UserService) AddUser(ctx context.Context, req *v1.AddUserRequest) (*v1.AddUserResponse, error) {
	if err := u.sys.Check(ctx, constant.UserAdd); err != nil {
		return nil, err
	}
	userInfo, err := u.uc.AddUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.AddUserResponse{User: userInfo}, nil
}

func (u *UserService) EditUser(ctx context.Context, req *v1.EditUserRequest) (*v1.EditUserResponse, error) {
	if err := u.sys.Check(ctx, constant.UserEdit); err != nil {
		return nil, err
	}
	userInfo, err := u.uc.EditUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.EditUserResponse{User: userInfo}, nil
}

func (u *UserService) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	if err := u.sys.Check(ctx, constant.UserDelete); err != nil {
		return nil, err
	}
	err := u.uc.DeleteUser(ctx, req.UserIds)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteUserResponse{}, nil
}
