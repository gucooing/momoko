package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/predicate"
	"momoko/internal/data/ent/gen/userapikey"
	"momoko/pkg/constant"
)

type mcpRepo struct {
	data *Data
}

func NewMcpRepo(data *Data) biz.McpRepo {
	return &mcpRepo{data: data}
}

// ListMcpCredentials 授予范围（permissions JSON 数组）含 perms 中任一项的 API Key，在 SQL 里筛。
func (r *mcpRepo) ListMcpCredentials(ctx context.Context, userID string, perms []constant.Permissions) ([]*gen.UserAPIKey, error) {
	if len(perms) == 0 {
		return []*gen.UserAPIKey{}, nil
	}
	granted := predicate.UserAPIKey(func(s *sql.Selector) {
		preds := make([]*sql.Predicate, 0, len(perms))
		for _, perm := range perms {
			preds = append(preds, sqljson.ValueContains(s.C(userapikey.FieldPermissions), string(perm)))
		}
		s.Where(sql.Or(preds...))
	})
	return r.data.db.UserAPIKey.Query().
		Where(userapikey.UserIDEQ(userID), granted).
		Order(userapikey.ByCreateTime()).
		All(ctx)
}
