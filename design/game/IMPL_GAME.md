# 游戏服务端管理与 GM · 实现规范（本地版）

给本机 Claude Code 用。仓库根目录是 Momoko（Go Kratos 后端 + `frontend/` Vue 3 前端）。
这是一个**新功能**：在 Momoko 里管理游戏服务端（托管 / 仅 GM 两种接入），提供 GM 运营后台，以及给玩家用的「用户通行证」。第一个适配的游戏是学マス（学園アイドルマスター），服务端代码在 `C:\Users\gucooing\Desktop\gakumas\server`。

## 0. 先读

1. 本目录的 `Main.dc.html`（架构）和 `SpecPC.dc.html`（MGP 接口、数据表、权限、学マス GM 目录、素材来源）。**规格以 SpecPC 为准**（MGP 接口以第 3 节为准：学マス 的接口表是游戏服务端仓库的 `server/docs/mgp.md`），本文件是它的落地说明和用户反馈过的硬性要求。
2. `../IMPL_BRIEF.md`：前端通用规范（令牌只用 `--m-*`、通用组件、三语文案写在 `frontend/src/locales/extra/<模块>.ts`、视觉规则、`?tab=` 页内标签、检查命令）。这里全部沿用，不再重复。
3. 原型的读法同 `../project/`：不要搬 HTML 或 CSS 类，用项目组件实现同样的布局和信息。原型里的人物、数值是示例数据，但凡是用户能看到、有意义的信息，后端没有就补上（与 IMPL_BRIEF 的授权一致：不需要兼容分支）。

## 1. 用户反馈过的硬性要求（实现时逐条自查）

- **用户通行证是独立 server**：玩家端单独一个 HTTP 端口（默认 `:22634`，后台是 `:22633`），只注册玩家路由。管理接口在这个端口上根本不存在；两边的会话、Cookie、令牌互不认。目的是从结构上避免越权。
- **正式服 / 测试服只是名字**：服务器之间完全独立，没有「环境」「从测试服复制」之类的关联字段或功能。
- **服务器可能有几百个**：任何需要选服务器的地方都用「搜索 + 最近查看」的下拉（结果最多 20 条），不要把所有服务器铺成 chips、tab 或列表项；用户列表里的绑定只显示「N 个服务器 · M 款游戏」和游戏图标。
- **管理端不做按操作的权限**：能进 GM 运营的管理员就能执行这台服务器的全部操作。风险等级只用于展示与玩家自助的两步验证；执行前不做任何二次确认（不输入昵称、不弹确认框），「危险」操作由服务端先备份存档。没有第二位管理员复核。
- **对玩家的操作都能同时作用于多名玩家**：除了针对具体一张卡或一条回忆的援助设置、玩家详情里的存档备份与回档，GM 操作的接口都收 `pids`（1–200 名），逐个玩家执行，单个失败不影响其他玩家。
- **「GM 操作」页的「玩家权限」只管玩家能执行的 GM**：列出的操作与「执行操作」里的完全一致，每项只有 开放 / 每 X 天 N 次机会（服务器时区 0 点刷新；可设为永久不刷新；N 为 0 不限） / 需要审批 / 玩家看到的说明。不掺角色、统计、记录等其他东西。玩家的审批申请在「GM 记录」页处理，批准即按申请时的参数执行。
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
- 只注册玩家能用的服务：`passport.v1`（注册、登录、找回密码、两步验证、登录设备、授权的应用、OIDC Provider（游戏 SDK「用户通行证登录」走这里）、绑定 / 解绑游戏账号、游戏空间的服务器与绑定、自助服务规则、我的申请、操作记录、兑换码、消息）和各游戏的通行证服务（学マス：`v1.gakumas.GakumasPassportService`，查看自己的游戏数据、目录、执行开放给玩家的操作）。注册到这个 server 的接口就是玩家能访问的全部接口：玩家不该用的接口不注册，不在处理函数里另加限制。
- 会话：独立 Cookie 名和 JWT `aud=passport`；主 server 拒绝 `aud=passport` 的令牌，passport server 拒绝后台令牌。CORS 各自配置。
- 账号就是 Momoko 用户表里带「玩家」角色的用户（默认角色「玩家」没有任何后台菜单）。后台的「用户通行证」页管理这些用户。
- 前端：独立的 Vite 入口（例如 `frontend/passport.html` + `frontend/src/passport/main.ts`，独立 router、独立 store），构建到 `dist/passport/`，passport server 只提供这一份静态产物（SPA 回退同主站）。和后台共享 `src/components/ui/*`、令牌和 `src/games/*`。
- 验收：在 `:22634` 上请求任何后台接口（`/api/v1/...` 下的管理路由）必须 404；用后台令牌请求 passport 接口必须 401。

