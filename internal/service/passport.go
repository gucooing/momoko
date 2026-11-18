package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/conf"
	"momoko/pkg/passporttoken"
	"momoko/pkg/response"
)

// PassportService 用户通行证玩家端（只注册在 passport server 上）。
type PassportService struct {
	v1.UnimplementedPassportServiceServer

	uc      *biz.PassportUsecase
	gakumas *biz.GakumasPassportUsecase
	conf    *conf.Passport
}

func NewPassportService(uc *biz.PassportUsecase, gakumas *biz.GakumasPassportUsecase, c *conf.Passport) *PassportService {
	return &PassportService{
		uc:      uc,
		gakumas: gakumas,
		conf:    c,
	}
}

// publicURL 通行证的对外地址（配置文件 passport.public_url，去掉末尾的 /；没填时为空）。
func (s *PassportService) publicURL() string {
	if s.conf == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(s.conf.PublicUrl), "/")
}

// Usecase 通行证用例（passport server 的鉴权过滤器用）。
func (s *PassportService) Usecase() *biz.PassportUsecase { return s.uc }

// me 当前登录的通行证身份（鉴权过滤器放进上下文）。
func me(ctx context.Context) (*passporttoken.Claims, error) {
	c, ok := passporttoken.FromContext(ctx)
	if !ok {
		return nil, biz.ErrPassportUnauthorized
	}
	return c, nil
}

// ---------------- 公开 ----------------

func (s *PassportService) GetPassportConfig(ctx context.Context, _ *v1.GetPassportConfigRequest) (*v1.GetPassportConfigResponse, error) {
	return s.uc.PublicConfig(ctx), nil
}

func (s *PassportService) SendPassportRegisterCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	return s.uc.SendRegisterCode(ctx, req.Email)
}

func (s *PassportService) PassportRegister(ctx context.Context, req *v1.PassportRegisterRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.Register(ctx, req)
}

func (s *PassportService) SendPassportLoginCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	return s.uc.SendLoginCode(ctx, req.Email)
}

func (s *PassportService) PassportLogin(ctx context.Context, req *v1.PassportLoginRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.Login(ctx, req)
}

func (s *PassportService) PassportLoginMfa(ctx context.Context, req *v1.PassportLoginMfaRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.LoginMfa(ctx, req)
}

func (s *PassportService) PassportRefresh(ctx context.Context, req *v1.PassportRefreshRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.Refresh(ctx, req.RefreshToken)
}

func (s *PassportService) SendPassportResetCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	return s.uc.SendResetCode(ctx, req.Email)
}

