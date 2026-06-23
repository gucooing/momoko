package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	v1 "momoko/api/gen/v1"
	"strings"
	"sync"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	mounttypes "github.com/docker/docker/api/types/mount"
	networktypes "github.com/docker/docker/api/types/network"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	ErrDisabled     = errors.New("docker 未启用")
	ErrNotReady     = errors.New("docker 未连接")
	ErrTaskNotFound = errors.New("docker 任务不存在")
)

type Manager struct {
	mu     sync.RWMutex
	cfg    *v1.DockerConfigInfo
	client *client.Client
	tasks  *taskRunner
}

func NewManager(cfg *v1.DockerConfigInfo) (*Manager, error) {
	m := &Manager{
		cfg:   cfg,
		tasks: newTaskRunner(),
	}
	if cfg.Enabled {
		if err := m.Reconfigure(cfg); err != nil {
			return m, err
		}
	}
	return m, nil
}

func (m *Manager) Reconfigure(cfg *v1.DockerConfigInfo) error {
	var cli *client.Client
	if cfg.Enabled {
		var err error
		cli, err = newClient(cfg)
		if err != nil {
			return err
		}
	}

	m.mu.Lock()
	old := m.client
	m.cfg = cfg
	m.client = cli
	m.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return nil
}

func (m *Manager) Config() *v1.DockerConfigInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Status(ctx context.Context) Status {
	cfg := m.Config()
	status := Status{Enabled: cfg.Enabled}
	if !cfg.Enabled {
		status.Error = ErrDisabled.Error()
		return status
	}
	cli, err := m.getClient()
	if err != nil {
		status.Error = err.Error()
		return status
	}
	info, err := cli.Info(ctx)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Connected = true
	status.Info = toEngineInfo(info)
	status.Version = toEngineVersion(version)
	return status
}

func (m *Manager) Test(ctx context.Context, cfg *v1.DockerConfigInfo) (Status, error) {
	cli, err := newClient(cfg)
	if err != nil {
		return Status{Enabled: cfg.Enabled, Error: err.Error()}, err
	}
	defer cli.Close()
	info, err := cli.Info(ctx)
	if err != nil {
		return Status{Enabled: cfg.Enabled, Error: err.Error()}, err
	}
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		return Status{Enabled: cfg.Enabled, Error: err.Error()}, err
	}
	return Status{
		Enabled:   cfg.Enabled,
		Connected: true,
		Info:      toEngineInfo(info),
		Version:   toEngineVersion(version),
	}, nil
}

func (m *Manager) Task(id string) (*Task, error) {
	task, ok := m.tasks.Get(id)
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *Manager) Tasks() []*Task {
	return m.tasks.List()
}

func (m *Manager) SubscribeTask(id string) (<-chan TaskEvent, func(), error) {
	ch, cancel, ok := m.tasks.Subscribe(id)
	if !ok {
		return nil, nil, ErrTaskNotFound
	}
	return ch, cancel, nil
}

func (m *Manager) getClient() (*client.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.cfg.Enabled {
		return nil, ErrDisabled
	}
	if m.client == nil {
		return nil, ErrNotReady
	}
	return m.client, nil
}

func (m *Manager) timeout() time.Duration {
	cfg := m.Config()
	if cfg.RequestTimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(cfg.RequestTimeoutSeconds) * time.Second
}

func (m *Manager) taskTimeout() time.Duration {
	cfg := m.Config()
	if cfg.TaskTimeoutSeconds <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(cfg.TaskTimeoutSeconds) * time.Second
}

func newClient(cfg *v1.DockerConfigInfo) (*client.Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if cfg.Host != "" {
		opts = append(opts, client.WithHost(cfg.Host))
	}
	if cfg.ApiVersion != "" {
		opts = append(opts, client.WithVersion(cfg.ApiVersion))
	}
	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	opts = append(opts, client.WithTimeout(timeout))
	if cfg.TlsEnabled {
		if cfg.TlsCaPath == "" || cfg.TlsCertPath == "" || cfg.TlsKeyPath == "" {
			return nil, errors.New("docker TLS 证书路径不能为空")
		}
		opts = append(opts, client.WithTLSClientConfig(
			cfg.TlsCaPath,
			cfg.TlsCertPath,
			cfg.TlsKeyPath,
		))
	}
	return client.NewClientWithOpts(opts...)
}

func (m *Manager) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, m.timeout())
}

func registryAuthHeader(auth *RegistryAuth) (string, error) {
	if auth == nil {
		return "", nil
	}
	cfg := registrytypes.AuthConfig{
		Username:      auth.Username,
		Password:      auth.Password,
		ServerAddress: auth.ServerAddress,
		IdentityToken: auth.Token,
		RegistryToken: auth.Token,
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

func parsePlatform(platform string) (*ocispec.Platform, error) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return nil, nil
	}
	parts := strings.Split(platform, "/")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid platform %q", platform)
	}
	p := &ocispec.Platform{OS: parts[0], Architecture: parts[1]}
	if len(parts) == 3 {
		p.Variant = parts[2]
	}
	return p, nil
}