### 2.2 玩家执行 GM 的规则（passport server 内强制）

- 只允许 `gm_self_service.enabled = true` 的操作；玩家能执行的操作就是管理端操作表里标了允许自助的（`internal/biz/gakumas.go` 的 `GakumasOperations`，「执行操作」里的操作都允许），通行证用例由它生成 `GakumasSelfService`（`internal/biz/gakumas_passport.go`），每项操作在通行证上有自己的接口（`/self-service/<操作>`），参数与管理端相同：发礼物从游戏目录选奖励，撤回礼物从自己礼物箱里管理端发出且未领取的礼物中勾选（一次一种来源），调整道具从道具目录选，封禁的原因字段为 `ban_reason`（`reason` 是给审批管理员的申请理由）。
- 目标强制为「当前用户在这台服务器绑定的 player_id」，忽略请求里的 targets。
- 次数在服务端计数：周期从 2024-01-01（周一）起按 X 天切分，服务器时区 0 点刷新（X 为 7 时每周一刷新）；永久不刷新时统计全部。
- `need_approval` 的操作只写一条待审批的 `gm_execution`（status = pending，params 为游戏服务端接口的 body），管理员在「GM 记录」页批准后由管理端按 params 调用 MGP。
- 每次执行都写 `gm_execution`（actor_kind = player，结果格式为 `internal/data/ent/schema` 的 `GmOutcome`，与管理端执行的记录相同）。
- 自助操作与兑换码的执行在通行证一侧（`PassportUsecase.RunSelfService`、`PassportUsecase.Redeem`），经数据层直接读写 GM 记录、自助规则与兑换码的表，不调用管理端的用例。
- 资料设置（学マス 的援助支援卡、援助回忆）与游戏里的资料设置相同，是玩家对自己数据的设置，不是 GM 操作：通行证直接调用 MGP（`X-MGP-Operator` 为通行证显示名，每次一个新的幂等键），不经自助规则，也不写 GM 记录。管理端设置同样的内容走 GM 操作（`supportCard.setRental`、`memory.setRental`），照常试运行、确认并写 GM 记录。

### 2.3 管理端

- 新增菜单组「游戏」：游戏服务器、GM 运营、用户通行证（`internal/data/default_rbac.go`）。
- 权限点只加两个：`game:server`（服务器管理）、`game:gm`（GM 运营的全部页面与全部操作，含 `v1.gakumas.GakumasGMService`）。「用户通行证」页沿用用户管理的权限。
- 主 server 调 MGP 时：支持试运行的操作先 `?dryRun=true`（邮件、发礼物、卡池发布都有「试运行 / 预览」）；执行结果逐个对象写进 `gm_execution` 的结果（对象、状态、错误码与游戏服务端返回的 JSON）。

## 3. MGP v1（游戏服务端实现的接口）

完整定义见 `SpecPC.dc.html` 第 1、2 节。要点：

- 基础路径 `/mgp/v1`；鉴权头 `X-MGP-Key / X-MGP-Timestamp / X-MGP-Nonce / X-MGP-Signature`，签名 = HMAC-SHA256(方法、路径、时间戳、随机数、body 摘要)，±5 分钟、拒绝重放。直接复用 Momoko 现有节点 API Key 的签名代码。托管模式下密钥由 Momoko 生成并写进服务端配置。
- 接口按 REST 标准逐个定义：每项查询与每项操作都是独立的 HTTP 接口，带自己的参数、筛选与分页（例如 `GET /players`、`GET /players/{pid}/gasha-draws?gashaIds=&cardIds=&cursor=&limit=`、`POST /players/{pid}/items/adjust`、`GET /catalog/items`、`GET|PUT /config/gasha`），不设按名称分发的统一接口。列表的筛选与分页都在服务端完成，前端不为搜索读出全部数据。
- 操作接口支持 `Idempotency-Key`（同一个键只执行一次，重放时返回原结果并带 `Idempotent-Replayed`）、`X-MGP-Operator` 与 `?dryRun=true`。
- 学マス 的完整接口表是游戏服务端仓库的 `server/docs/mgp.md`，以它为准。
- **没有图片接口**。所有返回只带资源 ID（卡 ID、卡池 ID、道具 ID、角色 ID），前端自己查素材包。

