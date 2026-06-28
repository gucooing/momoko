package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"momoko/internal/conf"
	"momoko/internal/initialize"
	"momoko/pkg/auth"
	"momoko/pkg/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func newInitializeApp(logger log.Logger, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{
			"mode": "initialize",
		}),
		kratos.Logger(logger),
		kratos.StopTimeout(5*time.Second),
		kratos.Server(hs),
	)
}

func main() {
	flag.Parse()
	version.Set(Version)
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	auth.AuthSecretKey = bc.GetAuth().Secret

	if !initialize.IsInitialized() {
		app := wireInitializeApp(bc.Server, flagconf, logger)
		go func() {
			<-initialize.RestartRequested()
			if err := app.Stop(); err != nil {
				log.Errorf("stop initialize server failed: %v", err)
			}
		}()
		if err := app.Run(); err != nil {
			panic(err)
		}
		main()
	}

	// 启动完整服务前强校验主密钥强度：弱/默认/公开密钥直接拒绝启动（fail-closed）。
	// 全新部署经初始化向导会自动写入高强度随机密钥，不会触发此处。
	if err := auth.ValidateSecret(auth.AuthSecretKey); err != nil {
		panic(fmt.Sprintf("不安全的 auth.secret 配置: %v（请在 configs/config.yaml 设置高强度随机密钥后重启）", err))
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, flagconf, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
