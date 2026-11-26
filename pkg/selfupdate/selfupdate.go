// Package selfupdate 在线更新 Momoko 自己：从 GitHub 发行版下载本机系统对应的包（走统一下载管理，能暂停、继续、取消），
// 或者接收上传的程序文件；校验（是不是 Momoko 主程序、系统和架构对不对、发行版的 SHA256）后替换程序文件，
// 再由 main 正常停服后退出，由系统服务（Windows 服务 / systemd / launchd）把新程序拉起来。
// 只有用 momoko install 装成系统服务、并由服务管理器启动时才能在线更新：直接运行的没人负责重启，只能手动替换程序。
// 被替换下来的旧程序留在程序旁边（momoko.old），新版本有问题时可以手动换回去。
package selfupdate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"momoko/pkg/constant"
	"momoko/pkg/download"
	"momoko/pkg/sysservice"
	"momoko/pkg/version"
)

// notInstalledReason 不是作为系统服务运行时不能在线更新的原因。
const notInstalledReason = "Momoko 没有安装成系统服务，更新后没人把新程序拉起来。在服务器上运行 momoko install（会申请管理员 / root 权限）装好后就能在线更新；" +
	"现在这样直接运行的，请下载新版本，停掉 Momoko 后手动替换程序文件"

// managed 当前进程是不是由服务管理器启动的（测试里可以换掉）。
var managed = sysservice.Managed

// State 更新进行到哪一步。
type State int

const (
	StateIdle        State = iota // 没有进行中的更新
	StateDownloading              // 正在下载 / 接收上传
	StateReady                    // 上传的文件校验通过，等确认安装
	StateInstalling               // 正在替换程序
	StateRestarting               // 已替换，等重启
	StateFailed                   // 失败
)

// Source 更新包从哪来。
type Source int

const (
	SourceGitHub Source = iota
	SourceUpload
)

// DownloadSource 是更新在下载管理里的来源标记。
const DownloadSource = "system.update"

// ErrBusy 已有更新在进行。
var ErrBusy = errors.New("已有一个更新在进行，请等它完成或先取消")

// BootID 本次进程启动的标识，重启后会变（前端据此判断重启完成）。
var BootID = newBootID()

func newBootID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Status 是更新的当前状态。
type Status struct {
	State             State
	Source            Source
	TargetVersion     string
	Downloaded        int64 // 已下载 / 已接收的字节数
	Total             int64 // 总字节数（未知为 0）
	Error             string
	Package           *PackageInfo
	StartedAt         time.Time
	Operator          string
	Executable        string
	Supported         bool
	UnsupportedReason string
	Backup            string
	Download          *download.Info // 从 GitHub 下载时，下载管理里的那条下载
}

// stagedSuffix 暂存的新程序的后缀：Windows 上要带 .exe 才能运行（安装前要先跑一次 -version）。
var stagedSuffix = func() string {
	if runtime.GOOS == "windows" {
		return ".new.exe"
	}
	return ".new"
}()

// Platform 本机系统/架构，如 linux/amd64。
func Platform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// Updater 管理一次在线更新（同一时间只有一个）。
type Updater struct {
	mu        sync.Mutex
	st        Status
	job       int                // 每开始 / 取消一次更新加一，旧任务回来时据此丢弃结果
	cancel    context.CancelFunc // 停掉正在接收的上传
	dl        *download.Download // 正在从 GitHub 下载的那条
	staged    string             // 解出、校验过的新程序（与程序同目录的临时文件）
	received  atomic.Int64       // 上传已接收的字节数
	exe       string
	unsupport string
	client    *http.Client
}

// New 创建更新器，并清理上次更新中断时留下的临时文件。
func New() *Updater {
	u := &Updater{client: &http.Client{Timeout: 30 * time.Second}}
	exe, err := executablePath()
	if err != nil {
		u.unsupport = "找不到当前程序文件：" + err.Error()
		return u
	}
	u.exe = exe
	removeStaleBackups(exe)
	if leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(exe), ".momoko-update-*")); len(leftovers) > 0 {
		for _, f := range leftovers {
			_ = os.Remove(f)
		}
	}
	u.refreshSupported()
	return u
}

