package biz

import (
	"context"
	"crypto/rand"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	gk "momoko/api/gen/v1/gakumas"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/gameaccountlink"
	"momoko/internal/data/ent/gen/passportuser"
	"momoko/pkg/common"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
	"momoko/pkg/oidcserver"
)

// PassportAdminUsecase 用户通行证的后台管理（主 server 上的 PassportAdminService，权限 game:passport）：
// 通行证用户、替玩家绑定 / 解绑、邀请码、OIDC 应用与设置。与玩家端的 PassportUsecase 互不调用；
// 两边读写同一组通行证表，绑定、解绑与消息按同样的规则（passportCreateLink、passportUnlink、passportNotify）。
type PassportAdminUsecase struct {
	servers  *GameServerUsecase
	repo     PassportRepo
	users    UserRepo
	config   ConfigRepo
	configUc *ConfigUsecase
	log      *log.Helper

	// bindLocks 串行化同一台服务器上的后台替他绑定。
	bindLocks sync.Map
}

func NewPassportAdminUsecase(servers *GameServerUsecase, repo PassportRepo, users UserRepo, config ConfigRepo, configUc *ConfigUsecase, logger log.Logger) *PassportAdminUsecase {
	return &PassportAdminUsecase{
		servers:  servers,
		repo:     repo,
		users:    users,
		config:   config,
		configUc: configUc,
		log:      log.NewHelper(log.With(logger, "module", "biz/passport_admin")),
	}
}

