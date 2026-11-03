package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

// IQCheckService 同时承担后台管理接口（IQCheckManager）和公开页接口（IQCheckPublic）。
type IQCheckService struct {
	v1.UnimplementedIQCheckManagerServer
	v1.UnimplementedIQCheckPublicServer

	uc *biz.IQCheckUsecase
}

func NewIQCheckService(uc *biz.IQCheckUsecase) *IQCheckService {
	return &IQCheckService{uc: uc}
}

func (s *IQCheckService) ListIQCheckPrompts(ctx context.Context, req *v1.ListIQCheckPromptsRequest) (*v1.ListIQCheckPromptsResponse, error) {
	return s.uc.ListPrompts(ctx, req)
}

func (s *IQCheckService) CreateIQCheckPrompt(ctx context.Context, req *v1.CreateIQCheckPromptRequest) (*v1.CreateIQCheckPromptResponse, error) {
	prompt, err := s.uc.CreatePrompt(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateIQCheckPromptResponse{Prompt: prompt}, nil
}

func (s *IQCheckService) UpdateIQCheckPrompt(ctx context.Context, req *v1.UpdateIQCheckPromptRequest) (*v1.UpdateIQCheckPromptResponse, error) {
	prompt, err := s.uc.UpdatePrompt(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateIQCheckPromptResponse{Prompt: prompt}, nil
}

func (s *IQCheckService) DeleteIQCheckPrompt(ctx context.Context, req *v1.DeleteIQCheckPromptRequest) (*v1.DeleteIQCheckPromptResponse, error) {
	if err := s.uc.DeletePrompt(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteIQCheckPromptResponse{}, nil
}

func (s *IQCheckService) ListIQCheckSites(ctx context.Context, req *v1.ListIQCheckSitesRequest) (*v1.ListIQCheckSitesResponse, error) {
	return s.uc.ListSites(ctx, req)
}

func (s *IQCheckService) CreateIQCheckSite(ctx context.Context, req *v1.CreateIQCheckSiteRequest) (*v1.CreateIQCheckSiteResponse, error) {
	site, err := s.uc.CreateSite(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateIQCheckSiteResponse{Site: site}, nil
}

func (s *IQCheckService) UpdateIQCheckSite(ctx context.Context, req *v1.UpdateIQCheckSiteRequest) (*v1.UpdateIQCheckSiteResponse, error) {
	site, err := s.uc.UpdateSite(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateIQCheckSiteResponse{Site: site}, nil
}

func (s *IQCheckService) DeleteIQCheckSite(ctx context.Context, req *v1.DeleteIQCheckSiteRequest) (*v1.DeleteIQCheckSiteResponse, error) {
	if err := s.uc.DeleteSite(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteIQCheckSiteResponse{}, nil
}

func (s *IQCheckService) FetchIQCheckModels(ctx context.Context, req *v1.FetchIQCheckModelsRequest) (*v1.FetchIQCheckModelsResponse, error) {
	models, err := s.uc.FetchModels(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.FetchIQCheckModelsResponse{Models: models}, nil
}

func (s *IQCheckService) TriggerIQCheck(ctx context.Context, req *v1.TriggerIQCheckRequest) (*v1.TriggerIQCheckResponse, error) {
	count, err := s.uc.TriggerCheck(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.TriggerIQCheckResponse{ModelCount: count}, nil
}

func (s *IQCheckService) PublicIQCheckSite(ctx context.Context, req *v1.PublicIQCheckSiteRequest) (*v1.PublicIQCheckSiteResponse, error) {
	site, err := s.uc.PublicSite(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return &v1.PublicIQCheckSiteResponse{Site: site}, nil
}

func (s *IQCheckService) PublicIQCheckResults(ctx context.Context, req *v1.PublicIQCheckResultsRequest) (*v1.PublicIQCheckResultsResponse, error) {
	results, total, err := s.uc.PublicResults(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.PublicIQCheckResultsResponse{Page: req.Page, PageSize: req.PageSize, Total: total, Results: results}, nil
}
