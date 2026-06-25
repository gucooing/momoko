# 内网穿透（frp）集成计划书

> 状态：设计稿 · 2026-06-25
> 调研基准：frp `github.com/fatedier/frp` v0.69.1（已对照源码核实）

## 1. 目标与范围

在「工具」菜单下新增「内网穿透」子菜单，基于 frp 实现：momoko 内嵌 frps（服务端），
允许**官方 frpc 二进制**直接连入；momoko 负责认证与数据统计，隧道本身由 frpc 声明驱动。

**在范围内**
- 内嵌 frps（进程内 `server.NewService` + `Run`），不开 frp 自带 web 面板。
- momoko 通过 frp 服务端插件接管 `Login` / `NewProxy` 认证。
- momoko 通过 `pkg/metrics/mem.StatsCollector` 在进程内读取统计，定时落库（与端口转发一致）。
- 隧道规则 CRUD、frpc 配置生成与下载、实时状态与流量时序展示。

**不在范围内**
- 多 frps 实例 / 每租户独立 frps（本期单实例；`mem.StatsCollector` 为全局单例，多实例需命名空间）。
- 自定义控制循环（路 B：momoko 预建 proxy + 自写 frpc 协议握手）。留作后续评估。
- 嵌入 frpc 客户端（暴露 momoko 本地服务，与「官方 frpc 连入」用途不同）。

## 2. 可行性结论（摘要）

- frp 无 `internal/` 目录，`server` 包及 `pkg/*` 原语均可导入。Go 1.25 要求，momoko `go 1.26.3` 满足。License Apache-2.0。
- frps 生命周期入口：`server.NewService(cfg *v1.ServerConfig) (*Service, error)` → `svr.Run(ctx)` → `svr.Close()`。配置为纯内存 struct，`Complete()` 填默认值，无需 cobra/flag/配置文件。
- 认证接管：`pkg/plugin/server` 提供 `OpLogin`/`OpNewProxy`/`OpCloseProxy`/`OpNewWorkConn`/`OpNewUserConn` 钩子；`cfg.HTTPPlugins` 让 frps 把事件 POST 到 momoko 的 HTTP 端点，momoko 回 `Response{Reject:true, RejectReason:...}` 即拒绝。
- 统计接管：`pkg/metrics/mem.StatsCollector`（全局）暴露 `GetServer()` / `GetProxiesByType(type)` / `GetProxyTraffic(name)` / `GetProxyByName(name)` / `ClearOfflineProxies()`，进程内直接调用，无需 HTTP 面板。
- **frps 被动**：隧道监听由 frpc 发 `NewProxy` 且 momoko 插件放行后才开；无法「点添加即开 listener」（与端口转发不同，UI 需表达「待连接/在线」状态）。

## 3. 总体架构

```
              ┌─────────────────── momoko 进程 ───────────────────┐
              │                                                   │
   frpc ─────▶│  frps (server.NewService+Run, 无 WebServer)       │
              │     │  控制面: Login/NewProxy/WorkConn/Ping ...    │
              │     │  数据面: proxy.NewProxy+Run (frp 内部)       │
              │     │                                             │
              │     ▼ cfg.HTTPPlugins (HTTP 回调)                  │
              │  TunnelPlugin HTTP 端点                            │
              │   ├─ Login    → 查 frp_tunnel: user+credential     │
              │   └─ NewProxy → 查 frp_tunnel: user+proxy 规则匹配 │
              │                                                   │
              │  TunnelUsecase.sampleLoop                          │
              │   └─ mem.StatsCollector.* → frp_tunnel_stat (ent)  │
              │                                                   │
              │  TunnelService (HTTP/gRPC) ◀── 前端 CRUD/统计/下载 │
              └───────────────────────────────────────────────────┘
```

