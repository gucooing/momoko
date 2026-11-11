package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

// ThemeService 自定义外观主题：除「设为全站默认」（system:config:edit，鉴权在中间件完成）外都是仅需登录的自助操作，
// 全站默认主题对未登录访客公开（/api/v1/public/site-theme）。只在本机处理，不转发到远程节点。
type ThemeService struct {
	v1.UnimplementedThemeServiceServer

	uc *biz.ThemeUsecase
}

func NewThemeService(uc *biz.ThemeUsecase) *ThemeService {
	return &ThemeService{uc: uc}
}

func currentUserID(ctx context.Context) (string, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok || authCtx.UserID == "" {
		return "", biz.ErrTokenInvalid
	}
	return authCtx.UserID, nil
}

func (s *ThemeService) ListThemes(ctx context.Context, _ *v1.ListThemesRequest) (*v1.ListThemesResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	themes, err := s.uc.ListThemes(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &v1.ListThemesResponse{Themes: themes}, nil
}

func (s *ThemeService) GetTheme(ctx context.Context, req *v1.GetThemeRequest) (*v1.GetThemeResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	theme, err := s.uc.GetTheme(ctx, userID, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetThemeResponse{Theme: theme}, nil
}

func (s *ThemeService) CreateTheme(ctx context.Context, req *v1.CreateThemeRequest) (*v1.CreateThemeResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	theme, err := s.uc.CreateTheme(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateThemeResponse{Theme: theme}, nil
}

func (s *ThemeService) UpdateTheme(ctx context.Context, req *v1.UpdateThemeRequest) (*v1.UpdateThemeResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	theme, err := s.uc.UpdateTheme(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateThemeResponse{Theme: theme}, nil
}

func (s *ThemeService) DeleteTheme(ctx context.Context, req *v1.DeleteThemeRequest) (*v1.DeleteThemeResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.DeleteTheme(ctx, userID, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteThemeResponse{}, nil
}

func (s *ThemeService) DuplicateTheme(ctx context.Context, req *v1.DuplicateThemeRequest) (*v1.DuplicateThemeResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	theme, err := s.uc.DuplicateTheme(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &v1.DuplicateThemeResponse{Theme: theme}, nil
}

func (s *ThemeService) UploadThemeAsset(ctx context.Context, req *v1.UploadThemeAssetRequest) (*v1.UploadThemeAssetResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	asset, err := s.uc.UploadAsset(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &v1.UploadThemeAssetResponse{Asset: asset}, nil
}

func (s *ThemeService) InspectThemePackage(ctx context.Context, req *v1.InspectThemePackageRequest) (*v1.InspectThemePackageResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	info, err := s.uc.InspectPackage(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &v1.InspectThemePackageResponse{Info: info}, nil
}

func (s *ThemeService) ImportThemePackage(ctx context.Context, req *v1.ImportThemePackageRequest) (*v1.ImportThemePackageResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	theme, err := s.uc.ImportPackage(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &v1.ImportThemePackageResponse{Theme: theme}, nil
}

func (s *ThemeService) ExportThemePackage(ctx context.Context, req *v1.ExportThemePackageRequest) (*v1.ExportThemePackageResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.ExportPackage(ctx, userID, req)
}

func (s *ThemeService) GetSiteTheme(ctx context.Context, _ *v1.GetSiteThemeRequest) (*v1.GetSiteThemeResponse, error) {
	theme, err := s.uc.SiteTheme(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetSiteThemeResponse{Theme: theme}, nil
}

func (s *ThemeService) SetSiteTheme(ctx context.Context, req *v1.SetSiteThemeRequest) (*v1.SetSiteThemeResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	theme, err := s.uc.SetSiteTheme(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &v1.SetSiteThemeResponse{Theme: theme}, nil
}
