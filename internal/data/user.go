package data

import (
	"context"

	"momoko/internal/biz"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/user"
)

type userRepo struct {
	data *Data
}

func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{
		data: data,
	}
}

func (ur *userRepo) FindByName(ctx context.Context, name string) (*ent.User, error) {
	query := ur.data.db.User.Query()

	query.Where(user.UsernameEQ(name))

	return query.First(ctx)
}

func (ur *userRepo) FindByID(ctx context.Context, id string) (*ent.User, error) {
	query := ur.data.db.User.Query()

	query.Where(user.IDEQ(id))

	return query.First(ctx)
}

func (ur *userRepo) FindWithRoleByID(ctx context.Context, id string) (*ent.User, error) {
	query := ur.data.db.User.Query()

	query.Where(user.IDEQ(id)).WithRole()

	return query.First(ctx)
}