数据流：
1. 用户在 momoko 建隧道规则（type/remote_port/allowed_user/credential/local_*）→ `frp_tunnel` 入库。
2. momoko 按规则生成 `frpc.toml`（含 server 地址、`user`、`metas.credential`、`[[proxies]]`；无统一 token）供下载。
3. frpc 用该配置连 frps。frps 在 `Login`/`NewProxy` 时 POST momoko 插件端点。
4. momoko 查 `frp_tunnel` 校验：是否 momoko 创建、认证是否匹配、proxy 是否与规则一致 → 放行/拒绝。
5. 放行后 frps 开公共监听，流量起来；`mem.StatsCollector` 累计。
6. momoko `sampleLoop` 周期采样 → `frp_tunnel_stat`；`cleanupLoop` 按保留期清理。

## 4. 认证模型（每隧道独立，无统一 token）

每条隧道一套独立凭证，互不相关。frps 自身的 token 校验**中性化**（`Auth.Method=token` + 空 token，等价 `pkg/auth.AlwaysPassVerifier` 的 pass-through 语义），**不作为统一认证**；唯一认证权威是 momoko 插件，按每条隧道的 `credential` 独立判断。

| 角色 | 验证方 | 凭证 | 作用 |
|---|---|---|---|
| frps 握手 | frps 自身（中性化，空 token） | 无（空 token，仅完成协议握手） | 不拦截；真正认证交给插件 |
| 隧道认证 | momoko 插件 `Login` | 每隧道 `credential`（frpc `metas.credential`） | 这条隧道是否在 momoko 建过、credential 是否匹配 |
| 隧道授权 | momoko 插件 `NewProxy` | 规则匹配（user+name+type+remotePort/域名） | 声明的 proxy 是否与规则一致 |

插件契约（`pkg/plugin/server/types.go`）：
```go
type Request  struct { Op string; Content any }
type Response struct { Reject bool; RejectReason string; Unchange bool; Content any }
type UserInfo struct { User string; Metas map[string]string; RunID string }
type NewProxyContent  struct { User UserInfo; /* + proxy 配置 name/type/remotePort... */ }
type NewUserConnContent struct { User UserInfo; ProxyName, ProxyType, RemoteAddr string }
```

- `Login`：取 `UserInfo.User` + `UserInfo.Metas["credential"]`，查 `frp_tunnel` 中该 user 的规则，比对 credential；不符 → `Reject`。
- `NewProxy`：取 `UserInfo.User` + proxy 的 name/type/remotePort（或 customDomains/subdomain），查是否存在匹配且 `is_enable` 的规则；不符 → `Reject`。
- 可选 `NewUserConn`：最终用户连入时做细粒度 ACL 或连接计数。
- 插件端点为服务间调用，走 public-route 豁免（参见现有公开路由豁免约定），但加独立 path token 防外部调用。

## 5. 模块拆分

| 层 | 文件 | 职责 | 参照 |
|---|---|---|---|
| pkg | `pkg/frp_tunnel/service.go` | 封装 `server.NewService`+`Run(ctx)`+`Close`；构造 `v1.ServerConfig`（`WebServer` 空、`HTTPPlugins` 指向 momoko 端点）；持有 `mem.StatsCollector` 读取入口 | `pkg/port_forward` 风格 |
| pkg | `pkg/frp_tunnel/plugin.go` | 插件 HTTP handler：解析 `Request`，调 biz 校验，回 `Response` | — |
| pkg | `pkg/frp_tunnel/stats.go` | 封装 `mem.StatsCollector` 采样为 `SnapshotAll()` 风格 | `port_forward.Manager.SnapshotAll` |
| ent | `internal/data/ent/schema/frp_tunnel.go` | 隧道规则表 | `port_forward.go` |
| ent | `internal/data/ent/schema/frp_tunnel_stat.go` | 统计采样表 | `port_forward_stat.go` |
| data | `internal/data/tunnel.go` | `TunnelRepo`（ent 查询 + stat 落库） | `internal/data/network.go` |
| data/biz | 现有 `internal/biz/config.go`（ConfigRepo/ConfigUsecase） | 增 `FrpsConfig`/`UpdateFrpsConfig`，frps 实例配置存 `configs` 表 | `DockerConfig` 同构 |
| biz | `internal/biz/tunnel.go` | `TunnelUsecase`：CRUD + `sampleLoop`/`cleanupLoop` + 插件校验回调 + frpc 配置生成 | `internal/biz/network.go` |
| service | `internal/service/tunnel.go` | HTTP/gRPC handler → biz | `internal/service/network.go` |
| proto | `api/proto/v1/tunnel.proto` | `service TunnelManager` + 消息 | `network.proto` |
| 前端 | `frontend/src/views/tools/tunnel/index.vue` 等 | 列表/CRUD/统计/下载 | `views/tools/port-forward/` |

