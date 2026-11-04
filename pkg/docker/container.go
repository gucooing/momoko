package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

type ExecSession struct {
	ID     string
	Closer func()
	Reader io.Reader
	Writer io.Writer
}

type containerStatsPayload struct {
	CPUStats    cpuStatsPayload                `json:"cpu_stats"`
	PreCPUStats cpuStatsPayload                `json:"precpu_stats"`
	MemoryStats memoryStatsPayload             `json:"memory_stats"`
	Networks    map[string]networkStatsPayload `json:"networks"`
	BlkioStats  blkioStatsPayload              `json:"blkio_stats"`
}

type cpuStatsPayload struct {
	CPUUsage       cpuUsagePayload `json:"cpu_usage"`
	SystemCPUUsage uint64          `json:"system_cpu_usage"`
	OnlineCPUs     uint32          `json:"online_cpus"`
}

type cpuUsagePayload struct {
	TotalUsage  uint64   `json:"total_usage"`
	PercpuUsage []uint64 `json:"percpu_usage"`
}

type memoryStatsPayload struct {
	Usage uint64            `json:"usage"`
	Limit uint64            `json:"limit"`
	Stats map[string]uint64 `json:"stats"`
}

type networkStatsPayload struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type blkioStatsPayload struct {
	IOServiceBytesRecursive []blkioEntryPayload `json:"io_service_bytes_recursive"`
}

type blkioEntryPayload struct {
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

// 容器状态分类（列表 chips）：running / stopped / exited / paused
const (
	containerCatRunning = "running"
	containerCatStopped = "stopped"
	containerCatExited  = "exited"
	containerCatPaused  = "paused"
)

var exitedStatusRe = regexp.MustCompile(`^Exited \((-?\d+)\)`)

// summaryExitCode 从 docker ps 的状态描述（如 "Exited (137) 12 minutes ago"）解析退出码。
func summaryExitCode(status string) int32 {
	match := exitedStatusRe.FindStringSubmatch(status)
	if len(match) < 2 {
		return 0
	}
	code, _ := strconv.Atoi(match[1])
	return int32(code)
}

// containerCategory 把引擎状态归为列表分类：正常退出（退出码 0）或尚未启动算「已停止」，
// 异常退出（退出码非 0）或已死亡算「已退出」。
func containerCategory(item containertypes.Summary) string {
	switch item.State {
	case "running", "restarting":
		return containerCatRunning
	case "paused":
		return containerCatPaused
	case "exited":
		if summaryExitCode(item.Status) != 0 {
			return containerCatExited
		}
		return containerCatStopped
	case "dead":
		return containerCatExited
	default:
		return containerCatStopped
	}
}

func containerMatches(item containertypes.Summary, keyword string) bool {
	if strings.HasPrefix(strings.ToLower(item.ID), keyword) || strings.Contains(strings.ToLower(item.Image), keyword) {
		return true
	}
	for _, name := range item.Names {
		if strings.Contains(strings.ToLower(strings.TrimPrefix(name, "/")), keyword) {
			return true
		}
	}
	for _, port := range item.Ports {
		if port.PublicPort > 0 && strings.Contains(strconv.Itoa(int(port.PublicPort)), keyword) {
			return true
		}
		if port.PrivatePort > 0 && strings.Contains(strconv.Itoa(int(port.PrivatePort)), keyword) {
			return true
		}
	}
	return false
}

func (m *Manager) ListContainers(ctx context.Context, req *v1.ListDockerContainersRequest) (*v1.ListDockerContainersResponse, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(req.GetLabels())
	if req.GetNetwork() != "" {
		filters.Add("network", req.GetNetwork())
	}
	items, err := cli.ContainerList(ctx, containertypes.ListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, err
	}
	stats := &v1.DockerContainerListStats{}
	status := strings.TrimSpace(req.GetStatus())
	keyword := strings.ToLower(strings.TrimSpace(req.GetKeyword()))
	filtered := make([]containertypes.Summary, 0, len(items))
	for _, item := range items {
		cat := containerCategory(item)
		stats.Total++
		switch cat {
		case containerCatRunning:
			stats.Running++
		case containerCatPaused:
			stats.Paused++
		case containerCatExited:
			stats.Exited++
		default:
			stats.Stopped++
		}
		if status != "" && cat != status {
			continue
		}
		if keyword != "" && !containerMatches(item, keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	total := int64(len(filtered))
	filtered = pageSlice(filtered, req.GetPage(), req.GetPageSize())
	result := make([]*v1.DockerContainerSummary, 0, len(filtered))
	for _, item := range filtered {
		result = append(result, toContainerSummary(item))
	}
	fillContainerTimes(ctx, cli, result)
	return &v1.ListDockerContainersResponse{Items: result, Total: total, Stats: stats}, nil
}

// fillContainerTimes 并发 inspect 当前页容器，补齐启动 / 停止时间与退出码（列表只给相对时间的英文描述）。
func fillContainerTimes(ctx context.Context, cli *client.Client, items []*v1.DockerContainerSummary) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, item := range items {
		wg.Add(1)
		go func(item *v1.DockerContainerSummary) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			data, err := cli.ContainerInspect(ctx, item.Id)
			if err != nil || data.State == nil {
				return
			}
			item.ExitCode = int32(data.State.ExitCode)
			item.StartedAt = parseDockerTime(data.State.StartedAt)
			item.FinishedAt = parseDockerTime(data.State.FinishedAt)
		}(item)
	}
	wg.Wait()
}

func parseDockerTime(value string) *timestamppb.Timestamp {
	if value == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || t.Year() < 1971 {
		return nil
	}
	return timestamppb.New(t)
}

// ContainersUsage 并发取运行中容器的一次性资源占用（CPU% 与内存，口径同 docker stats）。
func (m *Manager) ContainersUsage(ctx context.Context, ids []string) ([]*v1.DockerContainerUsage, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	if len(ids) > 100 {
		ids = ids[:100]
	}
	result := make([]*v1.DockerContainerUsage, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resp, err := cli.ContainerStats(ctx, id, false)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			var stats containerStatsPayload
			if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
				return
			}
			result[i] = &v1.DockerContainerUsage{
				Id:          id,
				CpuPercent:  cpuPercent(stats),
				MemoryUsage: memoryUsage(stats.MemoryStats),
				MemoryLimit: stats.MemoryStats.Limit,
			}
		}(i, id)
	}
	wg.Wait()
	out := make([]*v1.DockerContainerUsage, 0, len(result))
	for _, item := range result {
		if item != nil {
			out = append(out, item)
		}
	}
	return out, nil
}

