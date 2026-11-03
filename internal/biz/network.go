package biz

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	// ListPortForwardStatsByForwards 返回多条转发在 [start, end] 内的采样，按转发 id、时间升序。
	ListPortForwardStatsByForwards(ctx context.Context, ids []string, start, end time.Time) ([]*gen.PortForwardStat, error)
	// AddPortForwardTraffic 在累计流量上加上增量。
	AddPortForwardTraffic(ctx context.Context, id string, bytesIn, bytesOut int64) error
	// ResetPortForwardTraffic 将累计流量清零。
	ResetPortForwardTraffic(ctx context.Context, userID *string, id string) (*gen.PortForward, error)
}

// trafficFlush 记录某转发本次运行中已累加进数据库的流量，用于计算增量。
type trafficFlush struct {
	start    time.Time
	bytesIn  int64
	bytesOut int64
}

// trafficSummary 是某用户全部转发的今日/昨日流量与今日并发峰值（按采样计算，短时缓存）。
type trafficSummary struct {
	at          time.Time
	ids         string
	todayIn     int64
	todayOut    int64
	yesterday   int64
	peak        int64
	peakAt      time.Time
	lastSamples map[string][2]int64 // 每条转发最后一个采样的累计流量
}

type NetworkUsecase struct {
	repo    NetworkRepo
	manager *port_forward.Manager

	flushMu sync.Mutex
	flushed map[string]trafficFlush

	summaryMu sync.Mutex
	summaries map[string]*trafficSummary
}

func NewNetworkUsecase(repo NetworkRepo) (*NetworkUsecase, func(), error) {
	uc := &NetworkUsecase{
		repo:      repo,
		manager:   port_forward.NewManager(),
		flushed:   make(map[string]trafficFlush),
		summaries: make(map[string]*trafficSummary),
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
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
		for id, snap := range uc.manager.SnapshotAll() {
			uc.flushTraffic(flushCtx, id, snap)
		}
		flushCancel()
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
			for id, snap := range snapshots {
				n.flushTraffic(writeCtx, id, snap)
			}
			cancel()
		}
	}
}

// flushTraffic 把转发本次运行中尚未落库的流量增量累加进数据库。
func (n *NetworkUsecase) flushTraffic(ctx context.Context, id string, snap port_forward.Snapshot) {
	n.flushMu.Lock()
	defer n.flushMu.Unlock()
	f := n.flushed[id]
	if !f.start.Equal(snap.StartTime) {
		f = trafficFlush{start: snap.StartTime}
	}
	dIn, dOut := snap.BytesIn-f.bytesIn, snap.BytesOut-f.bytesOut
	if dIn <= 0 && dOut <= 0 {
		n.flushed[id] = f
		return
	}
	if err := n.repo.AddPortForwardTraffic(ctx, id, max(dIn, 0), max(dOut, 0)); err != nil {
		return
	}
	f.bytesIn, f.bytesOut = snap.BytesIn, snap.BytesOut
	n.flushed[id] = f
}

// flushRunning 在停止/重启转发前把其流量落库。
func (n *NetworkUsecase) flushRunning(ctx context.Context, id string) {
	if snap, ok := n.manager.Stats(id); ok {
		n.flushTraffic(ctx, id, snap)
	}
}

// unflushed 返回转发本次运行中尚未落库的流量。
func (n *NetworkUsecase) unflushed(id string, snap port_forward.Snapshot) (int64, int64) {
	n.flushMu.Lock()
	defer n.flushMu.Unlock()
	f, ok := n.flushed[id]
	if !ok || !f.start.Equal(snap.StartTime) {
		return snap.BytesIn, snap.BytesOut
	}
	return max(snap.BytesIn-f.bytesIn, 0), max(snap.BytesOut-f.bytesOut, 0)
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
	forwards, _, err := n.repo.ListPortForwards(ctx, nil, new(userID), nil)
	if err != nil {
		return nil, err
	}

	all := make([]*v1.PortForwardInfo, 0, len(forwards))
	overview := &v1.PortForwardOverview{Total: int64(len(forwards))}
	ids := make([]string, 0, len(forwards))
	for _, forward := range forwards {
		info := n.toV1PortForwardInfo(forward)
		all = append(all, info)
		ids = append(ids, forward.ID)
		switch info.Status {
		case v1.PortForwardStatus_PORT_FORWARD_STATUS_RUNNING:
			overview.Running++
		case v1.PortForwardStatus_PORT_FORWARD_STATUS_STOPPED:
			overview.Stopped++
		default:
			overview.Error++
		}
		if info.Type == v1.PortForwardType_PORT_FORWARD_TYPE_UDP {
			overview.Udp++
		} else {
			overview.Tcp++
		}
		overview.ActiveConnections += info.ActiveConnections
		overview.RateIn += info.RateIn
		overview.RateOut += info.RateOut
	}
	n.fillTrafficOverview(ctx, userID, ids, overview)

	keywords := strings.ToLower(strings.TrimSpace(req.GetKeywords()))
	filtered := make([]*v1.PortForwardInfo, 0, len(all))
	for _, info := range all {
		if req.Type != nil && info.Type != *req.Type {
			continue
		}
		if req.IsEnable != nil && info.IsEnable != *req.IsEnable {
			continue
		}
		if req.Status != nil && *req.Status != v1.PortForwardStatus_PORT_FORWARD_STATUS_UNSPECIFIED && info.Status != *req.Status {
			continue
		}
		if keywords != "" && !matchPortForward(info, keywords) {
			continue
		}
		filtered = append(filtered, info)
	}

	rsp := &v1.ListPortForwardsResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    int64(len(filtered)),
		Overview: overview,
		Infos:    networkPage(filtered, req.Page, req.PageSize),
	}
	return rsp, nil
}

