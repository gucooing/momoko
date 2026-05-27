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
)

func (m *Manager) ListVolumes(ctx context.Context, opts VolumeListOptions) ([]VolumeInfo, int64, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(opts.Labels)
	if opts.Name != "" {
		filters.Add("name", opts.Name)
	}
	if opts.Driver != "" {
		filters.Add("driver", opts.Driver)
	}
	resp, err := cli.VolumeList(ctx, volumetypes.ListOptions{Filters: filters})
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(resp.Volumes))
	items := pageSlice(resp.Volumes, opts.Page, opts.PageSize)
	result := make([]VolumeInfo, 0, len(items))
	for _, item := range items {
		result = append(result, toVolumeInfo(item))
	}
	return result, total, nil
}

func (m *Manager) Volume(ctx context.Context, name string) (*VolumeInfo, error) {
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
	info := toVolumeInfo(&data)
	return &info, nil
}

func (m *Manager) CreateVolume(ctx context.Context, opts CreateVolumeOptions) (*VolumeInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	data, err := cli.VolumeCreate(ctx, volumetypes.CreateOptions{
		Name:       opts.Name,
		Driver:     opts.Driver,
		Labels:     opts.Labels,
		DriverOpts: opts.DriverOpts,
	})
	if err != nil {
		return nil, err
	}
	info := toVolumeInfo(&data)
	return &info, nil
}

func (m *Manager) UpdateVolume(ctx context.Context, opts UpdateVolumeOptions) *Task {
	create := opts.Create
	if create.Name == "" {
		create.Name = opts.Name
	}
	if create.Labels == nil {
		create.Labels = opts.Labels
	}
	if create.DriverOpts == nil {
		create.DriverOpts = opts.DriverOpts
	}
	return m.RecreateVolume(ctx, RecreateVolumeOptions{
		Name:   opts.Name,
		Create: create,
		Force:  opts.Force,
	})
}

func (m *Manager) RecreateVolume(ctx context.Context, opts RecreateVolumeOptions) *Task {
	return m.tasks.Start(ctx, "volume_recreate", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		if _, err := m.Volume(taskCtx, opts.Name); err != nil {
			return "", err
		}
		emit(TaskEvent{Message: "删除旧储存卷"})
		if err := m.DeleteVolume(taskCtx, opts.Name, opts.Force); err != nil {
			return "", err
		}
		emit(TaskEvent{Message: "创建新储存卷"})
		info, err := m.CreateVolume(taskCtx, opts.Create)
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

func (m *Manager) PruneVolumes(ctx context.Context) *Task {
	return m.tasks.Start(ctx, "volume_prune", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		report, err := cli.VolumesPrune(taskCtx, filtersFromLabels(nil))
		if err != nil {
			return "", err
		}
		raw, _ := json.Marshal(report)
		emit(TaskEvent{Message: string(raw)})
		return "", nil
	})
}

func (m *Manager) ExportVolume(ctx context.Context, opts VolumeArchiveOptions) *Task {
	return m.tasks.Start(ctx, "volume_export", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		if strings.TrimSpace(opts.ArchivePath) == "" {
			return "", errors.New("导出路径不能为空")
		}
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		containerID, err := m.CreateContainer(taskCtx, CreateContainerOptions{
			Name:  "momoko-volume-export-" + opts.VolumeName,
			Image: "busybox:latest",
			Cmd:   []string{"sh", "-c", "sleep 3600"},
			Mounts: []Mount{
				{Type: "volume", Source: opts.VolumeName, Target: "/volume"},
			},
		})
		if err != nil {
			return "", err
		}
		defer m.DeleteContainer(context.Background(), containerID, true, false)
		if err := m.StartContainer(taskCtx, containerID); err != nil {
			return "", err
		}
		reader, _, err := cli.CopyFromContainer(taskCtx, containerID, "/volume")
		if err != nil {
			return "", err
		}
		defer reader.Close()
		file, err := os.Create(opts.ArchivePath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		emit(TaskEvent{Message: "导出储存卷数据"})
		if _, err := io.Copy(file, reader); err != nil {
			return "", err
		}
		return opts.ArchivePath, nil
	})
}

func (m *Manager) RestoreVolume(ctx context.Context, opts VolumeArchiveOptions) *Task {
	return m.tasks.Start(ctx, "volume_restore", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		if opts.VolumeName == "" {
			return "", errors.New("储存卷名称不能为空")
		}
		if _, err := m.Volume(taskCtx, opts.VolumeName); err != nil {
			if _, createErr := m.CreateVolume(taskCtx, CreateVolumeOptions{Name: opts.VolumeName}); createErr != nil {
				return "", createErr
			}
		}
		containerID, err := m.CreateContainer(taskCtx, CreateContainerOptions{
			Name:  "momoko-volume-restore-" + opts.VolumeName,
			Image: "busybox:latest",
			Cmd:   []string{"sh", "-c", "sleep 3600"},
			Mounts: []Mount{
				{Type: "volume", Source: opts.VolumeName, Target: "/volume"},
			},
		})
		if err != nil {
			return "", err
		}
		defer m.DeleteContainer(context.Background(), containerID, true, false)
		if err := m.StartContainer(taskCtx, containerID); err != nil {
			return "", err
		}
		file, err := os.Open(opts.ArchivePath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		emit(TaskEvent{Message: "恢复储存卷数据"})
		if err := cli.CopyToContainer(taskCtx, containerID, "/", file, containertypes.CopyToContainerOptions{
			AllowOverwriteDirWithFile: true,
		}); err != nil {
			return "", err
		}
		return opts.VolumeName, nil
	})
}
