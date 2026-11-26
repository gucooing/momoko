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
	"momoko/pkg/sysservice"
	"momoko/pkg/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	"go.uber.org/automaxprocs/maxprocs"
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
	version.Set(Version)
	// momoko install / status / … 这些子命令（见 cli.go）
	if len(os.Args) > 1 && isCommand(os.Args[1]) {
		os.Exit(runCommand(os.Args[1], os.Args[2:]))
	}
	flag.Usage = func() { usage(os.Stderr) }
	flag.Parse()
	// 读配置之前就退出：不碰数据、不开端口
	if showVersion {
		fmt.Println(selfupdate.VersionLine(Name, version.Current()))
		return
	}
	// 作为系统服务运行（momoko install 装好的）：在安装目录里跑；在线更新换好程序后退出，由服务管理器拉起新程序
	if sysservice.Managed() {
		dir, err := sysservice.EnterInstallDir()
		if err != nil {
			panic(fmt.Sprintf("进入安装目录失败：%v", err))
		}
		// 初始化标记在包初始化时按启动时的工作目录读过一次（Windows 服务是 System32），换了目录要重读
		initialize.Re()
		sysservice.PrepareServiceProcess(dir)
		// 全局日志也写到（可能换过的）标准输出：Windows 服务的输出改到了日志文件
		log.SetLogger(log.NewStdLogger(os.Stdout))
		os.Exit(sysservice.Run(serve))
	}
	// 直接运行了已安装的那份（双击、终端里敲 momoko）：服务已经在后台跑着，不再起第二份
	if _, ok := sysservice.Installed(); ok {
		os.Exit(installedHint())
	}
	// 没安装：和普通程序一样在当前目录前台运行（有控制台窗口，Ctrl+C 停止）
	os.Exit(serve(nil))
}

func newLogger() log.Logger {
	return log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
}

// loadConfig 读配置（初始化会改写配置文件，每次启动服务前重新读）。
func loadConfig() (*conf.Bootstrap, func()) {
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	if err := c.Load(); err != nil {
		panic(err)
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	return &bc, func() { _ = c.Close() }
}

// stopOn 收到 stop（系统服务要求停止）或 also 时停掉 app。stop / also 为 nil 时只等另一个。
func stopOn(app *kratos.App, stop <-chan struct{}, also <-chan struct{}, onAlso func()) {
	go func() {
		select {
		case <-stop:
		case <-also:
			if onAlso != nil {
				onAlso()
			}
		}
		if err := app.Stop(); err != nil {
			log.Errorf("stop server failed: %v", err)
		}
	}()
}

// serve 运行面板直到停止，返回进程退出码。stop 关闭时（系统服务要求停止）正常停服；
// 没初始化时先跑初始化向导，初始化完再启动完整服务。
// 在线更新换好程序后返回 selfupdate.RestartExitCode，由服务管理器（systemd / launchd / Windows 服务）重新拉起新程序。
func serve(stop <-chan struct{}) int {
	// 按容器的 CPU 配额设 GOMAXPROCS（只在真正运行服务时设，命令行子命令不用打印这行日志）
	_, _ = maxprocs.Set()
	logger := newLogger()

	if !initialize.IsInitialized() {
		bc, closeConf := loadConfig()
		auth.AuthSecretKey = bc.GetAuth().Secret
		app := wireInitializeApp(bc.Server, flagconf, logger)
		stopOn(app, stop, initialize.RestartRequested(), nil)
		err := app.Run()
		closeConf()
		if err != nil {
			panic(err)
		}
		// 不是因为初始化完成而停下的（Ctrl+C、服务停止）：直接退出
		if !initialize.IsInitialized() {
			return 0
		}
	}

	bc, closeConf := loadConfig()
	defer closeConf()
	auth.AuthSecretKey = bc.GetAuth().Secret
	// 启动完整服务前强校验主密钥强度：弱/默认/公开密钥直接拒绝启动（fail-closed）。
	// 全新部署经初始化向导会自动写入高强度随机密钥，不会触发此处。
	if err := auth.ValidateSecret(auth.AuthSecretKey); err != nil {
		panic(fmt.Sprintf("不安全的 auth.secret 配置: %v（请在 configs/config.yaml 设置高强度随机密钥后重启）", err))
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Passport, flagconf, logger)
	if err != nil {
		panic(err)
	}
	// 在线更新替换好程序后：像收到停止信号一样正常停服，再退出让服务管理器拉起新程序
	stopOn(app, stop, selfupdate.RestartRequested(), func() { log.Info("程序已更新，正在重启") })

	// start and wait for stop signal
	runErr := app.Run()
	cleanup()
	if selfupdate.RestartPending() {
		return selfupdate.RestartExitCode
	}
	if runErr != nil {
		panic(runErr)
	}
	return 0
}
