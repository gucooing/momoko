package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/menu"
	"momoko/internal/data/ent/gen/role"
	"momoko/pkg/auth"
	"momoko/pkg/cache"
	"momoko/pkg/constant"
	"momoko/pkg/sysinfo"
)

type SystemUsecase struct {
	sys      SystemRepo
	userRepo UserRepo
	sysInfo  *sysinfo.Collector

	cache cache.Cache[string, *RoleOjb]
}

// 权限缓存
type RoleOjb struct {
	Menus       []*gen.Menu                       // 原始数据
	Permissions map[constant.Permissions]struct{} // 权限快速获取数据
}

func (s *SystemUsecase) Check(ctx context.Context, permissions constant.Permissions) error {
	refreshAuth, ok := auth.FromContext(ctx)
	if !ok {
		return ErrNoPermission
	}
	r, err := s.GetRoleOjbByUserID(ctx, refreshAuth.UserID)
	if err != nil {
		return ErrNoPermission
	}
	if r.Permissions == nil {
		return ErrNoPermission
	}
	_, ok = r.Permissions[permissions]
	if !ok {
		return ErrNoPermission
	}
	return nil
}

type SystemRepo interface {
	GetMenusByRoleId(ctx context.Context, roleId string) ([]*gen.Menu, error)
	GetMenus(ctx context.Context) ([]*gen.Menu, error)
	GetMenu(ctx context.Context, menuId string) (*gen.Menu, error)
	CreateMenu(ctx context.Context, menu *gen.Menu) (*gen.Menu, error)
	UpdateMenu(ctx context.Context, menuInfo *gen.Menu) (*gen.Menu, error)
	DeleteMenu(ctx context.Context, menuId string) error

	GetRoles(ctx context.Context, page, pageSize int64, status *role.Status, name *string) ([]*gen.Role, int64, error)
	GetRole(ctx context.Context, roleId string) (*gen.Role, error)
	CreateRole(ctx context.Context, roleInfo *gen.Role, menuIds []string) (*gen.Role, error)
	UpdateRole(ctx context.Context, roleInfo *gen.Role, menuIds []string) (*gen.Role, error)
	DeleteRole(ctx context.Context, roleIds []string) error
}

func NewSystemUsecase(sys SystemRepo, userRepo UserRepo) *SystemUsecase {
	return &SystemUsecase{
		sys:      sys,
		userRepo: userRepo,
		sysInfo:  sysinfo.NewCollector(),
	}
}

