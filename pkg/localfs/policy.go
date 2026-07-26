package localfs

import (
	"path/filepath"
	"runtime"
	"strings"
)

// 默认上限。全部可经 Option 覆盖，但都刻意给了有限值——不设上限等于把 DoS 面白送出去。
const (
	// DefaultMaxFileSize 在线读取/保存单个文件的体积上限（编辑器场景）。
	DefaultMaxFileSize = 10 << 20 // 10 MiB
	// DefaultMaxSearchResults 一次搜索返回的最大条目数。
	DefaultMaxSearchResults = 2000
	// DefaultMaxSearchVisits 一次搜索允许遍历的最大条目数（防止全盘递归卡死服务）。
	DefaultMaxSearchVisits = 200_000
	// DefaultMaxArchiveEntries 解压时允许的最大条目数。
	DefaultMaxArchiveEntries = 20_000
	// DefaultMaxArchiveBytes 解压时允许的最大总解压体积。
	DefaultMaxArchiveBytes = 2 << 30 // 2 GiB
	// DefaultMaxArchiveRatio 单条目允许的最大压缩比（解压后/压缩前），用于识别 zip 炸弹。
	DefaultMaxArchiveRatio = 200
	// DefaultMaxUploadSize 分片上传的单文件体积上限。
	// 与 MaxFileSize（在线编辑用，10 MiB）分开：上传整包备份动辄数十 GiB 是正常的，
	// 但仍必须有界——声明一个天文数字的体积会让缓冲文件在稀疏文件系统上直接铺开。
	DefaultMaxUploadSize = 64 << 30 // 64 GiB
)

// Policy 是作用于一个视图内每次操作的安全策略。
// 零值不可用，请经 [Open] / [OpenSystem] 构造，它们会填入上面的默认值。
type Policy struct {
	// ReadOnly 为 true 时，一切写操作（创建/删除/改名/复制/移动/压缩/解压/上传）直接拒绝。
	ReadOnly bool
	// MaxFileSize 限制 ReadFile / WriteFile 的体积。
	MaxFileSize int64
	// MaxUploadSize 限制一次分片上传声明的文件总体积。
	MaxUploadSize int64
	// MaxSearchResults / MaxSearchVisits 限制搜索的输出量与遍历量。
	MaxSearchResults int
	MaxSearchVisits  int
	// MaxArchiveEntries / MaxArchiveBytes / MaxArchiveRatio 是解压的三道 zip 炸弹闸门。
	MaxArchiveEntries int
	MaxArchiveBytes   int64
	MaxArchiveRatio   int64
	// deny 为已规范化的受保护真实绝对路径；命中者（含其子孙）一律拒绝访问。
	deny []string
}

// Option 用于定制视图策略。
type Option func(*Policy)

// ReadOnly 把视图设为只读。
func ReadOnly() Option {
	return func(p *Policy) { p.ReadOnly = true }
}

// WithMaxFileSize 设置在线读写单文件的体积上限（<=0 表示沿用默认）。
func WithMaxFileSize(n int64) Option {
	return func(p *Policy) {
		if n > 0 {
			p.MaxFileSize = n
		}
	}
}

// WithMaxUploadSize 设置分片上传的单文件体积上限（<=0 表示沿用默认）。
func WithMaxUploadSize(n int64) Option {
	return func(p *Policy) {
		if n > 0 {
			p.MaxUploadSize = n
		}
	}
}

// WithSearchLimits 设置搜索的返回条目上限与遍历上限（<=0 的项沿用默认）。
func WithSearchLimits(results, visits int) Option {
	return func(p *Policy) {
		if results > 0 {
			p.MaxSearchResults = results
		}
		if visits > 0 {
			p.MaxSearchVisits = visits
		}
	}
}

// WithArchiveLimits 设置解压上限（<=0 的项沿用默认）。
func WithArchiveLimits(entries int, totalBytes, ratio int64) Option {
	return func(p *Policy) {
		if entries > 0 {
			p.MaxArchiveEntries = entries
		}
		if totalBytes > 0 {
			p.MaxArchiveBytes = totalBytes
		}
		if ratio > 0 {
			p.MaxArchiveRatio = ratio
		}
	}
}

