package server

import (
	"strings"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/constant"
)

// nodeServicePrefixes 是可在远程节点上使用的服务及其默认（最低）权限。
// 服务内的写/管理类操作沿用 operationWritePermissions 登记的更高权限，与本地鉴权保持一致；
// OpenSSH 本地仅需登录，远程需显式授予 ssh:manage。
var nodeServicePrefixes = []struct {
	prefix string
	perm   constant.Permissions
}{
	{"/v1.InstanceManager/", constant.Instance},
	{"/v1.FileManager/", constant.Terminal},
	{"/v1.DockerManager/", constant.DockerView},
	{"/v1.NetworkManager/", constant.Network},
	{"/v1.TunnelManager/", constant.Tunnel},
	{"/v1.TaskManager/", constant.TaskManage},
	{"/v1.OpenSSHManager/", constant.SSHManage},
}

// nodeOperationPermissions 是上述服务之外单独开放给远程节点的操作。
// 空权限表示鉴权通过即可调用（权限在接口内部或更细粒度处校验）。
var nodeOperationPermissions = map[string]constant.Permissions{
	v1.OperationSystemSystemOverview: constant.SystemStatus,
	v1.OperationSystemSystemStatus:   constant.SystemStatus,
	"/v1.NodeAgent/AgentInfo":        "",
	"/v1.NodeAgent/Tunnel":           "",
}

// nodeDeniedOperations 属于远程服务、但依赖本机上下文（公开分享链接、本机用户列表）而不开放的操作。
var nodeDeniedOperations = map[string]struct{}{
	v1.OperationFileManagerCreateShare:        {},
	v1.OperationFileManagerListShares:         {},
	v1.OperationFileManagerUpdateShare:        {},
	v1.OperationFileManagerDeleteShare:        {},
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
	if perm, ok := nodeOperationPermissions[operation]; ok {
		return perm, true
	}
	for _, sp := range nodeServicePrefixes {
		if strings.HasPrefix(operation, sp.prefix) {
			if perm, ok := operationWritePermissions[operation]; ok {
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
	biz.InstanceWsPath:            constant.Instance,
	biz.DockerContainerExecWSPath: constant.DockerContainerManage,
	biz.DockerContainerLogsWSPath: constant.DockerView,
	biz.DockerTaskWSPath:          constant.DockerView,
	biz.OpenSSHWSPath:             constant.SSHManage,
	biz.PreFileDownload:           "",
	biz.PreFileUpload:             "",
}

// isPresignedPath 判断是否为签名自授权的预签名路径（主控侧无需登录态即可转发）。
func isPresignedPath(path string) bool {
	return path == biz.PreFileDownload || path == biz.PreFileUpload
}
