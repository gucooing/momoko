package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewAuthService,
	NewFileService,
	NewUserService,
	NewSystemService,
	NewInstanceService,
	NewOpenSSHService,
	NewNodeService,
	NewNetworkService,
	NewTunnelService,
	NewOperationLogMiddleware,
	NewInitializeService,
	NewDockerService,
	NewSub2APIService,
	NewImageGenService,
)