### 3.0 Momoko 侧的分层

**管理端与用户通行证完全分开**：接口定义、用例与 service 各自独立，互不调用。

- 接口：管理端在 `api/proto/v1/gakumas/gm.proto`（`GakumasGMService`，主 server），用户通行证在 `api/proto/v1/gakumas/passport.proto`（`GakumasPassportService`，passport server）；各自定义请求与响应，通行证的 proto 不引用管理端的消息（例如自助操作只返回 `SelfServiceResult`，不返回管理端的 GM 记录）。两边共用的只有游戏数据 `api/proto/v1/gakumas/types.proto`（玩家、卡、礼物、列表页、目录页、操作结果等 MGP 返回的内容）。通用部分同理：`passport.proto` / `passport_admin.proto`、`game_gm.proto`。
- 用例：管理端 `GmUsecase`、`GameOpsUsecase`、`GakumasUsecase`、`PassportAdminUsecase`；用户通行证 `PassportUsecase`、`GakumasPassportUsecase`。两边都可以用的只有数据层（各表的 repo）、游戏服务器的读取与连接（`GameServerUsecase` 的读取、搜索、时区与 MGP 客户端，不含服务器管理）和对共用表的无状态规则函数（兑换码规则 `gift_code.go`；绑定、解绑、消息、设置的 `passportCreateLink` / `passportUnlink` / `passportNotify` / `passportSettings` 等）。
- service：`internal/service/gakumas.go`（管理端）与 `gakumas_passport.go`（通行证）、`passport.go`（通行证）与 `passport_admin.go`（管理端）分文件。

- `pkg/mgp`：通用客户端（签名、超时、幂等键、错误码），只有所有游戏都有的 `Manifest` 与 `Status`。
- `pkg/mgp/<游戏>`（学マス：`pkg/mgp/gakumas`）：这个游戏的每个 MGP 接口一个函数，直接返回接口内容（`api/gen/v1/gakumas` 的强类型）。没有跨游戏的通用游戏接口。
- `internal/biz`：只做身份、接口权限、Momoko 自己的记录（GM 记录、审批、自助规则、配置修订），再调用 `pkg/mgp/<游戏>` 的函数并原样返回内容；学マス 的用例在 `internal/biz/gakumas*.go`。
- `internal/service`：解析请求、调用 biz；接口定义在 `api/proto/v1`（学マス：`api/proto/v1/gakumas`）。
- 前端：管理端接口 `frontend/src/api/gakumas.ts`、操作表 `frontend/src/views/game/gm/operations.ts`；玩家端接口 `frontend/passport/src/api/gakumas.ts`（只用 `passport.ts` 与 `types.ts` 的类型）；两端共用的 `frontend/src/games/gakumas`（游戏数据类型、目录查询、玩法、素材、名称表、详情展示的标签与目录查询接口 `display.ts`）不依赖任何一端的接口模块与请求类型。两端共用的展示组件只有 `frontend/src/components/game` 下的 `GameImage` 与记忆详情 `GakumasMemoryDetail`（培育 / 舞台 / 培育记录三页，文案是组件自己的三种语言，名称与效果文字由调用方传入的目录查询给出）；弹窗、接口调用与「设为援助」各端自己实现。
- 列表分页：游戏数据列表（两端的卡牌、回忆、培育记录、抽卡记录、礼物箱、自助申请与操作记录，以及详情里的列表）一律页码分页，默认每页 20 条：管理端用 `Pagination`，通行证用 `PPager` 与 `utils/paged.ts`；不用「加载更多」追加。礼物箱的到期、来源筛选同样由服务端完成（MGP `expiring`、`source`）。
- 列表搜索：服务端只认 ID 与游戏原文名。前端先用名称表（当前语言的译名与原文名）把关键字换成 ID 交给服务端按 ID 筛选；玩家的偶像卡、支援卡、记忆列表接口还带 `q`（服务端模糊匹配 ID 与游戏原文名），关键字换不出 ID 时原样交给 `q`。
- 回忆详情（`GakumasMemoryDetail`）照游戏的回忆详情与培育记录排版：回忆照片经 `GetMemoryPhoto`（MGP `GET /memory-photo`，base64）读取，没有照片的回忆用游戏图片；技能卡、P 道具与没有照片的回忆图片是前端素材（`extract-assets.py` 的 skillcards、pitems、memories 分组；按偶像区分图片的技能卡，目录带 characterAsset，图片为 assetId 加「-」和这条回忆的角色 ID；导出脚本使用客户端工具目录 vendor/unitypy 里的 UnityPy 与 Pillow）；技能卡名（强化过的为 `<id>+<次数>`）、P 道具名与效果文字、记忆能力文字取名称表（`export-names.py` 的 produceCard、produceItem、produceItemText、memoryAbilityText），当前语言没有译名时用游戏原文；培育记录里编成的支援卡与回忆点开看详情，自己的回忆可以打开完整详情。偶像卡详情里点皮肤只在页面上切换预览。卡牌详情用高清原画：偶像卡 9:16、支援卡 16:9（`extract-assets.py` 的 `-full`；客户端把原画存成二的幂尺寸的贴图，导出时按比例还原）。管理端偶像卡列表有角色头像快捷筛选，与稀有度一样由服务端按表的列筛选，头像与胶囊上的张数来自列表的 summary。
- 记忆与培育记录按稀有度（R / SR / SSR，表的 rarity 列）与评价（S、SS、SSS…… 每档含 +，A+ 及以下一类，`grades` 传 ResultGrade）筛选，都由服务端筛选分页；列表的 `summary` 给按稀有度、按评价的条数（各自带上另一项筛选），筛选胶囊只显示有条数的项。评价分类在 `frontend/src/games/gakumas/display.ts`，两端共用。

