package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"momoko/pkg/version"
)

// 测试程序被当作新程序运行 -version 时冒充 Momoko：正常报版本号；
// 设了 SELFUPDATE_TEST_OLD=1 时像还没有 -version 的老版本一样报错退出。
func TestMain(m *testing.M) {
	for _, a := range os.Args[1:] {
		if a == "-version" {
			if os.Getenv("SELFUPDATE_TEST_OLD") == "1" {
				fmt.Fprintln(os.Stderr, "flag provided but not defined: -version")
				os.Exit(2)
			}
			fmt.Println(VersionLine("momoko", "v9.9.9"))
			os.Exit(0)
		}
	}
	os.Exit(m.Run())
}

// 测试程序本身就是一个 Go 程序，主包路径和 selfModulePath() 一致，可以当作「合法的新程序」。
func selfBinary(t *testing.T) []byte {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInspectBinaryAcceptsSameProgram(t *testing.T) {
	exe, _ := os.Executable()
	info, err := inspectBinary(exe)
	if err != nil {
		t.Fatal(err)
	}
	if info.GOOS != runtime.GOOS || info.GOARCH != runtime.GOARCH || info.Size == 0 || len(info.SHA256) != 64 {
		t.Fatalf("信息不对：%+v", info)
	}
}

func TestInspectBinaryRejectsOtherFiles(t *testing.T) {
	f := filepath.Join(t.TempDir(), "momoko")
	_ = os.WriteFile(f, []byte("#!/bin/sh\necho hi\n"), 0o755)
	if _, err := inspectBinary(f); err == nil {
		t.Fatal("脚本不该通过校验")
	}
}

func TestExtractBinaryFromArchives(t *testing.T) {
	want := []byte("pretend this is momoko")
	dir := t.TempDir()

	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	w, _ := zw.Create("momoko_v1.0.0_windows-amd64/README.md")
	_, _ = w.Write([]byte("readme"))
	w, _ = zw.Create("momoko_v1.0.0_windows-amd64/momoko.exe")
	_, _ = w.Write(want)
	_ = zw.Close()

	var tbuf bytes.Buffer
	gz := gzip.NewWriter(&tbuf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "momoko_v1.0.0_linux-amd64/configs/", Typeflag: tar.TypeDir, Mode: 0o755})
	_ = tw.WriteHeader(&tar.Header{Name: "momoko_v1.0.0_linux-amd64/momoko", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(want))})
	_, _ = tw.Write(want)
	_ = tw.Close()
	_ = gz.Close()

	for name, data := range map[string][]byte{"pkg.zip": zbuf.Bytes(), "pkg.tar.gz": tbuf.Bytes(), "momoko": want} {
		src := filepath.Join(dir, name)
		dst := filepath.Join(dir, name+".out")
		_ = os.WriteFile(src, data, 0o644)
		if err := extractBinary(src, dst); err != nil {
			t.Fatalf("%s：%v", name, err)
		}
		got, _ := os.ReadFile(dst)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s 解出来的内容不对", name)
		}
	}

	var empty bytes.Buffer
	zw = zip.NewWriter(&empty)
	w, _ = zw.Create("other/tool")
	_, _ = w.Write(want)
	_ = zw.Close()
	src := filepath.Join(dir, "empty.zip")
	_ = os.WriteFile(src, empty.Bytes(), 0o644)
	if err := extractBinary(src, src+".out"); err == nil || !strings.Contains(err.Error(), "没有找到") {
		t.Fatalf("没有主程序的压缩包应该报错，得到 %v", err)
	}
}

func TestReplaceExecutableKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "momoko")
	staged := filepath.Join(dir, ".momoko-update-1.new")
	_ = os.WriteFile(exe, []byte("old"), 0o755)
	_ = os.WriteFile(staged, []byte("new"), 0o644)
	_ = os.WriteFile(backupPath(exe), []byte("older"), 0o755)
	if err := replaceExecutable(exe, staged); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "new" {
		t.Fatalf("程序没换上：%q", got)
	}
	if got, _ := os.ReadFile(backupPath(exe)); string(got) != "old" {
		t.Fatalf("备份不对：%q", got)
	}
	if err := restoreBackup(exe); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Fatalf("没换回旧程序：%q", got)
	}
}

func TestPickAssets(t *testing.T) {
	target := runtime.GOOS + "-" + runtime.GOARCH
	rel := &version.Release{TagName: "v1.2.3", Assets: []version.Asset{
		{Name: "momoko_v1.2.3_plan9-mips.tar.gz"},
		{Name: "momoko_v1.2.3_" + target + ".zip"},
		{Name: "momoko_v1.2.3_checksums.txt"},
	}}
	pkg, sums := pickAssets(rel)
	if pkg == nil || !strings.Contains(pkg.Name, target) || sums == nil {
		t.Fatalf("没挑对：%v %v", pkg, sums)
	}
	if HasAsset(&version.Release{Assets: []version.Asset{{Name: "momoko_v1_plan9-mips.zip"}}}) {
		t.Fatal("没有本机的包时应该返回 false")
	}
}

