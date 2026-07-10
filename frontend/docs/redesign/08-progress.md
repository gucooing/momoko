# 08 · 进度清单(Living Progress)

> **活文档**:每次会话收尾更新。状态图例:⬜ 未开始 · 🟡 进行中 · ✅ 完成(桌面+移动+明暗+验收过)。
> 验收标准见 `README` §5;每页需桌面(1440)+ 移动(390)截图核对。

---

## 里程碑

| 阶段 | 内容 | 状态 |
|---|---|---|
| 任务书 | `docs/redesign/*` 全套 | ✅ |
| 基线撤回 | 探索性改动还原 | ⬜ |
| Phase 0 | Nuxt UI 接入 + 令牌 + 图标 + 横切设施 + `components/ui/*` | ⬜ |
| Phase 1 | 外壳(侧栏/顶栏/标签/内容,桌面+移动) | ⬜ |
| Phase 2 | 样板页:工作台 + 用户管理(定方向) | ⬜ |
| Phase 3 | 全量推广(见下逐页) | ⬜ |
| Phase 4 | 清理待决 + 暗色/可访问性/i18n 全量核对 + **EP/VXE 彻底卸载** | ⬜ |

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
| 工作台 | dashboard/home | P4 | ⬜ | ⬜ | ⬜ | ⬜ |
| 分析页(待决) | dashboard/analysis | P5 | ⬜ | ⬜ | ⬜ | ⬜ |
| 监控页 | dashboard/monitor | P5 | ⬜ | ⬜ | ⬜ | ⬜ |
| 个人中心 | profile | P2 | ⬜ | ⬜ | ⬜ | ⬜ |

### 实例/基建(`06b`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 应用列表 | instance/list | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例控制台 | instance/console/:id | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例文件 | instance/files/:id | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例类型 | instance/type | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 容器 | docker/container | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 镜像 | docker/image | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 网络 | docker/network | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 配置 | docker/config | P3 | ⬜ | ⬜ | ⬜ | ⬜ |
| API Key | node/key | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| SSH 管理 | openssh/management | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| SSH 终端 | openssh/terminal | P6 | ⬜ | ⬜ | ⬜ | ⬜ |

### 工具/文件/Sub2API(`06c`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 内网穿透 | tools/tunnel | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 端口转发 | tools/port-forward | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 文件管理 | file/index | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 文件分享 | file/share | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 文件来源 | file/source | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 首页 | sub2api/home | P4 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 配置 | sub2api/config | P3 | ⬜ | ⬜ | ⬜ | ⬜ |

### 系统管理(`06d`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 用户管理 | system/user | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 角色管理 | system/role | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 菜单管理 | system/menu | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| OIDC 客户端 | system/oidc | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 系统设置 | system/settings | P3 | ⬜ | ⬜ | ⬜ | ⬜ |
| 定时任务 | system/task | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 操作日志 | system/operation | P1 | ⬜ | ⬜ | ⬜ | ⬜ |

### 认证/独立(`06e`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 登录/注册/找回 | login | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| 初始化向导 | initialize | P3 | ⬜ | ⬜ | ⬜ | ⬜ |
| 分享落地 | public/share/:token | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| OIDC 授权 | oidc/authorize | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 门户 | public/sub2api/home | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 统计 | public/sub2api/stats | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 绘图 | public/sub2api/imagine | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| 403 | exception/403 | P8 | ⬜ | ⬜ | ⬜ | ⬜ |
| 404 | exception/404 | P8 | ⬜ | ⬜ | ⬜ | ⬜ |
| redirect | redirect | 工具 | ⬜ | ⬜ | ⬜ | ⬜ |

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

## 会话日志(每次追加一行)
| 日期 | 会话做了什么 | 下一步 |
|---|---|---|
| (首个会话) | 探索期尝试(令牌/组件/仪表盘,已撤回);定方向(精致侧边栏 + Nuxt UI + EP/VXE 下线);编写完整任务书;还原基线 | 下会话从 Phase 0 接入 spike 开始 |
