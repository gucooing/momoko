// Package sysenv 给新起的进程（本机终端、Agent）用的环境变量。
//
// 进程的环境变量是启动那一刻定下来的：Momoko 跑着的时候装了新工具（安装程序把目录加进 PATH、设了 JAVA_HOME 之类），
// Momoko 自己看不到，从它起的终端、Agent 也就找不到新装的命令。Windows 上这些改动都在注册表里，
// Environ 按注册表现在的样子把 Momoko 启动后新增、改过、删掉的变量补上，和在资源管理器里新开一个终端一样；
// Momoko 启动时就有的值（包括启动它的人特意设的）不动。其它系统靠 shell 自己读配置文件，原样用当前进程的。
package sysenv

import (
	"os"
	"runtime"
	"strings"
)

// Environ 新起进程用的环境变量，格式同 os.Environ。
func Environ() []string {
	return refresh(os.Environ())
}

// vars 注册表里的环境变量：键是大写的变量名（Windows 的变量名不分大小写），Path 单独按目录拆开。
type vars struct {
	names  map[string]string // 大写名 → 注册表里写的名字
	values map[string]string // 大写名 → 展开后的值（不含 PATH）
	path   []string          // 系统的 Path 接着用户的 Path，展开后按目录拆开
}

// merge 把注册表从 before（Momoko 启动时）到 after（现在）的变化合进 base（Momoko 自己的环境）。
func merge(base []string, before, after *vars) []string {
	type entry struct{ name, value string }
	env := make(map[string]entry, len(base))
	order := make([]string, 0, len(base))
	for _, kv := range base {
		name, value, ok := strings.Cut(kv, "=")
		// Windows 的环境块里有 =C: 这种以 = 开头的隐藏变量，原样留着
		if !ok || name == "" {
			order = append(order, "\x00"+kv)
			continue
		}
		key := keyOf(name)
		if _, seen := env[key]; !seen {
			order = append(order, key)
		}
		env[key] = entry{name, value}
	}

	for key, value := range after.values {
		old, had := before.values[key]
		cur, has := env[key]
		// 注册表里这一项是 Momoko 启动后新加或改过的，或者 Momoko 的环境里本来就没有它
		if !has || !had || old != value {
			if !has {
				order = append(order, key)
				cur.name = after.names[key]
			}
			env[key] = entry{cur.name, value}
		}
	}
	for key, old := range before.values {
		if _, still := after.values[key]; still {
			continue
		}
		// 启动后从注册表里删掉了：Momoko 的值就是当时注册表里的那个，才跟着删
		if cur, has := env[key]; has && cur.value == old {
			delete(env, key)
		}
	}

	if !samePath(before.path, after.path) || !containsAll(splitPath(env["PATH"].value), after.path) {
		cur := env["PATH"]
		if cur.name == "" {
			cur.name = "Path"
			if _, seen := env["PATH"]; !seen {
				order = append(order, "PATH")
			}
		}
		// Momoko 启动时 PATH 里注册表以外的目录（启动它的程序加的）放在前面，后面是注册表现在的
		var out []string
		seen := map[string]bool{}
		add := func(dir string) {
			if k := dirKey(dir); dir != "" && !seen[k] {
				seen[k] = true
				out = append(out, dir)
			}
		}
		known := map[string]bool{}
		for _, dir := range before.path {
			known[dirKey(dir)] = true
		}
		for _, dir := range after.path {
			known[dirKey(dir)] = true
		}
		for _, dir := range splitPath(cur.value) {
			if !known[dirKey(dir)] {
				add(dir)
			}
		}
		for _, dir := range after.path {
			add(dir)
		}
		env["PATH"] = entry{cur.name, strings.Join(out, string(os.PathListSeparator))}
	}

	out := make([]string, 0, len(order))
	for _, key := range order {
		if raw, hidden := strings.CutPrefix(key, "\x00"); hidden {
			out = append(out, raw)
			continue
		}
		if e, ok := env[key]; ok {
			out = append(out, e.name+"="+e.value)
		}
	}
	return out
}

func keyOf(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(name)
	}
	return name
}

// dirKey 比较 PATH 里两个目录是不是同一个：不分大小写，末尾的斜杠不算。
func dirKey(dir string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(dir), `\/`))
}

func splitPath(path string) []string {
	var out []string
	for _, dir := range strings.Split(path, string(os.PathListSeparator)) {
		if dir = strings.TrimSpace(dir); dir != "" {
			out = append(out, dir)
		}
	}
	return out
}

func samePath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if dirKey(a[i]) != dirKey(b[i]) {
			return false
		}
	}
	return true
}

func containsAll(have, want []string) bool {
	set := make(map[string]bool, len(have))
	for _, dir := range have {
		set[dirKey(dir)] = true
	}
	for _, dir := range want {
		if !set[dirKey(dir)] {
			return false
		}
	}
	return true
}

// expand 展开 %NAME%：先找 lookup 里的，没有的原样留着（和 Windows 一样）。
func expand(s string, lookup func(key string) (string, bool)) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var out strings.Builder
	for {
		start := strings.IndexByte(s, '%')
		if start < 0 {
			break
		}
		end := strings.IndexByte(s[start+1:], '%')
		if end < 0 {
			break
		}
		end += start + 1
		name := s[start+1 : end]
		if value, ok := lookup(keyOf(name)); ok && name != "" {
			out.WriteString(s[:start])
			out.WriteString(value)
			s = s[end+1:]
			continue
		}
		// 不认识的：前一个 % 原样输出，从后一个 % 接着找
		out.WriteString(s[:end])
		s = s[end:]
	}
	out.WriteString(s)
	return out.String()
}