> 命名待定：`tunnel` 还是 `intranet` 还是 `frp_tunnel`。下文统一用 `tunnel`，DB/ent 用 `frp_tunnel`。

## 6. 数据模型

### 6.1 `frp_tunnel`（规则）

```go
field.String("id").NotEmpty().Unique()
field.String("name").NotEmpty().Unique().Comment("frp proxy name，frpc 声明需一致")
field.String("user_id").NotEmpty().Comment("所属用户")
field.Enum("proxy_type").Values("tcp","udp","http","https","stcp","xtcp","tcpmux")
field.Int("remote_port").Default(0).Comment("tcp/udp/tcpmux 公共端口")
field.String("custom_domains").Default("").Comment("http/https 自定义域名，逗号分隔")
field.String("subdomain").Default("").Comment("http/https 子域名")
field.String("local_ip").Default("127.0.0.1").Comment("frpc 端本地地址（生成 frpc 配置用）")
field.Int("local_port").Default(0).Comment("frpc 端本地端口")
field.String("credential").NotEmpty().Comment("每隧道认证，置于 frpc metas.credential")
field.String("allow_users").Default("").Comment("允许的 frpc user，逗号分隔；空=规则 owner")
field.Bool("is_enable").Default(false)
// index: user_id, is_enable, name(unique)
// mixin: TimeMixin
// edge: -> User (user_id)
```

### 6.2 `frp_tunnel_stat`（统计采样，追加 only）

与 `port_forward_stat` 同构：
```go
field.String("frp_tunnel_id").NotEmpty().Comment("所属隧道id")
field.Time("sample_time")
field.Int64("active_connections").Default(0)
field.Int64("bytes_in").Default(0)   // 累计入站，前端按差值算速率
field.Int64("bytes_out").Default(0)  // 累计出站
// index: (frp_tunnel_id, sample_time), (sample_time)
```

### 6.3 frps 实例配置（存入 `config` 表，非隧道规则）

复用现有 `SystemConfig`（key/value，表名 `configs`）模式，与 `LoginConfig`/`EmailConfig`/`DockerConfig` 同构：新增一个 `common.ConfigKey`（如 `ConfigKeyFrps`），value 存 `FrpsConfig` 的 JSON。这是 **frps 实例级**配置（是否启用、认证端口等），与 `frp_tunnel` 里的逐条隧道规则无关。

proto 消息（加到 `system.proto`，与其它配置并列）：
```proto
message FrpsConfig {
  bool is_enable = 1;              // 是否启用 frps
  string bind_addr = 2;            // 监听地址，默认 0.0.0.0
  int32 bind_port = 3;             // frpc 连接与认证端口（frps 主端口），默认 7000
  int32 vhost_http_port = 4;       // http 代理 vhost 端口，0=不监听
  int32 vhost_https_port = 5;      // https 代理 vhost 端口，0=不监听
  int32 kcp_bind_port = 6;         // kcp 传输端口，0=不监听
  int32 quic_bind_port = 7;        // quic 传输端口，0=不监听
  bool tls_only = 8;               // 是否强制 TLS
  string plugin_path_token = 9;    // 插件回调端点的 path token（防外部调用）
  int32 stat_sample_interval = 10; // 统计采样间隔(秒)
  int32 stat_retention = 11;       // 统计保留天数
}
// 注：无 auth_token 字段。frps 自身 Auth 中性化（空 token），不作为统一认证；
// 每隧道独立凭证存于 frp_tunnel.credential，由插件 Login 校验（见 §4）。
```

