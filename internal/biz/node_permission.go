package biz

import (
	"momoko/pkg/constant"
)

// nodePermissionSpec 描述一项可授予 API Key 的节点权限。
type nodePermissionSpec struct {
	code constant.Permissions
	// rbac 为 true 时该权限在本地走 RBAC：远程调用还要求 API Key 所属用户拥有它（API Key 不能超出所属用户）；
	// 为 false 时本地仅需登录（机器状态、SSH），远程调用只看授予范围。
	rbac bool
}

// nodePermissionSpecs 可授予 API Key 的节点权限，顺序即展示与存储顺序。
var nodePermissionSpecs = []nodePermissionSpec{
	{code: constant.SystemStatus, rbac: false},
	{code: constant.Instance, rbac: true},
	{code: constant.Terminal, rbac: true},
	{code: constant.FileSource, rbac: true},
	{code: constant.DockerView, rbac: true},
	{code: constant.DockerContainerManage, rbac: true},
	{code: constant.DockerImageManage, rbac: true},
	{code: constant.DockerNetworkManage, rbac: true},
	{code: constant.DockerConfigEdit, rbac: true},
	{code: constant.Network, rbac: true},
	{code: constant.Tunnel, rbac: true},
	{code: constant.SSHManage, rbac: false},
	{code: constant.TaskManage, rbac: true},
}

var nodePermissionIndex = func() map[constant.Permissions]nodePermissionSpec {
	index := make(map[constant.Permissions]nodePermissionSpec, len(nodePermissionSpecs))
	for _, spec := range nodePermissionSpecs {
		index[spec.code] = spec
	}
	return index
}()

// IsRBACNodePermission 判断节点权限在本地是否走 RBAC 校验。
func IsRBACNodePermission(perm constant.Permissions) bool {
	return nodePermissionIndex[perm].rbac
}

// normalizeNodePermissions 校验并去重授予范围，按目录顺序返回；未知标识直接拒绝。
func normalizeNodePermissions(perms []string) ([]string, error) {
	selected := make(map[constant.Permissions]struct{}, len(perms))
	for _, perm := range perms {
		code := constant.Permissions(perm)
		if _, ok := nodePermissionIndex[code]; !ok {
			return nil, ErrAPIKeyPermission
		}
		selected[code] = struct{}{}
	}
	normalized := make([]string, 0, len(selected))
	for _, spec := range nodePermissionSpecs {
		if _, ok := selected[spec.code]; ok {
			normalized = append(normalized, string(spec.code))
		}
	}
	return normalized, nil
}
