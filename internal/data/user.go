package data

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
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

func (ur *userRepo) FindByID(ctx context.Context, id string) (*gen.User, error) {
	query := ur.data.db.User.Query()

	query.Where(user.IDEQ(id)).WithRole()

	return query.First(ctx)
}

func (ur *userRepo) ListUsers(ctx context.Context, page, pageSize int64, status *user.Status, username *string) ([]*gen.User, int64, error) {
	query := ur.data.db.User.Query()

	if status != nil {
		query = query.Where(user.StatusEQ(*status))
	}
	if username != nil && *username != "" {
		query = query.Where(user.UsernameContains(*username))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	users, err := query.
		WithRole().
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Order(gen.Asc(user.FieldID)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, int64(total), nil
}

func (ur *userRepo) CreateUser(ctx context.Context, userInfo *gen.User, roleId string) (*gen.User, error) {
	if roleId == "" {
		roleId = noPermissionRoleID
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
	return ur.data.db.User.UpdateOneID(userId).SetPassword(passwordHash).Save(ctx)
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

	return update.Save(ctx)
}
