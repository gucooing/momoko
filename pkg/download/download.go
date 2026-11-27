// Package download 是服务器从外网下载文件的统一入口（系统更新、Agent 安装包、技能压缩包……）。
// 每个下载都登记在管理器里，页面上能看到进度和速度，也能暂停、继续、取消；
// 网络断了会自动重试，服务器支持 Range 时从断点接着下，否则从头再来。
// 发起下载的业务照常拿结果：Start 之后 Wait 到下载结束（暂停期间 Wait 一直等着）。
package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"momoko/pkg/utils"
)

// State 下载的状态。
type State string

const (
	StateDownloading State = "downloading" // 正在下载
	StatePaused      State = "paused"      // 已暂停（Wait 继续等着）
	StateRetrying    State = "retrying"    // 网络出错，等一会儿自动重试
	StateCompleted   State = "completed"   // 下完了，校验也过了
	StateFailed      State = "failed"      // 失败
	StateCanceled    State = "canceled"    // 已取消
)

// Terminal 是否已经结束。
func (s State) Terminal() bool {
	return s == StateCompleted || s == StateFailed || s == StateCanceled
}

const (
	// stallTimeout 这么久收不到一个字节就算断了，掐掉重连（连接半死不活时 Read 自己不会返回）。
	stallTimeout = time.Minute
	// maxRetries 网络出错后自动重试的次数（手动继续会重新计数）。
	maxRetries = 5
	// keepFinished 结束的下载在列表里留多久、最多留几条。
	keepFinished    = 24 * time.Hour
	maxFinishedKept = 50
)

var (
	// ErrCanceled 下载被取消（页面上点了取消，或发起方不再等了）。
	ErrCanceled = errors.New("下载已取消")
	// ErrNotFound 没有这个下载。
	ErrNotFound = errors.New("没有这个下载，可能已经被清掉了")

	errPause   = errors.New("download: paused")
	errCancel  = errors.New("download: canceled")
	errStalled = errors.New("download: stalled")
)

// Request 描述一个下载。
type Request struct {
	URL       string // 下载地址（http / https）
	Dest      string // 存到哪个文件（发起方负责用完删除）
	Name      string // 显示的文件名，空时取地址最后一段
	Title     string // 用途，如「更新 Momoko 到 v0.2.0」
	Source    string // 发起模块，如 system.update、agent.runtime、agent.skill
	Perm      string // 查看和操作这个下载要的权限，空时登录即可
	Size      int64  // 预计大小（字节），已知时下完核对
	SHA256    string // 期望的 SHA256（十六进制），给了就下完核对
	MaxSize   int64  // 大小上限（字节），0 为不限
	UserAgent string // 空时用 momoko
	Site      string // 报错时对站点的称呼（如 GitHub），空时用主机名
}

// Info 是一个下载的只读快照。
type Info struct {
	ID         string
	Name       string
	Title      string
	Source     string
	URL        string
	Perm       string
	Operator   string
	State      State
	Downloaded int64
	Total      int64 // 未知时为 0
	Speed      int64 // 字节/秒
	Resumable  bool  // 服务器支持断点续传（暂停后继续不用从头下）
	Retries    int   // 已自动重试的次数
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	FinishedAt time.Time
}

// Manager 管理所有下载。
type Manager struct {
	mu       sync.Mutex
	items    map[string]*Download
	client   *http.Client
	operator func(ctx context.Context) string
}

// Default 是进程内唯一的下载管理器，各业务包直接用它发起下载，页面通过它列出和操作。
var Default = NewManager()

// NewManager 创建下载管理器。
func NewManager() *Manager {
	return &Manager{items: map[string]*Download{}, client: newClient()}
}

// newClient 用默认传输（代理环境变量照常生效），只是连接和等响应头要快点失败：下载本身可以很久。
func newClient() *http.Client {
	var t *http.Transport
	if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		t = dt.Clone()
	} else {
		t = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	t.DialContext = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	t.TLSHandshakeTimeout = 30 * time.Second
	t.ResponseHeaderTimeout = time.Minute
	return &http.Client{Transport: t}
}

