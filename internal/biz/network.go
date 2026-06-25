package biz

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/portforward"
	"momoko/pkg/port_forward"

	v1 "momoko/api/gen/v1"
)

const (
	// statSampleInterval 端口转发统计采样并落库的间隔。
	statSampleInterval = 30 * time.Second
	// statRetention 统计采样在数据库中的保留时长。
	statRetention = 7 * 24 * time.Hour
	// statCleanupInterval 过期统计采样的清理间隔。
	statCleanupInterval = 6 * time.Hour
)

// PortForwardStatSample 表示一条待落库的端口转发统计采样。
type PortForwardStatSample struct {
	PortForwardID string
	Time          time.Time
	ActiveConns   int64
	BytesIn       int64
	BytesOut      int64
}

type NetworkRepo interface {
	CreatePortForward(ctx context.Context, userID string, req *v1.CreatePortForwardRequest) (*gen.PortForward, error)
	ListPortForwards(ctx context.Context, isEnable *bool, userID *string, protocol *v1.PortForwardType) ([]*gen.PortForward, int64, error)
	UpdatePortForward(ctx context.Context, userID *string, req *v1.UpdatePortForwardRequest) (*gen.PortForward, error)
	GetPortForward(ctx context.Context, userID *string, id string) (*gen.PortForward, error)
	DeletePortForward(ctx context.Context, userID *string, id string) error
	SavePortForwardStats(ctx context.Context, samples []PortForwardStatSample) error
	ListPortForwardStats(ctx context.Context, portForwardID string, start, end time.Time) ([]*gen.PortForwardStat, error)
	DeletePortForwardStatsBefore(ctx context.Context, cutoff time.Time) (int, error)
	DeletePortForwardStatsByForward(ctx context.Context, portForwardID string) error
}

type NetworkUsecase struct {
	repo    NetworkRepo
	manager *port_forward.Manager
}

func NewNetworkUsecase(repo NetworkRepo) (*NetworkUsecase, func(), error) {
	uc := &NetworkUsecase{
		repo:    repo,
		manager: port_forward.NewManager(),
	}

	list, _, err := repo.ListPortForwards(context.Background(), new(true), nil, nil)
	if err != nil {
		uc.manager.Stop()
		return nil, nil, err
	}
	for _, info := range list {
		uc.manager.RegisterExample(toPortForwardOption(info))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go uc.sampleLoop(ctx)
	go uc.cleanupLoop(ctx)

	cleanup := func() {
		cancel()
		uc.manager.Stop()
	}
	return uc, cleanup, nil
}

// sampleLoop 周期性地为所有运行中的端口转发采样统计快照并落库。
func (n *NetworkUsecase) sampleLoop(ctx context.Context) {
	ticker := time.NewTicker(statSampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			snapshots := n.manager.SnapshotAll()
			if len(snapshots) == 0 {
				continue
			}
			samples := make([]PortForwardStatSample, 0, len(snapshots))
			for id, snap := range snapshots {
				samples = append(samples, PortForwardStatSample{
					PortForwardID: id,
					Time:          now,
					ActiveConns:   snap.ActiveConns,
					BytesIn:       snap.BytesIn,
					BytesOut:      snap.BytesOut,
				})
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_ = n.repo.SavePortForwardStats(writeCtx, samples)
			cancel()
		}
	}
}

// cleanupLoop 周期性地清理超过保留时长的统计采样。
func (n *NetworkUsecase) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(statCleanupInterval)
	defer ticker.Stop()

	n.cleanupExpiredStats(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.cleanupExpiredStats(ctx)
		}
	}
}

func (n *NetworkUsecase) cleanupExpiredStats(ctx context.Context) {
	writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, _ = n.repo.DeletePortForwardStatsBefore(writeCtx, time.Now().Add(-statRetention))
}

func (n *NetworkUsecase) ListPortForwards(ctx context.Context, userID string, req *v1.ListPortForwardsRequest) (*v1.ListPortForwardsResponse, error) {
	infos, count, err := n.repo.ListPortForwards(ctx, req.IsEnable, new(userID), req.Type)
	if err != nil {
		return nil, err
	}
	rsp := &v1.ListPortForwardsResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    count,
		Infos:    make([]*v1.PortForwardInfo, 0, len(infos)),
	}
	for _, info := range infos {
		rsp.Infos = append(rsp.Infos, n.toV1PortForwardInfo(info))
	}
	return rsp, nil
}

