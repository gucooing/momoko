package docker

import (
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
)

func toEngineInfo(info systemtypes.Info) *EngineInfo {
	return &EngineInfo{
		ID:                info.ID,
		Name:              info.Name,
		ServerVersion:     info.ServerVersion,
		OperatingSystem:   info.OperatingSystem,
		OSType:            info.OSType,
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
		CPUs:              int32(info.NCPU),
		Labels:            append([]string(nil), info.Labels...),
	}
}

func toEngineVersion(version types.Version) *EngineVersion {
	return &EngineVersion{
		Version:       version.Version,
		APIVersion:    version.APIVersion,
		MinAPIVersion: version.MinAPIVersion,
		GitCommit:     version.GitCommit,
		GoVersion:     version.GoVersion,
		OS:            version.Os,
		Arch:          version.Arch,
		KernelVersion: version.KernelVersion,
		BuildTime:     version.BuildTime,
	}
}

func toContainerSummary(data containertypes.Summary) ContainerSummary {
	ports := make([]Port, 0, len(data.Ports))
	for _, item := range data.Ports {
		ports = append(ports, Port{
			IP:          item.IP,
			PrivatePort: uint32(item.PrivatePort),
			PublicPort:  uint32(item.PublicPort),
			Type:        item.Type,
		})
	}
	networks := []string{}
	if data.NetworkSettings != nil {
		for name := range data.NetworkSettings.Networks {
			networks = append(networks, name)
		}
	}
	return ContainerSummary{
		ID:          data.ID,
		Names:       append([]string(nil), data.Names...),
		Image:       data.Image,
		ImageID:     data.ImageID,
		Command:     data.Command,
		Created:     time.Unix(data.Created, 0),
		State:       string(data.State),
		Status:      data.Status,
		Labels:      cloneStringMap(data.Labels),
		Ports:       ports,
		Mounts:      toMountPoints(data.Mounts),
		NetworkMode: data.HostConfig.NetworkMode,
		Networks:    networks,
	}
}

func toContainerInfo(data containertypes.InspectResponse) ContainerInfo {
	info := ContainerInfo{
		ID:           data.ID,
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
	}
	if data.Config != nil {
		info.ImageID = data.Config.Image
		info.Config = ContainerConfig{
			Hostname:     data.Config.Hostname,
			User:         data.Config.User,
			Env:          append([]string(nil), data.Config.Env...),
			Cmd:          []string(data.Config.Cmd),
			Image:        data.Config.Image,
			WorkingDir:   data.Config.WorkingDir,
			Entrypoint:   []string(data.Config.Entrypoint),
			Labels:       cloneStringMap(data.Config.Labels),
			ExposedPorts: portSetToStrings(data.Config.ExposedPorts),
			Tty:          data.Config.Tty,
			OpenStdin:    data.Config.OpenStdin,
		}
	}
	if data.State != nil {
		info.State = ContainerState{
			Status:     data.State.Status,
			Running:    data.State.Running,
			Paused:     data.State.Paused,
			Restarting: data.State.Restarting,
			OOMKilled:  data.State.OOMKilled,
			Dead:       data.State.Dead,
			Pid:        int32(data.State.Pid),
			ExitCode:   int32(data.State.ExitCode),
			Error:      data.State.Error,
			StartedAt:  data.State.StartedAt,
			FinishedAt: data.State.FinishedAt,
		}
	}
	if data.HostConfig != nil {
		info.HostConfig = HostConfig{
			Binds:         append([]string(nil), data.HostConfig.Binds...),
			NetworkMode:   string(data.HostConfig.NetworkMode),
			RestartPolicy: string(data.HostConfig.RestartPolicy.Name),
			AutoRemove:    data.HostConfig.AutoRemove,
			Privileged:    data.HostConfig.Privileged,
			PortBindings:  portMapToBindings(data.HostConfig.PortBindings),
			Mounts:        fromDockerMounts(data.HostConfig.Mounts),
			Memory:        data.HostConfig.Resources.Memory,
			MemorySwap:    data.HostConfig.Resources.MemorySwap,
			CPUShares:     data.HostConfig.Resources.CPUShares,
			CPUQuota:      data.HostConfig.Resources.CPUQuota,
			CPUPeriod:     data.HostConfig.Resources.CPUPeriod,
			NanoCPUs:      data.HostConfig.Resources.NanoCPUs,
		}
	}
	if data.NetworkSettings != nil {
		info.Network = NetworkSettings{
			Networks:   make(map[string]EndpointSettings, len(data.NetworkSettings.Networks)),
			IPAddress:  data.NetworkSettings.IPAddress,
			Gateway:    data.NetworkSettings.Gateway,
			MacAddress: data.NetworkSettings.MacAddress,
		}
		for name, settings := range data.NetworkSettings.Networks {
			if settings == nil {
				continue
			}
			info.Network.Networks[name] = EndpointSettings{
				NetworkID:           settings.NetworkID,
				EndpointID:          settings.EndpointID,
				Gateway:             settings.Gateway,
				IPAddress:           settings.IPAddress,
				IPPrefixLen:         int32(settings.IPPrefixLen),
				IPv6Gateway:         settings.IPv6Gateway,
				GlobalIPv6Address:   settings.GlobalIPv6Address,
				GlobalIPv6PrefixLen: int32(settings.GlobalIPv6PrefixLen),
				MacAddress:          settings.MacAddress,
				Aliases:             append([]string(nil), settings.Aliases...),
			}
		}
	}
	return info
}

