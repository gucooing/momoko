package biz

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/user"
)

func encodePassword(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}

type UserRepo interface {
	FindByName(ctx context.Context, name string) (*ent.User, error)
	FindByID(ctx context.Context, id string) (*ent.User, error)
}

type UserUsecase struct {
	user UserRepo
}

func NewUserUsecase(user UserRepo) *UserUsecase {
	return &UserUsecase{user: user}
}

func (u *UserUsecase) LoginByUsername(ctx context.Context, username, password string) (*ent.User, error) {
	userInfo, err := u.user.FindByName(ctx, username)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	if userInfo.Password != encodePassword(password) {
		return nil, ErrInvalidPassword
	}
	return userInfo, nil
}

func (u *UserUsecase) UserInfo(ctx context.Context, userId string) (*v1.UserInfo, error) {
	userDb, err := u.user.FindByID(ctx, userId)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	return u.toUserInfo(userDb), nil
}

func (u *UserUsecase) toUserInfo(user *ent.User) *v1.UserInfo {
	info := &v1.UserInfo{
		UserId:     user.ID,
		Bio:        user.Bio,
		CreateTime: timestamppb.New(user.CreateTime),
		Email:      user.Email,
		Avatar:     user.Avatar,
		Name:       user.Name,
		Status:     u.toUserStatus(user.Status),
		Tags:       user.Tags,
		UpdateTime: timestamppb.New(user.UpdateTime),
		Username:   user.Username,
		RoleName:   "",
	}
	if role := user.Edges.Role; role != nil {
		info.RoleName = role.Name
	}

	return info
}

func (u *UserUsecase) toUserStatus(e user.Status) v1.UserStatus {
	switch e {
	case user.StatusActive:
		return v1.UserStatus_Active
	case user.StatusInactive:
		return v1.UserStatus_InActive
	case user.StatusFreeze:
		return v1.UserStatus_Freeze
	default:
		return v1.UserStatus_InActive
	}
}