落点：
- `ConfigRepo` 增 `FrpsConfig(ctx) (*v1.FrpsConfig, error)` + `UpdateFrpsConfig(ctx, *v1.FrpsConfig) (*v1.FrpsConfig, error)`（仿 `DockerConfig`/`UpdateDockerConfig`）。
- `ConfigUsecase` 暴露同名方法 + 校验（端口范围、`is_enable=true` 时 `bind_port`>0 且 `auth_token` 非空等）。
- `system.proto` 的系统服务增 `GetFrpsConfig`/`UpdateFrpsConfig` HTTP RPC，供管理界面编辑。
- `pkg/frp_tunnel` 启动时读 `FrpsConfig`：`is_enable=false` → 不起 frps；否则装配 `v1.ServerConfig`（`Complete()` 填默认、`WebServer` 空、`Auth.Method="token"`+`Auth.Token=""` 中性化、`HTTPPlugins` 指向 momoko 插件端点）。
- 热切换：`is_enable` false→true 触发启动，true→false 触发 `Close`（可选；首期可要求重启进程，降低复杂度）。

## 7. Proto API（`api/proto/v1/tunnel.proto`）

```proto
service TunnelManager {
  rpc ListTunnels(ListTunnelsRequest) returns (ListTunnelsResponse) {
    option (google.api.http) = { get: "/api/v1/tunnel/tunnels" };
  }
  rpc CreateTunnel(CreateTunnelRequest) returns (CreateTunnelResponse) {
    option (google.api.http) = { post: "/api/v1/tunnel/tunnels" body: "*" };
  }
  rpc GetTunnel(GetTunnelRequest) returns (GetTunnelResponse) {
    option (google.api.http) = { get: "/api/v1/tunnel/tunnels/{id}" };
  }
  rpc UpdateTunnel(UpdateTunnelRequest) returns (UpdateTunnelResponse) {
    option (google.api.http) = { put: "/api/v1/tunnel/tunnels/{id}" body: "*" };
  }
  rpc DeleteTunnel(DeleteTunnelRequest) returns (DeleteTunnelResponse) {
    option (google.api.http) = { delete: "/api/v1/tunnel/tunnels/{id}" };
  }
  rpc GetTunnelStats(GetTunnelStatsRequest) returns (GetTunnelStatsResponse) {
    option (google.api.http) = { get: "/api/v1/tunnel/tunnels/{id}/stats" };
  }
  rpc GetTunnelFrpcConfig(GetTunnelFrpcConfigRequest) returns (GetTunnelFrpcConfigResponse) {
    option (google.api.http) = { get: "/api/v1/tunnel/tunnels/{id}/frpc-config" };
  }
}
```
消息：`TunnelInfo`、`TunnelStat`、`TunnelStatPoint`、`TunnelType` enum，结构对齐 `PortForwardInfo`/`PortForwardStat`/`PortForwardStatPoint`。`GetTunnelFrpcConfigResponse` 返回完整 `frpc.toml` 文本。

## 8. 统计与持久化

- 采样源：`mem.StatsCollector`（全局单例，需 blank import `_ "github.com/fatedier/frp/pkg/metrics"` 注册）。
  - `GetProxiesByType(type)` → 各 proxy 当前流量/连接。
  - `GetProxyTraffic(name)` → 单 proxy `TrafficIn/TrafficOut` 时序（frp 内存按天保留）。
  - `GetServer()` → 服务端汇总（总流量、在线客户端数、各类型 proxy 计数）。
