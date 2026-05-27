package biz

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	authpkg "momoko/pkg/auth"
	"momoko/pkg/common"
	"momoko/pkg/constant"
	dockerpkg "momoko/pkg/docker"
	"momoko/pkg/response"
	"momoko/pkg/secretbox"
)

const (
	DockerContainerLogsWSPath  = "/api/v1/docker/container/logs/ws"
	DockerContainerStatsWSPath = "/api/v1/docker/container/stats/ws"
	DockerContainerExecWSPath  = "/api/v1/docker/container/exec/ws"
	DockerTaskWSPath           = "/api/v1/docker/task/ws"
)

type DockerUsecase struct {
	sys    *SystemUsecase
	config ConfigRepo
	docker *dockerpkg.Manager
}

func NewDockerUsecase(sys *SystemUsecase, config ConfigRepo) (*DockerUsecase, error) {
	cfg, err := loadDockerConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	manager, err := dockerpkg.NewManager(cfg)
	if err != nil {
		return nil, err
	}
	return &DockerUsecase{sys: sys, config: config, docker: manager}, nil
}

func (d *DockerUsecase) Config(ctx context.Context) (*v1.DockerConfigInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	cfg, err := loadDockerConfig(ctx, d.config)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerConfigInfo(cfg), nil
}

func (d *DockerUsecase) UpdateConfig(ctx context.Context, req *v1.UpdateDockerConfigRequest) (*v1.DockerConfigInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerConfigEdit); err != nil {
		return nil, err
	}
	cfg := fromDockerConfigInfo(req.GetConfig())
	if err := validateDockerConfig(cfg); err != nil {
		return nil, err
	}
	storeCfg := cfg
	storeCfg.RegistryAuths = append([]dockerpkg.RegistryAuth(nil), cfg.RegistryAuths...)
	if err := encryptDockerRegistryAuths(storeCfg.RegistryAuths); err != nil {
		return nil, ErrSystem(err)
	}
	rawAuths, err := json.Marshal(storeCfg.RegistryAuths)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if err := d.config.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigDockerEnabled:               strconv.FormatBool(cfg.Enabled),
		common.ConfigDockerHost:                  cfg.Host,
		common.ConfigDockerTLSEnabled:            strconv.FormatBool(cfg.TLSEnabled),
		common.ConfigDockerTLSCAPath:             cfg.TLSCAPath,
		common.ConfigDockerTLSCertPath:           cfg.TLSCertPath,
		common.ConfigDockerTLSKeyPath:            cfg.TLSKeyPath,
		common.ConfigDockerAPIVersion:            cfg.APIVersion,
		common.ConfigDockerRequestTimeoutSeconds: strconv.FormatInt(int64(cfg.RequestTimeoutSeconds), 10),
		common.ConfigDockerDefaultPlatform:       cfg.DefaultPlatform,
		common.ConfigDockerDefaultLogTail:        strconv.FormatInt(int64(cfg.DefaultLogTail), 10),
		common.ConfigDockerTaskTimeoutSeconds:    strconv.FormatInt(int64(cfg.TaskTimeoutSeconds), 10),
		common.ConfigDockerRegistryAuths:         string(rawAuths),
	}); err != nil {
		return nil, ErrSystem(err)
	}
	if err := d.docker.Reconfigure(cfg); err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerConfigInfo(cfg), nil
}

func (d *DockerUsecase) TestConfig(ctx context.Context, req *v1.TestDockerConfigRequest) (*v1.DockerStatusResponse, error) {
	if err := d.sys.Check(ctx, constant.DockerConfigEdit); err != nil {
		return nil, err
	}
	cfg := fromDockerConfigInfo(req.GetConfig())
	if err := validateDockerConfig(cfg); err != nil {
		return nil, err
	}
	status, _ := d.docker.Test(ctx, cfg)
	return toDockerStatus(status), nil
}

func (d *DockerUsecase) Status(ctx context.Context) (*v1.DockerStatusResponse, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	return toDockerStatus(d.docker.Status(ctx)), nil
}

func (d *DockerUsecase) Tasks(ctx context.Context) ([]*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	items := d.docker.Tasks()
	result := make([]*v1.DockerTaskInfo, 0, len(items))
	for _, item := range items {
		result = append(result, toDockerTask(item))
	}
	return result, nil
}

func (d *DockerUsecase) Task(ctx context.Context, id string) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	task, err := d.docker.Task(id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerTask(task), nil
}

func (d *DockerUsecase) SubscribeTask(ctx context.Context, id string) (<-chan dockerpkg.TaskEvent, func(), error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, nil, err
	}
	return d.docker.SubscribeTask(id)
}

