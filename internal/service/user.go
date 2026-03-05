package service

import (
	"context"

	"momoko/api/gen/user/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type UserService struct {
	v1.UnimplementedUserServiceServer

	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{
		uc: uc,
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
