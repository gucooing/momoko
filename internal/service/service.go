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
	NewOperationLogMiddleware,
)
