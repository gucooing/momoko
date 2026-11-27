// Package agentinstall 实现「Momoko 下载并更新」：从 GitHub 发布页把 ACP 适配器下载到 Momoko 数据目录下的一个目录，
// 报告已装的版本和最新版本，并告诉调用方怎么启动。下载哪个仓库、哪个文件由调用方给的 Release 决定
// （见 pkg/agentdriver 下各家的驱动），这里只管下载、校验、解压和原子替换。
package agentinstall

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Release 一个能由 Momoko 下载的发行包：某个 GitHub 仓库最新发布里的一个文件，以及里面要用到的可执行文件。
type Release struct {
	Name  string   // 给人看的名称
	Repo  string   // GitHub 仓库，版本号看它的发布
	Asset string   // 发布页上这个系统对应的文件名（.zip 或 .tar.gz）；里面的 {version} 换成版本号（不带 v）
	Files []string // 包里的可执行文件（不带 .exe），第一个是入口；包里只有一个顶层目录时相对那个目录
}

// Launch 怎么启动一份已安装的适配器。
type Launch struct {
	Version string // 已安装的版本，例如 0.108.1
	Command string // 入口可执行文件的绝对路径
}

// Platform 这个系统在安装包文件名里的叫法：linux / darwin / windows，x64 / arm64。不认识的系统 ok 为 false。
func Platform() (goos, arch string, ok bool) {
	p, ok := map[string][2]string{
		"linux/amd64":   {"linux", "x64"},
		"linux/arm64":   {"linux", "arm64"},
		"darwin/amd64":  {"darwin", "x64"},
		"darwin/arm64":  {"darwin", "arm64"},
		"windows/amd64": {"windows", "x64"},
		"windows/arm64": {"windows", "arm64"},
	}[runtime.GOOS+"/"+runtime.GOARCH]
	return p[0], p[1], ok
}

// Musl 这台 Linux 用的是不是 musl（Alpine 等）：按 glibc 构建的程序在上面跑不起来。
func Musl() bool {
	if _, err := os.Stat("/etc/alpine-release"); err == nil {
		return true
	}
	if found, _ := filepath.Glob("/lib/ld-musl-*"); len(found) == 0 {
		return false
	}
	// glibc 的发行版也可能另外装了 musl：两种加载器都在时按 glibc 算。
	for _, pattern := range []string{"/lib/ld-linux-*", "/lib64/ld-linux-*"} {
		if found, _ := filepath.Glob(pattern); len(found) > 0 {
			return false
		}
	}
	return true
}

const (
	// 整个安装（下载、解压）最多这么久。
	installTimeout = 30 * time.Minute
	// 只查版本号最多这么久。
	lookupTimeout = time.Minute
	markerName    = "momoko-install.json"
)

// marker 是安装目录里的一个小文件，记着装的是什么：有了它，Installed 不用联网、也不用启动进程。
// 里面的路径都相对安装目录。
type marker struct {
	Repo    string   `json:"repo"`
	Version string   `json:"version"` // 不带 v
	Files   []string `json:"files"`   // 第一个是入口
	Source  string   `json:"source"`  // 是从哪个安装包装出来的
}

func readMarker(dir string) (marker, bool) {
	var m marker
	raw, err := os.ReadFile(filepath.Join(dir, markerName))
	if err != nil || json.Unmarshal(raw, &m) != nil || m.Version == "" || len(m.Files) == 0 {
		return m, false
	}
	for _, f := range m.Files {
		// 文件被人删了、挪了，就当没装过。
		if !filepath.IsLocal(f) {
			return m, false
		}
		if info, err := os.Stat(filepath.Join(dir, f)); err != nil || !info.Mode().IsRegular() {
			return m, false
		}
	}
	return m, true
}

// Installed 读 dir 里已经装好的那一份；没装过返回 ok=false。
func Installed(dir string) (launch *Launch, ok bool) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, false
	}
	m, ok := readMarker(dir)
	if !ok {
		return nil, false
	}
	return &Launch{Version: m.Version, Command: filepath.Join(dir, m.Files[0])}, true
}

