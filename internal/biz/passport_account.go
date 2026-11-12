package biz

import (
	"context"
	"encoding/base64"
	"regexp"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/gmexecution"
	auth2 "momoko/pkg/auth"
	"momoko/pkg/passporttoken"
	"momoko/pkg/totp"
)

// avatarAssetPattern 游戏素材头像：<game>/<素材 key>，例如 gakumas/chara/hski。
var avatarAssetPattern = regexp.MustCompile(`^[a-z0-9_-]{1,32}/[a-z0-9_-]+(?:/[a-z0-9_.-]+){1,4}$`)

func tsOrNil(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

// user 读取当前用户（不存在视为登录失效）。
func (uc *PassportUsecase) user(ctx context.Context, userID string) (*gen.PassportUser, error) {
	u, err := uc.repo.GetUser(ctx, userID)
	if gen.IsNotFound(err) {
		return nil, ErrPassportUnauthorized
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return u, nil
}

// Me 当前账号：资料、安全状态与计数。
func (uc *PassportUsecase) Me(ctx context.Context, userID string) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	return uc.meOf(ctx, u), nil
}

func (uc *PassportUsecase) meOf(ctx context.Context, u *gen.PassportUser) *v1.PassportMe {
	uc.syncReviews(ctx, u)
	out := &v1.PassportMe{
		Id:                  u.ID,
		Username:            u.Username,
		Name:                u.Name,
		Email:               u.Email,
		EmailVerified:       u.EmailVerified,
		Bio:                 u.Bio,
		Avatar:              u.Avatar,
		AvatarAsset:         u.AvatarAsset,
		Language:            u.Language,
		Timezone:            u.Timezone,
		EmailCodeLogin:      u.EmailCodeLogin,
		TotpEnabled:         u.TotpSecret != "" && u.TotpEnabledAt != nil,
		TotpEnabledAt:       tsOrNil(u.TotpEnabledAt),
		RecoveryCodesLeft:   int32(len(u.RecoveryCodes)),
		PasswordUpdatedAt:   tsOrNil(u.PasswordUpdatedAt),
		CreateTime:          timestamppb.New(u.CreateTime),
		LastLoginAt:         tsOrNil(u.LastLoginAt),
		LastLoginIp:         u.LastLoginIP,
		LastLoginDevice:     u.LastLoginDevice,
		DeletionScheduledAt: tsOrNil(u.DeletionScheduledAt),
		AvatarUploadEnabled: uc.configUc.AvatarUploadEnabled(ctx),
		AvatarMaxSize:       AvatarMaxSize,
	}
	if n, err := uc.repo.CountSessions(ctx, u.ID); err == nil {
		out.DeviceCount = int32(n)
	}
	if n, err := uc.repo.CountGrants(ctx, u.ID); err == nil {
		out.AppCount = int32(n)
	}
	if links, err := uc.repo.ListLinks(ctx, u.ID); err == nil {
		games := map[string]struct{}{}
		for _, l := range links {
			games[l.Game] = struct{}{}
		}
		out.LinkCount, out.GameCount = int32(len(links)), int32(len(games))
	}
	if n, err := uc.repo.CountUnread(ctx, u.ID); err == nil {
		out.UnreadMessages = int32(n)
	}
	if logs, err := uc.repo.ListLoginLogs(ctx, u.ID, 1); err == nil && len(logs) > 0 {
		out.LastLoginMethod = logs[0].Method
	}
	return out
}

// UpdateProfile 修改资料。
func (uc *PassportUsecase) UpdateProfile(ctx context.Context, userID string, req *v1.UpdatePassportProfileRequest) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = u.Username
	}
	if len([]rune(name)) > 32 {
		return nil, ErrPassportNameTooLong
	}
	bio := strings.TrimSpace(req.Bio)
	if len([]rune(bio)) > 100 {
		return nil, ErrPassportBioTooLong
	}
	tz := strings.TrimSpace(req.Timezone)
	if tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			return nil, ErrPassportTimezoneInvalid
		}
	}
	asset := strings.TrimSpace(req.AvatarAsset)
	if asset != "" {
		game := asset[:max(strings.Index(asset, "/"), 0)]
		if _, ok := GameAdapterByID(game); !ok || !avatarAssetPattern.MatchString(asset) {
			return nil, ErrPassportAvatarAsset
		}
	}
	oldKey := ""
	if asset != "" || req.ClearAvatar {
		oldKey = u.AvatarKey
		u.Avatar, u.AvatarKey = "", ""
	}
	u.Name, u.Bio, u.Language, u.Timezone, u.AvatarAsset = name, bio, normalizeLanguage(req.Language), tz, asset
	if u, err = uc.repo.SaveUser(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	uc.configUc.deleteAvatarObject(ctx, oldKey)
	return uc.meOf(ctx, u), nil
}

