# 游戏服务端管理与 GM · 实现规范（本地版）

给本机 Claude Code 用。仓库根目录是 Momoko（Go Kratos 后端 + `frontend/` Vue 3 前端）。
这是一个**新功能**：在 Momoko 里管理游戏服务端（托管 / 仅 GM 两种接入），提供 GM 运营后台，以及给玩家用的「用户通行证」。第一个适配的游戏是学マス（学園アイドルマスター），服务端代码在 `C:\Users\gucooing\Desktop\gakumas\server`。

## 0. 先读

1. 本目录的 `Main.dc.html`（架构）和 `SpecPC.dc.html`（MGP 接口、数据表、权限、学マス GM 目录、素材来源）。**规格以 SpecPC 为准**，本文件是它的落地说明和用户反馈过的硬性要求。
2. `../IMPL_BRIEF.md`：前端通用规范（令牌只用 `--m-*`、通用组件、三语文案写在 `frontend/src/locales/extra/<模块>.ts`、视觉规则、`?tab=` 页内标签、检查命令）。这里全部沿用，不再重复。
3. 原型的读法同 `../project/`：不要搬 HTML 或 CSS 类，用项目组件实现同样的布局和信息。原型里的人物、数值是示例数据，但凡是用户能看到、有意义的信息，后端没有就补上（与 IMPL_BRIEF 的授权一致：不需要兼容分支）。

## 1. 用户反馈过的硬性要求（实现时逐条自查）

- **用户通行证是独立 server**：玩家端单独一个 HTTP 端口（默认 `:22634`，后台是 `:22633`），只注册玩家路由。管理接口在这个端口上根本不存在；两边的会话、Cookie、令牌互不认。目的是从结构上避免越权。
- **正式服 / 测试服只是名字**：服务器之间完全独立，没有「环境」「从测试服复制」之类的关联字段或功能。
- **服务器可能有几百个**：任何需要选服务器的地方都用「搜索 + 最近查看」的下拉（结果最多 20 条），不要把所有服务器铺成 chips、tab 或列表项；用户列表里的绑定只显示「N 个服务器 · M 款游戏」和游戏图标。
- **管理端不做按指令的权限**：能进 GM 运营的管理员就能执行这台服务器的全部指令。风险等级只决定交互：「高」必须填原因；「危险」执行前输入玩家昵称确认，存档类自动先备份。没有第二位管理员复核。
- **「玩家自助」页只管玩家能执行的 GM**：每条指令只有 开放 / 每天次数 / 间隔 / 需要审批 / 玩家看到的说明。不掺角色、统计、记录等其他东西。玩家的审批申请在「GM 记录」页处理。
- **工作台只做数据观测和预览**，不放操作入口；邮件补偿、兑换码、公告各自是独立页面。
- **卡池编辑是表单**：填规则、先选角色再选卡；不让人写 JSON（配置的 JSON 只在后端和适配器里出现）。
- **玩法页**：竞赛 PvP 只显示排名 + 选中玩家当前配置的阵容（不显示段位规则、奖励、结算按钮）；培育就是全体玩家的培育列表（和玩家详情里的一样，只是范围不同，不要「进行中 / 中止清理」）；只显示服务端实际配置了的玩法，每个玩法一个 tab，没配置的不显示。
- **删除「中止卡住的培育」这条 GM**（`produce.abandon`），任何地方都不要实现。
- **GM 记录的详情不显示请求参数**。
- **游戏素材当前端资源管理**，不从游戏服务端取图（第 7 节）。
- 玩家邮箱在后台列表里不完整显示（原型里写作 `[邮箱]`）。

## 2. 架构

```
玩家浏览器 / 游戏 SDK ──► 用户通行证 server :22634 ──┐
                                                    ├─ 共享：数据库、MGP 客户端库
管理员浏览器 ─────────► Momoko 主 server :22633 ────┘
                                                    │  MGP v1（HMAC 签名，HTTP）
                                                    ▼
                                游戏服务端（学マス 原生实现 MGP；MC 基岩版走适配器桥）
```

### 2.1 用户通行证 server（新增）

