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

	"github.com/distribution/reference"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
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
	ref := strings.TrimSpace(req.GetReference())
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_IMAGE_PULL, "拉取镜像 "+ref, ref, m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		if err := m.pullImage(taskCtx, ref, req.GetPlatform(), req.GetRegistryAuth(), emit); err != nil {
			return "", err
		}
		return ref, nil
	})
}

// pullImage 拉取镜像：未显式给凭据时按镜像所在仓库自动带上已保存的仓库认证；
// 拉取输出一路给任务终端日志，一路解析成分层进度写进任务 Pull / Progress。
func (m *Manager) pullImage(ctx context.Context, ref, platform string, auth *v1.DockerRegistryAuth, emit func(*v1.DockerTaskInfo)) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	if auth == nil {
		auth = storedRegistryAuth(m.Config(), ref)
	}
	authHeader, err := registryAuthHeader(auth)
	if err != nil {
		return err
	}
	if strings.TrimSpace(platform) == "" {
		platform = m.Config().GetDefaultPlatform()
	}
	reader, err := cli.ImagePull(ctx, ref, imagetypes.PullOptions{
		RegistryAuth: authHeader,
		Platform:     platform,
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	pr, pw := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		trackPullProgress(pr, emit)
		_, _ = io.Copy(io.Discard, pr)
	}()
	streamErr := streamTaskOutput(ctx, io.TeeReader(reader, pw), emit)
	_ = pw.CloseWithError(streamErr)
	<-done
	return streamErr
}

// registryHost 返回镜像引用所在仓库的主机名（Docker Hub 统一为 docker.io）。
func registryHost(ref string) string {
	named, err := reference.ParseNormalizedNamed(strings.TrimSpace(ref))
	if err != nil {
		return ""
	}
	return reference.Domain(named)
}

func normalizeRegistryHost(addr string) string {
	host := strings.ToLower(strings.TrimSpace(addr))
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	host = strings.TrimSuffix(host, "/")
	host = strings.TrimSuffix(host, "/v1")
	host = strings.TrimSuffix(host, "/v2")
	switch host {
	case "index.docker.io", "registry-1.docker.io", "registry.hub.docker.com", "hub.docker.com":
		return "docker.io"
	}
	return host
}

// storedRegistryAuth 在已保存的仓库认证里找与镜像仓库匹配的一条。
func storedRegistryAuth(cfg *v1.DockerConfigInfo, ref string) *v1.DockerRegistryAuth {
	host := registryHost(ref)
	if host == "" || cfg == nil {
		return nil
	}
	for _, item := range cfg.GetRegistryAuths() {
		if item != nil && normalizeRegistryHost(item.ServerAddress) == host {
			return item
		}
	}
	return nil
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
	// waiting / downloading / extracting / done / exists
	status string
}

// trackPullProgress 解析 docker pull 的 JSON 消息流，按层汇总进度（含各层状态、下载速度）并节流上报。
func trackPullProgress(r io.Reader, emit func(*v1.DockerTaskInfo)) {
	dec := json.NewDecoder(r)
	layers := map[string]*pullLayer{}
	order := []string{}
	last := time.Time{}
	var speedAt time.Time
	var speedBytes int64
	var speed float64
	report := func(force bool) {
		if !force && time.Since(last) < 500*time.Millisecond {
			return
		}
		if len(order) == 0 {
			return
		}
		now := time.Now()
		last = now
		progress := &v1.DockerPullProgress{LayersTotal: int32(len(order))}
		for _, id := range order {
			l := layers[id]
			switch l.status {
			case "done", "exists":
				progress.LayersDone++
			case "downloading":
				progress.LayersDownloading++
			}
			if l.total > 0 {
				progress.Total += l.total
				if l.status == "done" || l.status == "extracting" {
					progress.Current += l.total
				} else if l.status != "exists" {
					progress.Current += l.current
				} else {
					progress.Total -= l.total
				}
			}
			progress.Layers = append(progress.Layers, &v1.DockerPullLayer{Id: id, Status: l.status, Current: l.current, Total: l.total})
		}
		// 速度：相邻两次上报的字节差做指数平滑
		if !speedAt.IsZero() {
			if dt := now.Sub(speedAt).Seconds(); dt > 0 {
				inst := float64(progress.Current-speedBytes) / dt
				if inst < 0 {
					inst = 0
				}
				speed = speed*0.6 + inst*0.4
			}
		}
		speedAt, speedBytes = now, progress.Current
		if progress.LayersDownloading == 0 {
			speed = 0
		}
		progress.BytesPerSecond = int64(speed)
		switch {
		case progress.Total > 0:
			progress.Percent = int32(progress.Current * 100 / progress.Total)
		case progress.LayersDone == progress.LayersTotal:
			progress.Percent = 100
		}
		if progress.Percent > 100 {
			progress.Percent = 100
		}
		emit(&v1.DockerTaskInfo{
			Pull:     progress,
			Progress: fmt.Sprintf("%d%% (%s/%s) %d/%d", progress.Percent, units.HumanSize(float64(progress.Current)), units.HumanSize(float64(progress.Total)), progress.LayersDone, progress.LayersTotal),
		})
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
			l = &pullLayer{status: "waiting"}
			layers[msg.ID] = l
			order = append(order, msg.ID)
		}
		status := strings.ToLower(msg.Status)
		switch {
		case strings.HasPrefix(status, "downloading"):
			l.status = "downloading"
			if msg.Progress != nil {
				l.current = msg.Progress.Current
				if msg.Progress.Total > 0 {
					l.total = msg.Progress.Total
				}
			}
		case strings.Contains(status, "download complete"), strings.HasPrefix(status, "verifying"):
			l.current = l.total
			l.status = "extracting"
		case strings.HasPrefix(status, "extracting"):
			l.current = l.total
			l.status = "extracting"
		case strings.Contains(status, "pull complete"):
			l.current = l.total
			l.status = "done"
		case strings.Contains(status, "already exists"):
			l.status = "exists"
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

const (
	// ImageExportPath 导出镜像为 tar（docker save），GET ?ids=…
	ImageExportPath = "/api/v1/docker/image-export"
	// ImageImportPath 从 tar 导入镜像（docker load），POST 请求体为 tar
	ImageImportPath = "/api/v1/docker/image-import"
)

// SaveImages 把若干镜像（ID 或 引用）导出为一个 tar 流，调用方负责关闭。
func (m *Manager) SaveImages(ctx context.Context, ids []string) (io.ReadCloser, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.New("没有要导出的镜像")
	}
	return cli.ImageSave(ctx, ids)
}

// LoadImages 从 tar 流导入镜像，返回导入的镜像引用（无标签时为镜像 ID）。
func (m *Manager) LoadImages(ctx context.Context, input io.Reader) ([]string, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	resp, err := cli.ImageLoad(ctx, input, client.ImageLoadWithQuiet(true))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	loaded := []string{}
	dec := json.NewDecoder(resp.Body)
	for {
		var msg jsonmessage.JSONMessage
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return loaded, err
		}
		if msg.Error != nil {
			return loaded, errors.New(msg.Error.Message)
		}
		for _, line := range strings.Split(msg.Stream, "\n") {
			line = strings.TrimSpace(line)
			if ref, ok := strings.CutPrefix(line, "Loaded image: "); ok {
				loaded = append(loaded, ref)
			} else if id, ok := strings.CutPrefix(line, "Loaded image ID: "); ok {
				loaded = append(loaded, id)
			}
		}
	}
	return loaded, nil
}
