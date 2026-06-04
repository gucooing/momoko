package biz

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/portforward"
	"momoko/pkg/port_forward"

	v1 "momoko/api/gen/v1"
)

type NetworkRepo interface {
	CreatePortForward(ctx context.Context, userID string, req *v1.CreatePortForwardRequest) (*gen.PortForward, error)
	ListPortForwards(ctx context.Context, isEnable *bool, userID *string, protocol *v1.PortForwardType) ([]*gen.PortForward, int64, error)
	UpdatePortForward(ctx context.Context, userID *string, req *v1.UpdatePortForwardRequest) (*gen.PortForward, error)
	GetPortForward(ctx context.Context, userID *string, id string) (*gen.PortForward, error)
	DeletePortForward(ctx context.Context, userID *string, id string) error
}

type NetworkUsecase struct {
	repo    NetworkRepo
	manager *port_forward.Manager
}

func NewNetworkUsecase(repo NetworkRepo) (*NetworkUsecase, func(), error) {
	uc := &NetworkUsecase{
		repo:    repo,
		manager: port_forward.NewManager(),
	}
	ctx := context.Background()
	list, _, err := repo.ListPortForwards(ctx, new(true), nil, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, info := range list {
		uc.manager.RegisterExample(toPortForwardOption(info))
	}

	return uc, uc.manager.Stop, nil
}

func (n *NetworkUsecase) ListPortForwards(ctx context.Context, userID string, req *v1.ListPortForwardsRequest) (*v1.ListPortForwardsResponse, error) {
	infos, count, err := n.repo.ListPortForwards(ctx, req.IsEnable, new(userID), req.Type)
	if err != nil {
		return nil, err
	}
	rsp := &v1.ListPortForwardsResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    count,
		Infos:    make([]*v1.PortForwardInfo, 0, len(infos)),
	}
	for _, info := range infos {
		rsp.Infos = append(rsp.Infos, n.toV1PortForwardInfo(info))
	}
	return rsp, nil
}

func (n *NetworkUsecase) CreatePortForward(ctx context.Context, userID string, req *v1.CreatePortForwardRequest) (*v1.PortForwardInfo, error) {
	info, err := n.repo.CreatePortForward(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	if info.IsEnable {
		err = n.manager.Retry(toPortForwardOption(info))
	}
	item := n.toV1PortForwardInfo(info)
	if err != nil {
		item.Error = err.Error()
	}
	return item, nil
}

func (n *NetworkUsecase) GetPortForward(ctx context.Context, userID, id string) (*v1.PortForwardInfo, error) {
	info, err := n.repo.GetPortForward(ctx, new(userID), id)
	if err != nil {
		return nil, err
	}
	return n.toV1PortForwardInfo(info), nil
}

func (n *NetworkUsecase) UpdatePortForward(ctx context.Context, userID string, req *v1.UpdatePortForwardRequest) (*v1.PortForwardInfo, error) {
	info, err := n.repo.UpdatePortForward(ctx, new(userID), req)
	if err != nil {
		return nil, err
	}
	if info.IsEnable {
		err = n.manager.Retry(toPortForwardOption(info))
	} else {
		n.manager.UnRegisterExample(info.ID)
	}
	item := n.toV1PortForwardInfo(info)
	if err != nil {
		item.Error = err.Error()
	}

	return item, nil
}

func (n *NetworkUsecase) DeletePortForward(ctx context.Context, userID, id string) error {
	n.manager.UnRegisterExample(id)
	return n.repo.DeletePortForward(ctx, new(userID), id)
}

func (n *NetworkUsecase) toV1PortForwardInfo(forward *gen.PortForward) *v1.PortForwardInfo {
	info := &v1.PortForwardInfo{
		Id:            forward.ID,
		Name:          forward.Name,
		UserId:        forward.UserID,
		Type:          toV1PortForwardProtocol(forward.Protocol),
		ListenAddress: forward.ListenAddress,
		ListenPort:    int32(forward.ListenPort),
		TargetAddress: forward.TargetAddress,
		TargetPort:    int32(forward.TargetPort),
		IsEnable:      forward.IsEnable && n.manager.Running(forward.ID),
		Remark:        forward.Remark,
		Tags:          forward.Tags,
		CreateTime:    timestamppb.New(forward.CreateTime),
		UpdateTime:    timestamppb.New(forward.UpdateTime),
	}

	return info
}

func toPortForwardOption(forward *gen.PortForward) *port_forward.Option {
	return &port_forward.Option{
		ID:            forward.ID,
		Protocol:      toPortForwardProtocol(forward.Protocol),
		ListenAddress: forward.ListenAddress,
		ListenPort:    forward.ListenPort,
		TargetPort:    forward.TargetPort,
		TargetAddress: forward.TargetAddress,
	}
}

func toV1PortForwardProtocol(forward portforward.Protocol) v1.PortForwardType {
	switch forward {
	case portforward.ProtocolTCP:
		return v1.PortForwardType_PORT_FORWARD_TYPE_TCP
	case portforward.ProtocolUDP:
		return v1.PortForwardType_PORT_FORWARD_TYPE_UDP
	default:
		return v1.PortForwardType_PORT_FORWARD_TYPE_UNSPECIFIED
	}
}

func toPortForwardProtocol(forward portforward.Protocol) port_forward.Protocol {
	switch forward {
	case portforward.ProtocolTCP:
		return port_forward.TCP
	case portforward.ProtocolUDP:
		return port_forward.UDP
	default:
		return port_forward.TCP
	}
}
