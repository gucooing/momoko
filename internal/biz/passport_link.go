package biz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/gameaccountlink"
	"momoko/internal/data/ent/gen/gameserver"
	"momoko/pkg/mgp"
)

// MGP 能力：校验游戏账号密码。
const mgpCapAccountsVerify = "accounts.verify"

// 绑定方式（game_account_link.method）。
const (
	PassportBindPassword = "password"
	PassportBindAdmin    = "admin"
)

// linkProto 绑定转为接口结构。
func (uc *PassportUsecase) linkProto(ctx context.Context, l *gen.GameAccountLink, servers map[string]*gen.GameServer, selfService map[string]int) *v1.PassportLink {
	out := &v1.PassportLink{
		Id:                 l.ID,
		Game:               l.Game,
		PlayerId:           l.PlayerID,
		DisplayId:          l.DisplayID,
		Account:            l.Account,
		Nickname:           l.Nickname,
		AvatarAsset:        l.AvatarAsset,
		Level:              int32(l.Level),
		Method:             string(l.Method),
		IsPrimary:          l.IsPrimary,
		LinkedAt:           timestamppb.New(l.LinkedAt),
		LastSeenAt:         tsOrNil(l.LastSeenAt),
		SelfServiceEnabled: int32(selfService[l.ServerID]),
	}
	if s, ok := servers[l.ServerID]; ok {
		out.Server = uc.Servers().Brief(ctx, s)
	} else {
		out.Server = &v1.GameServerBrief{Id: l.ServerID, Game: l.Game, Tags: []string{}, Online: -1, Registered: -1}
	}
	return out
}

func (uc *PassportUsecase) selfServiceCounts(ctx context.Context) map[string]int {
	counts, err := uc.repo.SelfServiceCounts(ctx)
	if err != nil {
		return map[string]int{}
	}
	return counts
}

// sortLinks 按最近使用排序（没进过游戏空间的按绑定时间）。
func sortLinks(list []*gen.GameAccountLink) {
	key := func(l *gen.GameAccountLink) time.Time {
		if l.LastSeenAt != nil && l.LastSeenAt.After(l.LinkedAt) {
			return *l.LastSeenAt
		}
		return l.LinkedAt
	}
	sort.SliceStable(list, func(i, j int) bool { return key(list[i]).After(key(list[j])) })
}