func TestParseChecksums(t *testing.T) {
	list, err := parseChecksums(strings.NewReader("ABC  momoko_v1_linux-amd64.tar.gz\ndef *momoko_v1_windows-amd64.zip\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if list["momoko_v1_linux-amd64.tar.gz"] != "abc" || list["momoko_v1_windows-amd64.zip"] != "def" {
		t.Fatalf("解析不对：%v", list)
	}
}

func TestUploadValidatesAndCanBeCanceled(t *testing.T) {
	u := New()
	if !u.Status().Supported {
		t.Skip("测试程序所在目录不可写：" + u.Status().UnsupportedReason)
	}
	if _, err := u.Upload(strings.NewReader("not a program"), -1, "momoko", "tester"); err == nil {
		t.Fatal("随便一个文件不该通过校验")
	}
	if st := u.Status(); st.State != StateIdle {
		t.Fatalf("上传失败后应回到空闲，得到 %v", st.State)
	}

	data := selfBinary(t)
	st, err := u.Upload(bytes.NewReader(data), int64(len(data)), "momoko", "tester")
	if err != nil {
		t.Fatal(err)
	}
	if st.State != StateReady || st.Package == nil || st.Package.FileName != "momoko" || st.Downloaded != int64(len(data)) {
		t.Fatalf("状态不对：%+v", st)
	}
	if st.Package.Version != "v9.9.9" || st.TargetVersion != "v9.9.9" {
		t.Fatalf("版本号应以 -version 报的为准，得到 %q", st.Package.Version)
	}
	staged := u.staged
	if _, err := os.Stat(staged); err != nil {
		t.Fatal("暂存的新程序不在")
	}
	if _, err := u.Upload(bytes.NewReader(data), -1, "momoko", "tester"); err != nil {
		t.Fatalf("再次上传应替换掉等待安装的那个：%v", err)
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatal("旧的暂存文件应该删掉")
	}
	if st, err := u.Cancel(); err != nil || st.State != StateIdle {
		t.Fatalf("取消后应回到空闲：%v %v", st.State, err)
	}
	if _, err := u.Apply("tester"); err == nil {
		t.Fatal("没有等待安装的更新时不能安装")
	}
}

func TestVersionLineRoundTrip(t *testing.T) {
	line := VersionLine("", "v1.2.3")
	ver, platform, ok := ParseVersionLine("一些别的输出\n" + line + "\n")
	if !ok || ver != "v1.2.3" || platform != Platform() {
		t.Fatalf("解析不对：%q → %q %q %v", line, ver, platform, ok)
	}
	if _, _, ok := ParseVersionLine("Usage of momoko:\n  -conf string"); ok {
		t.Fatal("用法说明不该被当成版本号")
	}
}

func TestProbeRejectsOldVersionWithoutFlag(t *testing.T) {
	exe, _ := os.Executable()
	if ver, err := probeBinary(exe); err != nil || ver != "v9.9.9" {
		t.Fatalf("支持 -version 的程序应该通过：%q %v", ver, err)
	}
	t.Setenv("SELFUPDATE_TEST_OLD", "1")
	_, err := probeBinary(exe)
	if err == nil || !strings.Contains(err.Error(), "老版本") {
		t.Fatalf("不认识 -version 的老版本应该被拒绝，得到 %v", err)
	}
	u := New()
	if !u.Status().Supported {
		t.Skip("测试程序所在目录不可写")
	}
	if _, err := u.Upload(bytes.NewReader(selfBinary(t)), -1, "momoko", "tester"); err == nil || !strings.Contains(err.Error(), "-version") {
		t.Fatalf("上传老版本应该被拒绝，得到 %v", err)
	}
}

func TestReplaceExecutableMovesLockedBackupAside(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "momoko")
	staged := filepath.Join(dir, ".momoko-update-2.new")
	_ = os.WriteFile(exe, []byte("v2"), 0o755)
	_ = os.WriteFile(staged, []byte("v3"), 0o644)
	// 模拟「删不掉的备份」：把 .old 做成一个非空目录（os.Remove 删不掉，但能改名）
	_ = os.MkdirAll(filepath.Join(backupPath(exe), "busy"), 0o755)
	if err := replaceExecutable(exe, staged); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "v3" {
		t.Fatalf("程序没换上：%q", got)
	}
	if got, _ := os.ReadFile(backupPath(exe)); string(got) != "v2" {
		t.Fatalf("备份不对：%q", got)
	}
	aside, _ := filepath.Glob(staleBackupPattern(exe))
	if len(aside) != 1 {
		t.Fatalf("删不掉的旧备份应该被挪开：%v", aside)
	}
}
