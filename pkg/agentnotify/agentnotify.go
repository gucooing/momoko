// Package agentnotify 把 Agent 的通知（在等批准、一轮结束、出错）发到聊天平台：
// Discord、Telegram、飞书自定义机器人、QQ 官方机器人。
//
// 四个渠道发的是同一条通知，只是各自套上自己的格式。Telegram 的批准通知带真正的按钮，
// 点了以后由 PollTelegram 收回调；其它渠道给的是打开批准页的链接。
package agentnotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// Message 一条通知。
type Message struct {
	Title   string   // 标题，例如：「文章列表加分页」在等你批准
	Lines   []string // 正文，一行一条（Agent、工作区、要执行的命令或结束说明）
	Link    string   // 打开会话的地址；没有站点地址时为空
	Actions []Action // 批准用的动作；不是批准通知时为空
}

// Action 批准通知上的一个动作。
type Action struct {
	Label string // 允许这一次 / 拒绝 …
	URL   string // 打开批准页并选中这个选项的地址；没有站点地址时为空
	Data  string // Telegram 按钮的 callback_data（≤ 64 字节）；为空则这个动作在 Telegram 上也只用链接
	Deny  bool   // 是拒绝类的动作（用来决定按钮样式和排列）
}

// Sender 往一种渠道发通知。
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Fields 一种渠道的配置项，键见各渠道。
type Fields map[string]string

// New 按渠道种类建发送器：discord（url）、telegram（token、chat）、feishu（url、secret 可选）、qq（appid、secret、to）。
// 必填项缺失时返回错误。
func New(kind string, fields Fields) (Sender, error) {
	get := func(key string) string { return strings.TrimSpace(fields[key]) }
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "discord":
		endpoint, err := webhookURL("Discord", get("url"))
		if err != nil {
			return nil, err
		}
		return &discord{url: endpoint}, nil
	case "telegram":
		token, chat := get("token"), get("chat")
		if token == "" {
			return nil, errors.New("Telegram 的机器人令牌还没填")
		}
		if strings.ContainsAny(token, " \t\r\n/?#") {
			return nil, errors.New("Telegram 的机器人令牌不对")
		}
		if chat == "" {
			return nil, errors.New("Telegram 的会话 ID 还没填")
		}
		return &telegram{api: telegramAPI{client: httpClient, token: token}, chat: chat}, nil
	case "feishu":
		endpoint, err := webhookURL("飞书", get("url"))
		if err != nil {
			return nil, err
		}
		return &feishu{url: endpoint, secret: get("secret")}, nil
	case "qq":
		appID, secret, to := get("appid"), get("secret"), get("to")
		if appID == "" {
			return nil, errors.New("QQ 的 AppID 还没填")
		}
		if secret == "" {
			return nil, errors.New("QQ 的 AppSecret 还没填")
		}
		path, err := qqMessagePath(to)
		if err != nil {
			return nil, err
		}
		return &qq{appID: appID, secret: secret, path: path}, nil
	}
	return nil, fmt.Errorf("不支持的通知渠道：%s", kind)
}

const (
	maxLineRunes  = 300      // 一行最多多少个字：要执行的命令可能非常长
	maxTitleRunes = 200      // 标题同理
	maxResponse   = 64 << 10 // 响应体最多读这么多
	openLabel     = "打开会话"
)

// 不设 Transport：用默认的那个，代理环境变量才生效。
var httpClient = &http.Client{Timeout: 15 * time.Second}

// link 一个带地址的入口：批准动作，或「打开会话」。
type link struct {
	label string
	url   string
}

// layout 一条通知整理之后的样子。四个渠道都从这里取内容，文字才不会各写各的。
type layout struct {
	title    string
	lines    []string // 正文：按行拆开，每行截断过
	actions  []Action // 批准动作：允许类在前，拒绝类在后；只留用得上的
	link     string   // 打开会话的地址
	approval bool     // 是批准通知
}

