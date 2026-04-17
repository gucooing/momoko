package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/user"
	"momoko/pkg/auth"
	"momoko/pkg/avatar"
	"momoko/pkg/response"
)

type UserRepo interface {
	FindByName(ctx context.Context, name string) (*gen.User, error)
	FindByID(ctx context.Context, id string) (*gen.User, error)
	ListUsers(ctx context.Context, page, pageSize int64, status *user.Status, username *string) ([]*gen.User, int64, error)
	CreateUser(ctx context.Context, userInfo *gen.User, roleId string) (*gen.User, error)
	UpdateUser(ctx context.Context, userInfo *gen.User, roleId string) (*gen.User, error)
	DeleteUser(ctx context.Context, userIds []string) error
	UpdatePassword(ctx context.Context, userId string, passwordHash string) (*gen.User, error)
	UpdateMe(ctx context.Context, userId string, req *v1.UpdateMeRequest) (*gen.User, error)
}

type UserUsecase struct {
	user   UserRepo
	auth   AuthRepo
	avatar *avatar.Manager
}

func NewUserUsecase(user UserRepo, auth AuthRepo, avatarManager *avatar.Manager) *UserUsecase {
	return &UserUsecase{
		user:   user,
		auth:   auth,
		avatar: avatarManager,
	}
}

func (u *UserUsecase) LoginByUsername(ctx context.Context, username, password string) (*gen.User, error) {
	userInfo, err := u.user.FindByName(ctx, username)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	if userInfo.Password != auth.EncodePassword(password) {
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
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	return u.toUserInfo(userDb), nil
}

func (u *UserUsecase) UpdateMe(ctx context.Context, userId string, req *v1.UpdateMeRequest) (*v1.UserInfo, error) {
	commit := func() error { return nil }
	rollback := func() {}

	if req.Avatar != nil {
		newAvatar, nextCommit, nextRollback, err := u.avatar.Prepare(userId, *req.Avatar)
		if err != nil {
			return nil, response.BadRequest(500, err.Error())
		}

		req.Avatar = &newAvatar
		commit = nextCommit
		rollback = nextRollback
	}

	info, err := u.user.UpdateMe(ctx, userId, req)
	if err != nil {
		rollback()
		return nil, ErrSystem(err)
	}
	if err := commit(); err != nil {
		return nil, ErrSystem(err)
	}
	return u.toUserInfo(info), nil
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
	userInfo, err := u.user.CreateUser(ctx, &gen.User{
		ID:       fmt.Sprintf("user_z:%06d_%s", time.Now().Unix()%1000000, uuid.NewString()[:8]),
		Username: req.Username,
		Password: auth.EncodePassword(req.Password),
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
	userInfo, err := u.user.UpdateUser(ctx, &gen.User{
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

	for _, userID := range userIds {
		_ = u.avatar.DeleteByUserID(userID)
	}
	return nil
}

func (u *UserUsecase) UpdatePassword(ctx context.Context, userId, oldPassword, newPassword string) (*v1.UserInfo, error) {
	userDb, err := u.user.FindByID(ctx, userId)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	if auth.EncodePassword(oldPassword) != userDb.Password {
		return nil, ErrInvalidPassword
	}
	if err = u.auth.DeleteAuth(ctx, userId, nil); err != nil {
		return nil, ErrSystem(err)
	}
	info, err := u.user.UpdatePassword(ctx, userId, auth.EncodePassword(newPassword))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return u.toUserInfo(info), nil
}

func (u *UserUsecase) toUserInfo(user *gen.User) *v1.UserInfo {
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
