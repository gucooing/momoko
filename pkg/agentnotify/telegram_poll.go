package agentnotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// getUpdates 最多挂 50 秒；客户端超时要比它长，不然每次都被自己掐断。
	telegramPollSeconds = 50
	telegramPollTimeout = 65 * time.Second
	// 令牌不对（401），或者同一个机器人还有别的程序在轮询、设了 Webhook（409）：马上重试没有用，隔久一点。
	telegramConflictWait = 30 * time.Second
	telegramRetryMin     = time.Second
	telegramRetryMax     = 30 * time.Second
	// answerCallbackQuery 的提示最多 200 个字。
	telegramToastMax = 200
)

var telegramPollClient = &http.Client{Timeout: telegramPollTimeout}

// TelegramCallback 用户在 Telegram 里点了通知上的按钮。
type TelegramCallback struct {
	Data string // 按钮的 callback_data
	From string // 点按钮的人（用户名或名字），只用于显示
}

// PollTelegram 长轮询 getUpdates，只收 callback_query。每收到一次调用 handle：
// 返回的 text 作为点按钮后的提示（answerCallbackQuery），done 为 true 时去掉这条消息上的按钮并在正文末尾补一行 text。
// 一直运行到 ctx 结束；网络错误退避后重试，不返回。chatID 不为空时只处理来自这个会话的回调。
func PollTelegram(ctx context.Context, token, chatID string, handle func(cb TelegramCallback) (text string, done bool)) {
	token, chatID = strings.TrimSpace(token), strings.TrimSpace(chatID)
	if token == "" || handle == nil {
		return
	}
	poll := telegramAPI{client: telegramPollClient, token: token}
	reply := telegramAPI{client: httpClient, token: token}

	var offset int64
	wait := telegramRetryMin
	for ctx.Err() == nil {
		var updates []telegramUpdate
		err := poll.call(ctx, "getUpdates", map[string]any{
			"offset":          offset,
			"timeout":         telegramPollSeconds,
			"allowed_updates": []string{"callback_query"},
		}, &updates)
		if err != nil {
			pause := wait
			var apiErr *telegramError
			if errors.As(err, &apiErr) && (apiErr.Status == http.StatusConflict || apiErr.Status == http.StatusUnauthorized) {
				pause = telegramConflictWait
			} else {
				wait = min(wait*2, telegramRetryMax)
			}
			if !sleep(ctx, pause) {
				return
			}
			continue
		}
		wait = telegramRetryMin
		for _, u := range updates {
			// 先推进 offset：处理到一半出错也不会把同一次点击再收一遍。
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if u.Callback == nil {
				continue
			}
			answerTelegram(ctx, reply, chatID, u.Callback, handle)
		}
	}
}

// answerTelegram 处理一次点击：交给 handle，回一个提示，需要时收起按钮。
// 这几步失败了都不重试：点击太久没回应 Telegram 自己会作废，用户再点一次就是了。
func answerTelegram(ctx context.Context, api telegramAPI, chatID string, q *telegramCallbackQuery, handle func(cb TelegramCallback) (string, bool)) {
	if chatID != "" && (q.Message == nil || !q.Message.Chat.is(chatID)) {
		// 别的会话里的按钮不归这里管，只回一个空应答让客户端的转圈停下。
		_ = api.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": q.ID}, nil)
		return
	}

	text, done := handle(TelegramCallback{Data: q.Data, From: q.From.display()})
	text = strings.TrimSpace(text)

	answer := map[string]any{"callback_query_id": q.ID}
	if text != "" {
		answer["text"] = clip(text, telegramToastMax)
	}
	_ = api.call(ctx, "answerCallbackQuery", answer, nil)

	if !done || q.Message == nil {
		return
	}
	m := q.Message
	if m.Text != "" && text != "" {
		edit := map[string]any{
			"chat_id":              m.Chat.ID,
			"message_id":           m.MessageID,
			"text":                 m.Text + "\n" + text,
			"link_preview_options": map[string]any{"is_disabled": true},
		}
		// 回调里的正文是去掉格式的纯文本，格式在 entities 里：原样带回去，标题的加粗才不会丢。
		// 新加的一行在最后，前面的偏移都不变。不带 reply_markup，按钮就没了。
		if len(m.Entities) > 0 {
			edit["entities"] = m.Entities
		}
		if api.call(ctx, "editMessageText", edit, nil) == nil {
			return
		}
	}
	// 拿不到原文（消息太旧），或者改正文失败：至少把按钮收掉。
	_ = api.call(ctx, "editMessageReplyMarkup", map[string]any{
		"chat_id":      m.Chat.ID,
		"message_id":   m.MessageID,
		"reply_markup": map[string]any{"inline_keyboard": [][]telegramButton{}},
	}, nil)
}

type telegramUpdate struct {
	UpdateID int64                  `json:"update_id"`
	Callback *telegramCallbackQuery `json:"callback_query"`
}

type telegramCallbackQuery struct {
	ID      string           `json:"id"`
	From    telegramUser     `json:"from"`
	Message *telegramMessage `json:"message"`
	Data    string           `json:"data"`
}

type telegramUser struct {
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (u telegramUser) display() string {
	if u.Username != "" {
		return u.Username
	}
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

type telegramMessage struct {
	MessageID int64           `json:"message_id"`
	Chat      telegramChat    `json:"chat"`
	Text      string          `json:"text"`
	Entities  json.RawMessage `json:"entities"`
}

type telegramChat struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// is 这个会话是不是配置里填的那个：填的可以是数字 ID，也可以是 @频道名。
func (c telegramChat) is(configured string) bool {
	if strconv.FormatInt(c.ID, 10) == configured {
		return true
	}
	return c.Username != "" && strings.EqualFold("@"+c.Username, configured)
}

// sleep 等一段时间；ctx 先结束时返回 false。
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
