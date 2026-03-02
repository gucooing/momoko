package service

import (
	"context"
	"google.golang.org/protobuf/types/known/durationpb"
	"momoko/api/gen/auth/v1"
	"time"
)

type AuthService struct {
	v1.UnimplementedAuthServiceServer
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	return &v1.LoginResponse{
		AccessToken:  "token_user1_114514_1872430676_111",
		RefreshToken: "114514111",
		ExpiresIn:    durationpb.New(time.Hour),
	}, nil
}
