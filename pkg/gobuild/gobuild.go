// Package gobuild 读取 Go 可执行文件里嵌入的构建信息（不运行它）：模块、Go 版本、
// VCS 修订与 -ldflags -X 注入的变量。用于在更新前确认一个文件确实是某个服务端的构建并取出版本。
package gobuild

import (
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

// Info 是一个可执行文件的构建信息。
type Info struct {
	Path      string
	Size      int64
	ModTime   time.Time
	GoVersion string
	// MainPath main 包路径，例如 example.com/server/cmd/app
	MainPath string
	// Module 主模块路径
	Module string
	// VCSRevision / VCSTime 构建时的版本库信息（-buildvcs=false 时为空）
	VCSRevision string
	VCSTime     time.Time
	// LDFlags 构建参数里的 -ldflags 原文
	LDFlags string
}

// ErrNotGoBinary 文件不是带构建信息的 Go 可执行文件。
var ErrNotGoBinary = errors.New("gobuild: not a Go executable with build info")

// Read 读取文件的构建信息。
func Read(path string) (*Info, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, os.ErrInvalid
	}
	out := &Info{Path: path, Size: st.Size(), ModTime: st.ModTime()}
	bi, err := buildinfo.ReadFile(path)
	if err != nil {
		return out, ErrNotGoBinary
	}
	out.GoVersion = bi.GoVersion
	out.MainPath = bi.Path
	out.Module = bi.Main.Path
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			out.VCSRevision = s.Value
		case "vcs.time":
			if t, err := time.Parse(time.RFC3339, s.Value); err == nil {
				out.VCSTime = t
			}
		case "-ldflags":
			out.LDFlags = s.Value
		}
	}
	return out, nil
}

// Var 返回 -ldflags 里 -X <name>=<value> 注入的值（name 为完整的包路径.变量名）。
func (i *Info) Var(name string) string {
	if i == nil || i.LDFlags == "" {
		return ""
	}
	fields := splitFlags(i.LDFlags)
	for idx := 0; idx < len(fields); idx++ {
		f := fields[idx]
		var assign string
		switch {
		case f == "-X" && idx+1 < len(fields):
			assign = fields[idx+1]
			idx++
		case strings.HasPrefix(f, "-X="):
			assign = strings.TrimPrefix(f, "-X=")
		default:
			continue
		}
		if k, v, ok := strings.Cut(assign, "="); ok && k == name {
			return v
		}
	}
	return ""
}

// splitFlags 按空白拆分，支持单双引号。
func splitFlags(s string) []string {
	var out []string
	var cur strings.Builder
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// SHA256 计算文件的 SHA-256。
func SHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
