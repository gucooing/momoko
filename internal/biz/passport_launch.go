package biz

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	gk "momoko/api/gen/v1/gakumas"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
)

// 启动器：游戏客户端（PC 启动器、Android APK）用通行证进入游戏，用客户端应用 ID 指明目标服务器。
// 每台可启动的服务器有一组 Momoko 生成的客户端应用凭据：应用 ID（game_servers.client_app_id，公开标识，
// 也打包进单服务器客户端）与应用密钥（加密保存，只配置在游戏服务端；学マス：momoko.app_id、momoko.app_secret）。客户端先登录通行证，再查询启动目标
// （GET /api/v1/launch/apps 列出全部，GET /api/v1/launch/apps/{appId} 查一个），没有游戏账号时绑定已有账号
// （POST /api/v1/launch/apps/{appId}/bind）或新建一个（POST /api/v1/launch/apps/{appId}/account），
// 最后申请启动票据（POST /api/v1/launch/apps/{appId}/ticket），交给那台服务器的 SDK（POST /passport/ticket-login）换取游戏会话。
//
// 启动票据用这台服务器的应用密钥签名（密钥字符串的 UTF-8 字节；MGP 密钥不参与）：
//
//	"lt1." + b64url(payload) + "." + b64url(HMAC-SHA256(app secret, "momoko-launch-ticket.v1." + b64url(payload)))
//
// payload 为紧凑 JSON，字段依次为 {"aid": 应用 ID, "sid": 服务器 id, "pid": 游戏账号 id, "uid": 通行证用户 id,
// "jti": 16 字节随机数的 b64url, "iat": 签发时间, "exp": iat+120}（时间为 Unix 秒），b64url 为无填充的 base64url。
// 游戏服务端校验 aid、签名与有效期，并保证同一个 jti 只能用一次。

const (
	launchTicketPrefix  = "lt1."
	launchTicketContext = "momoko-launch-ticket.v1."
	// LaunchTicketTTL 启动票据的有效期。
	LaunchTicketTTL = 120 * time.Second
)

// 不能在服务器上绑定或新建游戏账号的原因（block_reason）。
const (
	launchBlockUnsupported = "unsupported"
	launchBlockNotVisible  = "not_visible"
	launchBlockCooldown    = "cooldown"
)

