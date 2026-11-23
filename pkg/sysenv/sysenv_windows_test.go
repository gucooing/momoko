//go:build windows

package sysenv

import (
	"slices"
	"strings"
	"testing"
)

func regVars(path string, kv ...string) *vars {
	v := &vars{names: map[string]string{}, values: map[string]string{}, path: splitPath(path)}
	for i := 0; i+1 < len(kv); i += 2 {
		v.names[strings.ToUpper(kv[i])], v.values[strings.ToUpper(kv[i])] = kv[i], kv[i+1]
	}
	return v
}

func get(env []string, name string) (string, bool) {
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && strings.EqualFold(k, name) {
			return v, true
		}
	}
	return "", false
}

func TestMergeUnchangedKeepsEnvironment(t *testing.T) {
	base := []string{"=C:=C:\\work", "Path=C:\\Windows;C:\\Tools", "GOPATH=D:\\mine", "USERNAME=alice"}
	reg := regVars(`C:\Windows;C:\Tools`, "GOPATH", `C:\Users\alice\go`, "USERNAME", "SYSTEM")
	got := merge(base, reg, reg)
	if !slices.Equal(got, base) {
		t.Fatalf("注册表没变，环境应该原样：%v", got)
	}
}

func TestMergePicksUpInstall(t *testing.T) {
	base := []string{"Path=C:\\Launcher;C:\\Windows;C:\\Old", "GOPATH=D:\\mine", "USERNAME=alice"}
	before := regVars(`C:\Windows;C:\Old`, "GOPATH", `C:\go`, "USERNAME", "SYSTEM", "GONE", "x")
	after := regVars(`C:\Windows;C:\Program Files\Go\bin\`, "GOPATH", `C:\go`, "USERNAME", "SYSTEM", "JAVA_HOME", `C:\jdk`)
	got := merge(append(base, "GONE=x"), before, after)

	path, _ := get(got, "PATH")
	if path != `C:\Launcher;C:\Windows;C:\Program Files\Go\bin\` {
		t.Fatalf("PATH 不对：%q", path)
	}
	if v, _ := get(got, "GOPATH"); v != `D:\mine` {
		t.Fatalf("注册表没改的变量应该保留启动时的值：%q", v)
	}
	if v, _ := get(got, "USERNAME"); v != "alice" {
		t.Fatalf("USERNAME 不该被系统键里的 SYSTEM 盖掉：%q", v)
	}
	if v, ok := get(got, "JAVA_HOME"); !ok || v != `C:\jdk` {
		t.Fatalf("新装的变量应该补上：%q %v", v, ok)
	}
	if _, ok := get(got, "GONE"); ok {
		t.Fatal("注册表里删掉的变量应该跟着删")
	}
}

func TestMergeStalePathAtStartup(t *testing.T) {
	// Momoko 启动时自己的 PATH 就比注册表旧（从一个早就开着的终端里启动的）
	base := []string{"PATH=C:\\Windows"}
	reg := regVars(`C:\Windows;C:\Go\bin`)
	path, _ := get(merge(base, reg, reg), "PATH")
	if path != `C:\Windows;C:\Go\bin` {
		t.Fatalf("PATH 不对：%q", path)
	}
}

func TestExpand(t *testing.T) {
	lookup := func(key string) (string, bool) {
		v, ok := map[string]string{"USERPROFILE": `C:\Users\a`, "JAVA_HOME": `C:\jdk`}[key]
		return v, ok
	}
	cases := map[string]string{
		`%UserProfile%\go\bin`:    `C:\Users\a\go\bin`,
		`%JAVA_HOME%\bin;%NOPE%;`: `C:\jdk\bin;%NOPE%;`,
		`100%`:                    `100%`,
		`%%JAVA_HOME%`:            `%C:\jdk`,
		`plain`:                   `plain`,
	}
	for in, want := range cases {
		if got := expand(in, lookup); got != want {
			t.Errorf("expand(%q) = %q，应该是 %q", in, got, want)
		}
	}
}

func TestReadRegistry(t *testing.T) {
	v := read([]string{"SystemRoot=C:\\Windows"})
	if v == nil {
		t.Fatal("读不出注册表")
	}
	if len(v.path) == 0 {
		t.Fatal("系统 Path 是空的")
	}
	for _, dir := range v.path {
		if strings.Contains(strings.ToUpper(dir), "%SYSTEMROOT%") {
			t.Fatalf("Path 没展开：%q", dir)
		}
	}
}