// avatarOwner 头像上传凭证的申请人（加前缀，与管理后台用户的凭证不能互换）。
func avatarOwner(userID string) string { return "passport:" + userID }

// PresignAvatar 申请头像直传地址。
func (uc *PassportUsecase) PresignAvatar(ctx context.Context, userID, contentType string, size int64) (*v1.PresignPassportAvatarResponse, error) {
	if _, err := uc.user(ctx, userID); err != nil {
		return nil, err
	}
	url, token, err := uc.configUc.presignAvatar(ctx, avatarOwner(userID), contentType, size)
	if err != nil {
		return nil, err
	}
	return &v1.PresignPassportAvatarResponse{UploadUrl: url, Token: token}, nil
}

// CompleteAvatar 确认直传的头像（替换素材头像与旧头像）。
func (uc *PassportUsecase) CompleteAvatar(ctx context.Context, userID, token string) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	link, key, err := uc.configUc.completeAvatar(ctx, avatarOwner(userID), token)
	if err != nil {
		return nil, err
	}
	oldKey := u.AvatarKey
	u.Avatar, u.AvatarKey, u.AvatarAsset = link, key, ""
	if u, err = uc.repo.SaveUser(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	if oldKey != key {
		uc.configUc.deleteAvatarObject(ctx, oldKey)
	}
	return uc.meOf(ctx, u), nil
}

// ChangePassword 修改密码，其他设备下线。
func (uc *PassportUsecase) ChangePassword(ctx context.Context, c *passporttoken.Claims, oldPassword, newPassword string) error {
	u, err := uc.user(ctx, c.UserID)
	if err != nil {
		return err
	}
	if !auth2.VerifyPassword(oldPassword, u.Password) {
		return ErrPassportPasswordInvalid
	}
	if err := validPassportPassword(newPassword); err != nil {
		return err
	}
	hash, err := auth2.HashPassword(newPassword)
	if err != nil {
		return ErrSystem(err)
	}
	now := time.Now()
	u.Password, u.PasswordUpdatedAt = hash, &now
	if _, err := uc.repo.SaveUser(ctx, u); err != nil {
		return ErrSystem(err)
	}
	if err := uc.repo.DeleteSessions(ctx, u.ID, c.SessionID); err != nil {
		return ErrSystem(err)
	}
	client := PassportClientOf(ctx)
	uc.notify(ctx, u.ID, PassportMsgPasswordChanged, "", map[string]string{"ip": client.IP, "ip_scope": IPScope(client.IP)})
	return nil
}

// SendEmailChangeCode 发送更换邮箱的验证码到新邮箱。
func (uc *PassportUsecase) SendEmailChangeCode(ctx context.Context, userID, email string) (*v1.SendPassportEmailCodeResponse, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if other, err := uc.repo.FindUserByEmail(ctx, email); err == nil && other.ID != u.ID {
		return nil, ErrPassportEmailTaken
	} else if err != nil && !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	return uc.sendCode(ctx, passportCodeChange+":"+u.ID, email, v1.EmailTemplateType_EmailTemplateType_Register, map[string]string{"name": displayName(u)})
}

