package docker

import (
	"context"
	"encoding/json"

	networktypes "github.com/docker/docker/api/types/network"
)

func (m *Manager) ListNetworks(ctx context.Context, opts NetworkListOptions) ([]NetworkInfo, int64, error) {
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
	if opts.Scope != "" {
		filters.Add("scope", opts.Scope)
	}
	items, err := cli.NetworkList(ctx, networktypes.ListOptions{Filters: filters})
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(items))
	items = pageSlice(items, opts.Page, opts.PageSize)
	result := make([]NetworkInfo, 0, len(items))
	for _, item := range items {
		result = append(result, toNetworkInfo(item))
	}
	return result, total, nil
}

func (m *Manager) Network(ctx context.Context, id string) (*NetworkInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	data, err := cli.NetworkInspect(ctx, id, networktypes.InspectOptions{})
	if err != nil {
		return nil, err
	}
	info := toNetworkInfo(data)
	return &info, nil
}

func (m *Manager) CreateNetwork(ctx context.Context, opts CreateNetworkOptions) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err := cli.NetworkCreate(ctx, opts.Name, networktypes.CreateOptions{
		Driver:     opts.Driver,
		Scope:      opts.Scope,
		EnableIPv4: opts.EnableIPv4,
		EnableIPv6: opts.EnableIPv6,
		IPAM:       toDockerIPAM(opts.IPAM),
		Internal:   opts.Internal,
		Attachable: opts.Attachable,
		Ingress:    opts.Ingress,
		Options:    opts.Options,
		Labels:     opts.Labels,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (m *Manager) UpdateNetwork(ctx context.Context, opts UpdateNetworkOptions) *Task {
	return m.RecreateNetwork(ctx, RecreateNetworkOptions{
		ID:     opts.ID,
		Create: opts.Create,
		Force:  opts.Force,
	})
}

func (m *Manager) RecreateNetwork(ctx context.Context, opts RecreateNetworkOptions) *Task {
	return m.tasks.Start(ctx, "network_recreate", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		oldInfo, err := m.Network(taskCtx, opts.ID)
		if err != nil {
			return "", err
		}
		if len(oldInfo.Containers) > 0 && !opts.Force {
			return "", ErrNetworkInUse
		}
		emit(TaskEvent{Message: "删除旧网络"})
		if err := m.DeleteNetwork(taskCtx, opts.ID); err != nil {
			return "", err
		}
		emit(TaskEvent{Message: "创建新网络"})
		return m.CreateNetwork(taskCtx, opts.Create)
	})
}

func (m *Manager) DeleteNetwork(ctx context.Context, id string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.NetworkRemove(ctx, id)
}

func (m *Manager) ConnectNetwork(ctx context.Context, opts ConnectNetworkOptions) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.NetworkConnect(ctx, opts.NetworkID, opts.ContainerID, toNetworkEndpoint(opts))
}

func (m *Manager) DisconnectNetwork(ctx context.Context, networkID, containerID string, force bool) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.NetworkDisconnect(ctx, networkID, containerID, force)
}

func (m *Manager) PruneNetworks(ctx context.Context) *Task {
	return m.tasks.Start(ctx, "network_prune", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		report, err := cli.NetworksPrune(taskCtx, filtersFromLabels(nil))
		if err != nil {
			return "", err
		}
		raw, _ := json.Marshal(report)
		emit(TaskEvent{Message: string(raw)})
		return "", nil
	})
}

var ErrNetworkInUse = &resourceInUseError{message: "网络仍有容器连接"}

type resourceInUseError struct {
	message string
}

func (e *resourceInUseError) Error() string {
	return e.message
}