func cpuPercent(stats containerStatsPayload) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(stats.CPUStats.SystemCPUUsage) - float64(stats.PreCPUStats.SystemCPUUsage)
	online := float64(stats.CPUStats.OnlineCPUs)
	if online == 0 {
		online = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if online == 0 {
		online = 1
	}
	if cpuDelta <= 0 || sysDelta <= 0 {
		return 0
	}
	return cpuDelta / sysDelta * online * 100
}

// memoryUsage 与 docker CLI 一致：扣除可回收的页缓存（cgroup v2 inactive_file / v1 total_inactive_file）。
func memoryUsage(mem memoryStatsPayload) uint64 {
	inactive, ok := mem.Stats["inactive_file"]
	if !ok {
		inactive = mem.Stats["total_inactive_file"]
	}
	if inactive < mem.Usage {
		return mem.Usage - inactive
	}
	return mem.Usage
}

func (m *Manager) Container(ctx context.Context, id string) (*v1.DockerContainerInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	data, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	return toContainerInfo(data), nil
}

// CreateContainer 本地已有镜像时直接创建（按需启动）并返回容器 ID；
// 本地没有镜像时返回「拉取镜像并创建」后台任务，拉取进度在任务里可见。
func (m *Manager) CreateContainer(ctx context.Context, opts *v1.DockerContainerCreateOptions) (string, *v1.DockerTaskInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", nil, err
	}
	image := strings.TrimSpace(opts.GetImage())
	if image == "" {
		return "", nil, fmt.Errorf("镜像不能为空")
	}
	local, err := m.imageExists(ctx, cli, image)
	if err != nil {
		return "", nil, err
	}
	if local {
		id, err := m.createContainer(ctx, opts)
		return id, nil, err
	}
	target := strings.TrimSpace(opts.GetName())
	if target == "" {
		target = image
	}
	task := m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_CONTAINER_CREATE, "创建容器 "+target, target, m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		emit(&v1.DockerTaskInfo{Message: "本地没有镜像 " + image + "，开始拉取\r\n"})
		if err := m.pullImage(taskCtx, image, opts.GetPlatform(), nil, emit); err != nil {
			return "", err
		}
		emit(&v1.DockerTaskInfo{Message: "创建容器\r\n"})
		return m.createContainer(taskCtx, opts)
	})
	return "", task, nil
}

