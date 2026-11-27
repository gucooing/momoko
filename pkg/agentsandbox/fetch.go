package agentsandbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// fetchArg 沙箱里替 Momoko 发一个 HTTP 请求：从标准输入读请求，结果写到标准输出。
const fetchArg = "__momoko_sandbox_fetch__"

// fetchMaxBody 请求和应答的内容最多这么大。
const fetchMaxBody = 16 << 20

type fetchRequest struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

type fetchResponse struct {
	Status int         `json:"status"`
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
	Error  string      `json:"error"`
}

// Transport Momoko 自己的 AI 请求（测试接口、拉模型、查余额、测试代理）也在沙箱里发：每个请求起一次沙箱，
// 由沙箱里的 Momoko 发出去，网络是 sing-box 接管的那张网卡。应答整段读回来（不支持流式）。
type Transport struct {
	Config Config
}

func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(io.LimitReader(req.Body, fetchMaxBody))
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
	}
	in, err := json.Marshal(fetchRequest{Method: req.Method, URL: req.URL.String(), Header: req.Header, Body: body})
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(req.Context(), selfExe, fetchArg)
	cmd.Stdin = bytes.NewReader(in)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	wait, err := Start(cmd, t.Config)
	if err != nil {
		return nil, err
	}
	if err := wait(); err != nil {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if msg := strings.TrimSpace(errOut.String()); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, err
	}
	var resp fetchResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		return nil, errors.New("沙箱里的请求没有给出结果")
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", resp.Status, http.StatusText(resp.Status)),
		StatusCode:    resp.Status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        resp.Header,
		Body:          io.NopCloser(bytes.NewReader(resp.Body)),
		ContentLength: int64(len(resp.Body)),
		Request:       req,
	}, nil
}

// runFetch 沙箱里的那一头：读请求、发出去、写结果。环境里没有代理变量，直接发，经的是 sing-box 接管的网卡。
func runFetch() int {
	write := func(r fetchResponse) int {
		if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
			return 1
		}
		return 0
	}
	var req fetchRequest
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 2*fetchMaxBody)).Decode(&req); err != nil {
		return write(fetchResponse{Error: "请求不对"})
	}
	hr, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return write(fetchResponse{Error: "请求地址不对"})
	}
	hr.Header = req.Header
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil
	resp, err := (&http.Client{Transport: tr}).Do(hr)
	if err != nil {
		return write(fetchResponse{Error: err.Error()})
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBody))
	if err != nil {
		return write(fetchResponse{Error: err.Error()})
	}
	return write(fetchResponse{Status: resp.StatusCode, Header: resp.Header, Body: body})
}
