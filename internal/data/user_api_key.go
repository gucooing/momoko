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
	entuser "momoko/internal/data/ent/gen/user"
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

	item, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.withOwner(ctx, item.ID)
}

// withOwnerUser 只取所属用户展示用字段。
func withOwnerUser(q *gen.UserQuery) {
	q.Select(entuser.FieldID, entuser.FieldName, entuser.FieldUsername, entuser.FieldAvatar)
}

// withOwner 重新读取一条 API Key 并带上所属用户。
func (r *UserAPIKeyRepo) withOwner(ctx context.Context, id string) (*gen.UserAPIKey, error) {
	return r.data.db.UserAPIKey.Query().
		Where(userapikey.IDEQ(id)).
		WithUser(withOwnerUser).
		Only(ctx)
}

func (r *UserAPIKeyRepo) GetUserAPIKey(ctx context.Context, userID, id string) (*gen.UserAPIKey, error) {
	return r.data.db.UserAPIKey.Query().
		Where(
			userapikey.IDEQ(id),
			userapikey.UserIDEQ(userID),
		).
		WithUser(withOwnerUser).
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

// ListUserAPIKeys userID 为空表示全部用户（关键字同时匹配所属用户的昵称 / 用户名）。
func (r *UserAPIKeyRepo) ListUserAPIKeys(ctx context.Context, userID string, page, pageSize int64, keywords *string, status *v1.APIKeyStatus) ([]*gen.UserAPIKey, int64, *biz.APIKeyStatusCounts, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	now := time.Now()
	query := r.data.db.UserAPIKey.Query()
	if userID != "" {
		query = query.Where(userapikey.UserIDEQ(userID))
	}
	if keywords != nil && strings.TrimSpace(*keywords) != "" {
		kw := strings.TrimSpace(*keywords)
		match := []predicate.UserAPIKey{
			userapikey.NameContains(kw),
			userapikey.RemarkContains(kw),
			userapikey.KeyIDHasPrefix(kw),
		}
		if userID == "" {
			match = append(match, userapikey.HasUserWith(entuser.Or(entuser.NameContains(kw), entuser.UsernameContains(kw))))
		}
		query = query.Where(userapikey.Or(match...))
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
		WithUser(withOwnerUser).
		Order(userapikey.ByCreateTime()).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	return apiKeys, int64(total), counts, nil
}

// APIKeyOwners 全部 API Key 的所属用户及各自的密钥数（按用户 id 升序）。
func (r *UserAPIKeyRepo) APIKeyOwners(ctx context.Context) ([]*biz.APIKeyOwnerCount, error) {
	var rows []struct {
		UserID string `json:"user_id"`
		Count  int64  `json:"count"`
	}
	if err := r.data.db.UserAPIKey.Query().
		GroupBy(userapikey.FieldUserID).
		Aggregate(gen.Count()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []*biz.APIKeyOwnerCount{}, nil
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.UserID)
	}
	users, err := r.data.db.User.Query().
		Where(entuser.IDIn(ids...)).
		Select(entuser.FieldID, entuser.FieldName, entuser.FieldUsername, entuser.FieldAvatar).
		Order(gen.Asc(entuser.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.UserID] = row.Count
	}
	out := make([]*biz.APIKeyOwnerCount, 0, len(users))
	for _, u := range users {
		out = append(out, &biz.APIKeyOwnerCount{User: u, Count: counts[u.ID]})
	}
	return out, nil
}

// UpdateUserAPIKey userID 为空表示不限所属用户（管理全部 API Key）。
func (r *UserAPIKeyRepo) UpdateUserAPIKey(ctx context.Context, userID string, id string, name, remark *string, expiresAt *time.Time, clearExpiresAt *bool, permissions *[]string) (*gen.UserAPIKey, error) {
	update := r.data.db.UserAPIKey.UpdateOneID(id)
	if userID != "" {
		update.Where(userapikey.UserIDEQ(userID))
	}
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
	if err := update.Exec(ctx); err != nil {
		return nil, err
	}
	return r.withOwner(ctx, id)
}

func (r *UserAPIKeyRepo) RefreshUserAPIKey(ctx context.Context, userID, id, apiKey string) (*gen.UserAPIKey, error) {
	if err := r.data.db.UserAPIKey.UpdateOneID(id).
		Where(userapikey.UserIDEQ(userID)).
		SetAPIKey(apiKey).
		SetKeyID(nodeauth.KeyID(apiKey)).
		Exec(ctx); err != nil {
		return nil, err
	}
	return r.withOwner(ctx, id)
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

// DeleteUserAPIKey userID 为空表示不限所属用户（管理全部 API Key）。
func (r *UserAPIKeyRepo) DeleteUserAPIKey(ctx context.Context, userID, id string) error {
	del := r.data.db.UserAPIKey.DeleteOneID(id)
	if userID != "" {
		del.Where(userapikey.UserIDEQ(userID))
	}
	return del.Exec(ctx)
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
