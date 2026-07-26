package file

import (
	"os"
	"sync"

	"momoko/pkg/localfs"
)

// momoko 自身在工作目录下使用的固定目录。
const (
	// ServersPath 实例工作目录的默认根。
	ServersPath = "./servers"
	// DataDir 运行数据目录：内含 SQLite 库、初始化标记、头像与生图存储。
	DataDir = "./data"
	// ConfigDir 配置目录：内含 auth.secret 等密钥材料。
	ConfigDir = "./configs"

	// TempDir 是 momoko 的临时目录，位于 DataDir 之下，因而天然落在保护清单内。
	TempDir = DataDir + "/tmp"
	// UploadTempDir 是「缓冲型远端来源」(FTP/WebDAV) 的分片缓冲目录。
	//
	// 只有远端来源用它：那类上传收尾本就要把缓冲整流推送到远端，缓冲放哪儿都得读一遍，
	// 集中存放没有额外代价。本地来源的缓冲则必须与目标同目录——收尾的 rename 要同卷才瞬时，
	// 挪到这里会退化成一次全量复制。本地缓冲改由列表隐藏解决观感问题（见 localfs.BufferPrefix）。
	UploadTempDir = TempDir + "/uploads"
)

// MaxEditSize 在线编辑器读写单个文件的体积上限。
const MaxEditSize = localfs.DefaultMaxFileSize

// ProtectedPaths 返回必须对文件管理器完全隐藏的 momoko 自身路径。
//
// 这是纵深防御的一环：configs/ 里的 auth.secret 一旦泄露，攻击者就能伪造 JWT 与
// 全部预签名 URL；data/ 里是 SQLite 库（含会话、令牌、来源凭据密文）。
// 虽然拥有文件管理权限的用户通常也有终端权限，但把这两处封死可以确保
// 「分享链接、预签名下载、实例文件管理」这些低权限入口即便出现逻辑缺陷也够不到密钥。
// 结果只算一次：来源在每个请求上都会重建，而这里要做 os.Executable 与若干次
// EvalSymlinks，逐请求重算纯属浪费；进程生命周期内工作目录与可执行文件不会变。
var ProtectedPaths = sync.OnceValue(func() []string {
	paths := []string{DataDir, ConfigDir}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, exe)
	}
	return paths
})

// systemOptions 同理只构造一次（Deny 内部已把路径规范化好）。
var systemOptions = sync.OnceValue(func() []localfs.Option {
	return []localfs.Option{
		localfs.Deny(ProtectedPaths()...),
		localfs.WithMaxFileSize(MaxEditSize),
	}
})

// UploadTemp 返回缓冲型远端来源共用的分片缓冲视图（首次调用时创建目录）。
// 做成进程内单例，免得每个 usecase 各自穿线。
var UploadTemp = sync.OnceValues(func() (*localfs.FS, error) {
	return localfs.OpenDir(UploadTempDir)
})

// SystemStoreOptions 返回整机来源应有的策略：保护清单 + 编辑体积上限。
func SystemStoreOptions() []localfs.Option { return systemOptions() }

// ScopedStoreOptions 返回受限来源（实例目录等）应有的策略。
// 受限根本身已经把范围锁死，这里仍带上保护清单以防某个根恰好覆盖了 data/ 或 configs/。
func ScopedStoreOptions() []localfs.Option {
	return SystemStoreOptions()
}
