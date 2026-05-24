//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"momoko/internal/biz"
	"momoko/internal/conf"
	"momoko/internal/data"
	"momoko/internal/server"
	"momoko/internal/service"
	"momoko/pkg/avatar"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, string, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, avatar.NewManager, newApp))
}

func wireInitializeApp(*conf.Server, string, log.Logger) *kratos.App {
	panic(wire.Build(data.NewInitializeRepo, biz.NewInitializeUsecase, service.NewInitializeService, server.NewInitializeHTTPServer, newInitializeApp))
}