func arrange(msg Message) layout {
	l := layout{
		title:    clip(strings.Join(strings.Fields(msg.Title), " "), maxTitleRunes),
		link:     strings.TrimSpace(msg.Link),
		approval: len(msg.Actions) > 0,
	}
	for _, entry := range msg.Lines {
		// 一条里可能自带换行（多行命令），拆开后逐行截断。
		for _, line := range strings.Split(entry, "\n") {
			line = strings.TrimRight(line, " \t\r")
			if strings.TrimSpace(line) == "" {
				continue
			}
			l.lines = append(l.lines, clip(line, maxLineRunes))
		}
	}
	var deny []Action
	for _, a := range msg.Actions {
		a.Label = strings.TrimSpace(a.Label)
		a.URL = strings.TrimSpace(a.URL)
		if a.Label == "" || (a.URL == "" && a.Data == "") {
			continue
		}
		if a.Deny {
			deny = append(deny, a)
		} else {
			l.actions = append(l.actions, a)
		}
	}
	l.actions = append(l.actions, deny...)
	return l
}

// links 所有带地址的入口：各个动作，最后是「打开会话」。
func (l layout) links() []link {
	var out []link
	for _, a := range l.actions {
		if a.URL != "" {
			out = append(out, link{label: a.Label, url: a.URL})
		}
	}
	if l.link != "" {
		out = append(out, link{label: openLabel, url: l.link})
	}
	return out
}

// plain 排成纯文本的行：标题、正文，withLinks 时再跟上各个地址。正文超过 budget 个字时从后面丢。
func (l layout) plain(withLinks bool, budget int) []string {
	var tail []string
	if withLinks {
		tail = linkLines(l.links())
	}
	budget -= utf8.RuneCountInString(l.title) + 1 + runeLen(tail)
	out := make([]string, 0, 1+len(l.lines)+len(tail))
	if l.title != "" {
		out = append(out, l.title)
	}
	out = append(out, fit(l.lines, budget)...)
	return append(out, tail...)
}

// linkLines 把地址写成「名称：地址」的行，给放不了按钮和链接的地方用。
func linkLines(links []link) []string {
	out := make([]string, 0, len(links))
	for _, k := range links {
		out = append(out, k.label+"："+k.url)
	}
	return out
}

// clip 把一行截到 max 个字以内，截掉的部分用省略号代替。
func clip(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}

// fit 让这些行加起来不超过 budget 个字：放不下的从后面丢掉，留一行省略号。
func fit(lines []string, budget int) []string {
	used := 2 // 给省略号那一行留的
	for i, line := range lines {
		used += utf8.RuneCountInString(line) + 1
		if used > budget {
			return append(append([]string{}, lines[:i]...), "…")
		}
	}
	return lines
}

// runeLen 这些行连同换行一共多少个字。
func runeLen(lines []string) int {
	n := 0
	for _, line := range lines {
		n += utf8.RuneCountInString(line) + 1
	}
	return n
}

// webhookURL 检查 Webhook 地址。
func webhookURL(platform, raw string) (string, error) {
	if raw == "" {
		return "", errors.New(say("", platform, "的 Webhook 地址还没填"))
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New(say("", platform, "的 Webhook 地址不对，要以 https:// 开头"))
	}
	return raw, nil
}

// say 把平台名嵌进一句话里：英文名和中文之间留空格（连不上 Discord），中文名不留（连不上飞书）。
func say(before, platform, after string) string {
	if platform != "" && platform[0] < utf8.RuneSelf {
		if before != "" {
			before += " "
		}
		if after != "" {
			after = " " + after
		}
	}
	return before + platform + after
}

// postJSON 发一个 JSON 请求，返回状态码和响应体（最多 64 KiB）。
//
// 出错时不把原始错误带出去：net/http 的错误里有完整的请求地址，而地址里就是 Webhook 密钥或机器人令牌。
func postJSON(ctx context.Context, client *http.Client, platform, endpoint string, header map[string]string, payload any) (int, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, errors.New(say("", platform, "的通知内容编码失败"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, errors.New(say("", platform, "的地址不对"))
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, netError(ctx, platform, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil && len(body) == 0 {
		return 0, nil, netError(ctx, platform, err)
	}
	return resp.StatusCode, body, nil
}

// netError 把网络错误换成能给人看、又不带地址的说法。
func netError(ctx context.Context, platform string, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		// 调用方自己取消的，原样交回去。
		return context.Canceled
	}
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		return errors.New(say("连接", platform, "超时"))
	}
	return errors.New(say("连不上", platform, ""))
}

func ok2xx(status int) bool { return status >= 200 && status < 300 }