// refreshSupported 重新检查能不能在线更新：要作为系统服务运行，且程序目录能写（调用方持锁或在初始化时调用）。
func (u *Updater) refreshSupported() {
	if u.exe == "" {
		return
	}
	if !managed() {
		u.unsupport = notInstalledReason
	} else if err := checkWritable(filepath.Dir(u.exe)); err != nil {
		u.unsupport = err.Error()
	} else {
		u.unsupport = ""
	}
}

// Status 返回当前状态。
func (u *Updater) Status() Status {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.statusLocked()
}

func (u *Updater) statusLocked() Status {
	st := u.st
	if st.Package != nil {
		pkg := *st.Package
		st.Package = &pkg
	}
	if st.State == StateDownloading {
		if st.Source == SourceGitHub && u.dl != nil {
			info := u.dl.Info()
			st.Download = &info
			st.Downloaded, st.Total = info.Downloaded, info.Total
		} else if st.Source == SourceUpload {
			st.Downloaded = u.received.Load()
		}
	}
	st.Executable = u.exe
	st.Supported = u.unsupport == ""
	st.UnsupportedReason = u.unsupport
	if u.exe != "" {
		if _, err := os.Stat(backupPath(u.exe)); err == nil {
			st.Backup = backupPath(u.exe)
		}
	}
	return st
}

// begin 开始一次新的更新：检查没有别的更新在跑、程序目录可写，丢掉等待安装的旧文件。调用方持锁。
func (u *Updater) begin(source Source, operator string) (int, error) {
	switch u.st.State {
	case StateDownloading, StateInstalling, StateRestarting:
		return 0, ErrBusy
	}
	u.refreshSupported()
	if u.unsupport != "" {
		return 0, errors.New(u.unsupport)
	}
	u.dropStagedLocked()
	u.job++
	u.dl = nil
	u.cancel = nil
	u.received.Store(0)
	u.st = Status{State: StateDownloading, Source: source, StartedAt: time.Now(), Operator: operator}
	return u.job, nil
}

// dropStagedLocked 删掉暂存的新程序。调用方持锁。
func (u *Updater) dropStagedLocked() {
	if u.staged != "" {
		_ = os.Remove(u.staged)
		u.staged = ""
	}
}

// tempFile 在程序旁边建一个临时文件（同一个分区，替换时改名是原子的；启动时会清掉残留）。
func (u *Updater) tempFile(suffix string) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(u.exe), ".momoko-update-*"+suffix)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败：%w", err)
	}
	name := f.Name()
	_ = f.Close()
	return name, nil
}

// fail 把第 job 次更新标记为失败（job 已被取消或替换时忽略）。
func (u *Updater) fail(job int, err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if job != u.job {
		return
	}
	u.dropStagedLocked()
	u.st.State = StateFailed
	u.st.Error = err.Error()
	u.dl = nil
	u.cancel = nil
}

