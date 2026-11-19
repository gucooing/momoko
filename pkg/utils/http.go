package utils

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"unicode/utf8"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ip := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		if ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func ClientIPFromContext(ctx context.Context) string {
	httpReq, ok := httptransport.RequestFromServerContext(ctx)
	if !ok {
		return ""
	}
	return ClientIP(httpReq)
}

// NetError 把网络错误换成给人看、不带地址的说法：原始错误里有完整的请求地址，地址里可能带密钥或令牌。
// who 是对方的名字或主机名。调用方自己取消的原样交回 context.Canceled。
func NetError(ctx context.Context, who string, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	// 英文名和中文之间留空格（连不上 GitHub），中文名不留（连不上飞书）。
	name := who
	if who != "" && who[0] < utf8.RuneSelf {
		name = " " + who + " "
	}
	var ne net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()):
		return errors.New(strings.TrimSpace("连接" + name + "超时"))
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New(strings.TrimSpace(name + "中途断开了连接"))
	}
	return errors.New(strings.TrimSpace("连不上" + name))
}
