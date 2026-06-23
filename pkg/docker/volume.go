package docker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	containertypes "github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"

	v1 "momoko/api/gen/v1"
)

func (m *Manager) ListVolumes(ctx context.Context, req *v1.ListDockerVolumesRequest) (*v1.ListDockerVolumesResponse, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(req.GetLabels())
	if req.GetName() != "" {
		filters.Add("name", req.GetName())
	}
	if req.GetDriver() != "" {
		filters.Add("driver", req.GetDriver())
	}
	resp, err := cli.VolumeList(ctx, volumetypes.ListOptions{Filters: filters})
	if err != nil {
		return nil, err
	}
	total := int64(len(resp.Volumes))
	items := pageSlice(resp.Volumes, req.GetPage(), req.GetPageSize())
	result := make([]*v1.DockerVolumeInfo, 0, len(items))
	for _, item := range items {
		result = append(result, toVolumeInfo(item))
	}
	return &v1.ListDockerVolumesResponse{Items: result, Total: total}, nil
}

func (m *Manager) Volume(ctx context.Context, name string) (*v1.DockerVolumeInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	data, err := cli.VolumeInspect(ctx, name)
	if err != nil {
		return nil, err
	}
	return toVolumeInfo(&data), nil
}

func (m *Manager) CreateVolume(ctx context.Context, opts *v1.DockerVolumeCreateOptions) (*v1.DockerVolumeInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	data, err := cli.VolumeCreate(ctx, volumetypes.CreateOptions{
		Name:       opts.GetName(),
		Driver:     opts.GetDriver(),
		Labels:     opts.GetLabels(),
		DriverOpts: opts.GetDriverOpts(),
	})
	if err != nil {
		return nil, err
	}
	return toVolumeInfo(&data), nil
}

func (m *Manager) UpdateVolume(ctx context.Context, req *v1.UpdateDockerVolumeRequest) *v1.DockerTaskInfo {
	create := normalizeVolumeCreateOptions(req.GetName(), req.GetOptions())
	if create.Labels == nil {
		create.Labels = req.GetLabels()
	}
	if create.DriverOpts == nil {
		create.DriverOpts = req.GetDriverOpts()
	}
	return m.RecreateVolume(ctx, &v1.RecreateDockerVolumeRequest{
		Name:    req.GetName(),
		Options: create,
		Force:   req.GetForce(),
	})
}

func (m *Manager) RecreateVolume(ctx context.Context, req *v1.RecreateDockerVolumeRequest) *v1.DockerTaskInfo {
	create := normalizeVolumeCreateOptions(req.GetName(), req.GetOptions())
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_VOLUME_RECREATE, "重建储存卷 "+req.GetName(), m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		if _, err := m.Volume(taskCtx, req.GetName()); err != nil {
			return "", err
		}
		emit(&v1.DockerTaskInfo{Message: "删除旧储存卷"})
		if err := m.DeleteVolume(taskCtx, req.GetName(), req.GetForce()); err != nil {
			return "", err
		}
		emit(&v1.DockerTaskInfo{Message: "创建新储存卷"})
		info, err := m.CreateVolume(taskCtx, create)
		if err != nil {
			return "", err
		}
		return info.Name, nil
	})
}

func (m *Manager) DeleteVolume(ctx context.Context, name string, force bool) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.VolumeRemove(ctx, name, force)
}

func (m *Manager) PruneVolumes(ctx context.Context) *v1.DockerTaskInfo {
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_VOLUME_PRUNE, "清理储存卷", m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		report, err := cli.VolumesPrune(taskCtx, filtersFromLabels(nil))
		if err != nil {
			return "", err
		}
		raw, _ := json.Marshal(report)
		emit(&v1.DockerTaskInfo{Message: string(raw)})
		return "", nil
	})
}

func (m *Manager) ExportVolume(ctx context.Context, req *v1.ExportDockerVolumeRequest) *v1.DockerTaskInfo {
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_VOLUME_EXPORT, "导出储存卷 "+req.GetName(), m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		if strings.TrimSpace(req.GetArchivePath()) == "" {
			return "", errors.New("导出路径不能为空")
		}
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		containerID, err := m.CreateContainer(taskCtx, &v1.DockerContainerCreateOptions{
			Name:  "momoko-volume-export-" + req.GetName(),
			Image: "busybox:latest",
			Cmd:   []string{"sh", "-c", "sleep 3600"},
			Mounts: []*v1.DockerMount{
				{Type: "volume", Source: req.GetName(), Target: "/volume"},
			},
		})
		if err != nil {
			return "", err
		}
		defer m.DeleteContainer(taskCtx, containerID, true, false)
		if err := m.StartContainer(taskCtx, containerID); err != nil {
			return "", err
		}
		reader, _, err := cli.CopyFromContainer(taskCtx, containerID, "/volume")
		if err != nil {
			return "", err
		}
		defer reader.Close()
		file, err := os.Create(req.GetArchivePath())
		if err != nil {
			return "", err
		}
		defer file.Close()
		emit(&v1.DockerTaskInfo{Message: "导出储存卷数据"})
		if _, err := io.Copy(file, reader); err != nil {
			return "", err
		}
		return req.GetArchivePath(), nil
	})
}

func (m *Manager) RestoreVolume(ctx context.Context, req *v1.RestoreDockerVolumeRequest) *v1.DockerTaskInfo {
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_VOLUME_RESTORE, "恢复储存卷 "+req.GetName(), m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		if req.GetName() == "" {
			return "", errors.New("储存卷名称不能为空")
		}
		if _, err := m.Volume(taskCtx, req.GetName()); err != nil {
			if _, createErr := m.CreateVolume(taskCtx, &v1.DockerVolumeCreateOptions{Name: req.GetName()}); createErr != nil {
				return "", createErr
			}
		}
		containerID, err := m.CreateContainer(taskCtx, &v1.DockerContainerCreateOptions{
			Name:  "momoko-volume-restore-" + req.GetName(),
			Image: "busybox:latest",
			Cmd:   []string{"sh", "-c", "sleep 3600"},
			Mounts: []*v1.DockerMount{
				{Type: "volume", Source: req.GetName(), Target: "/volume"},
			},
		})
		if err != nil {
			return "", err
		}
		defer m.DeleteContainer(taskCtx, containerID, true, false)
		if err := m.StartContainer(taskCtx, containerID); err != nil {
			return "", err
		}
		file, err := os.Open(req.GetArchivePath())
		if err != nil {
			return "", err
		}
		defer file.Close()
		emit(&v1.DockerTaskInfo{Message: "恢复储存卷数据"})
		if err := cli.CopyToContainer(taskCtx, containerID, "/", file, containertypes.CopyToContainerOptions{
			AllowOverwriteDirWithFile: true,
		}); err != nil {
			return "", err
		}
		return req.GetName(), nil
	})
}

func normalizeVolumeCreateOptions(name string, opts *v1.DockerVolumeCreateOptions) *v1.DockerVolumeCreateOptions {
	create := &v1.DockerVolumeCreateOptions{
		Name:       opts.GetName(),
		Driver:     opts.GetDriver(),
		Labels:     opts.GetLabels(),
		DriverOpts: opts.GetDriverOpts(),
	}
	if create.Name == "" {
		create.Name = name
	}
	return create
}