func (s *PassportService) PassportResetPassword(ctx context.Context, req *v1.PassportResetPasswordRequest) (*v1.PassportEmpty, error) {
	if err := s.uc.ResetPassword(ctx, req); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

// ---------------- 客户端浏览器授权 ----------------

func (s *PassportService) StartPassportClientAuth(ctx context.Context, req *v1.StartPassportClientAuthRequest) (*v1.StartPassportClientAuthResponse, error) {
	return s.uc.StartClientAuth(ctx, req, s.publicURL())
}

func (s *PassportService) ClaimPassportClientAuth(ctx context.Context, req *v1.ClaimPassportClientAuthRequest) (*v1.PassportClientToken, error) {
	return s.uc.ClaimClientAuth(ctx, req.DeviceCode)
}

func (s *PassportService) GetPassportClientAuth(ctx context.Context, req *v1.PassportClientAuthCodeRequest) (*v1.PassportClientAuth, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.GetClientAuth(ctx, c.UserID, req.UserCode)
}

func (s *PassportService) ApprovePassportClientAuth(ctx context.Context, req *v1.PassportClientAuthCodeRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.ApproveClientAuth(ctx, c.UserID, req.UserCode); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) DenyPassportClientAuth(ctx context.Context, req *v1.PassportClientAuthCodeRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.DenyClientAuth(ctx, c.UserID, req.UserCode); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

// ---------------- 账号 ----------------

func (s *PassportService) PassportLogout(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Logout(ctx, c); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) GetPassportMe(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Me(ctx, c.UserID)
}

func (s *PassportService) UpdatePassportProfile(ctx context.Context, req *v1.UpdatePassportProfileRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.UpdateProfile(ctx, c.UserID, req)
}

func (s *PassportService) PresignPassportAvatar(ctx context.Context, req *v1.PresignPassportAvatarRequest) (*v1.PresignPassportAvatarResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.PresignAvatar(ctx, c.UserID, req.ContentType, req.Size)
}

func (s *PassportService) CompletePassportAvatar(ctx context.Context, req *v1.CompletePassportAvatarRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.CompleteAvatar(ctx, c.UserID, req.Token)
}

func (s *PassportService) ChangePassportPassword(ctx context.Context, req *v1.ChangePassportPasswordRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.ChangePassword(ctx, c, req.OldPassword, req.NewPassword); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) SendPassportEmailChangeCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SendEmailChangeCode(ctx, c.UserID, req.Email)
}

func (s *PassportService) ChangePassportEmail(ctx context.Context, req *v1.ChangePassportEmailRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.ChangeEmail(ctx, c.UserID, req)
}

func (s *PassportService) SetPassportEmailLogin(ctx context.Context, req *v1.SetPassportEmailLoginRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SetEmailLogin(ctx, c.UserID, req.Enabled)
}

func (s *PassportService) SetupPassportTotp(ctx context.Context, _ *v1.PassportEmpty) (*v1.SetupPassportTotpResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SetupTotp(ctx, c.UserID)
}

func (s *PassportService) EnablePassportTotp(ctx context.Context, req *v1.EnablePassportTotpRequest) (*v1.PassportRecoveryCodes, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.EnableTotp(ctx, c.UserID, req.Code)
}

func (s *PassportService) DisablePassportTotp(ctx context.Context, req *v1.DisablePassportTotpRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.DisableTotp(ctx, c.UserID, req.Password, req.Code)
}

func (s *PassportService) RegeneratePassportRecoveryCodes(ctx context.Context, req *v1.PassportTotpCodeRequest) (*v1.PassportRecoveryCodes, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.RegenerateRecoveryCodes(ctx, c.UserID, req.Code)
}

func (s *PassportService) ListPassportDevices(ctx context.Context, _ *v1.PassportEmpty) (*v1.ListPassportDevicesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Devices(ctx, c)
}

func (s *PassportService) RevokePassportDevice(ctx context.Context, req *v1.RevokePassportDeviceRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.RevokeDevice(ctx, c.UserID, req.SessionId); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) LogoutPassportOthers(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.LogoutOthers(ctx, c); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) RequestPassportDeletion(ctx context.Context, req *v1.RequestPassportDeletionRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.RequestDeletion(ctx, c.UserID, req.Password)
}

func (s *PassportService) CancelPassportDeletion(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.CancelDeletion(ctx, c.UserID)
}

func (s *PassportService) ListPassportMessages(ctx context.Context, req *v1.ListPassportMessagesRequest) (*v1.ListPassportMessagesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Messages(ctx, c.UserID, req)
}

func (s *PassportService) ReadPassportMessages(ctx context.Context, req *v1.ReadPassportMessagesRequest) (*v1.ReadPassportMessagesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.ReadMessages(ctx, c.UserID, req.Ids, req.All)
}

func (s *PassportService) ListPassportActivity(ctx context.Context, req *v1.ListPassportActivityRequest) (*v1.ListPassportActivityResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Activity(ctx, c.UserID, int(req.Limit))
}

// ---------------- 游戏账号 ----------------

func (s *PassportService) ListPassportLinks(ctx context.Context, _ *v1.PassportEmpty) (*v1.ListPassportLinksResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Links(ctx, c.UserID)
}

func (s *PassportService) SetPassportPrimaryLink(ctx context.Context, req *v1.PassportLinkRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.SetPrimary(ctx, c.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) UnbindPassportLink(ctx context.Context, req *v1.PassportLinkRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Unbind(ctx, c.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) SearchPassportBindServers(ctx context.Context, req *v1.SearchPassportBindServersRequest) (*v1.SearchPassportBindServersResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SearchBindServers(ctx, c.UserID, req.Q, req.Game, strings.TrimSpace(req.ServerId))
}

func (s *PassportService) BindPassportAccount(ctx context.Context, req *v1.BindPassportAccountRequest) (*v1.PassportLink, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Bind(ctx, c.UserID, req)
}

// ---------------- 游戏空间 ----------------

func (s *PassportService) GetPassportGame(ctx context.Context, req *v1.PassportGameRequest) (*v1.PassportGameResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Game(ctx, c.UserID, req.ServerId, true)
}

func (s *PassportService) GetPassportGameBrief(ctx context.Context, req *v1.PassportGameRequest) (*v1.PassportGameResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Game(ctx, c.UserID, req.ServerId, false)
}

func (s *PassportService) ListPassportSelfService(ctx context.Context, req *v1.PassportGameRequest) (*v1.ListPassportSelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SelfService(ctx, c.UserID, req.ServerId)
}

func (s *PassportService) ListPassportGameRequests(ctx context.Context, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Requests(ctx, c.UserID, req)
}

func (s *PassportService) ListPassportGameRecords(ctx context.Context, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Records(ctx, c.UserID, req)
}

func (s *PassportService) RedeemPassportGiftCode(ctx context.Context, req *v1.RedeemPassportGiftCodeRequest) (*v1.RedeemPassportGiftCodeResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Redeem(ctx, c.UserID, req.ServerId, req.Code)
}

// ---------------- 启动器 ----------------

func (s *PassportService) ListPassportLaunchApps(ctx context.Context, _ *v1.PassportEmpty) (*v1.ListPassportLaunchAppsResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.LaunchApps(ctx, c.UserID)
}

func (s *PassportService) GetPassportLaunchApp(ctx context.Context, req *v1.PassportLaunchAppRequest) (*v1.PassportLaunchApp, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.LaunchApp(ctx, c.UserID, req.AppId)
}

func (s *PassportService) BindPassportLaunchApp(ctx context.Context, req *v1.BindPassportLaunchAppRequest) (*v1.PassportLink, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.BindLaunchApp(ctx, c.UserID, req)
}

func (s *PassportService) CreatePassportLaunchAccount(ctx context.Context, req *v1.PassportLaunchAppRequest) (*v1.PassportLink, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.CreateLaunchAccount(ctx, c.UserID, req.AppId)
}

func (s *PassportService) IssuePassportLaunchTicket(ctx context.Context, req *v1.PassportLaunchAppRequest) (*v1.PassportLaunchTicket, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.IssueLaunchTicket(ctx, c.UserID, req.AppId)
}

// ---------------- 游戏客户端日志上传 ----------------

// PassportClientLogPath 游戏客户端上传错误日志：原始 HTTP 路由，不用通行证令牌（鉴权过滤器放行），
// 处理函数自己校验游戏服务端签发的上传票据（Authorization: Bearer <票据>）。
const PassportClientLogPath = "/api/v1/game-client-logs"

// clientLogStoreTimeout 读完请求体之后保存日志的时限（不受请求超时影响，慢速上传读完后仍能保存）。
const clientLogStoreTimeout = 30 * time.Second

// RegisterClientLogServer 注册客户端日志上传路由。
func (s *PassportService) RegisterClientLogServer(srv *khttp.Server) {
	srv.HandleFunc(PassportClientLogPath, s.uploadClientLog)
}

// clientLogRequest 上传的请求体（JSON；Content-Encoding: gzip 时为 gzip 压缩的 JSON）。
type clientLogRequest struct {
	Reason        string `json:"reason"`
	Summary       string `json:"summary"`
	Note          string `json:"note"`
	Device        string `json:"device"`
	ClientVersion string `json:"clientVersion"`
	// OccurredAt Unix 毫秒
	OccurredAt int64  `json:"occurredAt"`
	Content    string `json:"content"`
}

// readClientLogBody 读取请求体：压缩后（或不压缩时）不超过 8 MiB，解压后不超过 32 MiB，超过时 PASSPORT_CLIENT_LOG_TOO_LARGE。
func readClientLogBody(r *http.Request) ([]byte, error) {
	if r.ContentLength > biz.ClientLogMaxBody {
		return nil, biz.ErrClientLogTooLarge
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, biz.ClientLogMaxBody+1))
	if err != nil {
		return nil, biz.ErrClientLogInvalid
	}
	if len(raw) > biz.ClientLogMaxBody {
		return nil, biz.ErrClientLogTooLarge
	}
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding"))) {
	case "", "identity":
		return raw, nil
	case "gzip":
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, biz.ErrClientLogInvalid
		}
		defer zr.Close()
		body, err := io.ReadAll(io.LimitReader(zr, biz.ClientLogMaxContent+1))
		if err != nil {
			return nil, biz.ErrClientLogInvalid
		}
		if len(body) > biz.ClientLogMaxContent {
			return nil, biz.ErrClientLogTooLarge
		}
		return body, nil
	default:
		return nil, biz.ErrClientLogEncoding
	}
}

