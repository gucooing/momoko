package sysinfo

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	gcpu "github.com/shirou/gopsutil/v4/cpu"
	gdisk "github.com/shirou/gopsutil/v4/disk"
	ghost "github.com/shirou/gopsutil/v4/host"
	gmem "github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
)

type Collector struct {
	mu              sync.Mutex
	previousNetwork map[string]byteSample
	previousDisk    map[string]byteSample
}

type Overview struct {
	Version           Version
	BootTime          time.Time
	UptimeSeconds     uint64
	CPU               CPUOverview
	Memory            MemoryOverview
	NetworkInterfaces []NetworkInterfaceOverview
	DiskPartitions    []DiskPartitionOverview
	DiskIOs           []DiskIOOverview
	SampleTime        time.Time
}

type Status struct {
	CPU        CPUStatus
	Memory     MemoryStatus
	Network    NetworkStatus
	Disk       DiskStatus
	SampleTime time.Time
}

type StatusFilter struct {
	InterfaceName *string
	DiskName      *string
	Mountpoint    *string
}

type Version struct {
	Hostname        string
	OS              string
	Platform        string
	PlatformFamily  string
	PlatformVersion string
	KernelVersion   string
	KernelArch      string
}

type CPUOverview struct {
	LogicalCount  int
	PhysicalCount int
	ModelName     string
}

type CPUStatus struct {
	TotalPercent  float64
	Cores         []CPUCoreStatus
	LogicalCount  int
	PhysicalCount int
}

type CPUCoreStatus struct {
	CoreID  int
	Percent float64
}

type MemoryOverview struct {
	PhysicalMemory PhysicalMemoryOverview
	VirtualMemory  VirtualMemoryOverview
}

type MemoryStatus struct {
	PhysicalMemory PhysicalMemoryStatus
	VirtualMemory  VirtualMemoryStatus
}

type PhysicalMemoryOverview struct {
	TotalBytes uint64
}

type PhysicalMemoryStatus struct {
	TotalBytes     uint64
	AvailableBytes uint64
	UsedBytes      uint64
	FreeBytes      uint64
	UsedPercent    float64
}

type VirtualMemoryOverview struct {
	TotalBytes uint64
}

type VirtualMemoryStatus struct {
	TotalBytes        uint64
	UsedBytes         uint64
	FreeBytes         uint64
	UsedPercent       float64
	SwapInBytes       uint64
	SwapOutBytes      uint64
	PageInCount       uint64
	PageOutCount      uint64
	PageFaultCount    uint64
	PageMajFaultCount uint64
}

type NetworkInterfaceOverview struct {
	Name         string
	HardwareAddr string
	MTU          int
	Flags        []string
	Addrs        []string
	IsUp         bool
	IsLoopback   bool
}

type NetworkStatus struct {
	Total             *NetworkInterfaceStatus
	Interfaces        []*NetworkInterfaceStatus
	SelectedInterface *NetworkInterfaceStatus
	Connections       NetworkConnections
}

type NetworkInterfaceStatus struct {
	NetworkInterfaceOverview
	BytesSent                  uint64
	BytesRecv                  uint64
	PacketsSent                uint64
	PacketsRecv                uint64
	ErrIn                      uint64
	ErrOut                     uint64
	DropIn                     uint64
	DropOut                    uint64
	UploadRateBytesPerSecond   float64
	DownloadRateBytesPerSecond float64
}

type NetworkConnections struct {
	Supported   bool
	Error       string
	TCPCount    int64
	UDPCount    int64
	TotalCount  int64
	TCPStatuses map[string]int64
}

type DiskPartitionOverview struct {
	Device     string
	Mountpoint string
	Fstype     string
	Opts       []string
}

type DiskIOOverview struct {
	Name         string
	SerialNumber string
	Label        string
}

type DiskStatus struct {
	Total             *DiskPartitionStatus
	Partitions        []*DiskPartitionStatus
	SelectedPartition *DiskPartitionStatus
	TotalIO           *DiskIOStatus
	IOs               []*DiskIOStatus
	SelectedIO        *DiskIOStatus
	IOSupported       bool
	IOError           string
}

