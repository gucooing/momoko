package service

import (
	"context"
	"momoko/pkg/utils"
	"net"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type OperationLogMiddleware struct {
	uc     *biz.OperationLogUsecase
	logger *log.Helper
}

func NewOperationLogMiddleware(uc *biz.OperationLogUsecase, logger log.Logger) *OperationLogMiddleware {
	return &OperationLogMiddleware{
		uc:     uc,
		logger: log.NewHelper(logger),
	}
}

func (m *OperationLogMiddleware) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			start := time.Now()
			ctx, trace := biz.WithNodeTrace(ctx)
			ctx, _ = biz.WithLoginTrace(ctx)
			reply, err = handler(ctx, req)
			m.write(ctx, req, reply, err, time.Since(start), trace)
			return reply, err
		}
	}
}

func (m *OperationLogMiddleware) write(ctx context.Context, req any, reply any, handlerErr error, duration time.Duration, trace *biz.NodeTrace) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok || m == nil || m.uc == nil {
		return
	}

	operation := tr.Operation()
	operationType, ok := toOperationType(operation)
	if !ok {
		return
	}

	detail := utils.Detail{
		Operation:  operation,
		Request:    req,
		Success:    handlerErr == nil,
		Code:       200,
		DurationMS: duration.Milliseconds(),
		Node:       trace.NodeName,
	}
	if handlerErr != nil {
		detail.Error = operationErrorText(handlerErr)
		detail.Code = kerrors.FromError(handlerErr).Code
	}
	entry := &v1.OperationLogInfo{
		UserId:        operationUserID(ctx, operation, reply, handlerErr),
		OperationType: operationType,
		Success:       handlerErr == nil,
		DurationMs:    duration.Milliseconds(),
		OperationTime: timestamppb.New(time.Now()),
	}
	// 登录失败时也记到对应用户名下（登录日志里能看到失败记录）。
	if entry.UserId == nil {
		if lt, ok := biz.LoginTraceFromContext(ctx); ok && lt.UserID != "" {
			entry.UserId = &lt.UserID
		}
	}
	if httpReq, ok := httptransport.RequestFromServerContext(ctx); ok {
		detail.Method = httpReq.Method
		detail.Path = httpReq.URL.Path
		entry.Ip = utils.ClientIP(httpReq)
		entry.UserAgent = httpReq.UserAgent()
		entry.Path = httpReq.URL.Path
	} else {
		// 节点 gRPC 端口：调用方是持 API Key 的远程主控，记录其地址与主控侧操作者。
		detail.Method = "GRPC"
		detail.Path = operation
		entry.Path = operation
		if p, ok := peer.FromContext(ctx); ok {
			if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
				entry.Ip = host
			} else {
				entry.Ip = p.Addr.String()
			}
		}
		if principal, ok := biz.NodePrincipalFromContext(ctx); ok {
			detail.APIKey = principal.APIKeyName
			detail.Operator = principal.Operator
			entry.UserAgent = "momoko-master"
		}
	}
	entry.Detail = detail.MarshalDetail()

	go func() {
		writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.uc.CreateOperationLog(writeCtx, entry); err != nil {
			m.logger.Warnf("create operation log failed: %v", err)
		}
	}()
}

func operationUserID(ctx context.Context, operation string, reply any, handlerErr error) *string {
	if userID := auth.GetUserIDFromContext(ctx); userID != nil && *userID != "" {
		return userID
	}
	if handlerErr != nil {
		return nil
	}

	switch operation {
	case v1.OperationAuthServiceLogin:
		loginReply, ok := reply.(*v1.LoginResponse)
		if !ok || loginReply.GetAccessToken() == "" {
			return nil
		}
		claims, err := auth.ParseToken(loginReply.GetAccessToken())
		if err != nil || claims.UserID == "" {
			return nil
		}
		return new(claims.UserID)
	case v1.OperationAuthServiceRegister:
		registerReply, ok := reply.(*v1.RegisterResponse)
		if !ok || registerReply.GetUserId() == "" {
			return nil
		}
		return new(registerReply.GetUserId())
	default:
		return nil
	}
}

