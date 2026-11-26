package initialize

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"

	"momoko/pkg/utils"
)

// DefaultHTTPPort 后台默认端口。
const DefaultHTTPPort = 22633

// defaultConfig 安装成系统服务、目录里还没有配置时写入的配置（和发行包里的 configs/config.yaml 一样）。
// 数据库和管理员在第一次打开网页时的初始化里设置，初始化完会重新生成主密钥。
const defaultConfig = `server:
    http:
        addr: 0.0.0.0:%d
        timeout: 15s
    grpc:
        addr: 0.0.0.0:22733
        timeout: 15s
passport:
    # 用户通行证（玩家端）独立端口；后台接口在这个端口上不存在
    enabled: true
    addr: 0.0.0.0:22634
    timeout: 15s
    # 对外地址：玩家与游戏客户端访问通行证用的地址（如 https://passport.example.com），在后台的通行证设置里展示；
    # 客户端（启动器、APK）浏览器授权的授权页地址由它拼出，不填时客户端无法发起授权
    public_url: ""
data:
    database:
        driver: sqlite3
        source: file:./data/momoko.db?_pragma=foreign_keys(1)
auth:
    # JWT 签名与敏感数据加密的主密钥，必须为高强度随机字符串（建议 ≥32 字符）。
    # 留空或使用弱/默认密钥将导致服务拒绝启动；初始化时会重新生成。
    secret: %s
`

// WriteDefaultConfig 在 configPath（目录或文件）写入默认配置，后台端口为 port。
func WriteDefaultConfig(configPath string, port int) error {
	file, err := resolveConfigFile(configPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(defaultConfig, port, utils.GenerateRandomString(32))
	return os.WriteFile(file, []byte(content), 0o600)
}

// Ports 配置里的几个监听端口（没配时为 0）。
type Ports struct {
	HTTP            int
	GRPC            int
	Passport        int
	PassportEnabled bool
}

// ConfigFile 配置文件路径（configPath 可以是目录或文件）。
func ConfigFile(configPath string) (string, error) { return resolveConfigFile(configPath) }

// ReadPorts 读配置里的监听端口。
func ReadPorts(configPath string) (Ports, error) {
	file, err := resolveConfigFile(configPath)
	if err != nil {
		return Ports{}, err
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return Ports{}, err
	}
	var c struct {
		Server struct {
			HTTP struct {
				Addr string `yaml:"addr"`
			} `yaml:"http"`
			GRPC struct {
				Addr string `yaml:"addr"`
			} `yaml:"grpc"`
		} `yaml:"server"`
		Passport struct {
			Enabled bool   `yaml:"enabled"`
			Addr    string `yaml:"addr"`
		} `yaml:"passport"`
	}
	if err := yaml.Unmarshal(content, &c); err != nil {
		return Ports{}, fmt.Errorf("读取 %s 失败：%w", file, err)
	}
	return Ports{
		HTTP:            addrPort(c.Server.HTTP.Addr),
		GRPC:            addrPort(c.Server.GRPC.Addr),
		Passport:        addrPort(c.Passport.Addr),
		PassportEnabled: c.Passport.Enabled,
	}, nil
}

func addrPort(addr string) int {
	_, p, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(p)
	return n
}

// SetHTTPPort 把配置里后台的监听端口改成 port（地址部分不变，其它内容原样保留）。
func SetHTTPPort(configPath string, port int) error {
	file, err := resolveConfigFile(configPath)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return errors.New("配置文件格式不对：最外层应该是映射")
	}
	rootMap := root.Content[0]
	server := mappingValue(rootMap, "server")
	if server == nil {
		server = addMapping(rootMap, "server")
	}
	httpNode := mappingValue(server, "http")
	if httpNode == nil {
		httpNode = addMapping(server, "http")
	}
	host := "0.0.0.0"
	if cur := mappingValue(httpNode, "addr"); cur != nil {
		if h, _, err := net.SplitHostPort(cur.Value); err == nil {
			host = h
		}
	}
	setScalar(httpNode, "addr", net.JoinHostPort(host, strconv.Itoa(port)))
	next, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}
	return os.WriteFile(file, next, 0o600)
}