type DiskPartitionStatus struct {
	DiskPartitionOverview
	Supported         bool
	Error             string
	TotalBytes        uint64
	FreeBytes         uint64
	UsedBytes         uint64
	UsedPercent       float64
	InodesTotal       uint64
	InodesUsed        uint64
	InodesFree        uint64
	InodesUsedPercent float64
}

type DiskIOStatus struct {
	Name                    string
	SerialNumber            string
	Label                   string
	ReadCount               uint64
	WriteCount              uint64
	ReadBytes               uint64
	WriteBytes              uint64
	ReadRateBytesPerSecond  float64
	WriteRateBytesPerSecond float64
	IopsInProgress          uint64
	IOTime                  uint64
}

type byteSample struct {
	readOrRecv  uint64
	writeOrSent uint64
	at          time.Time
}

func NewCollector() *Collector {
	return &Collector{
		previousNetwork: make(map[string]byteSample),
		previousDisk:    make(map[string]byteSample),
	}
}

func (c *Collector) Overview(ctx context.Context) (*Overview, error) {
	now := time.Now()
	hostInfo, err := ghost.InfoWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("host info: %w", err)
	}
	cpuInfo := readCPUOverview(ctx)
	memInfo, err := readMemoryOverview(ctx)
	if err != nil {
		return nil, err
	}
	networkInterfaces, err := readNetworkOverview(ctx)
	if err != nil {
		return nil, err
	}
	diskPartitions, err := readDiskOverview(ctx)
	if err != nil {
		return nil, err
	}
	diskIOs := readDiskIOOverview(ctx)

	return &Overview{
		Version: Version{
			Hostname:        hostInfo.Hostname,
			OS:              hostInfo.OS,
			Platform:        hostInfo.Platform,
			PlatformFamily:  hostInfo.PlatformFamily,
			PlatformVersion: hostInfo.PlatformVersion,
			KernelVersion:   hostInfo.KernelVersion,
			KernelArch:      hostInfo.KernelArch,
		},
		BootTime:          time.Unix(int64(hostInfo.BootTime), 0),
		UptimeSeconds:     hostInfo.Uptime,
		CPU:               cpuInfo,
		Memory:            memInfo,
		NetworkInterfaces: networkInterfaces,
		DiskPartitions:    diskPartitions,
		DiskIOs:           diskIOs,
		SampleTime:        now,
	}, nil
}

func (c *Collector) Status(ctx context.Context, filter StatusFilter) (*Status, error) {
	now := time.Now()
	cpuInfo, err := readCPUStatus(ctx)
	if err != nil {
		return nil, err
	}
	memInfo, err := readMemoryStatus(ctx)
	if err != nil {
		return nil, err
	}
	netInfo, err := c.readNetworkStatus(ctx, stringPtrValue(filter.InterfaceName), now)
	if err != nil {
		return nil, err
	}
	diskInfo, err := c.readDiskStatus(ctx, filter, now)
	if err != nil {
		return nil, err
	}

	return &Status{
		CPU:        cpuInfo,
		Memory:     memInfo,
		Network:    netInfo,
		Disk:       diskInfo,
		SampleTime: now,
	}, nil
}

func readCPUOverview(ctx context.Context) CPUOverview {
	logicalCount, _ := gcpu.CountsWithContext(ctx, true)
	physicalCount, _ := gcpu.CountsWithContext(ctx, false)
	infos, _ := gcpu.InfoWithContext(ctx)
	modelName := ""
	if len(infos) > 0 {
		modelName = infos[0].ModelName
	}
	return CPUOverview{
		LogicalCount:  logicalCount,
		PhysicalCount: physicalCount,
		ModelName:     modelName,
	}
}