// SetOperatorResolver 设置从发起下载的 ctx 里取发起人名字的方法（由业务层注入）。
func (m *Manager) SetOperatorResolver(fn func(ctx context.Context) string) {
	m.mu.Lock()
	m.operator = fn
	m.mu.Unlock()
}

// Start 登记并开始一个下载。ctx 只用来取发起人，下载本身不跟着它结束（要停就 Cancel，或 Wait 时传入的 ctx 结束）。
func (m *Manager) Start(ctx context.Context, req Request) (*Download, error) {
	u, err := url.Parse(strings.TrimSpace(req.URL))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, errors.New("下载地址要以 http:// 或 https:// 开头")
	}
	if req.Dest == "" {
		return nil, errors.New("没有指定下载到哪里")
	}
	req.URL = u.String()
	if req.Name == "" {
		req.Name = path.Base(u.Path)
	}
	if req.Site == "" {
		req.Site = u.Hostname()
	}
	if req.UserAgent == "" {
		req.UserAgent = "momoko"
	}
	req.SHA256 = strings.ToLower(strings.TrimSpace(req.SHA256))
	now := time.Now()
	d := &Download{
		m:         m,
		id:        uuid.NewString(),
		req:       req,
		createdAt: now,
		updatedAt: now,
		state:     StateDownloading,
		total:     req.Size,
		signal:    make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	m.mu.Lock()
	if m.operator != nil && ctx != nil {
		d.operator = m.operator(ctx)
	}
	m.items[d.id] = d
	m.pruneLocked(now)
	m.mu.Unlock()
	go d.run()
	return d, nil
}

// Get 按 id 取下载。
func (m *Manager) Get(id string) (*Download, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.items[id]
	return d, ok
}

