package data

import (
	"context"

	"momoko/internal/biz"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/role"
)

type systemRepo struct {
	data *Data
}

func NewSystemRepo(data *Data) biz.SystemRepo {
	return &systemRepo{
		data: data,
	}
}

func (s *systemRepo) GetMenusByRoleId(ctx context.Context, roleId string) ([]*ent.Menu, error) {
	return s.data.db.Role.Query().
		Where(role.IDEQ(roleId)).
		QueryMenus().
		All(ctx)
}
