package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/predicate"
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

func (r *UserAPIKeyRepo) CreateUserAPIKey(ctx context.Context, userID, name, remark, apiKey string, expiresAt *time.Time, permissions []string) (*gen.UserAPIKey, error) {
	create := r.data.db.UserAPIKey.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetName(name).
		SetRemark(remark).
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

// apiKeyStatusPredicate 按有效状态筛选（以 now 为基准，与 biz.APIKeyStatusOf 一致）。
func apiKeyStatusPredicate(status v1.APIKeyStatus, now time.Time) predicate.UserAPIKey {
	soon := now.Add(biz.APIKeyExpiringWindow)
	switch status {
	case v1.APIKeyStatus_APIKeyStatus_Expired:
		return userapikey.ExpiresAtLTE(now)
	case v1.APIKeyStatus_APIKeyStatus_Expiring:
		return userapikey.And(userapikey.ExpiresAtGT(now), userapikey.ExpiresAtLTE(soon))
	default:
		return userapikey.Or(userapikey.ExpiresAtIsNil(), userapikey.ExpiresAtGT(soon))
	}
}

func (r *UserAPIKeyRepo) ListUserAPIKeys(ctx context.Context, userID string, page, pageSize int64, keywords *string, status *v1.APIKeyStatus) ([]*gen.UserAPIKey, int64, *biz.APIKeyStatusCounts, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	now := time.Now()
	query := r.data.db.UserAPIKey.Query().
		Where(userapikey.UserIDEQ(userID))
	if keywords != nil && strings.TrimSpace(*keywords) != "" {
		kw := strings.TrimSpace(*keywords)
		query = query.Where(userapikey.Or(userapikey.NameContains(kw), userapikey.RemarkContains(kw), userapikey.KeyIDHasPrefix(kw)))
	}

	counts := &biz.APIKeyStatusCounts{}
	for _, item := range []struct {
		status v1.APIKeyStatus
		dst    *int64
	}{
		{v1.APIKeyStatus_APIKeyStatus_Valid, &counts.Valid},
		{v1.APIKeyStatus_APIKeyStatus_Expiring, &counts.Expiring},
		{v1.APIKeyStatus_APIKeyStatus_Expired, &counts.Expired},
	} {
		n, err := query.Clone().Where(apiKeyStatusPredicate(item.status, now)).Count(ctx)
		if err != nil {
			return nil, 0, nil, err
		}
		*item.dst = int64(n)
	}

	if status != nil {
		query = query.Where(apiKeyStatusPredicate(*status, now))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	apiKeys, err := query.
		Order(userapikey.ByCreateTime()).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	return apiKeys, int64(total), counts, nil
}

func (r *UserAPIKeyRepo) UpdateUserAPIKey(ctx context.Context, userID string, id string, name, remark *string, expiresAt *time.Time, clearExpiresAt *bool, permissions *[]string) (*gen.UserAPIKey, error) {
	update := r.data.db.UserAPIKey.UpdateOneID(id).
		Where(userapikey.UserIDEQ(userID))
	if name != nil {
		update.SetName(*name)
	}
	if remark != nil {
		update.SetRemark(*remark)
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

// TouchAPIKeyUsed 只写最近使用信息，保持 update_time 不变（它表示配置修改时间）。
func (r *UserAPIKeyRepo) TouchAPIKeyUsed(ctx context.Context, id string, at time.Time, ip string) error {
	item, err := r.data.db.UserAPIKey.Get(ctx, id)
	if err != nil {
		return err
	}
	return r.data.db.UserAPIKey.UpdateOne(item).
		SetLastUsedAt(at).
		SetLastUsedIP(ip).
		SetUpdateTime(item.UpdateTime).
		Exec(ctx)
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
