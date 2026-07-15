package sub2api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// 共享 HTTP 客户端（包级单例）+ 泛型出站 API。
// 业务侧只传 URL / body 结构体 / 返回类型 T，禁止自行 NewRequest 或 new http.Client。

var (
	sharedHTTPClient     *http.Client
	sharedHTTPClientOnce sync.Once
)

const (
	sharedHTTPClientTimeout = 180 * time.Second
	sharedDialTimeout       = 10 * time.Second
	sharedTLSHandshake      = 10 * time.Second
	sharedIdleConnTimeout   = 90 * time.Second
	sharedMaxIdleConns      = 100
	sharedMaxIdlePerHost    = 20
	defaultMaxBody          = 32 << 20
)

// SharedHTTPClient 返回包内唯一的 *http.Client（懒初始化、并发安全）。
func SharedHTTPClient() *http.Client {
	sharedHTTPClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   sharedDialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          sharedMaxIdleConns,
			MaxIdleConnsPerHost:   sharedMaxIdlePerHost,
			IdleConnTimeout:       sharedIdleConnTimeout,
			TLSHandshakeTimeout:   sharedTLSHandshake,
			ExpectContinueTimeout: 1 * time.Second,
		}
		sharedHTTPClient = &http.Client{
			Timeout:   sharedHTTPClientTimeout,
			Transport: transport,
		}
	})
	return sharedHTTPClient
}

// DoBytes 统一出站：构造请求 → 共享 Client 发送 → 读 body。
// 这是包内唯一允许创建 *http.Request 的地方；不接调用方 context。
func DoBytes(method, rawURL string, headers map[string]string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			if len(v) > 0 {
				bodyReader = bytes.NewReader(v)
			}
		case string:
			if v != "" {
				bodyReader = bytes.NewReader([]byte(v))
			}
		default:
			payload, err := json.Marshal(v)
			if err != nil {
				return nil, 0, fmt.Errorf("序列化请求体失败: %w", err)
			}
			bodyReader = bytes.NewReader(payload)
		}
	}

	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := SharedHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, defaultMaxBody))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

// Do 发送请求并把响应 JSON 解码为 T。
// 用法：out, err := Do[MyResp](http.MethodGet, url, headers, nil)
func Do[T any](method, rawURL string, headers map[string]string, body any) (T, error) {
	var zero T
	data, status, err := DoBytes(method, rawURL, headers, body)
	if err != nil {
		return zero, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return zero, fmt.Errorf("Sub2API 返回 HTTP %d", status)
	}
	if len(data) == 0 || string(data) == "null" {
		return zero, nil
	}
	if err = json.Unmarshal(data, &zero); err != nil {
		return zero, fmt.Errorf("解析响应失败: %w", err)
	}
	return zero, nil
}

// Get 等价于 Do[T](GET, ...)。
func Get[T any](rawURL string, headers map[string]string) (T, error) {
	return Do[T](http.MethodGet, rawURL, headers, nil)
}

// Post 等价于 Do[T](POST, body)。
func Post[T any](rawURL string, headers map[string]string, body any) (T, error) {
	return Do[T](http.MethodPost, rawURL, headers, body)
}

// DoEnvelope 发送请求，解包 Sub2API {code,message,data}，把 data 解码为 T。
func DoEnvelope[T any](method, rawURL string, headers map[string]string, body any) (T, error) {
	var zero T
	data, status, err := DoBytes(method, rawURL, headers, body)
	if err != nil {
		return zero, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		// 仍尝试解 envelope message
		var env apiEnvelope
		if json.Unmarshal(data, &env) == nil && env.Message != "" {
			return zero, fmt.Errorf("%s", env.Message)
		}
		return zero, fmt.Errorf("Sub2API 返回 HTTP %d", status)
	}
	var env apiEnvelope
	if err = json.Unmarshal(data, &env); err != nil {
		return zero, fmt.Errorf("解析 Sub2API 响应失败: %w", err)
	}
	if env.Code != 0 && env.Code != http.StatusOK {
		msg := env.Message
		if msg == "" {
			msg = fmt.Sprintf("Sub2API 返回错误码 %d", env.Code)
		}
		return zero, fmt.Errorf("%s", msg)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return zero, nil
	}
	if err = json.Unmarshal(env.Data, &zero); err != nil {
		return zero, fmt.Errorf("解析 Sub2API data 失败: %w", err)
	}
	return zero, nil
}

// GetEnvelope / PostEnvelope 语义同 Get/Post，但走 {code,message,data} 信封。
func GetEnvelope[T any](rawURL string, headers map[string]string) (T, error) {
	return DoEnvelope[T](http.MethodGet, rawURL, headers, nil)
}

func PostEnvelope[T any](rawURL string, headers map[string]string, body any) (T, error) {
	return DoEnvelope[T](http.MethodPost, rawURL, headers, body)
}
