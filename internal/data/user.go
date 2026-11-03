package data

import (
	"context"
	"strings"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/role"
	"momoko/internal/data/ent/gen/user"
)

type userRepo struct {
	data *Data
}

func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{
		data: data,
	}
}

func (ur *userRepo) FindByName(ctx context.Context, name string) (*gen.User, error) {
	query := ur.data.db.User.Query()

	query.Where(user.UsernameEQ(name)).WithRole()

	return query.First(ctx)
}

func (ur *userRepo) FindByEmail(ctx context.Context, email string) (*gen.User, error) {
	query := ur.data.db.User.Query()

	query.Where(user.EmailEQ(email)).WithRole()

	return query.First(ctx)
}

func (ur *userRepo) FindByID(ctx context.Context, id string) (*gen.User, error) {
	query := ur.data.db.User.Query()

	query.Where(user.IDEQ(id)).WithRole()

	return query.First(ctx)
}

func (ur *userRepo) ListUsers(ctx context.Context, page, pageSize int64, filter biz.UserListFilter) ([]*gen.User, int64, *biz.UserListCounts, error) {
	query := ur.data.db.User.Query()

	if filter.Keywords != nil && strings.TrimSpace(*filter.Keywords) != "" {
		kw := strings.TrimSpace(*filter.Keywords)
		query = query.Where(user.Or(
			user.UsernameContains(kw),
			user.NameContains(kw),
			user.EmailContains(kw),
			user.LastLoginIPHasPrefix(kw),
		))
	}
	if filter.RoleID != nil && *filter.RoleID != "" {
		query = query.Where(user.HasRoleWith(role.IDEQ(*filter.RoleID)))
	}

	counts := &biz.UserListCounts{}
	active, err := query.Clone().Where(user.StatusEQ(user.StatusActive)).Count(ctx)
	if err != nil {
		return nil, 0, nil, err
	}
	inactive, err := query.Clone().Where(user.StatusNEQ(user.StatusActive)).Count(ctx)
	if err != nil {
		return nil, 0, nil, err
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayLogin, err := query.Clone().Where(user.LastLoginAtGTE(today)).Count(ctx)
	if err != nil {
		return nil, 0, nil, err
	}
	counts.Active, counts.Inactive, counts.TodayLogin = int64(active), int64(inactive), int64(todayLogin)

	if filter.Status != nil {
		query = query.Where(user.StatusEQ(*filter.Status))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	users, err := query.
		WithRole().
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Order(gen.Asc(user.FieldID)).
		All(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	return users, int64(total), counts, nil
}

func (ur *userRepo) CreateUser(ctx context.Context, userInfo *gen.User, roleId string) (*gen.User, error) {
	if roleId == "" {
		roleId = noPermissionRoleID // 注册时默认给予无权限
	}
	builder := ur.data.db.User.Create().
		SetID(userInfo.ID).
		SetUsername(userInfo.Username).
		SetPassword(userInfo.Password).
		SetEmail(userInfo.Email).
		SetStatus(userInfo.Status).
		SetAvatar(userInfo.Avatar).
		SetBio(userInfo.Bio).
		SetName(userInfo.Name).
		SetTags(userInfo.Tags).
		SetRoleID(roleId)

	return builder.Save(ctx)
}

func (ur *userRepo) UpdateUser(ctx context.Context, userInfo *gen.User, roleId string) (*gen.User, error) {
	if roleId == "" {
		roleId = noPermissionRoleID
	}
	builder := ur.data.db.User.UpdateOneID(userInfo.ID).
		SetEmail(userInfo.Email).
		SetStatus(userInfo.Status).
		SetAvatar(userInfo.Avatar).
		SetBio(userInfo.Bio).
		SetName(userInfo.Name).
		SetTags(userInfo.Tags).
		SetRoleID(roleId)

	return builder.Save(ctx)
}

func (ur *userRepo) DeleteUser(ctx context.Context, userIds []string) error {
	_, err := ur.data.db.User.Delete().
		Where(user.IDIn(userIds...)).
		Exec(ctx)

	return err
}

func (ur *userRepo) UpdatePassword(ctx context.Context, userId string, passwordHash string) (*gen.User, error) {
	return ur.data.db.User.UpdateOneID(userId).
		SetPassword(passwordHash).
		SetPasswordUpdatedAt(time.Now()).
		Save(ctx)
}

// TouchLastLogin 记录用户最近一次登录的时间、来源 IP 与方式（password / email_code）。
func (ur *userRepo) TouchLastLogin(ctx context.Context, userId, ip, method string) error {
	return ur.data.db.User.UpdateOneID(userId).
		SetLastLoginAt(time.Now()).
		SetLastLoginIP(ip).
		SetLastLoginMethod(method).
		Exec(ctx)
}

func (ur *userRepo) BatchUpdateUsers(ctx context.Context, userIds []string, status *user.Status, roleId *string) (int64, error) {
	update := ur.data.db.User.Update().Where(user.IDIn(userIds...))
	if status != nil {
		update.SetStatus(*status)
	}
	if roleId != nil {
		update.SetRoleID(*roleId)
	}
	n, err := update.Save(ctx)
	return int64(n), err
}

func (ur *userRepo) UpdateMe(ctx context.Context, userId string, req *v1.UpdateMeRequest) (*gen.User, error) {
	update := ur.data.db.User.UpdateOneID(userId)
	if req.Bio != nil {
		update.SetBio(*req.Bio)
	}
	if req.Email != nil {
		update.SetEmail(*req.Email)
	}
	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Tags != nil {
		update.SetTags(*req.Tags)
	}
	if req.Avatar != nil {
		update.SetAvatar(*req.Avatar)
	}
	if req.Username != nil {
		update.SetUsername(*req.Username)
	}
	if req.Phone != nil {
		update.SetPhone(strings.TrimSpace(*req.Phone))
	}

	return update.Save(ctx)
}
