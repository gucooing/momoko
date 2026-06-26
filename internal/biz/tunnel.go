package biz

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/frptunnel"
	"momoko/pkg/frp_tunnel"
	"momoko/pkg/response"
)

// defaultTunnelSampleInterval 是 FrpsConfig 未配置采样间隔时的回退值。
// 统计采样仅追加入库、长期保留，不做按期删除（保留历史，见持久化诉求）。
const defaultTunnelSampleInterval = 30 * time.Second

// FrpTunnelStatSample 表示一条待落库的隧道统计采样。
type FrpTunnelStatSample struct {
	TunnelID    string
	Time        time.Time
	ActiveConns int64
	BytesIn     int64
	BytesOut    int64
}

type TunnelRepo interface {
	CreateTunnel(ctx context.Context, userID string, req *v1.CreateTunnelRequest, credential string) (*gen.FrpTunnel, error)
	ListTunnels(ctx context.Context, isEnable *bool, userID *string, proxyType *v1.TunnelType, keywords *string) ([]*gen.FrpTunnel, int64, error)
	UpdateTunnel(ctx context.Context, userID *string, req *v1.UpdateTunnelRequest) (*gen.FrpTunnel, error)
	GetTunnel(ctx context.Context, userID *string, id string) (*gen.FrpTunnel, error)
	GetTunnelByName(ctx context.Context, name string) (*gen.FrpTunnel, error)
	ListEnabledTunnels(ctx context.Context) ([]*gen.FrpTunnel, error)
	DeleteTunnel(ctx context.Context, userID *string, id string) error
	SaveFrpTunnelStats(ctx context.Context, samples []FrpTunnelStatSample) error
	ListFrpTunnelStats(ctx context.Context, tunnelID string, start, end time.Time) ([]*gen.FrpTunnelStat, error)
	DeleteFrpTunnelStatsByTunnel(ctx context.Context, tunnelID string) error
}

type TunnelUsecase struct {
	repo    TunnelRepo
	config  ConfigRepo
	manager *frp_tunnel.Manager

	sampleInterval time.Duration
}

func NewTunnelUsecase(repo TunnelRepo, config ConfigRepo) (*TunnelUsecase, func(), error) {
	uc := &TunnelUsecase{
		repo:           repo,
		config:         config,
		sampleInterval: defaultTunnelSampleInterval,
	}
	// frps 相关机制（生命周期、专用回环鉴权服务、统计读取）全部内聚在 pkg/frp_tunnel.Manager。
	// 业务层只提供网络设置与一个按名查规则的 TunnelLookup 实现（uc 自身）。
	uc.manager = frp_tunnel.NewManager(frp_tunnel.ManagerConfig{Lookup: uc})

	cfg, err := config.FrpsConfig(context.Background())
	if err != nil {
		// 读取配置失败不阻断启动：以 frps 未启用状态降级运行，运行时可在配置页修复。
		log.Errorf("读取 frps 配置失败，内网穿透以未启用状态启动: %v", err)
		cfg = &v1.FrpsConfig{}
	}
	uc.applyStatConfig(cfg)
	if err := uc.manager.Apply(toFrpsOptions(cfg)); err != nil {
		log.Errorf("启动 frps 失败，内网穿透降级启动: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go uc.sampleLoop(ctx)

	cleanup := func() {
		cancel()
		uc.manager.Close()
	}
	return uc, cleanup, nil
}

// LookupTunnel 实现 frp_tunnel.TunnelLookup：按 frp proxy 名称返回鉴权所需的隧道视图。
// pkg/frp_tunnel 的鉴权逻辑通过此接口取数据，业务层只负责查库与字段映射。
func (t *TunnelUsecase) LookupTunnel(ctx context.Context, name string) (*frp_tunnel.Tunnel, bool) {
	tn, err := t.repo.GetTunnelByName(ctx, name)
	if err != nil {
		return nil, false
	}
	return &frp_tunnel.Tunnel{
		Name:           tn.Name,
		Credential:     tn.Credential,
		ProxyType:      string(tn.ProxyType),
		RemotePort:     tn.RemotePort,
		AllowUsers:     splitCSV(tn.AllowUsers),
		Enabled:        tn.IsEnable,
		MaxBandwidth:   tn.MaxBandwidth,
		MaxActiveConns: tn.MaxActiveConns,
	}, true
}

// applyStatConfig 从 FrpsConfig 更新采样间隔。
func (t *TunnelUsecase) applyStatConfig(cfg *v1.FrpsConfig) {
	if cfg.StatSampleInterval > 0 {
		t.sampleInterval = time.Duration(cfg.StatSampleInterval) * time.Second
	} else {
		t.sampleInterval = defaultTunnelSampleInterval
	}
}

// sampleLoop 周期性地为所有启用且在线的隧道采样统计快照并落库。
func (t *TunnelUsecase) sampleLoop(ctx context.Context) {
	ticker := time.NewTicker(t.sampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			tunnels, err := t.repo.ListEnabledTunnels(ctx)
			if err != nil || len(tunnels) == 0 {
				continue
			}
			samples := make([]FrpTunnelStatSample, 0, len(tunnels))
			for _, tn := range tunnels {
				stat, ok := t.manager.Stat(tn.Name)
				if !ok {
					continue
				}
				samples = append(samples, FrpTunnelStatSample{
					TunnelID:    tn.ID,
					Time:        now,
					ActiveConns: stat.ActiveConns,
					BytesIn:     stat.BytesIn,
					BytesOut:    stat.BytesOut,
				})
			}
			if len(samples) == 0 {
				continue
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_ = t.repo.SaveFrpTunnelStats(writeCtx, samples)
			cancel()
		}
	}
}

func (t *TunnelUsecase) ListTunnels(ctx context.Context, userID string, req *v1.ListTunnelsRequest) (*v1.ListTunnelsResponse, error) {
	infos, count, err := t.repo.ListTunnels(ctx, req.IsEnable, &userID, req.Type, req.Keywords)
	if err != nil {
		return nil, ErrSystem(err)
	}
	rsp := &v1.ListTunnelsResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    count,
		Infos:    make([]*v1.TunnelInfo, 0, len(infos)),
	}
	for _, info := range infos {
		rsp.Infos = append(rsp.Infos, t.toV1TunnelInfo(info))
	}
	return rsp, nil
}

func (t *TunnelUsecase) CreateTunnel(ctx context.Context, userID string, req *v1.CreateTunnelRequest) (*v1.TunnelInfo, error) {
	info, err := t.repo.CreateTunnel(ctx, userID, req, genCredential())
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, response.BadRequest(400, "隧道名称已存在")
		}
		return nil, ErrSystem(err)
	}
	return t.toV1TunnelInfo(info), nil
}