// Links 我绑定的游戏账号。
func (uc *PassportUsecase) Links(ctx context.Context, userID string) (*v1.ListPassportLinksResponse, error) {
	list, err := uc.repo.ListLinks(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	sortLinks(list)
	servers, ss := uc.serverMap(ctx), uc.selfServiceCounts(ctx)
	out := &v1.ListPassportLinksResponse{}
	for _, l := range list {
		out.Items = append(out.Items, uc.linkProto(ctx, l, servers, ss))
	}
	return out, nil
}

// myLink 读取当前用户的一条绑定。
func (uc *PassportUsecase) myLink(ctx context.Context, userID, linkID string) (*gen.GameAccountLink, error) {
	l, err := uc.repo.GetLink(ctx, linkID)
	if gen.IsNotFound(err) || (err == nil && (l.PassportUserID != userID || l.State != gameaccountlink.StateActive)) {
		return nil, ErrPassportLinkNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return l, nil
}

// SetPrimary 设为这款游戏的常用账号。
func (uc *PassportUsecase) SetPrimary(ctx context.Context, userID, linkID string) error {
	l, err := uc.myLink(ctx, userID, linkID)
	if err != nil {
		return err
	}
	if err := uc.repo.SetPrimary(ctx, userID, l.Game, l.ID); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// unlink 解绑并在同一游戏下补一个常用账号。
func (uc *PassportUsecase) unlink(ctx context.Context, l *gen.GameAccountLink, by string) error {
	if err := uc.repo.Unlink(ctx, l, by); err != nil {
		return ErrSystem(err)
	}
	if l.IsPrimary {
		if rest, err := uc.repo.ListLinks(ctx, l.PassportUserID); err == nil {
			sortLinks(rest)
			for _, other := range rest {
				if other.Game == l.Game {
					_ = uc.repo.SetPrimary(ctx, l.PassportUserID, l.Game, other.ID)
					break
				}
			}
		}
	}
	return nil
}

// Unbind 玩家自己解绑。
func (uc *PassportUsecase) Unbind(ctx context.Context, userID, linkID string) error {
	l, err := uc.myLink(ctx, userID, linkID)
	if err != nil {
		return err
	}
	return uc.unlink(ctx, l, "self")
}

// cooldownUntil 解绑冷却结束时间（不在冷却中为 nil）。
func (uc *PassportUsecase) cooldownUntil(ctx context.Context, userID, serverID string) (*time.Time, string) {
	last, err := uc.repo.LastUnlink(ctx, userID, serverID)
	if err != nil || last == nil {
		return nil, ""
	}
	until := last.UnlinkedAt.Add(PassportUnbindCooldown)
	if time.Now().Before(until) {
		return &until, last.PlayerID
	}
	return nil, ""
}

// inviteServer 用户注册时用的邀请码附带的服务器（可空）。
func (uc *PassportUsecase) inviteServer(ctx context.Context, u *gen.PassportUser) string {
	if u.InviteID == "" {
		return ""
	}
	inv, err := uc.repo.GetInvite(ctx, u.InviteID)
	if err != nil {
		return ""
	}
	return inv.ServerID
}

// passportVisible 服务器能不能在绑定页出现：公开的；或凭邀请码（服务器的邀请码、或注册邀请码附带了这台服务器）。
func passportVisible(s *gen.GameServer, inviteCode, inviteServer string) (visible, viaInvite bool) {
	switch s.PassportVisibility {
	case gameserver.PassportVisibilityPublic:
		return true, false
	case gameserver.PassportVisibilityInvite:
		if inviteServer == s.ID {
			return true, true
		}
		code := strings.TrimSpace(inviteCode)
		if code != "" && s.PassportInviteCode != "" && strings.EqualFold(code, s.PassportInviteCode) {
			return true, true
		}
	}
	return false, false
}

// bindMethods 服务器支持的绑定方式（由 manifest 声明的能力决定）。
func bindMethods(s *gen.GameServer) []string {
	m := cachedManifestOf(s)
	if m == nil || !MGPConfigured(s) {
		return nil
	}
	var out []string
	if m.HasCapability(mgpCapAccountsVerify) {
		out = append(out, PassportBindPassword)
	}
	return out
}

func gameNameOf(s *gen.GameServer) (name, icon string) {
	if m := cachedManifestOf(s); m != nil {
		return m.Game.Name, m.Game.Icon
	}
	return "", ""
}

// SearchBindServers 绑定页选服务器：搜索（名称或邀请码，最多 20 条）+ 推荐（最多 5 个公开服务器）。
func (uc *PassportUsecase) SearchBindServers(ctx context.Context, userID, q, game string) (*v1.SearchPassportBindServersResponse, error) {
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
	linked := map[string]bool{}
	for _, l := range links {
		linked[l.ServerID] = true
	}
	counts, _ := uc.repo.LinkCounts(ctx)
	ss := uc.selfServiceCounts(ctx)
	invServer := uc.inviteServer(ctx, u)
	q = strings.TrimSpace(q)
	lq := strings.ToLower(q)

	toBind := func(s *gen.GameServer, viaInvite bool) *v1.PassportBindServer {
		b := &v1.PassportBindServer{
			Server:         uc.Servers().Brief(ctx, s),
			LinkedAccounts: int32(counts[s.ID]),
			SelfService:    int32(ss[s.ID]),
			Methods:        bindMethods(s),
			ViaInvite:      viaInvite,
		}
		if b.Methods == nil {
			b.Methods = []string{}
		}
		switch {
		case linked[s.ID]:
			b.Reason = "linked"
		case !MGPConfigured(s):
			b.Reason = "no_mgp"
		case len(b.Methods) == 0:
			b.Reason = "unsupported"
		default:
			if until, _ := uc.cooldownUntil(ctx, userID, s.ID); until != nil {
				b.Reason, b.CooldownUntil = "cooldown", timestamppb.New(*until)
			} else {
				b.Bindable = true
			}
		}
		return b
	}

	out := &v1.SearchPassportBindServersResponse{}
	games := map[string]*v1.PassportBindGame{}
	var matched, public []*gen.GameServer
	invite := map[string]bool{}
	for _, s := range all {
		visible, viaInvite := passportVisible(s, q, invServer)
		if !visible {
			continue
		}
		out.Total++
		g := games[s.Game]
		if g == nil {
			name, icon := gameNameOf(s)
			g = &v1.PassportBindGame{Id: s.Game, Name: name, Icon: icon}
			games[s.Game] = g
		}
		g.Servers++
		if game != "" && s.Game != game {
			continue
		}
		if !viaInvite || s.PassportVisibility == gameserver.PassportVisibilityPublic {
			if s.PassportVisibility == gameserver.PassportVisibilityPublic {
				public = append(public, s)
			}
		}
		if viaInvite {
			invite[s.ID] = true
		}
		if q != "" && (viaInvite || gameServerMatches(s, lq)) {
			matched = append(matched, s)
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if invite[matched[i].ID] != invite[matched[j].ID] {
			return invite[matched[i].ID]
		}
		return matched[i].Name < matched[j].Name
	})
	for i, s := range matched {
		if i >= 20 {
			break
		}
		out.Items = append(out.Items, toBind(s, invite[s.ID]))
	}
	// 推荐：公开、可绑定、绑定的人多的在前。
	sort.SliceStable(public, func(i, j int) bool {
		bi, bj := len(bindMethods(public[i])) > 0, len(bindMethods(public[j])) > 0
		if bi != bj {
			return bi
		}
		if counts[public[i].ID] != counts[public[j].ID] {
			return counts[public[i].ID] > counts[public[j].ID]
		}
		return public[i].Name < public[j].Name
	})
	for i, s := range public {
		if i >= 5 {
			break
		}
		out.Recommended = append(out.Recommended, toBind(s, false))
	}
	// 凭注册邀请码可见的服务器也放进推荐。
	if invServer != "" && q == "" {
		for _, s := range all {
			if s.ID == invServer && s.PassportVisibility == gameserver.PassportVisibilityInvite && (game == "" || s.Game == game) {
				out.Recommended = append([]*v1.PassportBindServer{toBind(s, true)}, out.Recommended...)
			}
		}
	}
	for _, g := range games {
		out.Games = append(out.Games, g)
	}
	sort.SliceStable(out.Games, func(i, j int) bool { return out.Games[i].Id < out.Games[j].Id })
	return out, nil
}

func (uc *PassportUsecase) bindLock(serverID string) *sync.Mutex {
	l, _ := uc.bindLocks.LoadOrStore(serverID, &sync.Mutex{})
	return l.(*sync.Mutex)
}

// Bind 用游戏账号 + 密码绑定：游戏服务端校验，密码不保存。
func (uc *PassportUsecase) Bind(ctx context.Context, userID string, req *v1.BindPassportAccountRequest) (*v1.PassportLink, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	s, err := uc.Servers().Get(ctx, req.ServerId)
	if err != nil {
		return nil, ErrPassportServerNotBindable
	}
	if visible, _ := passportVisible(s, req.InviteCode, uc.inviteServer(ctx, u)); !visible {
		return nil, ErrPassportServerNotBindable
	}
	account := strings.TrimSpace(req.Account)
	if account == "" || req.Password == "" {
		return nil, ErrPassportInvalidArgument
	}
	client, err := uc.Servers().Client(s)
	if err != nil {
		return nil, ErrPassportBindUnsupported
	}
	m, err := uc.Servers().Manifest(ctx, s)
	if err != nil {
		return nil, ErrPassportGameUnavailable
	}
	if !m.HasCapability(mgpCapAccountsVerify) {
		return nil, ErrPassportBindUnsupported
	}
	if existing, err := uc.repo.LinkOf(ctx, userID, s.ID); err == nil && existing != nil {
		return nil, ErrPassportAlreadyLinked
	} else if err != nil && !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	failKey := userID + "\x00" + s.ID
	if f, ok := uc.bindFails.Get(failKey); ok && f.N >= passportBindAttempts && time.Since(f.First) < passportLockWindow {
		return nil, ErrPassportBindTooMany
	}
	raw, err := client.Do(ctx, mgp.Call{
		Method: http.MethodPost,
		Path:   "/accounts/verify",
		Body:   map[string]string{"account": account, "password": req.Password},
	})
	if err != nil {
		switch {
		case mgp.CodeOf(err) == "CREDENTIALS_INVALID":
			bumpFail(uc.bindFails, failKey)
			return nil, ErrPassportGameCredentials
		case mgp.IsUnreachable(err):
			return nil, ErrPassportGameUnavailable
		default:
			return nil, GameMGPError(err)
		}
	}
	uc.bindFails.Del(failKey)
	var p mgp.Player
	if err := json.Unmarshal(raw, &p); err != nil || p.PID == "" {
		return nil, ErrPassportGameUnavailable
	}
	if until, lastPID := uc.cooldownUntil(ctx, userID, s.ID); until != nil && lastPID != p.PID {
		return nil, ErrPassportBindCooldown
	}
	l, err := uc.createLink(ctx, u, s, &p, PassportBindPassword)
	if err != nil {
		return nil, err
	}
	return uc.linkProto(ctx, l, map[string]*gen.GameServer{s.ID: s}, uc.selfServiceCounts(ctx)), nil
}

// createLink 写入绑定（同一服务器上一个游戏账号只能绑一个通行证，一个通行证只能绑一个游戏账号）。
func (uc *PassportUsecase) createLink(ctx context.Context, u *gen.PassportUser, s *gen.GameServer, p *mgp.Player, method string) (*gen.GameAccountLink, error) {
	lock := uc.bindLock(s.ID)
	lock.Lock()
	defer lock.Unlock()
	if other, err := uc.repo.LinkByPlayer(ctx, s.ID, p.PID); err == nil && other != nil {
		if other.PassportUserID == u.ID {
			return nil, ErrPassportAlreadyLinked
		}
		return nil, ErrPassportAccountTaken
	} else if err != nil && !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	if existing, err := uc.repo.LinkOf(ctx, u.ID, s.ID); err == nil && existing != nil {
		return nil, ErrPassportAlreadyLinked
	}
	primary := true
	if links, err := uc.repo.ListLinks(ctx, u.ID); err == nil {
		for _, l := range links {
			if l.Game == s.Game {
				primary = false
				break
			}
		}
	}
	l, err := uc.repo.CreateLink(ctx, &gen.GameAccountLink{
		ID:             uuid.NewString(),
		PassportUserID: u.ID,
		ServerID:       s.ID,
		Game:           s.Game,
		PlayerID:       p.PID,
		DisplayID:      p.DisplayID,
		Account:        p.Account,
		Nickname:       p.Name,
		AvatarAsset:    p.AvatarIdolCardID,
		Level:          p.Level,
		Method:         gameaccountlink.Method(method),
		State:          gameaccountlink.StateActive,
		IsPrimary:      primary,
		LinkedAt:       time.Now(),
	})
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, ErrPassportAlreadyLinked
		}
		return nil, ErrSystem(err)
	}
	return l, nil
}

// refreshSnapshot 用 MGP 玩家条目刷新绑定上的昵称、等级、头像快照（变化时才写库）。
func (uc *PassportUsecase) refreshSnapshot(ctx context.Context, l *gen.GameAccountLink, raw json.RawMessage) {
	var p mgp.Player
	if json.Unmarshal(raw, &p) != nil || p.PID == "" {
		return
	}
	if p.Name == l.Nickname && p.Level == l.Level && p.DisplayID == l.DisplayID && p.AvatarIdolCardID == l.AvatarAsset && p.Account == l.Account {
		return
	}
	l.Nickname, l.Level, l.DisplayID, l.AvatarAsset, l.Account = p.Name, p.Level, p.DisplayID, p.AvatarIdolCardID, p.Account
	if err := uc.repo.UpdateLinkSnapshot(ctx, l); err != nil {
		uc.log.Warnf("update link snapshot: %v", err)
	}
}

// playerPath MGP 玩家路径。
func playerPath(pid string) string { return "/players/" + url.PathEscape(pid) }
