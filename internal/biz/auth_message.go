package biz

import (
	"context"
	"net"
	"strings"
	"time"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/response"
	"momoko/pkg/utils"
)

// 站内消息分类。
const (
	MessageCategorySecurity = "security"
	MessageCategorySystem   = "system"
	MessageCategoryNotice   = "notice"
)

// 站内消息种类（前端按 kind 本地化标题与内容）。
const (
	MessageKindLoginNewDevice  = "login_new_device"
	MessageKindLoginFailed     = "login_failed"
	MessageKindPasswordChanged = "password_changed"
	MessageKindPasswordReset   = "password_reset"
	MessageKindNotice          = "notice"
)

// 设备类型。
const (
	DeviceTypeDesktop = "desktop"
	DeviceTypeMobile  = "mobile"
	DeviceTypeTablet  = "tablet"
	DeviceTypeCLI     = "cli"
)

// IP 网络范围。
const (
	IPScopeLoopback = "loopback"
	IPScopeLAN      = "lan"
	IPScopePublic   = "public"
)

const (
	loginFailNotifyInterval = 10 * time.Minute
	messageContentMaxLen    = 2000
	messageTitleMaxLen      = 100
)

var (
	ErrMessageContentEmpty = response.BadRequest(500, "消息内容不能为空")
	ErrMessageTooLong      = response.BadRequest(500, "消息内容过长")
)

// UserMessage 写入站内消息时的输入。
type UserMessage struct {
	UserID   string
	Category string
	Kind     string
	Title    string
	Content  string
	Params   map[string]string
	SenderID string
}

// MessageRepo 站内消息存储（由 AuthRepo 一并实现）。
type MessageRepo interface {
	CreateMessages(ctx context.Context, msgs []*UserMessage) error
	ListMessages(ctx context.Context, userID string, unreadOnly bool, page, pageSize int64) ([]*gen.UserMessage, int64, error)
	// CountMessages 返回全部消息数与未读数。
	CountMessages(ctx context.Context, userID string) (all int64, unread int64, err error)
	// ReadMessages 标记已读；all 为 true 时忽略 ids。
	ReadMessages(ctx context.Context, userID string, ids []string, all bool) error
	// DeleteMessages 删除消息；all 为 true 时忽略 ids。
	DeleteMessages(ctx context.Context, userID string, ids []string, all bool) error
	// ActiveUserIDs 返回全部启用用户的 id（用于发布通知）。
	ActiveUserIDs(ctx context.Context) ([]string, error)
}

// ---- 登录追踪：让操作日志能把失败的登录记到对应用户名下 ----

type loginTraceKey struct{}

// LoginTrace 登录请求中识别出的用户（凭据错误时也会填写）。
type LoginTrace struct {
	UserID string
}

// WithLoginTrace 在请求上下文中挂一个可写的登录追踪。
func WithLoginTrace(ctx context.Context) (context.Context, *LoginTrace) {
	trace := &LoginTrace{}
	return context.WithValue(ctx, loginTraceKey{}, trace), trace
}

// MarkLoginTrace 记录登录请求对应的用户。
func MarkLoginTrace(ctx context.Context, userID string) {
	if trace, ok := ctx.Value(loginTraceKey{}).(*LoginTrace); ok {
		trace.UserID = userID
	}
}

// LoginTraceFromContext 返回登录追踪。
func LoginTraceFromContext(ctx context.Context) (*LoginTrace, bool) {
	trace, ok := ctx.Value(loginTraceKey{}).(*LoginTrace)
	return trace, ok
}

// ---- 设备 / IP 归类 ----

func userAgentFromContext(ctx context.Context) string {
	if req, ok := httptransport.RequestFromServerContext(ctx); ok {
		return req.UserAgent()
	}
	return ""
}

// DeviceType 根据 User-Agent 与上报的设备名粗略判断设备类型。
func DeviceType(userAgent, device string) string {
	ua := strings.ToLower(userAgent)
	dev := strings.ToLower(device)
	switch {
	case strings.Contains(ua, "ipad") || strings.Contains(dev, "ipad") || strings.Contains(ua, "tablet"):
		return DeviceTypeTablet
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android") ||
		strings.Contains(dev, "iphone") || strings.Contains(dev, "android") || strings.Contains(dev, "ios"):
		return DeviceTypeMobile
	case ua == "" || !strings.Contains(ua, "mozilla"):
		return DeviceTypeCLI
	default:
		return DeviceTypeDesktop
	}
}

// IPScope 判断 IP 属于本机、局域网还是公网。
func IPScope(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ""
	}
	switch {
	case parsed.IsLoopback():
		return IPScopeLoopback
	case parsed.IsPrivate() || parsed.IsLinkLocalUnicast():
		return IPScopeLAN
	default:
		return IPScopePublic
	}
}

// ---- 消息生产 ----

// notify 写一条站内消息；失败只记录，不影响主流程。
func (a *AuthUsecase) notify(ctx context.Context, userID, category, kind string, params map[string]string) {
	if userID == "" {
		return
	}
	_ = a.auth.CreateMessages(context.WithoutCancel(ctx), []*UserMessage{{
		UserID:   userID,
		Category: category,
		Kind:     kind,
		Params:   params,
	}})
}

