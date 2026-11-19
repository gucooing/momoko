package agentnotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	// Telegram 一条消息最多 4096 个字，留一点余量。
	telegramLimit = 3900
	// callback_data 最多 64 字节，超过的按钮 Telegram 整条消息都不收。
	telegramDataMax = 64
	// 一行最多放几个动作按钮：再多手机上字就显示不全了。
	telegramRowMax = 3
)

// telegramBase 是 Bot API 的地址。
const telegramBase = "https://api.telegram.org"

// telegram 用机器人往一个会话发消息。
type telegram struct {
	api  telegramAPI
	chat string
}

func (t *telegram) Send(ctx context.Context, msg Message) error {
	l := arrange(msg)
	rows, _ := telegramKeyboard(l, true)
	err := t.send(ctx, l, nil, rows)

	// 指向 localhost 或内网地址的链接按钮 Telegram 不收：改成把地址写进正文再发一次。
	var apiErr *telegramError
	if errors.As(err, &apiErr) && apiErr.badURL() {
		if rows, dropped := telegramKeyboard(l, false); len(dropped) > 0 {
			return t.send(ctx, l, linkLines(dropped), rows)
		}
	}
	return err
}

func (t *telegram) send(ctx context.Context, l layout, extra []string, rows [][]telegramButton) error {
	var head []string
	if l.title != "" {
		head = append(head, "<b>"+telegramEscape(l.title)+"</b>")
	}
	body := make([]string, 0, len(l.lines))
	for _, line := range l.lines {
		body = append(body, telegramEscape(line))
	}
	tail := make([]string, 0, len(extra))
	for _, line := range extra {
		tail = append(tail, telegramEscape(line))
	}
	body = fit(body, telegramLimit-runeLen(head)-runeLen(tail))

	payload := map[string]any{
		"chat_id":              t.chat,
		"text":                 strings.Join(append(append(head, body...), tail...), "\n"),
		"parse_mode":           "HTML",
		"link_preview_options": map[string]any{"is_disabled": true},
	}
	if len(rows) > 0 {
		payload["reply_markup"] = map[string]any{"inline_keyboard": rows}
	}
	return t.api.call(ctx, "sendMessage", payload, nil)
}

// telegramButton 是消息下面的一个按钮：要么带 callback_data（点了回调给 Momoko），要么带地址。
type telegramButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
	Style        string `json:"style,omitempty"`
}

// telegramKeyboard 排按钮：动作一行（太多时允许类、拒绝类各一行），「打开会话」另起一行。
// withURLs 为 false 时不放链接按钮，放不上去的入口从 dropped 返回。
func telegramKeyboard(l layout, withURLs bool) (rows [][]telegramButton, dropped []link) {
	var allow, deny []telegramButton
	for _, a := range l.actions {
		b := telegramButton{Text: a.Label}
		switch {
		case a.Data != "" && len(a.Data) <= telegramDataMax:
			b.CallbackData = a.Data
		case a.URL != "" && withURLs:
			b.URL = a.URL
		case a.URL != "":
			dropped = append(dropped, link{label: a.Label, url: a.URL})
			continue
		default:
			continue
		}
		if a.Deny {
			b.Style = "danger"
			deny = append(deny, b)
		} else {
			if len(allow) == 0 {
				b.Style = "success"
			}
			allow = append(allow, b)
		}
	}
	if len(allow)+len(deny) <= telegramRowMax {
		allow, deny = append(allow, deny...), nil
	}
	for _, row := range [][]telegramButton{allow, deny} {
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	if l.link != "" {
		if withURLs {
			rows = append(rows, []telegramButton{{Text: openLabel, URL: l.link}})
		} else {
			dropped = append(dropped, link{label: openLabel, url: l.link})
		}
	}
	return rows, dropped
}

var telegramEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// telegramEscape 转义 HTML 模式下的三个特殊字符。
func telegramEscape(s string) string { return telegramEscaper.Replace(s) }

// telegramError 是 Bot API 明确返回的错误。
type telegramError struct {
	Status      int
	Description string
}

func (e *telegramError) Error() string {
	if e.Description == "" {
		return fmt.Sprintf("Telegram 返回 %d", e.Status)
	}
	return "Telegram 返回：" + e.Description
}

// badURL 是不是因为按钮上的地址不被接受（BUTTON_URL_INVALID、Wrong HTTP URL 之类）。
func (e *telegramError) badURL() bool {
	if e.Status != http.StatusBadRequest {
		return false
	}
	d := strings.ToLower(e.Description)
	return strings.Contains(d, "url") && (strings.Contains(d, "invalid") || strings.Contains(d, "wrong"))
}

// telegramAPI 调 Bot API 的方法。令牌在请求地址里，所以任何错误都不带地址。
type telegramAPI struct {
	client *http.Client
	token  string
}

// call 调一个方法，result 不为空时把返回的 result 解进去。
func (a telegramAPI) call(ctx context.Context, method string, payload, result any) error {
	endpoint := telegramBase + "/bot" + url.PathEscape(a.token) + "/" + method
	status, raw, err := postJSON(ctx, a.client, "Telegram", endpoint, nil, payload)
	if err != nil {
		return err
	}
	var reply struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil || !reply.OK {
		return &telegramError{Status: status, Description: clip(reply.Description, maxLineRunes)}
	}
	if result != nil && len(reply.Result) > 0 {
		if err := json.Unmarshal(reply.Result, result); err != nil {
			return errors.New("Telegram 返回的内容看不懂")
		}
	}
	return nil
}