// Latest 查最新版本号。
func Latest(ctx context.Context, rel Release) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, lookupTimeout)
		defer cancel()
	}
	r, err := latestRelease(ctx, rel.Repo)
	if err != nil {
		return "", err
	}
	return r.version(), nil
}

// Install 把最新版本装进 dir（已是最新就直接返回）。progress 可为 nil，用来上报「正在下载 42%」这类文字。
// 下载到临时目录，装好后再替换，失败不破坏已有的那一份。
//
// 替换时旧的那份会被删掉，所以调用方应当挑没有会话在用的时候更新。
// Windows 上发现装着的程序还在运行就不换，返回「有会话正在使用…」；别的系统查不出来，照常替换。
func Install(ctx context.Context, rel Release, dir string, progress func(text string)) (*Launch, error) {
	if rel.Asset == "" || len(rel.Files) == 0 {
		return nil, fmt.Errorf("%s 没有适用于这个系统的安装包", rel.Name)
	}
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("安装目录还没指定")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.New("安装目录不对")
	}
	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	unlock, err := lock(ctx, dir)
	if err != nil {
		return nil, err
	}
	defer unlock()
	report := &reporter{fn: progress}

	report.say("正在查询最新版本")
	r, err := latestRelease(ctx, rel.Repo)
	if err != nil {
		return nil, err
	}
	m := marker{Repo: rel.Repo, Version: r.version()}
	if cur, ok := Installed(dir); ok && !Newer(m.Version, cur.Version) {
		return cur, nil
	}
	a, ok := r.find(strings.ReplaceAll(rel.Asset, "{version}", m.Version))
	if !ok {
		return nil, fmt.Errorf("%s 没有适用于这个系统的安装包", rel.Name)
	}
	m.Source = a.Name
	busy := fmt.Errorf("有会话正在使用 %s，等它们空闲后再更新", rel.Name)
	// 先看一眼换不换得掉，免得白下载几百兆。
	if running(dir) {
		return nil, busy
	}

	sweep(dir)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, fmt.Errorf("建不了安装目录：%s", reason(err))
	}
	tmp := dir + ".tmp-" + suffix()
	if err := os.Mkdir(tmp, 0o755); err != nil {
		return nil, fmt.Errorf("建不了安装目录：%s", reason(err))
	}
	defer os.RemoveAll(tmp) // 装成功后它已经被改名成 dir，这里删不到东西

	// 安装包放在 tmp 旁边而不是里面：tmp 整个会变成安装目录。
	report.say("正在下载 0%")
	archive := tmp + ".download"
	defer os.Remove(archive)
	if err := fetchAsset(ctx, fmt.Sprintf("下载 %s %s", rel.Name, m.Version), a, archive, func(n int64) { report.percent("正在下载", n, a.Size) }); err != nil {
		return nil, ctxError(ctx, err)
	}
	report.say("正在解压")
	if err := extract(ctx, archive, a.Name, tmp); err != nil {
		return nil, ctxError(ctx, fmt.Errorf("解压失败：%s", reason(err)))
	}

	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
	// 包里只有一个顶层目录（sing-box-1.14.5-linux-amd64/ 这样）时，文件在那个目录里
	base := ""
	if entries, err := os.ReadDir(tmp); err == nil && len(entries) == 1 && entries[0].IsDir() {
		base = entries[0].Name()
	}
	for _, f := range rel.Files {
		f = filepath.Join(base, f)
		full := filepath.Join(tmp, f+exe)
		if info, err := os.Stat(full); err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("安装包 %s 里没有 %s", a.Name, f+exe)
		}
		// zip 不一定带着可执行位。
		if exe == "" {
			if err := os.Chmod(full, 0o755); err != nil {
				return nil, fmt.Errorf("解压失败：%s", reason(err))
			}
		}
		m.Files = append(m.Files, f+exe)
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(tmp, markerName), raw, 0o644)
	}
	if err != nil {
		return nil, fmt.Errorf("写不了安装记录：%s", reason(err))
	}

	// 下载要好几分钟，这期间可能又有会话开起来了，换之前再看一次。
	if running(dir) {
		return nil, busy
	}
	if err := swap(dir, tmp); err != nil {
		if errors.Is(err, errInUse) {
			return nil, busy
		}
		return nil, fmt.Errorf("替换 %s 的安装目录失败：%s", rel.Name, reason(err))
	}
	launch, ok := Installed(dir)
	if !ok {
		return nil, fmt.Errorf("%s 装好了，但启动文件不在，再装一次试试", rel.Name)
	}
	return launch, nil
}