func (t *TunnelUsecase) GetTunnel(ctx context.Context, userID, id string) (*v1.TunnelInfo, error) {
	info, err := t.repo.GetTunnel(ctx, &userID, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, response.BadRequest(404, "隧道不存在")
		}
		return nil, ErrSystem(err)
	}
	return t.toV1TunnelInfo(info), nil
}

func (t *TunnelUsecase) UpdateTunnel(ctx context.Context, userID string, req *v1.UpdateTunnelRequest) (*v1.TunnelInfo, error) {
	info, err := t.repo.UpdateTunnel(ctx, &userID, req)
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, response.BadRequest(400, "隧道名称已存在")
		}
		return nil, ErrSystem(err)
	}
	return t.toV1TunnelInfo(info), nil
}

func (t *TunnelUsecase) DeleteTunnel(ctx context.Context, userID, id string) error {
	if err := t.repo.DeleteTunnel(ctx, &userID, id); err != nil {
		return ErrSystem(err)
	}
	_ = t.repo.DeleteFrpTunnelStatsByTunnel(ctx, id)
	return nil
}

// GetTunnelStats 返回隧道的实时快照与指定时间范围内的时间序列。
func (t *TunnelUsecase) GetTunnelStats(ctx context.Context, userID, id string, startMs, endMs int64) (*v1.GetTunnelStatsResponse, error) {
	tunnel, err := t.repo.GetTunnel(ctx, &userID, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, response.BadRequest(404, "隧道不存在")
		}
		return nil, ErrSystem(err)
	}

	resp := &v1.GetTunnelStatsResponse{
		SampleIntervalSeconds: int64(t.sampleInterval / time.Second),
		Snapshot:              &v1.TunnelStat{},
	}
	if stat, ok := t.manager.Stat(tunnel.Name); ok {
		resp.Snapshot = &v1.TunnelStat{
			Online:            stat.Online,
			ActiveConnections: stat.ActiveConns,
			BytesIn:           stat.BytesIn,
			BytesOut:          stat.BytesOut,
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
	samples, err := t.repo.ListFrpTunnelStats(ctx, tunnel.ID, start, end)
	if err != nil {
		return nil, ErrSystem(err)
	}
	resp.Points = make([]*v1.TunnelStatPoint, 0, len(samples))
	for _, sample := range samples {
		resp.Points = append(resp.Points, &v1.TunnelStatPoint{
			Time:              timestamppb.New(sample.SampleTime),
			ActiveConnections: sample.ActiveConnections,
			BytesIn:           sample.BytesIn,
			BytesOut:          sample.BytesOut,
		})
	}
	return resp, nil
}

func (t *TunnelUsecase) GetFrpsConfig(ctx context.Context) (*v1.FrpsConfig, error) {
	cfg, err := t.config.FrpsConfig(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return cfg, nil
}

func (t *TunnelUsecase) UpdateFrpsConfig(ctx context.Context, req *v1.FrpsConfig) (*v1.FrpsConfig, error) {
	cfg, err := t.config.UpdateFrpsConfig(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	t.applyStatConfig(cfg)
	// 配置有效性由 pkg 兜底：frps 装配失败（端口非法/占用等）时 Apply 返回错误。
	if err := t.manager.Apply(toFrpsOptions(cfg)); err != nil {
		return nil, response.BadRequest(500, "应用 frps 配置失败："+err.Error())
	}
	return cfg, nil
}

func (t *TunnelUsecase) toV1TunnelInfo(tunnel *gen.FrpTunnel) *v1.TunnelInfo {
	info := &v1.TunnelInfo{
		Id:             tunnel.ID,
		Name:           tunnel.Name,
		UserId:         tunnel.UserID,
		Type:           entProxyTypeToV1(tunnel.ProxyType),
		RemotePort:     int32(tunnel.RemotePort),
		CustomDomains:  tunnel.CustomDomains,
		Subdomain:      tunnel.Subdomain,
		LocalIp:        tunnel.LocalIP,
		LocalPort:      int32(tunnel.LocalPort),
		Credential:     tunnel.Credential,
		AllowUsers:     tunnel.AllowUsers,
		IsEnable:       tunnel.IsEnable,
		MaxBandwidth:   tunnel.MaxBandwidth,
		MaxActiveConns: int32(tunnel.MaxActiveConns),
		CreateTime:     timestamppb.New(tunnel.CreateTime),
		UpdateTime:     timestamppb.New(tunnel.UpdateTime),
	}

	stat, ok := t.manager.Stat(tunnel.Name)
	switch {
	case !tunnel.IsEnable:
		info.Status = v1.TunnelStatus_TUNNEL_STATUS_OFFLINE
	case ok && stat.Online:
		info.Status = v1.TunnelStatus_TUNNEL_STATUS_ONLINE
		info.ActiveConnections = stat.ActiveConns
		info.BytesIn = stat.BytesIn
		info.BytesOut = stat.BytesOut
	default:
		info.Status = v1.TunnelStatus_TUNNEL_STATUS_PENDING
	}
	return info
}

// ---- 辅助函数 ----

func toFrpsOptions(cfg *v1.FrpsConfig) frp_tunnel.Options {
	return frp_tunnel.Options{
		Enable:         cfg.IsEnable,
		BindAddr:       cfg.BindAddr,
		BindPort:       int(cfg.BindPort),
		VhostHTTPPort:  int(cfg.VhostHttpPort),
		VhostHTTPSPort: int(cfg.VhostHttpsPort),
		KCPBindPort:    int(cfg.KcpBindPort),
		QUICBindPort:   int(cfg.QuicBindPort),
		SubdomainHost:  cfg.SubdomainHost,
	}
}

func entProxyTypeToV1(p frptunnel.ProxyType) v1.TunnelType {
	switch p {
	case frptunnel.ProxyTypeTCP:
		return v1.TunnelType_TUNNEL_TYPE_TCP
	case frptunnel.ProxyTypeUDP:
		return v1.TunnelType_TUNNEL_TYPE_UDP
	case frptunnel.ProxyTypeHTTP:
		return v1.TunnelType_TUNNEL_TYPE_HTTP
	case frptunnel.ProxyTypeHTTPS:
		return v1.TunnelType_TUNNEL_TYPE_HTTPS
	case frptunnel.ProxyTypeStcp:
		return v1.TunnelType_TUNNEL_TYPE_STCP
	case frptunnel.ProxyTypeXtcp:
		return v1.TunnelType_TUNNEL_TYPE_XTCP
	case frptunnel.ProxyTypeTcpmux:
		return v1.TunnelType_TUNNEL_TYPE_TCPMUX
	default:
		return v1.TunnelType_TUNNEL_TYPE_UNSPECIFIED
	}
}

func splitCSV(s string) []string {
	out := make([]string, 0)
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func genCredential() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
