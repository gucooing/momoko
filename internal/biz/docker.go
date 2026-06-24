package biz

import (
	"context"
	"io"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/go-kratos/kratos/v2/log"

	v1 "momoko/api/gen/v1"
	dockerpkg "momoko/pkg/docker"
)

const (
	DockerContainerLogsWSPath = dockerpkg.ContainerLogsWSPath
	DockerContainerExecWSPath = dockerpkg.ContainerExecWSPath
	DockerTaskWSPath          = dockerpkg.TaskWSPath
)

type DockerUsecase struct {
	config ConfigRepo
	docker *dockerpkg.Manager
}

func NewDockerUsecase(config ConfigRepo) (*DockerUsecase, error) {
	ctx := context.Background()
	cfg, err := config.DockerConfig(ctx)
	if err != nil {
		return nil, err
	}
	manager, err := dockerpkg.NewManager(cfg)
	if err != nil {
		// Docker 客户端初始化失败不应阻断服务启动，否则一份无法连接的持久化配置
		// （如启用了 TLS 却缺少证书路径）会导致整个面板无法启动且无界面可修复。
		// 此处以降级状态启动，运行时可通过 Docker 状态/配置页面查看并修正。
		log.Errorf("初始化 Docker 管理器失败，已降级启动: %v", err)
	}
	return &DockerUsecase{config: config, docker: manager}, nil
}

func (d *DockerUsecase) Config(ctx context.Context) (*v1.DockerConfigInfo, error) {
	return d.config.DockerConfig(ctx)
}

func (d *DockerUsecase) UpdateConfig(ctx context.Context, req *v1.UpdateDockerConfigRequest) (*v1.DockerConfigInfo, error) {
	cfg, err := d.config.UpdateDockerConfig(ctx, req.Config)
	if err != nil {
		return nil, err
	}
	if err = d.docker.Reconfigure(cfg); err != nil {
		return nil, ErrSystem(err)
	}
	return cfg, nil
}

func (d *DockerUsecase) TestConfig(ctx context.Context, req *v1.TestDockerConfigRequest) (*v1.DockerStatusResponse, error) {
	status, _ := d.docker.Test(ctx, req.GetConfig())
	return status, nil
}

func (d *DockerUsecase) Status(ctx context.Context) (*v1.DockerStatusResponse, error) {
	return d.docker.Status(ctx), nil
}

func (d *DockerUsecase) Tasks(ctx context.Context) ([]*v1.DockerTaskInfo, error) {
	return d.docker.Tasks(), nil
}

func (d *DockerUsecase) SubscribeTask(ctx context.Context, id string) (<-chan *v1.DockerTaskInfo, func(), error) {
	return d.docker.SubscribeTask(id)
}

