# 06b · 逐页规格 · 实例 / Docker / 节点 / OpenSSH

> 格式同 `06a`。页型见 `05`,移动见 `02`,组件见 `03`/`03b`。

---

## 实例(应用管理)

### 应用列表 `instance/list` · P1(卡片) · 优先级:高
- **文件**:`index.vue` + `overviewStats/instanceCard/instanceEditor`;`stores/instance/list`。
- **数据**:`getInstances`(分页、按状态/类型/关键词筛选)、启停/重启/删除、批量。**保留 store 逻辑**。
- **桌面**:
  1. `PageHeader`(标题 + `创建实例`)。
  2. **概览指标** `overviewStats` → `MetricStrip`:总实例 / 运行中 / 已停止(单色数字 + caption "当前筛选/当前页")。
  3. `FilterBar`:搜索(名称/标签)+ 类型 + 状态 + 搜索/重置;`Toolbar`:批量启动/停止/删除 + 全选 + 刷新。
  4. **卡片流** `EntityCard`(替 `instanceCard`):图标/类型徽标 + 名称 + 状态胶囊(运行/停止)+ 备注 + 创建时间 + 路径;底部操作:`控制台` `配置` `更多(ActionMenu)` + 主 `启动/停止`。
  5. `Pagination`。
- **移动**(M-1):指标带 3→1;卡片单列;筛选折叠;批量走多选模式 + 底部条;主操作在卡内。
- **弹窗**:`instanceEditor` → `FormDialog`(创建/编辑,schema 校验)。
- **状态**:空态引导创建;操作 loading 在按钮上;删除 `AdaptiveConfirm`。
- **权限**:创建/启停/删除按 `PERM.*`(实现时核对 `config/permission.ts`)。

### 实例控制台 `instance/console/:instanceId` · P6(终端) · 优先级:中
- **文件**:`views/instance/console/index.vue`;`stores/instance/console`、`consoleSession`。
- **保留** xterm 终端、WebSocket 会话、多会话逻辑。仅重绘工具条(实例名 + 状态 + 操作:重连/清屏/字号)与外壳(`fullBleed`)。
- **移动**(M-2):终端占满宽,软键盘工具条(Ctrl/Tab/方向/Esc),字号可调。

### 实例文件 `instance/files/:instanceId` · P6(文件) · 优先级:中
- **文件**:`views/instance/fileManager/index.vue`;复用 `components/file/*`(FileBrowser/FileTree/FileEditor/…)。
- **保留** 文件浏览/上传/编辑(Monaco)逻辑。重绘工具条 + 面包屑 + 列表/树观感。
- **移动**(M-2):双栏→单栏面包屑下钻;操作走长按/更多。

### 实例类型 `instance/type` · P1 · 优先级:中
- **文件**:`index.vue` + `create.vue`;`stores/instance/type`。
- **桌面**:`PageHeader`(+ `新增类型`)+(可无筛选)+ 卡片流或 `DataTable`(类型少用卡)。字段:类型名、镜像/模板、说明。
- **弹窗**:`create.vue` → `FormDialog`。移动转卡。删除 `AdaptiveConfirm`。

---

## Docker

> 统一 P1。四页共用外壳:`PageHeader` + `FilterBar` + `Toolbar` + 主体 + `Pagination`。**保留 API 与任务对话框逻辑**
> (`DockerContainerExecDialog/LogsDialog/StatsDialog/DockerTaskDialogs`,重写为 `FormDialog`/`UModal` + 观感)。
> Docker 未连接时(如截图的 `EOF`)顶部给**明确错误条 + 重试/去配置**,不是空白。

### 容器 `docker/container` · P1(卡片优先) · 优先级:高
- **数据**:容器列表(名称/镜像/状态/运行/网络)、启停/重启/删除、日志(`LogsDialog`)、终端(`ExecDialog`)、监控(`StatsDialog`)。
- **桌面**:`FilterBar`(名称/状态/镜像)+ `Toolbar`(创建容器、任务)+ **卡片流** `EntityCard`:容器名 + 状态胶囊 + 镜像 + 端口/网络 + 底部 `日志` `终端` `监控` `更多`。
- **移动**(M-1):卡片单列;操作收 `ActionMenu`;`StatsDialog` 全屏。
- **状态**:未连接=错误条;空=引导创建。

### 镜像 `docker/image` · P1 · 优先级:中
- 卡片或 `DataTable`(镜像多用表):仓库/标签、大小、创建时间;操作:拉取/删除/运行。移动转卡。

### 网络 `docker/network` · P1 · 优先级:中
- `DataTable`(网络型强列):名称/驱动/范围/网关;操作:创建/删除。移动转卡。

### 配置 `docker/config` · P3(表单) · 优先级:中
- Docker 连接配置(主机/TLS 等):整页 `AppPanel` + `UForm`;保存后可"测试连接"。移动全宽表单。

---

## 节点管理

### API Key 管理 `node/key` · P1 · 优先级:中
- **文件**:`index.vue` + `create.vue`;`api/node`。
- **桌面**:`PageHeader`(+ `新建 Key`)+ `FilterBar`(名称/状态)+ 卡片流 `EntityCard`:Key 名 + 状态胶囊 + 掩码 key(带复制)+ 权限范围 + 到期/创建时间;操作:复制、禁用/启用、删除。
- **安全**:新建后**仅一次**明文展示(`FormDialog` 结果态 + 复制 + 提醒);列表只显掩码。
- **移动**(M-1)转卡;复制按钮触控友好。

---

## OpenSSH

### SSH 管理 `openssh/management` · P1 · 优先级:中
- **文件**:`index.vue` + `create.vue`;`api/openssh`。
- **桌面**:`PageHeader`(+ `新增主机`)+ `FilterBar`(名称/主机)+ 表/卡:`user@host:port` + 认证方式 + 权限 + 备注;**不展示**持久连接状态。行内**连接**(外侧主按钮,跳 `openssh/terminal`)+ ActionMenu(编辑/分享/删除,仅所有者)。**测试连接**仅在新建/编辑 `FormDialog` 页脚,测当前表单草稿(可带 id 回落库中凭据);不写库 status。
- **弹窗**:`create.vue` → `FormDialog`(主机/端口/用户/认证/密钥,schema)。
- **移动**(M-1)转卡;连接为主操作。

### SSH 终端 `openssh/terminal` · P6(终端) · 优先级:中
- **保留** xterm + 会话逻辑。重绘工具条(主机切换/新开会话/字号)+ `fullBleed`。移动(M-2)软键盘工具条。

---

## 本组验收要点
- 列表页一律 `PageHeader + FilterBar +(卡/表)+ Pagination`,不再包大卡。
- 终端/文件页只换壳,保留 xterm/Monaco/WebSocket 逻辑。
- 危险操作(删除/下线)统一 `AdaptiveConfirm`;密钥类一次性明文 + 复制。
- 每页桌面 + 移动截图留档,`08` 勾选。
