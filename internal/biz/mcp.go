package biz

import (
	"context"
	"crypto/subtle"
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/user"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
	"momoko/pkg/nodeauth"
	"momoko/pkg/response"
)

// Momoko MCP 的凭证就是 API Key：请求头带 Authorization: Bearer <API Key>。
// 一把 Key 能调用哪个工具和它能在远程节点上做什么是同一个判断（NodeAgentUsecase.Authorize）：
// 授予范围要包含工具要的权限，所属用户也要有它。协议、工具和分发在 pkg/mcp。
var (
	ErrMcpCredential       = response.Unauthorized("MCP 凭证无效：请求头要带 Authorization: Bearer <API Key>")
	ErrMcpPermissionDenied = func(perm constant.Permissions) error {
		return response.BadRequest("这把 API Key 不能调用这个工具，需要权限：" + string(perm))
	}
)

// McpRepo 列出能当 MCP 凭证的 API Key。
type McpRepo interface {
	// ListMcpCredentials 某个用户的 API Key 里授予了 perms 中任一项的，按创建时间排。
	ListMcpCredentials(ctx context.Context, userID string, perms []constant.Permissions) ([]*gen.UserAPIKey, error)
}

// McpUsecase 只管两件事：请求带的凭证在不在，以及它能不能调用某个工具要的权限。
type McpUsecase struct {
	repo  McpRepo
	keys  APIKeyRepo
	users UserRepo
	node  *NodeAgentUsecase
}

func NewMcpUsecase(repo McpRepo, keys APIKeyRepo, users UserRepo, node *NodeAgentUsecase) *McpUsecase {
	return &McpUsecase{repo: repo, keys: keys, users: users, node: node}
}

type mcpCallerKey struct{}

// mcpPrincipal 一把 API Key 作为调用主体：以所属用户的身份执行，授予范围就是 Key 的权限。
func mcpPrincipal(key *gen.UserAPIKey) *NodePrincipal {
	scopes := make(map[constant.Permissions]struct{}, len(key.Permissions))
	for _, perm := range key.Permissions {
		scopes[constant.Permissions(perm)] = struct{}{}
	}
	return &NodePrincipal{
		APIKeyID:   key.ID,
		APIKeyName: key.Name,
		KeyID:      key.KeyID,
		UserID:     key.UserID,
		ExpiresAt:  key.ExpiresAt,
		scopes:     scopes,
	}
}

func apiKeyExpired(key *gen.UserAPIKey, now time.Time) bool {
	return key.ExpiresAt != nil && !now.Before(*key.ExpiresAt)
}

// Authenticate 认请求带的 API Key：不存在、过期、所属用户停用都是 401。
// 返回的 ctx 以所属用户的身份继续处理，并带上这把 Key，调用工具时据此检查权限。ip 是请求来源，记为最近使用。
func (u *McpUsecase) Authenticate(ctx context.Context, apiKey, ip string) (context.Context, error) {
	if apiKey == "" {
		return nil, ErrMcpCredential
	}
	key, err := u.keys.GetAPIKeyByKeyID(ctx, nodeauth.KeyID(apiKey))
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrMcpCredential
		}
		return nil, ErrSystem(err)
	}
	if subtle.ConstantTimeCompare([]byte(key.APIKey), []byte(apiKey)) != 1 {
		return nil, ErrMcpCredential
	}
	now := time.Now()
	if apiKeyExpired(key, now) {
		return nil, ErrNodeAPIKeyExpired
	}
	owner, err := u.users.FindByID(ctx, key.UserID)
	if err != nil || owner.Status != user.StatusActive {
		return nil, ErrNodeAPIKeyOwnerInactive
	}
	u.touch(key, now, ip)
	p := mcpPrincipal(key)
	p.Username = owner.Username
	return context.WithValue(auth.NewContext(ctx, p.Auth()), mcpCallerKey{}, p), nil
}

// touch 异步记录最近一次使用的时间与来源 IP（节流，与节点调用共用间隔）。
func (u *McpUsecase) touch(key *gen.UserAPIKey, now time.Time, ip string) {
	if key.LastUsedAt != nil && now.Sub(*key.LastUsedAt) < apiKeyTouchInterval && key.LastUsedIP == ip {
		return
	}
	id := key.ID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = u.keys.TouchAPIKeyUsed(ctx, id, now, ip)
	}()
}

// Authorize 这次请求的凭证能不能调用要 perm 的工具；不能时 400。
func (u *McpUsecase) Authorize(ctx context.Context, perm constant.Permissions) error {
	p, ok := ctx.Value(mcpCallerKey{}).(*NodePrincipal)
	if !ok {
		return ErrMcpCredential
	}
	if err := u.node.Authorize(ctx, p, perm); err != nil {
		return ErrMcpPermissionDenied(perm)
	}
	return nil
}

// McpCredential 一把能当 MCP 凭证的 API Key，和它现在能用的权限。
type McpCredential struct {
	Key     *gen.UserAPIKey
	Granted map[constant.Permissions]bool
}

// Credentials 当前用户的 API Key 里授予了 perms 中任一项的；每把 Key 现在能用哪些权限，和调用工具时是同一个判断。
func (u *McpUsecase) Credentials(ctx context.Context, perms []constant.Permissions) ([]*McpCredential, error) {
	caller, ok := auth.FromContext(ctx)
	if !ok {
		return nil, ErrNoPermission
	}
	keys, err := u.repo.ListMcpCredentials(ctx, caller.UserID, perms)
	if err != nil {
		return nil, ErrSystem(err)
	}
	now := time.Now()
	out := make([]*McpCredential, 0, len(keys))
	for _, key := range keys {
		c := &McpCredential{Key: key, Granted: map[constant.Permissions]bool{}}
		if !apiKeyExpired(key, now) {
			p := mcpPrincipal(key)
			for _, perm := range perms {
				c.Granted[perm] = u.node.Authorize(ctx, p, perm) == nil
			}
		}
		out = append(out, c)
	}
	return out, nil
}