func toContainerResources(opts CreateContainerOptions) containertypes.Resources {
	return containertypes.Resources{
		Memory:     opts.Memory,
		MemorySwap: opts.MemorySwap,
		CPUShares:  opts.CPUShares,
		CPUQuota:   opts.CPUQuota,
		CPUPeriod:  opts.CPUPeriod,
		NanoCPUs:   opts.NanoCPUs,
	}
}

func toUpdateResources(opts UpdateContainerOptions) containertypes.Resources {
	return containertypes.Resources{
		Memory:     opts.Memory,
		MemorySwap: opts.MemorySwap,
		CPUShares:  opts.CPUShares,
		CPUQuota:   opts.CPUQuota,
		CPUPeriod:  opts.CPUPeriod,
		NanoCPUs:   opts.NanoCPUs,
	}
}

func toDockerMounts(mounts []Mount) []mounttypes.Mount {
	result := make([]mounttypes.Mount, 0, len(mounts))
	for _, item := range mounts {
		mountType := mounttypes.Type(item.Type)
		if mountType == "" {
			mountType = mounttypes.TypeBind
		}
		result = append(result, mounttypes.Mount{
			Type:     mountType,
			Source:   item.Source,
			Target:   item.Target,
			ReadOnly: item.ReadOnly,
		})
	}
	return result
}

func toPortBindings(bindings []PortBinding) (nat.PortSet, nat.PortMap, error) {
	exposed := nat.PortSet{}
	portMap := nat.PortMap{}
	for _, item := range bindings {
		if strings.Contains(item.ContainerPort, ":") {
			mappings, err := nat.ParsePortSpec(item.ContainerPort)
			if err != nil {
				return nil, nil, err
			}
			for _, mapping := range mappings {
				exposed[mapping.Port] = struct{}{}
				portMap[mapping.Port] = append(portMap[mapping.Port], mapping.Binding)
			}
			continue
		}
		proto, portNumber := nat.SplitProtoPort(item.ContainerPort)
		port, err := nat.NewPort(proto, portNumber)
		if err != nil {
			return nil, nil, err
		}
		exposed[port] = struct{}{}
		portMap[port] = append(portMap[port], nat.PortBinding{
			HostIP:   item.HostIP,
			HostPort: item.HostPort,
		})
	}
	return exposed, portMap, nil
}

func toRestartPolicy(name string) containertypes.RestartPolicy {
	name = strings.TrimSpace(name)
	if name == "" {
		return containertypes.RestartPolicy{}
	}
	return containertypes.RestartPolicy{Name: containertypes.RestartPolicyMode(name)}
}

func toNetworkEndpoint(opts ConnectNetworkOptions) *networktypes.EndpointSettings {
	return &networktypes.EndpointSettings{
		Aliases: opts.Aliases,
		IPAMConfig: &networktypes.EndpointIPAMConfig{
			IPv4Address: opts.IPv4Address,
			IPv6Address: opts.IPv6Address,
		},
	}
}

func toDockerIPAM(data IPAM) *networktypes.IPAM {
	if data.Driver == "" && len(data.Options) == 0 && len(data.Config) == 0 {
		return nil
	}
	configs := make([]networktypes.IPAMConfig, 0, len(data.Config))
	for _, item := range data.Config {
		configs = append(configs, networktypes.IPAMConfig{
			Subnet:     item.Subnet,
			IPRange:    item.IPRange,
			Gateway:    item.Gateway,
			AuxAddress: item.AuxAddress,
		})
	}
	return &networktypes.IPAM{
		Driver:  data.Driver,
		Options: data.Options,
		Config:  configs,
	}
}

func imageRef(repository, tag string) string {
	if tag == "" {
		return repository
	}
	return repository + ":" + tag
}

func splitImageRef(ref string) (string, string) {
	idx := strings.LastIndex(ref, ":")
	if idx <= strings.LastIndex(ref, "/") {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
}

func stringMapStatus(data map[string]interface{}) map[string]string {
	if len(data) == 0 {
		return nil
	}
	result := make(map[string]string, len(data))
	for k, v := range data {
		result[k] = fmt.Sprint(v)
	}
	return result
}

func filtersFromLabels(labels map[string]string) filters.Args {
	args := filters.NewArgs()
	for k, v := range labels {
		if v == "" {
			args.Add("label", k)
			continue
		}
		args.Add("label", k+"="+v)
	}
	return args
}

func pageSlice[T any](items []T, page, pageSize int64) []T {
	if page <= 0 || pageSize <= 0 {
		return items
	}
	start := (page - 1) * pageSize
	if start < 0 || start >= int64(len(items)) {
		return []T{}
	}
	end := start + pageSize
	if end > int64(len(items)) {
		end = int64(len(items))
	}
	return items[start:end]
}