- `TunnelUsecase.sampleLoop(ctx)`：按 `stat_sample_interval` tick → 遍历启用的 `frp_tunnel` → 读 `mem.StatsCollector` → 组 `FrpTunnelStatSample` → `repo.SaveFrpTunnelStats`。完全复刻 `NetworkUsecase.sampleLoop`。
- `cleanupLoop` / `cleanupExpiredStats`：按 `stat_retention` 删旧采样点（复刻 `DeletePortForwardStatsBefore`）。
- 重启后历史保留（落库即持久），`mem.StatsCollector` 内存值丢失不影响历史。
- 状态字段：规则需暴露「待连接 / 在线 / 离线」——通过 `registry.ClientRegistry` 或 `mem.StatsCollector.GetProxyByName(name)` 是否存在判定。

## 9. frps 生命周期

- `pkg/frp_tunnel.NewService(cfg)`：从 `FrpsConfig`（`config` 表）装配 `v1.ServerConfig`（`Complete()` 填默认、`Auth` 中性化为空 token），`server.NewService` → `go svr.Run(ctx)`；ctx 取消 → `svr.Close()`。`FrpsConfig.is_enable=false` 时不启动。
- `cfg.WebServer` 留空（无 frp 面板）；`cfg.HTTPPlugins` 配 `Login`+`NewProxy`（+可选 `NewUserConn`/`CloseProxy`）指向 momoko 插件端点。
- 依赖：`go.mod` 引入 `github.com/fatedier/frp v0.69.1`；blank import `_ "github.com/fatedier/frp/pkg/metrics"`（统计注册）；`_ "github.com/fatedier/frp/web/frps"` 仅在需要面板时引入（本期不需要，可省）。
- 注：`server` 包 `init()` 会设全局 `crypto.DefaultSalt="frp"`；与 momoko 别处若用同包需注意。
- 与 momoko 进程同生命周期，由 wire 注入启动；frps 监听端口需在 momoko 已有端口规划中避开冲突。

## 10. 前端

- 菜单：DB menu seed（参照现有 tools 子菜单），i18n 命名空间 `tools.tunnel.*`（`frontend/src/locales/messages.ts`）。
- 页面 `frontend/src/views/tools/tunnel/`：
  - `index.vue`：列表 + 查询（类型/启用状态）+ CRUD（参照 `port-forward/index.vue`）。
  - `TunnelStatsDialog.vue`：实时快照 + 时序图（参照 `PortForwardStatsDialog.vue`）。
  - `frpc-config` 下载按钮：调 `GetTunnelFrpcConfig`，Blob 下载 `frpc.toml`。
- 状态展示：「待连接 / 在线 / 离线」+ 当前流量。

## 11. 实施步骤（对齐 codegen）

> codegen 命令：`make api`（proto→Go+HTTP+gRPC+openapi+ts）、`make gen`（`go generate`→ent + wire + tidy + pnpm build）、`make all`。

**P0 最小验证（不接 DB）**
- [ ] `go.mod` 引 frp v0.69.1；`pkg/frp_tunnel` 起 frps（`WebServer` 空）+ 一个 always-allow 的插件端点。
- [ ] 用官方 frpc + 手写 `frpc.toml` 连入，验证：能连上、插件收到 `Login`/`NewProxy`、`mem.StatsCollector` 能读到 proxy 与流量。
- [ ] 确认 frps `Auth` 中性化（空 token）后 frpc 能连上、frps 自身不拦截；插件 `Login` 为唯一认证权威（错误 credential 被拒、正确放行）。

**P1 数据模型 + API**
- [ ] ent schema：`frp_tunnel.go`、`frp_tunnel_stat.go` → `make gen`。
- [ ] `api/proto/v1/tunnel.proto` → `make api`。
- [ ] `system.proto` 增 `FrpsConfig` 消息 + `GetFrpsConfig`/`UpdateFrpsConfig` RPC → `make api`。

