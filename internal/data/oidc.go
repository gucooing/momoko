package data

import (
	"context"

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
func (r *oidcRepo) ListClients(ctx context.Context, page, pageSize int64, keywords *string) ([]*gen.OIDCClient, int64, error) {
	query := r.data.db.OIDCClient.Query()
	if keywords != nil && *keywords != "" {
		query = query.Where(oidcclient.NameContains(*keywords))
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
		Save(ctx)
}

func (r *oidcRepo) UpdateClient(ctx context.Context, client *gen.OIDCClient) (*gen.OIDCClient, error) {
	return r.data.db.OIDCClient.UpdateOneID(client.ID).
		SetName(client.Name).
		SetRedirectUris(client.RedirectUris).
		SetScopes(client.Scopes).
		SetActive(client.Active).
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
