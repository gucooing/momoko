# Momoko 前端重写任务书 · 总纲与索引

> 本目录是 Momoko 前端**从零重写**的唯一事实来源(single source of truth)。
> 重写工作量巨大、横跨多天与多个会话,任何一次会话都必须**先读本目录**再动手,
> 并在收尾时更新 [`08-progress.md`](./08-progress.md) 的进度。
>
> 目标:一套**精致、清爽、自成体系**的设计语言 —— 中性色为主 + 一个克制的强调色、
> 细线分隔、充足留白、精致排版、极少阴影。**绝不**堆卡片、堆颜色、堆进度条与阴影
> (那是"廉价 AI 模板"的观感,是本次重写要彻底摆脱的东西)。
> 移动端是**一等公民**,每一页都必须有明确的移动端方案,不允许"桌面能用、移动端草草了事"。

---

## 0. 怎么用这份任务书

1. **每次会话开始**:读 `README.md`(本文件)→ `01-design-language.md` → 当前阶段相关的页面规格文档 → `08-progress.md` 看进度。
2. **动手前**:确认要做的页面在 `06-*` 里有规格;若没有,先补规格再写代码。
3. **写代码时**:严格遵循 `01`(设计语言)、`02`(移动端)、`03`(组件)、`07`(约定)。不要即兴发挥视觉风格。
4. **每完成一页**:在浏览器里(见下"可视化验证")桌面 + 移动两种视口各截图核对,达标后在 `08-progress.md` 勾选。
5. **发现规格缺陷**:直接修订对应文档并在 `08-progress.md` 记一笔,保持文档与实现一致。

### 可视化验证(浏览器 MCP)

- 已配置 `chrome-devtools-mcp`(local 作用域,`--autoConnect`)。**下个会话**启动 Claude Code 后可直接用
  `navigate_page` / `take_screenshot` / `resize_page` 等工具。
- 本会话使用常驻守护脚本 `.browser-tmp/drive-daemon.mjs`(已在后台)+ 瘦客户端 `.browser-tmp/b.mjs`:
  - `node .browser-tmp/b.mjs tour <url> [waitMs]` 打开页面并全屏截图
  - `node .browser-tmp/b.mjs call resize_page '{"width":390,"height":844}'` 切到移动端视口
  - 截图落在 `.browser-tmp/shots/`,用 Read 查看
- **每页验收必须包含**:桌面(1440×900)+ 移动(390×844)两种视口截图。
- 登录:`admin / admin`;前端 dev `http://localhost:3007`,后端 API `:22633`。

---

## 1. 文档地图

