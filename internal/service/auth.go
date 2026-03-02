package service

import (
	"context"

	"google.golang.org/protobuf/types/known/durationpb"

	"momoko/api/gen/auth/v1"
	"momoko/internal/biz"

	"time"
)

type AuthService struct {
	v1.UnimplementedAuthServiceServer

	uc *biz.AuthUsecase
}

func NewAuthService(uc *biz.AuthUsecase) *AuthService {
	return &AuthService{
		uc: uc,
	}
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {

	switch v := req.Identity.(type) {
	case *v1.LoginRequest_Username:
	case *v1.LoginRequest_Email:
	}

	return &v1.LoginResponse{
		AccessToken:  "token_user1_114514_1872430676_111",
		RefreshToken: "114514111",
		ExpiresIn:    durationpb.New(time.Hour),
	}, nil
}
