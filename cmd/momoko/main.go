package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"momoko/internal/conf"
	"momoko/internal/initialize"
	"momoko/internal/server"
	"momoko/pkg/auth"
	"momoko/pkg/selfupdate"
	"momoko/pkg/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport"
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
	// showVersion：-version 只打印版本号就退出（在线更新安装前用它确认新程序能在本机运行）。
	showVersion bool

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs", "config path, eg: -conf config.yaml")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, ps *server.PassportServer) *kratos.App {
	servers := []transport.Server{gs, hs}
	// 用户通行证没启用时没有这个 server：空指针放进去，启动时取它的地址会崩
	if ps != nil {
		servers = append(servers, ps)
	}
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(servers...),
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
	// 读配置之前就退出：不碰数据、不开端口
	if showVersion {
		fmt.Println(selfupdate.VersionLine(Name, version.Current()))
		return
	}
	// Windows：用户启动的这个进程当守护进程，真正的服务在子进程里跑，在线更新时由它重启（见 pkg/selfupdate/supervise.go）
	if !devBuild && selfupdate.ShouldSupervise() {
		os.Exit(selfupdate.Supervise())
	}
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

	app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Passport, flagconf, logger)
	if err != nil {
		panic(err)
	}
	// 在线更新替换好程序后：像收到停止信号一样正常停服，再换成新程序
	go func() {
		<-selfupdate.RestartRequested()
		log.Info("程序已更新，正在重启")
		if err := app.Stop(); err != nil {
			log.Errorf("stop server for update failed: %v", err)
		}
	}()

	// start and wait for stop signal
	runErr := app.Run()
	cleanup()
	if selfupdate.RestartPending() {
		// 成功时不会返回（Linux / macOS 原地 exec，Windows 启动新进程后退出）
		if err := selfupdate.Relaunch(); err != nil {
			log.Errorf("重启失败：%v", err)
			os.Exit(1)
		}
	}
	if runErr != nil {
		panic(runErr)
	}
}