func (n *NetworkUsecase) CreatePortForward(ctx context.Context, userID string, req *v1.CreatePortForwardRequest) (*v1.PortForwardInfo, error) {
	info, err := n.repo.CreatePortForward(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	if info.IsEnable {
		err = n.manager.Retry(toPortForwardOption(info))
	}
	item := n.toV1PortForwardInfo(info)
	if err != nil {
		item.Error = err.Error()
	}
	return item, nil
}

func (n *NetworkUsecase) GetPortForward(ctx context.Context, userID, id string) (*v1.PortForwardInfo, error) {
	info, err := n.repo.GetPortForward(ctx, new(userID), id)
	if err != nil {
		return nil, err
	}
	return n.toV1PortForwardInfo(info), nil
}

func (n *NetworkUsecase) UpdatePortForward(ctx context.Context, userID string, req *v1.UpdatePortForwardRequest) (*v1.PortForwardInfo, error) {
	info, err := n.repo.UpdatePortForward(ctx, new(userID), req)
	if err != nil {
		return nil, err
	}
	if info.IsEnable {
		err = n.manager.Retry(toPortForwardOption(info))
	} else {
		n.manager.UnRegisterExample(info.ID)
	}
	item := n.toV1PortForwardInfo(info)
	if err != nil {
		item.Error = err.Error()
	}

	return item, nil
}

func (n *NetworkUsecase) DeletePortForward(ctx context.Context, userID, id string) error {
	n.manager.UnRegisterExample(id)
	if err := n.repo.DeletePortForward(ctx, new(userID), id); err != nil {
		return err
	}
	_ = n.repo.DeletePortForwardStatsByForward(ctx, id)
	return nil
}

func (n *NetworkUsecase) toV1PortForwardInfo(forward *gen.PortForward) *v1.PortForwardInfo {
	info := &v1.PortForwardInfo{
		Id:            forward.ID,
		Name:          forward.Name,
		UserId:        forward.UserID,
		Type:          toV1PortForwardProtocol(forward.Protocol),
		ListenAddress: forward.ListenAddress,
		ListenPort:    int32(forward.ListenPort),
		TargetAddress: forward.TargetAddress,
		TargetPort:    int32(forward.TargetPort),
		IsEnable:      forward.IsEnable && n.manager.Running(forward.ID),
		CreateTime:    timestamppb.New(forward.CreateTime),
		UpdateTime:    timestamppb.New(forward.UpdateTime),
	}

	if snap, ok := n.manager.Stats(forward.ID); ok {
		info.ActiveConnections = snap.ActiveConns
		info.TotalConnections = snap.TotalConns
		info.BytesIn = snap.BytesIn
		info.BytesOut = snap.BytesOut
	}

	return info
}

// GetPortForwardStats 返回端口转发的实时快照与指定时间范围内的时间序列。
func (n *NetworkUsecase) GetPortForwardStats(ctx context.Context, userID, id string, startMs, endMs int64) (*v1.GetPortForwardStatsResponse, error) {
	forward, err := n.repo.GetPortForward(ctx, new(userID), id)
	if err != nil {
		return nil, err
	}

	resp := &v1.GetPortForwardStatsResponse{
		SampleIntervalSeconds: int64(statSampleInterval / time.Second),
		Snapshot:              &v1.PortForwardStat{},
	}

	if snap, running := n.manager.Stats(forward.ID); running {
		resp.Snapshot = &v1.PortForwardStat{
			Running:           true,
			ActiveConnections: snap.ActiveConns,
			TotalConnections:  snap.TotalConns,
			BytesIn:           snap.BytesIn,
			BytesOut:          snap.BytesOut,
			StartTime:         timestamppb.New(snap.StartTime),
		}
	}

	end := time.Now()
	if endMs > 0 {
		end = time.UnixMilli(endMs)
	}
	start := end.Add(-time.Hour)
	if startMs > 0 {
		start = time.UnixMilli(startMs)
	}
	if start.After(end) {
		start, end = end, start
	}
	samples, err := n.repo.ListPortForwardStats(ctx, forward.ID, start, end)
	if err != nil {
		return nil, err
	}
	resp.Points = make([]*v1.PortForwardStatPoint, 0, len(samples))
	for _, sample := range samples {
		resp.Points = append(resp.Points, &v1.PortForwardStatPoint{
			Time:              timestamppb.New(sample.SampleTime),
			ActiveConnections: sample.ActiveConnections,
			BytesIn:           sample.BytesIn,
			BytesOut:          sample.BytesOut,
		})
	}

	return resp, nil
}

func toPortForwardOption(forward *gen.PortForward) *port_forward.Option {
	return &port_forward.Option{
		ID:            forward.ID,
		Protocol:      toPortForwardProtocol(forward.Protocol),
		ListenAddress: forward.ListenAddress,
		ListenPort:    forward.ListenPort,
		TargetPort:    forward.TargetPort,
		TargetAddress: forward.TargetAddress,
	}
}

func toV1PortForwardProtocol(forward portforward.Protocol) v1.PortForwardType {
	switch forward {
	case portforward.ProtocolTCP:
		return v1.PortForwardType_PORT_FORWARD_TYPE_TCP
	case portforward.ProtocolUDP:
		return v1.PortForwardType_PORT_FORWARD_TYPE_UDP
	default:
		return v1.PortForwardType_PORT_FORWARD_TYPE_UNSPECIFIED
	}
}

func toPortForwardProtocol(forward portforward.Protocol) port_forward.Protocol {
	switch forward {
	case portforward.ProtocolTCP:
		return port_forward.TCP
	case portforward.ProtocolUDP:
		return port_forward.UDP
	default:
		return port_forward.TCP
	}
}
