package service

import (
	"context"
	"strings"

	"google.golang.org/protobuf/types/known/durationpb"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	genuser "momoko/internal/data/ent/gen/user"
	"momoko/pkg/auth"
)

type AuthService struct {
	v1.UnimplementedAuthServiceServer

	uc     *biz.AuthUsecase
	user   *biz.UserUsecase
	conf   *biz.ConfigUsecase
	system *biz.SystemUsecase
}

func NewAuthService(uc *biz.AuthUsecase, user *biz.UserUsecase, conf *biz.ConfigUsecase, system *biz.SystemUsecase) *AuthService {
	return &AuthService{
		uc:     uc,
		user:   user,
		conf:   conf,
		system: system,
	}
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	var (
		user *gen.User
		err  error
	)
	cfg, err := s.conf.LoginConfig(ctx)
	if err != nil {
		return nil, err
	}
	// 登录失败锁定（账号 + IP）
	attemptKey := biz.LoginAttemptKey(ctx, req)
	if err := biz.CheckLoginLocked(attemptKey, cfg); err != nil {
		return nil, err
	}
	switch req.Identity.(type) {
	case *v1.LoginRequest_Username:
		if !cfg.UsernameLoginEnabled {
			return nil, biz.ErrUsernameLoginDisabled
		}
		user, err = s.uc.LoginByUsername(ctx, req.GetUsername(), req.GetPassword())
	case *v1.LoginRequest_Email:
		if !cfg.EmailLoginEnabled {
			return nil, biz.ErrEmailLoginDisabled
		}
		user, err = s.uc.LoginByEmail(ctx, req.GetEmail(), req.GetCode())
	default:
		return nil, biz.ErrNotLoginType
	}
	if err != nil {
		if biz.IsLoginCredentialError(err) {
			if lockErr := biz.RecordLoginFailure(attemptKey, cfg); lockErr != nil {
				return nil, lockErr
			}
		}
		return nil, err
	}
	biz.ClearLoginFailures(attemptKey)
	if req.GetDeviceId() == "" {
		return nil, biz.ErrNotDeviceID
	}

	session, err := s.uc.NewSession(ctx, user.ID, req)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateAccessToken(session)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateRefreshToken(session)
	if err != nil {
		return nil, err
	}

	return &v1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    durationpb.New(auth.AccessTokenTTL()),
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	loginConfig, err := s.conf.LoginConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !loginConfig.RegisterEnabled {
		return nil, biz.ErrRegisterDisabled
	}

	userInfo, err := s.uc.Register(ctx, req, loginConfig.RegisterEmailVerificationRequired)
	if err != nil {
		return nil, err
	}
	return &v1.RegisterResponse{UserId: userInfo.ID}, nil
}

func (s *AuthService) SendRegisterEmailCode(ctx context.Context, req *v1.SendRegisterEmailCodeRequest) (*v1.SendRegisterEmailCodeResponse, error) {
	loginConfig, err := s.conf.LoginConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !loginConfig.RegisterEnabled {
		return nil, biz.ErrRegisterDisabled
	}

	if _, err = s.user.FindByEmail(ctx, req.Email); err == nil {
		return nil, biz.ErrEmailRegistered
	} else if !gen.IsNotFound(err) {
		return nil, biz.ErrSystem(err)
	}

	if err := s.uc.IssueEmailCode(req.Email, biz.EmailCodeTypeRegister, func(code string) error {
		return s.system.SendEmail(ctx, v1.EmailTemplateType_EmailTemplateType_Register, req.Email, map[string]string{
			"code":  code,
			"email": req.Email,
		})
	}); err != nil {
		return nil, err
	}
	return &v1.SendRegisterEmailCodeResponse{ExpiresInSeconds: int64(biz.EmailCodeTTL.Seconds())}, nil
}

func (s *AuthService) SendLoginEmailCode(ctx context.Context, req *v1.SendLoginEmailCodeRequest) (*v1.SendLoginEmailCodeResponse, error) {
	loginConfig, err := s.conf.LoginConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !loginConfig.EmailLoginEnabled {
		return nil, biz.ErrEmailLoginDisabled
	}

	userInfo, err := s.user.FindByEmail(ctx, req.Email)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, biz.ErrAdminNotFound
		}
		return nil, biz.ErrSystem(err)
	}
	if userInfo.Status != genuser.StatusActive {
		return nil, biz.ErrUserInactive
	}

	if err := s.uc.IssueEmailCode(req.Email, biz.EmailCodeTypeLogin, func(code string) error {
		return s.system.SendEmail(ctx, v1.EmailTemplateType_EmailTemplateType_Login, req.Email, map[string]string{
			"code":  code,
			"email": req.Email,
			"name":  userInfo.Name,
		})
	}); err != nil {
		return nil, err
	}
	return &v1.SendLoginEmailCodeResponse{ExpiresInSeconds: int64(biz.EmailCodeTTL.Seconds())}, nil
}