// Deny 把若干真实路径（及其全部子孙）加入保护清单。
// 典型用途是保护 momoko 自身：data/（内含 SQLite 库与会话数据）、configs/（内含
// auth.secret，泄露即可伪造 JWT 与所有预签名 URL）、以及可执行文件自身。
// 这是纵深防御——即便某条业务路径的边界判定出错，也仍然拿不到密钥。
func Deny(paths ...string) Option {
	return func(p *Policy) {
		for _, raw := range paths {
			if norm := normalizeDeny(raw); norm != "" {
				p.deny = append(p.deny, norm)
			}
		}
	}
}

// normalizeDeny 把保护项规整为可比较的绝对路径（尽力解引用符号链接）。
func normalizeDeny(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	return filepath.Clean(abs)
}

func defaultPolicy(opts ...Option) Policy {
	p := Policy{
		MaxFileSize:       DefaultMaxFileSize,
		MaxUploadSize:     DefaultMaxUploadSize,
		MaxSearchResults:  DefaultMaxSearchResults,
		MaxSearchVisits:   DefaultMaxSearchVisits,
		MaxArchiveEntries: DefaultMaxArchiveEntries,
		MaxArchiveBytes:   DefaultMaxArchiveBytes,
		MaxArchiveRatio:   DefaultMaxArchiveRatio,
	}
	for _, o := range opts {
		if o != nil {
			o(&p)
		}
	}
	return p
}

// pathEqualFold 报告两个已 Clean 的路径是否指向同一位置（Windows 下大小写不敏感）。
func pathEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// withinPath 报告 target 是否等于 root 或位于 root 之下。
// 按分隔符边界比较，避免 "/data-backup" 被 "/data" 误判为子路径。
func withinPath(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if pathEqual(root, target) {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(target), strings.ToLower(prefix))
	}
	return strings.HasPrefix(target, prefix)
}

// affectsDenied 报告一次「会波及整棵子树」的操作是否触及保护清单。
//
// 单点访问只需判断「目标是否落在保护项内」（denied），但删除、改名、移动、复制、打包
// 这类操作会连带处理目标之下的所有内容，因此还必须判断「目标是否包含某个保护项」——
// 否则对着保护目录的父级下手就能整棵搬走或删掉，保护清单形同虚设。
func (p *Policy) affectsDenied(real string) bool {
	if len(p.deny) == 0 {
		return false
	}
	if p.denied(real) {
		return true
	}
	target := filepath.Clean(real)
	for _, d := range p.deny {
		if withinPath(target, d) {
			return true
		}
	}
	// 目标可能是指向某个保护项祖先的链接，解引用后复判。
	if resolved, err := filepath.EvalSymlinks(target); err == nil && !pathEqual(resolved, target) {
		for _, d := range p.deny {
			if withinPath(resolved, d) {
				return true
			}
		}
	}
	return false
}

// denied 报告 real（真实绝对路径）是否命中保护清单。
// 先做词法判定；再解引用符号链接后复判，封堵「在允许目录里放一个指向受保护目录的链接」这种绕过。
func (p *Policy) denied(real string) bool {
	if len(p.deny) == 0 {
		return false
	}
	if p.deniedLexical(real) {
		return true
	}
	// 目标存在则按其真实位置复判；不存在则退一步判其父目录，
	// 覆盖「父目录是链接、目标待创建」的写入场景。
	if resolved, err := filepath.EvalSymlinks(real); err == nil {
		return p.deniedLexical(resolved)
	}
	if resolvedDir, err := filepath.EvalSymlinks(filepath.Dir(real)); err == nil {
		return p.deniedLexical(filepath.Join(resolvedDir, filepath.Base(real)))
	}
	return false
}

func (p *Policy) deniedLexical(real string) bool {
	for _, d := range p.deny {
		if withinPath(d, real) {
			return true
		}
	}
	return false
}
