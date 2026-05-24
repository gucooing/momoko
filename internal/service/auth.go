package service

import (
	"context"

	"google.golang.org/protobuf/types/known/durationpb"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	auth2 "momoko/internal/data/ent/gen/auth"
	genuser "momoko/internal/data/ent/gen/user"
	"momoko/pkg/auth"
	"momoko/pkg/response"

	"time"
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
	loginConfig, err := s.conf.LoginConfig(ctx)
	if err != nil {
		return nil, err
	}
	switch req.Identity.(type) {
	case *v1.LoginRequest_Username:
		if !loginConfig.UsernameLoginEnabled {
			return nil, biz.ErrUsernameLoginDisabled
		}
		user, err = s.uc.LoginByUsername(ctx, req.GetUsername(), req.GetPassword())
	case *v1.LoginRequest_Email:
		if !loginConfig.EmailLoginEnabled {
			return nil, biz.ErrEmailLoginDisabled
		}
		user, err = s.uc.LoginByEmail(ctx, req.GetEmail(), req.GetCode())
	default:
		return nil, response.BadRequest(400, "请选择登录方式")
	}
	if err != nil {
		return nil, err
	}
	access, err := s.uc.NewAccessToken(ctx, user.ID, req)
	if err != nil {
		return nil, err
	}
	refresh, err := s.uc.NewRefreshToken(ctx, user.ID, req)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateToken(access)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateToken(refresh)
	if err != nil {
		return nil, err
	}

	return &v1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    durationpb.New(time.Hour),
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

	code, err := s.uc.NewEmailCode(req.Email, biz.EmailCodeTypeRegister)
	if err != nil {
		return nil, err
	}
	if err := s.system.SendEmail(ctx, v1.EmailTemplateType_EmailTemplateType_Register, req.Email, map[string]string{
		"code":  code,
		"email": req.Email,
	}); err != nil {
		return nil, err
	}
	return &v1.SendRegisterEmailCodeResponse{}, nil
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

	code, err := s.uc.NewEmailCode(req.Email, biz.EmailCodeTypeLogin)
	if err != nil {
		return nil, err
	}
	if err := s.system.SendEmail(ctx, v1.EmailTemplateType_EmailTemplateType_Login, req.Email, map[string]string{
		"code":  code,
		"email": req.Email,
		"name":  userInfo.Name,
	}); err != nil {
		return nil, err
	}
	return &v1.SendLoginEmailCodeResponse{}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req *v1.RefreshRequest) (*v1.RefreshResponse, error) {
	refreshAuth, err := auth.ParseToken(req.RefreshToken)
	if err != nil {
		return nil, biz.ErrTokenInvalid
	}
	if refreshAuth.NotBefore.Add(7 * 24 * time.Hour).Before(time.Now()) {
		return nil, biz.ErrTokenInvalid
	}
	if !s.uc.VerifyToken(ctx, refreshAuth, auth2.TypeRefreshToken) {
		return nil, biz.ErrTokenInvalid
	}
	// 更新token
	access, refresh, err := s.uc.RefreshToken(ctx, refreshAuth.UserID, refreshAuth.DeviceId)
	if err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateToken(access)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateToken(refresh)
	if err != nil {
		return nil, err
	}

	return &v1.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    durationpb.New(time.Hour),
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
		Devices:  list,
		DeviceId: authInfo.DeviceId,
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
	return &v1.UpdatePasswordResponse{}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	err := s.uc.Logout(ctx, authInfo.UserID, authInfo.DeviceId)
	if err != nil {
		return nil, err
	}
	return &v1.LogoutResponse{}, nil
}

func (s *AuthService) DelLogin(ctx context.Context, req *v1.DelLoginRequest) (*v1.DelLoginResponse, error) {
	authInfo, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	err := s.uc.Logout(ctx, authInfo.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.DelLoginResponse{}, nil
}
