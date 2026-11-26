package sysservice

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSamePath(t *testing.T) {
	a := filepath.Join(t.TempDir(), "Momoko", "momoko")
	if !SamePath(a, a+string(filepath.Separator)) {
		t.Fatal("末尾的分隔符不影响")
	}
	upper := filepath.Join(filepath.Dir(a), "MOMOKO")
	if got := SamePath(a, upper); got != (runtime.GOOS == "windows") {
		t.Fatalf("大小写：Windows 上不区分，其它系统区分，得到 %v", got)
	}
}

func TestEnterInstallDir(t *testing.T) {
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	dir, err := EnterInstallDir()
	if err != nil {
		t.Fatal(err)
	}
	exe, _ := Executable()
	cur, _ := os.Getwd()
	if !SamePath(dir, filepath.Dir(exe)) || !SamePath(cur, dir) {
		t.Fatalf("工作目录应换到程序所在目录：%s / %s", cur, dir)
	}
}

func TestValidateDir(t *testing.T) {
	if err := ValidateDir("momoko"); err == nil {
		t.Fatal("相对路径不行")
	}
	if err := ValidateDir(DefaultDir()); err != nil {
		t.Fatalf("默认目录应该可以：%v", err)
	}
}