func (d *DockerUsecase) ListContainers(ctx context.Context, req *v1.ListDockerContainersRequest) (*v1.ListDockerContainersResponse, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	items, total, err := d.docker.ListContainers(ctx, dockerpkg.ContainerListOptions{
		All: req.All, Status: req.Status, Name: req.Name, Image: req.Image,
		Network: req.Network, Labels: req.Labels, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	result := make([]*v1.DockerContainerSummary, 0, len(items))
	for _, item := range items {
		result = append(result, toDockerContainerSummary(item))
	}
	return &v1.ListDockerContainersResponse{Items: result, Total: total}, nil
}

func (d *DockerUsecase) Container(ctx context.Context, id string) (*v1.DockerContainerInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	info, err := d.docker.Container(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerContainerInfo(info), nil
}

func (d *DockerUsecase) CreateContainer(ctx context.Context, req *v1.CreateDockerContainerRequest) (string, error) {
	if err := d.sys.Check(ctx, constant.DockerContainerManage); err != nil {
		return "", err
	}
	id, err := d.docker.CreateContainer(ctx, fromDockerContainerCreateOptions(req.GetOptions()))
	if err != nil {
		return "", ErrSystem(err)
	}
	return id, nil
}

func (d *DockerUsecase) UpdateContainer(ctx context.Context, req *v1.UpdateDockerContainerRequest) (*v1.DockerContainerInfo, *v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerContainerManage); err != nil {
		return nil, nil, err
	}
	if req.Recreate {
		task := d.docker.RecreateContainer(ctx, dockerpkg.RecreateContainerOptions{
			ID:            req.Id,
			Create:        fromDockerContainerCreateOptions(req.GetOptions()),
			Force:         req.Force,
			RemoveVolumes: req.RemoveVolumes,
		})
		return nil, toDockerTask(task), nil
	}
	err := d.docker.UpdateContainer(ctx, req.Id, dockerpkg.UpdateContainerOptions{
		Name: req.Name, RestartPolicy: req.RestartPolicy, Memory: req.Memory,
		MemorySwap: req.MemorySwap, CPUShares: req.CpuShares, CPUQuota: req.CpuQuota,
		CPUPeriod: req.CpuPeriod, NanoCPUs: req.NanoCpus,
	})
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	info, err := d.docker.Container(ctx, req.Id)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	return toDockerContainerInfo(info), nil, nil
}

func (d *DockerUsecase) RecreateContainer(ctx context.Context, req *v1.RecreateDockerContainerRequest) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerContainerManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.RecreateContainer(ctx, dockerpkg.RecreateContainerOptions{
		ID: req.Id, Create: fromDockerContainerCreateOptions(req.GetOptions()),
		Force: req.Force, RemoveVolumes: req.RemoveVolumes,
	})), nil
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

func (d *DockerUsecase) ContainerLogs(ctx context.Context, req *v1.ContainerLogsRequest) (string, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return "", err
	}
	reader, err := d.docker.ContainerLogs(ctx, req.Id, dockerpkg.LogOptions{
		Stdout: req.Stdout, Stderr: req.Stderr, Since: req.Since, Until: req.Until,
		Timestamps: req.Timestamps, Tail: req.Tail, Details: req.Details,
	})
	if err != nil {
		return "", ErrSystem(err)
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", ErrSystem(err)
	}
	return string(raw), nil
}

func (d *DockerUsecase) ContainerLogStream(ctx context.Context, req *v1.ContainerLogsRequest) (io.ReadCloser, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	return d.docker.ContainerLogs(ctx, req.Id, dockerpkg.LogOptions{
		Stdout: req.Stdout, Stderr: req.Stderr, Since: req.Since, Until: req.Until,
		Timestamps: req.Timestamps, Follow: true, Tail: req.Tail, Details: req.Details,
	})
}

func (d *DockerUsecase) ContainerStats(ctx context.Context, id string) (string, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return "", err
	}
	raw, err := d.docker.ContainerStatsOnce(ctx, id)
	if err != nil {
		return "", ErrSystem(err)
	}
	return string(raw), nil
}

func (d *DockerUsecase) ContainerStatsStream(ctx context.Context, id string) (io.ReadCloser, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	return d.docker.ContainerStats(ctx, id, true)
}

func (d *DockerUsecase) CreateExec(ctx context.Context, req *v1.CreateContainerExecRequest) (string, error) {
	if err := d.sys.Check(ctx, constant.DockerContainerManage); err != nil {
		return "", err
	}
	id, err := d.docker.CreateExec(ctx, req.ContainerId, dockerpkg.ExecOptions{
		Cmd: req.Cmd, Env: req.Env, User: req.User, WorkingDir: req.WorkingDir,
		Tty: req.Tty, AttachStdin: true, AttachStdout: true, AttachStderr: true,
	})
	if err != nil {
		return "", ErrSystem(err)
	}
	return id, nil
}

func (d *DockerUsecase) AttachExec(ctx context.Context, execID string, tty bool) (*dockerpkg.ExecSession, error) {
	if err := d.sys.Check(ctx, constant.DockerContainerManage); err != nil {
		return nil, err
	}
	return d.docker.AttachExec(ctx, execID, tty)
}

