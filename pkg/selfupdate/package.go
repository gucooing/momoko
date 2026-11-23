package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// maxBinarySize 解出的程序文件大小上限，防止压缩炸弹撑满磁盘。
const maxBinarySize = 1 << 30

// binaryNames 发行包里主程序的文件名（见 .github/workflows/build.yml）。
var binaryNames = map[string]struct{}{"momoko": {}, "momoko.exe": {}}

// PackageInfo 是一个程序文件的信息，读自 Go 写进程序里的构建信息。
type PackageInfo struct {
	Version    string    // 版本号（读不到时为空）
	GOOS       string    // 目标系统
	GOARCH     string    // 目标架构
	Revision   string    // 构建时的代码提交
	CommitTime time.Time // 代码提交时间
	Size       int64     // 程序文件大小
	SHA256     string    // 程序文件 SHA256
	FileName   string    // 上传的文件名 / 发行版里的包名
}

// fileKind 按文件开头几个字节判断是 zip、tar.gz 还是程序本身。
func fileKind(f *os.File) (string, error) {
	head := make([]byte, 4)
	n, err := f.ReadAt(head, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	head = head[:n]
	switch {
	case len(head) >= 4 && string(head) == "PK\x03\x04":
		return "zip", nil
	case len(head) >= 2 && head[0] == 0x1f && head[1] == 0x8b:
		return "tar.gz", nil
	default:
		return "binary", nil
	}
}

// extractBinary 把 src（程序本身，或装着程序的 zip / tar.gz）里的主程序写到 dst。
func extractBinary(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	kind, err := fileKind(f)
	if err != nil {
		return err
	}
	switch kind {
	case "zip":
		st, err := f.Stat()
		if err != nil {
			return err
		}
		zr, err := zip.NewReader(f, st.Size())
		if err != nil {
			return fmt.Errorf("压缩包打不开：%w", err)
		}
		for _, zf := range zr.File {
			if zf.FileInfo().IsDir() || !isBinaryName(zf.Name) {
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				return fmt.Errorf("压缩包里的 %s 读不出来：%w", zf.Name, err)
			}
			defer rc.Close()
			return writeLimited(dst, rc)
		}
		return errors.New("压缩包里没有找到 momoko 主程序")
	case "tar.gz":
		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("压缩包打不开：%w", err)
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			hdr, err := tr.Next()
			if errors.Is(err, io.EOF) {
				return errors.New("压缩包里没有找到 momoko 主程序")
			}
			if err != nil {
				return fmt.Errorf("压缩包读不出来：%w", err)
			}
			if hdr.Typeflag != tar.TypeReg || !isBinaryName(hdr.Name) {
				continue
			}
			return writeLimited(dst, tr)
		}
	default:
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return writeLimited(dst, f)
	}
}

func isBinaryName(name string) bool {
	_, ok := binaryNames[path.Base(strings.ReplaceAll(name, "\\", "/"))]
	return ok
}

func writeLimited(dst string, r io.Reader) error {
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(r, maxBinarySize+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > maxBinarySize {
		return errors.New("程序文件太大")
	}
	return nil
}

// extractAndInspect 从 src 里解出主程序写到 staged，再校验它是本机能用的 Momoko：
// 先读 Go 写在程序里的构建信息（主包、系统、架构），再真的运行一次 -version，版本号以它报的为准。
func extractAndInspect(src, staged string) (*PackageInfo, error) {
	if err := extractBinary(src, staged); err != nil {
		return nil, err
	}
	// 暂存文件是 CreateTemp 建的（0600），要能运行
	if err := os.Chmod(staged, 0o755); err != nil {
		return nil, err
	}
	info, err := inspectBinary(staged)
	if err != nil {
		return nil, err
	}
	ver, err := probeBinary(staged)
	if err != nil {
		return nil, err
	}
	info.Version = ver
	return info, nil
}

// ldflagsVersion 从 -ldflags 里取 -X main.Version=…（不带 -trimpath 构建时 Go 会记下 -ldflags）。
var ldflagsVersion = regexp.MustCompile(`-X[ =]?main\.Version=([^\s"']+)`)

// selfModulePath 当前程序的主包路径，校验上传的文件是不是同一个程序。
func selfModulePath() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Path != "" {
		return bi.Path
	}
	return "momoko/cmd/momoko"
}

// inspectBinary 校验 file 是本机能用的 Momoko 主程序，并读出版本等信息。
func inspectBinary(file string) (*PackageInfo, error) {
	bi, err := buildinfo.ReadFile(file)
	if err != nil {
		return nil, errors.New("不是 Go 编译的程序文件，或者文件已损坏")
	}
	if want := selfModulePath(); bi.Path != want {
		return nil, fmt.Errorf("不是 Momoko 主程序（这个文件是 %s）", bi.Path)
	}
	info := &PackageInfo{}
	settings := map[string]string{}
	for _, s := range bi.Settings {
		settings[s.Key] = s.Value
	}
	info.GOOS, info.GOARCH = settings["GOOS"], settings["GOARCH"]
	if info.GOOS != runtime.GOOS || info.GOARCH != runtime.GOARCH {
		return nil, fmt.Errorf("这个程序是给 %s/%s 用的，本机是 %s/%s", info.GOOS, info.GOARCH, runtime.GOOS, runtime.GOARCH)
	}
	if m := ldflagsVersion.FindStringSubmatch(settings["-ldflags"]); m != nil {
		info.Version = m[1]
	} else if v := bi.Main.Version; v != "" && v != "(devel)" {
		// Go 1.24 起按 git tag 给主模块记版本（如 v0.1.38、v0.1.38+dirty）
		info.Version = v
	}
	info.Revision = settings["vcs.revision"]
	if t, err := time.Parse(time.RFC3339, settings["vcs.time"]); err == nil {
		info.CommitTime = t
	}
	sum, size, err := fileSHA256(file)
	if err != nil {
		return nil, err
	}
	info.SHA256, info.Size = sum, size
	return info, nil
}

func fileSHA256(file string) (string, int64, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// parseChecksums 解析 sha256sum 输出（「哈希  文件名」，二进制模式文件名前带 *）。
func parseChecksums(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		out[strings.TrimPrefix(fields[1], "*")] = strings.ToLower(fields[0])
	}
	return out, sc.Err()
}
