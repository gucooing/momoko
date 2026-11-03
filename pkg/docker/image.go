package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-units"

	v1 "momoko/api/gen/v1"
)

func (m *Manager) ListImages(ctx context.Context, req *v1.ListDockerImagesRequest) (*v1.ListDockerImagesResponse, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(req.GetLabels())
	if req != nil && req.Dangling != nil {
		filters.Add("dangling", strconv.FormatBool(req.GetDangling()))
	}
	items, err := cli.ImageList(ctx, imagetypes.ListOptions{
		All:     req.GetAll(),
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	// 镜像 → 使用它的容器名（含已停止容器），用于「使用情况」列与筛选
	usedBy := map[string][]string{}
	if containers, cerr := cli.ContainerList(ctx, containertypes.ListOptions{All: true}); cerr == nil {
		for _, c := range containers {
			name := c.ID
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			usedBy[c.ImageID] = append(usedBy[c.ImageID], name)
		}
	}

	stats := &v1.DockerImageStats{}
	all := make([]*v1.DockerImageSummary, 0, len(items))
	for _, item := range items {
		summary := toImageSummary(item)
		summary.UsedBy = usedBy[item.ID]
		summary.Containers = int64(len(summary.UsedBy))
		summary.Dangling = isDanglingImage(item.RepoTags)
		stats.Total++
		stats.TotalSize += item.Size
		switch {
		case summary.Containers > 0:
			stats.Used++
		case summary.Dangling:
			stats.Dangling++
			stats.ReclaimableSize += item.Size
		default:
			stats.Unused++
		}
		all = append(all, summary)
	}

	keyword := strings.ToLower(strings.TrimSpace(req.GetKeyword()))
	usage := req.GetUsage()
	filtered := make([]*v1.DockerImageSummary, 0, len(all))
	for _, item := range all {
		switch usage {
		case "used":
			if item.Containers == 0 {
				continue
			}
		case "unused":
			if item.Containers > 0 || item.Dangling {
				continue
			}
		case "dangling":
			if item.Containers > 0 || !item.Dangling {
				continue
			}
		}
		if keyword != "" && !imageMatches(item, keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	sortImages(filtered, req.GetSort())

	total := int64(len(filtered))
	filtered = pageSlice(filtered, req.GetPage(), req.GetPageSize())
	return &v1.ListDockerImagesResponse{Items: filtered, Total: total, Stats: stats}, nil
}

func isDanglingImage(tags []string) bool {
	for _, tag := range tags {
		if tag != "" && tag != "<none>:<none>" {
			return false
		}
	}
	return true
}

func imageMatches(item *v1.DockerImageSummary, keyword string) bool {
	if strings.Contains(strings.ToLower(strings.TrimPrefix(item.Id, "sha256:")), keyword) {
		return true
	}
	for _, tag := range item.RepoTags {
		if strings.Contains(strings.ToLower(tag), keyword) {
			return true
		}
	}
	for _, name := range item.UsedBy {
		if strings.Contains(strings.ToLower(name), keyword) {
			return true
		}
	}
	return false
}

func sortImages(items []*v1.DockerImageSummary, by string) {
	name := func(item *v1.DockerImageSummary) string {
		if len(item.RepoTags) > 0 && !item.Dangling {
			return item.RepoTags[0]
		}
		return "\uffff" + item.Id
	}
	sort.SliceStable(items, func(i, j int) bool {
		switch by {
		case "size":
			return items[i].Size > items[j].Size
		case "name":
			return name(items[i]) < name(items[j])
		default:
			return items[i].GetCreated().AsTime().After(items[j].GetCreated().AsTime())
		}
	})
}

// PruneImages 清理镜像：默认只清理悬空镜像；all=true 时清理所有未被容器使用的镜像。
func (m *Manager) PruneImages(ctx context.Context, all bool) (*v1.PruneDockerImagesResponse, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, m.taskTimeout())
	defer cancel()
	args := filters.NewArgs()
	args.Add("dangling", strconv.FormatBool(!all))
	report, err := cli.ImagesPrune(ctx, args)
	if err != nil {
		return nil, err
	}
	deleted := int64(0)
	for _, item := range report.ImagesDeleted {
		if item.Deleted != "" {
			deleted++
		}
	}
	return &v1.PruneDockerImagesResponse{Deleted: deleted, SpaceReclaimed: int64(report.SpaceReclaimed)}, nil
}

// SearchImages 通过引擎搜索 Docker Hub（创建容器时的镜像建议）。
func (m *Manager) SearchImages(ctx context.Context, term string, limit int32) ([]*v1.DockerImageSearchResult, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return []*v1.DockerImageSearchResult{}, nil
	}
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	items, err := cli.ImageSearch(ctx, term, registrytypes.SearchOptions{Limit: int(limit)})
	if err != nil {
		return nil, err
	}
	result := make([]*v1.DockerImageSearchResult, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.DockerImageSearchResult{
			Name:        item.Name,
			Description: item.Description,
			Stars:       int64(item.StarCount),
			Official:    item.IsOfficial,
		})
	}
	return result, nil
}

func (m *Manager) Image(ctx context.Context, id string) (*v1.DockerImageInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	data, err := cli.ImageInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	return toImageInfo(data), nil
}

func (m *Manager) PullImage(ctx context.Context, req *v1.PullDockerImageRequest) *v1.DockerTaskInfo {
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_IMAGE_PULL, "拉取镜像 "+req.GetReference(), m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		authHeader, err := registryAuthHeader(req.GetRegistryAuth())
		if err != nil {
			return "", err
		}
		reader, err := cli.ImagePull(taskCtx, req.GetReference(), imagetypes.PullOptions{
			RegistryAuth: authHeader,
			Platform:     req.GetPlatform(),
		})
		if err != nil {
			return "", err
		}
		defer reader.Close()
		// 一路给终端日志渲染，一路解析分层进度汇总成「68% (32.1 MB/47.2 MB) 2/6」写进任务 Progress
		pr, pw := io.Pipe()
		done := make(chan struct{})
		go func() {
			defer close(done)
			trackPullProgress(pr, emit)
			_, _ = io.Copy(io.Discard, pr)
		}()
		streamErr := streamTaskOutput(taskCtx, io.TeeReader(reader, pw), emit)
		_ = pw.CloseWithError(streamErr)
		<-done
		if streamErr != nil {
			return "", streamErr
		}
		return req.GetReference(), nil
	})
}

