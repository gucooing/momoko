package biz

import (
	"context"
	"errors"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	sub2apipkg "momoko/pkg/sub2api"
	"momoko/pkg/task"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Sub2APIRepo 聚合使用记录存储（供 pkg 处理）、抽奖存储与公告/时间线 CRUD。
type Sub2APIRepo interface {
	sub2apipkg.UsageStore
	sub2apipkg.LotteryStore

	ListAnnouncements(ctx context.Context) ([]*gen.Sub2APIAnnouncement, error)
	CreateAnnouncement(ctx context.Context, req *v1.CreateSub2APIAnnouncementRequest) (*gen.Sub2APIAnnouncement, error)
	UpdateAnnouncement(ctx context.Context, req *v1.UpdateSub2APIAnnouncementRequest) (*gen.Sub2APIAnnouncement, error)
	DeleteAnnouncement(ctx context.Context, id string) error

	ListTimeline(ctx context.Context) ([]*gen.Sub2APITimelineItem, error)
	CreateTimelineItem(ctx context.Context, req *v1.CreateSub2APITimelineItemRequest) (*gen.Sub2APITimelineItem, error)
	UpdateTimelineItem(ctx context.Context, req *v1.UpdateSub2APITimelineItemRequest) (*gen.Sub2APITimelineItem, error)
	DeleteTimelineItem(ctx context.Context, id string) error
}

type Sub2APIUsecase struct {
	repo    Sub2APIRepo
	service *sub2apipkg.Service
	lottery *sub2apipkg.LotteryService
	tasks   *task.Manager
}

func NewSub2APIUsecase(config ConfigRepo, repo Sub2APIRepo, tasks *task.Manager) (*Sub2APIUsecase, func(), error) {
	// 全模块共用一个 Manager（内部 SharedHTTPClient），禁止各处 NewManager 另起连接池。
	manager := sub2apipkg.NewManager()
	service := sub2apipkg.NewService(repo, config, manager)
	service.Start()
	lottery := sub2apipkg.NewLotteryService(repo, config, manager)
	// 抽奖定时交给全局任务管理器（实现 task.Task 即可被 EnsureSingleton 托管）。
	if tasks != nil {
		_ = tasks.EnsureSingleton(context.Background(), sub2apipkg.NewLotteryTickTask(lottery))
	}
	uc := &Sub2APIUsecase{repo: repo, service: service, lottery: lottery, tasks: tasks}
	cleanup := func() {
		service.Stop()
	}
	return uc, cleanup, nil
}

// ---------- 配置与用量 ----------

func (s *Sub2APIUsecase) Config(ctx context.Context) (*v1.Sub2APIConfig, error) {
	cfg, err := s.service.Config(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return cfg, nil
}

func (s *Sub2APIUsecase) ListGroups(ctx context.Context) ([]*sub2apipkg.Group, error) {
	groups, err := s.service.ListGroups(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return groups, nil
}

func (s *Sub2APIUsecase) UpdateConfig(ctx context.Context, req *v1.UpdateSub2APIConfigRequest) (*v1.Sub2APIConfig, error) {
	cfg, err := s.service.UpdateConfig(ctx, req.GetConfig())
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	return cfg, nil
}

func (s *Sub2APIUsecase) TestConnection(ctx context.Context, req *v1.TestSub2APIConnectionRequest) (*v1.TestSub2APIConnectionResponse, error) {
	connected, message := s.service.TestConnection(ctx, req.GetConfig())
	return &v1.TestSub2APIConnectionResponse{Connected: connected, Message: message}, nil
}

func (s *Sub2APIUsecase) Snapshot(ctx context.Context) (*v1.Sub2APIUsageSnapshot, error) {
	snapshot, err := s.service.Snapshot(ctx)
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	return snapshot, nil
}

func (s *Sub2APIUsecase) SyncUsage(ctx context.Context, full bool) (*v1.Sub2APIUsageSnapshot, error) {
	snapshot, err := s.service.Sync(ctx, full)
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	return snapshot, nil
}

// PublicHome 公开首页（无需鉴权），聚合配置展示、用量快照与公告/时间线。
func (s *Sub2APIUsecase) PublicHome(ctx context.Context) (*v1.Sub2APIHome, error) {
	cfg, err := s.service.Config(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	home := &v1.Sub2APIHome{
		Enabled:      cfg.HomeEnabled,
		ConsoleUrl:   cfg.ConsoleUrl,
		Title:        cfg.Title,
		Subtitle:     cfg.Subtitle,
		Introduction: cfg.Introduction,
	}
	if !cfg.HomeEnabled {
		return home, nil
	}
	snapshot, err := s.service.PublicSnapshot(ctx)
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	home.Snapshot = snapshot

	announcements, err := s.repo.ListAnnouncements(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	home.Announcements = toV1Announcements(announcements)

	timeline, err := s.repo.ListTimeline(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	home.Timeline = toV1Timeline(timeline)
	return home, nil
}

// PublicStats 指定区间的公开用量统计（无需鉴权）。
func (s *Sub2APIUsecase) PublicStats(ctx context.Context, rangeDays int32) (*v1.Sub2APIStats, error) {
	stats, err := s.service.PublicStats(ctx, rangeDays)
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	return stats, nil
}

// AdminStats 管理端用量概览（按时间段统计）。最近请求改由 RecentRequests 分页返回。
func (s *Sub2APIUsecase) AdminStats(ctx context.Context, start, end time.Time) (*v1.Sub2APIStats, error) {
	stats, err := s.service.AdminStats(ctx, start, end)
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	return stats, nil
}

// RecentRequests 管理端最近请求分页（按时间段，倒序）。
func (s *Sub2APIUsecase) RecentRequests(ctx context.Context, start, end time.Time, page, pageSize int) ([]*v1.Sub2APIRecentRequest, int, error) {
	list, total, err := s.service.RecentRequests(ctx, start, end, page, pageSize)
	if err != nil {
		return nil, 0, mapSub2APIError(err)
	}
	return list, total, nil
}

// ---------- 公告 CRUD ----------

func (s *Sub2APIUsecase) ListAnnouncements(ctx context.Context) ([]*v1.Sub2APIAnnouncement, error) {
	list, err := s.repo.ListAnnouncements(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Announcements(list), nil
}

func (s *Sub2APIUsecase) CreateAnnouncement(ctx context.Context, req *v1.CreateSub2APIAnnouncementRequest) (*v1.Sub2APIAnnouncement, error) {
	created, err := s.repo.CreateAnnouncement(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Announcement(created), nil
}

func (s *Sub2APIUsecase) UpdateAnnouncement(ctx context.Context, req *v1.UpdateSub2APIAnnouncementRequest) (*v1.Sub2APIAnnouncement, error) {
	if req.GetId() == "" {
		return nil, ErrSub2APIRecordID
	}
	updated, err := s.repo.UpdateAnnouncement(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Announcement(updated), nil
}

func (s *Sub2APIUsecase) DeleteAnnouncement(ctx context.Context, id string) error {
	if id == "" {
		return ErrSub2APIRecordID
	}
	if err := s.repo.DeleteAnnouncement(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// ---------- 时间线 CRUD ----------

func (s *Sub2APIUsecase) ListTimeline(ctx context.Context) ([]*v1.Sub2APITimelineItem, error) {
	list, err := s.repo.ListTimeline(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Timeline(list), nil
}

func (s *Sub2APIUsecase) CreateTimelineItem(ctx context.Context, req *v1.CreateSub2APITimelineItemRequest) (*v1.Sub2APITimelineItem, error) {
	created, err := s.repo.CreateTimelineItem(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1TimelineItem(created), nil
}

func (s *Sub2APIUsecase) UpdateTimelineItem(ctx context.Context, req *v1.UpdateSub2APITimelineItemRequest) (*v1.Sub2APITimelineItem, error) {
	if req.GetId() == "" {
		return nil, ErrSub2APIRecordID
	}
	updated, err := s.repo.UpdateTimelineItem(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1TimelineItem(updated), nil
}

func (s *Sub2APIUsecase) DeleteTimelineItem(ctx context.Context, id string) error {
	if id == "" {
		return ErrSub2APIRecordID
	}
	if err := s.repo.DeleteTimelineItem(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// ---------- 映射与错误 ----------

func mapSub2APIError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sub2apipkg.ErrConfigRequired):
		return ErrSub2APIConfigRequired
	case errors.Is(err, sub2apipkg.ErrHomeConfigIncomplete):
		return ErrSub2APIHomeConfig
	case errors.Is(err, sub2apipkg.ErrSyncRunning):
		return ErrSub2APISyncRunning
	default:
		return ErrSystem(err)
	}
}

func toV1Announcements(list []*gen.Sub2APIAnnouncement) []*v1.Sub2APIAnnouncement {
	result := make([]*v1.Sub2APIAnnouncement, 0, len(list))
	for _, item := range list {
		result = append(result, toV1Announcement(item))
	}
	return result
}

func toV1Announcement(item *gen.Sub2APIAnnouncement) *v1.Sub2APIAnnouncement {
	if item == nil {
		return nil
	}
	out := &v1.Sub2APIAnnouncement{
		Id:      item.ID,
		Title:   item.Title,
		Content: item.Content,
		Level:   item.Level,
		Pinned:  item.Pinned,
	}
	if item.PublishedAt != nil {
		out.PublishedAt = timestamppb.New(*item.PublishedAt)
	}
	return out
}

func toV1Timeline(list []*gen.Sub2APITimelineItem) []*v1.Sub2APITimelineItem {
	result := make([]*v1.Sub2APITimelineItem, 0, len(list))
	for _, item := range list {
		result = append(result, toV1TimelineItem(item))
	}
	return result
}

func toV1TimelineItem(item *gen.Sub2APITimelineItem) *v1.Sub2APITimelineItem {
	if item == nil {
		return nil
	}
	out := &v1.Sub2APITimelineItem{
		Id:       item.ID,
		Title:    item.Title,
		Content:  item.Content,
		Category: item.Category,
	}
	if item.PublishedAt != nil {
		out.PublishedAt = timestamppb.New(*item.PublishedAt)
	}
	return out
}
