package sub2api

import (
	"strings"
	"unicode/utf8"
)

// MaskPublicDisplayName 公开页展示用脱敏：
//   - 看起来像邮箱（含 @ 且右侧有 .）→ 本地部分打码，域名保留
//     a@b.com          → a***@b.com
//     ab@b.com         → a***@b.com
//     abcdef@x.y       → ab***@x.y
//   - 普通用户名 → 原样返回
//
// 仅用于公开 API 出站；管理端仍返回原始 user_name。
func MaskPublicDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	at := strings.LastIndex(name, "@")
	if at <= 0 || at == len(name)-1 {
		return name // 无 @ 或 @ 在首/尾 → 当作用户名
	}
	local, domain := name[:at], name[at+1:]
	if !strings.Contains(domain, ".") {
		return name // 不像邮箱
	}
	// 本地部分：保留最多前 2 个 rune，其余 ***
	keep := 1
	if utf8.RuneCountInString(local) >= 3 {
		keep = 2
	}
	var b strings.Builder
	n := 0
	for _, r := range local {
		if n >= keep {
			break
		}
		b.WriteRune(r)
		n++
	}
	b.WriteString("***@")
	b.WriteString(domain)
	return b.String()
}