func (m *Manager) TagImage(ctx context.Context, id, target string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	repo, tag := splitImageRef(target)
	return cli.ImageTag(ctx, id, imageRef(repo, tag))
}

func (m *Manager) UpdateImageTags(ctx context.Context, req *v1.UpdateDockerImageTagsRequest) error {
	if req.GetImageId() == "" {
		return errors.New("镜像 ID 不能为空")
	}
	for _, tag := range req.GetAddTags() {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		if err := m.TagImage(ctx, req.GetImageId(), tag); err != nil {
			return err
		}
	}
	for _, tag := range req.GetDeleteTags() {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		if err := m.DeleteImage(ctx, tag, req.GetForceDelete(), false); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) DeleteImage(ctx context.Context, id string, force bool, pruneChildren bool) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	_, err = cli.ImageRemove(ctx, id, imagetypes.RemoveOptions{
		Force:         force,
		PruneChildren: pruneChildren,
	})
	return err
}

func (m *Manager) ImageHistory(ctx context.Context, id string) ([]*v1.DockerImageHistoryItem, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	items, err := cli.ImageHistory(ctx, id)
	if err != nil {
		return nil, err
	}
	result := make([]*v1.DockerImageHistoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, toImageHistory(item))
	}
	return result, nil
}

type pullLayer struct {
	current int64
	total   int64
	done    bool
}

// trackPullProgress 解析 docker pull 的 JSON 消息流，按层汇总进度并节流上报。
func trackPullProgress(r io.Reader, emit func(*v1.DockerTaskInfo)) {
	dec := json.NewDecoder(r)
	layers := map[string]*pullLayer{}
	order := []string{}
	last := time.Time{}
	report := func(force bool) {
		if !force && time.Since(last) < 500*time.Millisecond {
			return
		}
		last = time.Now()
		var cur, total int64
		finished := 0
		for _, id := range order {
			l := layers[id]
			if l.done {
				finished++
			}
			if l.total > 0 {
				total += l.total
				if l.done {
					cur += l.total
				} else {
					cur += l.current
				}
			}
		}
		if len(order) == 0 {
			return
		}
		pct := 0
		if total > 0 {
			pct = int(cur * 100 / total)
		} else if finished == len(order) {
			pct = 100
		}
		if pct > 100 {
			pct = 100
		}
		emit(&v1.DockerTaskInfo{Progress: fmt.Sprintf("%d%% (%s/%s) %d/%d", pct, units.HumanSize(float64(cur)), units.HumanSize(float64(total)), finished, len(order))})
	}
	for {
		var msg jsonmessage.JSONMessage
		if err := dec.Decode(&msg); err != nil {
			report(true)
			return
		}
		if msg.ID == "" || strings.Contains(msg.ID, ":") {
			continue
		}
		l, ok := layers[msg.ID]
		if !ok {
			l = &pullLayer{}
			layers[msg.ID] = l
			order = append(order, msg.ID)
		}
		status := strings.ToLower(msg.Status)
		switch {
		case strings.HasPrefix(status, "downloading") && msg.Progress != nil:
			l.current = msg.Progress.Current
			if msg.Progress.Total > 0 {
				l.total = msg.Progress.Total
			}
		case strings.Contains(status, "complete") || strings.Contains(status, "already exists"):
			if strings.Contains(status, "download complete") {
				l.current = l.total
			} else {
				l.done = true
			}
		}
		report(false)
	}
}

func streamTaskOutput(ctx context.Context, reader io.Reader, emit func(*v1.DockerTaskInfo)) error {
	return jsonmessage.DisplayJSONMessagesStream(reader, taskEventWriter{ctx: ctx, emit: emit}, 0, true, nil)
}

type taskEventWriter struct {
	ctx  context.Context
	emit func(*v1.DockerTaskInfo)
}

func (w taskEventWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) > 0 {
		w.emit(&v1.DockerTaskInfo{Message: string(p)})
	}
	return len(p), nil
}
