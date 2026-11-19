package agentinstall

import (
	"strconv"
	"strings"
)

// Newer 版本 a 是否比 b 新：按点分的数字段比较，带预发布后缀的排在同号正式版之前。
func Newer(a, b string) bool {
	return compareVersions(a, b) > 0
}

func compareVersions(a, b string) int {
	an, ap := splitVersion(a)
	bn, bp := splitVersion(b)
	for i := 0; i < max(len(an), len(bn)); i++ {
		var x, y int
		if i < len(an) {
			x = an[i]
		}
		if i < len(bn) {
			y = bn[i]
		}
		if x != y {
			return sign(x - y)
		}
	}
	switch {
	case ap == "" && bp == "":
		return 0
	case ap == "":
		return 1
	case bp == "":
		return -1
	}
	return comparePre(ap, bp)
}

// splitVersion 把版本号拆成数字段和预发布后缀：v0.108.1-preview.3 → [0 108 1]、preview.3。
func splitVersion(v string) (nums []int, pre string) {
	v = cleanVersion(v)
	// + 后面是构建信息，不参与比较。
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	// 数字和点之后的都算后缀（1.2.3-beta.1，也包括 1.2.3beta1 这种写法）。
	end := strings.IndexFunc(v, func(r rune) bool { return r != '.' && (r < '0' || r > '9') })
	if end >= 0 {
		v, pre = v[:end], strings.TrimLeft(v[end:], "-.")
	}
	for _, part := range strings.Split(strings.Trim(v, "."), ".") {
		n, _ := strconv.Atoi(part)
		nums = append(nums, n)
	}
	return nums, pre
}

// comparePre 比两个预发布后缀：按点分段，都是数字的按数值比，数字段排在文字段之前，段少的排在前。
func comparePre(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		x, xerr := strconv.Atoi(as[i])
		y, yerr := strconv.Atoi(bs[i])
		switch {
		case xerr == nil && yerr == nil:
			if x != y {
				return sign(x - y)
			}
		case xerr == nil:
			return -1
		case yerr == nil:
			return 1
		default:
			if c := strings.Compare(as[i], bs[i]); c != 0 {
				return c
			}
		}
	}
	return sign(len(as) - len(bs))
}

// cleanVersion 去掉空白和开头的 v。
func cleanVersion(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 1 && (v[0] == 'v' || v[0] == 'V') && v[1] >= '0' && v[1] <= '9' {
		v = v[1:]
	}
	return v
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}
