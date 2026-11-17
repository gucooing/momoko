package biz

import (
	"context"
	"strings"
	"time"

	"momoko/internal/data/ent/gen"
)

// 通行证一侧的客户端日志上传（POST /api/v1/game-client-logs，不用通行证令牌，凭游戏服务端签发的票据鉴权）。
// 票据格式与日志的存储规则见 game_client_log.go。

// ClientLogOwner 校验上传票据，返回日志所属的游戏账号。
func (uc *PassportUsecase) ClientLogOwner(ctx context.Context, ticket string) (ClientLogOwner, error) {
	return verifyClientLogTicket(ctx, uc.servers, strings.TrimSpace(ticket), time.Now())
}

// UploadClientLog 保存一份日志：校验内容，每个游戏账号每小时最多 30 份，只保留最新 5 份。
func (uc *PassportUsecase) UploadClientLog(ctx context.Context, owner ClientLogOwner, in ClientLogUpload) (*gen.GameClientLog, error) {
	now := time.Now()
	row, err := newClientLog(owner, in, now)
	if err != nil {
		return nil, err
	}
	if !uc.clientLogRate.allow(owner.ServerID+"\x00"+owner.PID, now) {
		return nil, ErrClientLogRateLimited
	}
	saved, err := uc.clientLogs.Create(ctx, row, ClientLogKeep)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return saved, nil
}
