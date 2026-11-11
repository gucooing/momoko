package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewAuthService,
	NewFileService,
	NewTaskService,
	NewUserService,
	NewSystemService,
	NewInstanceService,
	NewOpenSSHService,
	NewNodeService,
	NewNodeAgentService,
	NewNetworkService,
	NewTunnelService,
	NewOperationLogMiddleware,
	NewInitializeService,
	NewDockerService,
	NewIQCheckService,
	NewOIDCService,
	NewMonitorService,
	NewThemeService,
)