// StartRelease 从 GitHub 发行版更新：在后台下载本机系统对应的包，校验通过后直接安装并重启。
// ctx 只用来记下载的发起人，下载不跟着请求结束。
func (u *Updater) StartRelease(ctx context.Context, rel *version.Release, operator string) error {
	asset, sums := pickAssets(rel)
	if asset == nil {
		return fmt.Errorf("发行版 %s 里没有 %s 的程序包，请到发行版页面手动下载", rel.TagName, Platform())
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	job, err := u.begin(SourceGitHub, operator)
	if err != nil {
		return err
	}
	u.st.TargetVersion = rel.TagName
	u.st.Total = asset.Size
	go u.runRelease(context.WithoutCancel(ctx), job, rel.TagName, *asset, sums)
	return nil
}

// pickAssets 找本机系统对应的程序包和校验和文件（命名见 .github/workflows/build.yml）。
func pickAssets(rel *version.Release) (pkg, sums *version.Asset) {
	target := "_" + runtime.GOOS + "-" + runtime.GOARCH
	for i := range rel.Assets {
		a := &rel.Assets[i]
		switch {
		case strings.HasSuffix(a.Name, "_checksums.txt"):
			sums = a
		case strings.HasPrefix(a.Name, "momoko_") && (strings.HasSuffix(a.Name, target+".zip") || strings.HasSuffix(a.Name, target+".tar.gz")):
			pkg = a
		}
	}
	return pkg, sums
}

// HasAsset 发行版里有没有本机系统能用的包。
func HasAsset(rel *version.Release) bool {
	pkg, _ := pickAssets(rel)
	return pkg != nil
}

func (u *Updater) runRelease(ctx context.Context, job int, tag string, asset version.Asset, sums *version.Asset) {
	want, err := u.expectedSHA256(asset, sums)
	if err != nil {
		u.fail(job, err)
		return
	}
	archive, err := u.tempFile(".download")
	if err != nil {
		u.fail(job, err)
		return
	}
	defer os.Remove(archive)
	d, err := download.Default.Start(ctx, download.Request{
		URL:       asset.BrowserDownloadURL,
		Dest:      archive,
		Name:      asset.Name,
		Title:     "更新 Momoko 到 " + tag,
		Source:    DownloadSource,
		Perm:      string(constant.SystemUpdate),
		Size:      asset.Size,
		SHA256:    want,
		MaxSize:   maxBinarySize,
		UserAgent: "momoko-updater",
		Site:      "GitHub",
	})
	if err != nil {
		u.fail(job, err)
		return
	}
	u.mu.Lock()
	if job != u.job {
		u.mu.Unlock()
		_ = d.Cancel()
		return
	}
	u.dl = d
	u.mu.Unlock()

	if err := d.Wait(context.Background()); err != nil {
		if errors.Is(err, download.ErrCanceled) {
			// 在下载管理里取消的：这次更新就算了，不记失败
			u.mu.Lock()
			if job == u.job {
				u.job++
				u.dl = nil
				u.st = Status{}
			}
			u.mu.Unlock()
			return
		}
		u.fail(job, fmt.Errorf("下载 %s 失败：%w", asset.Name, err))
		return
	}

	staged, err := u.tempFile(stagedSuffix)
	if err != nil {
		u.fail(job, err)
		return
	}
	info, err := extractAndInspect(archive, staged)
	if err != nil {
		_ = os.Remove(staged)
		u.fail(job, err)
		return
	}
	info.FileName = asset.Name
	if info.Version == "" {
		info.Version = tag
	}

	u.mu.Lock()
	if job != u.job {
		u.mu.Unlock()
		_ = os.Remove(staged)
		return
	}
	final := d.Info()
	u.staged = staged
	u.dl = nil
	u.st.Package = info
	u.st.Downloaded, u.st.Total = final.Downloaded, final.Total
	u.st.State = StateInstalling
	u.mu.Unlock()
	u.install(job)
}

// expectedSHA256 发行版公布的 SHA256：优先用 GitHub 给附件算的 digest，没有就读发行版里的校验和文件。
func (u *Updater) expectedSHA256(asset version.Asset, sums *version.Asset) (string, error) {
	if want, ok := strings.CutPrefix(asset.Digest, "sha256:"); ok && want != "" {
		return want, nil
	}
	if sums == nil {
		return "", fmt.Errorf("发行版没有公布 %s 的校验值，不能确认下载的文件是对的，请到发行版页面手动下载后上传", asset.Name)
	}
	req, err := http.NewRequest(http.MethodGet, sums.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "momoko-updater")
	resp, err := u.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载校验和文件失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载校验和文件失败：GitHub 返回 %s", resp.Status)
	}
	list, err := parseChecksums(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("读取校验和文件失败：%w", err)
	}
	want, ok := list[asset.Name]
	if !ok {
		return "", fmt.Errorf("校验和文件里没有 %s", asset.Name)
	}
	return want, nil
}

// progressWriter 记录写入字节数，并在取消后中断。
type progressWriter struct {
	ctx context.Context
	w   io.Writer
	n   *atomic.Int64
}

func (p *progressWriter) Write(b []byte) (int, error) {
	if err := p.ctx.Err(); err != nil {
		return 0, errors.New("已取消")
	}
	n, err := p.w.Write(b)
	p.n.Add(int64(n))
	return n, err
}

