package docker

import (
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	mounttypes "github.com/docker/docker/api/types/mount"
	networktypes "github.com/docker/docker/api/types/network"
	systemtypes "github.com/docker/docker/api/types/system"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

func toEngineInfo(info systemtypes.Info) *v1.DockerEngineInfo {
	return &v1.DockerEngineInfo{
		Id:                info.ID,
		Name:              info.Name,
		ServerVersion:     info.ServerVersion,
		OperatingSystem:   info.OperatingSystem,
		OsType:            info.OSType,
		Architecture:      info.Architecture,
		DockerRootDir:     info.DockerRootDir,
		Containers:        int32(info.Containers),
		ContainersRunning: int32(info.ContainersRunning),
		ContainersPaused:  int32(info.ContainersPaused),
		ContainersStopped: int32(info.ContainersStopped),
		Images:            int32(info.Images),
		Driver:            info.Driver,
		CgroupDriver:      info.CgroupDriver,
		CgroupVersion:     info.CgroupVersion,
		MemoryTotal:       info.MemTotal,
		Cpus:              int32(info.NCPU),
		Labels:            append([]string(nil), info.Labels...),
	}
}

func toEngineVersion(version types.Version) *v1.DockerEngineVersion {
	return &v1.DockerEngineVersion{
		Version:       version.Version,
		ApiVersion:    version.APIVersion,
		MinApiVersion: version.MinAPIVersion,
		GitCommit:     version.GitCommit,
		GoVersion:     version.GoVersion,
		Os:            version.Os,
		Arch:          version.Arch,
		KernelVersion: version.KernelVersion,
		BuildTime:     version.BuildTime,
	}
}

func toContainerSummary(data containertypes.Summary) *v1.DockerContainerSummary {
	ports := make([]*v1.DockerPort, 0, len(data.Ports))
	for _, item := range data.Ports {
		ports = append(ports, &v1.DockerPort{
			Ip:          item.IP,
			PrivatePort: uint32(item.PrivatePort),
			PublicPort:  uint32(item.PublicPort),
			Type:        item.Type,
		})
	}
	networks := []string{}
	endpoints := []*v1.DockerContainerNetworkEndpoint{}
	if data.NetworkSettings != nil {
		for name, settings := range data.NetworkSettings.Networks {
			networks = append(networks, name)
			endpoint := &v1.DockerContainerNetworkEndpoint{Name: name}
			if settings != nil {
				endpoint.IpAddress = settings.IPAddress
			}
			endpoints = append(endpoints, endpoint)
		}
	}
	sort.Strings(networks)
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].Name < endpoints[j].Name
	})
	return &v1.DockerContainerSummary{
		Id:               data.ID,
		Names:            append([]string(nil), data.Names...),
		Image:            data.Image,
		ImageId:          data.ImageID,
		Command:          data.Command,
		Created:          timestamppb.New(time.Unix(data.Created, 0)),
		State:            string(data.State),
		Status:           data.Status,
		Labels:           cloneStringMap(data.Labels),
		Ports:            ports,
		Mounts:           toMountPoints(data.Mounts),
		NetworkMode:      data.HostConfig.NetworkMode,
		Networks:         networks,
		NetworkEndpoints: endpoints,
	}
}

