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
	NewNetworkUsecase,
	NewTunnelUsecase,
	NewOperationLogUsecase,
	NewInitializeUsecase,
	NewDockerUsecase,
	NewOIDCUsecase,
)