// List 列出所有下载（新的在前），顺带清掉过期的已结束记录。
func (m *Manager) List() []Info {
	m.mu.Lock()
	m.pruneLocked(time.Now())
	items := make([]*Download, 0, len(m.items))
	for _, d := range m.items {
		items = append(items, d)
	}
	m.mu.Unlock()
	out := make([]Info, 0, len(items))
	for _, d := range items {
		out = append(out, d.Info())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// pruneLocked 清掉结束超过 keepFinished 的记录，已结束的最多留 maxFinishedKept 条。调用方持 m.mu。
func (m *Manager) pruneLocked(now time.Time) {
	type finished struct {
		id string
		at time.Time
	}
	var done []finished
	for id, d := range m.items {
		d.mu.Lock()
		state, at := d.state, d.finishedAt
		d.mu.Unlock()
		if !state.Terminal() {
			continue
		}
		if now.Sub(at) > keepFinished {
			delete(m.items, id)
			continue
		}
		done = append(done, finished{id, at})
	}
	if len(done) <= maxFinishedKept {
		return
	}
	sort.Slice(done, func(i, j int) bool { return done[i].at.After(done[j].at) })
	for _, f := range done[maxFinishedKept:] {
		delete(m.items, f.id)
	}
}

// Pause 暂停下载。
func (m *Manager) Pause(id string) error {
	d, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	return d.Pause()
}

// Resume 继续已暂停的下载。
func (m *Manager) Resume(id string) error {
	d, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	return d.Resume()
}

// Cancel 取消下载（已下的部分会删掉，发起方的 Wait 返回 ErrCanceled）。
func (m *Manager) Cancel(id string) error {
	d, ok := m.Get(id)
	if !ok {
		return ErrNotFound
	}
	return d.Cancel()
}

// Remove 从列表里删掉一条已结束的记录。
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.items[id]
	if !ok {
		return ErrNotFound
	}
	if !d.State().Terminal() {
		return errors.New("下载还没结束，先取消再删")
	}
	delete(m.items, id)
	return nil
}

// RemoveFinished 清掉 keep 认可的已结束记录，返回清掉的条数。
func (m *Manager) RemoveFinished(keep func(Info) bool) int {
	m.mu.Lock()
	items := make([]*Download, 0, len(m.items))
	for _, d := range m.items {
		items = append(items, d)
	}
	m.mu.Unlock()
	n := 0
	for _, d := range items {
		info := d.Info()
		if !info.State.Terminal() || (keep != nil && !keep(info)) {
			continue
		}
		m.mu.Lock()
		delete(m.items, info.ID)
		m.mu.Unlock()
		n++
	}
	return n
}

// Download 是一个下载。
type Download struct {
	m         *Manager
	id        string
	req       Request
	operator  string
	createdAt time.Time

	mu         sync.Mutex
	state      State
	total      int64
	resumable  bool
	validator  string // ETag 或 Last-Modified，续传时用 If-Range 确认文件没变
	retries    int
	failedAt   int64 // 上次出错时已下的字节数：这次比它多就说明网络在好转，重试次数重新计
	err        error
	updatedAt  time.Time
	finishedAt time.Time
	stop       context.CancelCauseFunc // 停掉正在进行的这一次请求

	signal chan struct{} // 暂停 / 继续 / 取消时叫醒下载循环
	done   chan struct{} // 结束时关闭

	written atomic.Int64
	speed   atomic.Int64
}

// ID 下载的标识。
func (d *Download) ID() string { return d.id }

// State 当前状态。
func (d *Download) State() State {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.state
}

// Info 当前快照。
func (d *Download) Info() Info {
	d.mu.Lock()
	defer d.mu.Unlock()
	info := Info{
		ID:         d.id,
		Name:       d.req.Name,
		Title:      d.req.Title,
		Source:     d.req.Source,
		URL:        d.req.URL,
		Perm:       d.req.Perm,
		Operator:   d.operator,
		State:      d.state,
		Downloaded: d.written.Load(),
		Total:      d.total,
		Resumable:  d.resumable,
		Retries:    d.retries,
		CreatedAt:  d.createdAt,
		UpdatedAt:  d.updatedAt,
		FinishedAt: d.finishedAt,
	}
	if d.state == StateDownloading {
		info.Speed = d.speed.Load()
	}
	if d.err != nil {
		info.Error = d.err.Error()
	}
	return info
}

// Done 下载结束时关闭。
func (d *Download) Done() <-chan struct{} { return d.done }

// Wait 等下载结束：下完返回 nil，取消返回 ErrCanceled，失败返回原因。ctx 结束时取消下载并返回 ctx.Err()。
func (d *Download) Wait(ctx context.Context) error {
	select {
	case <-d.done:
	case <-ctx.Done():
		_ = d.Cancel()
		<-d.done
		return ctx.Err()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	switch d.state {
	case StateCompleted:
		return nil
	case StateCanceled:
		return ErrCanceled
	default:
		return d.err
	}
}

func (d *Download) wake() {
	select {
	case d.signal <- struct{}{}:
	default:
	}
}

// Pause 暂停：断开当前连接，已下的部分留着，继续时从断点接着下（服务器不支持续传时从头下）。
func (d *Download) Pause() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch d.state {
	case StatePaused:
		return nil
	case StateDownloading, StateRetrying:
		d.state = StatePaused
		d.updatedAt = time.Now()
		if d.stop != nil {
			d.stop(errPause)
		}
		d.wake()
		return nil
	default:
		return errors.New("下载已经结束了")
	}
}

// Resume 继续已暂停的下载（自动重试次数重新计）。
func (d *Download) Resume() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch d.state {
	case StatePaused:
		d.state = StateDownloading
		d.retries = 0
		d.err = nil
		d.updatedAt = time.Now()
		d.wake()
		return nil
	case StateDownloading, StateRetrying:
		return nil
	default:
		return errors.New("下载已经结束了")
	}
}

// Cancel 取消下载。
func (d *Download) Cancel() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state.Terminal() {
		if d.state == StateCanceled {
			return nil
		}
		return errors.New("下载已经结束了")
	}
	d.state = StateCanceled
	d.updatedAt = time.Now()
	if d.stop != nil {
		d.stop(errCancel)
	}
	d.wake()
	return nil
}

// permanent 不值得重试的错误（地址不对、文件太大、磁盘写不进去……）。
type permanent struct{ error }

func (p permanent) Unwrap() error { return p.error }