- 在 Momoko 进程里另起一个 Kratos HTTP server。配置：`passport.enabled`（默认 true）、`passport.addr`（默认 `0.0.0.0:22634`）、`passport.public_url`（OIDC issuer、邮件链接用）。
- 只注册 `passport.v1` 的服务：注册、登录、找回密码、两步验证、登录设备、授权的应用、OIDC Provider（游戏 SDK「用户通行证登录」走这里）、绑定 / 解绑游戏账号、查看自己的游戏数据、执行开放给玩家的 GM、我的申请、操作记录、消息。
- 会话：独立 Cookie 名和 JWT `aud=passport`；主 server 拒绝 `aud=passport` 的令牌，passport server 拒绝后台令牌。CORS 各自配置。
- 账号就是 Momoko 用户表里带「玩家」角色的用户（默认角色「玩家」没有任何后台菜单）。后台的「用户通行证」页管理这些用户。
- 前端：独立的 Vite 入口（例如 `frontend/passport.html` + `frontend/src/passport/main.ts`，独立 router、独立 store），构建到 `dist/passport/`，passport server 只提供这一份静态产物（SPA 回退同主站）。和后台共享 `src/components/ui/*`、令牌和 `src/games/*`。
- 验收：在 `:22634` 上请求任何后台接口（`/api/v1/...` 下的管理路由）必须 404；用后台令牌请求 passport 接口必须 401。

### 2.2 玩家执行 GM 的规则（passport server 内强制）

- 只允许 `gm_self_service.enabled = true` 的指令；指令还必须在该服务器 manifest 里标了 `selfService: true`。
- 目标强制为「当前用户在这台服务器绑定的 player_id」，忽略请求里的 targets。
- 每天次数、间隔在服务端计数（按服务器时区的自然日）。
- `need_approval` 的指令只写一条待审批的 `gm_execution`（status = pending），管理员在「GM 记录」页批准后再真正调用 MGP。
- 每次执行都写 `gm_execution`（actor_kind = player）。

### 2.3 管理端

- 新增菜单组「游戏」：游戏服务器、GM 运营、用户通行证（`internal/data/default_rbac.go`）。
- 权限点只加两个：`game:server`（服务器管理）、`game:gm`（GM 运营的全部页面与全部指令）。「用户通行证」页沿用用户管理的权限。
- 主 server 调 MGP 时：`dryRun` 先试运行（邮件、发礼物、卡池发布都有「试运行 / 预览」）；执行结果逐个对象写进 `gm_execution.changes`。

## 3. MGP v1（游戏服务端实现的接口）

完整定义见 `SpecPC.dc.html` 第 1、2 节。要点：

- 基础路径 `/mgp/v1`；鉴权头 `X-MGP-Key / X-MGP-Timestamp / X-MGP-Nonce / X-MGP-Signature`，签名 = HMAC-SHA256(方法、路径、时间戳、随机数、body 摘要)，±5 分钟、拒绝重放。直接复用 Momoko 现有节点 API Key 的签名代码。托管模式下密钥由 Momoko 生成并写进服务端配置。
- 接口：`GET /manifest`、`GET /status`、`GET /players?q=&cursor=`、`GET /players/{pid}`、`GET /players/{pid}/views/{view}`、`POST /commands/{cmd}`（targets、params、reason、requestId、dryRun）、`GET /catalogs/{kind}?q=`、`GET|PUT /configs/{domain}`（带 revision，发布后热重载）、`POST /accounts/verify`、`GET /events`（SSE）。
- **没有图片接口**。所有返回只带资源 ID（卡 ID、卡池 ID、道具 ID、角色 ID），前端自己查素材包。
- Momoko 侧写一个 `internal/game/mgp` 客户端（签名、超时、重试、manifest 与 status 缓存），两个 server 共用。

### 3.1 学マス 服务端（另一个仓库，单独做）

在 `gakumas/server` 新增 `server/mgp` 包实现上面的接口；`SpecPC.dc.html` 第 1 节表格的「学マス 服务端实现要点」列写了每个接口对应的现有代码（`game.Game.WithPlayer` 串行写档、gdconf 校验、`sdk/gm.go` 的现有 GM 等），第 5 节是按服务端能力整理的 GM 目录（已有 / 可直接做 / 需新增）。`sdk/gm.go` 的 `gm_token` 查询参数方式由 MGP 替换。这部分在 gakumas 仓库里另开任务，不在 Momoko 仓库里改。

## 4. 数据表（Momoko）

见 `SpecPC.dc.html` 第 3 节，按原样建表：