// errInUse 旧的那一份让不出位置：里面有程序还在运行。
var errInUse = errors.New("agentinstall: in use")

// swap 用装好的 tmp 换掉 dir：旧的先改名让位，新的到位后再删旧的；新的放不上去就把旧的放回来。
func swap(dir, tmp string) error {
	old := ""
	if _, err := os.Lstat(dir); err == nil {
		old = dir + ".old-" + suffix()
		if err := rename(dir, old, 2*time.Second); err != nil {
			if locked(err) {
				return errInUse
			}
			return err
		}
	}
	// 刚解出来的程序文件可能正被杀毒软件扫着，改不了名就等一小会儿。
	if err := rename(tmp, dir, 5*time.Second); err != nil {
		if old != "" {
			_ = rename(old, dir, 2*time.Second)
		}
		return err
	}
	if old != "" {
		// 删不掉（Windows 上偶尔还有文件被占着）也不要紧，下次安装时 sweep 会再清一次。
		_ = os.RemoveAll(old)
	}
	return nil
}

// rename 改名。Windows 上文件被占着时改不了，在 patience 之内隔一会儿再试。
func rename(from, to string, patience time.Duration) error {
	deadline := time.Now().Add(patience)
	for {
		err := os.Rename(from, to)
		if err == nil || runtime.GOOS != "windows" || !locked(err) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// running dir 里装的程序是不是有正在运行的。只在 Windows 上查。
//
// Windows 允许给一个里面有程序在运行的目录改名，所以替换本身不会失败；但旧目录删不掉，
// 还在运行的适配器之后按老路径启动的会是新版本的 CLI，两边对不上。所以有程序在运行就不更新。
// 别的系统上运行中的程序不受替换影响，不用查。
func running(dir string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".exe") {
			continue
		}
		// 正在运行的程序文件不让以写方式打开。
		file, err := os.OpenFile(filepath.Join(dir, e.Name()), os.O_WRONLY, 0)
		if err == nil {
			file.Close()
		} else if inUse(err) {
			return true
		}
	}
	return false
}

// sweep 清掉以前安装中途被打断留下的临时目录、安装包和没删干净的旧版本：一份就是几百兆。
func sweep(dir string) {
	parent, base := filepath.Dir(dir), filepath.Base(dir)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base+".tmp-") || strings.HasPrefix(e.Name(), base+".old-") {
			_ = os.RemoveAll(filepath.Join(parent, e.Name()))
		}
	}
}

func suffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var (
	locksMu sync.Mutex
	locks   = map[string]chan struct{}{}
)

// lock 同一个目录同时只装一份：并发的第二次调用等前一次装完，然后多半发现已经是最新的了。
func lock(ctx context.Context, dir string) (unlock func(), err error) {
	locksMu.Lock()
	ch, ok := locks[dir]
	if !ok {
		ch = make(chan struct{}, 1)
		locks[dir] = ch
	}
	locksMu.Unlock()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, ctxError(ctx, ctx.Err())
	}
}

// reporter 上报进度；百分比变化很快，每半秒最多报一次。
type reporter struct {
	fn   func(text string)
	last time.Time
}

func (r *reporter) say(text string) {
	if r.fn != nil {
		r.last = time.Now()
		r.fn(text)
	}
}

func (r *reporter) percent(label string, done, total int64) {
	if r.fn == nil || total <= 0 || time.Since(r.last) < 500*time.Millisecond {
		return
	}
	r.say(fmt.Sprintf("%s %d%%", label, min(done*100/total, 100)))
}

// ctxError 安装被取消或超时的时候，换成对应的说法；其余错误原样返回。
func ctxError(ctx context.Context, err error) error {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return context.Canceled
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return errors.New("安装超时了，稍后再试")
	}
	return err
}
