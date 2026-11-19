package service

import (
	"context"
	"slices"
	"time"

	"github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/constant"
	"momoko/pkg/mcp"
)

// McpService Momoko MCP：MCP 服务本身（原始路由 mcp.Path），和它的管理页接口。
type McpService struct {
	v1.UnimplementedMcpServiceServer

	uc     *biz.McpUsecase
	server *mcp.Server
}

// NewMcpService 两个 Agent 工具调 AgentUsecase，凭证能不能调用工具问 McpUsecase。
func NewMcpService(uc *biz.McpUsecase, agent *biz.AgentUsecase) *McpService {
	return &McpService{uc: uc, server: mcp.New(agent, uc)}
}

// RegisterRawRoutes MCP 服务。凭证（API Key）在 HTTP 鉴权中间件里认过，这里只收认过的请求。
func (s *McpService) RegisterRawRoutes(srv *http.Server) {
	srv.Handle(mcp.Path, s.server)
}

func (s *McpService) GetMcpOverview(ctx context.Context, _ *v1.GetMcpOverviewRequest) (*v1.GetMcpOverviewResponse, error) {
	tools := s.server.Tools()
	resp := &v1.GetMcpOverviewResponse{Path: mcp.Path}
	perms := make([]constant.Permissions, 0, len(tools))
	for _, t := range tools {
		resp.Tools = append(resp.Tools, &v1.McpTool{Name: t.Name, Permission: string(t.Permission)})
		if !slices.Contains(perms, t.Permission) {
			perms = append(perms, t.Permission)
		}
	}
	creds, err := s.uc.Credentials(ctx, perms)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for _, c := range creds {
		item := &v1.McpCredential{
			Id:         c.Key.ID,
			Name:       c.Key.Name,
			KeyId:      c.Key.KeyID,
			Status:     biz.APIKeyStatusOf(c.Key.ExpiresAt, now),
			LastUsedIp: c.Key.LastUsedIP,
		}
		if c.Key.ExpiresAt != nil {
			item.ExpiresAt = timestamppb.New(*c.Key.ExpiresAt)
		}
		if c.Key.LastUsedAt != nil {
			item.LastUsedTime = timestamppb.New(*c.Key.LastUsedAt)
		}
		for _, t := range tools {
			if c.Granted[t.Permission] {
				item.Tools = append(item.Tools, t.Name)
			}
		}
		resp.Credentials = append(resp.Credentials, item)
	}
	return resp, nil
}