| 文件 | 内容 | 何时读 |
|---|---|---|
| `README.md` | 总纲、索引、阶段规划、验收标准 | 每次会话 |
| [`01-design-language.md`](./01-design-language.md) | 设计语言:令牌、色彩、排版、间距、层级、动效、图标、Do/Don't | 每次会话 |
| [`02-responsive-mobile.md`](./02-responsive-mobile.md) | 断点、外壳移动端、各页型移动端模式、触控、弹层/抽屉 | 做任何页面前 |
| [`03-components.md`](./03-components.md) | 通用组件目录 + API + 使用规则 | 写页面前 |
| [`04-app-shell.md`](./04-app-shell.md) | 外壳:侧边栏 / 顶栏 / 标签 / 内容区(桌面 + 移动) | 做外壳时 |
| [`05-page-patterns.md`](./05-page-patterns.md) | 页型范式(列表/详情/表单/仪表盘/终端/文件…)的标准结构 | 做页面前 |
| [`06a-dashboard-profile.md`](./06a-dashboard-profile.md) | 工作台 / 分析 / 监控 / 个人中心 | 做对应模块 |
| [`06b-apps-infra.md`](./06b-apps-infra.md) | 实例 / Docker / 节点 / OpenSSH | 做对应模块 |
| [`06c-tools-files-sub2api.md`](./06c-tools-files-sub2api.md) | 工具(隧道/端口转发) / 文件 / Sub2API | 做对应模块 |
| [`06d-system.md`](./06d-system.md) | 系统管理(用户/角色/菜单/OIDC/设置/任务/操作日志) | 做对应模块 |
| [`06e-auth-standalone.md`](./06e-auth-standalone.md) | 登录/注册/找回、初始化向导、public/*、OIDC 授权、异常页 | 做对应模块 |
| [`07-conventions.md`](./07-conventions.md) | 代码约定:目录、命名、i18n、权限、API、状态、弹窗、keepAlive | 写代码时 |
| [`08-progress.md`](./08-progress.md) | 逐页进度清单(活文档,每次会话更新) | 每次会话 |

---

## 2. 技术栈与约束(不变量)

- **Vue 3.5 + `<script setup>` + TS**,Vite 7,组件/自动导入(`unplugin-vue-components`、`unplugin-auto-import`)。
  `src/components/**` 下的组件**自动全局注册**(按文件名 PascalCase),`src/stores/**` 与 vue/router/pinia API 自动导入。
- **组件库 = Nuxt UI v4(latest,≥4.9)**,底层 **Reka UI + Tailwind v4**,在**纯 Vue + Vite** 下用
  (`@nuxt/ui/vite` 插件 + `@nuxt/ui/vue-plugin` + `<UApp>` 根)。详见 [`03b-component-library.md`](./03b-component-library.md)。
- **Element Plus 全部下线**、**VXE Table 全部下线**。所有 `el-*` → Nuxt UI `U*`;表格统一 `UTable`(经 `DataTable` 封装);
  `ElMessage/ElMessageBox`→`useToast()/UModal`;`v-loading`→骨架/遮罩;日期→Nuxt UI 日历;表单→`UForm`;
  图标 `iconRegistry`→`UIcon`(Iconify)。逐条映射见 `03b`。
- **保留** 的三方:**ECharts**(`vue-echarts`)图表、**Monaco** 代码/文件编辑、**xterm** 终端 —— 与组件库无关,不动。
- 后端驱动的**动态菜单 + 权限**:菜单来自 `menuStore.menuList`,路由由 `menuToRoute` 生成
  (`menu.path` → `src/views/<path>/index.vue`)。**页面文件位置与路由强绑定,不能随意挪动 index.vue**。
- 权限:`v-permission` 指令 + `PERM` 常量(`@/config/permission`)+ `useButtonPermission`。
- i18n:`vue-i18n`,`translateKnownText()` 兜底翻译;新增文案必须加 `src/locales/messages.ts` 的键。
- 主题:`useThemeStore`,主色 `--el-color-primary` 由 store 内联注入(**可切换**),明/暗/自动三态。

---

## 3. 设计基调一句话(贴在心里)

> **克制即精致。** 靠留白、排版、细线和一个强调色建立秩序;数字与层次自己会说话。
> 每加一个颜色、一层阴影、一个卡片之前,先问:"没有它会不会更干净?"

反面清单(出现即算跑偏,见 `01` 详解):
- 多个彩色图标底色并排(rainbow KPI 卡)
- 白卡叠白底、边界靠猜
- 到处 `box-shadow` 大软阴影
- 什么数据都塞表格 / 什么内容都塞进大卡片
- 渐变滥用、圆角忽大忽小、图标风格混用

---

## 4. 阶段规划(Rollout)

> 原则:**先地基,后样板,再全量**。地基(令牌 + 外壳 + 组件)决定所有页面的下限;
> 样板页(工作台 + 一个列表页)确认方向;之后按模块推平。每阶段结束都要桌面 + 移动截图验收。

- **Phase 0 · 地基(换库 + 令牌 + 组件)**
  - **Nuxt UI v4 接入 spike**(装包 + Vite 插件 + CSS + `<UApp>` + 明暗 + 主色,先跑通)——见 `03b` §2
  - 设计令牌 `design-tokens.css`:按 `01` 数值覆盖 Nuxt UI `--ui-*` + Tailwind `@theme`
  - 图标迁移(`iconMap`,`UIcon`/Iconify)、横切设施(toast/confirm/loading/form schema)——见 `03b`
  - 自建组件 `components/ui/*`(按 `03` 定稿 API,基于 Nuxt UI/Reka)
  - 约定与目录(`07`)
- **Phase 1 · 外壳(从布局重写)**
  - 新侧边栏(手写导航,分组、折叠、激活态)、新顶栏(标题+面包屑+搜索+操作+头像)、
    标签栏(极简可选)、内容区(留白 + 可选最大宽度)。桌面 + 移动(抽屉)。见 `04`。
- **Phase 2 · 样板页**
  - 工作台(仪表盘)重做为样板;用户管理(列表/CRUD)重做为列表页样板(桌面表/卡 + 移动卡)。
  - 截图给用户确认方向。
- **Phase 3 · 全量推广(按模块)**
  - 顺序建议:系统管理 → 实例 → Docker → 工具 → 节点/SSH → 文件 → Sub2API → 个人中心 →
    分析/监控 → 登录/初始化/public → 异常。
- **Phase 4 · 收尾**
  - 模板遗留 demo 页处理(`extended/*`、`demo/*`、`dashboard/analysis` 假数据页)——
    与用户确认后删除或改造(见 `08` 的"待决"清单)。
  - 暗色模式全量核对、可访问性(对比度/焦点/键盘)、空/错/加载态统一、i18n 补全。

---

## 5. 验收标准(每页必过)

1. **视觉**:符合 `01` 设计语言;与已完成页面观感一致(同一套语言)。
2. **响应式**:390 / 768 / 1024 / 1440 四档无破版;移动端按 `02` 的模式(表→卡、筛选折叠、触控 ≥44px)。
3. **状态**:加载(骨架/占位)、空(EmptyState + 引导)、错误(可重试)、无权限 都有明确表现。
4. **功能**:原有业务逻辑与接口**不回归**(重写只换壳与结构,不改后端契约)。
5. **暗色**:明/暗两态都正确。
6. **i18n**:无硬编码漏翻;新文案入 `messages.ts`。
7. **权限**:按钮级权限(`v-permission`/`PERM`)保持。
8. **性能**:`keepAlive` 语义不变;大列表虚拟化/分页保持。

---

## 6. 代码现状(重写已推进,务必以 `08-progress.md` 为准)

- **栈**:Vue 3 + Vite + **Nuxt UI v4** + 令牌 `components/ui/*`;**Element Plus / VXE 已卸载**。
- **外壳**:`layouts/app/*`(`AppShell` 等);旧 `menu/leftMode/topMode/...` 已删。
- **反馈**:`feedback` / `useFeedback()` + `FeedbackBridge`;toast `z-index` 高于弹窗。
- **活进度 / 交接**:每次会话先读 **`08-progress.md`** 的「一句话现状」。
- **基建**:`.browser-tmp/`(gitignored)、chrome-devtools MCP(`--autoConnect`)。
