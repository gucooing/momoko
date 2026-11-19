package agentnotify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 飞书整个请求不能超过 30 KB 左右，正文留在这个字数以内。
const feishuLimit = 4000

// 正文用的文本类型。
const feishuTextTag = "lark_md"

// feishu 往群里的自定义机器人发卡片消息。
type feishu struct {
	url    string
	secret string // 机器人开了「签名校验」时填
}

func (f *feishu) Send(ctx context.Context, msg Message) error {
	l := arrange(msg)

	// 批准通知用橙色的标题栏，其余（结束、出错）用绿色。
	template := "green"
	if l.approval {
		template = "orange"
	}
	var elements []any
	if body := fit(l.lines, feishuLimit); len(body) > 0 {
		elements = append(elements, map[string]any{
			"tag":  "div",
			"text": map[string]any{"tag": feishuTextTag, "content": strings.Join(body, "\n")},
		})
	}
	if buttons := feishuButtons(l); len(buttons) > 0 {
		elements = append(elements, map[string]any{"tag": "action", "actions": buttons})
	}
	if len(elements) == 0 {
		// 卡片不能没有内容。
		elements = append(elements, map[string]any{
			"tag":  "div",
			"text": map[string]any{"tag": "plain_text", "content": l.title},
		})
	}
	payload := map[string]any{
		"msg_type": "interactive",
		"card": map[string]any{
			"config": map[string]any{"wide_screen_mode": true},
			"header": map[string]any{
				"title":    map[string]any{"tag": "plain_text", "content": l.title},
				"template": template,
			},
			"elements": elements,
		},
	}
	if f.secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		payload["timestamp"] = timestamp
		payload["sign"] = feishuSign(timestamp, f.secret)
	}

	status, raw, err := postJSON(ctx, httpClient, "飞书", f.url, nil, payload)
	if err != nil {
		return err
	}
	// 新旧两种应答：{"code":0,"msg":"success"} 和 {"StatusCode":0,"StatusMessage":"success"}。
	var reply struct {
		Code          *int   `json:"code"`
		Msg           string `json:"msg"`
		StatusCode    *int   `json:"StatusCode"`
		StatusMessage string `json:"StatusMessage"`
	}
	_ = json.Unmarshal(raw, &reply)
	code := reply.Code
	if code == nil {
		code = reply.StatusCode
	}
	if status == http.StatusOK && code != nil && *code == 0 {
		return nil
	}
	reason := reply.Msg
	if reason == "" {
		reason = reply.StatusMessage
	}
	if reason == "" {
		return fmt.Errorf("飞书返回 %d", status)
	}
	return fmt.Errorf("飞书返回：%s", clip(reason, maxLineRunes))
}

// feishuButtons 排卡片上的按钮：只有带地址的动作才放得上去，最后是「打开会话」。
func feishuButtons(l layout) []any {
	var buttons []any
	primary := false
	for _, a := range l.actions {
		if a.URL == "" {
			continue
		}
		// 第一个允许类的动作是主按钮，拒绝类用红色。
		kind := "default"
		switch {
		case a.Deny:
			kind = "danger"
		case !primary:
			kind, primary = "primary", true
		}
		buttons = append(buttons, feishuButton(a.Label, a.URL, kind))
	}
	if l.link != "" {
		buttons = append(buttons, feishuButton(openLabel, l.link, "default"))
	}
	return buttons
}

func feishuButton(label, target, kind string) map[string]any {
	return map[string]any{
		"tag":  "button",
		"text": map[string]any{"tag": "plain_text", "content": label},
		"type": kind,
		"url":  target,
	}
}

// feishuSign 算签名：密钥是「时间戳 + 换行 + 机器人密钥」，对空内容做 HMAC-SHA256 再 base64。
func feishuSign(timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(timestamp+"\n"+secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
