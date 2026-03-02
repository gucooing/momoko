package biz

import (
	"context"
)

type AuthRepo interface {
	FindByName(ctx context.Context)
}

type AuthUsecase struct {
	auth AuthRepo
}

func NewAuthUsecase(repo AuthRepo) *AuthUsecase {
	return &AuthUsecase{auth: repo}
}

func (a *AuthUsecase) LoginByUsername(ctx context.Context, username, password string) (*Admin, error) {

}
