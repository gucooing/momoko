# 06a · 逐页规格 · 仪表盘 / 分析 / 监控 / 个人中心

> 格式:**路由/文件 · 页型 · 桌面 · 移动 · 数据与逻辑 · 状态 · 权限 · 优先级**。
> 页型见 `05`,移动模式见 `02`,组件见 `03`/`03b`。"保留逻辑"= 只换壳,不改接口与业务流。

---

## 工作台 `dashboard/home` · P4 · 优先级:高(Phase 2 样板)

- **文件**:`views/dashboard/home/index.vue`(+ `welcomePanel/runningInstanceSection/shortcutSection/systemRealtimeCharts`;`systemOverviewCards.vue` 为**死代码**,删除)。
- **数据**:`useDashboardHomeStore`——`overview`(系统概览)、`status`(实时,`fetchStatus` 轮询)、
  `cpuHistory/memoryHistory/networkHistory/diskHistory`、`runningInstances`、`refreshInterval`。**保留 store 与轮询逻辑**。
- **桌面结构**:
  1. **问候区**(安静,非大卡):`早/午/晚上好,{name} 👋` + 一行元信息(主机名 · 运行时长 · 天气 · 时间,次色)+ 右侧小头像(`AppAvatar` 带在线点)。
  2. **指标带** `MetricStrip`:CPU / 内存 / 磁盘(值 + `percent` 细单色进度线 + caption)、网络(`value` slot 双行 ↓↑,tabular)。**单色,禁彩色图标盒。**
  3. **实时监控** `AppPanel`(标题 + 刷新间隔切换 `1s/3s/5s/10s`):内嵌 `systemRealtimeCharts`(ECharts,CPU / 内存·Swap;接口/磁盘可选)。保留其逻辑,重绘容器观感(去重边框、留白)。
  4. **底部两栏**:左 `运行中实例`(`SectionHeader` + `EntityCard` 流,空态 `EmptyState`);右 `快捷入口`(`shortcutSection`,图标+文案的安静网格,非彩色大块)。
  5. **系统信息**(可选,克制):`DescriptionList` 两列(设备名/CPU/OS/架构/内核/启动时间)。
- **移动**(M-4):问候单列;指标带 4→2→1;图表堆叠、图例下移;实例/快捷单列。
- **状态**:各区独立骨架;实例空态引导去"应用列表";接口异常图表占位。
- **权限**:自动更新检查仅内置超管(`role_1` + `system:update`)——**保留**该判定。
- **注意**:`welcomePanel.vue` 探索版已撤回,按上重做。

---

## 分析页 `dashboard/analysis` · **已删除**

- DFAN 模板假数据页;与 Momoko 业务无关。
- **已删除** 视图 + store + 菜单种子(`6398d24` / `1dbebed`)。若未来要真实业务分析,另开规格,勿复活模板页。

---

## 监控页 `dashboard/monitor` · **已删除**

- 原为 `Math.random` 假数据面板。
- **已删除** 视图 + store + 菜单种子。工作台 `dashboard/home` 已含真实资源实时曲线;独立监控页不再提供。

---

## 个人中心 `profile` · P2(选项卡式) · 优先级:中

- **文件**:`index.vue` + `gradientHeader/userMainPanel/personalInfoPanel/myInformation/myMessages/myPermission/loginDevices/loginLogs/archivesPanel`;`stores/user/profile`。
- **数据**:用户资料、消息、权限、登录设备、登录日志、地址/天气。**保留逻辑**。
- **桌面结构**:
  1. **头部** `gradientHeader`:重绘为**克制**版——低饱和主色渐变或纯色条 + 头像 + 名/角色/简介 + 关键操作(编辑资料、改密码)。**不要**花哨渐变(唯一允许主色面积处,仍需克制)。
  2. **主体**:左侧信息卡(`personalInfoPanel`/`myInformation`:资料 `DescriptionList` + 编辑);右侧或下方 `UTabs`:`我的消息`(`myMessages` 列表)、`我的权限`(`myPermission`:角色 + 按钮权限,分组展示)、`登录设备`(`loginDevices`:`EntityCard` 流,含"下线"操作 + `AdaptiveConfirm`)、`登录日志`(`loginLogs`:`DataTable` 或卡片)。
- **移动**:头部压缩(头像 + 名);Tabs 顶部横滑;各列表转卡;编辑走 `FormDialog` 全屏。
- **组件**:改密码复用 `components/dialog/UpdatePassword.vue`(重写为 `FormDialog` + schema)。
- **状态**:各 Tab 独立加载/空;设备/日志分页。
- **权限**:`myPermission` 只读展示当前用户权限。

---

## 本组验收要点
- 工作台指标带**单色、无彩色 KPI 卡**;问候不做大白卡。
- 个人中心头部克制,不花哨;各列表移动端转卡片。
- 分析/监控的假数据部分登记"待决",不在其上浪费精修。
