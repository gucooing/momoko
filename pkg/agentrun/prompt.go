package agentrun

import (
	"encoding/base64"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"momoko/pkg/acp"
)

// Upload 用户这条消息带的一个附件。
type Upload struct {
	// Kind：image / file / paste（粘贴的一大段文字存成的文本文件，在会话的临时目录里）/
	// ref（引用的会话目录里的文件）/ snippet（打开的文件里选中的一段）。
	Kind string
	// Name 页面上显示的名字（ref 是相对路径，snippet 是「相对路径:起止行」）。
	Name string
	// Text snippet 选中的文字。
	Text string
	// Data 不为空时直接作为图片内容交给 Agent；否则把 Path 指向的本机文件交给它自己去读。
	Data []byte
	Path string
	Mime string
	// File 页面之后取回这个附件用的名字（记进事件）；ref、snippet、paste 为空。
	File string
}

// blocksOf 把一条消息整理成交给 Agent 的内容块。
func blocksOf(text string, uploads []Upload) []acp.ContentBlock {
	var blocks []acp.ContentBlock
	if text != "" {
		blocks = append(blocks, acp.ContentBlock{Type: "text", Text: text})
	}
	for _, u := range uploads {
		if u.Kind == "snippet" {
			fence := "```"
			for strings.Contains(u.Text, fence) {
				fence += "`"
			}
			body := strings.TrimRight(u.Text, "\r\n")
			blocks = append(blocks, acp.ContentBlock{Type: "text", Text: u.Name + "\n" + fence + "\n" + body + "\n" + fence})
			continue
		}
		if len(u.Data) > 0 {
			blocks = append(blocks, acp.ContentBlock{Type: "image", Data: base64.StdEncoding.EncodeToString(u.Data), MimeType: u.Mime})
			continue
		}
		blocks = append(blocks, acp.ContentBlock{Type: "resource_link", URI: fileURI(u.Path), Name: u.Name, MimeType: u.Mime})
	}
	return blocks
}

// fileURI 把本机路径写成 file:// 地址。
func fileURI(path string) string {
	slash := filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	return (&url.URL{Scheme: "file", Path: slash}).String()
}

// McpServer 交给 Agent 的一个 MCP 服务器。HTTP 为 false 时是 stdio。
type McpServer struct {
	Name    string
	HTTP    bool
	Command string
	Args    []string
	Env     map[string]string
	URL     string
	Headers map[string]string
}

func (m McpServer) acp() acp.McpServer {
	if m.HTTP {
		out := acp.McpServer{Type: "http", Name: m.Name, URL: m.URL}
		for _, k := range sortedKeys(m.Headers) {
			out.Headers = append(out.Headers, acp.HTTPHeader{Name: k, Value: m.Headers[k]})
		}
		return out
	}
	out := acp.McpServer{Name: m.Name, Command: m.Command, Args: m.Args}
	for _, k := range sortedKeys(m.Env) {
		out.Env = append(out.Env, acp.EnvVariable{Name: k, Value: m.Env[k]})
	}
	return out
}

// mcpServers 交给 Agent 的 MCP 服务器：Agent 不支持 HTTP 类型时，HTTP 的那些不给它。
func mcpServers(servers []McpServer, caps Caps) []acp.McpServer {
	out := make([]acp.McpServer, 0, len(servers))
	for _, m := range servers {
		if m.HTTP && !caps.McpHTTP {
			continue
		}
		out = append(out, m.acp())
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