- `game_server`：id、game（适配器 id）、name（只是名字）、tags、mode（hosted | remote）、node_id、instance_id（托管时关联应用实例）、mgp_endpoint、key_id、secret（加密存储）、manifest 缓存、status 缓存。
- `game_account_link`：user_id、server_id、player_id、display_id、nickname、avatar_asset、method（密码 / 绑定码 / OIDC / 管理员）、is_primary、linked_at、last_seen_at。每个用户在每台服务器只能绑一个游戏账号；解绑后 7 天内不能绑别的账号。
- `gm_self_service`：server_id、command、enabled、daily_limit、cooldown_sec、need_approval、player_note。
- `gm_execution`：server_id、command、actor user_id、actor_kind（admin | player）、targets、params、reason、dry_run、status（success | failed | pending | rejected）、changes、reviewed_by、request_id。params 只存不展示。
- `game_config_revision`：server_id、domain、revision、content、author、comment、state（草稿 / 已发布）、published_at。

另外运营功能需要的表（原型里有，SpecPC 的 GM 目录标了「需新增」）：邮件（`game_mail`：标题、正文、奖励、收件条件或名单、期限、定时、状态、领取统计）、兑换码（`gift_code_batch` / `gift_code` / `gift_code_redeem`）、公告（落在游戏服务端的 `data/Notice.json` 配置域，Momoko 只做编辑）。

## 5. 页面、路由与要点

管理端（主 server）。GM 运营按 IMPL_BRIEF 第 5 条「一个菜单 = 一个页面」：`/game/gm/:serverId?tab=…`，页头左侧是服务器切换（搜索型下拉），下面是子导航。

| 原型 | 路由 | 要点 |
|---|---|---|
| `GameServersPC` | `/game/servers` | 服务器列表；按游戏、状态、标签筛选；搜索；不按「环境」分组 |
| `GameServerNewPC` | `/game/servers/new` | 选游戏 → 选托管 / 仅 GM → 表单；标签是自由输入 |
| `GameServerPC` | `/game/servers/:id?tab=overview\|console\|settings\|update\|backup\|files\|logs` | 托管服务器复用应用实例的控制台、文件、日志；「专用设置」是适配器给的表单 |
| `GkHomePC` | `?tab=home` | 只读：今日 KPI、活跃玩家与抽卡次数近 7/14 天（单系列图，无图例，悬停看数值）、卡池表现、需要留意、玩法与服务器概况、最近 GM 操作；每块只链接到对应页 |
| `GkPlayersPC` | `?tab=players` | 玩家表、筛选、批量操作条 |
| `GkPlayerPC` | `/game/gm/:serverId/players/:pid?tab=…` | 10 个分页：概览、偶像卡、支援卡、道具、礼物箱（GM 发出且未领取的可撤回）、抽卡记录、培育记录、竞赛（阵容 + 本赛季）、登录会话（在线那条可强制下线）、GM 记录；「发送礼物」是抽屉 |
| `GkGashaPC` | `?tab=gasha` | 卡池列表 + 表单编辑（基本信息、UP 角色与卡牌、概率、抽卡按钮、交换点）、模拟抽卡、发布前检查；底部草稿条（查看改动 / 放弃 / 发布） |
| `GkModesPC` | `?tab=modes` | 竞赛 PvP（排名 + 阵容）、培育（全体玩家列表）、研修、塔；tab 由 manifest 的已配置玩法决定 |
| `GkCalendarPC` | `?tab=calendar` | 运营日历（卡池、活动、PvP 赛季泳道） |
| `GkMailPC` | `?tab=mail` | 邮件列表 + 编写（指定玩家 / 按条件 / 全服、奖励、内容、期限、立即或定时、试运行、礼物箱预览）；指令 `mail.broadcast` |
| `GkCodesPC` | `?tab=codes` | 批次列表、批次详情与兑换记录、新建批次抽屉（通用码 / 一次性码、次数、有效期、适用玩家、兑换入口） |
| `GkNoticePC` | `?tab=notices` | 公告列表 + 编辑 + 游戏内预览；横幅「从游戏素材选」读前端素材包 |
| `GmPolicyPC` | `?tab=self-service` | 玩家自助：只列 manifest 里允许自助的指令 |
| `GkLogPC` | `?tab=logs` | 待审批卡片（批准 / 驳回）、筛选、记录表、详情（不含请求参数） |
| `PassportPC` | `/game/passport` | 用户通行证后台管理：统计、状态筛选 + 服务器搜索下拉、用户表、用户详情（绑定按最近使用，超过 5 个折叠） |

玩家端（用户通行证 server，独立前端入口）：

