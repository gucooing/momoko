package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/portforward"
	"momoko/internal/data/ent/gen/portforwardstat"
)

type networkRepo struct {
	data *Data
}

func NewNetworkRepo(data *Data) biz.NetworkRepo {
	return &networkRepo{data: data}
}

// 新建端口转发 传入 ctx userid CreatePortForwardRequest | 传出 实例 错误
func (r *networkRepo) CreatePortForward(ctx context.Context, userID string, req *v1.CreatePortForwardRequest) (*gen.PortForward, error) {
	return r.data.db.PortForward.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetName(req.Name).
		SetProtocol(toEntProtocol(req.Type)).
		SetListenAddress(req.ListenAddress).
		SetListenPort(int(req.ListenPort)).
		SetTargetAddress(req.TargetAddress).
		SetTargetPort(int(req.TargetPort)).
		SetIsEnable(req.IsEnable).
		Save(ctx)
}

// 获取端口转发列表 传入 ctx *是否启用 *用户id *协议 传出 列表 总数 错误
func (r *networkRepo) ListPortForwards(ctx context.Context, isEnable *bool, userID *string, protocol *v1.PortForwardType) ([]*gen.PortForward, int64, error) {
	query := r.data.db.PortForward.Query()

	if isEnable != nil {
		query = query.Where(portforward.IsEnableEQ(*isEnable))
	}
	if userID != nil && *userID != "" {
		query = query.Where(portforward.UserIDEQ(*userID))
	}
	if protocol != nil {
		query = query.Where(portforward.ProtocolEQ(toEntProtocol(*protocol)))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, int64(total), nil
}

// 更新端口转发 ，传入 ctx UpdatePortForwardRequest 可选更新 | 传出 实例 错误
func (r *networkRepo) UpdatePortForward(ctx context.Context, userID *string, req *v1.UpdatePortForwardRequest) (*gen.PortForward, error) {
	builder := r.data.db.PortForward.UpdateOneID(req.Id).
		SetNillableName(req.Name).
		SetNillableListenAddress(req.ListenAddress).
		SetNillableTargetAddress(req.TargetAddress).
		SetNillableIsEnable(req.IsEnable)

	if userID != nil {
		builder = builder.Where(portforward.UserIDEQ(*userID))
	}

	if req.Type != nil {
		builder.SetProtocol(toEntProtocol(*req.Type))
	}
	if req.ListenPort != nil {
		builder.SetListenPort(int(*req.ListenPort))
	}
	if req.TargetPort != nil {
		builder.SetTargetPort(int(*req.TargetPort))
	}

	return builder.Save(ctx)
}

// 获取端口转发 传入 *用户id id
func (r *networkRepo) GetPortForward(ctx context.Context, userID *string, id string) (*gen.PortForward, error) {
	query := r.data.db.PortForward.Query().Where(portforward.IDEQ(id))

	if userID != nil {
		query = query.Where(portforward.UserIDEQ(*userID))
	}

	return query.Only(ctx)
}

// 删除端口转发 传入 *用户id id
func (r *networkRepo) DeletePortForward(ctx context.Context, userID *string, id string) error {
	query := r.data.db.PortForward.DeleteOneID(id)

	if userID != nil {
		query = query.Where(portforward.UserIDEQ(*userID))
	}

	err := query.Exec(ctx)
	return err
}

// SavePortForwardStats 批量写入端口转发统计采样。
func (r *networkRepo) SavePortForwardStats(ctx context.Context, samples []biz.PortForwardStatSample) error {
	if len(samples) == 0 {
		return nil
	}
	builders := make([]*gen.PortForwardStatCreate, 0, len(samples))
	for _, sample := range samples {
		builders = append(builders, r.data.db.PortForwardStat.Create().
			SetPortForwardID(sample.PortForwardID).
			SetSampleTime(sample.Time).
			SetActiveConnections(sample.ActiveConns).
			SetBytesIn(sample.BytesIn).
			SetBytesOut(sample.BytesOut))
	}
	return r.data.db.PortForwardStat.CreateBulk(builders...).Exec(ctx)
}

// ListPortForwardStats 按时间升序返回某转发在 [start, end] 区间内的统计采样。
func (r *networkRepo) ListPortForwardStats(ctx context.Context, portForwardID string, start, end time.Time) ([]*gen.PortForwardStat, error) {
	return r.data.db.PortForwardStat.Query().
		Where(
			portforwardstat.PortForwardIDEQ(portForwardID),
			portforwardstat.SampleTimeGTE(start),
			portforwardstat.SampleTimeLTE(end),
		).
		Order(gen.Asc(portforwardstat.FieldSampleTime)).
		All(ctx)
}

// DeletePortForwardStatsBefore 删除采样时间早于 cutoff 的统计采样，返回删除条数。
func (r *networkRepo) DeletePortForwardStatsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	return r.data.db.PortForwardStat.Delete().
		Where(portforwardstat.SampleTimeLT(cutoff)).
		Exec(ctx)
}

// DeletePortForwardStatsByForward 删除某转发的全部统计采样。
func (r *networkRepo) DeletePortForwardStatsByForward(ctx context.Context, portForwardID string) error {
	_, err := r.data.db.PortForwardStat.Delete().
		Where(portforwardstat.PortForwardIDEQ(portForwardID)).
		Exec(ctx)
	return err
}

func toEntProtocol(a v1.PortForwardType) portforward.Protocol {
	switch a {
	case v1.PortForwardType_PORT_FORWARD_TYPE_TCP:
		return portforward.ProtocolTCP
	case v1.PortForwardType_PORT_FORWARD_TYPE_UDP:
		return portforward.ProtocolUDP
	default:
		return portforward.ProtocolTCP
	}
}
