package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewAuthUsecase,
	NewConfigUsecase,
	NewFileUsecase,
	NewTaskManager,
	NewTaskUsecase,
	NewUserUsecase,
	NewSystemUsecase,
	NewInstanceUsecase,
	NewOpenSSHUsecase,
	NewNodeUsecase,
	NewNodeAgentUsecase,
	NewNetworkUsecase,
	NewTunnelUsecase,
	NewOperationLogUsecase,
	NewInitializeUsecase,
	NewDockerUsecase,
	NewIQCheckUsecase,
	NewOIDCUsecase,
	NewMonitorUsecase,
	NewThemeUsecase,
	NewGameServerUsecase,
	NewGameHostingUsecase,
	NewGmUsecase,
	NewGameOpsUsecase,
	NewPassportUsecase,
)
