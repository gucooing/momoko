# 08 · 进度清单(Living Progress)

> **活文档**:每次会话收尾更新。状态图例:⬜ 未开始 · 🟡 进行中 · ✅ 完成(桌面+移动+明暗+验收过)。
> 验收标准见 `README` §5;每页需桌面(1440)+ 移动(390)截图核对。

---

## 里程碑

| 阶段 | 内容 | 状态 |
|---|---|---|
| 任务书 | `docs/redesign/*` 全套 | ✅ |
| 基线撤回 | 探索性改动还原 | ✅ |
| Phase 0 | Nuxt UI 接入 + 令牌 + 横切设施 + `components/ui/*` | ✅ 已跑通并浏览器验证 |
| Phase 1 | 外壳(侧栏/顶栏/标签/内容,桌面+移动) | ✅ 样板已建+验证(见下务实取舍) |
| Phase 2 | 样板页:工作台 + 用户管理(定方向) | ✅ 工作台 + 用户管理(卡/表/CRUD/移动/明暗)已完成并浏览器验证 |
| Phase 3 | 全量推广(见下逐页) | 🟡 系统管理 100% + 实例/Docker/工具/文件 + 登录/初始化/异常 + 个人中心 + **Sub2API 配置** 完成；余 Sub2API 首页/活动 + public/*；监控=假数据(Phase 4 待决) |
| Phase 3.5 | **伪终端重写(实例控制台 + SSH 终端 → 真 xterm.js + 后端真 PTY)** | 🟡 前后端完成并实机验证:codex TUI 直连键入/回显/resize、SSH htop 鼠标滚轮(ws 抓包证实);Win10 宿主 ConPTY 无鼠标属 OS 限制(OpenConsole 集成经用户裁决**不做**);移动+浅色待验 |
| Phase 4 | 清理待决 + 暗色/可访问性/i18n 全量核对 + **EP/VXE 彻底卸载** | ⬜ |

> **🔓 后端破坏性修改授权(2026-07-12,用户明确)**：若前端重写需要,**允许破坏性修改后端、不必考虑兼容性**(协议/RPC/表结构随意改)。后端从 GoLand 运行,改 Go 后需用户手动重跑。
>
> **🖥️ 终端(P6 伪终端)重写口径(2026-07-12,用户点名)**：实例控制台 + SSH 终端"都不好看、不符合新规范" → 一并重写为**真 xterm.js**。定稿:①**升级实例控制台为真 xterm**(旧为自建 `outputLines` 行渲染的"伪终端";SSH 本就是 xterm);②明暗**改为终端自身手动切换、不受全局主题影响、默认黑色**(自包含 `--term-*` 令牌 + `localStorage`,类似文件编辑器的独立主题);③外壳/工具条走**新令牌 + 薄荷强调 + 状态点**;④**终端页 fullBleed 全出血**(用户:"整个页面都属于终端,强行加框不伦不类");⑤**后端真 PTY**(用户:"一个是真实的终端,一个只是假装的终端" → 实例子进程跑在 `go-pty` 伪终端里,ws 协议与 SSH 完全同构:原始键盘流入 / `{"type":"resize"}` 控制帧 / 原始字节流出(二进制帧+前端流式 UTF-8 解码),后端**零加工输出**;实例控制台去掉命令输入条,xterm 直接键入)。共享 `components/terminal/` 的 **`TerminalConsole.vue`(两页唯一公共入口:外壳+xterm+主题+输入输出接线,页面只写传输层)** + `TerminalShell.vue`(外壳) + `useTerminalX.ts`(xterm 封装,含 onBinary 二进制鼠标通道) + `useTerminalTheme.ts`(手动主题)。**踩坑**:go-pty Windows 下把相对 argv0 拼到 Dir 上不查 PATH → `servercore.resolveCommandPath` 必须对裸命令名做 `exec.LookPath` 兜底(codex 等 PATH 安装的命令);Unix `findInDir` 要求可执行位防同名普通文件遮蔽;**Win10 inbox ConPTY 不透传 alt-buffer/鼠标开启序列且丢弃鼠标输入(探针实证)** → Windows 宿主上 TUI 滚轮无响应属 OS 限制,Linux 部署目标不受影响。

---

## ⭐ 交接手册（新会话从这里接手）

> **一句话现状（换设备从这里接）**：Phase 0–2 + Phase 3 大半完成。**Sub2API 管理端主线已提交 `c8d08f2`**。**✅ 工作树未提交（公开门户两页）**：① `public/sub2api/home` 拆接口+去 EP+渐进渲染；② **`public/sub2api/stats` 去 EP 全量重写**（MetricStrip 单色 + 令牌 seg + 桌面语义表/移动 rank + 轻 spinner，无整页 loading）。监控假数据→Phase 4。**下一步**：`public/sub2api/imagine` → lottery → `sub2api/activity`；`public/share`/`oidc` 仍 EP。

### ✅ 已提交 `c8d08f2`：Sub2API 后端「统计聚合全下沉 ent + 接口按模块拆分 + 列表去闪」
> 铁律见记忆 [[sub2api-backend-rewrite-mandates]]。管理端：`GetSub2APIAdminTotals/Trend/Top` + recent `RecordFilter`；ent 聚合；tps≥20；token Top；`bucket15m`；DataTable 重载不换骨架。浏览器已验。

### ✅ 未提交：公开门户 home 拆接口 + 去 EP + 渐进渲染（2026-07-20）
> 用户点名 home 包太大 + 禁止整页等请求。`Sub2APIHome` 删 `snapshot`；新 `GetPublicSub2APIOverview`；前端三请求并行 + 乐观外壳。桌面已验。

### ✅ 本阶段完成（未提交）：`public/sub2api/stats` 去 EP + 补齐规范（2026-07-20）
> 对齐公开 home 壳；**禁彩色 metric 边框** → `MetricStrip`+`MetricItem` 单色 + **Heroicons 指标图标**（`MetricItem` 新增 `icon` prop，读 store `card.icon`）；`el-*` 全清；顶栏 **`LanguageMenu` + `AppIconButton`（禁 EP 图标）**；区块 **stagger 入场**（`.reveal` fade+translateY 220ms，`--d` 0–300ms，`prefers-reduced-motion` 关闭）；渐进填 `—`/轻 spinner。公开 **home 顶栏同步**：LanguageMenu + AppIconButton 主题/公告。桌面 1440 暗/EN 切换 + 移动 390 rank + 语言菜单；vue-tsc/eslint 绿；零 console；DOM 验：`metric__icon`×5 SVG、`epIcons=0`、reveal delay 0/60/120/180/240/300ms。

**下一会话**
1. 工作树未 commit（公开 home 拆分 + stats 去 EP + MetricItem 图标）——换设备请带工作树或先 commit。
2. 继续：`public/sub2api/imagine` → lottery → `sub2api/activity`。

### 如何启动 / 可视化验证
- 前端 dev：`cd frontend && pnpm dev`（本次会话 :3007 被占用→实际跑在 **:3008**，新会话按提示端口）。登录 `admin / admin`；后端 API `:22633`。
- 浏览器 MCP `chrome-devtools`（local，autoConnect）：`navigate_page` / `take_screenshot` / `resize_page` / `evaluate_script`。**每页验收：桌面 1440×900 + 移动 390×844，明+暗。** 截图落 `.browser-tmp/shots/`（gitignored）。
- ⚠️ `.app-content` 是**内部滚动容器**（非 window）：截"整页底部"要 `document.querySelector('.app-content').scrollTop = ...`，`fullPage` 只截到视口。

### 新增文件清单（读这些即懂样板架构）
- **地基**：`vite.config.ts`（`@nuxt/ui/vite` `ui()`，auto-import/components 并入其中）、`src/main.ts`（导入 `design-tokens.css` + `@nuxt/ui/vue-plugin`）、`src/App.vue`（`<UApp>` 包裹）、`src/styles/index.css`（`@import '@nuxt/ui'`）、`src/styles/design-tokens.css`（**令牌核心**）、`src/stores/theme/index.ts`（默认色 `#14B8A6`）
- **外壳**：`src/layouts/index.vue`（渲染 `AppShell`+`ThemeConfig`）、`src/layouts/app/*` = `AppShell/AppSidebar/AppNav/AppNavItem/AppTopbar/AppTabs/AppContent/CommandSearch/LanguageMenu/NotificationMenu/UserMenu`
- **基础组件**：`src/components/ui/*` = `AppPanel/SectionHeader/StatusPill/AppAvatar/AppIconButton/MetricStrip/MetricItem/EntityCard/EmptyState/DescriptionList/AppDropdown`（全令牌驱动、自动全局注册）
- **工作台**：`src/views/dashboard/home/*` = `index.vue`（编排+问候+MetricStrip）、`runningInstanceSection.vue`、`shortcutSection.vue`、`systemRealtimeCharts.vue`（仅重写壳、保留 ECharts）。`welcomePanel.vue`/`systemOverviewCards.vue` 已弃用可删。

### 接手要点 / 已踩坑（务必先看）
1. **单实例 unplugin**：Nuxt UI 只允许一个 `unplugin-auto-import`/`unplugin-vue-components`，配置**并入 `ui({autoImport,components})`**（defu 合并、数组拼接），**别再加独立实例**（否则启动报 "Multiple instances"）。
2. **暗色**：`ui({colorMode:false})`，`.dark` 由 `themeStore`（`useDark()`）独占。`design-tokens.css` 必须在 `main.ts` 里**于 EP 暗色 css 之后**导入才能覆盖；暗色块用 `html.dark` 提特异性。teal-500 == 薄荷 `#14B8A6`。
3. **命名避让**：自建组件勿与既有全局组件撞名（已踩 `IconButton`→改 `AppIconButton`）。新建前先 grep `components/**`。
4. **多根 class 不透传**：`AppDropdown` 含 `Teleport` 属多根，外部 `class` 不落到触发器 → 响应式隐藏要包**普通 div**（见 `AppTopbar .app-topbar__desktop`）。
5. **内容区无 keep-alive**：复刻原语义——导航即重挂载，`:key = tabsStore.getRouteRenderKey(fullPath)`；页面轮询靠 `onBeforeUnmount` 停（勿包 `<keep-alive>`）。
6. **store 契约不改**：`menuStore/tabsStore/themeStore/userStore/userProfileStore/useDashboardHomeStore` 全部复用，只换壳。图标经 `menuStore.iconComponents['HOutline:XxxIcon']`（Heroicons，Proxy 按需）。

### P1 列表/CRUD 范式（已定稿 · `system/user` 样板 2026-07-11）
**页结构**（`views/system/user/index.vue`）：`PageHeader`(标题+描述+`新增用户`) → `FilterBar`(用户名/状态+搜索/重置) → 批量条(选中出现，`PERM.USER_DELETE`) → 卡/表切换分段控件(桌面；移动强制卡) → `EntityCard` 卡片流 / `DataTable` 表视图 → `Pagination` → `FormDialog`(create.vue)。删除走 `Dialog.info` 确认，toast 用 `ElMessage`(迁移期)。
**新增 `components/ui/`**：`PageHeader`、`FilterBar`(移动折叠)、`Pagination`(移动简化)、`DataTable`(令牌驱动语义表格：选择/sticky 表头/hover/空/加载/序号)、`ActionMenu`(基于 `AppDropdown` 的行内⋯)、`FormDialog`(令牌驱动模态外壳)。表单控件用 `design-tokens.css` 里的全局 `.app-input/.app-select/.app-textarea/.app-label`。
**务实取舍**（延续 Phase 1 自建令牌驱动路线，见下表）：
- `DataTable` **自建令牌表格**（非 `UTable`/TanStack 封装）——完全掌控观感、免 UTable API 试错、与卡片流一致。桌面渲染，移动由页面切卡片。
- 表单字段用**原生令牌化控件**（非 `UInput`/`USelect`）——免 U* 表单主题不齐；`UButton` 仍用于按钮，`i-lucide-*` 图标(iconify)。
- 校验**内联函数**（非 valibot schema）——移植原 rules；`schemas/*`+valibot 留待后续。
- 删除确认复用 `Dialog.info`、toast 复用 `ElMessage`（迁移期共存，全局横切后续统一换）。

### 下一步：Phase 3 全量推广（系统管理列表页已全部完成 → 接 `system/settings` 或 实例/文件/Docker 模块）
- 复用上面这套 P1 范式推平系统管理其余页（菜单/OIDC/任务/操作日志）与实例/文件等模块。
- ✅ **角色管理 `system/role` 已完成并浏览器验证**（2026-07-11）：新增自建 `components/ui/PermissionTree.vue`（令牌驱动三态勾选树，替代 `el-tree`；v-model=扁平 menuIds=全选∪半选，与后端语义一致）；`DataTable` 加 `rowSelectable` 逐行禁选（内置角色不可选/编辑/删除，用锁图标）；create.vue 用 FormDialog+PermissionTree。**踩坑**：后端 `role.go` 的 `description` 是 `NotEmpty()`，原表单未校验会 500——本次改为**必填**（内联校验 + `descriptionRequired` i18n）。
- 菜单/任务/操作日志是**表格主型**，直接用 `DataTable`（操作日志详情走弹窗，勿塞单元格）；菜单管理也可复用 `PermissionTree` 思路或树表。
- 待补：`FilterSheet`(移动底部 sheet，当前用内联折叠代替)、`schemas/*`+valibot、`utils/feedback.ts`(toast 收口)。

---

## Phase 0 · 地基(逐项)
- ⬜ 接入 spike:装 `@nuxt/ui` + `@iconify-json/lucide`+`heroicons`;Vite 插件;CSS 入口;`<UApp>`;`UButton` 跑通(见 `03b` §2)
- ⬜ 移除 EP 引导:`main.ts` EP 导入、`ElementPlusResolver`、`optimizeDeps` EP、`el-config-provider`→`<UApp>`
- ⬜ `design-tokens.css`:按 `01` 数值覆盖 `--ui-*` + `@theme`;主色 `#14B8A6` 可切换(改造 `themeStore.setPrimaryColor`→`--ui-primary`)
- ⬜ 图标:`config/iconMap.ts`(旧名→iconify);`UIcon`;删 `iconRegistry.ts`
- ⬜ 横切:`utils/feedback.ts`(toast)、`confirmDanger`、`useOverlay` 弹窗、`schemas/*`(valibot)
- ⬜ `components/ui/*`(按 `03`):IconBox/StatusPill/Chip/AppAvatar/Skeleton系/PageHeader/SectionHeader/AppPanel/Toolbar/MetricStrip+Metric/DataTable/EntityCard/DescriptionList/EmptyState/ErrorState/Pagination/FilterBar/FilterSheet/FormDialog/ActionMenu/FabButton
- ⬜ 明暗两态令牌核对

## Phase 1 · 外壳(逐项,见 `04`)
- ⬜ `layouts/app/AppSidebar` + `AppNav` + `AppNavItem`(手写导航,分组/激活/折叠/多级/权限过滤/图标/i18n)
- ⬜ `AppTopbar`(标题+面包屑+搜索+操作+头像;`I18nDropdown/NotificationDropdown/UserDropdown` 基于 Nuxt UI 重写,保留数据逻辑)
- ⬜ `AppTabs`(极简,复用 `tabsStore`)
- ⬜ `AppContent`(过渡/内边距/限宽/`fullBleed`/keepAlive)
- ⬜ `AppShell` 组装 + `layouts/index.vue` 切换
- ⬜ 移动:抽屉侧栏、压缩顶栏、隐藏/横滑标签
- ⬜ 桌面+移动+明暗截图,替换旧 `leftMode/topMode/menu/menuItem/header/tabsView`

---

## 逐页清单(Phase 2–3)

> 列:页面 · 路由 · 页型 · 规格 · 桌面 · 移动 · 暗色 · 验收

### 仪表盘/个人中心(`06a`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 工作台 | dashboard/home | P4 | ✅ | ✅ | ✅ | 🟡 待用户确认方向 |
| 分析页(待决) | dashboard/analysis | P5 | ⬜ | ⬜ | ⬜ | ⬜ |
| 监控页 | dashboard/monitor | P5 | ⬜ | ⬜ | ⬜ | ⬜ |
| 个人中心 | profile | P2 | ✅ | ✅ | ✅ | ✅ **06a 真重写+完整浏览器验收**：桌面1440 双栏/操作链路；移动390 emulate 编辑弹窗/设备卡/日志卡/下线确认；768 单列；修设备网格 minmax(0,1fr) 横溢 + 面板头窄屏；已提交 |

### 实例/基建(`06b`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 应用列表 | instance/list | P1 | ✅ | ✅ | ✅ | ✅ |
| 实例控制台 | instance/console/:id | P6 | 🟡 | ⬜ | 🟡 | 🟡 PTY 直连已实机验证(键入/回显/ws 抓包);移动+浅色待验 |
| 实例文件 | instance/files/:id | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例类型 | instance/type | P1 | ✅ | ✅ | ✅ | ✅ |
| Docker 容器 | docker/container | P1 | ✅ | ✅ | ✅ | 🟡 全量重写(列表+创建/编辑结构化表单+详情+TerminalConsole 日志/终端+Stats FormDialog)；浏览器桌面/移动/暗色验证；后端 docker 未启用，数据流未跑 |
| Docker 镜像 | docker/image | P1 | ✅ | ✅ | ✅ | 🟡 结构+弹窗验证（后端无 Docker 守护，数据流未跑） |
| Docker 网络 | docker/network | P1 | ✅ | ✅ | ✅ | 🟡 结构+弹窗验证（后端无 Docker 守护，数据流未跑） |
| Docker 配置 | docker/config | P3 | ✅ | ✅ | ✅ | 🟡 P3 四 Tab(状态/连接/默认/仓库) + AppPanel；浏览器验证；docker 未启用时状态正确显示 |
| API Key | node/key | P1 | ✅ | ✅ | ✅ | ✅ |
| SSH 管理 | openssh/management | P1 | ✅ | ✅ | ✅ | ✅ |
| SSH 终端 | openssh/terminal | P6 | 🟡 | ⬜ | 🟡 | 🟡 重写完成,htop+鼠标滚轮+初始 resize 实测通过;移动+浅色待验 |

### 工具/文件/Sub2API(`06c`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 内网穿透 | tools/tunnel | P1 | ✅ | ✅ | ✅ | 🟡 全量重写(列表+创建/frps/frpc/统计)；浏览器桌面/移动/暗色验证；**未提交，待用户检查** |
| 端口转发 | tools/port-forward | P1 | ✅ | ✅ | ✅ | 🟡 全量重写(列表+创建/统计)；浏览器桌面/移动/暗色验证；**未提交，待用户检查** |
| 文件管理 | file/index | P6 | ✅ | ✅ | ✅ | 🟡 真·从零重写完成；浏览器验：桌面/暗/网格/移动、toast、FormDialog、**FileEditor 创建即打开/重命名弹窗/主题分段/删除确认+删除成功**；修 joinPath 盘符根、AppDropdown 双 toggle、移动表头列；后端 `pkg/file.Create` 吞错已改**需用户重跑 Go**；未提交 |
| 文件分享 | file/share | P1 | ✅ | ✅ | ✅ | ✅ 桌面/移动/暗+创建/编辑弹窗全验;**有界内容**(列 `N 项·首名`+title 全清单;弹窗"已选N项+定高滚动+逐项移除")、**无序号** |
| 文件来源 | file/source | P1 | ✅ | ✅ | ✅ | ✅ 桌面/创建弹窗(动态 OSS/FTP/WebDAV 字段+AppSelect 浮层)验;移动暗沿用已验范式 |
| Sub2API 首页 | sub2api/home | P4 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 配置 | sub2api/config | P3 | ✅ | ✅ | ✅ | 🟡 P3 三 Tab(连接/首页/生图)去 EP：`AppPanel`+`set-row`+`AppSwitch`+展示分组 toggle-chip+可移除站点 chip；`PageHeader#actions` 同步状态 `StatusPill`；`useFeedback`。桌面1440/移动390(emulate)/明暗+真实后端(已连接)验；`configPageTitle` 三语；未提交 |

### 系统管理(`06d`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 用户管理 | system/user | P1 | ✅ | ✅ | ✅ | ✅ |
| 角色管理 | system/role | P1 | ✅ | ✅ | ✅ | ✅ |
| 菜单管理 | system/menu | P1 | ✅ | ✅ | ✅ | ✅ |
| OIDC 客户端 | system/oidc | P1 | ✅ | ✅ | ✅ | ✅ |
| 系统设置 | system/settings | P3 | ✅ | ✅ | ✅ | ✅ |
| 定时任务 | system/task | P1 | ✅ | ✅ | ✅ | ✅ |
| 操作日志 | system/operation | P1 | ✅ | ✅ | ✅ | ✅ |

### 认证/独立(`06e`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 登录/注册/找回 | login | P7 | ✅ | ✅ | ✅ | 🟡 居中卡重写完成；桌面/移动/暗色+登录链路已验；未提交 |
| 初始化向导 | initialize | P3 | ✅ | ✅ | ✅ | 🟡 居中卡三步向导去 EP；mock 全链路(库选/测试/管理员/确认/完成)验；桌面/移动/暗；未提交 |
| 分享落地 | public/share/:token | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| OIDC 授权 | oidc/authorize | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 门户 | public/sub2api/home | P7 | ✅ | ⬜ | 🟡 | 🟡 去 EP+拆接口+渐进渲染（桌面已验；移动/全明暗待抽；**未提交**） |
| Sub2API 统计 | public/sub2api/stats | P7 | ✅ | ✅ | ✅ | 🟡 **完整验收 2026-07-20**：桌面1440 暗/浅 + 移动390 EN + 语言/主题/返回首页 + 区间7→1 + Metric 5 SVG 图标 + stagger + loading 区 + ep=0 + home/stats 200；vue-tsc/eslint 绿；**未提交** |
| Sub2API 绘图 | public/sub2api/imagine | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| 403 | exception/403 | P8 | ✅ | ✅ | ✅ | 🟡 去插画克制居中+回首页/上一页；桌面/移动/暗验；未提交 |
| 404 | exception/404 | P8 | ✅ | ✅ | ✅ | 🟡 同 403 范式；桌面验；未提交 |
| redirect | redirect | 工具 | ✅ | — | ✅ | 🟡 令牌页底+logo 脉冲占位；未提交 |

---

## 待决(与用户确认后处理,Phase 4)
- ⬜ `dashboard/analysis` 假数据页:删除 or 改造真实分析?
- ⬜ `dashboard/monitor` 是否含假数据面板(核对 `stores/dashboard/monitor`)
- ⬜ `views/demo/*`(VXE 演示)——随 VXE 下线删除
- ⬜ `views/extended/*`(组件演示)——删除
- ⬜ 顶部导航布局(topMode)是否保留;ThemeConfig 是否精简
- ⬜ 表格默认视图(卡/表)与是否提供切换的最终口径
- ⬜ EP/VXE 彻底卸载 + 全仓 grep(`el-`/`El*`/`ElMessage`/`v-loading`/`vxe`/`@element-plus`)清零

---

## 本次样板的务实取舍(待与用户确认，确认后回填/修订计划)
> 目标是"最快看到真实设计方向"，故对少数非视觉项做了务实取舍。方向确认后可按需回正：
- **图标**：暂**复用现有 `iconComponents`(Heroicons，Proxy 按需加载)**，未做 `iconMap.ts` + `UIcon` 迁移。设计主图标本就是 Heroicons outline，视觉一致；iconify 迁移留待正式 Phase 0 收尾。
- **下拉弹层**：自建**令牌驱动 `AppDropdown`**(onClickOutside+Teleport)承载 语言/通知/用户/命令搜索，未用 `UDropdownMenu`/`UPopover`(避免库 API 试错、完全掌控观感)。可后续替换为 U* 或保留。
- **移动抽屉**：复用现有 **EP `el-drawer`** 承载侧栏(可靠、迁移期共存)；正式可换 `USlideover`。
- **实时图表**：`systemRealtimeCharts` **仅重写 template(AppPanel+令牌化分段/原生 select)+样式，保留全部 ECharts 逻辑**；控件由 EP 换为原生+令牌。
- **ThemeConfig**：暂**原样复用**(EP 抽屉，齿轮打开)；`layout` 的 topMode 选项在新外壳下暂为空操作(任务书允许移除 topMode)。
- **i18n**：`搜索菜单…` 等 2~3 处样板文案用中文字面量(zh-CN)，未入 `messages.ts`；`theme.colors.teal` 选项键未加。正式需补 `messages.ts`。
- **主色**：默认改薄荷青 `#14B8A6`(Nuxt UI `colors.primary:'teal'`=teal-500 同值)；`themeStore` 仅改默认值，未把主色切换同步给 Nuxt UI 全阶(切色时 U* 组件仍为 teal)。
- **命名冲突**：`components/ui/IconButton.vue` 与既有 `components/button/IconButton.vue` 撞名，已改名 `AppIconButton`。后续新自建组件注意避让既有全局组件名。

## 会话日志(每次追加一行)
| 日期 | 会话做了什么 | 下一步 |
|---|---|---|
| (首个会话) | 探索期尝试(令牌/组件/仪表盘,已撤回);定方向(精致侧边栏 + Nuxt UI + EP/VXE 下线);编写完整任务书;还原基线 | 下会话从 Phase 0 接入 spike 开始 |
| 2026-07-10 | **样板落地**：Phase 0(Nuxt UI 4.9 接入+`design-tokens.css`薄荷主色+明暗)跑通;Phase 1 外壳(手写侧栏/顶栏/命令搜索/三下拉/极简标签/内容区/移动抽屉,`layouts/app/*`)+`components/ui/*` 10 个基础组件;Phase 2 工作台重写(安静问候+单色 MetricStrip+实时面板+运行实例 EntityCard/空态+快捷入口+系统信息)。浏览器验证桌面/移动×明/暗、抽屉、下拉、徽标均 OK,零控制台报错,新增文件 lint 绿。见"务实取舍"。 | 等用户确认设计方向;确认后据反馈修订计划,再做 用户管理(列表页样板)与全量推广 |
| 2026-07-11 | **用户管理列表页样板（P1 定稿）**：新增 `components/ui/` 六件套（PageHeader/FilterBar/Pagination/DataTable/ActionMenu/FormDialog）+ 全局令牌化表单控件；重写 `system/user`（卡/表切换、批量删除、Dialog.info 删除确认、内联校验的 create/edit FormDialog），保留 userPage/deleteUser 接口与 PERM 权限。浏览器验证桌面卡/表、创建/编辑弹窗+校验、行内⋯菜单、批量条、分页、移动强制卡+筛选折叠、明/暗全 OK，零控制台报错，lint+tsc 绿。见"P1 范式"与务实取舍。 | Phase 3 全量推广（从 `system/role` 起） |
| 2026-07-11 | **Phase 3 首页-角色管理 `system/role`**：按 P1 范式重写 index（卡/表切换、批量删、Dialog.info 确认）+ create（FormDialog+校验）。新建自建令牌组件 **`components/ui/PermissionTree.vue`**（三态勾选树替代 el-tree，v-model=扁平 menuIds）；`DataTable` 加 `rowSelectable` 逐行禁选（内置角色锁图标、不可选/编辑/删除）。修复原表单 description 未必填导致的 500（后端 `NotEmpty`）→ 改必填。i18n 补 role.title/pageDesc/selectedCount/clearSelection/emptyDesc/descriptionRequired + 树控件 selectAll/clearAll/expandAll/collapseAll（三语）。浏览器验证：桌面卡/表、创建/编辑权限树全链路（父子联动+半选+后端往返恢复）、批量删、移动强制卡+折叠筛选、明/暗全 OK，lint+tsc 绿。 | 接 `system/menu`（菜单管理，表格主型）继续 Phase 3 |
| 2026-07-11 | **用户反馈-头像收敛**：原三处头像(顶栏右上角 `UserMenu`、工作台问候区、侧栏底部)统一收敛到**仅侧栏底部**。`UserMenu` 加 `variant='topbar'\|'sidebar'`——侧栏整行触发器、向上弹出,展开显示 用户信息/操作/版本号;`AppDropdown` 加 `side='top'`(向上弹)+`block`(填充 flex 父)。顶栏移除 `UserMenu`、工作台移除 `AppAvatar`。下拉底部常驻"版本号: dev";原"版本号"动作行改叫`layout.checkUpdate`(zh-CN/zh-TW/en 已补)避免重复。浏览器验证桌面(展开/折叠)×明/暗 + 移动抽屉均 OK,零控制台报错,lint+tsc 绿。 | 用户管理(列表页样板);顶栏右上角现仅剩 通知铃(用户菜单已下移) |
| 2026-07-11 | **菜单管理 `system/menu`（树表）**：`DataTable` 加 `tree` 树表模式（缩进+展开插入符+扁平化 displayRows）；index 用树表 + 客户端筛选（保留匹配祖先），create 用 FormDialog+令牌字段（类型/父级/条件路径/权限/图标/排序/状态+内联校验），父级选择用**扁平缩进下拉**（后接改为 AppSelect）规避 el-tree-select 嵌套层级；复用 IconSelectorDialog（spacious）。i18n 补 menu.title/pageDesc/topLevel/emptyDesc（三语）。 | 用户反馈"密度" → 全局紧凑化 |
| 2026-07-11 | **用户反馈-密度太松 + 原生 select 直角浮层（两次强调，所有已重写页通病）**：全局紧凑化（`design-tokens.css` 控件 36→32/字号 14→13/label 13→12；`DataTable` 行 padding 11→7px；`FilterBar` 16→12px 去阴影；段间距 16/14→12/10；`AppIconButton` 加 `box`、`ActionMenu` 用 28px → 表行 63→43px）。新建 **`components/ui/AppSelect.vue`**（令牌圆角下拉，teleport z-index 2300、泛型 v-model 类型安全、`fit`/`searchable` 本地过滤），**替换全部原生 `<select>`**（两筛选栏/菜单父级/用户角色/分页每页）。记忆 [[compact-density-appselect]]。**操作日志 `system/operation`**（P1 只读）：PageHeader+FilterBar（可搜索用户/类型 AppSelect+布尔结果+路径）+DataTable/移动卡+Pagination+详情 FormDialog(JSON+复制)；用户预加载一次做 userId→用户名 映射。i18n 补 operation.title/pageDesc/emptyDesc/allUsers/allTypes（三语）。浏览器验证 menu(树表/创建弹窗内 AppSelect 浮层高于弹窗/筛选) + role(卡/分页 AppSelect) + operation(搜索下拉/详情弹窗) + 明暗，lint+tsc 绿。 | 继续 Phase 3：`system/task` 或 `system/oidc` |
| 2026-07-11 | **用户反馈-总条数显示两次**：`Pagination` 组件自带的“共 N 条”与各页顶部 `X-page__bar-hint` 的总数重复（user/role/operation 都中招；menu 是树无分页故单次）。**统一从 `Pagination` 移除总数**（一处改动全解决）→ 全部页面总数只在顶部显示一次。**定时任务 `system/task`（P1 只读+行内操作）**：PageHeader(+刷新)+FilterBar(关键字+状态 AppSelect)+DataTable/移动卡+Pagination；令牌进度条（track+fill 按状态着色）、种类/状态 StatusPill、行内 `ActionMenu` 取消/重试/删除（按状态条件显示：active→取消、failed→重试、terminal→删除；删除走 Dialog.info）、2.5s 静默轮询。i18n 补 taskManager.pageTitle/pageDesc/allStatus/emptyDesc（三语）。浏览器验证桌面(表/进度/条件操作菜单)+移动卡+暗色，总数单次，lint+tsc 绿。 | 继续 Phase 3：`system/oidc`（系统管理最后一个列表页） |
| 2026-07-11 | **OIDC 客户端 `system/oidc`（P1，系统管理最后一个列表页）**：按 P1 范式重写 `index.vue`（PageHeader+两操作按钮 OIDC配置/生成客户端 → FilterBar → DataTable/移动卡 → Pagination）+ 三个子件：`configDialog.vue`（服务端配置：AppSwitch 启用 + Issuer/当前域名 + 3 列 TTL + 只读端点复制区）、`clientForm.vue`（创建/编辑：名称/回调必填内联校验 + Scopes + AppSwitch 状态；保存后若返回**完整** secret(无`*`)则 emit reveal）、内联 reveal `FormDialog`（完整 Client Secret 仅一次 + 13 端点字段 + 单项/全部复制）。行内 编辑/刷新密钥/删除（Dialog.info 确认）。新建自建令牌组件 **`AppSwitch.vue`**（替代 el-switch）。文案硬编码中文（唯一 i18n key=`system.common.total`；Phase 4 统一）。保留 list/create/update/delete/refreshSecret/getConfig/updateConfig 契约 + `PERM.OIDC_EDIT`。**浏览器全链路验证**：桌面暗(列表/配置弹窗/创建→reveal 完整secret/行菜单/校验/删除确认)+桌面浅+移动390(强制卡/折叠筛选/简化分页)，创建真实客户端走通后删除清理，零控制台报错，vue-tsc+eslint 绿。**⚠️ 本页及 role/menu/operation/task 全部 Phase 3 改动仍未提交。** | Phase 3：`system/settings`(P3 配置型) 或转 实例/文件/Docker 模块；提交 Phase 3 批次 |
| 2026-07-12 | **提交系统设置**（`f67e374`：`system/settings` P3 双 Tab + AppPanel 页脚 flex + i18n）→ 系统管理模块 100% 收口。**转 实例模块：应用列表 `instance/list`（P1，旗舰页）全量重写**：4 文件套 P1 范式——`index.vue`（PageHeader+FilterBar+批量条+卡/表切换+DataTable/卡片流+Pagination+受控 InstanceEditor，薄壳保留 `useInstanceListStore` 全部状态/动作契约与业务处理）、`instanceCard.vue`（EntityCard+StatusPill+ActionMenu+UButton，去 el-button/checkbox/dropdown/icon）、`instanceEditor.vue`（FormDialog+令牌字段+AppSelect+AppSwitch+内联校验，保留受控 props/emits）。表视图行操作收进 ActionMenu（控制台/配置/启动或停止/强制重启/文件管理/删除，删除按 owner 隐藏）。i18n 补 instance.title/pageDesc/allTypes/allStatus/emptyDesc/createTime（三语）。浏览器验证桌面暗（卡/表切换、行菜单全项、配置弹窗 fetch 回填 + 类型 AppSelect 浮层高于弹窗）+移动390（强制卡/折叠筛选/简化分页），零控制台报错，vue-tsc+eslint 绿。**⚠️ 用户第三次密度反馈**："总览占用大量空间 + 大组件大留白"——**删掉大号 MetricStrip 总览带**（P4 仪表盘才配），把 当前页运行/停止 计数**内联进列表工具条**（`共 N 条 · ●运行中 X · ●已停止 Y`，总数不再与「总实例」重复），删 `overviewStats.vue`。见 [[compact-density-appselect]] 新增「列表页禁大指标带」条。**未提交。** | 转 实例其余（`instance/type` 已存在待核）与 Docker/Node/工具/文件 P1 列表；套本页范式 + 密度铁律 |
| 2026-07-12 | **Docker 列表两页（image/network）**：按 P1 范式重写 `docker/image`（PageHeader[任务/拉取镜像]+FilterBar[关键字/悬空 AppSelect]+DataTable/移动卡+Pagination；拉取/打标签/编辑标签/详情[令牌 KV+chips]/历史[DataTable] 全部 FormDialog；行内 ActionMenu 详情/编辑标签/打标签/历史/删除，管理权限 + 容器数门控）与 `docker/network`（同范式+创建[名称/驱动 AppSelect/内部·IPv6·可附加 AppSwitch/子网/网关]/详情[hero+KV+IPAM 列表+标签+已连接容器带断开]/编辑[标签+重建开关→条件驱动/强制/子网/网关] FormDialog）。**沿用** `DockerTaskDialogs`（拉取/重建进度，迁移期 EP 保留）。i18n 三语补 docker.image/network title+pageDesc+emptyDesc。**后端无 Docker 守护** → 仅验证页面结构+空态+弹窗渲染（拉取/创建弹窗桌面暗已截图），行/详情数据流未跑；vue-tsc+eslint 绿。**未提交前项已提交 `c6e1a14`；本两页待提交。** 剩 `docker/container`（70KB 巨页：列表+创建表单+exec/logs/stats 终端弹窗，exec/logs 应复用 TerminalConsole）+ `docker/config`(P3)。 | Docker 容器(大)+配置；或转 工具/文件 | 
| 2026-07-12 | **P1 推广三页 + 新增 UserPicker**：①**实例类型 `instance/type`**（PageHeader+FilterBar[名称/状态]+内联计数[启用/禁用]+DataTable/移动卡+FormDialog[名称+AppSwitch 启用]，内置类型锁定，删除走 Dialog.info）。②**API Key `node/key`**（同范式+原生 `datetime-local` 过期时间+AppSwitch 永久有效[仅编辑]+复制 FormDialog[完整值一次+警示条]；行内 编辑/复制/刷新；**无删除接口**属原样保留）。③**SSH 管理 `openssh/management`**（可选表+批量测试/删除+行内 连接/测试/编辑/分享/删除[仅创建者]+分享 FormDialog；create 大表单 authType seg+条件密码/密钥[眼睛显隐]+patch-diff 更新；createTime 格式化修复宽度溢出）。新增自建 **`components/ui/UserPicker.vue`**（多选用户：预载全部用户 chips+可搜索 AppSelect 追加，替代 el-select multiple remote，SSH create/分享共用）。**踩坑**：`system.common.selectAll` 不存在（selectAll 在别的 `common` 命名空间）→ 移动端全选按钮显示原始 key，改用 `ssh.common.selectAll`（自加）。i18n 三语补 instance.type*/node.key.title+pageDesc/ssh.common.title+pageDesc+selectAll 等。**浏览器全验**：三页桌面暗/浅+移动，CRUD/弹窗/校验/批量/分享全链路，vue-tsc+eslint 绿。**未提交。** | 转 Docker(容器/镜像/网络+config) 或 工具(tunnel/port-forward) 或 文件(share/source) P1 列表 |
| 2026-07-11 | **提交 Phase 3 批次**（`902082e`：role/menu/operation/task/oidc + AppSelect/AppSwitch/PermissionTree + 密度化 + i18n，24 文件）。**系统设置 `system/settings`（P3 双 Tab 配置页，系统管理最后一页）**：EP 整页重写为 PageHeader + 令牌 seg Tab 条（安全与认证/邮件配置）+ `AppPanel`(flush) 分组 + `.set-row`（label+desc / 控件，分区保存页脚）。控件：`AppSwitch`(替 el-switch)、令牌 `.app-input`/number/`.app-textarea`、`AppSelect`(替 el-select，模板类型；`@change` 不存在→用 `watch` 承接重载)、密码眼睛显隐、占位符 chips 快捷插入。3 个 `FormDialog`：测试邮件/模板测试(动态占位符字段)/预览(iframe srcdoc + 占位符黄底高亮)。保留全部 login/email/template API + `PERM.SYSTEM_CONFIG_EDIT` + 占位符插入/渲染逻辑。i18n 补 `settings.pageTitle/pageDesc`(三语)。**踩坑**：模板测试字段标签 `{{ '{{.'+name+'}}' }}` 在 Vue 文本插值里字面 `{{` 触发 eslint 解析错误 → 抽 `fieldToken()` helper 返回该串。**浏览器全链路验证**：桌面暗/浅(双 Tab、开关/输入/密码眼睛、AppSelect 切换 watch 重载、预览 iframe 高亮、模板测试动态字段、空收件禁用发送)+移动390(行竖堆/Tab 撑满/控件全宽)，零控制台报错，vue-tsc+eslint 绿。**⚠️ 未提交。** | Phase 3 转其余模块：实例/Docker(多 P1 列表)、文件分享/来源、工具、仪表盘 profile/analysis/monitor |
| 2026-07-13 | **文件模块两页 P1 + 两条新铁律(用户点名)**：①**文件来源 `file/source`**（index + 新建 `create.vue`：FormDialog + 按类型动态字段 OSS/FTP/WebDAV + AppSelect/AppSwitch + 页脚测试连接；行内 测试/编辑/删除；内联启用/停用计数）。②**文件分享 `file/share`**（PageHeader+FilterBar+DataTable/移动卡+Pagination；有效/停用/过期/达上限 状态胶囊；复制链接/编辑/删除）+ **`ShareFormDialog` 从 EP BaseDialog 重写为 FormDialog**（令牌字段 + 原生 datetime-local + AppSwitch + 创建成功详情复制视图；保留 FilePicker 跨来源选择）。i18n 三语补 fileSource.title/pageDesc/keyword… + file.share.pageDesc/status*/content/itemsCount/selectedCount…。**用户中途两次点名的通病 → 提炼成 `01` 铁律 4「为使用者策展,不照搬 API」**：(a)**可变长内容有界化**——分享多文件不再逗号铺满/chips 无限堆叠,列表列改「`N 项 · 首名`(全清单走 title 悬浮)」、编辑弹窗改「已选 N 项 + 定高滚动列表 + 逐项移除」;(b)**去系统内置量**——`DataTable` 的 `seq` 序号列**全量下线**(14 个已重写列表页 sed 清除:docker×3/instance×2/node/ssh/system×4/tools×2/file×2)。踩坑:FormDialog 自定义 `#footer` 必须保留默认 `show-footer=true`(slot 在 `v-if="showFooter"` 内,设 false 会连按钮一起吞掉);`noUncheckedIndexedAccess` 下 `list[0]` 要先取局部变量再判空。浏览器全验(桌面/移动/暗+弹窗),vue-tsc+eslint 绿。**未提交。** | 主攻 `file/index` 文件管理大重写(用户点名);顺带 review 前两次提交(tunnel/port-forward + Docker) | 
| 2026-07-12 | **Docker 全模块收口**：①**容器 `docker/container` P1 全量**——PageHeader+FilterBar+DataTable/移动卡+Pagination；创建/编辑 FormDialog 结构化控件（`AppCombobox` 镜像、重启策略 seg、内存/CPU chips+数字、`ContainerPort/Mount/EnvEditor`）；详情令牌壳；日志/终端复用 `TerminalConsole` 浮层；统计 FormDialog+echarts。②**配置 `docker/config` P3**——四 Tab（状态/连接/默认/仓库）+ AppPanel+AppSwitch。③新组件 `AppCombobox` + 三个 spec 编辑器。**反人类修复**：列表失败置 `dockerUnavailable`，创建弹窗不再二次请求镜像/网络选项（拦截器 500 会连弹 toast）。浏览器：桌面/移动/暗色+创建表单结构化行 OK；后端 docker 未启用。vue-tsc 绿。**待提交本批。** | 提交 Docker 批次后停（用户指定）；下一会话 工具 tunnel/port-forward 或 文件 share/source |
| 2026-07-15 | **登录门面 `login` P7 重写（用户嫌半屏渐变难看 → 改居中卡）**：四文件去 EP——`index` 冷灰页底 + 居中 hairline 卡 + 角标(语言/主题配置/明暗)；`accountLogin/register/forgotPassword` 令牌 `.app-input` + 内联校验 + 密码眼睛内嵌 + `useFeedback` toast。**不做**半屏主色品牌区（空、廉价）。桌面 1440 / 移动 390 **均垂直居中**；暗色随 `themeStore`。登录 admin 链路 OK。i18n 补 slogan/copyright/toggleTheme/showPassword/hidePassword/passwordLabel（三语）。**未提交。** | 提交登录批次；下一页 `initialize` 或 个人中心/监控/Sub2API/public |
| 2026-07-15 | **初始化向导 `initialize` P3 重写（对齐登录门面）**：单文件去 EP——冷灰页底 + 居中 hairline 卡 + 角标(语言/主题/明暗)；自建紧凑步骤条(完成√/当前主色)；`AppSelect` 库类型 + 令牌表单(远程库条件字段+密码眼睛) + 内联校验；确认摘要 DescriptionList 风 + 结果态(成功/失败/轮询旋转)；`useFeedback` toast；版权复用 `login.copyright`。业务契约保留(status/test/confirm + 重启轮询)。i18n 补 `pageTitle/pageDesc`(三语)。浏览器：mock 拦截 `code:200` 全链路(SQLite→管理员→确认→完成)、桌面/移动/暗色，vue-tsc+eslint 绿。**未提交。** | 提交 initialize；下一页 个人中心 `profile` / 异常 403·404 / public/share 或 Sub2API |
| 2026-07-15 | **异常/工具页 P8**：`exception/403`/`404` 去掉 ~17KB 彩色插画 SVG，改为克制居中（大号码 + 标题 + 说明 + 回首页/上一页）；`redirect` 令牌页底 + logo 脉冲。i18n 补 forbiddenTitle/notFoundTitle/goHome/goBack（三语）。浏览器桌面/移动/暗色验，vue-tsc+eslint 绿。**已提交 `8f54c27`（与 initialize 同批）。** | 个人中心 `profile` |
| 2026-07-15 | **个人中心 `profile` 第一次换壳**（五 Tab 平铺，未对齐 06a）→ 用户点名返工。 | 按 06a 真重写 |
| 2026-07-15 | **个人中心 `profile` 按 06a 真·从零**：①克制浅主色条头部+编辑资料/改密 ②**左栏常驻**档案 DescriptionList + 桌面内嵌编辑；**右栏 Tabs** 仅 消息→权限→设备→日志（`personalInfo` 移出 store tabs） ③设备 **EntityCard 网格** + AdaptiveConfirm 下线 ④消息/设备危险操作用 AdaptiveConfirm ⑤移动：头压缩、Tab 横滑、列表转卡、**编辑 FormDialog 近全屏** ⑥改密 FormDialog+内联校验；去掉未实现危险区注销。i18n 补 editProfile/logoutDevice/thisSession。**完整浏览器验收**（用户要求后补做）：桌面1440 保存确认/一键已读/发消息/改密校验/下线确认/日志详情；移动390 编辑弹窗+设备卡+日志卡+移动下线 Dialog；768 单列无横溢；修 `dev-grid` `minmax(0,1fr)` 与 AppPanel 窄屏 head wrap。vue-tsc+eslint 绿。 | 下一页 监控/Sub2API/public |
| 2026-07-13 | **文件管理 `file/index` 真·从零重写 + 用户点名两条刚性标准**：全套文件组件除净老框架（EP-free）：`FileBrowser.vue`（外壳=来源 AppSelect/前进后退上级/面包屑/刷新 + 双栏 左树+右列表·网格 + 工具栏 UButton/AppDropdown更多/搜索/筛选/视图段 + 选择条 + 页脚 Pagination；全出血 `flex:1;min-height:0` 填满高度[修 278px 塌陷]；图标走 `menuStore.iconComponents['HOutline:*']`；删除确认=内联令牌 FormDialog）、`FileTree/FileTreeNode`（→ 可覆盖 `--ft-*` 局部变量+app 令牌兜底，heroicons）、`FilePromptDialog/FileMediaDialog/FileUploadDialog`（→ FormDialog+app 令牌+AppSelect）、**`FileEditor` 重写**（自包含 `--fe-*` 独立明暗主题[非旧 fm-*]、menuStore+本地保存 SVG、主题**分段**替代旧 FileMenu、`useFeedback`+Promise 门闸令牌确认）、`useFileUpload`（1 处 ElMessage→fb）。**新建 `src/utils/feedback.ts` = `useFeedback()` 封装 Nuxt UI `useToast`（全项目首个采用者，已浏览器实证渲染）**；`ElMessage`→`fb`、`showRequestError`→`fb.error(getRequestErrorMessage())`、`Dialog.confirm`→令牌 FormDialog。**删** orphaned `FileMenu.vue`；`icons.ts` 回退 HEAD（去 2 个没用的 view 图标）。**踩坑**：移动端表头 类型/时间 列 `.fb-thead .fb-col{display:flex}`(0,2,0) 盖过 `.fb-col--type{display:none}`(0,1,0)→ 表头列不隐藏挤压，改用 `.fb-thead .fb-col--type` 提特异性。**保留** `theme.css`(fm-*，薄荷)——仍服务未重写的 `FilePicker/FileViewerDialog/FileDialog/FilePager`(public/share 用，增量待迁)。**浏览器验证**：桌面浅/暗、网格、移动(tree 隐藏/单列/无横溢)、toast 渲染、全出血填高、新建文件夹 FormDialog；vue-tsc+eslint 绿。**未提交。** 记忆 [[rewritten-components-drop-all-legacy-framework]]。**浏览器 MCP 坑**：chrome-devtools 读默认 profile 的 DevToolsActivePort；用户 Chrome 无 debug 端口→ 另起隔离 debug Chrome `:9222`（`$TEMP\momoko-devtools-profile`）+ 把默认 profile 里陈旧 DevToolsActivePort 改指 9222。 | ①**用户重跑 Go 后端**以吃到 `pkg/file.Create` 吞错修复 ②提交本批文件管理重写 ③Monaco 自动化键入未通（保存 dirty 态未 E2E 到，手工点一下即可）④Docker 复查修复（容器网络列无界/Stats echarts 暗色/硬编码「连接」/engine-down 空态）⑤剩余页面 |
| 2026-07-20 | **Sub2API 配置 `sub2api/config`（P3，去 EP 全量重写）**：`el-tabs/el-card/el-form/el-switch/el-input(-number)/el-checkbox-group/el-tag/ElMessage/v-loading` → 令牌三 Tab 分段(连接/首页/生图·`menuStore` 图标) + `PageHeader`(`#actions` 挂同步状态 `StatusPill`，`store.statusType` 的 `danger`→`error` 映射) + `AppPanel`(flush)+`set-row`(label+desc) + `AppSwitch` + 令牌 `.app-input/.app-textarea` + adminApiKey 眼睛显隐(`AppIconButton`) + **展示分组 toggle-chip 多选**(选中 teal 描边+淡底+勾) + **允许站点可移除 chip + 行内追加输入**。toast 用 `useFeedback`。保留全部逻辑/接口(`useSub2APIStore.loadAdmin/testConfig/saveConfig`、`configForm`、`groups`、`PERM.SUB2API_EDIT`、`__deleted__` 分组合并)。`configPageTitle` 补三语(原页标题误用「连接配置」)。**浏览器全验**：桌面1440(连接/首页/生图) + 移动390 emulate(连接/首页) + 明+暗 + 真实后端(已连接、正常状态、真实分组回填) + 追加/移除站点交互；vue-tsc + eslint 绿。**⚠️ 核对 `stores/dashboard/monitor.ts`：监控页 100% `Math.random` 假数据 → 登记 Phase 4 待决(勿精修)**。**未提交。** | Sub2API 首页 `sub2api/home`(P4 大页) / public/*；或 Phase 4 清 demo + 卸 EP |
| 2026-07-20 | **Sub2API 管理首页 `home` 去 EP 重写并提交（`2fb1fe0`）**：PageHeader(#actions StatusPill+同步)+令牌三 Tab(概览/公告/时间线)；概览=range seg+datetime-local+**MetricStrip 单色(修原彩色边框违规)**+AppPanel 图表(保留 ECharts)+最近请求 DataTable/移动卡+Pagination+详情 FormDialog(KV)；公告/时间线=DataTable/卡+ActionMenu+FormDialog CRUD。桌面/移动/明暗+真实后端全验。**随后用户提 3 诉求→牵出后端大重构（见上 🚧 区块）**：Top 图表改 token 且按 token 排序、最近请求多维筛选、tps 剔除 output<20、**全部聚合下沉 ent(禁 SQL/禁内存)、页面按模块拆接口**。已落：前端 token-Top+recent 筛选前端侧、proto optional 筛选字段(regen)、ent `bucket15m` 列(regen)+写入、`pkg/types` 接口重设计、`internal/data` 全部 ent 聚合方法实现、snapshot.go 排序改 token。**⚠️ 未完成：snapshot.go 仍调已删的 `RecordsSince` → 构建断裂；proto 管理端接口未拆；前端未拆 loadAdmin* + 未加 FilterBar。记忆 [[sub2api-backend-rewrite-mandates]]。** | **先修构建**：按 🚧「剩余步骤」1–7 完成后端拆分+前端拆分+FilterBar，用户重跑 Go+全量重同步回填 bucket15m，再全验 |
| 2026-07-20 | **Sub2API 后端聚合重写全链路修好（构建断裂→全绿）**：①`snapshot.go` 去内存聚合、`BuildSnapshot/BuildStats` 改调 ent 聚合方法（`AggregateTotals/DailyTrend/TopItems/IntradaySeries/RecordsPage`）+映射助手（`mapTopItems/dailyTrendPoints/intradayTrendPoints/todaySeriesPoints/trendDayRange/dayStart`），删 `BuildRangeStats/totalsFromRecords/build*/records*/isRateEligible/tpsEligible`+`sort` import。②proto 删 admin `GetSub2APIStats`、加 `GetSub2APIAdminTotals/Trend/Top`（`/stats/totals\|trend\|top`），`make api`。③`pkg/service.go` 加 `adminWindow`+`AdminTotals/Trend/Top`、`RecentRequests(...,filter)`。④biz+service 3 handler + `RecordFilter{req.Model/GroupName/AccountName/Outcome(*string)}`。⑤前端 `api` 加 3 fn、`store` 拆 `adminTotals/adminTrend/adminModels/adminGroups`+`loadAdmin{Totals,Trend,Top}`（各自 loading）+`loadAdminRecent(...,filter)`+`buildStatsCards` 放宽 `StatsCardSource`、Top 图 `slice(0,10)`、下拉用全量枚举、`home` 三面板 `panel-loading`+最近请求 `FilterBar`(model/group/outcome 即时 + account 回车，改 filter 回第 1 页)+i18n `allModels/allGroups/allOutcomes/filterAccount` 三语。**`go build`/`vet`/`test`+`vue-tsc`/`eslint` 全绿；LSP "is not a type" 系 proto 重生后 gopls 陈旧误报（build/vet=0 证伪）。未提交。** | 用户重跑 Go + 全量重同步回填 `bucket15m` → 浏览器全验（三面板独立 loading/趋势桶/token Top/四维筛选链路/tps 口径）→ 提交本批 |
| 2026-07-20 | **Sub2API 后端重写浏览器实测通过 + 全局列表闪屏修复（用户点名）**：用户已重跑 Go+全量重同步；curl+浏览器实测新端点全对。**用户反馈整页闪** → 修共享 `DataTable`：骨架仅首屏，重载保留旧行+延迟 220ms veil；管理 home 概览改轻 spinner。已随 `c8d08f2` 提交。 | 公开门户 |
| 2026-07-20 | **公开门户 `public/sub2api/home` 阶段完成（未提交）**：用户点名 home 包太大 + 禁止整页等请求。**破坏性**：`Sub2APIHome` 删 `snapshot`；新 `GetPublicSub2APIOverview`(`/public/sub2api/overview`)；`PublicHome` 只元信息+公告/时间线。前端去 EP（令牌 btn/heroicons/StatusPill/EmptyState/FormDialog）；store 并行 `loadPublicHome`+`loadPublicOverview`+`loadStats(1)`，乐观外壳+hero 占位 `—`。实测：三请求 200、home≈251B、overview 填今日指标/曲线、stats 热门模型、公告弹窗、零 console。`go build`+`vue-tsc`+`eslint` 绿。 | `public/sub2api/stats` 去 EP；imagine/lottery/activity |
| 2026-07-20 | **公开统计 `public/sub2api/stats` 去 EP 全量重写（未提交）**：对齐 home 顶栏/令牌 btn/heroicons；`el-segmented`→令牌 `seg`；`el-table/el-empty/v-loading`→语义 table + 移动 rank + `EmptyState` + 轻 spinner；**MetricStrip 单色**（去 rainbow tone 顶边）；指标/列表渐进填 `—`，**禁止整页 loading**。保留 `loadStats`+range query。浏览器：桌面1440 暗（7 天 68k 请求/趋势/模型/分组/UA 表）+ 移动390 rank + 区间切今日 + 明暗切换；vue-tsc/eslint 绿、零 console。 | `public/sub2api/imagine` → lottery → activity |
| 2026-07-20 | **stats 返工补齐（用户点名）**：①`MetricItem` 加 `icon`（Heroicons 单色）+ stats 绑 store.icon；②区块 stagger 入场 `.reveal`；③顶栏改 `LanguageMenu`+`AppIconButton`（禁 EP）；④home 顶栏同步语言/主题/公告。浏览器再验：5 指标 SVG、语言 EN 切换、移动 rank、ep=0、零 console。 | imagine → lottery → activity |
