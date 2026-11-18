package biz

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/cache"
	"momoko/pkg/common"
	"momoko/pkg/response"
)

// 通行证的固定规则。
const (
	// PassportUnbindCooldown 解绑后不能在同一服务器绑定别的游戏账号的时长。
	PassportUnbindCooldown = 7 * 24 * time.Hour
	// PassportDeletionGrace 申请注销后可以撤销的时长。
	PassportDeletionGrace = 7 * 24 * time.Hour
	// PassportAccessTTL 访问令牌有效期。
	PassportAccessTTL = time.Hour
	// passportShortSession 不保持登录时的会话有效期。
	passportShortSession = 24 * time.Hour
	// passportMfaTTL 两步验证的登录挑战有效期。
	passportMfaTTL = 5 * time.Minute
	// passportLockThreshold / passportLockWindow 登录失败锁定：窗口内失败这么多次后锁定到窗口结束。
	passportLockThreshold = 5
	passportLockWindow    = 15 * time.Minute
	// passportBindAttempts 同一服务器上游戏账号密码校验失败的次数上限（窗口同上）。
	passportBindAttempts = 5
	// passportRecoveryCodes 恢复码个数。
	passportRecoveryCodes = 10
)

// 注册方式。
const (
	PassportRegisterOpen       = "open"
	PassportRegisterInviteOnly = "invite_only"
	PassportRegisterClosed     = "closed"
)

// 登录方式（passport_sessions.login_method、passport_login_logs.method）。
const (
	PassportLoginPassword  = "password"
	PassportLoginEmailCode = "email_code"
	PassportLoginRegister  = "register"
	// PassportLoginClient 客户端（启动器、APK）经浏览器授权取得的会话
	PassportLoginClient = "client"
)

// 消息种类（前端本地化）。
const (
	PassportMsgLoginNewDevice    = "login_new_device"
	PassportMsgPasswordChanged   = "password_changed"
	PassportMsgPasswordReset     = "password_reset"
	PassportMsgEmailChanged      = "email_changed"
	PassportMsgTotpEnabled       = "totp_enabled"
	PassportMsgTotpDisabled      = "totp_disabled"
	PassportMsgLinkBoundByAdmin  = "link_bound_by_admin"
	PassportMsgLinkUnboundAdmin  = "link_unbound_by_admin"
	PassportMsgRequestApproved   = "request_approved"
	PassportMsgRequestRejected   = "request_rejected"
	PassportMsgDeletionScheduled = "deletion_scheduled"
	PassportMsgDeletionCancelled = "deletion_cancelled"
	PassportMsgAccountEnabled    = "account_enabled"
)

// passportErr 通行证接口的错误：message 是错误代码，前端本地化（后端不返回给玩家看的成句文案）。
func passportErr(code string) error { return response.BadRequest(code) }

// passportUnauthorized 通行证凭证失效（401）。
func passportUnauthorized(code string) error { return response.Unauthorized(code) }

