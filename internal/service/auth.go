package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"

	"momoko/api/gen/auth/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent"
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
	access, err := s.uc.NewAccessToken(ctx, user.Username, deviceId, "")
	if err != nil {
		return nil, err
	}
	refresh, err := s.uc.NewRefreshToken(ctx, user.Username, deviceId, "")
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
