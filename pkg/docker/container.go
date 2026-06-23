package docker

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
)

func (m *Manager) ListContainers(ctx context.Context, opts ContainerListOptions) ([]ContainerSummary, int64, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(opts.Labels)
	if opts.Status != "" {
		filters.Add("status", opts.Status)
	}
	if opts.Name != "" {
		filters.Add("name", opts.Name)
	}
	if opts.Image != "" {
		filters.Add("ancestor", opts.Image)
	}
	if opts.Network != "" {
		filters.Add("network", opts.Network)
	}
	items, err := cli.ContainerList(ctx, containertypes.ListOptions{
		All:     opts.All,
		Filters: filters,
	})
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(items))
	items = pageSlice(items, opts.Page, opts.PageSize)
	result := make([]ContainerSummary, 0, len(items))
	for _, item := range items {
		result = append(result, toContainerSummary(item))
	}
	return result, total, nil
}

func (m *Manager) Container(ctx context.Context, id string) (*ContainerInfo, error) {
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
	info := toContainerInfo(data)
	return &info, nil
}

func (m *Manager) CreateContainer(ctx context.Context, opts CreateContainerOptions) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	exposed, portBindings, err := toPortBindings(opts.Ports)
	if err != nil {
		return "", err
	}
	platform, err := parsePlatform(opts.Platform)
	if err != nil {
		return "", err
	}
	hostConfig := &containertypes.HostConfig{
		RestartPolicy: toRestartPolicy(opts.RestartPolicy),
		AutoRemove:    opts.AutoRemove,
		Privileged:    opts.Privileged,
		PortBindings:  portBindings,
		Mounts:        toDockerMounts(opts.Mounts),
		Resources:     toContainerResources(opts),
	}
	if opts.Network != "" {
		hostConfig.NetworkMode = containertypes.NetworkMode(opts.Network)
	}
	networking := &networktypes.NetworkingConfig{}
	if opts.Network != "" {
		networking.EndpointsConfig = map[string]*networktypes.EndpointSettings{
			opts.Network: {},
		}
	}
	resp, err := cli.ContainerCreate(
		ctx,
		&containertypes.Config{
			Hostname:     opts.Hostname,
			User:         opts.User,
			Env:          opts.Env,
			Cmd:          opts.Cmd,
			Entrypoint:   opts.Entrypoint,
			Image:        opts.Image,
			WorkingDir:   opts.WorkingDir,
			Labels:       opts.Labels,
			Tty:          opts.Tty,
			OpenStdin:    opts.OpenStdin,
			ExposedPorts: exposed,
		},
		hostConfig,
		networking,
		platform,
		opts.Name,
	)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (m *Manager) UpdateContainer(ctx context.Context, id string, opts UpdateContainerOptions) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	if strings.TrimSpace(opts.Name) != "" {
		if err := cli.ContainerRename(ctx, id, opts.Name); err != nil {
			return err
		}
	}
	_, err = cli.ContainerUpdate(ctx, id, containertypes.UpdateConfig{
		Resources:     toUpdateResources(opts),
		RestartPolicy: toRestartPolicy(opts.RestartPolicy),
	})
	return err
}

func (m *Manager) RecreateContainer(ctx context.Context, opts RecreateContainerOptions) *Task {
	titleTarget := opts.Create.Name
	if titleTarget == "" {
		titleTarget = opts.ID
	}
	return m.tasks.Start(ctx, "container_recreate", "重建容器 "+titleTarget, m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		emit(TaskEvent{Message: "检查旧容器"})
		oldInfo, err := m.Container(taskCtx, opts.ID)
		if err != nil {
			return "", err
		}
		if oldInfo.State.Running {
			emit(TaskEvent{Message: "停止旧容器"})
			if err := m.StopContainer(taskCtx, opts.ID, 10); err != nil {
				return "", err
			}
		}
		emit(TaskEvent{Message: "删除旧容器"})
		if err := m.DeleteContainer(taskCtx, opts.ID, opts.Force, opts.RemoveVolumes); err != nil {
			return "", err
		}
		emit(TaskEvent{Message: "创建新容器"})
		id, err := m.CreateContainer(taskCtx, opts.Create)
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

func (m *Manager) ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	inspect, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	if !opts.Stdout && !opts.Stderr {
		opts.Stdout = true
		opts.Stderr = true
	}
	if opts.Tail == "" {
		tail := m.Config().DefaultLogTail
		if tail <= 0 {
			tail = 200
		}
		opts.Tail = strconv.Itoa(int(tail))
	}
	reader, err := cli.ContainerLogs(ctx, id, containertypes.LogsOptions{
		ShowStdout: opts.Stdout,
		ShowStderr: opts.Stderr,
		Since:      opts.Since,
		Until:      opts.Until,
		Timestamps: opts.Timestamps,
		Follow:     opts.Follow,
		Tail:       opts.Tail,
		Details:    opts.Details,
	})
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

func (m *Manager) ContainerStats(ctx context.Context, id string, stream bool) (io.ReadCloser, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	resp, err := cli.ContainerStats(ctx, id, stream)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (m *Manager) ContainerStatsOnce(ctx context.Context, id string) (json.RawMessage, error) {
	reader, err := m.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (m *Manager) CreateExec(ctx context.Context, containerID string, opts ExecOptions) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err := cli.ContainerExecCreate(ctx, containerID, containertypes.ExecOptions{
		Cmd:          opts.Cmd,
		Env:          opts.Env,
		User:         opts.User,
		WorkingDir:   opts.WorkingDir,
		Tty:          opts.Tty,
		AttachStdin:  opts.AttachStdin,
		AttachStdout: opts.AttachStdout,
		AttachStderr: opts.AttachStderr,
		Detach:       opts.Detach,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (m *Manager) AttachExec(ctx context.Context, execID string, tty bool) (*ExecSession, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	resp, err := cli.ContainerExecAttach(ctx, execID, containertypes.ExecAttachOptions{
		Tty: tty,
	})
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