### 3.1 学マス 服务端（另一个仓库，单独做）

在 `gakumas/server` 新增 `server/mgp` 包实现上面的接口；`SpecPC.dc.html` 第 1 节表格的「学マス 服务端实现要点」列写了每个接口对应的现有代码（`game.Game.WithPlayer` 串行写档、gdconf 校验、`sdk/gm.go` 的现有 GM 等），第 5 节是按服务端能力整理的 GM 目录（已有 / 可直接做 / 需新增）。`sdk/gm.go` 的 `gm_token` 查询参数方式由 MGP 替换。这部分在 gakumas 仓库里另开任务，不在 Momoko 仓库里改。

## 4. 数据表（Momoko）

见 `SpecPC.dc.html` 第 3 节，按原样建表：

- `game_server`：id、game（适配器 id）、name（只是名字）、tags、mode（hosted | remote）、node_id、instance_id（托管时关联应用实例）、mgp_endpoint、key_id、secret（加密存储）、manifest 缓存、status 缓存。
- `game_account_link`：user_id、server_id、player_id、display_id、nickname、avatar_asset、method（密码 / 绑定码 / OIDC / 管理员）、is_primary、linked_at、last_seen_at。每个用户在每台服务器只能绑一个游戏账号；解绑后 7 天内不能绑别的账号。
- `gm_self_service`：server_id、command、enabled、use_limit、period_days（0 永久不刷新）、need_approval、player_note。
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
| `GkPlayersPC` | `?tab=players` | 玩家表、筛选、批量操作条（勾选后可直接发礼物、强制下线、封禁、解封，或带着勾选的玩家去 GM 操作页） |
| `GkPlayerPC` | `/game/gm/:serverId/players/:pid?tab=…` | 玩家卡片的背景是玩家的资料名片（游戏里保存名片时上传的图片，`GetPlayerMeishiImage`；默认名片没有图片，这时右侧显示出借记忆的卡面）。分页：概览、偶像卡、支援卡、道具、礼物箱（GM 发出且未领取的可撤回）、抽卡记录、培育记录、记忆、竞赛（阵容 + 本赛季）、研修、登录会话（在线那条可强制下线）、存档备份、GM 记录；「发送礼物」是抽屉；「更多」里只有备份当前存档、回档到备份和去 GM 操作页的入口，其余操作都在 GM 操作页执行。偶像卡点开看详情（养成数值、持有的皮肤、这张卡保留的培育记录）；支援卡点开看详情并可「设为援助支援卡」；记忆与培育记录点开看记忆详情（培育 / 舞台 / 培育记录）并可「设为援助回忆」，两个设置都走操作对话框 |
| `GkGashaPC` | `?tab=gasha` | 卡池列表 + 表单编辑（基本信息、UP 角色与卡牌、概率、抽卡按钮、交换点）、模拟抽卡、发布前检查；底部草稿条（查看改动 / 放弃 / 发布） |
| `GkModesPC` | `?tab=modes` | 竞赛 PvP（排名 + 阵容）、培育（全体玩家列表）、研修、塔；tab 由 manifest 的 `modes`（已配置的玩法）决定 |
| `GkCalendarPC` | `?tab=calendar` | 运营日历（卡池、活动、PvP 赛季泳道） |
| `GkMailPC` | `?tab=mail` | 邮件列表 + 编写（指定玩家 / 按条件 / 全服、奖励、内容、期限、立即或定时、试运行、礼物箱预览）；由 Momoko 调游戏服务端的邮件接口投递 |
| `GkCodesPC` | `?tab=codes` | 批次列表、批次详情与兑换记录、新建批次抽屉（通用码 / 一次性码、次数、有效期、适用玩家、兑换入口） |
| `GkNoticePC` | `?tab=notices` | 公告列表 + 编辑 + 游戏内预览；横幅「从游戏素材选」读前端素材包 |
| `GmPolicyPC` | `?tab=operations` | GM 操作：「执行操作」左边是玩家列表（状态筛选、搜索、页码分页、勾选，翻页后勾选保留；从玩家详情、玩家列表进来时带 `?pid=a,b` 预先勾选），右边选择操作并执行，每个操作都对勾选的全部玩家执行（援助设置与存档备份、回档不在这里）；「玩家权限」（`?view=permissions`）管理玩家在通行证自助服务里能执行哪些操作（开放 / 每 X 天 N 次，0 点刷新或永久不刷新 / 需要审批 / 说明），列出的操作与「执行操作」相同 |
| `GkLogPC` | `?tab=logs` | 待审批卡片（批准 / 驳回）、筛选、记录表、详情（不含请求参数） |
| `PassportPC` | `/game/passport` | 用户通行证后台管理：统计、状态筛选 + 服务器搜索下拉、用户表、用户详情（绑定按最近使用，超过 5 个折叠） |

