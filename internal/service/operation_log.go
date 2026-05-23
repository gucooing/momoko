package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/common"
	"momoko/pkg/tools"
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

	detail := tools.Detail{
		Method:     httpReq.Method,
		Path:       httpReq.URL.Path,
		Operation:  tr.Operation(),
		Request:    req,
		Success:    handlerErr == nil,
		DurationMS: duration.Milliseconds(),
	}

	entry := &v1.OperationLogInfo{
		UserId:        auth.GetUserIDFromContext(ctx),
		OperationType: toOperationType(tr.Operation()),
		Success:       handlerErr == nil,
		Detail:        detail.MarshalDetail(),
		Ip:            tools.ClientIP(httpReq),
		UserAgent:     tools.UserAgent(httpReq),
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

func toOperationType(operation string) string {
	switch operation {
	case v1.OperationAuthServiceLogin: // 登录
		return common.OperationTypeLogin.String()
	default:
		return ""
	}
}