func (d *DockerUsecase) ListImages(ctx context.Context, req *v1.ListDockerImagesRequest) (*v1.ListDockerImagesResponse, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	items, total, err := d.docker.ListImages(ctx, dockerpkg.ImageListOptions{
		All: req.All, Dangling: req.Dangling, Keyword: req.Keyword,
		Labels: req.Labels, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	result := make([]*v1.DockerImageSummary, 0, len(items))
	for _, item := range items {
		result = append(result, toDockerImageSummary(item))
	}
	return &v1.ListDockerImagesResponse{Items: result, Total: total}, nil
}

func (d *DockerUsecase) Image(ctx context.Context, id string) (*v1.DockerImageInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	info, err := d.docker.Image(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerImageInfo(info), nil
}

func (d *DockerUsecase) PullImage(ctx context.Context, req *v1.PullDockerImageRequest) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerImageManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.PullImage(ctx, dockerpkg.PullImageOptions{
		Reference: req.Reference, Platform: req.Platform, RegistryAuth: fromDockerRegistryAuth(req.RegistryAuth),
	})), nil
}

func (d *DockerUsecase) BuildImage(ctx context.Context, req *v1.BuildDockerImageRequest) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerImageManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.BuildImage(ctx, dockerpkg.BuildImageOptions{
		ContextPath: req.ContextPath, Dockerfile: req.Dockerfile, Tags: req.Tags,
		BuildArgs: req.BuildArgs, Labels: req.Labels, Platform: req.Platform,
		NoCache: req.NoCache, PullParent: req.PullParent, Remove: req.Remove, ForceRemove: req.ForceRemove,
	})), nil
}

func (d *DockerUsecase) UpdateImageTags(ctx context.Context, req *v1.UpdateDockerImageTagsRequest) (*v1.DockerImageInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerImageManage); err != nil {
		return nil, err
	}
	if err := d.docker.UpdateImageTags(ctx, dockerpkg.UpdateImageTagsOptions{
		ImageID: req.ImageId, AddTags: req.AddTags, DeleteTags: req.DeleteTags, ForceDelete: req.ForceDelete,
	}); err != nil {
		return nil, ErrSystem(err)
	}
	info, err := d.docker.Image(ctx, req.ImageId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerImageInfo(info), nil
}

func (d *DockerUsecase) TagImage(ctx context.Context, id, target string) error {
	if err := d.sys.Check(ctx, constant.DockerImageManage); err != nil {
		return err
	}
	if err := d.docker.TagImage(ctx, id, target); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) DeleteImage(ctx context.Context, id string, force, pruneChildren bool) error {
	if err := d.sys.Check(ctx, constant.DockerImageManage); err != nil {
		return err
	}
	if err := d.docker.DeleteImage(ctx, id, force, pruneChildren); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) PruneImages(ctx context.Context, danglingOnly bool) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerImageManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.PruneImages(ctx, danglingOnly)), nil
}

func (d *DockerUsecase) ImageHistory(ctx context.Context, id string) ([]*v1.DockerImageHistoryItem, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	items, err := d.docker.ImageHistory(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	result := make([]*v1.DockerImageHistoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, toDockerImageHistory(item))
	}
	return result, nil
}

func (d *DockerUsecase) ListNetworks(ctx context.Context, req *v1.ListDockerNetworksRequest) (*v1.ListDockerNetworksResponse, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	items, total, err := d.docker.ListNetworks(ctx, dockerpkg.NetworkListOptions{
		Name: req.Name, Driver: req.Driver, Scope: req.Scope, Labels: req.Labels, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	result := make([]*v1.DockerNetworkInfo, 0, len(items))
	for _, item := range items {
		result = append(result, toDockerNetworkInfo(item))
	}
	return &v1.ListDockerNetworksResponse{Items: result, Total: total}, nil
}

func (d *DockerUsecase) Network(ctx context.Context, id string) (*v1.DockerNetworkInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	info, err := d.docker.Network(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerNetworkInfo(*info), nil
}

func (d *DockerUsecase) CreateNetwork(ctx context.Context, req *v1.CreateDockerNetworkRequest) (string, error) {
	if err := d.sys.Check(ctx, constant.DockerNetworkManage); err != nil {
		return "", err
	}
	id, err := d.docker.CreateNetwork(ctx, fromDockerNetworkCreateOptions(req.GetOptions()))
	if err != nil {
		return "", ErrSystem(err)
	}
	return id, nil
}

func (d *DockerUsecase) UpdateNetwork(ctx context.Context, req *v1.UpdateDockerNetworkRequest) (*v1.DockerNetworkInfo, *v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerNetworkManage); err != nil {
		return nil, nil, err
	}
	create := fromDockerNetworkCreateOptions(req.GetOptions())
	if req.Options == nil {
		info, err := d.docker.Network(ctx, req.Id)
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		create = networkCreateOptionsFromInfo(*info)
		create.Labels = req.Labels
	}
	task := d.docker.UpdateNetwork(ctx, dockerpkg.UpdateNetworkOptions{
		ID:     req.Id,
		Create: create,
		Force:  req.Force,
	})
	if task == nil {
		info, err := d.docker.Network(ctx, req.Id)
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		return toDockerNetworkInfo(*info), nil, nil
	}
	return nil, toDockerTask(task), nil
}

func (d *DockerUsecase) RecreateNetwork(ctx context.Context, req *v1.RecreateDockerNetworkRequest) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerNetworkManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.RecreateNetwork(ctx, dockerpkg.RecreateNetworkOptions{
		ID: req.Id, Create: fromDockerNetworkCreateOptions(req.GetOptions()), Force: req.Force,
	})), nil
}

