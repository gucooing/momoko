package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewAuthUsecase,
	NewConfigUsecase,
	NewFileUsecase,
	NewUserUsecase,
	NewSystemUsecase,
	NewInstanceUsecase,
	NewOpenSSHUsecase,
	NewNodeUsecase,
	NewNetworkUsecase,
	NewOperationLogUsecase,
	NewInitializeUsecase,
	NewDockerUsecase,
	NewSub2APIUsecase,
	NewImageGenUsecase,
)