func toContainerInfo(data containertypes.InspectResponse) *v1.DockerContainerInfo {
	info := &v1.DockerContainerInfo{
		Id:           data.ID,
		Name:         strings.TrimPrefix(data.Name, "/"),
		Image:        data.Image,
		Path:         data.Path,
		Args:         append([]string(nil), data.Args...),
		Created:      data.Created,
		Mounts:       toMountPoints(data.Mounts),
		RestartCount: int32(data.RestartCount),
		Platform:     data.Platform,
		Driver:       data.Driver,
		LogPath:      data.LogPath,
		LogsWsPath:   ContainerLogsWSPath,
		ExecWsPath:   ContainerExecWSPath,
	}
	if data.Config != nil {
		info.ImageId = data.Config.Image
		info.Config = &v1.DockerContainerConfig{
			Image: data.Config.Image,
			Env:   append([]string(nil), data.Config.Env...),
		}
	}
	if data.State != nil {
		info.State = &v1.DockerContainerState{
			Status:     data.State.Status,
			Running:    data.State.Running,
			Paused:     data.State.Paused,
			Restarting: data.State.Restarting,
			OomKilled:  data.State.OOMKilled,
			Dead:       data.State.Dead,
			Pid:        int32(data.State.Pid),
			ExitCode:   int32(data.State.ExitCode),
			Error:      data.State.Error,
			StartedAt:  data.State.StartedAt,
			FinishedAt: data.State.FinishedAt,
		}
	}
	if data.HostConfig != nil {
		info.HostConfig = &v1.DockerContainerHostConfig{
			NetworkMode:   string(data.HostConfig.NetworkMode),
			RestartPolicy: string(data.HostConfig.RestartPolicy.Name),
			AutoRemove:    data.HostConfig.AutoRemove,
			Privileged:    data.HostConfig.Privileged,
			PortBindings:  portMapToBindings(data.HostConfig.PortBindings),
			Mounts:        fromDockerMounts(data.HostConfig.Mounts),
			Memory:        data.HostConfig.Resources.Memory,
			CpuQuota:      data.HostConfig.Resources.CPUQuota,
			CpuPeriod:     data.HostConfig.Resources.CPUPeriod,
			NanoCpus:      data.HostConfig.Resources.NanoCPUs,
		}
	}
	if data.NetworkSettings != nil {
		info.Network = &v1.DockerContainerNetworkSettings{
			Networks: make(map[string]*v1.DockerEndpointSettings, len(data.NetworkSettings.Networks)),
		}
		for name, settings := range data.NetworkSettings.Networks {
			if settings == nil {
				continue
			}
			info.Network.Networks[name] = &v1.DockerEndpointSettings{
				IpAddress:  settings.IPAddress,
				Gateway:    settings.Gateway,
				MacAddress: settings.MacAddress,
			}
		}
	}
	return info
}

func toImageSummary(data imagetypes.Summary) *v1.DockerImageSummary {
	return &v1.DockerImageSummary{
		Id:          data.ID,
		RepoTags:    append([]string(nil), data.RepoTags...),
		RepoDigests: append([]string(nil), data.RepoDigests...),
		ParentId:    data.ParentID,
		Created:     timestamppb.New(time.Unix(data.Created, 0)),
		Size:        data.Size,
		SharedSize:  data.SharedSize,
		Containers:  data.Containers,
		Labels:      cloneStringMap(data.Labels),
	}
}

func toImageInfo(data imagetypes.InspectResponse) *v1.DockerImageInfo {
	labels := map[string]string(nil)
	if data.Config != nil {
		labels = cloneStringMap(data.Config.Labels)
	}
	return &v1.DockerImageInfo{
		Id:           data.ID,
		RepoTags:     append([]string(nil), data.RepoTags...),
		RepoDigests:  append([]string(nil), data.RepoDigests...),
		Parent:       data.Parent,
		Created:      data.Created,
		Author:       data.Author,
		Architecture: data.Architecture,
		Os:           data.Os,
		Size:         data.Size,
		VirtualSize:  data.VirtualSize,
		Labels:       labels,
		Layers:       append([]string(nil), data.RootFS.Layers...),
	}
}

func toImageHistory(data imagetypes.HistoryResponseItem) *v1.DockerImageHistoryItem {
	return &v1.DockerImageHistoryItem{
		Id:        data.ID,
		Created:   timestamppb.New(time.Unix(data.Created, 0)),
		CreatedBy: data.CreatedBy,
		Tags:      append([]string(nil), data.Tags...),
		Size:      data.Size,
		Comment:   data.Comment,
	}
}

