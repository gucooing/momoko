package agentinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"momoko/pkg/constant"
	"momoko/pkg/download"
	"momoko/pkg/utils"
)

const (
	userAgent = "momoko"
	githubAPI = "https://api.github.com"
	// 接口的应答最多读这么多（openai/codex 一个版本挂了近两百个文件，应答有几百 KB）。
	maxJSON = 8 << 20
)

// httpClient 查发布页用：默认传输（代理环境变量照常生效），只是连接和等响应头要快点失败。
// 安装包本身走统一下载管理（pkg/download）。
var httpClient = newClient()

type clientKey struct{}

// WithClient 之后用这个 ctx 查版本、下载都经 c（比如走运行时配的代理）。
func WithClient(ctx context.Context, c *http.Client) context.Context {
	return context.WithValue(ctx, clientKey{}, c)
}

// clientFrom ctx 指定的客户端；没指定时 override 为 false，用默认的。
func clientFrom(ctx context.Context) (c *http.Client, override bool) {
	if c, ok := ctx.Value(clientKey{}).(*http.Client); ok && c != nil {
		return c, true
	}
	return httpClient, false
}

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
	c, _ := clientFrom(ctx)
	resp, err := c.Do(req)
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

// fetchAsset 经统一下载管理把 a 下载到 dest（页面上能看到进度，也能暂停、继续、取消），下完核对 GitHub 给的校验和。
// progress 收到已下载的字节数。
func fetchAsset(ctx context.Context, title string, a asset, dest string, progress func(done int64)) error {
	// 几百兆的文件经过代理、镜像，坏掉并不稀奇：没有校验值就不装。
	want, ok := strings.CutPrefix(a.Digest, "sha256:")
	if !ok {
		return errors.New("发布页没有给出安装包的校验值，不能确认下载的文件是对的")
	}
	var client *http.Client
	if c, ok := clientFrom(ctx); ok {
		client = c
	}
	d, err := download.Default.Start(ctx, download.Request{
		Client:    client,
		URL:       a.URL,
		Dest:      dest,
		Name:      a.Name,
		Title:     title,
		Source:    "agent.runtime",
		Perm:      string(constant.AgentManage),
		Size:      a.Size,
		SHA256:    want,
		UserAgent: userAgent,
		Site:      "GitHub",
	})
	if err != nil {
		return err
	}
	if progress != nil {
		stop := make(chan struct{})
		defer close(stop)
		go func() {
			t := time.NewTicker(500 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					progress(d.Info().Downloaded)
				}
			}
		}()
	}
	return d.Wait(ctx)
}
