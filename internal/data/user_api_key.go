package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/userapikey"
	"momoko/pkg/nodeauth"
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

func (r *UserAPIKeyRepo) CreateUserAPIKey(ctx context.Context, userID, name, apiKey string, expiresAt *time.Time, permissions []string) (*gen.UserAPIKey, error) {
	create := r.data.db.UserAPIKey.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetName(name).
		SetAPIKey(apiKey).
		SetKeyID(nodeauth.KeyID(apiKey)).
		SetPermissions(permissions)
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

// GetAPIKeyByKeyID 按种子派生的公开标识查找 API Key（节点签名鉴权用）。
func (r *UserAPIKeyRepo) GetAPIKeyByKeyID(ctx context.Context, keyID string) (*gen.UserAPIKey, error) {
	return r.data.db.UserAPIKey.Query().
		Where(userapikey.KeyIDEQ(keyID)).
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

func (r *UserAPIKeyRepo) UpdateUserAPIKey(ctx context.Context, userID string, id string, name *string, expiresAt *time.Time, clearExpiresAt *bool, permissions *[]string) (*gen.UserAPIKey, error) {
	update := r.data.db.UserAPIKey.UpdateOneID(id).
		Where(userapikey.UserIDEQ(userID))
	if name != nil {
		update.SetName(*name)
	}
	if clearExpiresAt != nil && *clearExpiresAt {
		update.ClearExpiresAt()
	} else if expiresAt != nil {
		update.SetExpiresAt(*expiresAt)
	}
	if permissions != nil {
		update.SetPermissions(*permissions)
	}
	return update.Save(ctx)
}

func (r *UserAPIKeyRepo) RefreshUserAPIKey(ctx context.Context, userID, id, apiKey string) (*gen.UserAPIKey, error) {
	return r.data.db.UserAPIKey.UpdateOneID(id).
		Where(userapikey.UserIDEQ(userID)).
		SetAPIKey(apiKey).
		SetKeyID(nodeauth.KeyID(apiKey)).
		Save(ctx)
}

func (r *UserAPIKeyRepo) DeleteUserAPIKey(ctx context.Context, userID, id string) error {
	return r.data.db.UserAPIKey.DeleteOneID(id).
		Where(userapikey.UserIDEQ(userID)).
		Exec(ctx)
}

// backfillAPIKeyIDs 为引入签名鉴权前创建的 API Key 补齐派生标识。
func backfillAPIKeyIDs(ctx context.Context, client *gen.Client) error {
	keys, err := client.UserAPIKey.Query().
		Where(userapikey.Or(userapikey.KeyIDIsNil(), userapikey.KeyIDEQ(""))).
		All(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := client.UserAPIKey.UpdateOneID(key.ID).
			SetKeyID(nodeauth.KeyID(key.APIKey)).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