func (a *AuthUsecase) notifyLoginFailed(ctx context.Context, userID, method string) {
	if last, ok := a.failNotified.Get(userID); ok && time.Since(last) < loginFailNotifyInterval {
		return
	}
	a.failNotified.Set(userID, time.Now())
	ip := utils.ClientIPFromContext(ctx)
	a.notify(ctx, userID, MessageCategorySecurity, MessageKindLoginFailed, map[string]string{
		"ip":       ip,
		"ip_scope": IPScope(ip),
		"method":   method,
	})
}

// NotifyPasswordChanged 修改密码成功后提醒本人。
func (a *AuthUsecase) NotifyPasswordChanged(ctx context.Context, userID string) {
	ip := utils.ClientIPFromContext(ctx)
	a.notify(ctx, userID, MessageCategorySecurity, MessageKindPasswordChanged, map[string]string{
		"ip":       ip,
		"ip_scope": IPScope(ip),
	})
}

// ---- 消息读写 ----

func (a *AuthUsecase) ListMessages(ctx context.Context, userID string, req *v1.ListMessagesRequest) (*v1.ListMessagesResponse, error) {
	page, pageSize := req.GetPage(), req.GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rows, total, err := a.auth.ListMessages(ctx, userID, req.GetUnreadOnly(), page, pageSize)
	if err != nil {
		return nil, ErrSystem(err)
	}
	all, unread, err := a.auth.CountMessages(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]*v1.UserMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, toUserMessage(row))
	}
	return &v1.ListMessagesResponse{
		Messages:    out,
		Total:       total,
		AllCount:    all,
		UnreadCount: unread,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (a *AuthUsecase) ReadMessages(ctx context.Context, userID string, ids []string, all bool) (int64, error) {
	if !all && len(ids) == 0 {
		return a.unreadCount(ctx, userID)
	}
	if err := a.auth.ReadMessages(ctx, userID, ids, all); err != nil {
		return 0, ErrSystem(err)
	}
	return a.unreadCount(ctx, userID)
}

func (a *AuthUsecase) DeleteMessages(ctx context.Context, userID string, ids []string, all bool) (int64, error) {
	if !all && len(ids) == 0 {
		return a.unreadCount(ctx, userID)
	}
	if err := a.auth.DeleteMessages(ctx, userID, ids, all); err != nil {
		return 0, ErrSystem(err)
	}
	return a.unreadCount(ctx, userID)
}

func (a *AuthUsecase) unreadCount(ctx context.Context, userID string) (int64, error) {
	_, unread, err := a.auth.CountMessages(ctx, userID)
	if err != nil {
		return 0, ErrSystem(err)
	}
	return unread, nil
}

// PublishMessage 把一条通知推送给全部启用的用户（含发布者本人）。
func (a *AuthUsecase) PublishMessage(ctx context.Context, senderID string, req *v1.PublishMessageRequest) (int64, error) {
	content := strings.TrimSpace(req.GetContent())
	title := strings.TrimSpace(req.GetTitle())
	if content == "" {
		return 0, ErrMessageContentEmpty
	}
	if len([]rune(content)) > messageContentMaxLen || len([]rune(title)) > messageTitleMaxLen {
		return 0, ErrMessageTooLong
	}
	sender, err := a.user.FindByID(ctx, senderID)
	if err != nil {
		return 0, ErrSystem(err)
	}
	senderName := sender.Name
	if senderName == "" {
		senderName = sender.Username
	}
	ids, err := a.auth.ActiveUserIDs(ctx)
	if err != nil {
		return 0, ErrSystem(err)
	}
	params := map[string]string{"sender_name": senderName, "sender_avatar": sender.Avatar}
	msgs := make([]*UserMessage, 0, len(ids))
	for _, id := range ids {
		msgs = append(msgs, &UserMessage{
			UserID:   id,
			Category: MessageCategoryNotice,
			Kind:     MessageKindNotice,
			Title:    title,
			Content:  content,
			Params:   params,
			SenderID: senderID,
		})
	}
	if err := a.auth.CreateMessages(ctx, msgs); err != nil {
		return 0, ErrSystem(err)
	}
	return int64(len(msgs)), nil
}

func toUserMessage(row *gen.UserMessage) *v1.UserMessage {
	params := row.Params
	if params == nil {
		params = map[string]string{}
	}
	return &v1.UserMessage{
		Id:           row.ID,
		Category:     row.Category,
		Kind:         row.Kind,
		Title:        row.Title,
		Content:      row.Content,
		Params:       params,
		Read:         row.ReadAt != nil,
		CreateTime:   timestamppb.New(row.CreateTime),
		SenderName:   params["sender_name"],
		SenderAvatar: params["sender_avatar"],
	}
}

// NewMessageID 生成消息 id。
func NewMessageID() string {
	return "msg_" + uuid.NewString()
}

// RequestUserAgent 返回当前 HTTP 请求的 User-Agent。
func RequestUserAgent(ctx context.Context) string { return userAgentFromContext(ctx) }

// Platform 从 User-Agent 粗略识别平台（GM 记录「来源」展示用）：Windows / macOS / Android / iOS / Linux。
func Platform(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "iOS"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return ""
	}
}