func (m *Manager) imageExists(ctx context.Context, cli *client.Client, image string) (bool, error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	if _, err := cli.ImageInspect(ctx, image); err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *Manager) createContainer(ctx context.Context, opts *v1.DockerContainerCreateOptions) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	exposed, portBindings, err := toPortBindings(opts.GetPorts())
	if err != nil {
		return "", err
	}
	platform, err := parsePlatform(opts.GetPlatform())
	if err != nil {
		return "", err
	}
	hostConfig := &containertypes.HostConfig{
		RestartPolicy: toRestartPolicyWithRetries(opts.GetRestartPolicy(), opts.GetRestartMaxRetries()),
		AutoRemove:    opts.GetAutoRemove(),
		Privileged:    opts.GetPrivileged(),
		PortBindings:  portBindings,
		Mounts:        toDockerMounts(opts.GetMounts()),
		Resources:     toContainerResources(opts),
	}
	if opts.GetNetwork() != "" {
		hostConfig.NetworkMode = containertypes.NetworkMode(opts.GetNetwork())
	}
	networking := &networktypes.NetworkingConfig{}
	if opts.GetNetwork() != "" {
		networking.EndpointsConfig = map[string]*networktypes.EndpointSettings{
			opts.GetNetwork(): {},
		}
	}
	resp, err := cli.ContainerCreate(
		ctx,
		&containertypes.Config{
			Hostname:     opts.GetHostname(),
			User:         opts.GetUser(),
			Env:          opts.GetEnv(),
			Cmd:          opts.GetCmd(),
			Entrypoint:   opts.GetEntrypoint(),
			Image:        opts.GetImage(),
			WorkingDir:   opts.GetWorkingDir(),
			Labels:       opts.GetLabels(),
			Tty:          opts.GetTty(),
			OpenStdin:    opts.GetOpenStdin(),
			ExposedPorts: exposed,
		},
		hostConfig,
		networking,
		platform,
		opts.GetName(),
	)
	if err != nil {
		return "", err
	}
	if opts.GetStart() {
		if err := cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
			return resp.ID, fmt.Errorf("容器已创建，但启动失败: %w", err)
		}
	}
	return resp.ID, nil
}

func (m *Manager) UpdateContainer(ctx context.Context, req *v1.UpdateDockerContainerRequest) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	if strings.TrimSpace(req.GetName()) != "" {
		if err := cli.ContainerRename(ctx, req.GetId(), req.GetName()); err != nil {
			return err
		}
	}
	_, err = cli.ContainerUpdate(ctx, req.GetId(), containertypes.UpdateConfig{
		Resources:     toUpdateResources(req),
		RestartPolicy: toRestartPolicy(req.GetRestartPolicy()),
	})
	return err
}

func (m *Manager) RecreateContainer(ctx context.Context, req *v1.RecreateDockerContainerRequest) *v1.DockerTaskInfo {
	titleTarget := req.GetOptions().GetName()
	if titleTarget == "" {
		titleTarget = req.GetId()
	}
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_CONTAINER_RECREATE, "重建容器 "+titleTarget, titleTarget, m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		emit(&v1.DockerTaskInfo{Message: "检查旧容器"})
		oldInfo, err := m.Container(taskCtx, req.GetId())
		if err != nil {
			return "", err
		}
		// 新镜像本地没有时先拉取，避免删掉旧容器后才发现镜像拉不下来
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		image := strings.TrimSpace(req.GetOptions().GetImage())
		local, err := m.imageExists(taskCtx, cli, image)
		if err != nil {
			return "", err
		}
		if !local {
			emit(&v1.DockerTaskInfo{Message: "本地没有镜像 " + image + "，开始拉取\r\n"})
			if err := m.pullImage(taskCtx, image, req.GetOptions().GetPlatform(), nil, emit); err != nil {
				return "", err
			}
		}
		if oldInfo.GetState().GetRunning() {
			emit(&v1.DockerTaskInfo{Message: "停止旧容器"})
			if err := m.StopContainer(taskCtx, req.GetId(), 10); err != nil {
				return "", err
			}
		}
		emit(&v1.DockerTaskInfo{Message: "删除旧容器"})
		if err := m.DeleteContainer(taskCtx, req.GetId(), req.GetForce(), req.GetRemoveVolumes()); err != nil {
			return "", err
		}
		emit(&v1.DockerTaskInfo{Message: "创建新容器"})
		id, err := m.createContainer(taskCtx, req.GetOptions())
		if err != nil {
			return "", err
		}
		return id, nil
	})
}

func (m *Manager) StartContainer(ctx context.Context, id string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.ContainerStart(ctx, id, containertypes.StartOptions{})
}

func (m *Manager) StopContainer(ctx context.Context, id string, timeoutSeconds int32) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	var timeout *int
	if timeoutSeconds > 0 {
		value := int(timeoutSeconds)
		timeout = &value
	}
	return cli.ContainerStop(ctx, id, containertypes.StopOptions{Timeout: timeout})
}

func (m *Manager) RestartContainer(ctx context.Context, id string, timeoutSeconds int32) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	var timeout *int
	if timeoutSeconds > 0 {
		value := int(timeoutSeconds)
		timeout = &value
	}
	return cli.ContainerRestart(ctx, id, containertypes.StopOptions{Timeout: timeout})
}

