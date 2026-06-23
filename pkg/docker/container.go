package docker

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"

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
	TotalUsage uint64 `json:"total_usage"`
}

type memoryStatsPayload struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
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

func (m *Manager) ListContainers(ctx context.Context, req *v1.ListDockerContainersRequest) (*v1.ListDockerContainersResponse, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(req.GetLabels())
	if req.GetStatus() != "" {
		filters.Add("status", req.GetStatus())
	}
	if req.GetName() != "" {
		filters.Add("name", req.GetName())
	}
	if req.GetImage() != "" {
		filters.Add("ancestor", req.GetImage())
	}
	if req.GetNetwork() != "" {
		filters.Add("network", req.GetNetwork())
	}
	items, err := cli.ContainerList(ctx, containertypes.ListOptions{
		All:     req.GetAll(),
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}
	total := int64(len(items))
	items = pageSlice(items, req.GetPage(), req.GetPageSize())
	result := make([]*v1.DockerContainerSummary, 0, len(items))
	for _, item := range items {
		result = append(result, toContainerSummary(item))
	}
	return &v1.ListDockerContainersResponse{Items: result, Total: total}, nil
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

func (m *Manager) CreateContainer(ctx context.Context, opts *v1.DockerContainerCreateOptions) (string, error) {
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
		RestartPolicy: toRestartPolicy(opts.GetRestartPolicy()),
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
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_CONTAINER_RECREATE, "重建容器 "+titleTarget, m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		emit(&v1.DockerTaskInfo{Message: "检查旧容器"})
		oldInfo, err := m.Container(taskCtx, req.GetId())
		if err != nil {
			return "", err
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
		id, err := m.CreateContainer(taskCtx, req.GetOptions())
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
