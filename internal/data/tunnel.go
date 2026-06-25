package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/frptunnel"
	"momoko/internal/data/ent/gen/frptunnelstat"
)

type tunnelRepo struct {
	data *Data
}

func NewTunnelRepo(data *Data) biz.TunnelRepo {
	return &tunnelRepo{data: data}
}

// CreateTunnel 新建隧道规则。credential 由业务层生成传入。
func (r *tunnelRepo) CreateTunnel(ctx context.Context, userID string, req *v1.CreateTunnelRequest, credential string) (*gen.FrpTunnel, error) {
	return r.data.db.FrpTunnel.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetName(req.Name).
		SetProxyType(toEntProxyType(req.Type)).
		SetRemotePort(int(req.RemotePort)).
		SetCustomDomains(req.CustomDomains).
		SetSubdomain(req.Subdomain).
		SetLocalIP(req.LocalIp).
		SetLocalPort(int(req.LocalPort)).
		SetCredential(credential).
		SetAllowUsers(req.AllowUsers).
		SetIsEnable(req.IsEnable).
		Save(ctx)
}

// ListTunnels 获取隧道列表 传入 ctx *是否启用 *用户id *类型 *关键词 | 传出 列表 总数 错误
func (r *tunnelRepo) ListTunnels(ctx context.Context, isEnable *bool, userID *string, proxyType *v1.TunnelType, keywords *string) ([]*gen.FrpTunnel, int64, error) {
	query := r.data.db.FrpTunnel.Query()

	if isEnable != nil {
		query = query.Where(frptunnel.IsEnableEQ(*isEnable))
	}
	if userID != nil && *userID != "" {
		query = query.Where(frptunnel.UserIDEQ(*userID))
	}
	if proxyType != nil {
		query = query.Where(frptunnel.ProxyTypeEQ(toEntProxyType(*proxyType)))
	}
	if keywords != nil && *keywords != "" {
		query = query.Where(frptunnel.Or(
			frptunnel.NameContainsFold(*keywords),
			frptunnel.CustomDomainsContainsFold(*keywords),
		))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := query.Order(gen.Desc(frptunnel.FieldCreateTime)).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, int64(total), nil
}

// UpdateTunnel 更新隧道 传入 ctx *用户id UpdateTunnelRequest 可选更新 | 传出 实例 错误
func (r *tunnelRepo) UpdateTunnel(ctx context.Context, userID *string, req *v1.UpdateTunnelRequest) (*gen.FrpTunnel, error) {
	builder := r.data.db.FrpTunnel.UpdateOneID(req.Id).
		SetNillableName(req.Name).
		SetNillableCustomDomains(req.CustomDomains).
		SetNillableSubdomain(req.Subdomain).
		SetNillableLocalIP(req.LocalIp).
		SetNillableAllowUsers(req.AllowUsers).
		SetNillableIsEnable(req.IsEnable)

	if userID != nil {
		builder = builder.Where(frptunnel.UserIDEQ(*userID))
	}

	if req.Type != nil {
		builder.SetProxyType(toEntProxyType(*req.Type))
	}
	if req.RemotePort != nil {
		builder.SetRemotePort(int(*req.RemotePort))
	}
	if req.LocalPort != nil {
		builder.SetLocalPort(int(*req.LocalPort))
	}

	return builder.Save(ctx)
}

// GetTunnel 获取隧道 传入 ctx *用户id id
func (r *tunnelRepo) GetTunnel(ctx context.Context, userID *string, id string) (*gen.FrpTunnel, error) {
	query := r.data.db.FrpTunnel.Query().Where(frptunnel.IDEQ(id))
	if userID != nil {
		query = query.Where(frptunnel.UserIDEQ(*userID))
	}
	return query.Only(ctx)
}

// GetTunnelByName 按名称获取隧道（不限用户）。供 frps 插件鉴权使用。
func (r *tunnelRepo) GetTunnelByName(ctx context.Context, name string) (*gen.FrpTunnel, error) {
	return r.data.db.FrpTunnel.Query().
		Where(frptunnel.NameEQ(name)).
		Only(ctx)
}

// ListEnabledTunnels 返回所有已启用的隧道。供统计采样循环使用。
func (r *tunnelRepo) ListEnabledTunnels(ctx context.Context) ([]*gen.FrpTunnel, error) {
	return r.data.db.FrpTunnel.Query().
		Where(frptunnel.IsEnableEQ(true)).
		All(ctx)
}

// DeleteTunnel 删除隧道 传入 ctx *用户id id
func (r *tunnelRepo) DeleteTunnel(ctx context.Context, userID *string, id string) error {
	query := r.data.db.FrpTunnel.DeleteOneID(id)
	if userID != nil {
		query = query.Where(frptunnel.UserIDEQ(*userID))
	}
	return query.Exec(ctx)
}

// SaveFrpTunnelStats 批量写入隧道统计采样。
func (r *tunnelRepo) SaveFrpTunnelStats(ctx context.Context, samples []biz.FrpTunnelStatSample) error {
	if len(samples) == 0 {
		return nil
	}
	builders := make([]*gen.FrpTunnelStatCreate, 0, len(samples))
	for _, sample := range samples {
		builders = append(builders, r.data.db.FrpTunnelStat.Create().
			SetFrpTunnelID(sample.TunnelID).
			SetSampleTime(sample.Time).
			SetActiveConnections(sample.ActiveConns).
			SetBytesIn(sample.BytesIn).
			SetBytesOut(sample.BytesOut))
	}
	return r.data.db.FrpTunnelStat.CreateBulk(builders...).Exec(ctx)
}

// ListFrpTunnelStats 按时间升序返回某隧道在 [start, end] 区间内的统计采样。
func (r *tunnelRepo) ListFrpTunnelStats(ctx context.Context, tunnelID string, start, end time.Time) ([]*gen.FrpTunnelStat, error) {
	return r.data.db.FrpTunnelStat.Query().
		Where(
			frptunnelstat.FrpTunnelIDEQ(tunnelID),
			frptunnelstat.SampleTimeGTE(start),
			frptunnelstat.SampleTimeLTE(end),
		).
		Order(gen.Asc(frptunnelstat.FieldSampleTime)).
		All(ctx)
}

// DeleteFrpTunnelStatsByTunnel 删除某隧道的全部统计采样。
func (r *tunnelRepo) DeleteFrpTunnelStatsByTunnel(ctx context.Context, tunnelID string) error {
	_, err := r.data.db.FrpTunnelStat.Delete().
		Where(frptunnelstat.FrpTunnelIDEQ(tunnelID)).
		Exec(ctx)
	return err
}

func toEntProxyType(t v1.TunnelType) frptunnel.ProxyType {
	switch t {
	case v1.TunnelType_TUNNEL_TYPE_TCP:
		return frptunnel.ProxyTypeTCP
	case v1.TunnelType_TUNNEL_TYPE_UDP:
		return frptunnel.ProxyTypeUDP
	case v1.TunnelType_TUNNEL_TYPE_HTTP:
		return frptunnel.ProxyTypeHTTP
	case v1.TunnelType_TUNNEL_TYPE_HTTPS:
		return frptunnel.ProxyTypeHTTPS
	case v1.TunnelType_TUNNEL_TYPE_STCP:
		return frptunnel.ProxyTypeStcp
	case v1.TunnelType_TUNNEL_TYPE_XTCP:
		return frptunnel.ProxyTypeXtcp
	case v1.TunnelType_TUNNEL_TYPE_TCPMUX:
		return frptunnel.ProxyTypeTcpmux
	default:
		return frptunnel.ProxyTypeTCP
	}
}