func (d *DockerUsecase) DeleteNetwork(ctx context.Context, id string) error {
	if err := d.sys.Check(ctx, constant.DockerNetworkManage); err != nil {
		return err
	}
	if err := d.docker.DeleteNetwork(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) ConnectNetwork(ctx context.Context, req *v1.ConnectDockerNetworkRequest) error {
	if err := d.sys.Check(ctx, constant.DockerNetworkManage); err != nil {
		return err
	}
	if err := d.docker.ConnectNetwork(ctx, dockerpkg.ConnectNetworkOptions{
		NetworkID: req.NetworkId, ContainerID: req.ContainerId,
		Aliases: req.Aliases, IPv4Address: req.Ipv4Address, IPv6Address: req.Ipv6Address,
	}); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) DisconnectNetwork(ctx context.Context, req *v1.DisconnectDockerNetworkRequest) error {
	if err := d.sys.Check(ctx, constant.DockerNetworkManage); err != nil {
		return err
	}
	if err := d.docker.DisconnectNetwork(ctx, req.NetworkId, req.ContainerId, req.Force); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) PruneNetworks(ctx context.Context) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerNetworkManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.PruneNetworks(ctx)), nil
}

func (d *DockerUsecase) ListVolumes(ctx context.Context, req *v1.ListDockerVolumesRequest) (*v1.ListDockerVolumesResponse, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	items, total, err := d.docker.ListVolumes(ctx, dockerpkg.VolumeListOptions{
		Name: req.Name, Driver: req.Driver, Labels: req.Labels, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	result := make([]*v1.DockerVolumeInfo, 0, len(items))
	for _, item := range items {
		result = append(result, toDockerVolumeInfo(item))
	}
	return &v1.ListDockerVolumesResponse{Items: result, Total: total}, nil
}

func (d *DockerUsecase) Volume(ctx context.Context, name string) (*v1.DockerVolumeInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerView); err != nil {
		return nil, err
	}
	info, err := d.docker.Volume(ctx, name)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerVolumeInfo(*info), nil
}

func (d *DockerUsecase) CreateVolume(ctx context.Context, req *v1.CreateDockerVolumeRequest) (*v1.DockerVolumeInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerVolumeManage); err != nil {
		return nil, err
	}
	info, err := d.docker.CreateVolume(ctx, dockerpkg.CreateVolumeOptions{
		Name: req.GetOptions().GetName(), Driver: req.GetOptions().GetDriver(),
		Labels: req.GetOptions().GetLabels(), DriverOpts: req.GetOptions().GetDriverOpts(),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toDockerVolumeInfo(*info), nil
}

func (d *DockerUsecase) UpdateVolume(ctx context.Context, req *v1.UpdateDockerVolumeRequest) (*v1.DockerVolumeInfo, *v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerVolumeManage); err != nil {
		return nil, nil, err
	}
	create := dockerpkg.CreateVolumeOptions{
		Name:       req.Name,
		Labels:     req.Labels,
		DriverOpts: req.DriverOpts,
	}
	if req.Options != nil {
		create = dockerpkg.CreateVolumeOptions{
			Name:       req.GetOptions().GetName(),
			Driver:     req.GetOptions().GetDriver(),
			Labels:     req.GetOptions().GetLabels(),
			DriverOpts: req.GetOptions().GetDriverOpts(),
		}
	}
	task := d.docker.UpdateVolume(ctx, dockerpkg.UpdateVolumeOptions{
		Name:       req.Name,
		Labels:     req.Labels,
		DriverOpts: req.DriverOpts,
		Create:     create,
		Force:      req.Force,
	})
	return nil, toDockerTask(task), nil
}

func (d *DockerUsecase) RecreateVolume(ctx context.Context, req *v1.RecreateDockerVolumeRequest) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerVolumeManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.RecreateVolume(ctx, dockerpkg.RecreateVolumeOptions{
		Name: req.Name,
		Create: dockerpkg.CreateVolumeOptions{
			Name: req.GetOptions().GetName(), Driver: req.GetOptions().GetDriver(),
			Labels: req.GetOptions().GetLabels(), DriverOpts: req.GetOptions().GetDriverOpts(),
		},
		Force: req.Force,
	})), nil
}

func (d *DockerUsecase) DeleteVolume(ctx context.Context, name string, force bool) error {
	if err := d.sys.Check(ctx, constant.DockerVolumeManage); err != nil {
		return err
	}
	if err := d.docker.DeleteVolume(ctx, name, force); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (d *DockerUsecase) PruneVolumes(ctx context.Context) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerVolumeManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.PruneVolumes(ctx)), nil
}