func readCPUStatus(ctx context.Context) (CPUStatus, error) {
	totalPercent, err := gcpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return CPUStatus{}, fmt.Errorf("cpu total: %w", err)
	}
	corePercent, err := gcpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return CPUStatus{}, fmt.Errorf("cpu cores: %w", err)
	}
	overview := readCPUOverview(ctx)

	status := CPUStatus{
		Cores:         make([]CPUCoreStatus, 0, len(corePercent)),
		LogicalCount:  overview.LogicalCount,
		PhysicalCount: overview.PhysicalCount,
	}
	if len(totalPercent) > 0 {
		status.TotalPercent = totalPercent[0]
	}
	for i, percent := range corePercent {
		status.Cores = append(status.Cores, CPUCoreStatus{
			CoreID:  i,
			Percent: percent,
		})
	}
	return status, nil
}

func readMemoryOverview(ctx context.Context) (MemoryOverview, error) {
	memInfo, err := gmem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return MemoryOverview{}, fmt.Errorf("memory: %w", err)
	}
	return MemoryOverview{
		PhysicalMemory: PhysicalMemoryOverview{
			TotalBytes: memInfo.Total,
		},
		VirtualMemory: readVirtualMemoryOverview(ctx),
	}, nil
}

func readMemoryStatus(ctx context.Context) (MemoryStatus, error) {
	memInfo, err := gmem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return MemoryStatus{}, fmt.Errorf("memory: %w", err)
	}
	return MemoryStatus{
		PhysicalMemory: PhysicalMemoryStatus{
			TotalBytes:     memInfo.Total,
			AvailableBytes: memInfo.Available,
			UsedBytes:      memInfo.Used,
			FreeBytes:      memInfo.Free,
			UsedPercent:    memInfo.UsedPercent,
		},
		VirtualMemory: readVirtualMemoryStatus(ctx),
	}, nil
}

func readVirtualMemoryOverview(ctx context.Context) VirtualMemoryOverview {
	swapInfo, err := gmem.SwapMemoryWithContext(ctx)
	if err != nil {
		return VirtualMemoryOverview{}
	}
	return VirtualMemoryOverview{
		TotalBytes: swapInfo.Total,
	}
}

func readVirtualMemoryStatus(ctx context.Context) VirtualMemoryStatus {
	swapInfo, err := gmem.SwapMemoryWithContext(ctx)
	if err != nil {
		return VirtualMemoryStatus{}
	}
	return VirtualMemoryStatus{
		TotalBytes:        swapInfo.Total,
		UsedBytes:         swapInfo.Used,
		FreeBytes:         swapInfo.Free,
		UsedPercent:       swapInfo.UsedPercent,
		SwapInBytes:       swapInfo.Sin,
		SwapOutBytes:      swapInfo.Sout,
		PageInCount:       swapInfo.PgIn,
		PageOutCount:      swapInfo.PgOut,
		PageFaultCount:    swapInfo.PgFault,
		PageMajFaultCount: swapInfo.PgMajFault,
	}
}

