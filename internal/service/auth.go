package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"

	"momoko/api/gen/auth/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent"
	auth2 "momoko/internal/data/ent/auth"
	"momoko/pkg/auth"

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
	}
	if err != nil {
		return nil, err
	}
	deviceId := uuid.NewString()
	access, err := s.uc.NewAccessToken(ctx, user.ID, deviceId, req)
	if err != nil {
		return nil, err
	}
	refresh, err := s.uc.NewRefreshToken(ctx, user.ID, deviceId, req)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateToken(access, deviceId)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateToken(refresh, deviceId)
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
	if s.uc.VerifyToken(ctx, refreshAuth, auth2.TypeRefreshToken) {
		return nil, biz.ErrTokenInvalid
	}
	// 更新token
	access, err := s.uc.NewAccessToken(ctx,
		refreshAuth.ID,
		refreshAuth.DeviceId,
		nil)
	if err != nil {
		return nil, err
	}
	refresh, err := s.uc.NewRefreshToken(ctx,
		refreshAuth.ID,
		refreshAuth.DeviceId,
		nil)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateToken(access, refreshAuth.DeviceId)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateToken(refresh, refreshAuth.DeviceId)
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
	list, err := s.uc.ListLoginDevice(ctx, authInfo.ID)
	if err != nil {
		return nil, err
	}
	return &v1.DevicesResponse{
		Devices:  list,
		DeviceId: authInfo.DeviceId,
	}, nil
}
