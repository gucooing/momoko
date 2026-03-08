package biz

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

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
	ListUsers(ctx context.Context, page, pageSize int64, status *user.Status, username *string) ([]*ent.User, int64, error)
	CreateUser(ctx context.Context, userInfo *ent.User, roleId string) (*ent.User, error)
	UpdateUser(ctx context.Context, userInfo *ent.User, roleId string) (*ent.User, error)
	DeleteUser(ctx context.Context, userIds []string) error
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
	if userInfo.Status != user.StatusActive {
		return nil, ErrUserInactive
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

func (u *UserUsecase) ListUsers(ctx context.Context, req *v1.ListUserRequest) ([]*v1.UserInfo, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 500 {
		req.PageSize = 500
	}

	var status *user.Status
	if req.Status != nil {
		status = new(u.toEntUserStatus(*req.Status))
	}

	userInfos, total, err := u.user.ListUsers(ctx, req.Page, req.PageSize, status, req.Username)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}

	users := make([]*v1.UserInfo, 0, len(userInfos))
	for _, userInfo := range userInfos {
		users = append(users, u.toUserInfo(userInfo))
	}

	return users, total, nil
}

func (u *UserUsecase) AddUser(ctx context.Context, req *v1.AddUserRequest) (*v1.UserInfo, error) {
	userInfo, err := u.user.CreateUser(ctx, &ent.User{
		ID:       fmt.Sprintf("user_z:%06d_%s", time.Now().Unix()%1000000, uuid.NewString()[:8]),
		Username: req.Username,
		Password: encodePassword(req.Password),
		Email:    req.Email,
		Status:   u.toEntUserStatus(req.Status),
		Avatar:   req.Avatar,
		Bio:      req.Bio,
		Name:     req.Name,
		Tags:     req.Tags,
	}, req.RoleId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return u.toUserInfo(userInfo), nil
}

func (u *UserUsecase) EditUser(ctx context.Context, req *v1.EditUserRequest) (*v1.UserInfo, error) {
	userInfo, err := u.user.UpdateUser(ctx, &ent.User{
		ID:     req.UserId,
		Email:  req.Email,
		Status: u.toEntUserStatus(req.Status),
		Avatar: req.Avatar,
		Bio:    req.Bio,
		Name:   req.Name,
		Tags:   req.Tags,
	}, req.RoleId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return u.toUserInfo(userInfo), nil
}

func (u *UserUsecase) DeleteUser(ctx context.Context, userIds []string) error {
	err := u.user.DeleteUser(ctx, userIds)
	if err != nil {
		return ErrSystem(err)
	}
	return nil
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
		RoleId:     "",
	}
	if role := user.Edges.Role; role != nil {
		info.RoleName = role.Name
		info.RoleId = role.ID
	}

	return info
}

func (u *UserUsecase) toUserStatus(e user.Status) v1.UserStatus {
	switch e {
	case user.StatusActive:
		return v1.UserStatus_Active
	case user.StatusInactive:
		return v1.UserStatus_InActive
	default:
		return v1.UserStatus_InActive
	}
}

func (u *UserUsecase) toEntUserStatus(e v1.UserStatus) user.Status {
	switch e {
	case v1.UserStatus_Active:
		return user.StatusActive
	case v1.UserStatus_InActive:
		return user.StatusInactive
	default:
		return user.StatusInactive
	}
}