// Upload 接收上传的程序文件（也可以是发行版的 .zip / .tar.gz 包），校验通过后等确认安装。
// size 为请求声明的大小（未知时传 -1），只用于显示进度。
func (u *Updater) Upload(r io.Reader, size int64, fileName, operator string) (Status, error) {
	u.mu.Lock()
	job, err := u.begin(SourceUpload, operator)
	if err != nil {
		u.mu.Unlock()
		return Status{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	u.cancel = cancel
	if size > 0 {
		u.st.Total = size
	}
	u.mu.Unlock()

	staged, info, err := u.receive(ctx, r, fileName)
	u.mu.Lock()
	defer u.mu.Unlock()
	if job != u.job {
		if staged != "" {
			_ = os.Remove(staged)
		}
		return Status{}, errors.New("上传已取消")
	}
	u.cancel = nil
	if err != nil {
		// 上传失败直接告诉上传的人，不留「失败」状态
		u.st = Status{}
		return Status{}, err
	}
	u.staged = staged
	u.st.State = StateReady
	u.st.Package = info
	u.st.TargetVersion = info.Version
	u.st.Downloaded = u.received.Load()
	return u.statusLocked(), nil
}

// receive 把上传的内容收进临时文件，解出主程序并校验，返回暂存的新程序路径。
func (u *Updater) receive(ctx context.Context, r io.Reader, fileName string) (string, *PackageInfo, error) {
	upload, err := u.tempFile(".upload")
	if err != nil {
		return "", nil, err
	}
	defer os.Remove(upload)
	f, err := os.OpenFile(upload, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", nil, fmt.Errorf("创建临时文件失败：%w", err)
	}
	n, err := io.Copy(&progressWriter{ctx: ctx, w: f, n: &u.received}, io.LimitReader(r, maxBinarySize+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", nil, fmt.Errorf("接收文件失败：%w", err)
	}
	if n == 0 {
		return "", nil, errors.New("上传的文件是空的")
	}
	if n > maxBinarySize {
		return "", nil, errors.New("文件太大")
	}
	staged, err := u.tempFile(stagedSuffix)
	if err != nil {
		return "", nil, err
	}
	info, err := extractAndInspect(upload, staged)
	if err != nil {
		_ = os.Remove(staged)
		return "", nil, err
	}
	info.FileName = filepath.Base(fileName)
	return staged, info, nil
}

// Apply 安装已上传、校验通过的程序并重启。
func (u *Updater) Apply(operator string) (Status, error) {
	u.mu.Lock()
	if u.st.State != StateReady {
		u.mu.Unlock()
		return Status{}, errors.New("没有等待安装的更新，请先上传程序文件")
	}
	u.st.State = StateInstalling
	if operator != "" {
		u.st.Operator = operator
	}
	job := u.job
	u.mu.Unlock()
	u.install(job)
	st := u.Status()
	if st.State == StateFailed {
		return st, errors.New(st.Error)
	}
	return st, nil
}

// install 用暂存的新程序替换当前程序，成功后稍等一下（让这次请求的响应发出去）再通知 main 停服退出、由系统服务重启。
// 状态为 Installing 时 Cancel 不会动 job 和暂存文件，这里读它们不会和别的更新冲突。
func (u *Updater) install(job int) {
	u.mu.Lock()
	staged := u.staged
	u.mu.Unlock()
	if err := replaceExecutable(u.exe, staged); err != nil {
		u.fail(job, err)
		return
	}
	u.mu.Lock()
	u.staged = ""
	u.st.State = StateRestarting
	u.mu.Unlock()
	go func() {
		time.Sleep(time.Second)
		requestRestart()
	}()
}

// Cancel 停止正在进行的下载 / 上传，或丢掉等待安装的文件；失败状态则清掉。
func (u *Updater) Cancel() (Status, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	switch u.st.State {
	case StateInstalling, StateRestarting:
		return u.statusLocked(), errors.New("已经在安装，不能取消")
	case StateDownloading:
		if u.dl != nil {
			_ = u.dl.Cancel()
		}
		if u.cancel != nil {
			u.cancel()
		}
	}
	u.job++
	u.dl = nil
	u.cancel = nil
	u.dropStagedLocked()
	u.st = Status{}
	return u.statusLocked(), nil
}
