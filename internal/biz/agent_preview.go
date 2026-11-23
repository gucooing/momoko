package biz

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"momoko/pkg/agentfs"
	"momoko/pkg/auth"
	"momoko/pkg/response"
)

// AgentPreviewPrefix 文件页 HTML 预览用的静态站点：把会话目录（新会话页是工作区目录）原样当成网站给 iframe 加载，
// 页面里的相对链接、样式表、脚本、图片、fetch 都按目录里的真实路径取。
//
//	/api/v1/public/agent/preview/{节点}/{签名}/{相对路径}
//
// iframe 带不了登录头，所以走 /api/v1/public/ 免登录，靠签名鉴权：签名在登录后按会话或工作区签发，里面带着过期时间。
// 节点放在路径里（相对链接会丢掉查询参数），远程节点上的会话由 NodeProxy 按它转发过去。
const AgentPreviewPrefix = "/api/v1/public/agent/preview/"

// agentPreviewTTL 一个签名能用多久；页面开着时提前换新的。
const agentPreviewTTL = 12 * time.Hour

// previewCSP 返回的页面一律放进沙箱：单独在新窗口打开时也是一个隔离的来源，碰不到 Momoko 的登录信息。
const previewCSP = "sandbox allow-scripts allow-popups allow-forms allow-modals allow-downloads"

// previewShimPath 预览站点里保留的一个地址：存储替身脚本（编辑中的页面放在 srcdoc 里，用 <script src> 引它）。
const previewShimPath = "__momoko-preview/storage.js"

// previewStorageShim 每个预览页面最先跑的一段脚本：
//   - 告诉外面的 Momoko 现在到了哪个页面（点链接跳走了，文件标签跟着换，还能后退）；
//   - 沙箱里的页面用不了 localStorage / sessionStorage（一碰就报错），原型页常拿它记登录状态之类，
//     用不了时换成存在 window.name 里的替身：同一个窗口（iframe）里跳到别的页面也还在。
const previewStorageShim = `(function(){function report(){try{if(window.parent!==window)window.parent.postMessage({momokoPreview:1,href:location.href},"*")}catch(e){}}
report();addEventListener("hashchange",report);addEventListener("popstate",report)})();
(function(){try{window.localStorage.getItem("x");return}catch(e){}
var K="__momokoPreview";function read(){try{var d=JSON.parse(window.name||"{}");return d&&typeof d==="object"?d:{}}catch(e){return {}}}
var all=read()[K]||{};function save(){var d=read();d[K]=all;try{window.name=JSON.stringify(d)}catch(e){}}
function store(ns){var m=all[ns]=all[ns]||{};var has=function(k){return Object.prototype.hasOwnProperty.call(m,k)};
var s={getItem:function(k){k=String(k);return has(k)?m[k]:null},setItem:function(k,v){m[String(k)]=String(v);save()},
removeItem:function(k){delete m[String(k)];save()},clear:function(){for(var k in m)delete m[k];save()},key:function(i){return Object.keys(m)[i]||null}};
Object.defineProperty(s,"length",{get:function(){return Object.keys(m).length}});return s}
try{Object.defineProperty(window,"localStorage",{value:store("l"),configurable:true})}catch(e){}
try{Object.defineProperty(window,"sessionStorage",{value:store("s"),configurable:true})}catch(e){}})();`

// previewHTMLMax 超过这么大的 HTML 不注入替身，原样给
const previewHTMLMax = 8 << 20

// injectShim 把存储替身插到 HTML 最前面（<head> 之后，没有 head 就放在开头），让它比页面自己的脚本先跑。
func injectShim(html []byte) []byte {
	tag := []byte("<script>" + previewStorageShim + "</script>")
	lower := bytes.ToLower(html)
	at := 0
	for _, open := range [][]byte{[]byte("<head"), []byte("<html")} {
		if i := bytes.Index(lower, open); i >= 0 {
			if j := bytes.IndexByte(lower[i:], '>'); j >= 0 {
				at = i + j + 1
				break
			}
		}
	}
	out := make([]byte, 0, len(html)+len(tag))
	out = append(out, html[:at]...)
	out = append(out, tag...)
	return append(out, html[at:]...)
}

// previewKey 签名用的密钥：从主密钥派生，和登录 token 的签名分开。
func previewKey() []byte {
	mac := hmac.New(sha256.New, []byte(auth.AuthSecretKey))
	mac.Write([]byte("momoko/agent-preview/v1"))
	return mac.Sum(nil)
}

func previewSign(body string) string {
	mac := hmac.New(sha256.New, previewKey())
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:20])
}

// PreviewToken 给当前用户能打开的会话（或工作区）签发预览签名，返回签名和过期时间（Unix 秒）。
func (uc *AgentUsecase) PreviewToken(ctx context.Context, sessionID, workspaceID string) (string, int64, error) {
	if _, err := uc.fileRoot(ctx, sessionID, workspaceID); err != nil {
		return "", 0, err
	}
	kind, id := "s", sessionID
	if sessionID == "" {
		kind, id = "w", workspaceID
	}
	exp := time.Now().Add(agentPreviewTTL).Unix()
	body := base64.RawURLEncoding.EncodeToString([]byte(kind + ":" + id + ":" + strconv.FormatInt(exp, 10)))
	return body + "." + previewSign(body), exp, nil
}

// previewRoot 验签并找到签名对应的目录；签名不对、过期、会话或工作区没了都返回空。
func (uc *AgentUsecase) previewRoot(ctx context.Context, token string) string {
	body, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(previewSign(body))) {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return ""
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 {
		return ""
	}
	if exp, err := strconv.ParseInt(parts[2], 10, 64); err != nil || time.Now().Unix() > exp {
		return ""
	}
	switch parts[0] {
	case "s":
		if s, err := uc.repo.GetAgentSession(ctx, parts[1]); err == nil {
			return s.Cwd
		}
	case "w":
		if w, err := uc.repo.GetAgentWorkspace(ctx, parts[1]); err == nil {
			return w.Path
		}
	}
	return ""
}

// ServePreview 预览站点的一个文件。路径是文件夹时找它下面的 index.html。
func (uc *AgentUsecase) ServePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// {节点}/{签名}/{相对路径}：到了这里节点已经是本机
	rest := strings.TrimPrefix(r.URL.Path, AgentPreviewPrefix)
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	root := uc.previewRoot(r.Context(), parts[1])
	if root == "" {
		response.WriteError(w, r, response.BadRequest("预览链接已失效，重新打开文件试试"))
		return
	}
	rel := ""
	if len(parts) == 3 {
		rel = parts[2]
	}
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	// 文件随时在改（编辑器保存、Agent 改），不缓存
	h.Set("Cache-Control", "no-store")
	if rel == previewShimPath {
		h.Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write([]byte(previewStorageShim))
		return
	}
	full, err := agentfs.SafeJoin(root, rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(full)
	if err == nil && info.IsDir() {
		full = filepath.Join(full, "index.html")
		info, err = os.Stat(full)
	}
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	h.Set("Content-Security-Policy", previewCSP)
	ext := strings.ToLower(filepath.Ext(full))
	if (ext == ".html" || ext == ".htm") && info.Size() <= previewHTMLMax {
		raw, err := os.ReadFile(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		h.Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(injectShim(raw)))
		return
	}
	f, err := os.Open(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