var (
	ErrPassportUnauthorized        = passportUnauthorized("PASSPORT_UNAUTHORIZED")
	ErrPassportInvalidArgument     = passportErr("PASSPORT_INVALID_ARGUMENT")
	ErrPassportRegisterClosed      = passportErr("PASSPORT_REGISTER_CLOSED")
	ErrPassportInviteRequired      = passportErr("PASSPORT_INVITE_REQUIRED")
	ErrPassportInviteInvalid       = passportErr("PASSPORT_INVITE_INVALID")
	ErrPassportUsernameInvalid     = passportErr("PASSPORT_USERNAME_INVALID")
	ErrPassportUsernameTaken       = passportErr("PASSPORT_USERNAME_TAKEN")
	ErrPassportEmailRequired       = passportErr("PASSPORT_EMAIL_REQUIRED")
	ErrPassportEmailInvalid        = passportErr("PASSPORT_EMAIL_INVALID")
	ErrPassportEmailTaken          = passportErr("PASSPORT_EMAIL_TAKEN")
	ErrPassportEmailUnavailable    = passportErr("PASSPORT_EMAIL_UNAVAILABLE")
	ErrPassportEmailLoginOff       = passportErr("PASSPORT_EMAIL_LOGIN_DISABLED")
	ErrPassportPasswordEmpty       = passportErr("PASSPORT_PASSWORD_EMPTY")
	ErrPassportPasswordInvalid     = passportErr("PASSPORT_PASSWORD_INVALID")
	ErrPassportCodeInvalid         = passportErr("PASSPORT_CODE_INVALID")
	ErrPassportCodeTooFrequent     = passportErr("PASSPORT_CODE_TOO_FREQUENT")
	ErrPassportCredentials         = passportErr("PASSPORT_CREDENTIALS_INVALID")
	ErrPassportLocked              = passportErr("PASSPORT_LOCKED")
	ErrPassportDisabled            = passportErr("PASSPORT_ACCOUNT_DISABLED")
	ErrPassportMfaInvalid          = passportErr("PASSPORT_MFA_INVALID")
	ErrPassportMfaExpired          = passportErr("PASSPORT_MFA_EXPIRED")
	ErrPassportTotpNotSetup        = passportErr("PASSPORT_TOTP_NOT_SETUP")
	ErrPassportTotpEnabled         = passportErr("PASSPORT_TOTP_ALREADY_ENABLED")
	ErrPassportTotpDisabled        = passportErr("PASSPORT_TOTP_NOT_ENABLED")
	ErrPassportTotpRequired        = passportErr("PASSPORT_TOTP_REQUIRED")
	ErrPassportUserNotFound        = passportErr("PASSPORT_USER_NOT_FOUND")
	ErrPassportSessionNotFound     = passportErr("PASSPORT_SESSION_NOT_FOUND")
	ErrPassportLinkNotFound        = passportErr("PASSPORT_LINK_NOT_FOUND")
	ErrPassportAppNotFound         = passportErr("PASSPORT_APP_NOT_FOUND")
	ErrPassportServerNotBindable   = passportErr("PASSPORT_SERVER_NOT_BINDABLE")
	ErrPassportBindUnsupported     = passportErr("PASSPORT_BIND_UNSUPPORTED")
	ErrPassportAlreadyLinked       = passportErr("PASSPORT_ALREADY_LINKED")
	ErrPassportAccountTaken        = passportErr("PASSPORT_GAME_ACCOUNT_TAKEN")
	ErrPassportBindCooldown        = passportErr("PASSPORT_BIND_COOLDOWN")
	ErrPassportGameCredentials     = passportErr("PASSPORT_GAME_CREDENTIALS_INVALID")
	ErrPassportBindTooMany         = passportErr("PASSPORT_BIND_TOO_MANY_ATTEMPTS")
	ErrPassportGameAccountRequired = passportErr("PASSPORT_GAME_ACCOUNT_REQUIRED")
	ErrPassportSelfServiceClosed   = passportErr("PASSPORT_SELF_SERVICE_CLOSED")
	ErrPassportSelfServiceLimit    = passportErr("PASSPORT_SELF_SERVICE_LIMIT")
	ErrPassportSelfServicePending  = passportErr("PASSPORT_SELF_SERVICE_PENDING")
	ErrPassportGameUnavailable     = passportErr("PASSPORT_GAME_UNAVAILABLE")
	ErrPassportAvatarAsset         = passportErr("PASSPORT_AVATAR_ASSET_INVALID")
	ErrPassportNameTooLong         = passportErr("PASSPORT_NAME_TOO_LONG")
	ErrPassportBioTooLong          = passportErr("PASSPORT_BIO_TOO_LONG")
	ErrPassportTimezoneInvalid     = passportErr("PASSPORT_TIMEZONE_INVALID")
	ErrPassportInviteNotFound      = passportErr("PASSPORT_INVITE_NOT_FOUND")
	ErrPassportInviteCodeTaken     = passportErr("PASSPORT_INVITE_CODE_TAKEN")
	ErrPassportSettingsInvalid     = passportErr("PASSPORT_SETTINGS_INVALID")
	ErrPassportPlayerNotFound      = passportErr("PASSPORT_GAME_PLAYER_NOT_FOUND")
	ErrPassportTooManyRequests     = passportErr("PASSPORT_TOO_MANY_REQUESTS")
	ErrPassportPublicURLUnset      = passportErr("PASSPORT_PUBLIC_URL_UNSET")
	ErrPassportAuthNotFound        = passportErr("PASSPORT_AUTH_NOT_FOUND")
	ErrPassportAuthPending         = passportErr("PASSPORT_AUTH_PENDING")
	ErrPassportAuthSlowDown        = passportErr("PASSPORT_AUTH_SLOW_DOWN")
	ErrPassportAuthDenied          = passportErr("PASSPORT_AUTH_DENIED")
	ErrPassportAuthExpired         = passportErr("PASSPORT_AUTH_EXPIRED")
)

