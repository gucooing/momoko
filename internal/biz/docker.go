package biz

import (
	"context"
	"errors"
	"io"
	"strings"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/auth"
	dockerpkg "momoko/pkg/docker"
	"momoko/pkg/task"
)

const (
	DockerContainerLogsWSPath = dockerpkg.ContainerLogsWSPath
	DockerContainerExecWSPath = dockerpkg.ContainerExecWSPath
	DockerTaskWSPath          = dockerpkg.TaskWSPath
	DockerImageExportPath     = dockerpkg.ImageExportPath
	DockerImageImportPath     = dockerpkg.ImageImportPath
)

type DockerUsecase struct {
	config ConfigRepo
	docker *dockerpkg.Manager
}

func NewDockerUsecase(config ConfigRepo, tasks *task.Manager) (*DockerUsecase, error) {
	ctx := context.Background()
	cfg, err := config.DockerConfig(ctx)
	if err != nil {
		return nil, err
	}
	manager, err := dockerpkg.NewManager(cfg, tasks)
	if err != nil {
		// Docker 客户端初始化失败不应阻断服务启动，否则一份无法连接的持久化配置
		// （如启用了 TLS 却缺少证书路径）会导致整个面板无法启动且无界面可修复。
		// 此处以降级状态启动，运行时可通过 Docker 状态/配置页面查看并修正。
		log.Errorf("初始化 Docker 管理器失败，已降级启动: %v", err)
	}
	// 开机把上次中断（重启时仍在跑）的 Docker 任务标记为失败（拉取/重建不可重入）。
	_ = tasks.Resume(ctx, dockerpkg.DockerTaskTypes()...)
	return &DockerUsecase{config: config, docker: manager}, nil
}

// Config 返回给前端的配置：私钥内容不外传（只给 tls_key_meta），证书附带解析出的 CN / 有效期。
func (d *DockerUsecase) Config(ctx context.Context) (*v1.DockerConfigInfo, error) {
	cfg, err := d.config.DockerConfig(ctx)
	if err != nil {
		return nil, err
	}
	return publicDockerConfig(cfg), nil
}

func publicDockerConfig(cfg *v1.DockerConfigInfo) *v1.DockerConfigInfo {
	out := proto.Clone(cfg).(*v1.DockerConfigInfo)
	out.TlsCaMeta = dockerpkg.TLSCertMeta(out.TlsCa)
	out.TlsCertMeta = dockerpkg.TLSCertMeta(out.TlsCert)
	out.TlsKeyMeta = dockerpkg.TLSKeyMeta(out.TlsKey)
	out.TlsKey = ""
	out.TlsKeyClear = false
	// 仓库密码 / token 不回传；保存时留空即沿用已保存的
	for _, auth := range out.RegistryAuths {
		if auth != nil {
			auth.Password, auth.Token = "", ""
		}
	}
	return out
}

// mergeStoredSecrets 私钥留空表示沿用已保存的（除非显式清除）；仓库凭据密码留空时沿用同地址同用户名的已保存密码。
func (d *DockerUsecase) mergeStoredSecrets(ctx context.Context, cfg *v1.DockerConfigInfo) (*v1.DockerConfigInfo, error) {
	if cfg == nil {
		return nil, ErrSystem(errors.New("缺少 Docker 配置"))
	}
	stored, err := d.config.DockerConfig(ctx)
	if err != nil {
		return nil, err
	}
	cfg = proto.Clone(cfg).(*v1.DockerConfigInfo)
	if cfg.TlsKeyClear {
		cfg.TlsKey = ""
	} else if strings.TrimSpace(cfg.TlsKey) == "" {
		cfg.TlsKey = stored.TlsKey
	}
	cfg.TlsKeyClear = false
	cfg.TlsCaMeta, cfg.TlsCertMeta, cfg.TlsKeyMeta = nil, nil, nil
	for _, auth := range cfg.RegistryAuths {
		if auth == nil || auth.Password != "" || auth.Token != "" {
			continue
		}
		if old := findRegistryAuth(stored.RegistryAuths, auth.ServerAddress, auth.Username); old != nil {
			auth.Password, auth.Token = old.Password, old.Token
		}
	}
	return cfg, nil
}

func findRegistryAuth(list []*v1.DockerRegistryAuth, server, username string) *v1.DockerRegistryAuth {
	for _, item := range list {
		if item != nil && strings.EqualFold(strings.TrimSpace(item.ServerAddress), strings.TrimSpace(server)) && item.Username == username {
			return item
		}
	}
	return nil
}