func readNetworkOverview(ctx context.Context) ([]NetworkInterfaceOverview, error) {
	interfaces, err := gnet.InterfacesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("network interfaces: %w", err)
	}
	items := make([]NetworkInterfaceOverview, 0, len(interfaces))
	for _, item := range interfaces {
		items = append(items, toNetworkInterfaceOverview(item))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (c *Collector) readNetworkStatus(ctx context.Context, interfaceName string, now time.Time) (NetworkStatus, error) {
	counters, err := gnet.IOCountersWithContext(ctx, true)
	if err != nil {
		return NetworkStatus{}, fmt.Errorf("network counters: %w", err)
	}
	interfaces, err := gnet.InterfacesWithContext(ctx)
	if err != nil {
		interfaces = nil
	}

	interfaceMap := make(map[string]gnet.InterfaceStat, len(interfaces))
	for _, item := range interfaces {
		interfaceMap[item.Name] = item
	}

	network := NetworkStatus{
		Total:             new(NetworkInterfaceStatus),
		Interfaces:        make([]*NetworkInterfaceStatus, 0, len(counters)),
		SelectedInterface: nil,
		Connections:       readNetworkConnections(ctx),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	next := make(map[string]byteSample, len(counters)+1)
	for _, counter := range counters {
		item := toNetworkInterfaceStatus(counter, interfaceMap[counter.Name])
		applyNetworkRate(item, c.previousNetwork[item.Name], now)
		network.Interfaces = append(network.Interfaces, item)
		addNetworkTotal(network.Total, item)
		next[item.Name] = byteSample{
			readOrRecv:  item.BytesRecv,
			writeOrSent: item.BytesSent,
			at:          now,
		}
	}

	network.Total.NetworkInterfaceOverview.Name = "total"
	applyNetworkRate(network.Total, c.previousNetwork[network.Total.Name], now)
	next[network.Total.Name] = byteSample{
		readOrRecv:  network.Total.BytesRecv,
		writeOrSent: network.Total.BytesSent,
		at:          now,
	}
	sort.Slice(network.Interfaces, func(i, j int) bool {
		return network.Interfaces[i].Name < network.Interfaces[j].Name
	})
	network.SelectedInterface = selectNetworkInterface(network.Interfaces, interfaceName)
	c.previousNetwork = next
	return network, nil
}

func toNetworkInterfaceOverview(info gnet.InterfaceStat) NetworkInterfaceOverview {
	flags := append([]string(nil), info.Flags...)
	addrs := make([]string, 0, len(info.Addrs))
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.Addr)
	}
	return NetworkInterfaceOverview{
		Name:         info.Name,
		HardwareAddr: info.HardwareAddr,
		MTU:          info.MTU,
		Flags:        flags,
		Addrs:        addrs,
		IsUp:         hasFlag(flags, "up"),
		IsLoopback:   hasFlag(flags, "loopback"),
	}
}

func toNetworkInterfaceStatus(counter gnet.IOCountersStat, info gnet.InterfaceStat) *NetworkInterfaceStatus {
	overview := toNetworkInterfaceOverview(info)
	if overview.Name == "" {
		overview.Name = counter.Name
	}
	return &NetworkInterfaceStatus{
		NetworkInterfaceOverview: overview,
		BytesSent:                counter.BytesSent,
		BytesRecv:                counter.BytesRecv,
		PacketsSent:              counter.PacketsSent,
		PacketsRecv:              counter.PacketsRecv,
		ErrIn:                    counter.Errin,
		ErrOut:                   counter.Errout,
		DropIn:                   counter.Dropin,
		DropOut:                  counter.Dropout,
	}
}

func addNetworkTotal(total *NetworkInterfaceStatus, item *NetworkInterfaceStatus) {
	total.BytesSent += item.BytesSent
	total.BytesRecv += item.BytesRecv
	total.PacketsSent += item.PacketsSent
	total.PacketsRecv += item.PacketsRecv
	total.ErrIn += item.ErrIn
	total.ErrOut += item.ErrOut
	total.DropIn += item.DropIn
	total.DropOut += item.DropOut
}

func applyNetworkRate(item *NetworkInterfaceStatus, previous byteSample, now time.Time) {
	upload, download := rates(previous.writeOrSent, item.BytesSent, previous.readOrRecv, item.BytesRecv, previous.at, now)
	item.UploadRateBytesPerSecond = upload
	item.DownloadRateBytesPerSecond = download
}

func selectNetworkInterface(items []*NetworkInterfaceStatus, name string) *NetworkInterfaceStatus {
	if name != "" {
		for i := range items {
			if items[i].Name == name {
				return items[i]
			}
		}
	}
	for i := range items {
		if items[i].IsUp && !items[i].IsLoopback {
			return items[i]
		}
	}
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func readNetworkConnections(ctx context.Context) NetworkConnections {
	connections := NetworkConnections{
		Supported:   true,
		TCPStatuses: make(map[string]int64),
	}

	tcpItems, tcpErr := gnet.ConnectionsWithoutUidsWithContext(ctx, "tcp")
	if tcpErr == nil {
		connections.TCPCount = int64(len(tcpItems))
		for _, item := range tcpItems {
			status := item.Status
			if status == "" {
				status = "UNKNOWN"
			}
			connections.TCPStatuses[status]++
		}
	}

	udpItems, udpErr := gnet.ConnectionsWithoutUidsWithContext(ctx, "udp")
	if udpErr == nil {
		connections.UDPCount = int64(len(udpItems))
	}

	if tcpErr != nil || udpErr != nil {
		connections.Supported = false
		connections.Error = joinErrors(tcpErr, udpErr)
	}
	connections.TotalCount = connections.TCPCount + connections.UDPCount
	return connections
}

func readDiskOverview(ctx context.Context) ([]DiskPartitionOverview, error) {
	partitions, err := gdisk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("disk partitions: %w", err)
	}
	items := make([]DiskPartitionOverview, 0, len(partitions))
	for _, item := range partitions {
		items = append(items, toDiskPartitionOverview(item))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Mountpoint < items[j].Mountpoint
	})
	return items, nil
}