func (d *DockerUsecase) ExportVolume(ctx context.Context, req *v1.ExportDockerVolumeRequest) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerVolumeManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.ExportVolume(ctx, dockerpkg.VolumeArchiveOptions{
		VolumeName: req.Name, ArchivePath: req.ArchivePath,
	})), nil
}

func (d *DockerUsecase) RestoreVolume(ctx context.Context, req *v1.RestoreDockerVolumeRequest) (*v1.DockerTaskInfo, error) {
	if err := d.sys.Check(ctx, constant.DockerVolumeManage); err != nil {
		return nil, err
	}
	return toDockerTask(d.docker.RestoreVolume(ctx, dockerpkg.VolumeArchiveOptions{
		VolumeName: req.Name, ArchivePath: req.ArchivePath,
	})), nil
}

func (d *DockerUsecase) containerAction(ctx context.Context, fn func() error) error {
	if err := d.sys.Check(ctx, constant.DockerContainerManage); err != nil {
		return err
	}
	if err := fn(); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func loadDockerConfig(ctx context.Context, repo ConfigRepo) (dockerpkg.Config, error) {
	enabled, err := boolConfig(ctx, repo, common.ConfigDockerEnabled)
	if err != nil {
		return dockerpkg.Config{}, err
	}
	tlsEnabled, err := boolConfig(ctx, repo, common.ConfigDockerTLSEnabled)
	if err != nil {
		return dockerpkg.Config{}, err
	}
	requestTimeout, err := int32Config(ctx, repo, common.ConfigDockerRequestTimeoutSeconds)
	if err != nil {
		return dockerpkg.Config{}, err
	}
	logTail, err := int32Config(ctx, repo, common.ConfigDockerDefaultLogTail)
	if err != nil {
		return dockerpkg.Config{}, err
	}
	taskTimeout, err := int32Config(ctx, repo, common.ConfigDockerTaskTimeoutSeconds)
	if err != nil {
		return dockerpkg.Config{}, err
	}
	authsRaw, err := repo.Get(ctx, common.ConfigDockerRegistryAuths)
	if err != nil {
		return dockerpkg.Config{}, err
	}
	var auths []dockerpkg.RegistryAuth
	if strings.TrimSpace(authsRaw) != "" {
		if err := json.Unmarshal([]byte(authsRaw), &auths); err != nil {
			return dockerpkg.Config{}, err
		}
		if err := decryptDockerRegistryAuths(auths); err != nil {
			return dockerpkg.Config{}, err
		}
	}
	host, _ := repo.Get(ctx, common.ConfigDockerHost)
	caPath, _ := repo.Get(ctx, common.ConfigDockerTLSCAPath)
	certPath, _ := repo.Get(ctx, common.ConfigDockerTLSCertPath)
	keyPath, _ := repo.Get(ctx, common.ConfigDockerTLSKeyPath)
	apiVersion, _ := repo.Get(ctx, common.ConfigDockerAPIVersion)
	defaultPlatform, _ := repo.Get(ctx, common.ConfigDockerDefaultPlatform)
	return dockerpkg.Config{
		Enabled: enabled, Host: host, TLSEnabled: tlsEnabled, TLSCAPath: caPath,
		TLSCertPath: certPath, TLSKeyPath: keyPath, APIVersion: apiVersion,
		RequestTimeoutSeconds: requestTimeout, DefaultPlatform: defaultPlatform,
		DefaultLogTail: logTail, TaskTimeoutSeconds: taskTimeout, RegistryAuths: auths,
	}, nil
}

func boolConfig(ctx context.Context, repo ConfigRepo, key common.ConfigKey) (bool, error) {
	value, err := repo.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(value)
}

func int32Config(ctx context.Context, repo ConfigRepo, key common.ConfigKey) (int32, error) {
	value, err := repo.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	number, err := strconv.ParseInt(value, 10, 32)
	return int32(number), err
}

func validateDockerConfig(cfg dockerpkg.Config) error {
	if cfg.TLSEnabled && (cfg.TLSCAPath == "" || cfg.TLSCertPath == "" || cfg.TLSKeyPath == "") {
		return response.BadRequest(400, "Docker TLS证书路径不能为空")
	}
	return nil
}

func toDockerConfigInfo(cfg dockerpkg.Config) *v1.DockerConfigInfo {
	auths := make([]*v1.DockerRegistryAuth, 0, len(cfg.RegistryAuths))
	for _, item := range cfg.RegistryAuths {
		auths = append(auths, &v1.DockerRegistryAuth{
			ServerAddress: item.ServerAddress, Username: item.Username,
			Password: item.Password, Token: item.Token,
		})
	}
	return &v1.DockerConfigInfo{
		Enabled: cfg.Enabled, Host: cfg.Host, TlsEnabled: cfg.TLSEnabled,
		TlsCaPath: cfg.TLSCAPath, TlsCertPath: cfg.TLSCertPath, TlsKeyPath: cfg.TLSKeyPath,
		ApiVersion: cfg.APIVersion, RequestTimeoutSeconds: cfg.RequestTimeoutSeconds,
		DefaultPlatform: cfg.DefaultPlatform, DefaultLogTail: cfg.DefaultLogTail,
		TaskTimeoutSeconds: cfg.TaskTimeoutSeconds, RegistryAuths: auths,
	}
}

func fromDockerConfigInfo(info *v1.DockerConfigInfo) dockerpkg.Config {
	if info == nil {
		return dockerpkg.Config{RequestTimeoutSeconds: 30, DefaultLogTail: 200, TaskTimeoutSeconds: 1800}
	}
	auths := make([]dockerpkg.RegistryAuth, 0, len(info.RegistryAuths))
	for _, item := range info.RegistryAuths {
		auths = append(auths, dockerpkg.RegistryAuth{
			ServerAddress: item.ServerAddress, Username: item.Username,
			Password: item.Password, Token: item.Token,
		})
	}
	cfg := dockerpkg.Config{
		Enabled: info.Enabled, Host: strings.TrimSpace(info.Host), TLSEnabled: info.TlsEnabled,
		TLSCAPath: strings.TrimSpace(info.TlsCaPath), TLSCertPath: strings.TrimSpace(info.TlsCertPath),
		TLSKeyPath: strings.TrimSpace(info.TlsKeyPath), APIVersion: strings.TrimSpace(info.ApiVersion),
		RequestTimeoutSeconds: info.RequestTimeoutSeconds, DefaultPlatform: strings.TrimSpace(info.DefaultPlatform),
		DefaultLogTail: info.DefaultLogTail, TaskTimeoutSeconds: info.TaskTimeoutSeconds, RegistryAuths: auths,
	}
	if cfg.RequestTimeoutSeconds <= 0 {
		cfg.RequestTimeoutSeconds = 30
	}
	if cfg.DefaultLogTail <= 0 {
		cfg.DefaultLogTail = 200
	}
	if cfg.TaskTimeoutSeconds <= 0 {
		cfg.TaskTimeoutSeconds = 1800
	}
	return cfg
}

func encryptDockerRegistryAuths(auths []dockerpkg.RegistryAuth) error {
	box := secretbox.New(authpkg.AuthSecretKey)
	for i := range auths {
		password, err := box.Encrypt(auths[i].Password)
		if err != nil {
			return err
		}
		token, err := box.Encrypt(auths[i].Token)
		if err != nil {
			return err
		}
		auths[i].Password = password
		auths[i].Token = token
	}
	return nil
}

func decryptDockerRegistryAuths(auths []dockerpkg.RegistryAuth) error {
	box := secretbox.New(authpkg.AuthSecretKey)
	for i := range auths {
		if strings.HasPrefix(auths[i].Password, "v1:") {
			password, err := box.Decrypt(auths[i].Password)
			if err != nil {
				return err
			}
			auths[i].Password = password
		}
		if strings.HasPrefix(auths[i].Token, "v1:") {
			token, err := box.Decrypt(auths[i].Token)
			if err != nil {
				return err
			}
			auths[i].Token = token
		}
	}
	return nil
}

func toDockerStatus(status dockerpkg.Status) *v1.DockerStatusResponse {
	return &v1.DockerStatusResponse{
		Enabled: status.Enabled, Connected: status.Connected, Error: status.Error,
		Info: toDockerEngineInfo(status.Info), Version: toDockerEngineVersion(status.Version),
	}
}

func toDockerEngineInfo(info *dockerpkg.EngineInfo) *v1.DockerEngineInfo {
	if info == nil {
		return nil
	}
	return &v1.DockerEngineInfo{
		Id: info.ID, Name: info.Name, ServerVersion: info.ServerVersion,
		OperatingSystem: info.OperatingSystem, OsType: info.OSType, Architecture: info.Architecture,
		DockerRootDir: info.DockerRootDir, Containers: info.Containers,
		ContainersRunning: info.ContainersRunning, ContainersPaused: info.ContainersPaused,
		ContainersStopped: info.ContainersStopped, Images: info.Images, Driver: info.Driver,
		CgroupDriver: info.CgroupDriver, CgroupVersion: info.CgroupVersion,
		MemoryTotal: info.MemoryTotal, Cpus: info.CPUs, Labels: info.Labels,
	}
}

func toDockerEngineVersion(version *dockerpkg.EngineVersion) *v1.DockerEngineVersion {
	if version == nil {
		return nil
	}
	return &v1.DockerEngineVersion{
		Version: version.Version, ApiVersion: version.APIVersion, MinApiVersion: version.MinAPIVersion,
		GitCommit: version.GitCommit, GoVersion: version.GoVersion, Os: version.OS,
		Arch: version.Arch, KernelVersion: version.KernelVersion, BuildTime: version.BuildTime,
	}
}

func toDockerTask(task *dockerpkg.Task) *v1.DockerTaskInfo {
	if task == nil {
		return nil
	}
	events := make([]*v1.DockerTaskEvent, 0, len(task.Events))
	for _, item := range task.Events {
		events = append(events, &v1.DockerTaskEvent{
			Time: timestamppb.New(item.Time), Status: item.Status, Progress: item.Progress,
			Id: item.ID, Message: item.Message, Error: item.Error,
		})
	}
	info := &v1.DockerTaskInfo{
		Id: task.ID, Type: task.Type, Status: string(task.Status), Progress: task.Progress,
		Message: task.Message, Error: task.Error, ResultPath: task.ResultPath,
		StartTime: timestamppb.New(task.StartTime), Events: events,
	}
	if task.EndTime != nil {
		info.EndTime = timestamppb.New(*task.EndTime)
	}
	return info
}

func fromDockerContainerCreateOptions(req *v1.DockerContainerCreateOptions) dockerpkg.CreateContainerOptions {
	if req == nil {
		return dockerpkg.CreateContainerOptions{}
	}
	mounts := make([]dockerpkg.Mount, 0, len(req.Mounts))
	for _, item := range req.Mounts {
		mounts = append(mounts, dockerpkg.Mount{Type: item.Type, Source: item.Source, Target: item.Target, ReadOnly: item.ReadOnly})
	}
	ports := make([]dockerpkg.PortBinding, 0, len(req.Ports))
	for _, item := range req.Ports {
		ports = append(ports, dockerpkg.PortBinding{ContainerPort: item.ContainerPort, HostIP: item.HostIp, HostPort: item.HostPort})
	}
	return dockerpkg.CreateContainerOptions{
		Name: req.Name, Image: req.Image, Hostname: req.Hostname, User: req.User, Env: req.Env,
		Cmd: req.Cmd, Entrypoint: req.Entrypoint, WorkingDir: req.WorkingDir, Labels: req.Labels,
		Tty: req.Tty, OpenStdin: req.OpenStdin, Network: req.Network, RestartPolicy: req.RestartPolicy,
		AutoRemove: req.AutoRemove, Privileged: req.Privileged, Ports: ports, Mounts: mounts,
		Memory: req.Memory, MemorySwap: req.MemorySwap, CPUShares: req.CpuShares,
		CPUQuota: req.CpuQuota, CPUPeriod: req.CpuPeriod, NanoCPUs: req.NanoCpus, Platform: req.Platform,
	}
}

func toDockerContainerSummary(item dockerpkg.ContainerSummary) *v1.DockerContainerSummary {
	ports := make([]*v1.DockerPort, 0, len(item.Ports))
	for _, port := range item.Ports {
		ports = append(ports, &v1.DockerPort{Ip: port.IP, PrivatePort: port.PrivatePort, PublicPort: port.PublicPort, Type: port.Type})
	}
	return &v1.DockerContainerSummary{
		Id: item.ID, Names: item.Names, Image: item.Image, ImageId: item.ImageID,
		Command: item.Command, Created: timestamppb.New(item.Created), State: item.State,
		Status: item.Status, Labels: item.Labels, Ports: ports, Mounts: toDockerMountPoints(item.Mounts),
		NetworkMode: item.NetworkMode, Networks: item.Networks,
	}
}

func toDockerContainerInfo(item *dockerpkg.ContainerInfo) *v1.DockerContainerInfo {
	if item == nil {
		return nil
	}
	return &v1.DockerContainerInfo{
		Id: item.ID, Name: item.Name, Image: item.Image, ImageId: item.ImageID, Path: item.Path,
		Args: item.Args, Created: item.Created,
		State: &v1.DockerContainerState{
			Status: item.State.Status, Running: item.State.Running, Paused: item.State.Paused,
			Restarting: item.State.Restarting, OomKilled: item.State.OOMKilled, Dead: item.State.Dead,
			Pid: item.State.Pid, ExitCode: item.State.ExitCode, Error: item.State.Error,
			StartedAt: item.State.StartedAt, FinishedAt: item.State.FinishedAt,
		},
		Config: mustStruct(item.Config), HostConfig: mustStruct(item.HostConfig),
		Network: mustStruct(item.Network), Mounts: toDockerMountPoints(item.Mounts),
		RestartCount: item.RestartCount, Platform: item.Platform, Driver: item.Driver, LogPath: item.LogPath,
	}
}

func toDockerMountPoints(items []dockerpkg.MountPoint) []*v1.DockerMountPoint {
	result := make([]*v1.DockerMountPoint, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.DockerMountPoint{
			Type: item.Type, Name: item.Name, Source: item.Source, Destination: item.Destination,
			Driver: item.Driver, Mode: item.Mode, Rw: item.RW, Propagation: item.Propagation,
		})
	}
	return result
}