// uploadClientLog POST /api/v1/game-client-logs：先校验票据再读请求体，保存后返回 201 {"id","createdAt"}。
// 失败时为通行证的统一错误响应：票据无效 401，其余（内容不合规、太大、不支持的压缩方式、上传太频繁）400，按错误代码区分。
func (s *PassportService) uploadClientLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, r, kerrors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method Not Allowed"))
		return
	}
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		response.WriteError(w, r, biz.ErrClientLogTicket)
		return
	}
	owner, err := s.uc.ClientLogOwner(r.Context(), parts[1])
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	body, err := readClientLogBody(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req clientLogRequest
	if err := json.Unmarshal(body, &req); err != nil {
		response.WriteError(w, r, biz.ErrClientLogInvalid)
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), clientLogStoreTimeout)
	defer cancel()
	saved, err := s.uc.UploadClientLog(ctx, owner, biz.ClientLogUpload{
		Reason:        req.Reason,
		Summary:       req.Summary,
		Note:          req.Note,
		Device:        req.Device,
		ClientVersion: req.ClientVersion,
		OccurredAt:    req.OccurredAt,
		Content:       req.Content,
	})
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":        saved.ID,
		"createdAt": saved.CreatedAt.UTC().Format(time.RFC3339),
	})
}

// ---------------- 后台（主 server，权限 game:passport） ----------------