// launchTicketPayload 启动票据的内容；字段顺序即 JSON 里的顺序。
type launchTicketPayload struct {
	Aid string `json:"aid"`
	Sid string `json:"sid"`
	Pid string `json:"pid"`
	Uid string `json:"uid"`
	Jti string `json:"jti"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

// clientLaunchSupported 服务器的游戏支持客户端启动且接入了 MGP。
func clientLaunchSupported(s *gen.GameServer) bool {
	a, ok := GameAdapterByID(s.Game)
	return ok && a.ClientLaunch && MGPConfigured(s)
}

// launchable 服务器可以用通行证启动：游戏支持客户端启动、接入了 MGP、生成了客户端应用凭据、填写了 SDK 与游戏服务端地址。
// 资源服务器地址可以不填，客户端这时使用官方资源。不满足的服务器对客户端来说不存在。
func launchable(s *gen.GameServer) bool {
	return clientLaunchSupported(s) && derefString(s.ClientAppID) != "" && s.ClientAppSecret != "" &&
		s.ClientSdkURL != "" && s.ClientGameURL != ""
}

// launchServer 客户端应用 ID 对应的可启动服务器；ID 不存在或服务器不可启动时为 ErrPassportAppNotFound。
func (uc *PassportUsecase) launchServer(ctx context.Context, appID string) (*gen.GameServer, error) {
	s, err := uc.Servers().ByClientAppID(ctx, strings.TrimSpace(appID))
	if errors.Is(err, ErrGameServerNotFound) {
		return nil, ErrPassportAppNotFound
	}
	if err != nil {
		return nil, err
	}
	if !launchable(s) {
		return nil, ErrPassportAppNotFound
	}
	return s, nil
}

// gameAccountBlock 用户在服务器上绑定或新建游戏账号的限制：游戏支持客户端启动且接入了 MGP、
// 服务器对用户可见（公开，或是注册邀请码所属的服务器）、不在解绑冷却中（冷却中只能绑回原来的账号）。返回空串表示可以。
func (uc *PassportUsecase) gameAccountBlock(ctx context.Context, userID string, s *gen.GameServer, inviteServer string) string {
	if !clientLaunchSupported(s) {
		return launchBlockUnsupported
	}
	if visible, _ := passportVisible(s, "", inviteServer); !visible {
		return launchBlockNotVisible
	}
	if until, _ := uc.cooldownUntil(ctx, userID, s.ID); until != nil {
		return launchBlockCooldown
	}
	return ""
}

// launchApp 一个启动目标：有绑定时带上绑定，没有时带上能否绑定或新建。
func (uc *PassportUsecase) launchApp(ctx context.Context, userID string, s *gen.GameServer, l *gen.GameAccountLink, inviteServer string, servers map[string]*gen.GameServer, selfService map[string]int) *v1.PassportLaunchApp {
	out := &v1.PassportLaunchApp{
		AppId:  derefString(s.ClientAppID),
		Server: uc.Servers().Brief(ctx, s),
		Endpoints: &v1.PassportLaunchEndpoints{
			Sdk:      s.ClientSdkURL,
			Game:     s.ClientGameURL,
			Resource: s.ClientResourceURL,
		},
		Icon:       s.AppIcon,
		Background: s.AppBackground,
	}
	if l != nil {
		out.Link = uc.linkProto(ctx, l, servers, selfService)
		return out
	}
	reason := uc.gameAccountBlock(ctx, userID, s, inviteServer)
	out.Bindable, out.BlockReason = reason == "", reason
	return out
}

// LaunchApps 可启动、并且用户在上面有绑定或服务器对用户可见的启动目标（PC 启动器）。
// 有绑定的在前（按最近使用倒序），其余按服务器名称。
func (uc *PassportUsecase) LaunchApps(ctx context.Context, userID string) (*v1.ListPassportLaunchAppsResponse, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	all, err := uc.Servers().List(ctx)
	if err != nil {
		return nil, err
	}
	links, err := uc.repo.ListLinks(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	byServer := map[string]*gen.GameAccountLink{}
	for _, l := range links {
		if l.PlayerID != "" {
			byServer[l.ServerID] = l
		}
	}
	invServer := uc.inviteServer(ctx, u)
	servers := map[string]*gen.GameServer{}
	var linked []*gen.GameAccountLink
	var others []*gen.GameServer
	for _, s := range all {
		if !launchable(s) {
			continue
		}
		servers[s.ID] = s
		if l, ok := byServer[s.ID]; ok {
			linked = append(linked, l)
			continue
		}
		if visible, _ := passportVisible(s, "", invServer); visible {
			others = append(others, s)
		}
	}
	sortLinks(linked)
	sort.SliceStable(others, func(i, j int) bool { return others[i].Name < others[j].Name })

	ss := uc.selfServiceCounts(ctx)
	out := &v1.ListPassportLaunchAppsResponse{Apps: []*v1.PassportLaunchApp{}}
	for _, l := range linked {
		out.Apps = append(out.Apps, uc.launchApp(ctx, userID, servers[l.ServerID], l, invServer, servers, ss))
	}
	for _, s := range others {
		out.Apps = append(out.Apps, uc.launchApp(ctx, userID, s, nil, invServer, servers, ss))
	}
	return out, nil
}

// LaunchApp 一个启动目标（单服务器客户端）：服务器对用户不可见、也没有绑定时照常返回，bindable=false、block_reason=not_visible。
func (uc *PassportUsecase) LaunchApp(ctx context.Context, userID, appID string) (*v1.PassportLaunchApp, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	s, err := uc.launchServer(ctx, appID)
	if err != nil {
		return nil, err
	}
	l, err := uc.gameLink(ctx, userID, s.ID)
	if errors.Is(err, ErrPassportLinkNotFound) {
		l = nil
	} else if err != nil {
		return nil, err
	}
	servers := map[string]*gen.GameServer{s.ID: s}
	return uc.launchApp(ctx, userID, s, l, uc.inviteServer(ctx, u), servers, uc.selfServiceCounts(ctx)), nil
}

// BindLaunchApp 用游戏账号 + 密码把应用对应服务器上已有的游戏账号绑定到通行证（规则与通行证网页的绑定相同）。
func (uc *PassportUsecase) BindLaunchApp(ctx context.Context, userID string, req *v1.BindPassportLaunchAppRequest) (*v1.PassportLink, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	s, err := uc.launchServer(ctx, req.AppId)
	if err != nil {
		return nil, err
	}
	if visible, _ := passportVisible(s, "", uc.inviteServer(ctx, u)); !visible {
		return nil, ErrPassportServerNotBindable
	}
	return uc.bindAccount(ctx, u, s, req.Account, req.Password)
}

// CreateLaunchAccount 经 MGP 在应用对应的服务器上新建一个游戏账号（执行者为通行证显示名，新的幂等键）并以 create 方式绑定。
// 限制与绑定相同：服务器可见、游戏支持、不在解绑冷却中、还没有绑定。同一用户在同一服务器上的绑定与新建串行执行，
// 并发的两次请求只建一个账号。
func (uc *PassportUsecase) CreateLaunchAccount(ctx context.Context, userID, appID string) (*v1.PassportLink, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	s, err := uc.launchServer(ctx, appID)
	if err != nil {
		return nil, err
	}
	lock := uc.bindLock(userID + "\x00" + s.ID)
	lock.Lock()
	defer lock.Unlock()
	if existing, err := uc.repo.LinkOf(ctx, userID, s.ID); err == nil && existing != nil {
		return nil, ErrPassportAlreadyLinked
	} else if err != nil && !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	switch uc.gameAccountBlock(ctx, userID, s, uc.inviteServer(ctx, u)) {
	case launchBlockUnsupported:
		return nil, ErrPassportBindUnsupported
	case launchBlockNotVisible:
		return nil, ErrPassportServerNotBindable
	case launchBlockCooldown:
		return nil, ErrPassportBindCooldown
	}
	client, err := uc.Servers().Client(s)
	if err != nil {
		return nil, ErrPassportBindUnsupported
	}
	p, err := createGameAccount(ctx, s, client, displayName(u))
	if err != nil {
		return nil, err
	}
	l, err := uc.createLink(ctx, u, s, p, PassportBindCreate)
	if err != nil {
		uc.log.Errorf("launch: game account %s created on server %s but not linked to passport user %s: %v", p.GetPid(), s.ID, u.ID, err)
		return nil, err
	}
	return uc.linkProto(ctx, l, map[string]*gen.GameServer{s.ID: s}, uc.selfServiceCounts(ctx)), nil
}

// createGameAccount 调用游戏的 MGP 接口新建一个游戏账号，返回它的玩家条目。
func createGameAccount(ctx context.Context, s *gen.GameServer, client *mgp.Client, operator string) (*gk.Player, error) {
	if s.Game != gakumas.ID {
		return nil, ErrPassportBindUnsupported
	}
	p, err := gakumas.New(client).CreateAccount(ctx, mgp.OperationOptions{
		Operator:       operator,
		IdempotencyKey: "passport-" + uuid.NewString(),
	})
	if err != nil {
		if mgp.IsUnreachable(err) {
			return nil, ErrPassportGameUnavailable
		}
		return nil, GameMGPError(err)
	}
	if p.GetPid() == "" {
		return nil, ErrPassportGameUnavailable
	}
	return p, nil
}

// IssueLaunchTicket 为用户在应用对应服务器上绑定的游戏账号签发启动票据，并记录绑定的最近使用时间。
func (uc *PassportUsecase) IssueLaunchTicket(ctx context.Context, userID, appID string) (*v1.PassportLaunchTicket, error) {
	s, err := uc.launchServer(ctx, appID)
	if err != nil {
		return nil, err
	}
	l, err := uc.gameLink(ctx, userID, s.ID)
	if errors.Is(err, ErrPassportLinkNotFound) {
		return nil, ErrPassportGameAccountRequired
	}
	if err != nil {
		return nil, err
	}
	secret, err := gameBox().Decrypt(s.ClientAppSecret)
	if err != nil {
		uc.log.Errorf("launch: decrypt client app secret of server %s: %v", s.ID, err)
		return nil, ErrPassportGameUnavailable
	}
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return nil, ErrSystem(err)
	}
	now := time.Now()
	ticket, err := signLaunchTicket(secret, launchTicketPayload{
		Aid: derefString(s.ClientAppID),
		Sid: s.ID,
		Pid: l.PlayerID,
		Uid: userID,
		Jti: base64.RawURLEncoding.EncodeToString(jti),
		Iat: now.Unix(),
		Exp: now.Add(LaunchTicketTTL).Unix(),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	if err := uc.repo.TouchLink(ctx, l.ID, now); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.PassportLaunchTicket{
		Ticket:    ticket,
		ExpiresIn: int64(LaunchTicketTTL / time.Second),
		PlayerId:  l.PlayerID,
		DisplayId: l.DisplayID,
	}, nil
}

// signLaunchTicket 用应用密钥（UTF-8 字节）签名启动票据。
func signLaunchTicket(secret string, p launchTicketPayload) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(launchTicketContext + body))
	return launchTicketPrefix + body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
