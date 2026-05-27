package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/userapikey"
)

type UserAPIKeyRepo struct {
	data *Data
}

func NewUserAPIKeyRepo(data *Data) *UserAPIKeyRepo {
	return &UserAPIKeyRepo{
		data: data,
	}
}

func NewAPIKeyRepo(data *Data) biz.APIKeyRepo {
	return NewUserAPIKeyRepo(data)
}

func (r *UserAPIKeyRepo) CreateUserAPIKey(ctx context.Context, userID, name, apiKey string, expiresAt *time.Time) (*gen.UserAPIKey, error) {
	create := r.data.db.UserAPIKey.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetName(name).
		SetAPIKey(apiKey)
	if expiresAt != nil {
		create.SetExpiresAt(*expiresAt)
	}

	return create.Save(ctx)
}

func (r *UserAPIKeyRepo) GetUserAPIKey(ctx context.Context, userID, id string) (*gen.UserAPIKey, error) {
	return r.data.db.UserAPIKey.Query().
		Where(
			userapikey.IDEQ(id),
			userapikey.UserIDEQ(userID),
		).
		Only(ctx)
}

func (r *UserAPIKeyRepo) GetUserAPIKeyByAPIKey(ctx context.Context, apiKey string) (*gen.UserAPIKey, error) {
	return r.data.db.UserAPIKey.Query().
		Where(userapikey.APIKeyEQ(apiKey)).
		Only(ctx)
}

func (r *UserAPIKeyRepo) ListUserAPIKeys(ctx context.Context, userID string, page, pageSize int64, keywords *string) ([]*gen.UserAPIKey, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	query := r.data.db.UserAPIKey.Query().
		Where(userapikey.UserIDEQ(userID))
	if keywords != nil {
		query = query.Where(userapikey.NameContains(*keywords))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	apiKeys, err := query.
		Order(userapikey.ByCreateTime()).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return apiKeys, int64(total), nil
}

func (r *UserAPIKeyRepo) UpdateUserAPIKey(ctx context.Context, userID string, id string, name *string, expiresAt *time.Time, clearExpiresAt *bool) (*gen.UserAPIKey, error) {
	update := r.data.db.UserAPIKey.UpdateOneID(id).
		Where(userapikey.UserIDEQ(userID))
	if name != nil {
		update.SetName(*name)
	}
	if clearExpiresAt != nil {
		update.ClearExpiresAt()
	} else if expiresAt != nil {
		update.SetExpiresAt(*expiresAt)
	}
	return update.Save(ctx)
}

func (r *UserAPIKeyRepo) RefreshUserAPIKey(ctx context.Context, userID, id, apiKey string) (*gen.UserAPIKey, error) {
	return r.data.db.UserAPIKey.UpdateOneID(id).
		Where(userapikey.UserIDEQ(userID)).
		SetAPIKey(apiKey).
		Save(ctx)
}

func (r *UserAPIKeyRepo) DeleteUserAPIKey(ctx context.Context, userID, id string) error {
	return r.data.db.UserAPIKey.DeleteOneID(id).
		Where(userapikey.UserIDEQ(userID)).
		Exec(ctx)
}
