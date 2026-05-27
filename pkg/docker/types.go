package docker

import (
	"io"
	"time"
)

type Config struct {
	Enabled               bool
	Host                  string
	TLSEnabled            bool
	TLSCAPath             string
	TLSCertPath           string
	TLSKeyPath            string
	APIVersion            string
	RequestTimeoutSeconds int32
	DefaultPlatform       string
	DefaultLogTail        int32
	TaskTimeoutSeconds    int32
	RegistryAuths         []RegistryAuth
}

type RegistryAuth struct {
	ServerAddress string
	Username      string
	Password      string
	Token         string
}

type Status struct {
	Enabled   bool
	Connected bool
	Error     string
	Info      *EngineInfo
	Version   *EngineVersion
}

type EngineInfo struct {
	ID                string
	Name              string
	ServerVersion     string
	OperatingSystem   string
	OSType            string
	Architecture      string
	DockerRootDir     string
	Containers        int32
	ContainersRunning int32
	ContainersPaused  int32
	ContainersStopped int32
	Images            int32
	Driver            string
	CgroupDriver      string
	CgroupVersion     string
	MemoryTotal       int64
	CPUs              int32
	Labels            []string
}

type EngineVersion struct {
	Version       string
	APIVersion    string
	MinAPIVersion string
	GitCommit     string
	GoVersion     string
	OS            string
	Arch          string
	KernelVersion string
	BuildTime     string
}

type ContainerListOptions struct {
	All      bool
	Status   string
	Name     string
	Image    string
	Network  string
	Labels   map[string]string
	Page     int64
	PageSize int64
}

type ContainerSummary struct {
	ID          string
	Names       []string
	Image       string
	ImageID     string
	Command     string
	Created     time.Time
	State       string
	Status      string
	Labels      map[string]string
	Ports       []Port
	Mounts      []MountPoint
	NetworkMode string
	Networks    []string
}

type Port struct {
	IP          string
	PrivatePort uint32
	PublicPort  uint32
	Type        string
}

type MountPoint struct {
	Type        string
	Name        string
	Source      string
	Destination string
	Driver      string
	Mode        string
	RW          bool
	Propagation string
}

type ContainerInfo struct {
	ID           string
	Name         string
	Image        string
	ImageID      string
	Path         string
	Args         []string
	Created      string
	State        ContainerState
	Config       ContainerConfig
	HostConfig   HostConfig
	Network      NetworkSettings
	Mounts       []MountPoint
	RestartCount int32
	Platform     string
	Driver       string
	LogPath      string
}

type ContainerState struct {
	Status     string
	Running    bool
	Paused     bool
	Restarting bool
	OOMKilled  bool
	Dead       bool
	Pid        int32
	ExitCode   int32
	Error      string
	StartedAt  string
	FinishedAt string
}

type ContainerConfig struct {
	Hostname     string
	User         string
	Env          []string
	Cmd          []string
	Image        string
	WorkingDir   string
	Entrypoint   []string
	Labels       map[string]string
	ExposedPorts []string
	Tty          bool
	OpenStdin    bool
}

type HostConfig struct {
	Binds         []string
	NetworkMode   string
	RestartPolicy string
	AutoRemove    bool
	Privileged    bool
	PortBindings  []PortBinding
	Mounts        []Mount
	Memory        int64
	MemorySwap    int64
	CPUShares     int64
	CPUQuota      int64
	CPUPeriod     int64
	NanoCPUs      int64
}

type PortBinding struct {
	ContainerPort string
	HostIP        string
	HostPort      string
}

type Mount struct {
	Type     string
	Source   string
	Target   string
	ReadOnly bool
}

type NetworkSettings struct {
	Networks   map[string]EndpointSettings
	IPAddress  string
	Gateway    string
	MacAddress string
}

type EndpointSettings struct {
	NetworkID           string
	EndpointID          string
	Gateway             string
	IPAddress           string
	IPPrefixLen         int32
	IPv6Gateway         string
	GlobalIPv6Address   string
	GlobalIPv6PrefixLen int32
	MacAddress          string
	Aliases             []string
}

type CreateContainerOptions struct {
	Name          string
	Image         string
	Hostname      string
	User          string
	Env           []string
	Cmd           []string
	Entrypoint    []string
	WorkingDir    string
	Labels        map[string]string
	Tty           bool
	OpenStdin     bool
	Network       string
	RestartPolicy string
	AutoRemove    bool
	Privileged    bool
	Ports         []PortBinding
	Mounts        []Mount
	Memory        int64
	MemorySwap    int64
	CPUShares     int64
	CPUQuota      int64
	CPUPeriod     int64
	NanoCPUs      int64
	Platform      string
}

