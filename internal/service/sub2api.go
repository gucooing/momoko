package service

import (
	"context"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

type Sub2APIService struct {
	v1.UnimplementedSub2APIManagerServer

	uc *biz.Sub2APIUsecase
}

func NewSub2APIService(uc *biz.Sub2APIUsecase) *Sub2APIService {
	return &Sub2APIService{uc: uc}
}

func (s *Sub2APIService) PublicSub2APIHome(ctx context.Context, _ *v1.PublicSub2APIHomeRequest) (*v1.PublicSub2APIHomeResponse, error) {
	home, err := s.uc.PublicHome(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.PublicSub2APIHomeResponse{Home: home}, nil
}

func (s *Sub2APIService) GetSub2APIConfig(ctx context.Context, _ *v1.GetSub2APIConfigRequest) (*v1.GetSub2APIConfigResponse, error) {
	config, err := s.uc.Config(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APIConfigResponse{Config: config}, nil
}

func (s *Sub2APIService) UpdateSub2APIConfig(ctx context.Context, req *v1.UpdateSub2APIConfigRequest) (*v1.UpdateSub2APIConfigResponse, error) {
	config, err := s.uc.UpdateConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateSub2APIConfigResponse{Config: config}, nil
}

func (s *Sub2APIService) TestSub2APIConnection(ctx context.Context, req *v1.TestSub2APIConnectionRequest) (*v1.TestSub2APIConnectionResponse, error) {
	return s.uc.TestConnection(ctx, req)
}

func (s *Sub2APIService) SyncSub2APIUsage(ctx context.Context, req *v1.SyncSub2APIUsageRequest) (*v1.SyncSub2APIUsageResponse, error) {
	snapshot, err := s.uc.SyncUsage(ctx, req.GetFull())
	if err != nil {
		return nil, err
	}
	return &v1.SyncSub2APIUsageResponse{Snapshot: snapshot}, nil
}

func (s *Sub2APIService) GetSub2APISnapshot(ctx context.Context, _ *v1.GetSub2APISnapshotRequest) (*v1.GetSub2APISnapshotResponse, error) {
	snapshot, err := s.uc.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APISnapshotResponse{Snapshot: snapshot}, nil
}

func (s *Sub2APIService) GetPublicSub2APIStats(ctx context.Context, req *v1.GetSub2APIStatsRequest) (*v1.GetSub2APIStatsResponse, error) {
	stats, err := s.uc.PublicStats(ctx, req.GetRangeDays())
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APIStatsResponse{Stats: stats}, nil
}

func (s *Sub2APIService) GetSub2APIStats(ctx context.Context, req *v1.GetSub2APIAdminStatsRequest) (*v1.GetSub2APIAdminStatsResponse, error) {
	start, end := rangeFromMillis(req.GetStartTime(), req.GetEndTime())
	stats, err := s.uc.AdminStats(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APIAdminStatsResponse{Stats: stats}, nil
}

func (s *Sub2APIService) GetSub2APIRecentRequests(ctx context.Context, req *v1.GetSub2APIRecentRequestsRequest) (*v1.GetSub2APIRecentRequestsResponse, error) {
	start, end := rangeFromMillis(req.GetStartTime(), req.GetEndTime())
	list, total, err := s.uc.RecentRequests(ctx, start, end, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APIRecentRequestsResponse{
		RecentRequests: list,
		Total:          int64(total),
		Page:           req.GetPage(),
		PageSize:       req.GetPageSize(),
	}, nil
}

// rangeFromMillis 将 Unix 毫秒（<=0 视为未指定）转为时间段端点，归一化交由 usecase/service 处理。
func rangeFromMillis(startMs, endMs int64) (time.Time, time.Time) {
	var start, end time.Time
	if startMs > 0 {
		start = time.UnixMilli(startMs)
	}
	if endMs > 0 {
		end = time.UnixMilli(endMs)
	}
	return start, end
}

func (s *Sub2APIService) ListSub2APIAnnouncements(ctx context.Context, _ *v1.ListSub2APIAnnouncementsRequest) (*v1.ListSub2APIAnnouncementsResponse, error) {
	list, err := s.uc.ListAnnouncements(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListSub2APIAnnouncementsResponse{Announcements: list}, nil
}

func (s *Sub2APIService) CreateSub2APIAnnouncement(ctx context.Context, req *v1.CreateSub2APIAnnouncementRequest) (*v1.Sub2APIAnnouncementResponse, error) {
	item, err := s.uc.CreateAnnouncement(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.Sub2APIAnnouncementResponse{Announcement: item}, nil
}

func (s *Sub2APIService) UpdateSub2APIAnnouncement(ctx context.Context, req *v1.UpdateSub2APIAnnouncementRequest) (*v1.Sub2APIAnnouncementResponse, error) {
	item, err := s.uc.UpdateAnnouncement(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.Sub2APIAnnouncementResponse{Announcement: item}, nil
}

func (s *Sub2APIService) DeleteSub2APIAnnouncement(ctx context.Context, req *v1.DeleteSub2APIAnnouncementRequest) (*v1.DeleteSub2APIAnnouncementResponse, error) {
	if err := s.uc.DeleteAnnouncement(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteSub2APIAnnouncementResponse{}, nil
}

func (s *Sub2APIService) ListSub2APITimeline(ctx context.Context, _ *v1.ListSub2APITimelineRequest) (*v1.ListSub2APITimelineResponse, error) {
	list, err := s.uc.ListTimeline(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListSub2APITimelineResponse{Timeline: list}, nil
}

func (s *Sub2APIService) CreateSub2APITimelineItem(ctx context.Context, req *v1.CreateSub2APITimelineItemRequest) (*v1.Sub2APITimelineItemResponse, error) {
	item, err := s.uc.CreateTimelineItem(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.Sub2APITimelineItemResponse{Item: item}, nil
}

func (s *Sub2APIService) UpdateSub2APITimelineItem(ctx context.Context, req *v1.UpdateSub2APITimelineItemRequest) (*v1.Sub2APITimelineItemResponse, error) {
	item, err := s.uc.UpdateTimelineItem(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.Sub2APITimelineItemResponse{Item: item}, nil
}

func (s *Sub2APIService) DeleteSub2APITimelineItem(ctx context.Context, req *v1.DeleteSub2APITimelineItemRequest) (*v1.DeleteSub2APITimelineItemResponse, error) {
	if err := s.uc.DeleteTimelineItem(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteSub2APITimelineItemResponse{}, nil
}