玩家端（用户通行证 server，独立前端入口）：

| 原型 | 路由 | 要点 |
|---|---|---|
| `PassportHomePC` / `PassportHomeMobile` | `/` | 先账号（资料、安全提醒），再「我的游戏」（按游戏分组，每个服务器一行），消息、最近活动 |
| `PassportAccountPC` | `/account` | 个人资料、登录与安全、登录设备、授权的应用、游戏账号、注销 |
| `PortalBindMobile` | `/bind` | 选服务器（搜索 + 按游戏筛选 + 推荐，不列出全部）→ 账号密码 / 绑定码 / 游戏内登录 |
| `GameHomePC` / `GameHomeMobile` | `/games/:serverId` | 游戏空间，用游戏皮肤；页头是服务器名和「切换服务器」，背景是我的资料名片（`GetMyMeishiImage`；默认名片没有图片时用头像偶像卡的卡面） |
| `GameCardsPC` `GameGiftsPC` `GameRecordsPC` `GameSelfPC` | `/games/:serverId?tab=cards\|gifts\|records\|self` | 卡牌（稀有度、偶像与关键字筛选；偶像卡详情列出持有的皮肤与培育记录，支援卡详情可设为援助）、礼物箱、记录（抽卡 / コンテスト / 培育 / 研修；培育记录点开看记忆详情并可设为援助回忆）、自助服务（操作卡片带规则、兑换码、我的申请、操作记录） |

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
2. GM 运营（学マス 适配）：工作台、玩家、玩家详情、卡池、玩法、运营日历、邮件补偿、兑换码、公告、GM 操作、GM 记录。
3. 用户通行证 server 与玩家端全部页面、OIDC、绑定流程。
4. 素材导出与 `src/games/gakumas`。
5. 学マス 服务端实现 MGP（gakumas 仓库，另开任务）。在它完成前，Momoko 侧可以先写一个按 SpecPC 返回示例数据的 MGP mock，供前端联调。

## 9. 检查

- 前端：`cd frontend && pnpm exec vue-tsc --build`，`pnpm exec vite build`（两个入口都要构建成功）。
- 后端：`go test ./...`；新增接口照现有方式加鉴权注解和节点策略登记。
- 手动：`:22634` 上访问后台接口 404；后台令牌访问 passport 接口 401；玩家执行未开放的操作被拒；超过本周期的次数被拒；需要审批的操作只生成待审批记录。
