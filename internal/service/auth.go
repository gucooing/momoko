package service

import (
	"context"

	"google.golang.org/protobuf/types/known/durationpb"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent"
	auth2 "momoko/internal/data/ent/auth"
	"momoko/pkg/auth"
	"momoko/pkg/response"

	"time"
)

type AuthService struct {
	v1.UnimplementedAuthServiceServer

	uc   *biz.AuthUsecase
	user *biz.UserUsecase
}

func NewAuthService(uc *biz.AuthUsecase, user *biz.UserUsecase) *AuthService {
	return &AuthService{
		uc:   uc,
		user: user,
	}
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	var (
		user *ent.User
		err  error
	)
	switch req.Identity.(type) {
	case *v1.LoginRequest_Username:
		user, err = s.user.LoginByUsername(ctx, req.GetUsername(), req.GetPassword())
	case *v1.LoginRequest_Email:
		return nil, response.BadRequest(500, "Email validation failed")
	}
	if err != nil {
		return nil, err
	}
	access, err := s.uc.NewAccessToken(ctx, user.ID, req)
	if err != nil {
		return nil, err
	}
	refresh, err := s.uc.NewRefreshToken(ctx, user.ID, req)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateToken(access)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateToken(refresh)
	if err != nil {
		return nil, err
	}

	return &v1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    durationpb.New(time.Hour),
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req *v1.RefreshRequest) (*v1.RefreshResponse, error) {
	refreshAuth, err := auth.ParseToken(req.RefreshToken)
	if err != nil {
		return nil, biz.ErrTokenInvalid
	}
	if !s.uc.VerifyToken(ctx, refreshAuth, auth2.TypeRefreshToken) {
		return nil, biz.ErrTokenInvalid
	}
	// 更新token
	access, refresh, err := s.uc.RefreshToken(ctx, refreshAuth.UserID, refreshAuth.DeviceId)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateToken(access)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateToken(refresh)
	if err != nil {
		return nil, err
	}

	return &v1.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    durationpb.New(time.Hour),
	}, nil
}

func (s *AuthService) Devices(ctx context.Context, req *v1.DevicesRequest) (*v1.DevicesResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	list, err := s.uc.ListLoginDevice(ctx, authInfo.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.DevicesResponse{
		Devices:  list,
		DeviceId: authInfo.DeviceId,
	}, nil
}

func (s *AuthService) UpdatePassword(ctx context.Context, req *v1.UpdatePasswordRequest) (*v1.UpdatePasswordResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	_, err := s.user.UpdatePassword(ctx, authInfo.UserID, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePasswordResponse{}, nil
}