type UpdateContainerOptions struct {
	Name          string
	RestartPolicy string
	Memory        int64
	MemorySwap    int64
	CPUShares     int64
	CPUQuota      int64
	CPUPeriod     int64
	NanoCPUs      int64
}

type RecreateContainerOptions struct {
	ID            string
	Create        CreateContainerOptions
	Force         bool
	RemoveVolumes bool
}

type LogOptions struct {
	Stdout     bool
	Stderr     bool
	Since      string
	Until      string
	Timestamps bool
	Follow     bool
	Tail       string
	Details    bool
}

type ExecOptions struct {
	Cmd          []string
	Env          []string
	User         string
	WorkingDir   string
	Tty          bool
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Detach       bool
}

type ExecSession struct {
	ID     string
	Closer func()
	Reader io.Reader
	Writer io.Writer
}

type ImageListOptions struct {
	All      bool
	Dangling *bool
	Keyword  string
	Labels   map[string]string
	Page     int64
	PageSize int64
}

type ImageSummary struct {
	ID          string
	RepoTags    []string
	RepoDigests []string
	ParentID    string
	Created     time.Time
	Size        int64
	SharedSize  int64
	Containers  int64
	Labels      map[string]string
}

type ImageInfo struct {
	ID           string
	RepoTags     []string
	RepoDigests  []string
	Parent       string
	Created      string
	Author       string
	Architecture string
	OS           string
	Size         int64
	VirtualSize  int64
	Labels       map[string]string
	Layers       []string
}

type PullImageOptions struct {
	Reference    string
	Platform     string
	RegistryAuth *RegistryAuth
}

type BuildImageOptions struct {
	ContextPath string
	Dockerfile  string
	Tags        []string
	BuildArgs   map[string]string
	Labels      map[string]string
	Platform    string
	NoCache     bool
	PullParent  bool
	Remove      bool
	ForceRemove bool
}

type UpdateImageTagsOptions struct {
	ImageID     string
	AddTags     []string
	DeleteTags  []string
	ForceDelete bool
}

type ImageHistoryItem struct {
	ID        string
	Created   time.Time
	CreatedBy string
	Tags      []string
	Size      int64
	Comment   string
}

type NetworkListOptions struct {
	Name     string
	Driver   string
	Scope    string
	Labels   map[string]string
	Page     int64
	PageSize int64
}

type NetworkInfo struct {
	ID         string
	Name       string
	Created    string
	Scope      string
	Driver     string
	EnableIPv4 bool
	EnableIPv6 bool
	Internal   bool
	Attachable bool
	Ingress    bool
	IPAM       IPAM
	Containers map[string]NetworkContainer
	Options    map[string]string
	Labels     map[string]string
}

type IPAM struct {
	Driver  string
	Options map[string]string
	Config  []IPAMConfig
}

type IPAMConfig struct {
	Subnet     string
	IPRange    string
	Gateway    string
	AuxAddress map[string]string
}

type NetworkContainer struct {
	Name        string
	EndpointID  string
	MacAddress  string
	IPv4Address string
	IPv6Address string
}

type CreateNetworkOptions struct {
	Name       string
	Driver     string
	Scope      string
	EnableIPv4 *bool
	EnableIPv6 *bool
	Internal   bool
	Attachable bool
	Ingress    bool
	IPAM       IPAM
	Options    map[string]string
	Labels     map[string]string
}

type UpdateNetworkOptions struct {
	ID     string
	Create CreateNetworkOptions
	Force  bool
}

type RecreateNetworkOptions struct {
	ID     string
	Create CreateNetworkOptions
	Force  bool
}

type ConnectNetworkOptions struct {
	NetworkID   string
	ContainerID string
	Aliases     []string
	IPv4Address string
	IPv6Address string
}

type VolumeListOptions struct {
	Name     string
	Driver   string
	Labels   map[string]string
	Page     int64
	PageSize int64
}

type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	CreatedAt  string
	Status     map[string]string
	Labels     map[string]string
	Scope      string
	Options    map[string]string
	UsageSize  int64
	RefCount   int64
}

type CreateVolumeOptions struct {
	Name       string
	Driver     string
	Labels     map[string]string
	DriverOpts map[string]string
}

type UpdateVolumeOptions struct {
	Name       string
	Labels     map[string]string
	DriverOpts map[string]string
	Create     CreateVolumeOptions
	Force      bool
}

type RecreateVolumeOptions struct {
	Name   string
	Create CreateVolumeOptions
	Force  bool
}

type VolumeArchiveOptions struct {
	VolumeName  string
	ArchivePath string
}