func toDockerImageSummary(item dockerpkg.ImageSummary) *v1.DockerImageSummary {
	return &v1.DockerImageSummary{
		Id: item.ID, RepoTags: item.RepoTags, RepoDigests: item.RepoDigests, ParentId: item.ParentID,
		Created: timestamppb.New(item.Created), Size: item.Size, SharedSize: item.SharedSize,
		Containers: item.Containers, Labels: item.Labels,
	}
}

func toDockerImageInfo(item *dockerpkg.ImageInfo) *v1.DockerImageInfo {
	if item == nil {
		return nil
	}
	return &v1.DockerImageInfo{
		Id: item.ID, RepoTags: item.RepoTags, RepoDigests: item.RepoDigests, Parent: item.Parent,
		Created: item.Created, Author: item.Author, Architecture: item.Architecture, Os: item.OS,
		Size: item.Size, VirtualSize: item.VirtualSize, Labels: item.Labels, Layers: item.Layers,
	}
}

func fromDockerRegistryAuth(req *v1.DockerRegistryAuth) *dockerpkg.RegistryAuth {
	if req == nil {
		return nil
	}
	return &dockerpkg.RegistryAuth{
		ServerAddress: req.ServerAddress, Username: req.Username, Password: req.Password, Token: req.Token,
	}
}