| 原型 | 路由 | 要点 |
|---|---|---|
| `PassportHomePC` / `PassportHomeMobile` | `/` | 先账号（资料、安全提醒），再「我的游戏」（按游戏分组，每个服务器一行），消息、最近活动 |
| `PassportAccountPC` | `/account` | 个人资料、登录与安全、登录设备、授权的应用、游戏账号、注销 |
| `PortalBindMobile` | `/bind` | 选服务器（搜索 + 按游戏筛选 + 推荐，不列出全部）→ 账号密码 / 绑定码 / 游戏内登录 |
| `GameHomePC` / `GameHomeMobile` | `/games/:serverId` | 游戏空间，用游戏皮肤；页头是服务器名和「切换服务器」 |
| `GameCardsPC` `GameGiftsPC` `GameRecordsPC` `GameSelfPC` | `/games/:serverId?tab=cards\|gifts\|records\|self` | 卡牌、礼物箱、记录（抽卡 / コンテスト / 培育 / 研修）、自助服务（指令卡片带规则、兑换码、我的申请、操作记录） |

手机：原型只画了通行证首页、绑定、游戏空间概览三张。其余玩家端页面照这三张的写法做手机版；管理端手机按 IMPL_BRIEF 的通用规则（表格转卡片、单列表单）。

## 6. 游戏皮肤

学マス 的游戏空间、GM 运营页的强调色来自游戏皮肤，见 `momoko-game.css` 的 `.gk-light / .gk-dark`。实现为 `frontend/src/games/gakumas/theme.css`：在 `[data-game='gakumas']` 上定义 `--m-g --m-gOn --m-gInk --m-gSoft --m-gLine --m-g2 --m-g2Ink --m-g2Soft --m-g3 --m-g3Soft`（明暗两套），游戏空间区域再把 `--m-p --m-pOn --m-pInk --m-pSoft` 指向这组颜色。通行证外壳（账号、首页）仍用 Momoko 自己的配色。图表线条和柱子用 `--m-gInk` 或比 `--m-g` 深一档的颜色，保证与背景 3:1 对比。

## 7. 游戏素材（前端资源）

完整说明在 `SpecPC.dc.html` 第 6 节。落地步骤：

1. 把 `tools/extract_gakumas_assets.py` 放到 `frontend/scripts/games/gakumas/extract-assets.py`。
2. 运行（需要 `pip install UnityPy Pillow`）：
   ```
   python frontend/scripts/games/gakumas/extract-assets.py ^
     --resource C:/Users/gucooing/Desktop/gakumas/resource ^
     --codec    C:/Users/gucooing/Desktop/gakumas/client/tools ^
     --out      frontend/src/games/gakumas/assets
   ```
   按规则约导出 2,000 张（资源清单 version 705100 rev 83 共 49,411 个对象）；名片（2,349 张）默认不导出，要用时加 `--with-meishi`。输出 `index.json`（尺寸、源资源名、md5）和 `manifest.ts`（`gakumasAsset(key)`）。md5 没变的文件下次跳过。
3. 在 `frontend/src/games/index.ts` 提供 `gameAsset(game, key)`，查不到返回占位图。页面里把游戏数据的 ID 换算成 key，例如偶像卡 `cards/idol/hski-3-000-sq`、卡池 `gasha/gasha-01055-pickup-001`、道具 `items/jewel-free`、角色 `chara/hski`。
4. 提交导出的 WebP（体积大了就用 Git LFS）；Vite 打包带 hash，随前端嵌进二进制；passport 前端用同一个 `src/games` 目录。
5. 不要引用本目录 `assets/`（那只是原型用的 47 张）。

## 8. 分期建议

1. 后端基础：数据表、`internal/game/mgp` 客户端与签名、游戏服务器的增删改查和状态轮询、托管模式复用应用实例；前端「游戏服务器」三页。
2. GM 运营（学マス 适配）：工作台、玩家、玩家详情、卡池、玩法、运营日历、邮件补偿、兑换码、公告、玩家自助、GM 记录。
3. 用户通行证 server 与玩家端全部页面、OIDC、绑定流程。
4. 素材导出与 `src/games/gakumas`。
5. 学マス 服务端实现 MGP（gakumas 仓库，另开任务）。在它完成前，Momoko 侧可以先写一个按 SpecPC 返回示例数据的 MGP mock，供前端联调。

## 9. 检查

- 前端：`cd frontend && pnpm exec vue-tsc --build`，`pnpm exec vite build`（两个入口都要构建成功）。
- 后端：`go test ./...`；新增接口照现有方式加鉴权注解和节点策略登记。
- 手动：`:22634` 上访问后台接口 404；后台令牌访问 passport 接口 401；玩家执行未开放的指令被拒；超过每天次数被拒；需要审批的指令只生成待审批记录。
