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
	FindByEmail(ctx context.Context, email string) (*gen.User, error)
	FindByID(ctx context.Context, id string) (*gen.User, error)
	// ListUsers 分页列出；counts 按关键字、角色统计（不受状态筛选影响）。
	ListUsers(ctx context.Context, page, pageSize int64, filter UserListFilter) ([]*gen.User, int64, *UserListCounts, error)
	CreateUser(ctx context.Context, userInfo *gen.User, roleId string) (*gen.User, error)
	UpdateUser(ctx context.Context, userInfo *gen.User, roleId string) (*gen.User, error)
	DeleteUser(ctx context.Context, userIds []string) error
	UpdatePassword(ctx context.Context, userId string, passwordHash string) (*gen.User, error)
	UpdateMe(ctx context.Context, userId string, req *v1.UpdateMeRequest) (*gen.User, error)
	TouchLastLogin(ctx context.Context, userId, ip, method string) error
	// BatchUpdateUsers 批量修改状态 / 角色（nil 表示不修改），返回实际修改的行数。
	BatchUpdateUsers(ctx context.Context, userIds []string, status *user.Status, roleId *string) (int64, error)
}

// BuiltinAdminUserID 初始化时创建的内置超级管理员账号 id（见 data/initialize.go）。
const BuiltinAdminUserID = "admin_1"

var (
	ErrBuiltinUserProtected = response.BadRequest(500, "内置超级管理员不能删除、停用或修改角色")
	ErrUserIdsEmpty         = response.BadRequest(500, "请选择用户")
	ErrBatchUpdateEmpty     = response.BadRequest(500, "没有需要修改的内容")
)

// UserListFilter 用户列表筛选条件。
type UserListFilter struct {
	Status *user.Status
	// Keywords 匹配用户名 / 昵称 / 邮箱 / 最近登录 IP
	Keywords *string
	RoleID   *string
}

// UserListCounts 用户列表统计。
type UserListCounts struct {
	Active     int64
	Inactive   int64
	TodayLogin int64
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

func (u *UserUsecase) FindByEmail(ctx context.Context, email string) (*gen.User, error) {
	return u.user.FindByEmail(ctx, email)
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

func (u *UserUsecase) ListUsers(ctx context.Context, req *v1.ListUserRequest) ([]*v1.UserInfo, int64, *UserListCounts, error) {
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

	userInfos, total, counts, err := u.user.ListUsers(ctx, req.Page, req.PageSize, UserListFilter{
		Status:   status,
		Keywords: req.Username,
		RoleID:   req.RoleId,
	})
	if err != nil {
		return nil, 0, nil, ErrSystem(err)
	}

	users := make([]*v1.UserInfo, 0, len(userInfos))
	for _, userInfo := range userInfos {
		users = append(users, u.toUserInfo(userInfo))
	}

	return users, total, counts, nil
}

func (u *UserUsecase) AddUser(ctx context.Context, req *v1.AddUserRequest) (*v1.UserInfo, error) {
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, ErrSystem(err)
	}
	userInfo, err := u.user.CreateUser(ctx, &gen.User{
		ID:       fmt.Sprintf("user_z:%06d_%s", time.Now().Unix()%1000000, uuid.NewString()[:8]),
		Username: req.Username,
		Password: passwordHash,
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
	if req.UserId == BuiltinAdminUserID {
		current, err := u.user.FindByID(ctx, req.UserId)
		if err != nil {
			return nil, ErrSystem(err)
		}
		currentRole := ""
		if current.Edges.Role != nil {
			currentRole = current.Edges.Role.ID
		}
		if req.Status != v1.UserStatus_Active || (req.RoleId != "" && req.RoleId != currentRole) {
			return nil, ErrBuiltinUserProtected
		}
		req.RoleId = currentRole
	}
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
	for _, id := range userIds {
		if id == BuiltinAdminUserID {
			return ErrBuiltinUserProtected
		}
	}
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
	if !auth.VerifyPassword(oldPassword, userDb.Password) {
		return nil, ErrInvalidPassword
	}
	if err = u.auth.DeleteAuth(ctx, userId, nil); err != nil {
		return nil, ErrSystem(err)
	}
	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return nil, ErrSystem(err)
	}
	info, err := u.user.UpdatePassword(ctx, userId, newHash)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return u.toUserInfo(info), nil
}

// BatchUpdateUsers 批量启用 / 停用 / 分配角色；内置超级管理员自动跳过。停用后立即踢下线。
func (u *UserUsecase) BatchUpdateUsers(ctx context.Context, req *v1.BatchUpdateUserRequest) (int64, int64, error) {
	if req.Status == nil && (req.RoleId == nil || *req.RoleId == "") {
		return 0, 0, ErrBatchUpdateEmpty
	}
	ids := make([]string, 0, len(req.UserIds))
	seen := make(map[string]struct{}, len(req.UserIds))
	var skipped int64
	for _, id := range req.UserIds {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if id == BuiltinAdminUserID {
			skipped++
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		if skipped > 0 {
			return 0, skipped, ErrBuiltinUserProtected
		}
		return 0, 0, ErrUserIdsEmpty
	}
	var status *user.Status
	if req.Status != nil {
		status = new(u.toEntUserStatus(*req.Status))
	}
	var roleID *string
	if req.RoleId != nil && *req.RoleId != "" {
		roleID = req.RoleId
	}
	updated, err := u.user.BatchUpdateUsers(ctx, ids, status, roleID)
	if err != nil {
		return 0, skipped, ErrSystem(err)
	}
	if status != nil && *status != user.StatusActive {
		for _, id := range ids {
			_ = u.auth.DeleteAuth(ctx, id, nil)
		}
	}
	return updated, skipped, nil
}

// ResetPassword 管理员直接为用户设置新密码，并让该用户所有会话下线。
func (u *UserUsecase) ResetPassword(ctx context.Context, userId, password string) error {
	if password == "" {
		return ErrPasswordEmpty
	}
	if _, err := u.user.FindByID(ctx, userId); err != nil {
		if gen.IsNotFound(err) {
			return ErrAdminNotFound
		}
		return ErrSystem(err)
	}
	newHash, err := auth.HashPassword(password)
	if err != nil {
		return ErrSystem(err)
	}
	if _, err := u.user.UpdatePassword(ctx, userId, newHash); err != nil {
		return ErrSystem(err)
	}
	if authCtx, ok := auth.FromContext(ctx); !ok || authCtx.UserID != userId {
		_ = u.auth.DeleteAuth(ctx, userId, nil)
	}
	return nil
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
	info.Phone = user.Phone
	if user.LastLoginAt != nil {
		info.LastLoginTime = timestamppb.New(*user.LastLoginAt)
	}
	info.LastLoginIp = user.LastLoginIP
	info.LastLoginMethod = user.LastLoginMethod
	info.IsBuiltin = user.ID == BuiltinAdminUserID
	if user.PasswordUpdatedAt != nil {
		info.PasswordUpdateTime = timestamppb.New(*user.PasswordUpdatedAt)
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
