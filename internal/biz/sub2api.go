package biz

import (
	"context"
	"errors"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/constant"
	sub2apipkg "momoko/pkg/sub2api"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Sub2APIRepo 聚合使用记录存储（供 pkg 处理）与公告/时间线 CRUD。
type Sub2APIRepo interface {
	sub2apipkg.UsageStore

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
	sys     *SystemUsecase
	repo    Sub2APIRepo
	service *sub2apipkg.Service
}

func NewSub2APIUsecase(sys *SystemUsecase, config ConfigRepo, repo Sub2APIRepo) (*Sub2APIUsecase, func(), error) {
	service := sub2apipkg.NewService(repo, config)
	service.Start()
	uc := &Sub2APIUsecase{sys: sys, repo: repo, service: service}
	return uc, service.Stop, nil
}

// ---------- 配置与用量 ----------

func (s *Sub2APIUsecase) Config(ctx context.Context) (*v1.Sub2APIConfig, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIView); err != nil {
		return nil, err
	}
	cfg, err := s.service.Config(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return cfg, nil
}

func (s *Sub2APIUsecase) UpdateConfig(ctx context.Context, req *v1.UpdateSub2APIConfigRequest) (*v1.Sub2APIConfig, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return nil, err
	}
	cfg, err := s.service.UpdateConfig(ctx, req.GetConfig())
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	return cfg, nil
}

func (s *Sub2APIUsecase) TestConnection(ctx context.Context, req *v1.TestSub2APIConnectionRequest) (*v1.TestSub2APIConnectionResponse, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return nil, err
	}
	connected, message := s.service.TestConnection(ctx, req.GetConfig())
	return &v1.TestSub2APIConnectionResponse{Connected: connected, Message: message}, nil
}

func (s *Sub2APIUsecase) Snapshot(ctx context.Context) (*v1.Sub2APIUsageSnapshot, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIView); err != nil {
		return nil, err
	}
	snapshot, err := s.service.Snapshot(ctx)
	if err != nil {
		return nil, mapSub2APIError(err)
	}
	return snapshot, nil
}

func (s *Sub2APIUsecase) SyncUsage(ctx context.Context, full bool) (*v1.Sub2APIUsageSnapshot, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return nil, err
	}
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

// ---------- 公告 CRUD ----------

func (s *Sub2APIUsecase) ListAnnouncements(ctx context.Context) ([]*v1.Sub2APIAnnouncement, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIView); err != nil {
		return nil, err
	}
	list, err := s.repo.ListAnnouncements(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Announcements(list), nil
}

func (s *Sub2APIUsecase) CreateAnnouncement(ctx context.Context, req *v1.CreateSub2APIAnnouncementRequest) (*v1.Sub2APIAnnouncement, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return nil, err
	}
	created, err := s.repo.CreateAnnouncement(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Announcement(created), nil
}

func (s *Sub2APIUsecase) UpdateAnnouncement(ctx context.Context, req *v1.UpdateSub2APIAnnouncementRequest) (*v1.Sub2APIAnnouncement, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return nil, err
	}
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
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return err
	}
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
	if err := s.sys.Check(ctx, constant.Sub2APIView); err != nil {
		return nil, err
	}
	list, err := s.repo.ListTimeline(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Timeline(list), nil
}

func (s *Sub2APIUsecase) CreateTimelineItem(ctx context.Context, req *v1.CreateSub2APITimelineItemRequest) (*v1.Sub2APITimelineItem, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return nil, err
	}
	created, err := s.repo.CreateTimelineItem(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1TimelineItem(created), nil
}

func (s *Sub2APIUsecase) UpdateTimelineItem(ctx context.Context, req *v1.UpdateSub2APITimelineItemRequest) (*v1.Sub2APITimelineItem, error) {
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return nil, err
	}
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
	if err := s.sys.Check(ctx, constant.Sub2APIEdit); err != nil {
		return err
	}
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
