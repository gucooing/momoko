package biz

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/response"
)

// 游戏客户端错误日志：游戏客户端凭游戏服务端签发的上传票据，把日志传到通行证 server（POST /api/v1/game-client-logs），
// 每个游戏账号（服务器 + pid）只保留最新 5 份。管理端在「客户端日志」页查看全部服务器的日志（筛选、查看全文、下载与删除），
// GM 玩家详情的「客户端日志」是同一个列表按服务器与玩家筛选。
//
// 上传票据由游戏服务端用这台服务器的 MGP 密钥签发（学マス：SDK POST /passport/log-ticket）：
//
//	"v1." + b64url(payload) + "." + b64url(HMAC-SHA256(MGP secret, "mgp-client-log.v1." + b64url(payload)))
//
// payload 为 {"kid": MGP key_id, "pid": 游戏账号 id, "iat": 签发时间, "exp": 过期时间}（Unix 秒），b64url 为无填充的 base64url。

const (
	// ClientLogKeep 每个游戏账号保留的日志份数。
	ClientLogKeep = 5
	// ClientLogMaxBody 上传请求体（压缩时为压缩后）的大小上限。
	ClientLogMaxBody = 8 << 20
	// ClientLogMaxContent 解压后的请求体与日志原文的大小上限。
	ClientLogMaxContent = 32 << 20

	clientLogRateLimit  = 30
	clientLogRateWindow = time.Hour

	clientLogTicketVersion = "v1"
	clientLogTicketContext = "mgp-client-log.v1."
	clientLogTicketSkew    = 60 * time.Second
	clientLogTicketMaxLife = 7 * 24 * time.Hour

	clientLogMaxReason        = 32
	clientLogMaxSummary       = 512
	clientLogMaxNote          = 500
	clientLogMaxDevice        = 200
	clientLogMaxClientVersion = 64
)

// clientLogReasons 上传原因：exception 异常、crash 崩溃、fatal 致命错误由客户端自动上传，manual 是玩家手动上传。
var clientLogReasons = map[string]bool{
	"exception": true,
	"crash":     true,
	"fatal":     true,
	"manual":    true,
}

var (
	ErrClientLogTicket      = passportErr(401, "PASSPORT_CLIENT_LOG_TICKET_INVALID")
	ErrClientLogInvalid     = passportErr(400, "PASSPORT_CLIENT_LOG_INVALID")
	ErrClientLogTooLarge    = passportErr(413, "PASSPORT_CLIENT_LOG_TOO_LARGE")
	ErrClientLogEncoding    = passportErr(415, "PASSPORT_CLIENT_LOG_ENCODING_UNSUPPORTED")
	ErrClientLogRateLimited = passportErr(429, "PASSPORT_CLIENT_LOG_RATE_LIMITED")
	ErrGmClientLogNotFound  = response.BadRequest(404, "客户端日志不存在")
)

// ClientLogFilter 管理端客户端日志列表的筛选（空值表示不限）。
type ClientLogFilter struct {
	ServerID string
	PID      string
	Reason   string
	// Q 关键字：摘要、说明、设备（不区分大小写）
	Q            string
	Since, Until *time.Time
	Offset       int
	Limit        int
}

// GameClientLogRepo 游戏客户端日志的持久化（通行证写入，管理端读取与删除）。
type GameClientLogRepo interface {
	// Create 写入一份日志，并只保留这个游戏账号最新的 keep 份（更早的在同一事务里删除）。
	Create(ctx context.Context, l *gen.GameClientLog, keep int) (*gen.GameClientLog, error)
	// List 按筛选分页列出日志（不读日志原文），从新到旧；同时返回总数。
	List(ctx context.Context, f ClientLogFilter) ([]*gen.GameClientLog, int, error)
	Get(ctx context.Context, id string) (*gen.GameClientLog, error)
	Delete(ctx context.Context, id string) error
}

// ClientLogOwner 上传票据对应的游戏账号。
type ClientLogOwner struct {
	ServerID string
	PID      string
}

// ClientLogUpload 一份上传的日志（请求体解压、解码后的内容）。
type ClientLogUpload struct {
	Reason        string
	Summary       string
	Note          string
	Device        string
	ClientVersion string
	// OccurredAt 客户端记录日志的时间（Unix 毫秒；0 为上传时间）
	OccurredAt int64
	Content    string
}

