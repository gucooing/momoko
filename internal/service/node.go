package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type NodeService struct {
	v1.UnimplementedNodeServiceServer

	uc *biz.NodeUsecase
}

func NewNodeService(uc *biz.NodeUsecase) *NodeService {
	return &NodeService{uc: uc}
}

func (n *NodeService) CreateAPIKey(ctx context.Context, req *v1.CreateAPIKeyRequest) (*v1.CreateAPIKeyResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := n.uc.CreateAPIKey(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateAPIKeyResponse{Info: info}, nil
}

func (n *NodeService) ListAPIKeys(ctx context.Context, req *v1.ListAPIKeysRequest) (*v1.ListAPIKeysResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	infos, total, err := n.uc.ListAPIKeys(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.ListAPIKeysResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		Infos:    infos,
	}, nil
}

func (n *NodeService) CopyAPIKey(ctx context.Context, req *v1.CopyAPIKeyRequest) (*v1.CopyAPIKeyResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := n.uc.CopyAPIKey(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.CopyAPIKeyResponse{Info: info}, nil
}

func (n *NodeService) UpdateAPIKey(ctx context.Context, req *v1.UpdateAPIKeyRequest) (*v1.UpdateAPIKeyResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := n.uc.UpdateAPIKey(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAPIKeyResponse{Info: info}, nil
}

func (n *NodeService) RefreshAPIKey(ctx context.Context, req *v1.RefreshAPIKeyRequest) (*v1.RefreshAPIKeyResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := n.uc.RefreshAPIKey(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.RefreshAPIKeyResponse{Info: info}, nil
}