func (d *DockerUsecase) ListContainers(ctx context.Context, req *v1.ListDockerContainersRequest) (*v1.ListDockerContainersResponse, error) {
	result, err := d.docker.ListContainers(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return result, nil
}

func (d *DockerUsecase) Container(ctx context.Context, id string) (*v1.DockerContainerInfo, error) {
	info, err := d.docker.Container(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return info, nil
}

func (d *DockerUsecase) CreateContainer(ctx context.Context, req *v1.CreateDockerContainerRequest) (string, error) {
	id, err := d.docker.CreateContainer(ctx, req.GetOptions())
	if err != nil {
		return "", ErrSystem(err)
	}
	return id, nil
}

func (d *DockerUsecase) UpdateContainer(ctx context.Context, req *v1.UpdateDockerContainerRequest) (*v1.DockerContainerInfo, *v1.DockerTaskInfo, error) {
	if req.Recreate {
		task := d.docker.RecreateContainer(ctx, &v1.RecreateDockerContainerRequest{
			Id:            req.GetId(),
			Options:       req.GetOptions(),
			Force:         req.GetForce(),
			RemoveVolumes: req.GetRemoveVolumes(),
		})
		return nil, task, nil
	}
	err := d.docker.UpdateContainer(ctx, req)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	info, err := d.docker.Container(ctx, req.Id)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	return info, nil, nil
}

func (d *DockerUsecase) RecreateContainer(ctx context.Context, req *v1.RecreateDockerContainerRequest) (*v1.DockerTaskInfo, error) {
	return d.docker.RecreateContainer(ctx, req), nil
}

func (d *DockerUsecase) StartContainer(ctx context.Context, id string) error {
	return d.containerAction(ctx, func() error { return d.docker.StartContainer(ctx, id) })
}

func (d *DockerUsecase) StopContainer(ctx context.Context, id string, timeout int32) error {
	return d.containerAction(ctx, func() error { return d.docker.StopContainer(ctx, id, timeout) })
}

func (d *DockerUsecase) RestartContainer(ctx context.Context, id string, timeout int32) error {
	return d.containerAction(ctx, func() error { return d.docker.RestartContainer(ctx, id, timeout) })
}

func (d *DockerUsecase) KillContainer(ctx context.Context, id, signal string) error {
	return d.containerAction(ctx, func() error { return d.docker.KillContainer(ctx, id, signal) })
}

func (d *DockerUsecase) PauseContainer(ctx context.Context, id string) error {
	return d.containerAction(ctx, func() error { return d.docker.PauseContainer(ctx, id) })
}

func (d *DockerUsecase) UnpauseContainer(ctx context.Context, id string) error {
	return d.containerAction(ctx, func() error { return d.docker.UnpauseContainer(ctx, id) })
}

func (d *DockerUsecase) RenameContainer(ctx context.Context, id, name string) error {
	return d.containerAction(ctx, func() error { return d.docker.RenameContainer(ctx, id, name) })
}

func (d *DockerUsecase) DeleteContainer(ctx context.Context, id string, force, removeVolumes bool) error {
	return d.containerAction(ctx, func() error { return d.docker.DeleteContainer(ctx, id, force, removeVolumes) })
}

func (d *DockerUsecase) ContainerLogs(ctx context.Context, id string, opts containertypes.LogsOptions) (io.ReadCloser, error) {
	reader, err := d.docker.ContainerLogs(ctx, id, opts)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return reader, nil
}

func (d *DockerUsecase) ContainerStats(ctx context.Context, id string) (*v1.DockerContainerStats, error) {
	stats, err := d.docker.ContainerStats(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return stats, nil
}

func (d *DockerUsecase) CreateExec(ctx context.Context, containerID string, opts containertypes.ExecOptions) (string, error) {
	id, err := d.docker.CreateExec(ctx, containerID, opts)
	if err != nil {
		return "", ErrSystem(err)
	}
	return id, nil
}

func (d *DockerUsecase) AttachExec(ctx context.Context, execID string, opts containertypes.ExecAttachOptions) (*dockerpkg.ExecSession, error) {
	session, err := d.docker.AttachExec(ctx, execID, opts)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return session, nil
}

func (d *DockerUsecase) ListImages(ctx context.Context, req *v1.ListDockerImagesRequest) (*v1.ListDockerImagesResponse, error) {
	result, err := d.docker.ListImages(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return result, nil
}

func (d *DockerUsecase) Image(ctx context.Context, id string) (*v1.DockerImageInfo, error) {
	info, err := d.docker.Image(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return info, nil
}

func (d *DockerUsecase) PullImage(ctx context.Context, req *v1.PullDockerImageRequest) (*v1.DockerTaskInfo, error) {
	return d.docker.PullImage(ctx, req), nil
}

func (d *DockerUsecase) UpdateImageTags(ctx context.Context, req *v1.UpdateDockerImageTagsRequest) (*v1.DockerImageInfo, error) {
	if err := d.docker.UpdateImageTags(ctx, req); err != nil {
		return nil, ErrSystem(err)
	}
	info, err := d.docker.Image(ctx, req.ImageId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return info, nil
}

func (d *DockerUsecase) TagImage(ctx context.Context, id, target string) error {
	if err := d.docker.TagImage(ctx, id, target); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) DeleteImage(ctx context.Context, id string, force, pruneChildren bool) error {
	if err := d.docker.DeleteImage(ctx, id, force, pruneChildren); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) ImageHistory(ctx context.Context, id string) ([]*v1.DockerImageHistoryItem, error) {
	items, err := d.docker.ImageHistory(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return items, nil
}

func (d *DockerUsecase) ListNetworks(ctx context.Context, req *v1.ListDockerNetworksRequest) (*v1.ListDockerNetworksResponse, error) {
	result, err := d.docker.ListNetworks(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return result, nil
}

func (d *DockerUsecase) Network(ctx context.Context, id string) (*v1.DockerNetworkInfo, error) {
	info, err := d.docker.Network(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return info, nil
}

func (d *DockerUsecase) CreateNetwork(ctx context.Context, req *v1.CreateDockerNetworkRequest) (string, error) {
	id, err := d.docker.CreateNetwork(ctx, req.GetOptions())
	if err != nil {
		return "", ErrSystem(err)
	}
	return id, nil
}

func (d *DockerUsecase) UpdateNetwork(ctx context.Context, req *v1.UpdateDockerNetworkRequest) (*v1.DockerNetworkInfo, *v1.DockerTaskInfo, error) {
	return nil, d.docker.UpdateNetwork(ctx, req), nil
}

func (d *DockerUsecase) RecreateNetwork(ctx context.Context, req *v1.RecreateDockerNetworkRequest) (*v1.DockerTaskInfo, error) {
	return d.docker.RecreateNetwork(ctx, req), nil
}

func (d *DockerUsecase) DeleteNetwork(ctx context.Context, id string) error {
	if err := d.docker.DeleteNetwork(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) ConnectNetwork(ctx context.Context, req *v1.ConnectDockerNetworkRequest) error {
	if err := d.docker.ConnectNetwork(ctx, req); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) DisconnectNetwork(ctx context.Context, req *v1.DisconnectDockerNetworkRequest) error {
	if err := d.docker.DisconnectNetwork(ctx, req); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) containerAction(ctx context.Context, fn func() error) error {
	if err := fn(); err != nil {
		return ErrSystem(err)
	}
	return nil
}