func toNetworkInfo(data networktypes.Inspect) *v1.DockerNetworkInfo {
	containers := make(map[string]*v1.DockerNetworkContainer, len(data.Containers))
	for id, item := range data.Containers {
		containers[id] = &v1.DockerNetworkContainer{
			Name:        item.Name,
			EndpointId:  item.EndpointID,
			MacAddress:  item.MacAddress,
			Ipv4Address: item.IPv4Address,
			Ipv6Address: item.IPv6Address,
		}
	}
	return &v1.DockerNetworkInfo{
		Id:         data.ID,
		Name:       data.Name,
		Created:    data.Created.Format(time.RFC3339Nano),
		Scope:      data.Scope,
		Driver:     data.Driver,
		EnableIpv4: data.EnableIPv4,
		EnableIpv6: data.EnableIPv6,
		Internal:   data.Internal,
		Attachable: data.Attachable,
		Ingress:    data.Ingress,
		Ipam:       fromDockerIPAM(data.IPAM),
		Containers: containers,
		Options:    cloneStringMap(data.Options),
		Labels:     cloneStringMap(data.Labels),
	}
}

func toVolumeInfo(data *volumetypes.Volume) *v1.DockerVolumeInfo {
	if data == nil {
		return nil
	}
	info := &v1.DockerVolumeInfo{
		Name:       data.Name,
		Driver:     data.Driver,
		Mountpoint: data.Mountpoint,
		CreatedAt:  data.CreatedAt,
		Status:     stringMapStatus(data.Status),
		Labels:     cloneStringMap(data.Labels),
		Scope:      data.Scope,
		Options:    cloneStringMap(data.Options),
	}
	if data.UsageData != nil {
		info.UsageSize = data.UsageData.Size
		info.RefCount = data.UsageData.RefCount
	}
	return info
}

func toMountPoints(items []containertypes.MountPoint) []*v1.DockerMountPoint {
	result := make([]*v1.DockerMountPoint, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.DockerMountPoint{
			Type:        string(item.Type),
			Name:        item.Name,
			Source:      item.Source,
			Destination: item.Destination,
			Driver:      item.Driver,
			Mode:        item.Mode,
			Rw:          item.RW,
			Propagation: string(item.Propagation),
		})
	}
	return result
}

func fromDockerMounts(items []mounttypes.Mount) []*v1.DockerMount {
	result := make([]*v1.DockerMount, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.DockerMount{
			Type:     string(item.Type),
			Source:   item.Source,
			Target:   item.Target,
			ReadOnly: item.ReadOnly,
		})
	}
	return result
}

func portMapToBindings(portMap nat.PortMap) []*v1.DockerPortBinding {
	result := []*v1.DockerPortBinding{}
	for port, bindings := range portMap {
		for _, binding := range bindings {
			result = append(result, &v1.DockerPortBinding{
				ContainerPort: string(port),
				HostIp:        binding.HostIP,
				HostPort:      binding.HostPort,
			})
		}
	}
	return result
}

func fromDockerIPAM(data networktypes.IPAM) *v1.DockerIPAM {
	configs := make([]*v1.DockerIPAMConfig, 0, len(data.Config))
	for _, item := range data.Config {
		configs = append(configs, &v1.DockerIPAMConfig{
			Subnet:     item.Subnet,
			IpRange:    item.IPRange,
			Gateway:    item.Gateway,
			AuxAddress: cloneStringMap(item.AuxAddress),
		})
	}
	return &v1.DockerIPAM{
		Driver:  data.Driver,
		Options: cloneStringMap(data.Options),
		Config:  configs,
	}
}

func networkCreateOptionsFromInfo(info *v1.DockerNetworkInfo) *v1.DockerNetworkCreateOptions {
	if info == nil {
		return nil
	}
	enableIPv4 := info.EnableIpv4
	enableIPv6 := info.EnableIpv6
	return &v1.DockerNetworkCreateOptions{
		Name:       info.Name,
		Driver:     info.Driver,
		Scope:      info.Scope,
		EnableIpv4: &enableIPv4,
		EnableIpv6: &enableIPv6,
		Internal:   info.Internal,
		Attachable: info.Attachable,
		Ingress:    info.Ingress,
		Ipam:       info.Ipam,
		Options:    cloneStringMap(info.Options),
		Labels:     cloneStringMap(info.Labels),
	}
}

func cloneStringMap(data map[string]string) map[string]string {
	if len(data) == 0 {
		return nil
	}
	result := make(map[string]string, len(data))
	for k, v := range data {
		result[k] = v
	}
	return result
}
