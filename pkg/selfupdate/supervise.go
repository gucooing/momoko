package selfupdate

import "os"

// Windows 没有 exec，不能原地换成新程序。所以在 Windows 上 momoko 启动后分成两个进程：
// 外面一个很小的守护进程（就是用户启动的那个：双击、cmd、服务包装器、开机脚本看到的都是它），
// 里面一个真正干活的子进程。更新时子进程停服后以 RestartExitCode 退出，守护进程马上启动换好的新程序；
// 守护进程自己一直不退出，所以 cmd 不会提前回到提示符，服务包装器和循环启动脚本也不会再拉起第二个。
// Linux / macOS 用 exec 原地重启（进程号不变），不需要守护进程。

const (
	// supervisedEnv 守护进程启动子进程时带上，子进程据此知道更新后该退出等守护进程重启它。
	supervisedEnv = "MOMOKO_SUPERVISED"
	// RestartExitCode 子进程换好程序后的退出码：守护进程看到它就重新启动，其它退出码原样退出。
	RestartExitCode = 75
)

// supervised 当前进程是不是守护进程启动的子进程。读完就从环境变量里去掉，
// 免得它再启动的程序（终端、Agent、托管的应用）继承过去。
var supervised = func() bool {
	on := os.Getenv(supervisedEnv) == "1"
	_ = os.Unsetenv(supervisedEnv)
	return on
}()
