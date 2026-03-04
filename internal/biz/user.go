package biz

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"momoko/internal/data/ent"
)

func encodePassword(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}

type UserRepo interface {
	FindByName(ctx context.Context, name string) (*ent.User, error)
}

type UserUsecase struct {
	user UserRepo
}

func NewUserUsecase(user UserRepo) *UserUsecase {
	return &UserUsecase{user: user}
}

func (u *UserUsecase) LoginByUsername(ctx context.Context, username, password string) (*ent.User, error) {
	user, err := u.user.FindByName(ctx, username)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	if user.Password != encodePassword(password) {
		return nil, ErrInvalidPassword
	}
	return user, nil
}