**P2 业务/数据/服务 + 认证**
- [ ] `internal/data/tunnel.go`（TunnelRepo）。
- [ ] `ConfigRepo`/`ConfigUsecase` 增 `FrpsConfig`/`UpdateFrpsConfig`（含端口/启用校验）。
- [ ] `internal/biz/tunnel.go`（TunnelUsecase：CRUD + 插件校验 + frpc 配置生成）。
- [ ] `internal/service/tunnel.go`（HTTP handler）。
- [ ] `pkg/frp_tunnel/plugin.go`：`Login`/`NewProxy` 接 DB 校验，替换 P0 的 always-allow。
- [ ] wire 注入；`make gen`。

**P3 统计持久化**
- [ ] `pkg/frp_tunnel/stats.go`：封装 `mem.StatsCollector` 采样。
- [ ] `TunnelUsecase.sampleLoop`/`cleanupLoop` + `SaveFrpTunnelStats`/`ListFrpTunnelStats`/`DeleteFrpTunnelStatsBefore`。
- [ ] `GetTunnelStats` RPC 打通。

**P4 前端**
- [ ] menu seed + i18n。
- [ ] `views/tools/tunnel/`（index + stats dialog + frpc-config 下载）。
- [ ] `frontend/src/api/tunnel.ts`、`types/v1/tunnel.ts`（由 `make api` 生成 openapi/ts）。

**P5 收尾**
- [ ] 管理界面 frps 配置页（`GetFrpsConfig`/`UpdateFrpsConfig`，含启用开关与端口编辑）。
- [ ] 文档与 README 片段（frpc 连接说明）。
- [ ] 权限点（RBAC：tunnel 的 CRUD 权限，参照 `internal/data/default_rbac.go`）。

## 12. 风险与边界

- **frp Go API 非稳定库契约**：锁 v0.69.1；升级需回归。插件 HTTP 契约（Request/Response）跨版本相对稳定，认证层风险低。
- **插件 HTTP 一跳延迟**：每个 Login/NewProxy 多一次本地 HTTP 调用，可接受；若敏感，后续可评估 Level 2 进程内插件（需自建 ResourceController，因 `svr.rc` 未导出）。
- **frps 被动语义**：listener 由 frpc 声明才开；「添加规则」≠ 立即开监听。UI 必须表达「待连接/在线」状态，避免与端口转发混淆。
- **`mem.StatsCollector` 全局单例**：本期单 frps 实例无问题；未来多实例需按 `user.name` 命名空间区分统计。
- **凭证安全**：`credential` 存库需加密/哈希（参考现有敏感字段处理）；frpc 配置下载仅对规则 owner 授权。
- **端口冲突**：frps `bind_port`、`vhost_http_port`、`remote_port` 需纳入 momoko 端口规划，避免与端口转发/实例端口冲突。
- **frpc `local_*` 字段**：由用户在 frpc 端实际暴露的服务决定，momoko 仅用于模板生成，不保证与 frpc 端一致。

## 13. 与端口转发的对照

| 维度 | 端口转发 | 内网穿透（本方案） |
|---|---|---|
| 监听来源 | momoko 主动开 listener | frpc 声明 NewProxy 后 frps 开 |
| 添加规则 | 即时开监听 | 授权；frpc 声明后才活 |
| 删除规则 | `UnRegisterExample` 关监听 | `Control.CloseProxy`/踢客户端 + 删规则 |
| 认证 | 无（momoko 内部） | frps 传输层 token + momoko 插件业务层 |
| 统计 | `Manager.SnapshotAll` → `port_forward_stat` | `mem.StatsCollector` → `frp_tunnel_stat` |
| 协议 | tcp/udp | tcp/udp/http/https/stcp/xtcp/tcpmux + kcp/quic/websocket 传输 |