func readDiskIOOverview(ctx context.Context) []DiskIOOverview {
	counters, err := gdisk.IOCountersWithContext(ctx)
	if err != nil {
		return make([]DiskIOOverview, 0)
	}
	items := make([]DiskIOOverview, 0, len(counters))
	for name, counter := range counters {
		items = append(items, DiskIOOverview{
			Name:         name,
			SerialNumber: counter.SerialNumber,
			Label:        counter.Label,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func (c *Collector) readDiskStatus(ctx context.Context, filter StatusFilter, now time.Time) (DiskStatus, error) {
	partitions, err := gdisk.PartitionsWithContext(ctx, false)
	if err != nil {
		return DiskStatus{}, fmt.Errorf("disk partitions: %w", err)
	}
	status := DiskStatus{
		Total:             new(DiskPartitionStatus),
		Partitions:        make([]*DiskPartitionStatus, 0, len(partitions)),
		SelectedPartition: nil,
		TotalIO:           nil,
		IOs:               nil,
		SelectedIO:        nil,
		IOSupported:       true,
		IOError:           "",
	}
	for _, partition := range partitions {
		item := readDiskPartitionStatus(ctx, partition)
		status.Partitions = append(status.Partitions, item)
		addDiskPartitionTotal(status.Total, item)
	}
	sort.Slice(status.Partitions, func(i, j int) bool {
		return status.Partitions[i].Mountpoint < status.Partitions[j].Mountpoint
	})
	status.Total.DiskPartitionOverview.Mountpoint = "total"
	status.Total.Supported = true
	status.SelectedPartition = selectDiskPartition(status.Partitions, stringPtrValue(filter.DiskName), stringPtrValue(filter.Mountpoint))

	status.IOs, err = c.readDiskIOStatus(ctx, now)
	if err != nil {
		status.IOSupported = false
		status.IOError = err.Error()
		status.IOs = make([]*DiskIOStatus, 0)
		return status, nil
	}
	status.TotalIO = totalDiskIO(status.IOs)
	status.SelectedIO = selectDiskIO(status.IOs, stringPtrValue(filter.DiskName))
	return status, nil
}

func readDiskPartitionStatus(ctx context.Context, partition gdisk.PartitionStat) *DiskPartitionStatus {
	item := &DiskPartitionStatus{
		DiskPartitionOverview: toDiskPartitionOverview(partition),
		Supported:             true,
	}
	usage, err := gdisk.UsageWithContext(ctx, partition.Mountpoint)
	if err != nil {
		item.Supported = false
		item.Error = err.Error()
		return item
	}
	item.TotalBytes = usage.Total
	item.FreeBytes = usage.Free
	item.UsedBytes = usage.Used
	item.UsedPercent = usage.UsedPercent
	item.InodesTotal = usage.InodesTotal
	item.InodesUsed = usage.InodesUsed
	item.InodesFree = usage.InodesFree
	item.InodesUsedPercent = usage.InodesUsedPercent
	return item
}

func toDiskPartitionOverview(partition gdisk.PartitionStat) DiskPartitionOverview {
	return DiskPartitionOverview{
		Device:     partition.Device,
		Mountpoint: partition.Mountpoint,
		Fstype:     partition.Fstype,
		Opts:       append([]string(nil), partition.Opts...),
	}
}

func addDiskPartitionTotal(total *DiskPartitionStatus, item *DiskPartitionStatus) {
	if !item.Supported {
		return
	}
	total.TotalBytes += item.TotalBytes
	total.FreeBytes += item.FreeBytes
	total.UsedBytes += item.UsedBytes
	total.InodesTotal += item.InodesTotal
	total.InodesUsed += item.InodesUsed
	total.InodesFree += item.InodesFree
	if total.TotalBytes > 0 {
		total.UsedPercent = float64(total.UsedBytes) / float64(total.TotalBytes) * 100
	}
	if total.InodesTotal > 0 {
		total.InodesUsedPercent = float64(total.InodesUsed) / float64(total.InodesTotal) * 100
	}
}

func selectDiskPartition(items []*DiskPartitionStatus, diskName, mountpoint string) *DiskPartitionStatus {
	if mountpoint != "" {
		for i := range items {
			if items[i].Mountpoint == mountpoint {
				return items[i]
			}
		}
	}
	if diskName != "" {
		for i := range items {
			if items[i].Device == diskName || items[i].Mountpoint == diskName {
				return items[i]
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func (c *Collector) readDiskIOStatus(ctx context.Context, now time.Time) ([]*DiskIOStatus, error) {
	counters, err := gdisk.IOCountersWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("disk io: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	items := make([]*DiskIOStatus, 0, len(counters))
	next := make(map[string]byteSample, len(counters)+1)
	for name, counter := range counters {
		item := &DiskIOStatus{
			Name:           name,
			SerialNumber:   counter.SerialNumber,
			Label:          counter.Label,
			ReadCount:      counter.ReadCount,
			WriteCount:     counter.WriteCount,
			ReadBytes:      counter.ReadBytes,
			WriteBytes:     counter.WriteBytes,
			IopsInProgress: counter.IopsInProgress,
			IOTime:         counter.IoTime,
		}
		applyDiskRate(item, c.previousDisk[name], now)
		items = append(items, item)
		next[name] = byteSample{
			readOrRecv:  item.ReadBytes,
			writeOrSent: item.WriteBytes,
			at:          now,
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	c.previousDisk = next
	return items, nil
}

func applyDiskRate(item *DiskIOStatus, previous byteSample, now time.Time) {
	write, read := rates(previous.writeOrSent, item.WriteBytes, previous.readOrRecv, item.ReadBytes, previous.at, now)
	item.WriteRateBytesPerSecond = write
	item.ReadRateBytesPerSecond = read
}

func totalDiskIO(items []*DiskIOStatus) *DiskIOStatus {
	total := &DiskIOStatus{Name: "total"}
	for _, item := range items {
		total.ReadCount += item.ReadCount
		total.WriteCount += item.WriteCount
		total.ReadBytes += item.ReadBytes
		total.WriteBytes += item.WriteBytes
		total.ReadRateBytesPerSecond += item.ReadRateBytesPerSecond
		total.WriteRateBytesPerSecond += item.WriteRateBytesPerSecond
		total.IopsInProgress += item.IopsInProgress
		total.IOTime += item.IOTime
	}
	return total
}

func selectDiskIO(items []*DiskIOStatus, name string) *DiskIOStatus {
	if name != "" {
		for i := range items {
			if items[i].Name == name {
				return items[i]
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func rates(previousWrite, currentWrite, previousRead, currentRead uint64, previousTime, now time.Time) (float64, float64) {
	if previousTime.IsZero() {
		return 0, 0
	}
	seconds := now.Sub(previousTime).Seconds()
	if seconds <= 0 {
		return 0, 0
	}
	var writeRate float64
	if currentWrite >= previousWrite {
		writeRate = float64(currentWrite-previousWrite) / seconds
	}
	var readRate float64
	if currentRead >= previousRead {
		readRate = float64(currentRead-previousRead) / seconds
	}
	return writeRate, readRate
}

func hasFlag(flags []string, name string) bool {
	for _, flag := range flags {
		if strings.EqualFold(flag, name) {
			return true
		}
	}
	return false
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func joinErrors(errs ...error) string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	return strings.Join(parts, "; ")
}