// PassportUserFilter 后台通行证列表的筛选。
type PassportUserFilter struct {
	// Status all / linked / unlinked / disabled / deleting
	Status   string
	Q        string
	ServerID string
	Offset   int
	Limit    int
}

// PassportStats 后台统计。
type PassportStats struct {
	Users, LinkedAccounts, Servers, NewThisWeek, NewThisWeekLinked, ActiveThisWeek, Disabled, Deleting int
	Games                                                                                              []string
}

// PassportExecFilter 玩家自己的 GM 记录（玩家自助）。
type PassportExecFilter struct {
	UserID   string
	ServerID string
	// Reviewable 只要需要审批的（待审批，或已被批准 / 驳回的）
	Reviewable bool
	Offset     int
	Limit      int
}

// PassportRepo 用户通行证的持久化：passport_users / sessions / login_logs / messages / invites，
// 以及通行证用户在 game_account_link、game_account_unlink、gm_execution、gift_code_redeem 里的记录。
type PassportRepo interface {
	// ---- 用户 ----
	CreateUser(ctx context.Context, u *gen.PassportUser) (*gen.PassportUser, error)
	GetUser(ctx context.Context, id string) (*gen.PassportUser, error)
	FindUserByUsername(ctx context.Context, username string) (*gen.PassportUser, error)
	FindUserByEmail(ctx context.Context, email string) (*gen.PassportUser, error)
	// SaveUser 写回全部可修改字段。
	SaveUser(ctx context.Context, u *gen.PassportUser) (*gen.PassportUser, error)
	ListUsers(ctx context.Context, f PassportUserFilter) ([]*gen.PassportUser, int, error)
	// UserCounts 各状态 chip 的数量（忽略 f.Status）。
	UserCounts(ctx context.Context, f PassportUserFilter) (map[string]int32, error)
	Stats(ctx context.Context, since time.Time) (*PassportStats, error)
	// DeleteUserCascade 删除用户与其会话、消息、登录记录、绑定（为每个绑定写一条 by=self 的解绑记录）。
	DeleteUserCascade(ctx context.Context, id string) error
	DueDeletions(ctx context.Context, now time.Time) ([]*gen.PassportUser, error)

	// ---- 会话 ----
	// CreateSession 同一用户同一设备只保留一行（旧的被替换）。
	CreateSession(ctx context.Context, s *gen.PassportSession) (*gen.PassportSession, error)
	GetSession(ctx context.Context, id string) (*gen.PassportSession, error)
	RotateSession(ctx context.Context, id, accessNoise, refreshNoise string, expiresAt time.Time) (*gen.PassportSession, error)
	ListSessions(ctx context.Context, userID string) ([]*gen.PassportSession, error)
	CountSessions(ctx context.Context, userID string) (int, error)
	DeleteSession(ctx context.Context, userID, sessionID string) error
	// DeleteSessions 删除用户的全部会话（except 非空时保留这一个）。
	DeleteSessions(ctx context.Context, userID, except string) error
	// TouchSession 记录最近活跃（实现方节流）。
	TouchSession(ctx context.Context, sessionID, ip string)

	// ---- 登录记录 ----
	AddLoginLog(ctx context.Context, l *gen.PassportLoginLog) error
	ListLoginLogs(ctx context.Context, userID string, limit int) ([]*gen.PassportLoginLog, error)
	// LoginHistory 此前是否成功登录过，以及是否在这台设备上成功登录过。
	LoginHistory(ctx context.Context, userID, deviceID string) (any, same bool, err error)

	// ---- 消息 ----
	AddMessages(ctx context.Context, msgs []*gen.PassportMessage) error
	ListMessages(ctx context.Context, userID string, unreadOnly bool, offset, limit int) ([]*gen.PassportMessage, int, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	ReadMessages(ctx context.Context, userID string, ids []string, all bool) error

	// ---- 邀请码 ----
	CreateInvite(ctx context.Context, i *gen.PassportInvite) (*gen.PassportInvite, error)
	GetInvite(ctx context.Context, id string) (*gen.PassportInvite, error)
	GetInviteByCode(ctx context.Context, code string) (*gen.PassportInvite, error)
	ListInvites(ctx context.Context, offset, limit int) ([]*gen.PassportInvite, int, error)
	// UseInvite 原子地把已用次数加一（已作废、过期或用完时返回 false）。
	UseInvite(ctx context.Context, id string, now time.Time) (bool, error)
	RevokeInvite(ctx context.Context, id string) (*gen.PassportInvite, error)

	// ---- 游戏账号绑定 ----
	ListLinks(ctx context.Context, userID string) ([]*gen.GameAccountLink, error)
	LinksOfUsers(ctx context.Context, userIDs []string) (map[string][]*gen.GameAccountLink, error)
	GetLink(ctx context.Context, id string) (*gen.GameAccountLink, error)
	LinkOf(ctx context.Context, userID, serverID string) (*gen.GameAccountLink, error)
	LinkByPlayer(ctx context.Context, serverID, pid string) (*gen.GameAccountLink, error)
	LinksByPlayers(ctx context.Context, serverID string, pids []string) ([]*gen.GameAccountLink, error)
	CreateLink(ctx context.Context, l *gen.GameAccountLink) (*gen.GameAccountLink, error)
	UpdateLinkSnapshot(ctx context.Context, l *gen.GameAccountLink) error
	TouchLink(ctx context.Context, id string, at time.Time) error
	// SetPrimary 设为该用户这款游戏的常用账号（同游戏的其他绑定取消）。
	SetPrimary(ctx context.Context, userID, game, linkID string) error
	// Unlink 删除绑定并写一条解绑记录（by：self / admin）。
	Unlink(ctx context.Context, l *gen.GameAccountLink, by string) error
	LastUnlink(ctx context.Context, userID, serverID string) (*gen.GameAccountUnlink, error)
	ListUnlinks(ctx context.Context, userID string, limit int) ([]*gen.GameAccountUnlink, error)
	// LinkCounts 每台服务器上已绑定的游戏账号数。
	LinkCounts(ctx context.Context) (map[string]int, error)
	// SelfServiceCounts 每台服务器开放给玩家的自助指令数。
	SelfServiceCounts(ctx context.Context) (map[string]int, error)

	// ---- 玩家自己的 GM 记录与兑换 ----
	ListPlayerExecutions(ctx context.Context, f PassportExecFilter) ([]*gen.GmExecution, int, error)
	// ReviewedSince 被审批（批准或驳回）时间晚于 since 的玩家申请。
	ReviewedSince(ctx context.Context, userID string, since time.Time) ([]*gen.GmExecution, error)
	ListRedeems(ctx context.Context, userID, serverID string, offset, limit int) ([]*gen.GiftCodeRedeem, int, error)
}

// PassportUsecase 用户通行证（玩家端）：独立于 Momoko 管理后台用户的玩家账号体系（自己的账号、会话与令牌），
// 绑定游戏账号、游戏空间、玩家自助操作与兑换码，以及游戏客户端的启动（按客户端应用 ID 查询启动目标、绑定或新建游戏账号、启动票据）。只服务注册在独立 passport server 上的玩家接口；
// 后台管理（PassportAdminService）由 PassportAdminUsecase 负责，两边互不调用。
// 自助操作与兑换码写进管理端也读写的表（GM 记录、玩家自助规则、兑换码），经 records / codes 访问。
type PassportUsecase struct {
	servers    *GameServerUsecase
	records    GmRepo
	codes      GameOpsRepo
	clientLogs GameClientLogRepo
	repo       PassportRepo
	config     ConfigRepo
	configUc   *ConfigUsecase
	system     *SystemUsecase
	log        *log.Helper

	// 邮箱验证码：用途+邮箱 → 验证码；最近一次发送时间（重发节流）。
	emailCodes    *cache.Cache[string, expiring[string]]
	emailCodeSent *cache.Cache[string, time.Time]
	// 登录失败计数（账号）与绑定时游戏账号密码失败计数（用户+服务器）。
	loginFails *cache.Cache[string, failCount]
	bindFails  *cache.Cache[string, failCount]
	// 两步验证：登录挑战、设置中的密钥、已用过的时间步（拒绝重放）。
	mfaChallenges *cache.Cache[string, expiring[mfaChallenge]]
	totpPending   *cache.Cache[string, expiring[string]]
	totpUsed      *cache.Cache[string, int64]
	// 客户端日志上传：每个游戏账号（服务器+pid）每小时的上传次数。
	clientLogRate *slidingLimit
	// 客户端浏览器授权：待授权的请求（键为 device_code 的 SHA-256）、user_code → 请求、发起的限流（按 IP）、
	// 授权页输错 user_code 的次数（按用户）；状态变化都在 clientAuthMu 下进行。
	clientAuths      *cache.Cache[string, *clientAuth]
	clientAuthCodes  *cache.Cache[string, string]
	clientAuthRate   *slidingLimit
	clientAuthMisses *cache.Cache[string, failCount]
	clientAuthMu     sync.Mutex

	// bindLocks 串行化同一服务器上的绑定写入（键为服务器 id），以及同一用户在同一服务器上的绑定与新建游戏账号（键为用户 id + 服务器 id）。
	bindLocks sync.Map
	// selfServiceLocks 串行化同一玩家同一自助操作，避免并发绕过次数限制。
	selfServiceLocks sync.Map
	// redeemLocks 串行化同一玩家、同一批次的兑换。
	redeemLocks sync.Map
}

// expiring 带过期时间的缓存值（pkg/cache 只做滑动清理，这里自己判断过期）。
type expiring[T any] struct {
	V   T
	Exp time.Time
}

func (e expiring[T]) alive() bool { return time.Now().Before(e.Exp) }

type failCount struct {
	N     int
	First time.Time
}

// slidingLimit 按键限流：滑动窗口内最多 limit 次，被拒的不计数。
type slidingLimit struct {
	mu     sync.Mutex
	hits   *cache.Cache[string, []time.Time]
	limit  int
	window time.Duration
}

func newSlidingLimit(limit int, window time.Duration) *slidingLimit {
	return &slidingLimit{hits: cache.New[string, []time.Time](window), limit: limit, window: window}
}

// allow 记一次；窗口内已满时拒绝。
func (l *slidingLimit) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent, _ := l.hits.Get(key)
	kept := make([]time.Time, 0, len(recent)+1)
	for _, at := range recent {
		if now.Sub(at) < l.window {
			kept = append(kept, at)
		}
	}
	if len(kept) >= l.limit {
		l.hits.Set(key, kept)
		return false
	}
	l.hits.Set(key, append(kept, now))
	return true
}