func (s *AuthService) SendResetPasswordEmailCode(ctx context.Context, req *v1.SendResetPasswordEmailCodeRequest) (*v1.SendResetPasswordEmailCodeResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	if email == "" {
		return nil, biz.ErrEmailEmpty
	}
	userInfo, err := s.user.FindByEmail(ctx, email)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, biz.ErrAdminNotFound
		}
		return nil, biz.ErrSystem(err)
	}
	if userInfo.Status != genuser.StatusActive {
		return nil, biz.ErrUserInactive
	}

	if err := s.uc.IssueEmailCode(email, biz.EmailCodeTypeReset, func(code string) error {
		return s.system.SendEmail(ctx, v1.EmailTemplateType_EmailTemplateType_ResetPassword, email, map[string]string{
			"code":  code,
			"email": email,
			"name":  userInfo.Name,
		})
	}); err != nil {
		return nil, err
	}
	return &v1.SendResetPasswordEmailCodeResponse{ExpiresInSeconds: int64(biz.EmailCodeTTL.Seconds())}, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req *v1.ResetPasswordRequest) (*v1.ResetPasswordResponse, error) {
	if err := s.uc.ResetPassword(ctx, req.GetEmail(), req.GetCode(), req.GetNewPassword()); err != nil {
		return nil, err
	}
	return &v1.ResetPasswordResponse{}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req *v1.RefreshRequest) (*v1.RefreshResponse, error) {
	refreshClaims, err := auth.ParseToken(req.RefreshToken)
	if err != nil {
		return nil, biz.ErrTokenInvalid
	}
	if !s.uc.VerifyToken(ctx, refreshClaims, auth.TokenKindRefresh) {
		return nil, biz.ErrTokenInvalid
	}
	session, err := s.uc.RefreshToken(ctx, refreshClaims.SessionID)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateAccessToken(session)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateRefreshToken(session)
	if err != nil {
		return nil, err
	}

	return &v1.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    durationpb.New(auth.AccessTokenTTL()),
	}, nil
}

func (s *AuthService) Devices(ctx context.Context, req *v1.DevicesRequest) (*v1.DevicesResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	list, err := s.uc.ListLoginDevice(ctx, authInfo.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.DevicesResponse{
		Devices:   list,
		SessionId: authInfo.SessionID,
	}, nil
}

func (s *AuthService) UpdatePassword(ctx context.Context, req *v1.UpdatePasswordRequest) (*v1.UpdatePasswordResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	_, err := s.user.UpdatePassword(ctx, authInfo.UserID, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, err
	}
	s.uc.NotifyPasswordChanged(ctx, authInfo.UserID)
	return &v1.UpdatePasswordResponse{}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	var err error
	switch {
	case req.GetOthers():
		err = s.uc.LogoutOthers(ctx, authInfo.UserID, authInfo.SessionID)
	case req.SessionId != nil && *req.SessionId != "":
		err = s.uc.Logout(ctx, authInfo.UserID, req.SessionId)
	default:
		// 未指定会话：只退出当前会话（不影响其它设备）。
		current := authInfo.SessionID
		err = s.uc.Logout(ctx, authInfo.UserID, &current)
	}
	if err != nil {
		return nil, err
	}
	return &v1.LogoutResponse{}, nil
}

func (s *AuthService) ListMessages(ctx context.Context, req *v1.ListMessagesRequest) (*v1.ListMessagesResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return s.uc.ListMessages(ctx, authInfo.UserID, req)
}

func (s *AuthService) ReadMessages(ctx context.Context, req *v1.ReadMessagesRequest) (*v1.ReadMessagesResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	unread, err := s.uc.ReadMessages(ctx, authInfo.UserID, req.GetIds(), req.GetAll())
	if err != nil {
		return nil, err
	}
	return &v1.ReadMessagesResponse{UnreadCount: unread}, nil
}

func (s *AuthService) DeleteMessages(ctx context.Context, req *v1.DeleteMessagesRequest) (*v1.DeleteMessagesResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	unread, err := s.uc.DeleteMessages(ctx, authInfo.UserID, req.GetIds(), req.GetAll())
	if err != nil {
		return nil, err
	}
	return &v1.DeleteMessagesResponse{UnreadCount: unread}, nil
}

func (s *AuthService) PublishMessage(ctx context.Context, req *v1.PublishMessageRequest) (*v1.PublishMessageResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	n, err := s.uc.PublishMessage(ctx, authInfo.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.PublishMessageResponse{Recipients: n}, nil
}
