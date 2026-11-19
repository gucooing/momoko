package server

import (
	"strings"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/service"
	"momoko/pkg/constant"
)

// nodeServicePrefixes 是可在远程节点上使用的服务。所需权限与本机鉴权一致（requiredPermission：
// 服务默认权限，写/管理类操作沿用 operationWritePermissions 的更高权限）；perm 只给本机仅需登录的服务
// 指定远程所需权限（OpenSSH 远程需显式授予 ssh:manage）。
var nodeServicePrefixes = []struct {
	prefix string
	perm   constant.Permissions
}{
	{"/v1.InstanceManager/", ""},
	{"/v1.FileManager/", ""},
	{"/v1.DockerManager/", ""},
	{"/v1.NetworkManager/", ""},
	{"/v1.TunnelManager/", ""},
	{"/v1.TaskManager/", ""},
	{"/v1.OpenSSHManager/", constant.SSHManage},
	{"/v1.MonitorManager/", ""},
	{"/v1.IQCheckManager/", ""},
	{"/v1.GameServerService/", ""},
	{"/v1.GameGMService/", ""},
	{"/v1.gakumas.GakumasGMService/", ""},
	{"/v1.GameOpsService/", ""},
	{"/v1.PassportAdminService/", ""},
	{"/v1.AgentService/", ""},
}

// nodeOperations 是上述服务之外单独开放给远程节点的操作，值为本机仅需登录的操作的远程所需权限；
// 值为空时与本机鉴权一致（本机也没有登记权限的只需鉴权通过，权限在接口内部或更细粒度处校验）。
var nodeOperations = map[string]constant.Permissions{
	v1.OperationSystemSystemOverview: constant.SystemStatus,
	v1.OperationSystemSystemStatus:   constant.SystemStatus,
	"/v1.NodeAgent/AgentInfo":        "",
	"/v1.NodeAgent/Tunnel":           "",
	// 系统设置（常规 / 登录 / 邮件 / 储存）、版本更新与操作日志
	v1.OperationSystemSiteConfig:                  "",
	v1.OperationSystemUpdateSiteConfig:            "",
	v1.OperationSystemLoginConfig:                 "",
	v1.OperationSystemUpdateLoginConfig:           "",
	v1.OperationSystemEmailConfig:                 "",
	v1.OperationSystemUpdateEmailConfig:           "",
	v1.OperationSystemEmailTemplate:               "",
	v1.OperationSystemUpdateEmailTemplate:         "",
	v1.OperationSystemListEmailTemplates:          "",
	v1.OperationSystemTestEmailConfig:             "",
	v1.OperationSystemStorageConfig:               "",
	v1.OperationSystemUpdateStorageConfig:         "",
	v1.OperationSystemCheckUpdate:                 "",
	v1.OperationSystemListOperationLogs:           "",
	v1.OperationSystemOperationLogStats:           "",
	v1.OperationSystemUpdateOperationLogRetention: "",
}

// nodeDeniedOperations 属于远程服务、但不开放的操作：公开分享页的接口由访客直接访问节点自己的站点，
// 共享 SSH 主机依赖本机用户列表。
var nodeDeniedOperations = map[string]struct{}{
	v1.OperationFileManagerGetShareMeta:       {},
	v1.OperationFileManagerListShareDir:       {},
	v1.OperationFileManagerCreateShareSession: {},
	v1.OperationOpenSSHManagerShareSSHHost:    {},
}

// nodeOperationPermission 返回远程调用某操作所需的节点权限；ok=false 表示该操作不对远程开放。
func nodeOperationPermission(operation string) (constant.Permissions, bool) {
	if _, denied := nodeDeniedOperations[operation]; denied {
		return "", false
	}
	if perm, ok := nodeOperations[operation]; ok {
		if perm == "" {
			perm, _ = requiredPermission(operation)
		}
		return perm, true
	}
	for _, sp := range nodeServicePrefixes {
		if strings.HasPrefix(operation, sp.prefix) {
			if perm, ok := requiredPermission(operation); ok {
				return perm, true
			}
			return sp.perm, true
		}
	}
	return "", false
}

// isNodeScopedOperation 判断操作是否属于“按节点区分”的服务：
// 带节点标识却不开放远程的操作要直接拒绝，不能静默落到主控本机执行。
func isNodeScopedOperation(operation string) bool {
	if _, denied := nodeDeniedOperations[operation]; denied {
		return true
	}
	for _, sp := range nodeServicePrefixes {
		if strings.HasPrefix(operation, sp.prefix) {
			return true
		}
	}
	return false
}

// nodeTunnelPaths 是允许经隧道访问的原始 HTTP 路径及所需节点权限。
// 预签名上传/下载由签名本身授权（签名只能经已授权的一元接口获得），不再额外要求权限。
var nodeTunnelPaths = map[string]constant.Permissions{
	biz.GameServerConsoleWsPath:   constant.GameServer,
	biz.InstanceWsPath:            constant.Instance,
	biz.DockerContainerExecWSPath: constant.DockerContainerManage,
	biz.DockerContainerLogsWSPath: constant.DockerView,
	biz.DockerTaskWSPath:          constant.DockerView,
	biz.DockerImageExportPath:     constant.DockerImageManage,
	biz.DockerImageImportPath:     constant.DockerImageManage,
	biz.OpenSSHWSPath:             constant.SSHManage,
	biz.PreFileDownload:           "",
	biz.PreFileUpload:             "",
	biz.AgentWsPath:               constant.AgentUse,
}

// nodeTunnelPrefixes 是允许经隧道访问的原始 HTTP 路径前缀（游戏实时动态 SSE，路径里带服务器 id）。
var nodeTunnelPrefixes = []struct {
	prefix string
	perm   constant.Permissions
}{
	{biz.GameServerEventsPrefix, constant.GameServer},
	{service.GmEventsPrefix, constant.GameGM},
}

// nodeTunnelPermission 返回经隧道访问某原始路径所需的节点权限；ok=false 表示不允许经隧道访问。
func nodeTunnelPermission(path string) (constant.Permissions, bool) {
	if perm, ok := nodeTunnelPaths[path]; ok {
		return perm, true
	}
	for _, tp := range nodeTunnelPrefixes {
		if strings.HasPrefix(path, tp.prefix) {
			return tp.perm, true
		}
	}
	return "", false
}

// isPresignedPath 判断是否为签名自授权的预签名路径（主控侧无需登录态即可转发）。
func isPresignedPath(path string) bool {
	return path == biz.PreFileDownload || path == biz.PreFileUpload
}