func toDockerImageHistory(item dockerpkg.ImageHistoryItem) *v1.DockerImageHistoryItem {
	return &v1.DockerImageHistoryItem{
		Id: item.ID, Created: timestamppb.New(item.Created), CreatedBy: item.CreatedBy,
		Tags: item.Tags, Size: item.Size, Comment: item.Comment,
	}
}

func fromDockerNetworkCreateOptions(req *v1.DockerNetworkCreateOptions) dockerpkg.CreateNetworkOptions {
	if req == nil {
		return dockerpkg.CreateNetworkOptions{}
	}
	configs := make([]dockerpkg.IPAMConfig, 0, len(req.GetIpam().GetConfig()))
	for _, item := range req.GetIpam().GetConfig() {
		configs = append(configs, dockerpkg.IPAMConfig{
			Subnet: item.Subnet, IPRange: item.IpRange, Gateway: item.Gateway, AuxAddress: item.AuxAddress,
		})
	}
	return dockerpkg.CreateNetworkOptions{
		Name: req.Name, Driver: req.Driver, Scope: req.Scope, EnableIPv4: req.EnableIpv4,
		EnableIPv6: req.EnableIpv6, Internal: req.Internal, Attachable: req.Attachable, Ingress: req.Ingress,
		IPAM:    dockerpkg.IPAM{Driver: req.GetIpam().GetDriver(), Options: req.GetIpam().GetOptions(), Config: configs},
		Options: req.Options, Labels: req.Labels,
	}
}