func toImageSummary(data imagetypes.Summary) ImageSummary {
	return ImageSummary{
		ID:          data.ID,
		RepoTags:    append([]string(nil), data.RepoTags...),
		RepoDigests: append([]string(nil), data.RepoDigests...),
		ParentID:    data.ParentID,
		Created:     time.Unix(data.Created, 0),
		Size:        data.Size,
		SharedSize:  data.SharedSize,
		Containers:  data.Containers,
		Labels:      cloneStringMap(data.Labels),
	}
}

func toImageInfo(data imagetypes.InspectResponse) ImageInfo {
	labels := map[string]string(nil)
	if data.Config != nil {
		labels = cloneStringMap(data.Config.Labels)
	}
	return ImageInfo{
		ID:           data.ID,
		RepoTags:     append([]string(nil), data.RepoTags...),
		RepoDigests:  append([]string(nil), data.RepoDigests...),
		Parent:       data.Parent,
		Created:      data.Created,
		Author:       data.Author,
		Architecture: data.Architecture,
		OS:           data.Os,
		Size:         data.Size,
		VirtualSize:  data.VirtualSize,
		Labels:       labels,
		Layers:       append([]string(nil), data.RootFS.Layers...),
	}
}

func toImageHistory(data imagetypes.HistoryResponseItem) ImageHistoryItem {
	return ImageHistoryItem{
		ID:        data.ID,
		Created:   time.Unix(data.Created, 0),
		CreatedBy: data.CreatedBy,
		Tags:      append([]string(nil), data.Tags...),
		Size:      data.Size,
		Comment:   data.Comment,
	}
}

func toNetworkInfo(data networktypes.Inspect) NetworkInfo {
	containers := make(map[string]NetworkContainer, len(data.Containers))
	for id, item := range data.Containers {
		containers[id] = NetworkContainer{
			Name:        item.Name,
			EndpointID:  item.EndpointID,
			MacAddress:  item.MacAddress,
			IPv4Address: item.IPv4Address,
			IPv6Address: item.IPv6Address,
		}
	}
	return NetworkInfo{
		ID:         data.ID,
		Name:       data.Name,
		Created:    data.Created.Format(time.RFC3339Nano),
		Scope:      data.Scope,
		Driver:     data.Driver,
		EnableIPv4: data.EnableIPv4,
		EnableIPv6: data.EnableIPv6,
		Internal:   data.Internal,
		Attachable: data.Attachable,
		Ingress:    data.Ingress,
		IPAM:       fromDockerIPAM(data.IPAM),
		Containers: containers,
		Options:    cloneStringMap(data.Options),
		Labels:     cloneStringMap(data.Labels),
	}
}

func toVolumeInfo(data *volumetypes.Volume) VolumeInfo {
	if data == nil {
		return VolumeInfo{}
	}
	info := VolumeInfo{
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

func toMountPoints(items []containertypes.MountPoint) []MountPoint {
	result := make([]MountPoint, 0, len(items))
	for _, item := range items {
		result = append(result, MountPoint{
			Type:        string(item.Type),
			Name:        item.Name,
			Source:      item.Source,
			Destination: item.Destination,
			Driver:      item.Driver,
			Mode:        item.Mode,
			RW:          item.RW,
			Propagation: string(item.Propagation),
		})
	}
	return result
}

func fromDockerMounts(items []mounttypes.Mount) []Mount {
	result := make([]Mount, 0, len(items))
	for _, item := range items {
		result = append(result, Mount{
			Type:     string(item.Type),
			Source:   item.Source,
			Target:   item.Target,
			ReadOnly: item.ReadOnly,
		})
	}
	return result
}

func portMapToBindings(portMap nat.PortMap) []PortBinding {
	result := []PortBinding{}
	for port, bindings := range portMap {
		for _, binding := range bindings {
			result = append(result, PortBinding{
				ContainerPort: string(port),
				HostIP:        binding.HostIP,
				HostPort:      binding.HostPort,
			})
		}
	}
	return result
}

func portSetToStrings(portSet nat.PortSet) []string {
	result := make([]string, 0, len(portSet))
	for port := range portSet {
		result = append(result, string(port))
	}
	return result
}

func fromDockerIPAM(data networktypes.IPAM) IPAM {
	configs := make([]IPAMConfig, 0, len(data.Config))
	for _, item := range data.Config {
		configs = append(configs, IPAMConfig{
			Subnet:     item.Subnet,
			IPRange:    item.IPRange,
			Gateway:    item.Gateway,
			AuxAddress: cloneStringMap(item.AuxAddress),
		})
	}
	return IPAM{
		Driver:  data.Driver,
		Options: cloneStringMap(data.Options),
		Config:  configs,
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