func (s *SystemUsecase) SystemOverview(ctx context.Context) (*v1.SystemOverviewResponse, error) {
	overview, err := s.sysInfo.Overview(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toSystemOverviewResponse(overview), nil
}

func (s *SystemUsecase) SystemStatus(ctx context.Context, req *v1.SystemStatusRequest) (*v1.SystemStatusResponse, error) {
	status, err := s.sysInfo.Status(ctx, sysinfo.StatusFilter{
		InterfaceName: req.InterfaceName,
		DiskName:      req.DiskName,
		Mountpoint:    req.Mountpoint,
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.SystemStatusResponse{
		Cpu:        toCPUStatus(status.CPU),
		Memory:     toMemoryStatus(status.Memory),
		Network:    toNetworkStatus(status.Network),
		Disk:       toDiskStatus(status.Disk),
		SampleTime: timestamppb.New(status.SampleTime),
	}, nil
}

func (s *SystemUsecase) GetRoleOjbByUserID(ctx context.Context, userID string) (*RoleOjb, error) {
	userInfo, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if userInfo.Edges.Role == nil {
		return nil, ErrUserNoRole
	}
	add := func() (*RoleOjb, error) {
		menuInfos, err := s.sys.GetMenusByRoleId(ctx, userInfo.Edges.Role.ID)
		if err != nil {
			return nil, ErrSystem(err)
		}
		return &RoleOjb{
			Menus:       menuInfos,
			Permissions: toPermissions(menuInfos),
		}, nil
	}
	ojb, ok := s.cache.GetByAdd(userInfo.Edges.Role.ID, add)
	if !ok {
		return nil, ErrUserNoRole
	}
	return ojb, nil
}

func (s *SystemUsecase) GetMenusByUserID(ctx context.Context, userID string) ([]*v1.MenuInfo, []string, error) {
	ojb, err := s.GetRoleOjbByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	menus, permissions := toMenuInfos(ojb.Menus)
	return menus, permissions, nil
}

func (s *SystemUsecase) GetAllMenus(ctx context.Context) ([]*v1.MenuInfo, []string, error) {
	menuInfos, err := s.sys.GetMenus(ctx)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	menus, permissions := toMenuInfos(menuInfos)
	return menus, permissions, nil
}

func (s *SystemUsecase) GetMenu(ctx context.Context, menuId string) (*v1.MenuInfo, error) {
	menuInfo, err := s.sys.GetMenu(ctx, menuId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toMenuInfo(menuInfo), nil
}

func (s *SystemUsecase) AddMenu(ctx context.Context, menu *v1.AdminAddPermissionsRequest) error {
	_, err := s.sys.CreateMenu(ctx, &gen.Menu{
		ID:         uuid.NewString(),
		Type:       toEntMenuType(menu.Type),
		Path:       menu.Path,
		Title:      menu.Title,
		Permission: menu.Permissions,
		Order:      int(menu.Order),
		Icon:       menu.Icon,
		IsSystem:   false,
		Status:     toEntMenuStatus(menu.Status),
		ParentID:   menu.ParentId,
	})
	if err != nil {
		return ErrSystem(err)
	}
	s.cache.Clear()
	return nil
}

func (s *SystemUsecase) UpdateMenu(ctx context.Context, menu *v1.AdminEditPermissionsRequest) error {
	_, err := s.sys.UpdateMenu(ctx, &gen.Menu{
		ID:         menu.MenuId,
		Path:       menu.Path,
		Title:      menu.Title,
		Permission: menu.Permissions,
		Order:      int(menu.Order),
		Icon:       menu.Icon,
		Status:     toEntMenuStatus(menu.Status),
	})
	if err != nil {
		return ErrSystem(err)
	}
	s.cache.Clear()
	return nil
}

func (s *SystemUsecase) DeleteMenu(ctx context.Context, menuId string) error {
	err := s.sys.DeleteMenu(ctx, menuId)
	if err != nil {
		return ErrSystem(err)
	}
	s.cache.Clear()
	return nil
}

func (s *SystemUsecase) GetAllRoles(ctx context.Context, req *v1.AdminRolesRequest) ([]*v1.RoleInfo, int64, error) {
	if req.Page < 0 {
		req.Page = 1
	}
	if req.PageSize > 500 {
		req.PageSize = 500
	}
	var status *role.Status
	if req.Status != nil {
		status = new(toEntRoleStatus(*req.Status))
	}
	roleInfos, total, err := s.sys.GetRoles(ctx, req.Page, req.PageSize, status, req.Name)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}
	roles := make([]*v1.RoleInfo, 0, len(roleInfos))
	for _, roleInfo := range roleInfos {
		roles = append(roles, toRoleInfo(roleInfo))
	}
	return roles, total, nil
}

func (s *SystemUsecase) GetRole(ctx context.Context, roleId string) (*v1.RoleInfo, error) {
	roleInfo, err := s.sys.GetRole(ctx, roleId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toRoleInfo(roleInfo), nil
}

func (s *SystemUsecase) AddRole(ctx context.Context, req *v1.AdminAddRoleRequest) (*v1.RoleInfo, error) {
	roleInfo, err := s.sys.CreateRole(ctx, &gen.Role{
		ID:          fmt.Sprintf("role_z:%06d_%s", time.Now().Unix()%1000000, uuid.NewString()[:8]),
		Name:        req.Name,
		Description: req.Description,
		IsBuiltin:   false,
		Status:      toEntRoleStatus(req.Status),
	}, req.MenuIds)
	if err != nil {
		return nil, ErrSystem(err)
	}
	s.cache.Clear()
	return toRoleInfo(roleInfo), nil
}

func (s *SystemUsecase) UpdateRole(ctx context.Context, req *v1.AdminEditRoleRequest) (*v1.RoleInfo, error) {
	roleInfo, err := s.sys.UpdateRole(ctx, &gen.Role{
		ID:          req.RoleId,
		Name:        req.Name,
		Description: req.Description,
		Status:      toEntRoleStatus(req.Status),
	}, req.MenuIds)
	if err != nil {
		return nil, ErrSystem(err)
	}
	s.cache.Clear()
	return toRoleInfo(roleInfo), nil
}

func (s *SystemUsecase) DeleteRole(ctx context.Context, roleIds []string) error {
	err := s.sys.DeleteRole(ctx, roleIds)
	if err != nil {
		return ErrSystem(err)
	}
	s.cache.Clear()
	return nil
}

func toMenuInfos(menus []*gen.Menu) ([]*v1.MenuInfo, []string) {
	menuInfos := make([]*v1.MenuInfo, 0)
	permissions := make([]string, 0)

	menuInfoMap := make(map[string]*v1.MenuInfo, len(menus))
	permissionSet := make(map[string]struct{}, len(menus))

	for _, item := range menus {
		if item.Permission != "" &&
			item.Status == menu.StatusActive {
			if _, ok := permissionSet[item.Permission]; !ok {
				permissionSet[item.Permission] = struct{}{}
				permissions = append(permissions, item.Permission)
			}
		}
		menuInfoMap[item.ID] = toMenuInfo(item)
	}

	for _, item := range menuInfoMap {
		if item.ParentId == "" { // 没有上级表示顶层
			menuInfos = append(menuInfos, item)
			continue
		}
		parent, ok := menuInfoMap[item.ParentId]
		if !ok {
			continue
		}
		parent.Children = append(parent.Children, item)
	}

	return menuInfos, permissions
}

func toPermissions(menus []*gen.Menu) map[constant.Permissions]struct{} {
	permissions := make(map[constant.Permissions]struct{})
	for _, item := range menus {
		if item.Permission == "" ||
			item.Status != menu.StatusActive {
			continue
		}
		permissions[constant.Permissions(item.Permission)] = struct{}{}
	}
	return permissions
}

func toMenuInfo(data *gen.Menu) *v1.MenuInfo {
	return &v1.MenuInfo{
		Id:          data.ID,
		Icon:        data.Icon,
		IsSystem:    data.IsSystem,
		Order:       int32(data.Order),
		ParentId:    data.ParentID,
		Path:        data.Path,
		Status:      toMenuStatus(data.Status),
		Title:       data.Title,
		Type:        toMenuType(data.Type),
		Permissions: data.Permission,
		CreateTime:  timestamppb.New(data.CreateTime),
		UpdateTime:  timestamppb.New(data.UpdateTime),
		Children:    make([]*v1.MenuInfo, 0),
	}
}

func toRoleInfo(data *gen.Role) *v1.RoleInfo {
	info := &v1.RoleInfo{
		RoleId:      data.ID,
		Description: data.Description,
		IsBuiltin:   data.IsBuiltin,
		Name:        data.Name,
		Status:      toRoleStatus(data.Status),
		CreateTime:  timestamppb.New(data.CreateTime),
		UpdateTime:  timestamppb.New(data.UpdateTime),
		MenuIds:     make([]string, 0, len(data.Edges.Menus)),
	}
	for _, menuInfo := range data.Edges.Menus {
		info.MenuIds = append(info.MenuIds, menuInfo.ID)
	}
	return info
}

func toMenuStatus(data menu.Status) v1.MenuStatus {
	switch data {
	case menu.StatusActive:
		return v1.MenuStatus_MenuStatus_Active
	case menu.StatusInactive:
		return v1.MenuStatus_MenuStatus_InActive
	default:
		return v1.MenuStatus_MenuStatus_InActive
	}
}

func toEntMenuStatus(data v1.MenuStatus) menu.Status {
	switch data {
	case v1.MenuStatus_MenuStatus_Active:
		return menu.StatusActive
	case v1.MenuStatus_MenuStatus_InActive:
		return menu.StatusInactive
	default:
		return menu.StatusInactive
	}
}

func toMenuType(data menu.Type) v1.MenuType {
	switch data {
	case menu.TypeDirectory:
		return v1.MenuType_MenuType_Directory
	case menu.TypeMenu:
		return v1.MenuType_MenuType_Menu
	case menu.TypeButton:
		return v1.MenuType_MenuType_Button
	default:
		return v1.MenuType_MenuType_Menu
	}
}

func toEntMenuType(data v1.MenuType) menu.Type {
	switch data {
	case v1.MenuType_MenuType_Directory:
		return menu.TypeDirectory
	case v1.MenuType_MenuType_Menu:
		return menu.TypeMenu
	case v1.MenuType_MenuType_Button:
		return menu.TypeButton
	default:
		return menu.TypeMenu
	}
}

func toRoleStatus(data role.Status) v1.RoleStatus {
	switch data {
	case role.StatusActive:
		return v1.RoleStatus_RoleStatus_Active
	case role.StatusInactive:
		return v1.RoleStatus_RoleStatus_InActive
	default:
		return v1.RoleStatus_RoleStatus_InActive
	}
}

func toEntRoleStatus(data v1.RoleStatus) role.Status {
	switch data {
	case v1.RoleStatus_RoleStatus_Active:
		return role.StatusActive
	case v1.RoleStatus_RoleStatus_InActive:
		return role.StatusInactive
	default:
		return role.StatusInactive
	}
}

func toSystemOverviewResponse(data *sysinfo.Overview) *v1.SystemOverviewResponse {
	interfaces := make([]*v1.NetworkInterfaceOverview, 0, len(data.NetworkInterfaces))
	for _, item := range data.NetworkInterfaces {
		interfaces = append(interfaces, toNetworkInterfaceOverview(item))
	}
	partitions := make([]*v1.DiskPartitionOverview, 0, len(data.DiskPartitions))
	for _, item := range data.DiskPartitions {
		partitions = append(partitions, toDiskPartitionOverview(item))
	}
	diskIOs := make([]*v1.DiskIOOverview, 0, len(data.DiskIOs))
	for _, item := range data.DiskIOs {
		diskIOs = append(diskIOs, toDiskIOOverview(item))
	}
	return &v1.SystemOverviewResponse{
		Version: &v1.SystemVersionInfo{
			Hostname:        data.Version.Hostname,
			Os:              data.Version.OS,
			Platform:        data.Version.Platform,
			PlatformFamily:  data.Version.PlatformFamily,
			PlatformVersion: data.Version.PlatformVersion,
			KernelVersion:   data.Version.KernelVersion,
			KernelArch:      data.Version.KernelArch,
		},
		BootTime:      timestamppb.New(data.BootTime),
		UptimeSeconds: data.UptimeSeconds,
		Cpu: &v1.CpuOverview{
			LogicalCount:  int32(data.CPU.LogicalCount),
			PhysicalCount: int32(data.CPU.PhysicalCount),
			ModelName:     data.CPU.ModelName,
		},
		Memory: &v1.MemoryOverview{
			PhysicalMemory: toPhysicalMemoryOverview(data.Memory.PhysicalMemory),
			VirtualMemory:  toVirtualMemoryOverview(data.Memory.VirtualMemory),
		},
		NetworkInterfaces: interfaces,
		DiskPartitions:    partitions,
		DiskIos:           diskIOs,
		SampleTime:        timestamppb.New(data.SampleTime),
	}
}

func toCPUStatus(data sysinfo.CPUStatus) *v1.CpuStatus {
	cores := make([]*v1.CpuCoreStatus, 0, len(data.Cores))
	for _, item := range data.Cores {
		cores = append(cores, &v1.CpuCoreStatus{
			CoreId:  int32(item.CoreID),
			Percent: item.Percent,
		})
	}
	return &v1.CpuStatus{
		TotalPercent:  data.TotalPercent,
		Cores:         cores,
		LogicalCount:  int32(data.LogicalCount),
		PhysicalCount: int32(data.PhysicalCount),
	}
}

func toMemoryStatus(data sysinfo.MemoryStatus) *v1.MemoryStatus {
	return &v1.MemoryStatus{
		PhysicalMemory: toPhysicalMemoryStatus(data.PhysicalMemory),
		VirtualMemory:  toVirtualMemoryStatus(data.VirtualMemory),
	}
}

func toPhysicalMemoryOverview(data sysinfo.PhysicalMemoryOverview) *v1.PhysicalMemoryOverview {
	return &v1.PhysicalMemoryOverview{
		TotalBytes: data.TotalBytes,
	}
}

func toPhysicalMemoryStatus(data sysinfo.PhysicalMemoryStatus) *v1.PhysicalMemoryStatus {
	return &v1.PhysicalMemoryStatus{
		TotalBytes:     data.TotalBytes,
		AvailableBytes: data.AvailableBytes,
		UsedBytes:      data.UsedBytes,
		FreeBytes:      data.FreeBytes,
		UsedPercent:    data.UsedPercent,
	}
}

func toVirtualMemoryOverview(data sysinfo.VirtualMemoryOverview) *v1.VirtualMemoryOverview {
	return &v1.VirtualMemoryOverview{
		TotalBytes: data.TotalBytes,
	}
}

func toVirtualMemoryStatus(data sysinfo.VirtualMemoryStatus) *v1.VirtualMemoryStatus {
	return &v1.VirtualMemoryStatus{
		TotalBytes:        data.TotalBytes,
		UsedBytes:         data.UsedBytes,
		FreeBytes:         data.FreeBytes,
		UsedPercent:       data.UsedPercent,
		SwapInBytes:       data.SwapInBytes,
		SwapOutBytes:      data.SwapOutBytes,
		PageInCount:       data.PageInCount,
		PageOutCount:      data.PageOutCount,
		PageFaultCount:    data.PageFaultCount,
		PageMajFaultCount: data.PageMajFaultCount,
	}
}

func toNetworkInterfaceOverview(data sysinfo.NetworkInterfaceOverview) *v1.NetworkInterfaceOverview {
	return &v1.NetworkInterfaceOverview{
		Name:         data.Name,
		HardwareAddr: data.HardwareAddr,
		Mtu:          int32(data.MTU),
		Flags:        data.Flags,
		Addrs:        data.Addrs,
		IsUp:         data.IsUp,
		IsLoopback:   data.IsLoopback,
	}
}

func toNetworkStatus(data sysinfo.NetworkStatus) *v1.NetworkStatus {
	interfaces := make([]*v1.NetworkInterfaceStatus, 0, len(data.Interfaces))
	for _, item := range data.Interfaces {
		interfaces = append(interfaces, toNetworkInterfaceStatus(item))
	}
	return &v1.NetworkStatus{
		Total:             toNetworkInterfaceStatus(data.Total),
		Interfaces:        interfaces,
		SelectedInterface: toNetworkInterfaceStatus(data.SelectedInterface),
		Connections: &v1.NetworkConnectionStatus{
			Supported:   data.Connections.Supported,
			Error:       data.Connections.Error,
			TcpCount:    data.Connections.TCPCount,
			UdpCount:    data.Connections.UDPCount,
			TotalCount:  data.Connections.TotalCount,
			TcpStatuses: data.Connections.TCPStatuses,
		},
	}
}

func toNetworkInterfaceStatus(data *sysinfo.NetworkInterfaceStatus) *v1.NetworkInterfaceStatus {
	return &v1.NetworkInterfaceStatus{
		Name:                       data.Name,
		HardwareAddr:               data.HardwareAddr,
		Mtu:                        int32(data.MTU),
		Flags:                      data.Flags,
		Addrs:                      data.Addrs,
		IsUp:                       data.IsUp,
		IsLoopback:                 data.IsLoopback,
		BytesSent:                  data.BytesSent,
		BytesRecv:                  data.BytesRecv,
		PacketsSent:                data.PacketsSent,
		PacketsRecv:                data.PacketsRecv,
		ErrIn:                      data.ErrIn,
		ErrOut:                     data.ErrOut,
		DropIn:                     data.DropIn,
		DropOut:                    data.DropOut,
		UploadRateBytesPerSecond:   data.UploadRateBytesPerSecond,
		DownloadRateBytesPerSecond: data.DownloadRateBytesPerSecond,
	}
}

func toDiskPartitionOverview(data sysinfo.DiskPartitionOverview) *v1.DiskPartitionOverview {
	return &v1.DiskPartitionOverview{
		Device:     data.Device,
		Mountpoint: data.Mountpoint,
		Fstype:     data.Fstype,
		Opts:       data.Opts,
	}
}

func toDiskIOOverview(data sysinfo.DiskIOOverview) *v1.DiskIOOverview {
	return &v1.DiskIOOverview{
		Name:         data.Name,
		SerialNumber: data.SerialNumber,
		Label:        data.Label,
	}
}

func toDiskStatus(data sysinfo.DiskStatus) *v1.DiskStatus {
	partitions := make([]*v1.DiskPartitionStatus, 0, len(data.Partitions))
	for _, item := range data.Partitions {
		partitions = append(partitions, toDiskPartitionStatus(item))
	}
	ios := make([]*v1.DiskIOStatus, 0, len(data.IOs))
	for _, item := range data.IOs {
		ios = append(ios, toDiskIOStatus(item))
	}
	return &v1.DiskStatus{
		Total:             toDiskPartitionStatus(data.Total),
		Partitions:        partitions,
		SelectedPartition: toDiskPartitionStatus(data.SelectedPartition),
		TotalIo:           toDiskIOStatus(data.TotalIO),
		Ios:               ios,
		SelectedIo:        toDiskIOStatus(data.SelectedIO),
		IoSupported:       data.IOSupported,
		IoError:           data.IOError,
	}
}

func toDiskPartitionStatus(data *sysinfo.DiskPartitionStatus) *v1.DiskPartitionStatus {
	return &v1.DiskPartitionStatus{
		Device:            data.Device,
		Mountpoint:        data.Mountpoint,
		Fstype:            data.Fstype,
		Supported:         data.Supported,
		Error:             data.Error,
		TotalBytes:        data.TotalBytes,
		FreeBytes:         data.FreeBytes,
		UsedBytes:         data.UsedBytes,
		UsedPercent:       data.UsedPercent,
		InodesTotal:       data.InodesTotal,
		InodesUsed:        data.InodesUsed,
		InodesFree:        data.InodesFree,
		InodesUsedPercent: data.InodesUsedPercent,
	}
}

func toDiskIOStatus(data *sysinfo.DiskIOStatus) *v1.DiskIOStatus {
	return &v1.DiskIOStatus{
		Name:                    data.Name,
		SerialNumber:            data.SerialNumber,
		Label:                   data.Label,
		ReadCount:               data.ReadCount,
		WriteCount:              data.WriteCount,
		ReadBytes:               data.ReadBytes,
		WriteBytes:              data.WriteBytes,
		ReadRateBytesPerSecond:  data.ReadRateBytesPerSecond,
		WriteRateBytesPerSecond: data.WriteRateBytesPerSecond,
		IopsInProgress:          data.IopsInProgress,
		IoTime:                  data.IOTime,
	}
}
