package agentinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"momoko/pkg/utils"
)

const (
	userAgent = "momoko"
	githubAPI = "https://api.github.com"
	// 接口的应答最多读这么多（openai/codex 一个版本挂了近两百个文件，应答有几百 KB）。
	maxJSON = 8 << 20
	// 下载途中这么久收不到一个字节就算断了，不等到总的期限。
	stallTimeout = time.Minute
)

// httpClient 用默认传输（代理环境变量照常生效），只是连接和等响应头要快点失败：
// 下载本身可以很久，但连都连不上不该等半小时。
var httpClient = newClient()

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

// release 是 GitHub 发布页的一个版本。
type release struct {
	Tag    string  `json:"tag_name"`
	Assets []asset `json:"assets"`
}

// asset 是版本上挂的一个文件。
type asset struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"` // sha256:…
}

// version 由标签得出版本号：v0.108.1 → 0.108.1。
func (r release) version() string { return cleanVersion(r.Tag) }

func (r release) find(name string) (asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name && a.URL != "" {
			return a, true
		}
	}
	return asset{}, false
}

// latestRelease 取一个仓库最新的正式版。
func latestRelease(ctx context.Context, repo string) (release, error) {
	var r release
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPI+"/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return r, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	// 不带令牌每小时只能查 60 次。令牌只发给 GitHub 的接口，下载地址不带。
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return r, utils.NetError(ctx, "GitHub", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusOK:
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusTooManyRequests:
		return r, errors.New("GitHub 接口限流了，稍后再试")
	case resp.StatusCode == http.StatusUnauthorized && token != "":
		return r, errors.New("GitHub 不认环境变量 GITHUB_TOKEN 里的令牌")
	default:
		return r, fmt.Errorf("GitHub 返回 %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSON)).Decode(&r); err != nil {
		if ctx.Err() != nil {
			return r, utils.NetError(ctx, "GitHub", err)
		}
		return r, errors.New("GitHub 返回的内容看不懂")
	}
	if r.Tag == "" {
		return r, errors.New("GitHub 没有给出版本号")
	}
	return r, nil
}

var errStalled = errors.New("agentinstall: download stalled")

// download 把 a 下载到 dest，边下边算校验和。progress 收到已下载的字节数。
func download(ctx context.Context, a asset, dest string, progress func(done int64)) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return errors.New("下载地址不对")
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return utils.NetError(ctx, "GitHub", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败：GitHub 返回 %d", resp.StatusCode)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("下载失败：%s", reason(err))
	}
	defer file.Close()

	// 一段时间没有数据进来就掐断：连接半死不活的时候，Read 自己不会返回。
	watchdog := time.AfterFunc(stallTimeout, func() { cancel(errStalled) })
	defer watchdog.Stop()

	hash := sha256.New()
	buf := make([]byte, 256<<10)
	var done int64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			watchdog.Reset(stallTimeout)
			hash.Write(buf[:n])
			if _, err := file.Write(buf[:n]); err != nil {
				return fmt.Errorf("下载失败：%s", reason(err))
			}
			done += int64(n)
			progress(done)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			if errors.Is(context.Cause(ctx), errStalled) {
				return fmt.Errorf("下载中断了：%d 秒没有收到数据", int(stallTimeout/time.Second))
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("下载中断了，再试一次")
		}
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("下载失败：%s", reason(err))
	}
	if a.Size > 0 && done != a.Size {
		return errors.New("下载的安装包不完整，再试一次")
	}
	// 核对 GitHub 给的校验和：几百兆的文件经过代理、镜像，坏掉并不稀奇。
	want, ok := strings.CutPrefix(a.Digest, "sha256:")
	if !ok {
		return errors.New("发布页没有给出安装包的校验值，不能确认下载的文件是对的")
	}
	if !strings.EqualFold(want, hex.EncodeToString(hash.Sum(nil))) {
		return errors.New("下载的安装包校验没通过，再试一次")
	}
	return nil
}