func (d *Download) run() {
	for {
		d.mu.Lock()
		for d.state == StatePaused {
			d.mu.Unlock()
			<-d.signal
			d.mu.Lock()
		}
		if d.state == StateCanceled {
			d.mu.Unlock()
			d.finish(StateCanceled, ErrCanceled)
			return
		}
		ctx, stop := context.WithCancelCause(context.Background())
		d.stop = stop
		d.state = StateDownloading
		d.updatedAt = time.Now()
		d.mu.Unlock()

		err := d.attempt(ctx)
		cause := context.Cause(ctx)
		stop(nil)

		d.mu.Lock()
		d.stop = nil
		state := d.state
		d.mu.Unlock()
		switch {
		case state == StateCanceled:
			d.finish(StateCanceled, ErrCanceled)
			return
		case state == StatePaused || errors.Is(cause, errPause):
			continue
		case err == nil:
			if verr := d.verify(); verr != nil {
				d.finish(StateFailed, verr)
			} else {
				d.finish(StateCompleted, nil)
			}
			return
		}
		var p permanent
		if errors.As(err, &p) {
			d.finish(StateFailed, p.error)
			return
		}
		if !d.backoff(err) {
			continue
		}
		d.finish(StateFailed, err)
		return
	}
}

// backoff 网络出错后等一会儿再试；重试次数用完返回 true。等待期间暂停或取消会立刻叫醒。
func (d *Download) backoff(err error) (giveUp bool) {
	d.mu.Lock()
	if d.state != StateDownloading {
		d.mu.Unlock()
		return false
	}
	if n := d.written.Load(); n > d.failedAt {
		d.retries = 0
		d.failedAt = n
	}
	d.retries++
	if d.retries > maxRetries {
		d.mu.Unlock()
		return true
	}
	d.state = StateRetrying
	d.err = err
	d.updatedAt = time.Now()
	wait := time.Duration(1<<min(d.retries, 5)) * time.Second // 2s, 4s, 8s, 16s, 32s
	d.mu.Unlock()

	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-t.C:
	case <-d.signal:
	}
	return false
}

// attempt 发一次请求，能续传就从已下的位置接着下。
func (d *Download) attempt(ctx context.Context) error {
	d.mu.Lock()
	offset := int64(0)
	if d.resumable {
		if st, err := os.Stat(d.req.Dest); err == nil {
			offset = st.Size()
		}
	}
	validator := d.validator
	d.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.req.URL, nil)
	if err != nil {
		return permanent{errors.New("下载地址不对")}
	}
	req.Header.Set("User-Agent", d.req.UserAgent)
	if offset > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(offset, 10)+"-")
		if validator != "" {
			req.Header.Set("If-Range", validator)
		}
	}
	resp, err := d.m.client.Do(req)
	if err != nil {
		return d.netError(ctx, err)
	}
	defer resp.Body.Close()

	total := int64(0)
	switch resp.StatusCode {
	case http.StatusOK:
		offset = 0 // 服务器给的是整个文件（不支持续传，或文件变了）
		if resp.ContentLength > 0 {
			total = resp.ContentLength
		}
	case http.StatusPartialContent:
		start, size, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if !ok || start != offset {
			d.setResumable(false, "")
			return errors.New("服务器续传的位置不对，从头重新下载")
		}
		total = size
	case http.StatusRequestedRangeNotSatisfiable:
		d.setResumable(false, "")
		return errors.New("服务器不认续传的位置，从头重新下载")
	default:
		msg := fmt.Errorf("下载失败：%s 返回 %s", d.req.Site, resp.Status)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != http.StatusRequestTimeout {
			return permanent{msg}
		}
		return msg
	}
	if total <= 0 && d.req.Size > 0 {
		total = d.req.Size
	}
	if d.req.MaxSize > 0 && total > d.req.MaxSize {
		return permanent{errors.New("文件太大")}
	}
	d.setResumable(resp.StatusCode == http.StatusPartialContent || resp.Header.Get("Accept-Ranges") == "bytes", rangeValidator(resp.Header))
	d.mu.Lock()
	d.total = total
	d.mu.Unlock()

	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if offset > 0 {
		flag = os.O_WRONLY | os.O_APPEND
	}
	file, err := os.OpenFile(d.req.Dest, flag, 0o644)
	if err != nil {
		return permanent{fmt.Errorf("下载失败：%s", fileReason(err))}
	}
	defer file.Close()
	d.written.Store(offset)
	d.speed.Store(0) // 暂停 / 重连前的速度不算数，等这次的第一个采样

	stopWith := d.stopFunc()
	watchdog := time.AfterFunc(stallTimeout, func() { stopWith(errStalled) })
	defer watchdog.Stop()

	buf := make([]byte, 256<<10)
	written := offset
	lastAt, lastN := time.Now(), written
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			watchdog.Reset(stallTimeout)
			if _, err := file.Write(buf[:n]); err != nil {
				return permanent{fmt.Errorf("下载失败：%s", fileReason(err))}
			}
			written += int64(n)
			d.written.Store(written)
			if d.req.MaxSize > 0 && written > d.req.MaxSize {
				return permanent{errors.New("文件太大")}
			}
			if el := time.Since(lastAt); el >= time.Second {
				d.speed.Store(int64(float64(written-lastN) / el.Seconds()))
				lastAt, lastN = time.Now(), written
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			if errors.Is(context.Cause(ctx), errStalled) {
				return fmt.Errorf("下载中断了：%d 秒没有收到数据", int(stallTimeout/time.Second))
			}
			return d.netError(ctx, rerr)
		}
	}
	if err := file.Close(); err != nil {
		return permanent{fmt.Errorf("下载失败：%s", fileReason(err))}
	}
	if total > 0 && written != total {
		return errors.New("下载的文件不完整")
	}
	return nil
}

