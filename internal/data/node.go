package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entnode "momoko/internal/data/ent/gen/node"
	authpkg "momoko/pkg/auth"
	"momoko/pkg/secretbox"
)

type nodeRepo struct {
	data *Data
}

func NewNodeRepo(data *Data) biz.NodeRepo {
	return &nodeRepo{data: data}
}

func (r *nodeRepo) box() *secretbox.Box {
	return secretbox.New(authpkg.AuthSecretKey)
}

// toNodeConfig 解密节点种子；解密失败（主密钥变更）时种子留空，由调用方提示重新填写。
func (r *nodeRepo) toNodeConfig(item *gen.Node) *biz.NodeConfig {
	seed, err := r.box().Decrypt(item.APIKey)
	if err != nil {
		seed = ""
	}
	return &biz.NodeConfig{
		ID:            item.ID,
		Name:          item.Name,
		Address:       item.Address,
		APIKey:        seed,
		KeyID:         item.KeyID,
		TLS:           item.TLS,
		TLSSkipVerify: item.TLSSkipVerify,
		Remark:        item.Remark,
		CreateTime:    item.CreateTime,
		UpdateTime:    item.UpdateTime,
		LastOnlineAt:  item.LastOnlineAt,
	}
}

func (r *nodeRepo) ListNodes(ctx context.Context, keywords *string) ([]*biz.NodeConfig, error) {
	query := r.data.db.Node.Query()
	if keywords != nil && strings.TrimSpace(*keywords) != "" {
		kw := strings.TrimSpace(*keywords)
		query = query.Where(entnode.Or(entnode.NameContains(kw), entnode.AddressContains(kw)))
	}
	items, err := query.Order(entnode.ByCreateTime()).All(ctx)
	if err != nil {
		return nil, err
	}
	nodes := make([]*biz.NodeConfig, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, r.toNodeConfig(item))
	}
	return nodes, nil
}

func (r *nodeRepo) GetNode(ctx context.Context, id string) (*biz.NodeConfig, error) {
	item, err := r.data.db.Node.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toNodeConfig(item), nil
}

func (r *nodeRepo) CreateNode(ctx context.Context, node *biz.NodeConfig) (*biz.NodeConfig, error) {
	encrypted, err := r.box().Encrypt(node.APIKey)
	if err != nil {
		return nil, err
	}
	item, err := r.data.db.Node.Create().
		SetID(uuid.NewString()).
		SetName(node.Name).
		SetAddress(node.Address).
		SetAPIKey(encrypted).
		SetKeyID(node.KeyID).
		SetTLS(node.TLS).
		SetTLSSkipVerify(node.TLSSkipVerify).
		SetRemark(node.Remark).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toNodeConfig(item), nil
}

func (r *nodeRepo) UpdateNode(ctx context.Context, node *biz.NodeConfig) (*biz.NodeConfig, error) {
	encrypted, err := r.box().Encrypt(node.APIKey)
	if err != nil {
		return nil, err
	}
	item, err := r.data.db.Node.UpdateOneID(node.ID).
		SetName(node.Name).
		SetAddress(node.Address).
		SetAPIKey(encrypted).
		SetKeyID(node.KeyID).
		SetTLS(node.TLS).
		SetTLSSkipVerify(node.TLSSkipVerify).
		SetRemark(node.Remark).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toNodeConfig(item), nil
}

// TouchNodeOnline 只写在线时间，保持 update_time 不变（它表示配置修改时间）。
func (r *nodeRepo) TouchNodeOnline(ctx context.Context, id string, at time.Time) error {
	item, err := r.data.db.Node.Get(ctx, id)
	if err != nil {
		return err
	}
	return r.data.db.Node.UpdateOne(item).
		SetLastOnlineAt(at).
		SetUpdateTime(item.UpdateTime).
		Exec(ctx)
}

func (r *nodeRepo) DeleteNode(ctx context.Context, id string) error {
	return r.data.db.Node.DeleteOneID(id).Exec(ctx)
}