func (m *Manager) KillContainer(ctx context.Context, id, signal string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.ContainerKill(ctx, id, signal)
}

func (m *Manager) PauseContainer(ctx context.Context, id string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.ContainerPause(ctx, id)
}

func (m *Manager) UnpauseContainer(ctx context.Context, id string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.ContainerUnpause(ctx, id)
}

func (m *Manager) RenameContainer(ctx context.Context, id, name string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.ContainerRename(ctx, id, name)
}

func (m *Manager) DeleteContainer(ctx context.Context, id string, force bool, removeVolumes bool) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.ContainerRemove(ctx, id, containertypes.RemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	})
}

func (m *Manager) ContainerLogs(ctx context.Context, id string, opts containertypes.LogsOptions) (io.ReadCloser, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	inspectCtx, cancel := m.withTimeout(ctx)
	defer cancel()
	inspect, err := cli.ContainerInspect(inspectCtx, id)
	if err != nil {
		return nil, err
	}
	reader, err := cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	if inspect.Config != nil && inspect.Config.Tty {
		return reader, nil
	}
	return demuxDockerLogs(reader), nil
}

type demuxLogReader struct {
	*io.PipeReader
	source io.Closer
}

func (r *demuxLogReader) Close() error {
	_ = r.source.Close()
	return r.PipeReader.Close()
}

func demuxDockerLogs(source io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		_, err := stdcopy.StdCopy(pw, pw, source)
		_ = source.Close()
		_ = pw.CloseWithError(err)
	}()
	return &demuxLogReader{PipeReader: pr, source: source}
}

func (m *Manager) ContainerStats(ctx context.Context, id string) (*v1.DockerContainerStats, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err := cli.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	stats := containerStatsPayload{
		Networks: map[string]networkStatsPayload{},
		BlkioStats: blkioStatsPayload{
			IOServiceBytesRecursive: []blkioEntryPayload{},
		},
	}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}
	return toProtoContainerStats(stats), nil
}

func toProtoContainerStats(stats containerStatsPayload) *v1.DockerContainerStats {
	networks := make([]*v1.DockerNetworkStats, 0, len(stats.Networks))
	for _, item := range stats.Networks {
		networks = append(networks, &v1.DockerNetworkStats{
			RxBytes: item.RxBytes,
			TxBytes: item.TxBytes,
		})
	}

	blkioEntries := make([]*v1.DockerBlkioEntry, 0, len(stats.BlkioStats.IOServiceBytesRecursive))
	for _, item := range stats.BlkioStats.IOServiceBytesRecursive {
		op := toProtoBlkioOperation(item.Op)
		if op == v1.DockerBlkioOperation_DOCKER_BLKIO_OPERATION_UNKNOWN {
			continue
		}
		blkioEntries = append(blkioEntries, &v1.DockerBlkioEntry{
			Op:    op,
			Value: item.Value,
		})
	}

	return &v1.DockerContainerStats{
		CpuStats:    toProtoCPUStats(stats.CPUStats),
		PrecpuStats: toProtoCPUStats(stats.PreCPUStats),
		MemoryStats: &v1.DockerMemoryStats{
			Usage: stats.MemoryStats.Usage,
			Limit: stats.MemoryStats.Limit,
		},
		Networks: networks,
		BlkioStats: &v1.DockerBlkioStats{
			IoServiceBytesRecursive: blkioEntries,
		},
	}
}

func toProtoCPUStats(stats cpuStatsPayload) *v1.DockerCPUStats {
	return &v1.DockerCPUStats{
		CpuUsage: &v1.DockerCPUUsage{
			TotalUsage: stats.CPUUsage.TotalUsage,
		},
		SystemCpuUsage: stats.SystemCPUUsage,
		OnlineCpus:     stats.OnlineCPUs,
	}
}

func toProtoBlkioOperation(op string) v1.DockerBlkioOperation {
	switch strings.ToLower(op) {
	case "read":
		return v1.DockerBlkioOperation_DOCKER_BLKIO_OPERATION_READ
	case "write":
		return v1.DockerBlkioOperation_DOCKER_BLKIO_OPERATION_WRITE
	default:
		return v1.DockerBlkioOperation_DOCKER_BLKIO_OPERATION_UNKNOWN
	}
}

func (m *Manager) CreateExec(ctx context.Context, containerID string, opts containertypes.ExecOptions) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err := cli.ContainerExecCreate(ctx, containerID, opts)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (m *Manager) AttachExec(ctx context.Context, execID string, opts containertypes.ExecAttachOptions) (*ExecSession, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	resp, err := cli.ContainerExecAttach(ctx, execID, opts)
	if err != nil {
		return nil, err
	}
	return &ExecSession{
		ID:     execID,
		Closer: resp.Close,
		Reader: resp.Reader,
		Writer: resp.Conn,
	}, nil
}