// ChangeEmail 用新邮箱收到的验证码与当前密码更换邮箱。
func (uc *PassportUsecase) ChangeEmail(ctx context.Context, userID string, req *v1.ChangePassportEmailRequest) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !auth2.VerifyPassword(req.Password, u.Password) {
		return nil, ErrPassportPasswordInvalid
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if other, err := uc.repo.FindUserByEmail(ctx, email); err == nil && other.ID != u.ID {
		return nil, ErrPassportEmailTaken
	} else if err != nil && !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	if err := uc.verifyCode(passportCodeChange+":"+u.ID, email, req.Code); err != nil {
		return nil, err
	}
	old := u.Email
	u.Email, u.EmailVerified = email, true
	if u, err = uc.repo.SaveUser(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	uc.notify(ctx, u.ID, PassportMsgEmailChanged, "", map[string]string{"old": maskEmail(old), "new": maskEmail(email)})
	return uc.meOf(ctx, u), nil
}

// SetEmailLogin 开关邮箱验证码登录。
func (uc *PassportUsecase) SetEmailLogin(ctx context.Context, userID string, enabled bool) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.EmailCodeLogin = enabled
	if u, err = uc.repo.SaveUser(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	return uc.meOf(ctx, u), nil
}

// ---------------- 两步验证 ----------------

// SetupTotp 生成新密钥（确认前不生效）。
func (uc *PassportUsecase) SetupTotp(ctx context.Context, userID string) (*v1.SetupPassportTotpResponse, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.TotpSecret != "" && u.TotpEnabledAt != nil {
		return nil, ErrPassportTotpEnabled
	}
	secret, err := totp.GenerateSecret()
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.totpPending.Set(u.ID, expiring[string]{V: secret, Exp: time.Now().Add(15 * time.Minute)})
	issuer := uc.SiteName(ctx, uc.Settings(ctx))
	if issuer == "" {
		issuer = "Momoko"
	}
	uri := totp.URI(issuer, u.Username, secret)
	png, err := totp.QRCodePNG(uri)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.SetupPassportTotpResponse{
		Secret: secret,
		Uri:    uri,
		QrCode: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	}, nil
}

func (uc *PassportUsecase) newRecoveryCodes() ([]string, []string, error) {
	codes, err := totp.GenerateRecoveryCodes(passportRecoveryCodes)
	if err != nil {
		return nil, nil, err
	}
	hashes := make([]string, 0, len(codes))
	for _, c := range codes {
		h, err := auth2.HashPassword(c)
		if err != nil {
			return nil, nil, err
		}
		hashes = append(hashes, h)
	}
	return codes, hashes, nil
}

// EnableTotp 用验证码确认开启，返回恢复码。
func (uc *PassportUsecase) EnableTotp(ctx context.Context, userID, code string) (*v1.PassportRecoveryCodes, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.TotpSecret != "" && u.TotpEnabledAt != nil {
		return nil, ErrPassportTotpEnabled
	}
	pending, ok := uc.totpPending.Get(u.ID)
	if !ok || !pending.alive() {
		return nil, ErrPassportTotpNotSetup
	}
	step, ok := totp.Verify(pending.V, code, time.Now(), 1)
	if !ok {
		return nil, ErrPassportMfaInvalid
	}
	enc, err := gameBox().Encrypt(pending.V)
	if err != nil {
		return nil, ErrSystem(err)
	}
	codes, hashes, err := uc.newRecoveryCodes()
	if err != nil {
		return nil, ErrSystem(err)
	}
	now := time.Now()
	u.TotpSecret, u.TotpEnabledAt, u.RecoveryCodes = enc, &now, hashes
	if _, err := uc.repo.SaveUser(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	uc.totpPending.Del(u.ID)
	uc.totpUsed.Set(u.ID, step)
	uc.notify(ctx, u.ID, PassportMsgTotpEnabled, "", nil)
	return &v1.PassportRecoveryCodes{Codes: codes}, nil
}

// DisableTotp 关闭两步验证（密码 + 验证码或恢复码）。
func (uc *PassportUsecase) DisableTotp(ctx context.Context, userID, password, code string) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.TotpSecret == "" || u.TotpEnabledAt == nil {
		return nil, ErrPassportTotpDisabled
	}
	if !auth2.VerifyPassword(password, u.Password) {
		return nil, ErrPassportPasswordInvalid
	}
	if err := uc.verifySecond(ctx, u, code); err != nil {
		return nil, err
	}
	u.TotpSecret, u.TotpEnabledAt, u.RecoveryCodes = "", nil, nil
	if u, err = uc.repo.SaveUser(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	uc.notify(ctx, u.ID, PassportMsgTotpDisabled, "", nil)
	return uc.meOf(ctx, u), nil
}

// verifySecond 校验输入的是 6 位验证码还是恢复码。
func (uc *PassportUsecase) verifySecond(ctx context.Context, u *gen.PassportUser, code string) error {
	code = strings.TrimSpace(code)
	if len(code) == 6 && strings.Trim(code, "0123456789") == "" {
		return uc.checkSecondFactor(ctx, u, code, "")
	}
	return uc.checkSecondFactor(ctx, u, "", code)
}

// RegenerateRecoveryCodes 重新生成恢复码。
func (uc *PassportUsecase) RegenerateRecoveryCodes(ctx context.Context, userID, code string) (*v1.PassportRecoveryCodes, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := uc.checkSecondFactor(ctx, u, code, ""); err != nil {
		return nil, err
	}
	codes, hashes, err := uc.newRecoveryCodes()
	if err != nil {
		return nil, ErrSystem(err)
	}
	u.RecoveryCodes = hashes
	if _, err := uc.repo.SaveUser(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.PassportRecoveryCodes{Codes: codes}, nil
}

// ---------------- 设备与应用 ----------------

// Devices 登录设备（最近活跃的在前）。
func (uc *PassportUsecase) Devices(ctx context.Context, c *passporttoken.Claims) (*v1.ListPassportDevicesResponse, error) {
	list, err := uc.repo.ListSessions(ctx, c.UserID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	clients := map[string]string{}
	if cs, err := uc.repo.ListOidcClients(ctx); err == nil {
		for _, cl := range cs {
			clients[cl.ClientID] = cl.Name
		}
	}
	out := &v1.ListPassportDevicesResponse{}
	for _, s := range list {
		if time.Now().After(s.ExpiresAt) {
			continue
		}
		lastIP := passportFirst(s.LastIP, s.IP)
		last := s.CreateTime
		if s.LastActiveAt != nil && s.LastActiveAt.After(last) {
			last = *s.LastActiveAt
		}
		out.Items = append(out.Items, &v1.PassportDevice{
			SessionId:      s.ID,
			Device:         s.Device,
			DeviceType:     DeviceType(s.UserAgent, s.Device),
			UserAgent:      s.UserAgent,
			Ip:             s.IP,
			LastIp:         lastIP,
			IpScope:        IPScope(lastIP),
			LoginMethod:    s.LoginMethod,
			AppName:        clients[s.ClientID],
			CreateTime:     timestamppb.New(s.CreateTime),
			LastActiveTime: timestamppb.New(last),
			ExpireTime:     timestamppb.New(s.ExpiresAt),
			Current:        s.ID == c.SessionID,
		})
	}
	sort.SliceStable(out.Items, func(i, j int) bool {
		if out.Items[i].Current != out.Items[j].Current {
			return out.Items[i].Current
		}
		return out.Items[i].LastActiveTime.AsTime().After(out.Items[j].LastActiveTime.AsTime())
	})
	return out, nil
}

// RevokeDevice 让一台设备下线。
func (uc *PassportUsecase) RevokeDevice(ctx context.Context, userID, sessionID string) error {
	err := uc.repo.DeleteSession(ctx, userID, sessionID)
	if gen.IsNotFound(err) {
		return ErrPassportSessionNotFound
	}
	if err != nil {
		return ErrSystem(err)
	}
	return nil
}

// LogoutOthers 退出其他所有设备。
func (uc *PassportUsecase) LogoutOthers(ctx context.Context, c *passporttoken.Claims) error {
	if err := uc.repo.DeleteSessions(ctx, c.UserID, c.SessionID); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// Apps 授权的应用。
func (uc *PassportUsecase) Apps(ctx context.Context, userID string) (*v1.ListPassportAppsResponse, error) {
	grants, err := uc.repo.ListGrants(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListPassportAppsResponse{}
	for _, g := range grants {
		cl, err := uc.repo.GetOidcClient(ctx, g.ClientID)
		if err != nil {
			continue
		}
		app := &v1.PassportApp{
			Id:          g.ID,
			Name:        cl.Name,
			Description: cl.Description,
			Scopes:      g.Scopes,
			CreateTime:  timestamppb.New(g.CreateTime),
			LastUsedAt:  timestamppb.New(g.LastUsedAt),
		}
		if cl.ServerID != "" {
			if s, err := uc.Servers().Get(ctx, cl.ServerID); err == nil {
				app.Server = uc.Servers().Brief(ctx, s)
			}
		}
		out.Items = append(out.Items, app)
	}
	return out, nil
}

// RevokeApp 撤销授权。
func (uc *PassportUsecase) RevokeApp(ctx context.Context, userID, id string) error {
	err := uc.repo.DeleteGrant(ctx, userID, id)
	if gen.IsNotFound(err) {
		return ErrPassportAppNotFound
	}
	if err != nil {
		return ErrSystem(err)
	}
	return nil
}

// ---------------- 注销 ----------------

// RequestDeletion 申请注销（7 天后删除）。
func (uc *PassportUsecase) RequestDeletion(ctx context.Context, userID, password string) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !auth2.VerifyPassword(password, u.Password) {
		return nil, ErrPassportPasswordInvalid
	}
	if u.DeletionScheduledAt == nil {
		at := time.Now().Add(PassportDeletionGrace)
		u.DeletionScheduledAt = &at
		if u, err = uc.repo.SaveUser(ctx, u); err != nil {
			return nil, ErrSystem(err)
		}
		uc.notify(ctx, u.ID, PassportMsgDeletionScheduled, "", map[string]string{"at": at.UTC().Format(time.RFC3339)})
	}
	return uc.meOf(ctx, u), nil
}

// CancelDeletion 撤销注销申请。
func (uc *PassportUsecase) CancelDeletion(ctx context.Context, userID string) (*v1.PassportMe, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.DeletionScheduledAt != nil {
		u.DeletionScheduledAt = nil
		if u, err = uc.repo.SaveUser(ctx, u); err != nil {
			return nil, ErrSystem(err)
		}
		uc.notify(ctx, u.ID, PassportMsgDeletionCancelled, "", nil)
	}
	return uc.meOf(ctx, u), nil
}

// ---------------- 消息 ----------------

// syncReviews 为被管理员处理过的自助申请生成消息（审批在 GM 运营里完成，这里按审批时间增量同步）。
func (uc *PassportUsecase) syncReviews(ctx context.Context, u *gen.PassportUser) {
	since := u.CreateTime
	if u.ReviewsSyncedAt != nil {
		since = *u.ReviewsSyncedAt
	}
	list, err := uc.repo.ReviewedSince(ctx, u.ID, since)
	if err != nil || len(list) == 0 {
		return
	}
	latest := since
	msgs := make([]*gen.PassportMessage, 0, len(list))
	for _, e := range list {
		if e.ReviewedAt == nil {
			continue
		}
		if e.ReviewedAt.After(latest) {
			latest = *e.ReviewedAt
		}
		kind := PassportMsgRequestApproved
		if e.Status == gmexecution.StatusRejected {
			kind = PassportMsgRequestRejected
		}
		msgs = append(msgs, &gen.PassportMessage{
			ID:       NewMessageID(),
			UserID:   u.ID,
			Kind:     kind,
			ServerID: e.ServerID,
			Params: map[string]string{
				"command": e.Command,
				"title":   e.CommandTitle,
				"status":  string(e.Status),
				"note":    e.ReviewNote,
			},
		})
	}
	if len(msgs) > 0 {
		if err := uc.repo.AddMessages(ctx, msgs); err != nil {
			uc.log.Warnf("sync passport review messages: %v", err)
			return
		}
	}
	u.ReviewsSyncedAt = &latest
	if saved, err := uc.repo.SaveUser(ctx, u); err == nil {
		*u = *saved
	}
}

// Messages 消息列表。
func (uc *PassportUsecase) Messages(ctx context.Context, userID string, req *v1.ListPassportMessagesRequest) (*v1.ListPassportMessagesResponse, error) {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	uc.syncReviews(ctx, u)
	page, size := int(req.Page), int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	list, total, err := uc.repo.ListMessages(ctx, userID, req.UnreadOnly, (page-1)*size, size)
	if err != nil {
		return nil, ErrSystem(err)
	}
	servers := uc.serverMap(ctx)
	out := &v1.ListPassportMessagesResponse{Total: int32(total)}
	for _, m := range list {
		params := m.Params
		if params == nil {
			params = map[string]string{}
		}
		item := &v1.PassportMessage{
			Id:         m.ID,
			Kind:       m.Kind,
			Params:     params,
			Read:       m.ReadAt != nil,
			CreateTime: timestamppb.New(m.CreateTime),
		}
		if s, ok := servers[m.ServerID]; ok {
			item.Server = uc.Servers().Brief(ctx, s)
		}
		out.Items = append(out.Items, item)
	}
	if n, err := uc.repo.CountUnread(ctx, userID); err == nil {
		out.Unread = int32(n)
	}
	return out, nil
}

// ReadMessages 标记已读。
func (uc *PassportUsecase) ReadMessages(ctx context.Context, userID string, ids []string, all bool) (*v1.ReadPassportMessagesResponse, error) {
	if all || len(ids) > 0 {
		if err := uc.repo.ReadMessages(ctx, userID, ids, all); err != nil {
			return nil, ErrSystem(err)
		}
	}
	n, err := uc.repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.ReadPassportMessagesResponse{Unread: int32(n)}, nil
}

// serverMap 全部服务器（几百台以内）。
func (uc *PassportUsecase) serverMap(ctx context.Context) map[string]*gen.GameServer {
	return passportServerMap(ctx, uc.servers)
}

// passportServerMap 全部服务器（几百台以内），按 id。
func passportServerMap(ctx context.Context, gs *GameServerUsecase) map[string]*gen.GameServer {
	out := map[string]*gen.GameServer{}
	if list, err := gs.List(ctx); err == nil {
		for _, s := range list {
			out[s.ID] = s
		}
	}
	return out
}

// ---------------- 最近活动 ----------------

// Activity 最近活动：登录、绑定与解绑、自助操作、兑换码，按时间从新到旧。
func (uc *PassportUsecase) Activity(ctx context.Context, userID string, limit int) (*v1.ListPassportActivityResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	servers := uc.serverMap(ctx)
	var items []*v1.PassportActivity
	add := func(kind string, at time.Time, serverID string, params map[string]string) {
		a := &v1.PassportActivity{Kind: kind, Time: timestamppb.New(at), Params: params}
		if s, ok := servers[serverID]; ok {
			a.Server = uc.Servers().Brief(ctx, s)
		}
		items = append(items, a)
	}
	if logs, err := uc.repo.ListLoginLogs(ctx, userID, limit); err == nil {
		for _, l := range logs {
			add("login", l.CreateTime, "", map[string]string{
				"method":      l.Method,
				"device":      l.Device,
				"device_type": DeviceType(l.UserAgent, l.Device),
				"platform":    Platform(l.UserAgent),
				"ip":          l.IP,
			})
		}
	}
	if links, err := uc.repo.ListLinks(ctx, userID); err == nil {
		for _, l := range links {
			add("bind", l.LinkedAt, l.ServerID, map[string]string{"nickname": l.Nickname, "method": string(l.Method)})
		}
	}
	if unlinks, err := uc.repo.ListUnlinks(ctx, userID, limit); err == nil {
		for _, l := range unlinks {
			add("unbind", l.UnlinkedAt, l.ServerID, map[string]string{"nickname": l.Nickname, "by": string(l.By)})
		}
	}
	if execs, _, err := uc.repo.ListPlayerExecutions(ctx, PassportExecFilter{UserID: userID, Limit: limit}); err == nil {
		for _, e := range execs {
			add("self_service", e.CreateTime, e.ServerID, map[string]string{"command": e.Command, "title": e.CommandTitle, "status": string(e.Status)})
		}
	}
	if redeems, _, err := uc.repo.ListRedeems(ctx, userID, "", 0, limit); err == nil {
		for _, r := range redeems {
			add("redeem", r.CreateTime, r.ServerID, map[string]string{"code": r.Code, "result": string(r.Result)})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Time.AsTime().After(items[j].Time.AsTime()) })
	if len(items) > limit {
		items = items[:limit]
	}
	return &v1.ListPassportActivityResponse{Items: items}, nil
}
