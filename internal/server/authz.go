package server

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
)

// operationWritePermissions 登记“写/管理类”操作所需的更高权限。
// 未在此登记、但落在 servicePrefixPermissions 前缀下的操作，回退到该服务的默认（查看）权限，
// 因此即使将来新增方法忘记登记，也不会退化成“仅需登录”——fail-safe。
var operationWritePermissions = map[string]constant.Permissions{
	// —— Docker 容器（管理）——
	v1.OperationDockerManagerCreateDockerContainer:   constant.DockerContainerManage,
	v1.OperationDockerManagerUpdateDockerContainer:   constant.DockerContainerManage,
	v1.OperationDockerManagerRecreateDockerContainer: constant.DockerContainerManage,
	v1.OperationDockerManagerStartDockerContainer:    constant.DockerContainerManage,
	v1.OperationDockerManagerStopDockerContainer:     constant.DockerContainerManage,
	v1.OperationDockerManagerRestartDockerContainer:  constant.DockerContainerManage,
	v1.OperationDockerManagerKillDockerContainer:     constant.DockerContainerManage,
	v1.OperationDockerManagerPauseDockerContainer:    constant.DockerContainerManage,
	v1.OperationDockerManagerUnpauseDockerContainer:  constant.DockerContainerManage,
	v1.OperationDockerManagerRenameDockerContainer:   constant.DockerContainerManage,
	v1.OperationDockerManagerDeleteDockerContainer:   constant.DockerContainerManage,
	// —— Docker 镜像（管理）——
	v1.OperationDockerManagerPullDockerImage:       constant.DockerImageManage,
	v1.OperationDockerManagerUpdateDockerImageTags: constant.DockerImageManage,
	v1.OperationDockerManagerTagDockerImage:        constant.DockerImageManage,
	v1.OperationDockerManagerDeleteDockerImage:     constant.DockerImageManage,
	// —— Docker 网络（管理）——
	v1.OperationDockerManagerCreateDockerNetwork:     constant.DockerNetworkManage,
	v1.OperationDockerManagerUpdateDockerNetwork:     constant.DockerNetworkManage,
	v1.OperationDockerManagerRecreateDockerNetwork:   constant.DockerNetworkManage,
	v1.OperationDockerManagerDeleteDockerNetwork:     constant.DockerNetworkManage,
	v1.OperationDockerManagerConnectDockerNetwork:    constant.DockerNetworkManage,
	v1.OperationDockerManagerDisconnectDockerNetwork: constant.DockerNetworkManage,
	// —— Docker 配置（编辑）——
	v1.OperationDockerManagerUpdateDockerConfig: constant.DockerConfigEdit,
	v1.OperationDockerManagerTestDockerConfig:   constant.DockerConfigEdit,
	// —— 文件分享（管理）——
	v1.OperationFileManagerCreateShare: constant.FileShare,
	v1.OperationFileManagerListShares:  constant.FileShare,
	v1.OperationFileManagerUpdateShare: constant.FileShare,
	v1.OperationFileManagerDeleteShare: constant.FileShare,

	// —— 文件来源（管理）——
	// 列表(ListFileSources)不登记：回退到 FileManager 默认(constant.Terminal)，
	// 使文件浏览器的来源下拉对所有有文件管理权限的用户可用；增删改/测试需文件来源管理权限。
	v1.OperationFileManagerCreateFileSource: constant.FileSource,
	v1.OperationFileManagerUpdateFileSource: constant.FileSource,
	v1.OperationFileManagerDeleteFileSource: constant.FileSource,
	v1.OperationFileManagerTestFileSource:   constant.FileSource,

	// —— Sub2API（编辑）——
	v1.OperationSub2APIManagerUpdateSub2APIConfig:       constant.Sub2APIEdit,
	v1.OperationSub2APIManagerCreateSub2APIAnnouncement: constant.Sub2APIEdit,
	v1.OperationSub2APIManagerUpdateSub2APIAnnouncement: constant.Sub2APIEdit,
	v1.OperationSub2APIManagerDeleteSub2APIAnnouncement: constant.Sub2APIEdit,
	v1.OperationSub2APIManagerCreateSub2APITimelineItem: constant.Sub2APIEdit,
	v1.OperationSub2APIManagerUpdateSub2APITimelineItem: constant.Sub2APIEdit,
	v1.OperationSub2APIManagerDeleteSub2APITimelineItem: constant.Sub2APIEdit,

	// —— 用户管理 ——
	v1.OperationUserServiceListUser:   constant.UserView,
	v1.OperationUserServiceUserInfo:   constant.UserView,
	v1.OperationUserServiceAddUser:    constant.UserAdd,
	v1.OperationUserServiceEditUser:   constant.UserEdit,
	v1.OperationUserServiceDeleteUser: constant.UserDelete,

	// —— 系统：菜单/权限 ——
	v1.OperationSystemAdminPermissions:       constant.MenuView,
	v1.OperationSystemAdminPermissionsInfo:   constant.MenuView,
	v1.OperationSystemAdminAddPermissions:    constant.MenuAdd,
	v1.OperationSystemAdminEditPermissions:   constant.MenuEdit,
	v1.OperationSystemAdminDeletePermissions: constant.MenuDelete,
	// —— 系统：角色 ——
	v1.OperationSystemAdminRoles:      constant.RoleView,
	v1.OperationSystemAdminRole:       constant.RoleView,
	v1.OperationSystemAdminAddRole:    constant.RoleAdd,
	v1.OperationSystemAdminEditRole:   constant.RoleEdit,
	v1.OperationSystemAdminDeleteRole: constant.RoleDelete,
	// —— 系统：配置 / 邮件 / 操作日志 ——
	v1.OperationSystemUpdateLoginConfig:   constant.SystemConfigEdit,
	v1.OperationSystemEmailConfig:         constant.SystemConfigView,
	v1.OperationSystemUpdateEmailConfig:   constant.SystemConfigEdit,
	v1.OperationSystemEmailTemplate:       constant.SystemConfigView,
	v1.OperationSystemUpdateEmailTemplate: constant.SystemConfigEdit,
	v1.OperationSystemTestEmailConfig:     constant.SystemConfigEdit,
	v1.OperationSystemListOperationLogs:   constant.SystemConfigView,
}

