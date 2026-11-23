//go:build windows

package sysenv

import (
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const machineKey = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`

// startup Momoko 启动时注册表里的环境变量；读不出来时为 nil，那就不刷新。
var startup = read(os.Environ())

func refresh(base []string) []string {
	if startup == nil {
		return base
	}
	now := read(base)
	if now == nil {
		return base
	}
	return merge(base, startup, now)
}

// read 读注册表里系统和用户的环境变量，按 Windows 登录时的规矩拼起来：先系统后用户，用户的同名变量盖过系统的，
// Path 是系统的接着用户的；%NAME% 用 base 和前面已经读到的变量展开。系统那一项读不出来就当读不出来。
func read(base []string) *vars {
	lookup := map[string]string{}
	for _, kv := range base {
		if name, value, ok := strings.Cut(kv, "="); ok && name != "" {
			lookup[strings.ToUpper(name)] = value
		}
	}
	v := &vars{names: map[string]string{}, values: map[string]string{}}
	machine, err := registry.OpenKey(registry.LOCAL_MACHINE, machineKey, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer machine.Close()
	var userPath []string
	machinePath := v.load(machine, lookup)
	if user, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE); err == nil {
		userPath = v.load(user, lookup)
		user.Close()
	}
	v.path = append(machinePath, userPath...)
	return v
}

// load 读一个键里的变量放进 v（Path 不放，拆成目录返回），读到的也放进 lookup 供后面的展开用。
// 先放不用展开的，再展开 REG_EXPAND_SZ 的，这样 %JAVA_HOME%\bin 这种引用同一处新设的变量也能展开。
func (v *vars) load(k registry.Key, lookup map[string]string) []string {
	names, err := k.ReadValueNames(0)
	if err != nil {
		return nil
	}
	type raw struct {
		name, value string
		expand      bool
	}
	var items []raw
	for _, name := range names {
		value, typ, err := k.GetStringValue(name)
		if err != nil || name == "" {
			continue
		}
		items = append(items, raw{name, value, typ == registry.EXPAND_SZ})
	}
	var path []string
	put := func(r raw) {
		key := strings.ToUpper(r.name)
		value := r.value
		if r.expand {
			value = expand(value, func(key string) (string, bool) {
				s, ok := lookup[key]
				return s, ok
			})
		}
		if key == "PATH" {
			path = splitPath(value)
			return
		}
		v.names[key], v.values[key] = r.name, value
		lookup[key] = value
	}
	for _, r := range items {
		if !r.expand {
			put(r)
		}
	}
	for _, r := range items {
		if r.expand {
			put(r)
		}
	}
	return path
}