// stopFunc 取当前这次请求的停止函数（看门狗用）。
func (d *Download) stopFunc() context.CancelCauseFunc {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stop == nil {
		return func(error) {}
	}
	return d.stop
}

func (d *Download) setResumable(ok bool, validator string) {
	d.mu.Lock()
	d.resumable = ok
	d.validator = validator
	d.mu.Unlock()
}

func (d *Download) netError(ctx context.Context, err error) error {
	if errors.Is(context.Cause(ctx), errStalled) {
		return fmt.Errorf("下载中断了：%d 秒没有收到数据", int(stallTimeout/time.Second))
	}
	return utils.NetError(context.Background(), d.req.Site, err)
}

// verify 下完核对大小和 SHA256。
func (d *Download) verify() error {
	f, err := os.Open(d.req.Dest)
	if err != nil {
		return fmt.Errorf("下载失败：%s", fileReason(err))
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return fmt.Errorf("下载失败：%s", fileReason(err))
	}
	if d.req.Size > 0 && n != d.req.Size {
		return errors.New("下载的文件大小不对，再试一次")
	}
	if d.req.SHA256 != "" && d.req.SHA256 != hex.EncodeToString(h.Sum(nil)) {
		return errors.New("下载的文件校验没通过（SHA256 不一致），可能下载出错或被篡改，再试一次")
	}
	return nil
}

func (d *Download) finish(state State, err error) {
	if state != StateCompleted {
		_ = os.Remove(d.req.Dest)
	}
	d.mu.Lock()
	d.state = state
	if state == StateCompleted {
		d.err = nil
	} else {
		d.err = err
	}
	now := time.Now()
	d.updatedAt, d.finishedAt = now, now
	d.speed.Store(0)
	d.mu.Unlock()
	close(d.done)
}

// parseContentRange 解析「bytes 起-止/总数」，总数未知（*）时为 0。
func parseContentRange(v string) (start, total int64, ok bool) {
	v, ok = strings.CutPrefix(strings.TrimSpace(v), "bytes ")
	if !ok {
		return 0, 0, false
	}
	rng, size, ok := strings.Cut(v, "/")
	if !ok {
		return 0, 0, false
	}
	first, _, ok := strings.Cut(rng, "-")
	if !ok {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if size != "*" {
		if total, err = strconv.ParseInt(size, 10, 64); err != nil {
			return 0, 0, false
		}
	}
	return start, total, true
}

// rangeValidator If-Range 只认强 ETag，没有就用 Last-Modified。
func rangeValidator(h http.Header) string {
	if etag := h.Get("ETag"); etag != "" && !strings.HasPrefix(etag, "W/") {
		return etag
	}
	return h.Get("Last-Modified")
}

func fileReason(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) {
		err = pe.Err
	}
	return err.Error()
}