type clientLogTicket struct {
	Kid string `json:"kid"`
	Pid string `json:"pid"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

// verifyClientLogTicket 校验上传票据：先查有效期（exp 允许 60 秒时钟偏差，签发到过期不超过 7 天），
// 再找到 MGP key_id 为 kid 的服务器，用它解密后的 MGP 密钥以常量时间比对签名。
func verifyClientLogTicket(ctx context.Context, servers *GameServerUsecase, ticket string, now time.Time) (ClientLogOwner, error) {
	parts := strings.Split(ticket, ".")
	if len(parts) != 3 || parts[0] != clientLogTicketVersion || parts[1] == "" || parts[2] == "" {
		return ClientLogOwner{}, ErrClientLogTicket
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ClientLogOwner{}, ErrClientLogTicket
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return ClientLogOwner{}, ErrClientLogTicket
	}
	var p clientLogTicket
	if err := json.Unmarshal(raw, &p); err != nil || p.Kid == "" || strings.TrimSpace(p.Pid) == "" {
		return ClientLogOwner{}, ErrClientLogTicket
	}
	if p.Exp <= now.Add(-clientLogTicketSkew).Unix() || p.Exp-p.Iat > int64(clientLogTicketMaxLife/time.Second) {
		return ClientLogOwner{}, ErrClientLogTicket
	}
	list, err := servers.ByKeyID(ctx, p.Kid)
	if err != nil {
		return ClientLogOwner{}, err
	}
	msg := []byte(clientLogTicketContext + parts[1])
	for _, s := range list {
		secret, err := gameBox().Decrypt(s.Secret)
		if err != nil {
			continue
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(msg)
		if hmac.Equal(mac.Sum(nil), sig) {
			return ClientLogOwner{ServerID: s.ID, PID: p.Pid}, nil
		}
	}
	return ClientLogOwner{}, ErrClientLogTicket
}

// clientLogText 去掉首尾空白并检查长度（按字符数）。
func clientLogText(v string, max int) (string, bool) {
	v = strings.TrimSpace(v)
	return v, utf8.RuneCountInString(v) <= max
}

// newClientLog 校验上传的内容并生成一条记录（日志原文 gzip 压缩）。
func newClientLog(owner ClientLogOwner, in ClientLogUpload, now time.Time) (*gen.GameClientLog, error) {
	reason, ok := clientLogText(in.Reason, clientLogMaxReason)
	if !ok || !clientLogReasons[reason] {
		return nil, ErrClientLogInvalid
	}
	summary, ok1 := clientLogText(in.Summary, clientLogMaxSummary)
	note, ok2 := clientLogText(in.Note, clientLogMaxNote)
	device, ok3 := clientLogText(in.Device, clientLogMaxDevice)
	version, ok4 := clientLogText(in.ClientVersion, clientLogMaxClientVersion)
	if !ok1 || !ok2 || !ok3 || !ok4 || in.OccurredAt < 0 {
		return nil, ErrClientLogInvalid
	}
	if strings.TrimSpace(in.Content) == "" {
		return nil, ErrClientLogInvalid
	}
	if len(in.Content) > ClientLogMaxContent {
		return nil, ErrClientLogTooLarge
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(in.Content)); err != nil {
		return nil, ErrSystem(err)
	}
	if err := zw.Close(); err != nil {
		return nil, ErrSystem(err)
	}
	occurred := now
	if in.OccurredAt > 0 {
		occurred = time.UnixMilli(in.OccurredAt)
	}
	return &gen.GameClientLog{
		ID:            uuid.NewString(),
		ServerID:      owner.ServerID,
		PlayerID:      owner.PID,
		Reason:        reason,
		Summary:       summary,
		Note:          note,
		Device:        device,
		ClientVersion: version,
		Size:          len(in.Content),
		Content:       buf.Bytes(),
		OccurredAt:    occurred,
		CreatedAt:     now,
	}, nil
}

// clientLogContent 解压日志原文。
func clientLogContent(l *gen.GameClientLog) (string, error) {
	zr, err := gzip.NewReader(bytes.NewReader(l.Content))
	if err != nil {
		return "", err
	}
	defer zr.Close()
	raw, err := io.ReadAll(io.LimitReader(zr, ClientLogMaxContent+1))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ---------------- 管理端（客户端日志页与 GM 玩家详情） ----------------

// clientLogLinks 日志所属游戏账号绑定的通行证：按服务器分组批量读取，键为服务器 id + pid。
func (uc *GmUsecase) clientLogLinks(ctx context.Context, logs []*gen.GameClientLog) map[string]*GmLink {
	pids := map[string][]string{}
	for _, l := range logs {
		pids[l.ServerID] = append(pids[l.ServerID], l.PlayerID)
	}
	out := map[string]*GmLink{}
	for serverID, list := range pids {
		links, err := uc.repo.LinksByPIDs(ctx, serverID, list)
		if err != nil {
			uc.log.Warnf("client log links of server %s: %v", serverID, err)
			continue
		}
		for _, link := range links {
			out[serverID+"\x00"+link.PID] = link
		}
	}
	return out
}

func (uc *GmUsecase) clientLogProto(ctx context.Context, l *gen.GameClientLog, servers map[string]*gen.GameServer, links map[string]*GmLink) *v1.GmClientLog {
	out := &v1.GmClientLog{
		Id:            l.ID,
		Reason:        l.Reason,
		Summary:       l.Summary,
		Note:          l.Note,
		Device:        l.Device,
		ClientVersion: l.ClientVersion,
		Size:          int32(l.Size),
		OccurredAt:    timestamppb.New(l.OccurredAt),
		CreatedAt:     timestamppb.New(l.CreatedAt),
		PlayerId:      l.PlayerID,
	}
	if s, ok := servers[l.ServerID]; ok {
		out.Server = uc.servers.Brief(ctx, s)
	} else {
		out.Server = &v1.GameServerBrief{Id: l.ServerID, Tags: []string{}, Online: -1, Registered: -1}
	}
	if link, ok := links[l.ServerID+"\x00"+l.PlayerID]; ok {
		out.Link = gmLinkProto(link)
	}
	return out
}

// ListClientLogs 客户端日志（全部服务器，从新到旧，分页，不含日志原文）。
func (uc *GmUsecase) ListClientLogs(ctx context.Context, req *v1.ListGmClientLogsRequest) (*v1.ListGmClientLogsResponse, error) {
	page, size := pageOf(req.Page, req.PageSize, 20)
	f := ClientLogFilter{
		ServerID: strings.TrimSpace(req.ServerId),
		PID:      strings.TrimSpace(req.Pid),
		Reason:   strings.TrimSpace(req.Reason),
		Q:        strings.TrimSpace(req.Q),
		Offset:   (page - 1) * size,
		Limit:    size,
	}
	if req.Since != nil {
		t := req.Since.AsTime()
		f.Since = &t
	}
	if req.Until != nil {
		t := req.Until.AsTime()
		f.Until = &t
	}
	list, total, err := uc.clientLogs.List(ctx, f)
	if err != nil {
		return nil, ErrSystem(err)
	}
	servers := passportServerMap(ctx, uc.servers)
	links := uc.clientLogLinks(ctx, list)
	out := &v1.ListGmClientLogsResponse{Items: make([]*v1.GmClientLog, 0, len(list)), Total: int32(total)}
	for _, l := range list {
		out.Items = append(out.Items, uc.clientLogProto(ctx, l, servers, links))
	}
	return out, nil
}

// clientLog 读取一份日志。
func (uc *GmUsecase) clientLog(ctx context.Context, id string) (*gen.GameClientLog, error) {
	l, err := uc.clientLogs.Get(ctx, id)
	if gen.IsNotFound(err) {
		return nil, ErrGmClientLogNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return l, nil
}

// GetClientLog 一份客户端日志（含解压后的日志原文、服务器与绑定的通行证）。
func (uc *GmUsecase) GetClientLog(ctx context.Context, id string) (*v1.GmClientLog, error) {
	l, err := uc.clientLog(ctx, id)
	if err != nil {
		return nil, err
	}
	content, err := clientLogContent(l)
	if err != nil {
		return nil, ErrSystem(err)
	}
	servers := map[string]*gen.GameServer{}
	if s, err := uc.servers.Get(ctx, l.ServerID); err == nil {
		servers[s.ID] = s
	}
	out := uc.clientLogProto(ctx, l, servers, uc.clientLogLinks(ctx, []*gen.GameClientLog{l}))
	out.Content = content
	return out, nil
}

// DeleteClientLog 删除一份客户端日志。
func (uc *GmUsecase) DeleteClientLog(ctx context.Context, id string) error {
	l, err := uc.clientLog(ctx, id)
	if err != nil {
		return err
	}
	if err := uc.clientLogs.Delete(ctx, l.ID); err != nil {
		if gen.IsNotFound(err) {
			return ErrGmClientLogNotFound
		}
		return ErrSystem(err)
	}
	return nil
}
