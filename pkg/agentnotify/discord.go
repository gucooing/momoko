package agentnotify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Discord 一条消息最多 2000 个字，留一点余量。
const discordLimit = 1900

// discord 往频道的 Webhook 发消息。
type discord struct {
	url string
}

func (d *discord) Send(ctx context.Context, msg Message) error {
	l := arrange(msg)

	var head []string
	if l.title != "" {
		head = append(head, "**"+discordEscape(l.title)+"**")
	}
	// 动作排成一行，「打开会话」单独一行。
	var tail, actions []string
	for _, a := range l.actions {
		if a.URL != "" {
			actions = append(actions, discordLink(a.Label, a.URL))
		}
	}
	if len(actions) > 0 {
		tail = append(tail, strings.Join(actions, " · "))
	}
	if l.link != "" {
		tail = append(tail, discordLink(openLabel, l.link))
	}
	body := make([]string, 0, len(l.lines))
	for _, line := range l.lines {
		body = append(body, discordEscape(line))
	}
	body = fit(body, discordLimit-runeLen(head)-runeLen(tail))

	content := strings.Join(append(append(head, body...), tail...), "\n")
	status, raw, err := postJSON(ctx, httpClient, "Discord", d.url, nil, map[string]any{
		"content": content,
		// 4 = SUPPRESS_EMBEDS：链接不展开成预览卡片。
		"flags": 4,
		// 正文里是命令和输出，出现 @everyone 之类的字样也不能真的去提醒人。
		"allowed_mentions": map[string]any{"parse": []string{}},
	})
	if err != nil {
		return err
	}
	if ok2xx(status) {
		return nil
	}
	var reply struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &reply)
	if reply.Message != "" {
		return fmt.Errorf("Discord 返回 %d：%s", status, clip(reply.Message, maxLineRunes))
	}
	return fmt.Errorf("Discord 返回 %d", status)
}

// discordMarks 是 Discord Markdown 里会改变显示的记号：正文是命令和路径，星号、下划线这些要原样显示。
const discordMarks = "\\*_~`|[]<>"

// discordEscape 给 Markdown 记号加上反斜杠。
func discordEscape(line string) string {
	// 地址不动：Discord 会把它整个认成链接，里面加了反斜杠反而点不开。
	words := strings.Split(line, " ")
	for i, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			continue
		}
		words[i] = discordEscapeWord(word)
	}
	out := strings.Join(words, " ")
	// 行首的 # 和 - 会变成标题、列表。
	if rest := strings.TrimLeft(out, " "); strings.HasPrefix(rest, "#") || strings.HasPrefix(rest, "- ") {
		out = out[:len(out)-len(rest)] + `\` + rest
	}
	return out
}

func discordEscapeWord(word string) string {
	if !strings.ContainsAny(word, discordMarks) {
		return word
	}
	var b strings.Builder
	b.Grow(len(word) + 8)
	for _, r := range word {
		if r < utf8.RuneSelf && strings.ContainsRune(discordMarks, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// discordURL 让地址能放进 [名称](地址)：括号和空格会把链接截断。
var discordURL = strings.NewReplacer("(", "%28", ")", "%29", " ", "%20", "<", "%3C", ">", "%3E")

func discordLink(label, target string) string {
	return "[" + discordEscapeWord(label) + "](" + discordURL.Replace(target) + ")"
}
