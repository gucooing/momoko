package agentnotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	qqTokenURL = "https://bots.qq.com/app/getAppAccessToken"
	qqAPIBase  = "https://api.sgroup.qq.com"
	// 文本消息的长度上限官方没有写明，留在这个字数以内。
	qqLimit = 1500
	// 访问令牌到期前这么久就换新的，免得发到一半过期。
	qqTokenMargin = time.Minute
)

// qq 用 QQ 官方机器人给一个人或一个群发文本消息。
type qq struct {
	appID  string
	secret string
	path   string // /v2/users/{openid}/messages 或 /v2/groups/{group_openid}/messages

	mu      sync.Mutex
	token   string
	expires time.Time
}

// qqMessagePath 由「接收人」得出发消息的接口：group: 开头的是群，其余是单聊的用户。
func qqMessagePath(to string) (string, error) {
	kind, id := "users", to
	if rest, ok := strings.CutPrefix(to, "group:"); ok {
		kind, id = "groups", rest
	} else if rest, ok := strings.CutPrefix(to, "user:"); ok {
		id = rest
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("QQ 的接收人还没填")
	}
	return "/v2/" + kind + "/" + url.PathEscape(id) + "/messages", nil
}

func (q *qq) Send(ctx context.Context, msg Message) error {
	// 令牌取不到就不用往下试了（AppID、密钥不对）。
	if _, err := q.accessToken(ctx); err != nil {
		return err
	}
	l := arrange(msg)
	full := l.plain(true, qqLimit)
	err := q.post(ctx, full)

	// QQ 只放行在机器人后台登记过域名的地址，别的会整条拒收：去掉带地址的行再发一次，通知本身先送到。
	var apiErr *qqError
	if errors.As(err, &apiErr) && len(withoutURLs(full)) < len(full) {
		if bare := withoutURLs(l.plain(false, qqLimit)); len(bare) > 0 && q.post(ctx, bare) == nil {
			return nil
		}
	}
	return err
}

// withoutURLs 去掉含有地址的行。
func withoutURLs(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func (q *qq) post(ctx context.Context, lines []string) error {
	token, err := q.accessToken(ctx)
	if err != nil {
		return err
	}
	status, raw, err := postJSON(ctx, httpClient, "QQ", qqAPIBase+q.path,
		map[string]string{"Authorization": "QQBot " + token},
		map[string]any{"content": strings.Join(lines, "\n"), "msg_type": 0})
	if err != nil {
		return err
	}
	if ok2xx(status) {
		return nil
	}
	if status == http.StatusUnauthorized {
		// 令牌被提前作废了，下次重新取。
		q.mu.Lock()
		q.token = ""
		q.mu.Unlock()
	}
	return newQQError(status, raw)
}

// accessToken 取访问令牌：有效期内用缓存的，快到期了重新取。
func (q *qq) accessToken(ctx context.Context) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.token != "" && time.Now().Before(q.expires) {
		return q.token, nil
	}
	status, raw, err := postJSON(ctx, httpClient, "QQ", qqTokenURL, nil,
		map[string]string{"appId": q.appID, "clientSecret": q.secret})
	if err != nil {
		return "", err
	}
	var reply struct {
		AccessToken string          `json:"access_token"`
		ExpiresIn   json.RawMessage `json:"expires_in"` // 有时是字符串 "7200"，有时是数字
	}
	_ = json.Unmarshal(raw, &reply)
	if !ok2xx(status) || reply.AccessToken == "" {
		return "", newQQError(status, raw)
	}
	seconds, _ := strconv.ParseFloat(strings.Trim(string(reply.ExpiresIn), `" `), 64)
	q.token = reply.AccessToken
	q.expires = time.Now().Add(time.Duration(seconds)*time.Second - qqTokenMargin)
	return q.token, nil
}

// qqError 是 QQ 接口明确返回的错误。
type qqError struct {
	Status  int
	Code    int64
	Message string
}

func newQQError(status int, raw []byte) *qqError {
	var reply struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &reply)
	return &qqError{Status: status, Code: reply.Code, Message: clip(reply.Message, maxLineRunes)}
}

func (e *qqError) Error() string {
	switch {
	case e.Message != "" && e.Code != 0:
		return fmt.Sprintf("QQ 返回：%s（%d）", e.Message, e.Code)
	case e.Message != "":
		return "QQ 返回：" + e.Message
	case e.Code != 0:
		return fmt.Sprintf("QQ 返回 %d（%d）", e.Status, e.Code)
	}
	return fmt.Sprintf("QQ 返回 %d", e.Status)
}