func NewPassportUsecase(servers *GameServerUsecase, records GmRepo, codes GameOpsRepo, clientLogs GameClientLogRepo, repo PassportRepo, config ConfigRepo, configUc *ConfigUsecase, system *SystemUsecase, logger log.Logger) *PassportUsecase {
	uc := &PassportUsecase{
		servers:       servers,
		records:       records,
		codes:         codes,
		clientLogs:    clientLogs,
		repo:          repo,
		config:        config,
		configUc:      configUc,
		system:        system,
		log:           log.NewHelper(log.With(logger, "module", "biz/passport")),
		emailCodes:    cache.New[string, expiring[string]](EmailCodeTTL),
		emailCodeSent: cache.New[string, time.Time](EmailCodeResendInterval),
		loginFails:    cache.New[string, failCount](passportLockWindow),
		bindFails:     cache.New[string, failCount](passportLockWindow),
		mfaChallenges: cache.New[string, expiring[mfaChallenge]](passportMfaTTL),
		totpPending:   cache.New[string, expiring[string]](15 * time.Minute),
		totpUsed:      cache.New[string, int64](5 * time.Minute),
		clientLogRate: newSlidingLimit(clientLogRateLimit, clientLogRateWindow),

		clientAuths:      cache.New[string, *clientAuth](clientAuthTTL),
		clientAuthCodes:  cache.New[string, string](clientAuthTTL),
		clientAuthRate:   newSlidingLimit(clientAuthRateLimit, clientAuthRateWindow),
		clientAuthMisses: cache.New[string, failCount](passportLockWindow),
	}
	go uc.deletionLoop()
	return uc
}