// userLink 某个通行证用户的一条有效绑定。
func (uc *PassportAdminUsecase) userLink(ctx context.Context, userID, linkID string) (*gen.GameAccountLink, error) {
	l, err := uc.repo.GetLink(ctx, linkID)
	if gen.IsNotFound(err) || (err == nil && (l.PassportUserID != userID || l.State != gameaccountlink.StateActive)) {
		return nil, ErrPassportLinkNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return l, nil
}

// bind 替玩家写一条绑定（同一台服务器上的后台绑定串行执行）。
func (uc *PassportAdminUsecase) bind(ctx context.Context, u *gen.PassportUser, s *gen.GameServer, p *gk.Player) (*gen.GameAccountLink, error) {
	l, _ := uc.bindLocks.LoadOrStore(s.ID, &sync.Mutex{})
	mu := l.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	return passportCreateLink(ctx, uc.repo, u, s, p, PassportBindAdmin)
}

// ---------------- 后台：统计与列表 ----------------

// AdminStats 统计。
func (uc *PassportAdminUsecase) AdminStats(ctx context.Context) (*v1.GetPassportStatsResponse, error) {
	st, err := uc.repo.Stats(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.GetPassportStatsResponse{
		PlayerCount:        int32(st.Users),
		LinkedAccountCount: int32(st.LinkedAccounts),
		ServerCount:        int32(st.Servers),
		GameCount:          int32(len(st.Games)),
		NewThisWeek:        int32(st.NewThisWeek),
		NewThisWeekLinked:  int32(st.NewThisWeekLinked),
		ActiveThisWeek:     int32(st.ActiveThisWeek),
		DisabledCount:      int32(st.Disabled),
		DeletingCount:      int32(st.Deleting),
		Games:              st.Games,
	}, nil
}

func adminGameOf(game string, servers map[string]*gen.GameServer, serverID string) *v1.PassportAdminGame {
	g := &v1.PassportAdminGame{Id: game}
	if s, ok := servers[serverID]; ok {
		g.Name, g.Icon = gameNameOf(s)
	}
	return g
}

func (uc *PassportAdminUsecase) adminUser(u *gen.PassportUser, links []*gen.GameAccountLink, servers map[string]*gen.GameServer) *v1.PassportAdminUser {
	out := &v1.PassportAdminUser{
		Id:                  u.ID,
		Username:            u.Username,
		Name:                u.Name,
		EmailMasked:         maskEmail(u.Email),
		EmailVerified:       u.EmailVerified,
		Avatar:              u.Avatar,
		AvatarAsset:         u.AvatarAsset,
		Status:              string(u.Status),
		CreateTime:          timestamppb.New(u.CreateTime),
		LastLoginAt:         tsOrNil(u.LastLoginAt),
		DeletionScheduledAt: tsOrNil(u.DeletionScheduledAt),
		TotpEnabled:         u.TotpSecret != "" && u.TotpEnabledAt != nil,
		LinkCount:           int32(len(links)),
	}
	seen := map[string]bool{}
	for _, l := range links {
		if seen[l.Game] {
			continue
		}
		seen[l.Game] = true
		out.GameCount++
		if len(out.Games) < 3 {
			out.Games = append(out.Games, adminGameOf(l.Game, servers, l.ServerID))
		}
	}
	if len(links) == 1 {
		l := links[0]
		b := &v1.PassportAdminLinkBrief{ServerId: l.ServerID, Game: l.Game, Nickname: l.Nickname, DisplayId: l.DisplayID}
		if s, ok := servers[l.ServerID]; ok {
			b.ServerName = s.Name
			b.GameName, b.Icon = gameNameOf(s)
		}
		out.FirstLink = b
	}
	return out
}

// AdminListUsers 通行证列表。
func (uc *PassportAdminUsecase) AdminListUsers(ctx context.Context, req *v1.ListPassportUsersRequest) (*v1.ListPassportUsersResponse, error) {
	page, size := pageOf(req.Page, req.PageSize, 20)
	f := PassportUserFilter{Status: req.Status, Q: strings.TrimSpace(req.Q), ServerID: req.ServerId, Offset: (page - 1) * size, Limit: size}
	users, total, err := uc.repo.ListUsers(ctx, f)
	if err != nil {
		return nil, ErrSystem(err)
	}
	counts, err := uc.repo.UserCounts(ctx, f)
	if err != nil {
		return nil, ErrSystem(err)
	}
	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	links, err := uc.repo.LinksOfUsers(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	servers := passportServerMap(ctx, uc.servers)
	out := &v1.ListPassportUsersResponse{Total: int32(total), Counts: counts}
	for _, u := range users {
		ls := links[u.ID]
		sortLinks(ls)
		out.Items = append(out.Items, uc.adminUser(u, ls, servers))
	}
	if s, ok := servers[req.ServerId]; ok {
		out.Server = uc.servers.Brief(ctx, s)
	}
	return out, nil
}

// AdminGetUser 通行证详情。
func (uc *PassportAdminUsecase) AdminGetUser(ctx context.Context, id string) (*v1.GetPassportUserResponse, error) {
	u, err := uc.repo.GetUser(ctx, id)
	if gen.IsNotFound(err) {
		return nil, ErrPassportUserNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	links, err := uc.repo.ListLinks(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	sortLinks(links)
	servers, ss := passportServerMap(ctx, uc.servers), passportSelfServiceCounts(ctx, uc.repo)
	out := &v1.GetPassportUserResponse{
		User:            uc.adminUser(u, links, servers),
		Bio:             u.Bio,
		Language:        u.Language,
		Timezone:        u.Timezone,
		LastLoginIp:     u.LastLoginIP,
		LastLoginDevice: u.LastLoginDevice,
		RegisterIp:      u.RegisterIP,
		DisabledAt:      tsOrNil(u.DisabledAt),
		DisabledByName:  u.DisabledByName,
	}
	if u.InviteID != "" {
		if inv, err := uc.repo.GetInvite(ctx, u.InviteID); err == nil {
			out.InviteCode = inv.Code
		}
	}
	if n, err := uc.repo.CountSessions(ctx, id); err == nil {
		out.DeviceCount = int32(n)
	}
	if n, err := uc.repo.CountGrants(ctx, id); err == nil {
		out.AppCount = int32(n)
	}
	for _, l := range links {
		p := passportLinkProto(ctx, uc.servers, l, servers, ss)
		out.Links = append(out.Links, &v1.PassportAdminLink{
			Id:                 p.Id,
			Server:             p.Server,
			PlayerId:           p.PlayerId,
			DisplayId:          p.DisplayId,
			Account:            p.Account,
			Nickname:           p.Nickname,
			AvatarAsset:        p.AvatarAsset,
			Level:              p.Level,
			Method:             p.Method,
			IsPrimary:          p.IsPrimary,
			LinkedAt:           p.LinkedAt,
			LastSeenAt:         p.LastSeenAt,
			SelfServiceEnabled: p.SelfServiceEnabled,
		})
	}
	return out, nil
}

// AdminSetStatus 停用 / 启用（停用时全部设备下线）。
func (uc *PassportAdminUsecase) AdminSetStatus(ctx context.Context, adminName, id string, disabled bool) (*v1.GetPassportUserResponse, error) {
	u, err := uc.repo.GetUser(ctx, id)
	if gen.IsNotFound(err) {
		return nil, ErrPassportUserNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	if disabled && u.Status != passportuser.StatusDisabled {
		now := time.Now()
		u.Status, u.DisabledAt, u.DisabledByName = passportuser.StatusDisabled, &now, adminName
		if _, err := uc.repo.SaveUser(ctx, u); err != nil {
			return nil, ErrSystem(err)
		}
		if err := uc.repo.DeleteSessions(ctx, u.ID, ""); err != nil {
			return nil, ErrSystem(err)
		}
	} else if !disabled && u.Status == passportuser.StatusDisabled {
		u.Status, u.DisabledAt, u.DisabledByName = passportuser.StatusActive, nil, ""
		if _, err := uc.repo.SaveUser(ctx, u); err != nil {
			return nil, ErrSystem(err)
		}
		passportNotify(ctx, uc.repo, uc.log, u.ID, PassportMsgAccountEnabled, "", nil)
	}
	return uc.AdminGetUser(ctx, id)
}

// AdminUnbind 替玩家解绑。
func (uc *PassportAdminUsecase) AdminUnbind(ctx context.Context, adminName, userID, linkID string) (*v1.GetPassportUserResponse, error) {
	l, err := uc.userLink(ctx, userID, linkID)
	if err != nil {
		return nil, err
	}
	if err := passportUnlink(ctx, uc.repo, l, "admin"); err != nil {
		return nil, err
	}
	passportNotify(ctx, uc.repo, uc.log, userID, PassportMsgLinkUnboundAdmin, l.ServerID, map[string]string{"nickname": l.Nickname, "admin": adminName})
	return uc.AdminGetUser(ctx, userID)
}

// AdminBind 替玩家绑定一个游戏账号（不受解绑冷却限制，仍是一个服务器一个账号、一个账号一个通行证）。
func (uc *PassportAdminUsecase) AdminBind(ctx context.Context, adminName string, req *v1.AdminBindPassportLinkRequest) (*v1.GetPassportUserResponse, error) {
	u, err := uc.repo.GetUser(ctx, req.UserId)
	if gen.IsNotFound(err) {
		return nil, ErrPassportUserNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	s, client, err := uc.servers.ClientOf(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	if s.Game != gakumas.ID {
		return nil, ErrGmGameUnsupported
	}
	detail, err := gakumas.New(client).Player(ctx, strings.TrimSpace(req.Pid))
	if err != nil {
		if mgp.CodeOf(err) == gakumas.PlayerNotFound {
			return nil, ErrPassportPlayerNotFound
		}
		return nil, GameMGPError(err)
	}
	if detail.GetPlayer().GetPid() == "" {
		return nil, ErrPassportPlayerNotFound
	}
	l, err := uc.bind(ctx, u, s, detail.GetPlayer())
	if err != nil {
		return nil, err
	}
	passportNotify(ctx, uc.repo, uc.log, u.ID, PassportMsgLinkBoundByAdmin, s.ID, map[string]string{"nickname": l.Nickname, "admin": adminName})
	return uc.AdminGetUser(ctx, u.ID)
}

// AdminSearchServers 搜索服务器（筛选与替他绑定用）。
func (uc *PassportAdminUsecase) AdminSearchServers(ctx context.Context, adminID string, req *v1.SearchGameServersRequest) (*v1.SearchGameServersResponse, error) {
	return uc.servers.Search(ctx, adminID, GameRecentScopeAdmin, req.Q, req.Game, int(req.Limit), nil)
}

// AdminSearchPlayers 替他绑定时搜索游戏玩家（最多 20 条，标出已绑定的）。
func (uc *PassportAdminUsecase) AdminSearchPlayers(ctx context.Context, serverID, q string) (*v1.SearchPassportGamePlayersResponse, error) {
	s, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if s.Game != gakumas.ID {
		return nil, ErrGmGameUnsupported
	}
	page, err := gakumas.New(client).Players(ctx, gakumas.PlayerFilter{Q: strings.TrimSpace(q)}, "", 20)
	if err != nil {
		return nil, GameMGPError(err)
	}
	players := page.GetItems()
	pids := make([]string, 0, len(players))
	for _, p := range players {
		pids = append(pids, p.GetPid())
	}
	links, err := uc.repo.LinksByPlayers(ctx, s.ID, pids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	byPID := map[string]*gen.GameAccountLink{}
	for _, l := range links {
		byPID[l.PlayerID] = l
	}
	out := &v1.SearchPassportGamePlayersResponse{}
	for _, p := range players {
		item := &v1.PassportGamePlayer{
			Pid:              p.GetPid(),
			DisplayId:        p.GetDisplayId(),
			Account:          p.GetAccount(),
			Name:             p.GetName(),
			Level:            p.GetLevel(),
			AvatarIdolCardId: p.GetAvatarIdolCardId(),
		}
		if l, ok := byPID[p.GetPid()]; ok {
			item.LinkedUserId = l.PassportUserID
			if lu, err := uc.repo.GetUser(ctx, l.PassportUserID); err == nil {
				item.LinkedUsername = lu.Username
			}
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// ---------------- 后台：邀请码 ----------------

func inviteState(i *gen.PassportInvite) string {
	switch {
	case i.Revoked:
		return "revoked"
	case i.ExpiresAt != nil && time.Now().After(*i.ExpiresAt):
		return "expired"
	case i.Used >= i.MaxUses:
		return "used_up"
	default:
		return "active"
	}
}

func (uc *PassportAdminUsecase) inviteProto(ctx context.Context, i *gen.PassportInvite, servers map[string]*gen.GameServer) *v1.PassportInvite {
	out := &v1.PassportInvite{
		Id:            i.ID,
		Code:          i.Code,
		Email:         i.Email,
		MaxUses:       int32(i.MaxUses),
		Used:          int32(i.Used),
		ExpiresAt:     tsOrNil(i.ExpiresAt),
		CreatedByName: i.CreatedByName,
		Revoked:       i.Revoked,
		Note:          i.Note,
		CreateTime:    timestamppb.New(i.CreateTime),
		State:         inviteState(i),
	}
	if s, ok := servers[i.ServerID]; ok {
		out.Server = uc.servers.Brief(ctx, s)
	}
	return out
}

// AdminInvites 邀请码列表。
func (uc *PassportAdminUsecase) AdminInvites(ctx context.Context, req *v1.ListPassportInvitesRequest) (*v1.ListPassportInvitesResponse, error) {
	page, size := pageOf(req.Page, req.PageSize, 20)
	list, total, err := uc.repo.ListInvites(ctx, (page-1)*size, size)
	if err != nil {
		return nil, ErrSystem(err)
	}
	servers := passportServerMap(ctx, uc.servers)
	out := &v1.ListPassportInvitesResponse{Total: int32(total)}
	for _, i := range list {
		out.Items = append(out.Items, uc.inviteProto(ctx, i, servers))
	}
	return out, nil
}

const inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newInviteCode() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	var b strings.Builder
	for i, c := range buf {
		if i == 5 {
			b.WriteByte('-')
		}
		b.WriteByte(inviteAlphabet[int(c)%len(inviteAlphabet)])
	}
	return b.String(), nil
}

// AdminCreateInvite 新建邀请码。
func (uc *PassportAdminUsecase) AdminCreateInvite(ctx context.Context, adminID, adminName string, req *v1.CreatePassportInviteRequest) (*v1.PassportInvite, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		var err error
		if code, err = newInviteCode(); err != nil {
			return nil, ErrSystem(err)
		}
	} else if len(code) < 4 || len(code) > 32 || strings.ContainsAny(code, " \t\r\n") {
		return nil, ErrPassportInvalidArgument
	}
	if _, err := uc.repo.GetInviteByCode(ctx, code); err == nil {
		return nil, ErrPassportInviteCodeTaken
	}
	email := strings.TrimSpace(req.Email)
	if email != "" {
		var err error
		if email, err = normalizeEmail(email); err != nil {
			return nil, err
		}
	}
	maxUses := int(req.MaxUses)
	if maxUses <= 0 {
		maxUses = 1
	}
	if maxUses > 100000 || req.ExpiresDays < 0 || req.ExpiresDays > 3650 || len([]rune(req.Note)) > 200 {
		return nil, ErrPassportInvalidArgument
	}
	if req.ServerId != "" {
		if _, err := uc.servers.Get(ctx, req.ServerId); err != nil {
			return nil, err
		}
	}
	inv := &gen.PassportInvite{
		ID:            uuid.NewString(),
		Code:          code,
		Email:         email,
		ServerID:      req.ServerId,
		MaxUses:       maxUses,
		CreatedBy:     adminID,
		CreatedByName: adminName,
		Note:          strings.TrimSpace(req.Note),
	}
	if req.ExpiresDays > 0 {
		at := time.Now().Add(time.Duration(req.ExpiresDays) * 24 * time.Hour)
		inv.ExpiresAt = &at
	}
	created, err := uc.repo.CreateInvite(ctx, inv)
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, ErrPassportInviteCodeTaken
		}
		return nil, ErrSystem(err)
	}
	return uc.inviteProto(ctx, created, passportServerMap(ctx, uc.servers)), nil
}

// AdminRevokeInvite 作废邀请码。
func (uc *PassportAdminUsecase) AdminRevokeInvite(ctx context.Context, id string) (*v1.PassportInvite, error) {
	inv, err := uc.repo.RevokeInvite(ctx, id)
	if gen.IsNotFound(err) {
		return nil, ErrPassportInviteNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return uc.inviteProto(ctx, inv, passportServerMap(ctx, uc.servers)), nil
}

// ---------------- 后台：设置 ----------------

// PassportServerInfo 通行证 server 的配置（来自配置文件，service 层传入）。
type PassportServerInfo struct {
	Enabled   bool
	Addr      string
	PublicURL string
	// Issuer OIDC issuer（public_url 为空时按请求推断）
	Issuer string
}

// AdminSettings 通行证设置与只读的运行信息。
func (uc *PassportAdminUsecase) AdminSettings(ctx context.Context, info PassportServerInfo) (*v1.GetPassportSettingsResponse, error) {
	out := &v1.GetPassportSettingsResponse{
		Settings:            passportSettings(ctx, uc.config, uc.log),
		EmailConfigured:     passportEmailEnabled(ctx, uc.config),
		ServerEnabled:       info.Enabled,
		ListenAddr:          info.Addr,
		PublicUrl:           info.PublicURL,
		AvatarUploadEnabled: uc.configUc.AvatarUploadEnabled(ctx),
	}
	if ep, err := OidcEndpoints(info.Issuer); err == nil && info.Issuer != "" {
		out.OidcIssuer, out.OidcDiscoveryUrl = ep.IssuerURL, ep.DiscoveryURL
	}
	if list, err := uc.repo.ListOidcClients(ctx); err == nil {
		out.OidcClientCount = int32(len(list))
	}
	return out, nil
}

// AdminName 当前管理员的显示名称（审计快照用）。
func (uc *PassportAdminUsecase) AdminName(ctx context.Context, adminID string) string {
	u, err := uc.users.FindByID(ctx, adminID)
	if err != nil {
		return ""
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

// SaveSettings 保存通行证设置。
func (uc *PassportAdminUsecase) SaveSettings(ctx context.Context, s *v1.PassportSettings) (*v1.PassportSettings, error) {
	switch s.RegisterMode {
	case PassportRegisterOpen, PassportRegisterInviteOnly, PassportRegisterClosed:
	default:
		return nil, ErrPassportSettingsInvalid
	}
	if s.SessionDays < 1 || s.SessionDays > 365 || len([]rune(strings.TrimSpace(s.SiteName))) > 40 {
		return nil, ErrPassportSettingsInvalid
	}
	normalizePassportSettings(s)
	raw, err := protojson.Marshal(s)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if err := uc.config.BatchUpdate(ctx, map[common.ConfigKey]string{common.ConfigPassportSettings: string(raw)}); err != nil {
		return nil, ErrSystem(err)
	}
	return s, nil
}

// ---------------- 后台：OIDC 应用 ----------------

func (uc *PassportAdminUsecase) oidcClientProto(ctx context.Context, c *gen.PassportOidcClient, withSecret bool, grants map[string]int) *v1.PassportOidcClient {
	out := &v1.PassportOidcClient{
		Id:           c.ID,
		Name:         c.Name,
		ClientId:     c.ClientID,
		ClientSecret: oidcserver.MaskSecret(c.ClientSecret),
		RedirectUris: c.RedirectUris,
		Scopes:       c.Scopes,
		Active:       c.Active,
		Description:  c.Description,
		LastUsedAt:   tsOrNil(c.LastUsedAt),
		CreateTime:   timestamppb.New(c.CreateTime),
		GrantCount:   int32(grants[c.ID]),
	}
	if withSecret {
		out.ClientSecret = c.ClientSecret
	}
	if c.ServerID != "" {
		if s, err := uc.servers.Get(ctx, c.ServerID); err == nil {
			out.Server = uc.servers.Brief(ctx, s)
		}
	}
	return out
}

// ListOidcClients OIDC 应用列表。
func (uc *PassportAdminUsecase) ListOidcClients(ctx context.Context) (*v1.ListPassportOidcClientsResponse, error) {
	list, err := uc.repo.ListOidcClients(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	grants, _ := uc.repo.GrantCounts(ctx)
	out := &v1.ListPassportOidcClientsResponse{}
	for _, c := range list {
		out.Items = append(out.Items, uc.oidcClientProto(ctx, c, false, grants))
	}
	return out, nil
}

func validateOidcClient(req *v1.SavePassportOidcClientRequest) error {
	if strings.TrimSpace(req.Name) == "" || len([]rune(strings.TrimSpace(req.Name))) > 60 {
		return ErrPassportOidcClientName
	}
	if !oidcserver.ValidRedirectURIs(normalizeURIList(req.RedirectUris)) {
		return ErrPassportOidcRedirectURIs
	}
	return nil
}

// SaveOidcClient 创建（id 为空）或修改 OIDC 应用；创建时返回完整的 Client Secret。
func (uc *PassportAdminUsecase) SaveOidcClient(ctx context.Context, adminName string, req *v1.SavePassportOidcClientRequest) (*v1.PassportOidcClient, error) {
	if err := validateOidcClient(req); err != nil {
		return nil, err
	}
	if req.ServerId != "" {
		if _, err := uc.servers.Get(ctx, req.ServerId); err != nil {
			return nil, err
		}
	}
	row := &gen.PassportOidcClient{
		ID:           req.Id,
		Name:         strings.TrimSpace(req.Name),
		RedirectUris: normalizeURIList(req.RedirectUris),
		Scopes:       normalizeScopeList(req.Scopes),
		Active:       req.Active,
		Description:  strings.TrimSpace(req.Description),
		ServerID:     req.ServerId,
	}
	if req.Id == "" {
		clientID, err := oidcserver.GenerateClientID()
		if err != nil {
			return nil, ErrSystem(err)
		}
		secret, err := oidcserver.GenerateClientSecret()
		if err != nil {
			return nil, ErrSystem(err)
		}
		row.ID, row.ClientID, row.ClientSecret, row.CreatedByName = uuid.NewString(), "passport_"+strings.TrimPrefix(clientID, "momoko_"), secret, adminName
		created, err := uc.repo.CreateOidcClient(ctx, row)
		if err != nil {
			return nil, ErrSystem(err)
		}
		return uc.oidcClientProto(ctx, created, true, nil), nil
	}
	updated, err := uc.repo.UpdateOidcClient(ctx, row)
	if gen.IsNotFound(err) {
		return nil, ErrPassportOidcClientNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	grants, _ := uc.repo.GrantCounts(ctx)
	return uc.oidcClientProto(ctx, updated, false, grants), nil
}

// DeleteOidcClient 删除 OIDC 应用（玩家的授权一并删除）。
func (uc *PassportAdminUsecase) DeleteOidcClient(ctx context.Context, id string) error {
	if err := uc.repo.DeleteOidcClient(ctx, id); err != nil {
		if gen.IsNotFound(err) {
			return ErrPassportOidcClientNotFound
		}
		return ErrSystem(err)
	}
	return nil
}

// RefreshOidcClientSecret 重新生成 Client Secret。
func (uc *PassportAdminUsecase) RefreshOidcClientSecret(ctx context.Context, id string) (*v1.PassportOidcClient, error) {
	secret, err := oidcserver.GenerateClientSecret()
	if err != nil {
		return nil, ErrSystem(err)
	}
	c, err := uc.repo.UpdateOidcClientSecret(ctx, id, secret)
	if gen.IsNotFound(err) {
		return nil, ErrPassportOidcClientNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	grants, _ := uc.repo.GrantCounts(ctx)
	return uc.oidcClientProto(ctx, c, true, grants), nil
}
