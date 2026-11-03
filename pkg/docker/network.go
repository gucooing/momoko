package docker

import (
	"context"
	"sort"
	"strconv"
	"strings"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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
	sort.SliceStable(items, func(i, j int) bool {
		bi, bj := isBuiltinNetwork(items[i].Name), isBuiltinNetwork(items[j].Name)
		if bi != bj {
			return bi
		}
		return items[i].Name < items[j].Name
	})

	// 容器列表：补齐网络里各容器的状态、别名，并把已停止（inspect 里不出现）的容器也列进来
	containers, _ := cli.ContainerList(ctx, containertypes.ListOptions{All: true})

	stats := &v1.DockerNetworkListStats{}
	all := make([]*v1.DockerNetworkInfo, 0, len(items))
	for _, item := range items {
		// NetworkList 不返回已连接容器，逐个 inspect 以获得准确的容器信息
		info := toNetworkInfo(item)
		if detail, derr := cli.NetworkInspect(ctx, item.ID, networktypes.InspectOptions{}); derr == nil {
			info = toNetworkInfo(detail)
		}
		mergeNetworkContainers(info, containers)
		stats.Total++
		if info.Builtin {
			stats.Builtin++
		} else {
			stats.Custom++
		}
		for _, c := range info.Containers {
			if c.Ipv4Address != "" || c.Ipv6Address != "" {
				stats.AllocatedIps++
			}
		}
		all = append(all, info)
	}

	keyword := strings.ToLower(strings.TrimSpace(req.GetKeyword()))
	filtered := make([]*v1.DockerNetworkInfo, 0, len(all))
	for _, info := range all {
		switch req.GetKind() {
		case "builtin":
			if !info.Builtin {
				continue
			}
		case "custom":
			if info.Builtin {
				continue
			}
		}
		if keyword != "" && !networkMatches(info, keyword) {
			continue
		}
		filtered = append(filtered, info)
	}
	total := int64(len(filtered))
	filtered = pageSlice(filtered, req.GetPage(), req.GetPageSize())
	return &v1.ListDockerNetworksResponse{Items: filtered, Total: total, Stats: stats}, nil
}

// isBuiltinNetwork Docker 自带、不能删除或重建的网络。
func isBuiltinNetwork(name string) bool {
	switch name {
	case "bridge", "host", "none", "ingress", "docker_gwbridge":
		return true
	}
	return false
}

func mergeNetworkContainers(info *v1.DockerNetworkInfo, containers []containertypes.Summary) {
	if info.Containers == nil {
		info.Containers = map[string]*v1.DockerNetworkContainer{}
	}
	for _, c := range containers {
		if c.NetworkSettings == nil {
			continue
		}
		var endpoint *networktypes.EndpointSettings
		for netName, ep := range c.NetworkSettings.Networks {
			if ep == nil {
				continue
			}
			if netName == info.Name || ep.NetworkID == info.Id {
				endpoint = ep
				break
			}
		}
		if endpoint == nil {
			continue
		}
		entry, ok := info.Containers[c.ID]
		if !ok {
			name := c.ID
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			entry = &v1.DockerNetworkContainer{
				Name:       name,
				EndpointId: endpoint.EndpointID,
				MacAddress: endpoint.MacAddress,
			}
			if endpoint.IPAddress != "" {
				entry.Ipv4Address = endpoint.IPAddress
				if endpoint.IPPrefixLen > 0 {
					entry.Ipv4Address = endpoint.IPAddress + "/" + strconv.Itoa(endpoint.IPPrefixLen)
				}
			}
			info.Containers[c.ID] = entry
		}
		entry.State = string(c.State)
		entry.Aliases = append([]string(nil), endpoint.Aliases...)
	}
}

func networkMatches(info *v1.DockerNetworkInfo, keyword string) bool {
	if strings.Contains(strings.ToLower(info.Name), keyword) || strings.HasPrefix(strings.ToLower(info.Id), keyword) {
		return true
	}
	if info.Ipam != nil {
		for _, cfg := range info.Ipam.Config {
			if strings.Contains(cfg.Subnet, keyword) || strings.Contains(cfg.Gateway, keyword) {
				return true
			}
		}
	}
	for _, c := range info.Containers {
		if strings.Contains(strings.ToLower(c.Name), keyword) {
			return true
		}
	}
	return false
}

// PruneNetworks 清理没有任何容器使用的自定义网络。
func (m *Manager) PruneNetworks(ctx context.Context) ([]string, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	report, err := cli.NetworksPrune(ctx, filters.NewArgs())
	if err != nil {
		return nil, err
	}
	deleted := report.NetworksDeleted
	if deleted == nil {
		deleted = []string{}
	}
	return deleted, nil
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