// Servers 游戏服务器用例。
func (uc *PassportUsecase) Servers() *GameServerUsecase { return uc.servers }

// ---------------- 设置 ----------------

func defaultPassportSettings() *v1.PassportSettings {
	return &v1.PassportSettings{
		RegisterMode:              PassportRegisterOpen,
		RegisterEmailVerification: false,
		EmailLoginEnabled:         true,
		SessionDays:               30,
	}
}

// Settings 读取通行证设置（没保存过时为默认值）。
func (uc *PassportUsecase) Settings(ctx context.Context) *v1.PassportSettings {
	return passportSettings(ctx, uc.config, uc.log)
}

// passportSettings 读取通行证设置（没保存过时为默认值；玩家端按它运行，后台编辑它）。
func passportSettings(ctx context.Context, config ConfigRepo, logger *log.Helper) *v1.PassportSettings {
	out := defaultPassportSettings()
	raw, err := config.Get(ctx, common.ConfigPassportSettings)
	if err != nil || strings.TrimSpace(raw) == "" {
		return out
	}
	saved := &v1.PassportSettings{}
	if err := protojson.Unmarshal([]byte(raw), saved); err != nil {
		logger.Warnf("parse passport settings: %v", err)
		return out
	}
	normalizePassportSettings(saved)
	return saved
}

