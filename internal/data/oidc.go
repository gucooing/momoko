package data

import (
	"context"
	"strings"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/oidcclient"
)

type oidcRepo struct {
	data *Data
}

func NewOIDCRepo(data *Data) biz.OIDCRepo {
	return &oidcRepo{data: data}
}

// ListClients 查询后台 OIDC 客户端列表。
// 数据层只处理分页和关键字条件，不判断协议字段合法性。
func (r *oidcRepo) ListClients(ctx context.Context, page, pageSize int64, keywords *string, active *bool) ([]*gen.OIDCClient, int64, error) {
	query := r.clientQuery(keywords)
	if active != nil {
		query = query.Where(oidcclient.ActiveEQ(*active))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	clients, err := query.
		Order(gen.Desc(oidcclient.FieldCreateTime)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return clients, int64(total), nil
}

// clientQuery 按关键字（名称、描述、Client ID）构造查询。
func (r *oidcRepo) clientQuery(keywords *string) *gen.OIDCClientQuery {
	query := r.data.db.OIDCClient.Query()
	if keywords != nil && strings.TrimSpace(*keywords) != "" {
		kw := strings.TrimSpace(*keywords)
		query = query.Where(oidcclient.Or(
			oidcclient.NameContainsFold(kw),
			oidcclient.DescriptionContainsFold(kw),
			oidcclient.ClientIDContainsFold(kw),
		))
	}
	return query
}

// CountClients 统计同一关键字下的全部数量与已启用数量。
func (r *oidcRepo) CountClients(ctx context.Context, keywords *string) (int64, int64, error) {
	all, err := r.clientQuery(keywords).Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	active, err := r.clientQuery(keywords).Where(oidcclient.ActiveEQ(true)).Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	return int64(all), int64(active), nil
}

// MarkClientUsed 记录客户端最近一次成功换取令牌的时间。
func (r *oidcRepo) MarkClientUsed(ctx context.Context, id string, at time.Time) error {
	return r.data.db.OIDCClient.UpdateOneID(id).SetLastUsedTime(at).Exec(ctx)
}

func (r *oidcRepo) GetClientByClientID(ctx context.Context, clientID string) (*gen.OIDCClient, error) {
	return r.data.db.OIDCClient.Query().
		Where(oidcclient.ClientIDEQ(clientID)).
		Only(ctx)
}

func (r *oidcRepo) CreateClient(ctx context.Context, client *gen.OIDCClient) (*gen.OIDCClient, error) {
	return r.data.db.OIDCClient.Create().
		SetID(client.ID).
		SetName(client.Name).
		SetClientID(client.ClientID).
		SetClientSecret(client.ClientSecret).
		SetRedirectUris(client.RedirectUris).
		SetScopes(client.Scopes).
		SetActive(client.Active).
		SetDescription(client.Description).
		Save(ctx)
}

func (r *oidcRepo) UpdateClient(ctx context.Context, client *gen.OIDCClient) (*gen.OIDCClient, error) {
	return r.data.db.OIDCClient.UpdateOneID(client.ID).
		SetName(client.Name).
		SetRedirectUris(client.RedirectUris).
		SetScopes(client.Scopes).
		SetActive(client.Active).
		SetDescription(client.Description).
		Save(ctx)
}

func (r *oidcRepo) UpdateClientSecret(ctx context.Context, id, secret string) (*gen.OIDCClient, error) {
	return r.data.db.OIDCClient.UpdateOneID(id).
		SetClientSecret(secret).
		Save(ctx)
}

func (r *oidcRepo) DeleteClient(ctx context.Context, id string) error {
	return r.data.db.OIDCClient.DeleteOneID(id).Exec(ctx)
}