func (d *DockerUsecase) UpdateConfig(ctx context.Context, req *v1.UpdateDockerConfigRequest) (*v1.DockerConfigInfo, error) {
	merged, err := d.mergeStoredSecrets(ctx, req.GetConfig())
	if err != nil {
		return nil, err
	}
	cfg, err := d.config.UpdateDockerConfig(ctx, merged)
	if err != nil {
		return nil, err
	}
	if err = d.docker.Reconfigure(cfg); err != nil {
		return nil, ErrSystem(err)
	}
	return publicDockerConfig(cfg), nil
}

func (d *DockerUsecase) TestConfig(ctx context.Context, req *v1.TestDockerConfigRequest) (*v1.DockerStatusResponse, error) {
	merged, err := d.mergeStoredSecrets(ctx, req.GetConfig())
	if err != nil {
		return nil, err
	}
	status, _ := d.docker.Test(ctx, merged)
	return status, nil
}

// TestRegistryAuth 通过引擎登录仓库验证凭据；已保存的同地址同用户名凭据同步记下验证结果。
func (d *DockerUsecase) TestRegistryAuth(ctx context.Context, auth *v1.DockerRegistryAuth) (*v1.DockerRegistryAuth, error) {
	if auth == nil {
		return nil, ErrSystem(errors.New("缺少仓库凭据"))
	}
	stored, err := d.config.DockerConfig(ctx)
	if err != nil {
		return nil, err
	}
	probe := proto.Clone(auth).(*v1.DockerRegistryAuth)
	saved := findRegistryAuth(stored.RegistryAuths, probe.ServerAddress, probe.Username)
	if probe.Password == "" && probe.Token == "" && saved != nil {
		probe.Password, probe.Token = saved.Password, saved.Token
	}
	result, err := d.docker.RegistryLogin(ctx, probe)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if saved != nil {
		saved.VerifiedAt, saved.Verified, saved.VerifyError = result.VerifiedAt, result.Verified, result.VerifyError
		if _, err := d.config.UpdateDockerConfig(context.WithoutCancel(ctx), stored); err != nil {
			return nil, ErrSystem(err)
		}
	}
	return result, nil
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

// CreateContainer 本地有镜像时直接创建返回容器 ID；否则返回「拉取镜像并创建」任务。
func (d *DockerUsecase) CreateContainer(ctx context.Context, req *v1.CreateDockerContainerRequest) (string, *v1.DockerTaskInfo, error) {
	id, task, err := d.docker.CreateContainer(ctx, req.GetOptions())
	if err != nil {
		return "", nil, ErrSystem(err)
	}
	return id, task, nil
}

// ContainersUsage 批量取运行中容器的 CPU / 内存占用。
func (d *DockerUsecase) ContainersUsage(ctx context.Context, ids []string) ([]*v1.DockerContainerUsage, error) {
	items, err := d.docker.ContainersUsage(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return items, nil
}

// ParseCompose 把 compose 文件转换成各服务的容器创建参数。
func (d *DockerUsecase) ParseCompose(content string) ([]*v1.DockerComposeService, error) {
	services, err := dockerpkg.ParseCompose(content)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return services, nil
}

// CancelTask 取消进行中的 Docker 任务，记录取消人。
func (d *DockerUsecase) CancelTask(ctx context.Context, id string) error {
	by := ""
	if uid := auth.GetUserIDFromContext(ctx); uid != nil {
		by = *uid
	}
	if err := d.docker.CancelTask(id, by); err != nil {
		return ErrSystem(err)
	}
	return nil
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

// SaveImages 导出镜像为 tar 流。
func (d *DockerUsecase) SaveImages(ctx context.Context, ids []string) (io.ReadCloser, error) {
	reader, err := d.docker.SaveImages(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return reader, nil
}

// LoadImages 从 tar 流导入镜像。
func (d *DockerUsecase) LoadImages(ctx context.Context, input io.Reader) ([]string, error) {
	loaded, err := d.docker.LoadImages(ctx, input)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return loaded, nil
}

func (d *DockerUsecase) ImageHistory(ctx context.Context, id string) ([]*v1.DockerImageHistoryItem, error) {
	items, err := d.docker.ImageHistory(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return items, nil
}

func (d *DockerUsecase) PruneImages(ctx context.Context, all bool) (*v1.PruneDockerImagesResponse, error) {
	result, err := d.docker.PruneImages(ctx, all)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return result, nil
}

func (d *DockerUsecase) SearchImages(ctx context.Context, term string, limit int32) ([]*v1.DockerImageSearchResult, error) {
	items, err := d.docker.SearchImages(ctx, term, limit)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return items, nil
}

func (d *DockerUsecase) PruneNetworks(ctx context.Context) ([]string, error) {
	deleted, err := d.docker.PruneNetworks(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return deleted, nil
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