func networkCreateOptionsFromInfo(info dockerpkg.NetworkInfo) dockerpkg.CreateNetworkOptions {
	enableIPv4 := info.EnableIPv4
	enableIPv6 := info.EnableIPv6
	return dockerpkg.CreateNetworkOptions{
		Name:       info.Name,
		Driver:     info.Driver,
		Scope:      info.Scope,
		EnableIPv4: &enableIPv4,
		EnableIPv6: &enableIPv6,
		Internal:   info.Internal,
		Attachable: info.Attachable,
		Ingress:    info.Ingress,
		IPAM:       info.IPAM,
		Options:    info.Options,
		Labels:     info.Labels,
	}
}

func toDockerNetworkInfo(item dockerpkg.NetworkInfo) *v1.DockerNetworkInfo {
	containers := make(map[string]*v1.DockerNetworkContainer, len(item.Containers))
	for id, data := range item.Containers {
		containers[id] = &v1.DockerNetworkContainer{
			Name: data.Name, EndpointId: data.EndpointID, MacAddress: data.MacAddress,
			Ipv4Address: data.IPv4Address, Ipv6Address: data.IPv6Address,
		}
	}
	return &v1.DockerNetworkInfo{
		Id: item.ID, Name: item.Name, Created: item.Created, Scope: item.Scope, Driver: item.Driver,
		EnableIpv4: item.EnableIPv4, EnableIpv6: item.EnableIPv6, Internal: item.Internal,
		Attachable: item.Attachable, Ingress: item.Ingress, Ipam: toDockerIPAM(item.IPAM),
		Containers: containers, Options: item.Options, Labels: item.Labels,
	}
}

func toDockerIPAM(item dockerpkg.IPAM) *v1.DockerIPAM {
	configs := make([]*v1.DockerIPAMConfig, 0, len(item.Config))
	for _, cfg := range item.Config {
		configs = append(configs, &v1.DockerIPAMConfig{
			Subnet: cfg.Subnet, IpRange: cfg.IPRange, Gateway: cfg.Gateway, AuxAddress: cfg.AuxAddress,
		})
	}
	return &v1.DockerIPAM{Driver: item.Driver, Options: item.Options, Config: configs}
}

func toDockerVolumeInfo(item dockerpkg.VolumeInfo) *v1.DockerVolumeInfo {
	return &v1.DockerVolumeInfo{
		Name: item.Name, Driver: item.Driver, Mountpoint: item.Mountpoint, CreatedAt: item.CreatedAt,
		Status: item.Status, Labels: item.Labels, Scope: item.Scope, Options: item.Options,
		UsageSize: item.UsageSize, RefCount: item.RefCount,
	}
}

func mustStruct(data any) *structpb.Struct {
	raw, err := json.Marshal(data)
	if err != nil {
		return &structpb.Struct{}
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return &structpb.Struct{}
	}
	result, err := structpb.NewStruct(value)
	if err != nil {
		return &structpb.Struct{}
	}
	return result
}