func normalizePassportSettings(s *v1.PassportSettings) {
	switch s.RegisterMode {
	case PassportRegisterOpen, PassportRegisterInviteOnly, PassportRegisterClosed:
	default:
		s.RegisterMode = PassportRegisterOpen
	}
	if s.SessionDays <= 0 {
		s.SessionDays = 30
	}
	if s.SessionDays > 365 {
		s.SessionDays = 365
	}
	s.SiteName = strings.TrimSpace(s.SiteName)
}

// EmailEnabled 是否配置并开启了邮件。
func (uc *PassportUsecase) EmailEnabled(ctx context.Context) bool {
	return passportEmailEnabled(ctx, uc.config)
}

func passportEmailEnabled(ctx context.Context, config ConfigRepo) bool {
	cfg, err := config.EmailConfig(ctx)
	return err == nil && cfg.Enabled
}

// SiteName 通行证显示名称：通行证设置里的名称，空时用 Momoko 站点名称。
func (uc *PassportUsecase) SiteName(ctx context.Context, s *v1.PassportSettings) string {
	if s.SiteName != "" {
		return s.SiteName
	}
	if site, err := uc.config.SiteConfig(ctx); err == nil {
		return site.SiteName
	}
	return ""
}

// PublicConfig 公开配置（登录页、注册页用）。
func (uc *PassportUsecase) PublicConfig(ctx context.Context) *v1.GetPassportConfigResponse {
	s := uc.Settings(ctx)
	email := uc.EmailEnabled(ctx)
	return &v1.GetPassportConfigResponse{
		SiteName:                  uc.SiteName(ctx, s),
		RegisterMode:              s.RegisterMode,
		EmailLoginEnabled:         email && s.EmailLoginEnabled,
		RegisterEmailVerification: email && s.RegisterEmailVerification,
		EmailEnabled:              email,
		UnbindCooldownDays:        int32(PassportUnbindCooldown / (24 * time.Hour)),
		DeletionGraceDays:         int32(PassportDeletionGrace / (24 * time.Hour)),
	}
}

// ---------------- 小工具 ----------------

// maskEmail 只显示邮箱的一部分（后台列表）：a***@example.com。
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		if email == "" {
			return ""
		}
		return "***"
	}
	name := []rune(email[:at])
	return string(name[0]) + "***" + email[at:]
}

// displayName 通行证的显示名称（昵称为空时用用户名）。
func displayName(u *gen.PassportUser) string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

// deletionLoop 定时删除注销期满的通行证。
func (uc *PassportUsecase) deletionLoop() {
	time.Sleep(20 * time.Second)
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		uc.purgeDeleted()
		<-ticker.C
	}
}

func (uc *PassportUsecase) purgeDeleted() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	users, err := uc.repo.DueDeletions(ctx, time.Now())
	if err != nil {
		uc.log.Warnf("list due passport deletions: %v", err)
		return
	}
	for _, u := range users {
		if err := uc.repo.DeleteUserCascade(ctx, u.ID); err != nil {
			uc.log.Errorf("delete passport user %s: %v", u.ID, err)
			continue
		}
		uc.configUc.deleteMedia(ctx, u.AvatarKey)
		uc.log.Infof("passport user %s deleted after the grace period", u.ID)
	}
}
