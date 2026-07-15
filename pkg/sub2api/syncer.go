package sub2api

import (
	"context"
	"time"
)

const incrementalSyncLookback = time.Hour

// Syncer 负责把 Sub2API 的使用记录增量/全量同步到本地存储。
type Syncer struct {
	manager *Manager
}

func NewSyncer(manager *Manager) *Syncer {
	if manager == nil {
		manager = NewManager()
	}
	return &Syncer{manager: manager}
}

// NewFinishedSyncState 根据同步结果构造最终状态。
func NewFinishedSyncState(start, next time.Time, result *SyncResult, err error) SyncState {
	state := SyncState{
		Status:       SyncStatusSuccess,
		LastSyncTime: &start,
		NextSyncTime: &next,
	}
	if result != nil {
		state.LatestRecordTime = result.LatestRecordTime
		state.SyncedRecords = result.SyncedRecords
	}
	if err != nil {
		state.Status = SyncStatusError
		state.Error = err.Error()
	}
	return state
}

func (s *Syncer) Sync(ctx context.Context, cfg ClientConfig, store UsageStore, opts SyncOptions) (*SyncResult, error) {
	var usageCutoff *time.Time
	if !opts.Full {
		latest, err := store.LatestUsageRecordTime(ctx)
		if err != nil {
			return nil, err
		}
		if latest != nil {
			t := latest.Add(-incrementalSyncLookback)
			usageCutoff = &t
		}
	}
	upstreamCutoff, err := s.upstreamCutoff(ctx, store, opts.Full)
	if err != nil {
		return nil, err
	}

	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = int(DefaultPageSize)
	}

	result, err := s.syncPaged(ctx, store, usageCutoff, pageSize, func(page int) (*UsageListResult, error) {
		return s.manager.ListUsage(cfg, UsageListOptions{
			Page:     page,
			PageSize: pageSize,
			SortBy:   "created_at",
			SortDesc: true,
		})
	})
	if err != nil {
		return result, err
	}

	upstreamEnd := time.Now()
	upstreamResult, err := s.syncPaged(ctx, store, upstreamCutoff, pageSize, func(page int) (*UsageListResult, error) {
		return s.manager.ListUpstreamErrors(cfg, UsageListOptions{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    "created_at",
			SortDesc:  true,
			StartTime: upstreamErrorStart(upstreamCutoff, upstreamEnd),
			EndTime:   &upstreamEnd,
		})
	})
	result = mergeSyncResults(result, upstreamResult)
	if err != nil {
		return result, err
	}

	// 同步分组：以管理端存活列表为准，本地多出的标 deleted。
	live, gerr := s.manager.ListGroups(cfg)
	if gerr != nil {
		return result, gerr
	}
	if err := store.SyncGroups(ctx, live); err != nil {
		return result, err
	}

	if result.LatestRecordTime == nil {
		latest, err := store.LatestUsageRecordTime(ctx)
		if err != nil {
			return &SyncResult{SyncedRecords: result.SyncedRecords}, err
		}
		result.LatestRecordTime = latest
	}
	return result, nil
}

func (s *Syncer) upstreamCutoff(ctx context.Context, store UsageStore, full bool) (*time.Time, error) {
	if full {
		return nil, nil
	}
	latest, err := store.LatestUpstreamErrorRecordTime(ctx)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return nil, nil
	}
	t := latest.Add(-incrementalSyncLookback)
	return &t, nil
}

func (s *Syncer) syncPaged(ctx context.Context, store UsageStore, cutoff *time.Time, pageSize int, list func(page int) (*UsageListResult, error)) (*SyncResult, error) {
	result := &SyncResult{}
	for page := 1; ; page++ {
		listResult, err := list(page)
		if err != nil {
			return result, err
		}
		if len(listResult.Records) == 0 {
			break
		}

		insertRecords := listResult.Records[:0]
		oldRecords := 0
		for _, record := range listResult.Records {
			if cutoff != nil && !record.RequestTime.After(*cutoff) {
				oldRecords++
				continue
			}
			insertRecords = append(insertRecords, record)
			if result.LatestRecordTime == nil || record.RequestTime.After(*result.LatestRecordTime) {
				t := record.RequestTime
				result.LatestRecordTime = &t
			}
		}
		if len(insertRecords) > 0 {
			if err := store.SaveUsageRecords(ctx, insertRecords); err != nil {
				return result, err
			}
			result.SyncedRecords += int64(len(insertRecords))
		}
		if oldRecords == len(listResult.Records) || len(listResult.Records) < pageSize || (listResult.Total > 0 && page*pageSize >= listResult.Total) {
			break
		}
	}
	return result, nil
}

func mergeSyncResults(results ...*SyncResult) *SyncResult {
	merged := &SyncResult{}
	for _, result := range results {
		if result == nil {
			continue
		}
		merged.SyncedRecords += result.SyncedRecords
		if result.LatestRecordTime != nil && (merged.LatestRecordTime == nil || result.LatestRecordTime.After(*merged.LatestRecordTime)) {
			t := *result.LatestRecordTime
			merged.LatestRecordTime = &t
		}
	}
	return merged
}

func upstreamErrorStart(cutoff *time.Time, end time.Time) *time.Time {
	minStart := end.Add(-30 * 24 * time.Hour)
	if cutoff != nil {
		if cutoff.Before(minStart) {
			return &minStart
		}
		return cutoff
	}
	return &minStart
}
