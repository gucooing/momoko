package docker

import (
	"context"

	networktypes "github.com/docker/docker/api/types/network"

	v1 "momoko/api/gen/v1"
)

func (m *Manager) ListNetworks(ctx context.Context, req *v1.ListDockerNetworksRequest) (*v1.ListDockerNetworksResponse, error) {
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
	if req.GetScope() != "" {
		filters.Add("scope", req.GetScope())
	}
	items, err := cli.NetworkList(ctx, networktypes.ListOptions{Filters: filters})
	if err != nil {
		return nil, err
	}
	total := int64(len(items))
	items = pageSlice(items, req.GetPage(), req.GetPageSize())
	result := make([]*v1.DockerNetworkInfo, 0, len(items))
	for _, item := range items {
		// NetworkList 不返回已连接容器，逐个 inspect 以获得准确的容器数量
		if detail, derr := cli.NetworkInspect(ctx, item.ID, networktypes.InspectOptions{}); derr == nil {
			result = append(result, toNetworkInfo(detail))
			continue
		}
		result = append(result, toNetworkInfo(item))
	}
	return &v1.ListDockerNetworksResponse{Items: result, Total: total}, nil
}

func (m *Manager) Network(ctx context.Context, id string) (*v1.DockerNetworkInfo, error) {
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
	return toNetworkInfo(data), nil
}

func (m *Manager) CreateNetwork(ctx context.Context, opts *v1.DockerNetworkCreateOptions) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	var enableIPv4, enableIPv6 *bool
	if opts != nil {
		enableIPv4 = opts.EnableIpv4
		enableIPv6 = opts.EnableIpv6
	}
	resp, err := cli.NetworkCreate(ctx, opts.GetName(), networktypes.CreateOptions{
		Driver:     opts.GetDriver(),
		Scope:      opts.GetScope(),
		EnableIPv4: enableIPv4,
		EnableIPv6: enableIPv6,
		IPAM:       toDockerIPAM(opts.GetIpam()),
		Internal:   opts.GetInternal(),
		Attachable: opts.GetAttachable(),
		Ingress:    opts.GetIngress(),
		Options:    opts.GetOptions(),
		Labels:     opts.GetLabels(),
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (m *Manager) UpdateNetwork(ctx context.Context, req *v1.UpdateDockerNetworkRequest) *v1.DockerTaskInfo {
	titleTarget := req.GetOptions().GetName()
	if titleTarget == "" {
		titleTarget = req.GetId()
	}
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_NETWORK_RECREATE, "重建网络 "+titleTarget, m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		oldInfo, err := m.Network(taskCtx, req.GetId())
		if err != nil {
			return "", err
		}
		if len(oldInfo.GetContainers()) > 0 && !req.GetForce() {
			return "", ErrNetworkInUse
		}
		create := req.GetOptions()
		if create == nil {
			create = networkCreateOptionsFromInfo(oldInfo)
			create.Labels = cloneStringMap(req.GetLabels())
		}
		emit(&v1.DockerTaskInfo{Message: "删除旧网络"})
		if err := m.DeleteNetwork(taskCtx, req.GetId()); err != nil {
			return "", err
		}
		emit(&v1.DockerTaskInfo{Message: "创建新网络"})
		return m.CreateNetwork(taskCtx, create)
	})
}

func (m *Manager) RecreateNetwork(ctx context.Context, req *v1.RecreateDockerNetworkRequest) *v1.DockerTaskInfo {
	titleTarget := req.GetOptions().GetName()
	if titleTarget == "" {
		titleTarget = req.GetId()
	}
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_NETWORK_RECREATE, "重建网络 "+titleTarget, m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		oldInfo, err := m.Network(taskCtx, req.GetId())
		if err != nil {
			return "", err
		}
		if len(oldInfo.GetContainers()) > 0 && !req.GetForce() {
			return "", ErrNetworkInUse
		}
		create := req.GetOptions()
		if create == nil {
			create = networkCreateOptionsFromInfo(oldInfo)
		}
		emit(&v1.DockerTaskInfo{Message: "删除旧网络"})
		if err := m.DeleteNetwork(taskCtx, req.GetId()); err != nil {
			return "", err
		}
		emit(&v1.DockerTaskInfo{Message: "创建新网络"})
		return m.CreateNetwork(taskCtx, create)
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

func (m *Manager) ConnectNetwork(ctx context.Context, req *v1.ConnectDockerNetworkRequest) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.NetworkConnect(ctx, req.GetNetworkId(), req.GetContainerId(), toNetworkEndpoint(req))
}

func (m *Manager) DisconnectNetwork(ctx context.Context, req *v1.DisconnectDockerNetworkRequest) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	return cli.NetworkDisconnect(ctx, req.GetNetworkId(), req.GetContainerId(), req.GetForce())
}

var ErrNetworkInUse = &resourceInUseError{message: "网络仍有容器连接"}

type resourceInUseError struct {
	message string
}

func (e *resourceInUseError) Error() string {
	return e.message
}