var keyOperationTypes = map[string]v1.OperationType{
	v1.OperationAuthServiceLogin:                         v1.OperationType_OperationTypeAuthLogin,
	v1.OperationAuthServiceRegister:                      v1.OperationType_OperationTypeAuthRegister,
	v1.OperationAuthServiceSendRegisterEmailCode:         v1.OperationType_OperationTypeAuthRegisterEmailCode,
	v1.OperationAuthServiceSendLoginEmailCode:            v1.OperationType_OperationTypeAuthLoginEmailCode,
	v1.OperationAuthServiceLogout:                        v1.OperationType_OperationTypeAuthLogout,
	v1.OperationAuthServiceUpdatePassword:                v1.OperationType_OperationTypeAuthUpdatePassword,
	v1.OperationUserServiceUpdateMe:                      v1.OperationType_OperationTypeUserUpdateMe,
	v1.OperationUserServiceAddUser:                       v1.OperationType_OperationTypeUserCreate,
	v1.OperationUserServiceEditUser:                      v1.OperationType_OperationTypeUserUpdate,
	v1.OperationUserServiceDeleteUser:                    v1.OperationType_OperationTypeUserDelete,
	v1.OperationUserServiceBatchUpdateUser:               v1.OperationType_OperationTypeUserUpdate,
	v1.OperationUserServiceResetUserPassword:             v1.OperationType_OperationTypeUserUpdate,
	v1.OperationSystemAdminAddPermissions:                v1.OperationType_OperationTypeSystemPermissionCreate,
	v1.OperationSystemAdminEditPermissions:               v1.OperationType_OperationTypeSystemPermissionUpdate,
	v1.OperationSystemAdminDeletePermissions:             v1.OperationType_OperationTypeSystemPermissionDelete,
	v1.OperationSystemAdminAddRole:                       v1.OperationType_OperationTypeSystemRoleCreate,
	v1.OperationSystemAdminEditRole:                      v1.OperationType_OperationTypeSystemRoleUpdate,
	v1.OperationSystemAdminDeleteRole:                    v1.OperationType_OperationTypeSystemRoleDelete,
	v1.OperationSystemUpdateLoginConfig:                  v1.OperationType_OperationTypeSystemLoginConfigUpdate,
	v1.OperationSystemUpdateSiteConfig:                   v1.OperationType_OperationTypeSystemSiteConfigUpdate,
	v1.OperationThemeServiceSetSiteTheme:                 v1.OperationType_OperationTypeSystemSiteConfigUpdate,
	v1.OperationSystemUpdateEmailConfig:                  v1.OperationType_OperationTypeSystemEmailConfigUpdate,
	v1.OperationSystemUpdateEmailTemplate:                v1.OperationType_OperationTypeSystemEmailTemplateUpdate,
	v1.OperationSystemTestEmailConfig:                    v1.OperationType_OperationTypeSystemEmailConfigTest,
	v1.OperationSystemUpdateOperationLogRetention:        v1.OperationType_OperationTypeSystemOperationLogRetentionUpdate,
	v1.OperationSystemUpdateStorageConfig:                v1.OperationType_OperationTypeSystemStorageConfigUpdate,
	v1.OperationFileManagerCreateFileSystem:              v1.OperationType_OperationTypeFileCreate,
	v1.OperationFileManagerRenameFileSystem:              v1.OperationType_OperationTypeFileRename,
	v1.OperationFileManagerCopyFileSystem:                v1.OperationType_OperationTypeFileCopy,
	v1.OperationFileManagerCutFileSystem:                 v1.OperationType_OperationTypeFileMove,
	v1.OperationFileManagerBatchDeleteFileSystem:         v1.OperationType_OperationTypeFileDelete,
	v1.OperationFileManagerBatchCompressFileSystem:       v1.OperationType_OperationTypeFileCompress,
	v1.OperationFileManagerUnzipFileSystem:               v1.OperationType_OperationTypeFileDecompress,
	v1.OperationFileManagerEditFileSystemFile:            v1.OperationType_OperationTypeFileEdit,
	v1.OperationFileManagerCompleteFileUpload:            v1.OperationType_OperationTypeFileUploadComplete,
	v1.OperationFileManagerCancelFileUpload:              v1.OperationType_OperationTypeFileUploadCancel,
	v1.OperationInstanceManagerCreateInstanceType:        v1.OperationType_OperationTypeInstanceTypeCreate,
	v1.OperationInstanceManagerUpdateInstanceType:        v1.OperationType_OperationTypeInstanceTypeUpdate,
	v1.OperationInstanceManagerDelInstanceType:           v1.OperationType_OperationTypeInstanceTypeDelete,
	v1.OperationInstanceManagerCreateInstance:            v1.OperationType_OperationTypeInstanceCreate,
	v1.OperationInstanceManagerStartInstance:             v1.OperationType_OperationTypeInstanceStart,
	v1.OperationInstanceManagerStopInstance:              v1.OperationType_OperationTypeInstanceStop,
	v1.OperationInstanceManagerRestartInstance:           v1.OperationType_OperationTypeInstanceRestart,
	v1.OperationInstanceManagerDelInstance:               v1.OperationType_OperationTypeInstanceDelete,
	v1.OperationInstanceManagerUpdateInstance:            v1.OperationType_OperationTypeInstanceUpdate,
	v1.OperationInstanceManagerDelInstanceLog:            v1.OperationType_OperationTypeInstanceLogDelete,
	v1.OperationInstanceManagerCreateInstanceFile:        v1.OperationType_OperationTypeInstanceFileCreate,
	v1.OperationInstanceManagerRenameInstanceFile:        v1.OperationType_OperationTypeInstanceFileRename,
	v1.OperationInstanceManagerCopyInstanceFile:          v1.OperationType_OperationTypeInstanceFileCopy,
	v1.OperationInstanceManagerCutInstanceFile:           v1.OperationType_OperationTypeInstanceFileMove,
	v1.OperationInstanceManagerBatchDeleteInstanceFile:   v1.OperationType_OperationTypeInstanceFileDelete,
	v1.OperationInstanceManagerBatchCompressInstanceFile: v1.OperationType_OperationTypeInstanceFileCompress,
	v1.OperationInstanceManagerUnzipInstanceFile:         v1.OperationType_OperationTypeInstanceFileDecompress,
	v1.OperationInstanceManagerEditInstanceFile:          v1.OperationType_OperationTypeInstanceFileEdit,
	v1.OperationInstanceManagerInstanceFilePreSignUpload: v1.OperationType_OperationTypeInstanceFileUploadPreSign,
	v1.OperationOpenSSHManagerCreateSSHHost:              v1.OperationType_OperationTypeSSHHostCreate,
	v1.OperationOpenSSHManagerUpdateSSHHost:              v1.OperationType_OperationTypeSSHHostUpdate,
	v1.OperationOpenSSHManagerDeleteSSHHost:              v1.OperationType_OperationTypeSSHHostDelete,
	v1.OperationOpenSSHManagerShareSSHHost:               v1.OperationType_OperationTypeSSHHostShare,
	v1.OperationOpenSSHManagerTestSSHHost:                v1.OperationType_OperationTypeSSHHostTest,
	v1.OperationNodeServiceCreateAPIKey:                  v1.OperationType_OperationTypeNodeAPIKeyCreate,
	v1.OperationNodeServiceCopyAPIKey:                    v1.OperationType_OperationTypeNodeAPIKeyCopy,
	v1.OperationNodeServiceUpdateAPIKey:                  v1.OperationType_OperationTypeNodeAPIKeyUpdate,
	v1.OperationNodeServiceRefreshAPIKey:                 v1.OperationType_OperationTypeNodeAPIKeyRefresh,
	v1.OperationNodeServiceDeleteAPIKey:                  v1.OperationType_OperationTypeNodeAPIKeyDelete,
	v1.OperationNodeServiceCreateNode:                    v1.OperationType_OperationTypeNodeCreate,
	v1.OperationNodeServiceUpdateNode:                    v1.OperationType_OperationTypeNodeUpdate,
	v1.OperationNodeServiceDeleteNode:                    v1.OperationType_OperationTypeNodeDelete,
	v1.OperationNodeServiceTestNode:                      v1.OperationType_OperationTypeNodeTest,
	v1.OperationDockerManagerUpdateDockerConfig:          v1.OperationType_OperationTypeDockerConfigUpdate,
	v1.OperationDockerManagerTestDockerConfig:            v1.OperationType_OperationTypeDockerConfigTest,
	v1.OperationDockerManagerTestDockerRegistryAuth:      v1.OperationType_OperationTypeDockerRegistryAuthTest,
	v1.OperationDockerManagerCreateDockerContainer:       v1.OperationType_OperationTypeDockerContainerCreate,
	v1.OperationDockerManagerUpdateDockerContainer:       v1.OperationType_OperationTypeDockerContainerUpdate,
	v1.OperationDockerManagerDeleteDockerContainer:       v1.OperationType_OperationTypeDockerContainerDelete,
	v1.OperationDockerManagerStartDockerContainer:        v1.OperationType_OperationTypeDockerContainerStart,
	v1.OperationDockerManagerStopDockerContainer:         v1.OperationType_OperationTypeDockerContainerStop,
	v1.OperationDockerManagerRestartDockerContainer:      v1.OperationType_OperationTypeDockerContainerRestart,
	v1.OperationDockerManagerKillDockerContainer:         v1.OperationType_OperationTypeDockerContainerKill,
	v1.OperationDockerManagerPauseDockerContainer:        v1.OperationType_OperationTypeDockerContainerPause,
	v1.OperationDockerManagerUnpauseDockerContainer:      v1.OperationType_OperationTypeDockerContainerUnpause,
	v1.OperationDockerManagerRenameDockerContainer:       v1.OperationType_OperationTypeDockerContainerRename,
	v1.OperationDockerManagerRecreateDockerContainer:     v1.OperationType_OperationTypeDockerContainerRecreate,
	v1.OperationDockerManagerPullDockerImage:             v1.OperationType_OperationTypeDockerImagePull,
	v1.OperationDockerManagerUpdateDockerImageTags:       v1.OperationType_OperationTypeDockerImageTagsUpdate,
	v1.OperationDockerManagerTagDockerImage:              v1.OperationType_OperationTypeDockerImageTag,
	v1.OperationDockerManagerDeleteDockerImage:           v1.OperationType_OperationTypeDockerImageDelete,
	v1.OperationDockerManagerCreateDockerNetwork:         v1.OperationType_OperationTypeDockerNetworkCreate,
	v1.OperationDockerManagerUpdateDockerNetwork:         v1.OperationType_OperationTypeDockerNetworkUpdate,
	v1.OperationDockerManagerRecreateDockerNetwork:       v1.OperationType_OperationTypeDockerNetworkRecreate,
	v1.OperationDockerManagerDeleteDockerNetwork:         v1.OperationType_OperationTypeDockerNetworkDelete,
	v1.OperationDockerManagerConnectDockerNetwork:        v1.OperationType_OperationTypeDockerNetworkConnect,
	v1.OperationDockerManagerDisconnectDockerNetwork:     v1.OperationType_OperationTypeDockerNetworkDisconnect,
	v1.OperationDockerManagerPruneDockerNetworks:         v1.OperationType_OperationTypeDockerNetworkPrune,
	v1.OperationDockerManagerPruneDockerImages:           v1.OperationType_OperationTypeDockerImageDelete,
	v1.OperationOIDCServiceUpdateOIDCConfig:              v1.OperationType_OperationTypeOIDCConfigUpdate,
	v1.OperationOIDCServiceCreateOIDCClient:              v1.OperationType_OperationTypeOIDCClientCreate,
	v1.OperationOIDCServiceUpdateOIDCClient:              v1.OperationType_OperationTypeOIDCClientUpdate,
	v1.OperationOIDCServiceDeleteOIDCClient:              v1.OperationType_OperationTypeOIDCClientDelete,
	v1.OperationOIDCServiceRefreshOIDCClientSecret:       v1.OperationType_OperationTypeOIDCClientSecretRefresh,
	v1.OperationOIDCServiceCreateOIDCAuthorizationCode:   v1.OperationType_OperationTypeOIDCAuthorizationCodeCreate,
	v1.OperationOIDCServiceRotateOIDCSigningKey:          v1.OperationType_OperationTypeOIDCSigningKeyRotate,
	v1.OperationTaskManagerCancelTasks:                   v1.OperationType_OperationTypeTaskCancel,
	v1.OperationTaskManagerRetryTasks:                    v1.OperationType_OperationTypeTaskRetry,
	v1.OperationTaskManagerDeleteTasks:                   v1.OperationType_OperationTypeTaskDelete,
	v1.OperationTaskManagerClearTasks:                    v1.OperationType_OperationTypeTaskClear,
}

// operationErrorText 取错误里给人看的部分（Kratos 错误取 message，其余取 Error()）。
func operationErrorText(err error) string {
	if se := kerrors.FromError(err); se != nil && se.Message != "" {
		return se.Message
	}
	return err.Error()
}

func toOperationType(operation string) (v1.OperationType, bool) {
	operationType, ok := keyOperationTypes[operation]
	return operationType, ok
}