func matchPortForward(info *v1.PortForwardInfo, keywords string) bool {
	fields := []string{
		info.Name, info.Remark,
		info.ListenAddress + ":" + strconv.Itoa(int(info.ListenPort)),
		info.TargetAddress + ":" + strconv.Itoa(int(info.TargetPort)),
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), keywords) {
			return true
		}
	}
	return false
}

// networkPage 按页截取；pageSize<=0 时返回全部。
func networkPage[T any](items []T, page, pageSize int64) []T {
	if pageSize <= 0 {
		return items
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= int64(len(items)) {
		return []T{}
	}
	end := min(start+pageSize, int64(len(items)))
	return items[start:end]
}

// fillTrafficOverview 填充今日/昨日流量与今日并发峰值。
// 采样结果缓存一个采样周期，今日流量再叠加最后一个采样之后的实时增量。
func (n *NetworkUsecase) fillTrafficOverview(ctx context.Context, userID string, ids []string, overview *v1.PortForwardOverview) {
	summary := n.trafficSummary(ctx, userID, ids)
	if summary == nil {
		return
	}
	overview.TodayBytesIn = summary.todayIn
	overview.TodayBytesOut = summary.todayOut
	overview.YesterdayBytes = summary.yesterday
	for _, id := range ids {
		snap, ok := n.manager.Stats(id)
		if !ok {
			continue
		}
		last, has := summary.lastSamples[id]
		dIn, dOut := snap.BytesIn, snap.BytesOut
		if has && snap.BytesIn >= last[0] && snap.BytesOut >= last[1] {
			dIn, dOut = snap.BytesIn-last[0], snap.BytesOut-last[1]
		}
		overview.TodayBytesIn += dIn
		overview.TodayBytesOut += dOut
	}
	overview.TodayPeakConnections = summary.peak
	if !summary.peakAt.IsZero() {
		overview.TodayPeakTime = timestamppb.New(summary.peakAt)
	}
	if overview.ActiveConnections > overview.TodayPeakConnections {
		overview.TodayPeakConnections = overview.ActiveConnections
		overview.TodayPeakTime = timestamppb.Now()
	}
}

func (n *NetworkUsecase) trafficSummary(ctx context.Context, userID string, ids []string) *trafficSummary {
	key := strings.Join(ids, ",")
	now := time.Now()
	n.summaryMu.Lock()
	defer n.summaryMu.Unlock()
	if cached, ok := n.summaries[userID]; ok && cached.ids == key && now.Sub(cached.at) < statSampleInterval {
		return cached
	}

	summary := &trafficSummary{at: now, ids: key, lastSamples: make(map[string][2]int64)}
	if len(ids) == 0 {
		n.summaries[userID] = summary
		return summary
	}
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	samples, err := n.repo.ListPortForwardStatsByForwards(ctx, ids, yesterdayStart.Add(-2*statSampleInterval), now)
	if err != nil {
		return nil
	}

	peaks := make(map[int64]int64)
	var prev *gen.PortForwardStat
	for _, sample := range samples {
		if prev != nil && prev.PortForwardID == sample.PortForwardID {
			dIn, dOut := sample.BytesIn-prev.BytesIn, sample.BytesOut-prev.BytesOut
			if dIn < 0 || dOut < 0 {
				// 计数归零（转发重启）：本段流量即新计数值。
				dIn, dOut = sample.BytesIn, sample.BytesOut
			}
			switch {
			case !sample.SampleTime.Before(todayStart):
				summary.todayIn += dIn
				summary.todayOut += dOut
			case !sample.SampleTime.Before(yesterdayStart):
				summary.yesterday += dIn + dOut
			}
		}
		if !sample.SampleTime.Before(todayStart) {
			peaks[sample.SampleTime.Unix()] += sample.ActiveConnections
		}
		summary.lastSamples[sample.PortForwardID] = [2]int64{sample.BytesIn, sample.BytesOut}
		prev = sample
	}
	for ts, v := range peaks {
		if v > summary.peak {
			summary.peak = v
			summary.peakAt = time.Unix(ts, 0)
		}
	}
	n.summaries[userID] = summary
	return summary
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
	n.flushRunning(ctx, info.ID)
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
	if _, err := n.repo.GetPortForward(ctx, new(userID), id); err != nil {
		return err
	}
	n.manager.UnRegisterExample(id)
	if err := n.repo.DeletePortForward(ctx, new(userID), id); err != nil {
		return err
	}
	n.flushMu.Lock()
	delete(n.flushed, id)
	n.flushMu.Unlock()
	_ = n.repo.DeletePortForwardStatsByForward(ctx, id)
	return nil
}

// ClosePortForwardConnections 断开转发的活动连接。
func (n *NetworkUsecase) ClosePortForwardConnections(ctx context.Context, userID string, req *v1.ClosePortForwardConnectionsRequest) (int64, error) {
	if _, err := n.repo.GetPortForward(ctx, new(userID), req.Id); err != nil {
		return 0, err
	}
	if req.All {
		return int64(n.manager.CloseConnections(req.Id, nil)), nil
	}
	if len(req.ConnIds) == 0 {
		return 0, nil
	}
	return int64(n.manager.CloseConnections(req.Id, req.ConnIds)), nil
}

// ResetPortForwardTraffic 清零转发的累计流量。
func (n *NetworkUsecase) ResetPortForwardTraffic(ctx context.Context, userID, id string) (*v1.PortForwardInfo, error) {
	n.flushMu.Lock()
	forward, err := n.repo.ResetPortForwardTraffic(ctx, new(userID), id)
	if err == nil {
		if snap, ok := n.manager.Stats(id); ok {
			n.flushed[id] = trafficFlush{start: snap.StartTime, bytesIn: snap.BytesIn, bytesOut: snap.BytesOut}
		} else {
			delete(n.flushed, id)
		}
	}
	n.flushMu.Unlock()
	if err != nil {
		return nil, err
	}
	return n.toV1PortForwardInfo(forward), nil
}

// portForwardStatus 返回转发的运行状态与异常原因。
func (n *NetworkUsecase) portForwardStatus(forward *gen.PortForward) (v1.PortForwardStatus, string) {
	if !forward.IsEnable {
		return v1.PortForwardStatus_PORT_FORWARD_STATUS_STOPPED, ""
	}
	if snap, ok := n.manager.Stats(forward.ID); ok {
		if snap.DialError != "" {
			return v1.PortForwardStatus_PORT_FORWARD_STATUS_ERROR, "目标不可达：" + snap.DialError
		}
		return v1.PortForwardStatus_PORT_FORWARD_STATUS_RUNNING, ""
	}
	if reason := n.manager.Error(forward.ID); reason != "" {
		return v1.PortForwardStatus_PORT_FORWARD_STATUS_ERROR, reason
	}
	return v1.PortForwardStatus_PORT_FORWARD_STATUS_ERROR, "未运行"
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
		IsEnable:      forward.IsEnable,
		Remark:        forward.Remark,
		CreateTime:    timestamppb.New(forward.CreateTime),
		UpdateTime:    timestamppb.New(forward.UpdateTime),
		TotalBytesIn:  forward.TotalBytesIn,
		TotalBytesOut: forward.TotalBytesOut,
	}
	info.Status, info.Error = n.portForwardStatus(forward)

	if snap, ok := n.manager.Stats(forward.ID); ok {
		info.ActiveConnections = snap.ActiveConns
		info.TotalConnections = snap.TotalConns
		info.BytesIn = snap.BytesIn
		info.BytesOut = snap.BytesOut
		info.StartTime = timestamppb.New(snap.StartTime)
		dIn, dOut := n.unflushed(forward.ID, snap)
		info.TotalBytesIn += dIn
		info.TotalBytesOut += dOut
	}
	if rate, ok := n.manager.Rate(forward.ID); ok {
		info.RateIn = rate.In
		info.RateOut = rate.Out
		info.RateHistory = rate.History
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

	info := n.toV1PortForwardInfo(forward)
	resp.Snapshot.TotalBytesIn = info.TotalBytesIn
	resp.Snapshot.TotalBytesOut = info.TotalBytesOut
	if snap, running := n.manager.Stats(forward.ID); running {
		resp.Snapshot.Running = true
		resp.Snapshot.ActiveConnections = snap.ActiveConns
		resp.Snapshot.TotalConnections = snap.TotalConns
		resp.Snapshot.BytesIn = snap.BytesIn
		resp.Snapshot.BytesOut = snap.BytesOut
		resp.Snapshot.StartTime = timestamppb.New(snap.StartTime)
		resp.Snapshot.RateIn = info.RateIn
		resp.Snapshot.RateOut = info.RateOut
	}
	conns := n.manager.Connections(forward.ID)
	resp.Connections = make([]*v1.PortForwardConnection, 0, len(conns))
	for _, c := range conns {
		resp.Connections = append(resp.Connections, &v1.PortForwardConnection{
			Id:         c.ID,
			RemoteAddr: c.RemoteAddr,
			Lan:        isLANAddr(c.RemoteAddr),
			StartTime:  timestamppb.New(c.StartTime),
			BytesIn:    c.BytesIn,
			BytesOut:   c.BytesOut,
		})
	}
	// 最近建立的连接排在前面。
	sort.SliceStable(resp.Connections, func(i, j int) bool {
		return resp.Connections[i].StartTime.AsTime().After(resp.Connections[j].StartTime.AsTime())
	})

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

// isLANAddr 判断 ip:port 是否为本机或局域网地址。
func isLANAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
