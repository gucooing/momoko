package service

import (
	"context"
	"momoko/pkg/utils"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
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
			reply, err = handler(ctx, req)
			m.write(ctx, req, reply, err, time.Since(start))
			return reply, err
		}
	}
}

func (m *OperationLogMiddleware) write(ctx context.Context, req any, reply any, handlerErr error, duration time.Duration) {
	httpReq, ok := httptransport.RequestFromServerContext(ctx)
	tr, ok2 := transport.FromServerContext(ctx)
	if !ok || !ok2 || m == nil || m.uc == nil {
		return
	}

	operation := tr.Operation()
	operationType, ok := toOperationType(operation)
	if !ok {
		return
	}

	detail := utils.Detail{
		Method:     httpReq.Method,
		Path:       httpReq.URL.Path,
		Operation:  operation,
		Request:    req,
		Success:    handlerErr == nil,
		DurationMS: duration.Milliseconds(),
	}

	entry := &v1.OperationLogInfo{
		UserId:        operationUserID(ctx, operation, reply, handlerErr),
		OperationType: operationType,
		Success:       handlerErr == nil,
		Detail:        detail.MarshalDetail(),
		Ip:            utils.ClientIP(httpReq),
		UserAgent:     utils.UserAgent(httpReq),
		Path:          httpReq.URL.Path,
		DurationMs:    duration.Milliseconds(),
		OperationTime: timestamppb.New(time.Now()),
	}

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
	v1.OperationAuthServiceDelLogin:                      v1.OperationType_OperationTypeAuthDeviceDelete,
	v1.OperationUserServiceUpdateMe:                      v1.OperationType_OperationTypeUserUpdateMe,
	v1.OperationUserServiceAddUser:                       v1.OperationType_OperationTypeUserCreate,
	v1.OperationUserServiceEditUser:                      v1.OperationType_OperationTypeUserUpdate,
	v1.OperationUserServiceDeleteUser:                    v1.OperationType_OperationTypeUserDelete,
	v1.OperationSystemAdminAddPermissions:                v1.OperationType_OperationTypeSystemPermissionCreate,
	v1.OperationSystemAdminEditPermissions:               v1.OperationType_OperationTypeSystemPermissionUpdate,
	v1.OperationSystemAdminDeletePermissions:             v1.OperationType_OperationTypeSystemPermissionDelete,
	v1.OperationSystemAdminAddRole:                       v1.OperationType_OperationTypeSystemRoleCreate,
	v1.OperationSystemAdminEditRole:                      v1.OperationType_OperationTypeSystemRoleUpdate,
	v1.OperationSystemAdminDeleteRole:                    v1.OperationType_OperationTypeSystemRoleDelete,
	v1.OperationSystemUpdateLoginConfig:                  v1.OperationType_OperationTypeSystemLoginConfigUpdate,
	v1.OperationSystemUpdateEmailConfig:                  v1.OperationType_OperationTypeSystemEmailConfigUpdate,
	v1.OperationSystemUpdateEmailTemplate:                v1.OperationType_OperationTypeSystemEmailTemplateUpdate,
	v1.OperationSystemTestEmailConfig:                    v1.OperationType_OperationTypeSystemEmailConfigTest,
	v1.OperationFileManagerCreateFileSystem:              v1.OperationType_OperationTypeFileCreate,
	v1.OperationFileManagerRenameFileSystem:              v1.OperationType_OperationTypeFileRename,
	v1.OperationFileManagerCopyFileSystem:                v1.OperationType_OperationTypeFileCopy,
	v1.OperationFileManagerCutFileSystem:                 v1.OperationType_OperationTypeFileMove,
	v1.OperationFileManagerBatchDeleteFileSystem:         v1.OperationType_OperationTypeFileDelete,
	v1.OperationFileManagerBatchCompressFileSystem:       v1.OperationType_OperationTypeFileCompress,
	v1.OperationFileManagerUnzipFileSystem:               v1.OperationType_OperationTypeFileDecompress,
	v1.OperationFileManagerCompleteFileUpload:            v1.OperationType_OperationTypeFileUploadComplete,
	v1.OperationFileManagerCancelFileUpload:              v1.OperationType_OperationTypeFileUploadCancel,
	v1.OperationInstanceManagerCreateInstanceType:        v1.OperationType_OperationTypeInstanceTypeCreate,
	v1.OperationInstanceManagerUpdateInstanceType:        v1.OperationType_OperationTypeInstanceTypeUpdate,
	v1.OperationInstanceManagerDelInstanceType:           v1.OperationType_OperationTypeInstanceTypeDelete,
	v1.OperationInstanceManagerStartTerminal:             v1.OperationType_OperationTypeInstanceTerminalStart,
	v1.OperationInstanceManagerStopTerminal:              v1.OperationType_OperationTypeInstanceTerminalStop,
	v1.OperationInstanceManagerRestartTerminal:           v1.OperationType_OperationTypeInstanceTerminalRestart,
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
	v1.OperationInstanceManagerInstanceFilePreSignUpload: v1.OperationType_OperationTypeInstanceFileUploadPreSign,
	v1.OperationOpenSSHManagerCreateSSHHost:              v1.OperationType_OperationTypeSSHHostCreate,
	v1.OperationOpenSSHManagerUpdateSSHHost:              v1.OperationType_OperationTypeSSHHostUpdate,
	v1.OperationOpenSSHManagerDeleteSSHHost:              v1.OperationType_OperationTypeSSHHostDelete,
	v1.OperationOpenSSHManagerShareSSHHost:               v1.OperationType_OperationTypeSSHHostShare,
	v1.OperationOpenSSHManagerTestSSHHost:                v1.OperationType_OperationTypeSSHHostTest,
	v1.OperationOpenSSHManagerBatchTestSSHHosts:          v1.OperationType_OperationTypeSSHHostBatchTest,
	v1.OperationNodeServiceCreateAPIKey:                  v1.OperationType_OperationTypeNodeAPIKeyCreate,
	v1.OperationNodeServiceCopyAPIKey:                    v1.OperationType_OperationTypeNodeAPIKeyCopy,
	v1.OperationNodeServiceUpdateAPIKey:                  v1.OperationType_OperationTypeNodeAPIKeyUpdate,
	v1.OperationNodeServiceRefreshAPIKey:                 v1.OperationType_OperationTypeNodeAPIKeyRefresh,
}

func toOperationType(operation string) (v1.OperationType, bool) {
	operationType, ok := keyOperationTypes[operation]
	return operationType, ok
}
