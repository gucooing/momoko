package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type NetworkService struct {
	v1.UnimplementedNetworkManagerServer

	uc *biz.NetworkUsecase
}

func NewNetworkService(uc *biz.NetworkUsecase) *NetworkService {
	return &NetworkService{uc: uc}
}

func (n *NetworkService) ListPortForwards(ctx context.Context, req *v1.ListPortForwardsRequest) (*v1.ListPortForwardsResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return n.uc.ListPortForwards(ctx, authCtx.UserID, req)
}

func (n *NetworkService) CreatePortForward(ctx context.Context, req *v1.CreatePortForwardRequest) (*v1.CreatePortForwardResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := n.uc.CreatePortForward(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreatePortForwardResponse{Info: info}, nil
}

func (n *NetworkService) GetPortForward(ctx context.Context, req *v1.GetPortForwardRequest) (*v1.GetPortForwardResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := n.uc.GetPortForward(ctx, authCtx.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetPortForwardResponse{Info: info}, nil
}

func (n *NetworkService) UpdatePortForward(ctx context.Context, req *v1.UpdatePortForwardRequest) (*v1.UpdatePortForwardResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := n.uc.UpdatePortForward(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePortForwardResponse{Info: info}, nil
}

func (n *NetworkService) DeletePortForward(ctx context.Context, req *v1.DeletePortForwardRequest) (*v1.DeletePortForwardResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := n.uc.DeletePortForward(ctx, authCtx.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeletePortForwardResponse{}, nil
}