// servicePrefixPermissions 为高危服务设定默认（最低）权限，按操作名前缀匹配。
// 这些服务此前仅校验登录、不校验权限，这里统一兜底。
var servicePrefixPermissions = []struct {
	prefix string
	perm   constant.Permissions
}{
	{"/v1.DockerManager/", constant.DockerView},
	{"/v1.FileManager/", constant.Terminal},
	{"/v1.TaskManager/", constant.TaskManage},
	{"/v1.InstanceManager/", constant.Instance},
	{"/v1.NetworkManager/", constant.Network},
	{"/v1.TunnelManager/", constant.Tunnel},
	{"/v1.Sub2APIManager/", constant.Sub2APIView},
}

// requiredPermission 返回某操作所需的权限；ok=false 表示该操作无需特定权限（仅需登录的自助类操作）。
func requiredPermission(operation string) (constant.Permissions, bool) {
	if perm, ok := operationWritePermissions[operation]; ok {
		return perm, true
	}
	for _, sp := range servicePrefixPermissions {
		if strings.HasPrefix(operation, sp.prefix) {
			return sp.perm, true
		}
	}
	return "", false
}

// wsRoutePermissions 为绕过 Kratos 中间件的原始 WebSocket 路由按路径设定所需权限。
// 这些 handler 经 srv.Handle 注册，不走 operation 中间件，故在 HTTP filter 内单独校验。
var wsRoutePermissions = map[string]constant.Permissions{
	biz.DockerContainerExecWSPath: constant.DockerContainerManage,
	biz.DockerContainerLogsWSPath: constant.DockerView,
	biz.DockerTaskWSPath:          constant.DockerView,
	biz.InstanceWsPath:            constant.Instance,
}

// PermissionMiddleware 对已认证请求按操作所需权限做集中式鉴权（功能级越权防护）。
// 公开/未认证路由没有 auth 上下文，直接放行，由各自 handler 自校验（如 sub2api token、签名链接）。
func (a *Authorization) PermissionMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// 无认证上下文 = 公开路由（login/register/initialize/public/*），跳过权限校验。
			if _, ok := auth.FromContext(ctx); !ok {
				return handler(ctx, req)
			}
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			if perm, need := requiredPermission(tr.Operation()); need {
				if err := a.system.Check(ctx, perm); err != nil {
					return nil, err
				}
			}
			return handler(ctx, req)
		}
	}
}

// checkWSPermission 校验原始 WS 路由所需权限，由 HTTP filter 在认证成功后调用。
// 返回 nil 表示无需特定权限或校验通过。
func (a *Authorization) checkWSPermission(ctx context.Context, path string) error {
	if perm, ok := wsRoutePermissions[path]; ok {
		return a.system.Check(ctx, perm)
	}
	return nil
}
