package biz

import (
	"errors"
	"strings"
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/pkg/response"
)

// 兑换码的通用规则：管理端（批次管理）与用户通行证（兑换）读写同一组表，按同样的规则判断码与批次。

// 兑换入口。游戏内（game）入口需要游戏服务端把玩家输入的码转给 Momoko，MGP v1 没有这条通道，
// 所以 game 只能作为批次设置保存，目前只有通行证（passport）会真正调用兑换。
const (
	GiftChannelPassport = "passport"
	GiftChannelGame     = "game"
	GiftChannelAdmin    = "admin"
)

// NormalizeGiftCode 规范化玩家输入的兑换码：去掉空白、转大写。
func NormalizeGiftCode(code string) string {
	return strings.ToUpper(strings.Join(strings.Fields(code), ""))
}

// GiftBatchStatus 批次状态：disabled / expired / exhausted / not_started / active。
func GiftBatchStatus(b *gen.GiftCodeBatch, now time.Time) string {
	switch {
	case b.DisabledAt != nil:
		return "disabled"
	case b.EndAt != nil && !now.Before(*b.EndAt):
		return "expired"
	case b.TotalLimit > 0 && b.UsedCount >= b.TotalLimit:
		return "exhausted"
	case b.StartAt != nil && now.Before(*b.StartAt):
		return "not_started"
	default:
		return "active"
	}
}

// errText 把执行错误转成存储与展示用的文字（接口错误取它的信息）。
func errText(err error) string {
	var re response.Error
	if errors.As(err, &re) {
		return re.Message
	}
	return err.Error()
}
