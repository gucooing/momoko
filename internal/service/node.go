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
	infos, total, counts, err := n.uc.ListAPIKeys(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.ListAPIKeysResponse{
		Page:          req.Page,
		PageSize:      req.PageSize,
		Total:         total,
		Infos:         infos,
		ValidCount:    counts.Valid,
		ExpiringCount: counts.Expiring,
		ExpiredCount:  counts.Expired,
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

func (n *NodeService) DeleteAPIKey(ctx context.Context, req *v1.DeleteAPIKeyRequest) (*v1.DeleteAPIKeyResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := n.uc.DeleteAPIKey(ctx, authCtx.UserID, req); err != nil {
		return nil, err
	}
	return &v1.DeleteAPIKeyResponse{}, nil
}

func (n *NodeService) ListNodes(ctx context.Context, req *v1.ListNodesRequest) (*v1.ListNodesResponse, error) {
	infos, err := n.uc.ListNodes(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ListNodesResponse{Infos: infos, Local: biz.LocalNodeInfo(ctx)}, nil
}

func (n *NodeService) CreateNode(ctx context.Context, req *v1.CreateNodeRequest) (*v1.CreateNodeResponse, error) {
	info, err := n.uc.CreateNode(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateNodeResponse{Info: info}, nil
}

func (n *NodeService) UpdateNode(ctx context.Context, req *v1.UpdateNodeRequest) (*v1.UpdateNodeResponse, error) {
	info, err := n.uc.UpdateNode(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateNodeResponse{Info: info}, nil
}

func (n *NodeService) DeleteNode(ctx context.Context, req *v1.DeleteNodeRequest) (*v1.DeleteNodeResponse, error) {
	if err := n.uc.DeleteNode(ctx, req); err != nil {
		return nil, err
	}
	return &v1.DeleteNodeResponse{}, nil
}

func (n *NodeService) TestNode(ctx context.Context, req *v1.TestNodeRequest) (*v1.TestNodeResponse, error) {
	status, err := n.uc.TestNode(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.TestNodeResponse{Status: status}, nil
}

func (n *NodeService) GetNodeContext(ctx context.Context, req *v1.GetNodeContextRequest) (*v1.GetNodeContextResponse, error) {
	info, permissions, err := n.uc.GetNodeContext(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.GetNodeContextResponse{Info: info, Permissions: permissions}, nil
}
