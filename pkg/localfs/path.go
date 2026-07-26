package localfs

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// maxPathLen 逻辑路径总长度上限。
	maxPathLen = 4096
	// maxPathDepth 逻辑路径分段数上限。
	maxPathDepth = 128
)

// errVirtualRoot 是内部哨兵：整机视图下 vpath 为空表示「此电脑」虚拟根，
// 它不对应任何单一 os.Root，只有列目录与目录统计两个操作有意义。
var errVirtualRoot = errors.New("localfs: virtual root")

// cleanRel 把一段相对逻辑路径规整为 fs.ValidPath 形式（"." 表示自身，否则形如 "a/b/c"）。
// 正反斜杠都当分隔符，空段与 "." 段丢弃；出现 ".." 段一律拒绝而不是静默折叠——
// 静默折叠会让穿越尝试无声无息，显式报错才能被日志与审计捕获。
//
// 注意：本函数只做规整，不承担安全职责。真正的边界由 os.Root 在内核层保证。
func cleanRel(p string) (string, error) {
	if strings.IndexByte(p, 0) >= 0 {
		return "", fmt.Errorf("%w：不能包含 NUL 字节", ErrInvalidPath)
	}
	if len(p) > maxPathLen {
		return "", fmt.Errorf("%w：路径过长", ErrInvalidPath)
	}
	segs := strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' })
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "." {
			continue
		}
		// 全点分段（".."、"..."、"...." …）一律拒绝。".." 显然是上跳；三点及以上则是
		// Windows 路径规范化的灰色地带（会剥掉结尾的点），实测虽不足以逃出 os.Root，
		// 但它作为文件名毫无意义，直接封掉可消除平台间的行为差异。
		if isAllDots(s) {
			return "", ErrTraversal
		}
		// Windows 会剥掉分段结尾的点与空格，于是「我们算出的路径」与「实际打开的文件」不一致，
		// 保护清单等一切基于路径的判定都会因此错位（"configs." 实际落在 "configs" 里）。
		// 类 Unix 无此改写，故只在 Windows 上拒绝。
		if runtime.GOOS == "windows" {
			if last := s[len(s)-1]; last == '.' || last == ' ' {
				return "", fmt.Errorf("%w：分段不能以 '.' 或空格结尾", ErrInvalidPath)
			}
		}
		out = append(out, s)
	}
	if len(out) > maxPathDepth {
		return "", fmt.Errorf("%w：路径层级过深", ErrInvalidPath)
	}
	if len(out) == 0 {
		return ".", nil
	}
	return strings.Join(out, "/"), nil
}

// isAllDots 报告分段是否只由 '.' 组成（"."、".."、"..." …）。
func isAllDots(s string) bool {
	if s == "" {
		return false
	}
	return strings.Count(s, ".") == len(s)
}

// hasVolume 报告路径是否带 Windows 卷名前缀（"C:"、"\\\\host\\share"）。
func hasVolume(p string) bool {
	return filepath.VolumeName(filepath.FromSlash(p)) != ""
}

// isRooted 报告路径是否以分隔符或卷名开头，即「不是纯相对路径」。
func isRooted(p string) bool {
	if p == "" {
		return false
	}
	if p[0] == '/' || p[0] == '\\' {
		return true
	}
	return hasVolume(p)
}

// splitMount 把真实绝对路径拆为「挂载点 + 挂载点内相对逻辑路径」。
// Windows 的挂载点是盘符根（"C:\"），类 Unix 恒为 "/"。
//
// UNC 路径（"\\host\share\..."）一律拒绝：让服务端按请求去连任意 SMB 主机，
// 在 Windows 上会把机器账号的 NTLM 凭据发给对方，是一条现成的凭据外泄通道。
func splitMount(abs string) (mount, rest string, err error) {
	abs = filepath.FromSlash(abs)
	if runtime.GOOS != "windows" {
		if !strings.HasPrefix(abs, "/") {
			return "", "", fmt.Errorf("%w：需要绝对路径", ErrInvalidPath)
		}
		rel, err := cleanRel(strings.TrimPrefix(abs, "/"))
		if err != nil {
			return "", "", err
		}
		return "/", rel, nil
	}

	vol := filepath.VolumeName(abs)
	if vol == "" {
		return "", "", fmt.Errorf("%w：需要带盘符的绝对路径", ErrInvalidPath)
	}
	if len(vol) != 2 || vol[1] != ':' {
		return "", "", fmt.Errorf("%w：不支持 UNC / 设备路径", ErrTraversal)
	}
	rel, err := cleanRel(strings.TrimPrefix(abs, vol))
	if err != nil {
		return "", "", err
	}
	return vol + `\`, rel, nil
}

// stripBase 把一个「视图内的真实绝对路径」转成相对 base 的逻辑路径。
// 不在 base 之下则返回 ErrTraversal。
//
// 这里的词法比较只用于把客户端回传的绝对路径映射回相对路径；判断失误只会导致误拒，
// 不会放行——放行与否最终由 os.Root 决定。
func stripBase(base, abs string) (string, error) {
	base = filepath.Clean(base)
	abs = filepath.Clean(filepath.FromSlash(abs))
	if pathEqual(base, abs) {
		return ".", nil
	}
	// 按分段比对再取余，绝不按 len(base) 切字节：Windows 下比较是大小写不敏感的，
	// 而 Unicode 大小写折叠会改变字节长度（U+212A KELVIN SIGN 三字节，小写 'k' 一字节），
	// 于是「前缀匹配成功」并不意味着 len(abs) >= len(base)——旧写法会直接切出 panic。
	baseSegs := pathSegments(base)
	absSegs := pathSegments(abs)
	if len(absSegs) <= len(baseSegs) {
		return "", ErrTraversal
	}
	for i, seg := range baseSegs {
		if !segmentEqual(seg, absSegs[i]) {
			return "", ErrTraversal
		}
	}
	return cleanRel(strings.Join(absSegs[len(baseSegs):], "/"))
}

// pathSegments 把一个已 Clean 的绝对路径拆成分段，首段为卷名（Windows 下如 "C:"，类 Unix 下为空串）。
func pathSegments(p string) []string {
	vol := filepath.VolumeName(p)
	rest := p[len(vol):]
	segs := []string{vol}
	for _, s := range strings.FieldsFunc(rest, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		segs = append(segs, s)
	}
	return segs
}

// segmentEqual 比较单个路径分段（Windows 下大小写不敏感）。
func segmentEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// joinReal 拼出用于展示与保护清单判定的真实绝对路径。
func joinReal(mount, rel string) string {
	if rel == "." || rel == "" {
		return filepath.Clean(mount)
	}
	return filepath.Join(mount, filepath.FromSlash(rel))
}

// relParent 返回相对路径的父级（fs.ValidPath 形式）。
func relParent(rel string) string {
	if rel == "." {
		return "."
	}
	return path.Dir(rel)
}

// relJoin 在相对路径上追加一个分段。
func relJoin(rel, name string) string {
	if rel == "." || rel == "" {
		return name
	}
	return rel + "/" + name
}
