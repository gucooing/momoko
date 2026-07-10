# 06c · 逐页规格 · 工具(隧道/端口转发) / 文件 / Sub2API

> 格式同 `06a`。页型见 `05`。

---

## 工具

### 内网穿透 `tools/tunnel` · P1 · 优先级:中
- **文件**:`index.vue` + `create.vue` + `FrpcConfigDialog/FrpsConfigDialog/TunnelStatsDialog`;`api/tunnel`。
- **数据**:隧道列表(名称/类型/状态/公共端口/本地目标/启用/连接数/流量)、启停、frpc/frps 配置、统计。**保留逻辑**。
- **桌面**:`PageHeader`(+ `新增隧道`)+ `FilterBar`(关键词/类型/启用状态)+ `Toolbar`(批量删除、`frps 配置`)+ **卡片流** `EntityCard`:
  隧道名 + 状态胶囊 + 类型徽标 + `:公共端口 → 本地目标` + 启用开关(`USwitch`)+ 连接数/流量(↓↑ tabular)+ 操作 `详情/统计` `frpc 配置` `编辑` `删除`。
  (流量/连接数是"强列型"倾向,但实体属性可读——用卡;若用户偏好可加表视图切换。)
- **弹窗**:`create.vue`→`FormDialog`;`Frpc/FrpsConfigDialog`→`FormDialog`(配置文本/字段,含 Monaco 或结构化表单);`TunnelStatsDialog`→`UModal`(图表 + 指标)。
- **移动**(M-1):卡片单列;开关/操作触控友好;统计弹窗全屏。
- **状态**:空态引导;frps 未配置提示去配置。

### 端口转发 `tools/port-forward` · P1 · 优先级:中
- **文件**:`index.vue` + `create.vue` + `PortForwardStatsDialog`;`api` 对应。
- **桌面**:`PageHeader`(+ `新增转发`)+ `FilterBar` + 卡片流 `EntityCard`:规则名 + 状态胶囊 + `源 → 目标`(协议/端口)+ 流量/连接;操作:统计、编辑、启停、删除。
- **弹窗**:`create.vue`→`FormDialog`;`PortForwardStatsDialog`→`UModal`(统计)。
- **移动**(M-1)转卡。

---

## 文件

### 文件管理 `file/index` · P6(文件浏览) · 优先级:中
- **文件**:`views/file/index/index.vue`;复用 `components/file/*`(FileBrowser/FileTree/FileTreeNode/FileEditor/FileMenu/FilePager/FilePicker/FileUploadDialog/FileViewerDialog/FileMediaDialog…)。
- **保留** 浏览/上传/下载/编辑(Monaco)/预览/多源逻辑与 `fileClient`。仅重绘:工具条(路径面包屑 + 新建/上传/视图切换)、树与列表观感(hairline、行 hover、图标统一 `UIcon`)、右键/更多菜单(`ActionMenu`)。
- **桌面**:双栏(左树 + 右列表/网格);列表可切换 详情/网格。
- **移动**(M-2):单栏 + 顶部面包屑下钻;操作长按/更多;上传用系统选择器;预览/编辑全屏。
- **状态**:目录空态;加载骨架;上传进度(`UProgress`)。

### 文件分享 `file/share` · P1/P2 · 优先级:中
- **文件**:`views/file/share/index.vue`;`components/share/ShareFormDialog.vue`;`api/share`。
- **说明**:近期做过移动端优化与"混合多来源"分享(见 git)。**保留逻辑**。
- **桌面**:`PageHeader`(+ `新建分享`)+ `FilterBar` + 卡片流 `EntityCard`:分享名/来源 + 状态(有效/过期/受密码保护)+ 链接(复制)+ 到期/下载次数 + 操作(复制链接、二维码、编辑、删除)。
- **弹窗**:`ShareFormDialog`→`FormDialog`(选文件[可多源]、有效期、密码、下载限制)。
- **移动**(M-1)转卡;链接复制/二维码触控友好。对外分享落地页见 `06e` `public/share`。

### 文件来源 `file/source` · P1 · 优先级:中
- **文件**:`views/file/source/index.vue`;`api/fileSource`。
- **说明**:存储后端(本地/S3/…)。遵循记忆约束:**逻辑路径 + sourceID**,不在前端区分本地/远程细节。
- **桌面**:`PageHeader`(+ `新增来源`)+ 卡片流 `EntityCard`:来源名 + 类型徽标(本地/对象存储…)+ 状态(可用/异常)+ 关键配置摘要(打码)+ 操作(测试、编辑、删除、设默认)。
- **弹窗**:`FormDialog`(按来源类型动态字段,schema)。移动转卡。

---

## Sub2API(管理端)

> Sub2API 有管理端(需登录,走外壳)与 public 端(免登录,独立页,见 `06e`)。管理端优先级中。

### Sub2API 首页 `sub2api/home` · P4/P2 · 优先级:中
- **文件**:`views/sub2api/home/index.vue`;`stores/sub2api/index`、`imagine`。
- **数据**:账号/渠道/用量/统计。**保留 store**。
- **桌面**:`PageHeader` + `MetricStrip`(账号数/请求量/额度…单色)+ `AppPanel` 图表(用量趋势)+ 账号/渠道 `EntityCard` 流或 `DataTable`。
- **移动**(M-4/M-1):指标带堆叠;列表转卡。

### Sub2API 配置 `sub2api/config` · P3(表单) · 优先级:中
- **文件**:`views/sub2api/config/index.vue`。
- **桌面**:整页 `AppPanel` 分区 `UForm`(渠道/密钥/模型映射/限流…);分组用 `SectionHeader`;敏感字段打码 + 显隐。
- **移动**(M-3):全宽表单,吸底保存。

---

## 本组验收要点
- 文件/终端类只换壳保逻辑;`components/file/*` 内部逻辑不动,只改观感。
- 分享/来源/隧道的"链接复制/二维码/开关"在移动端触控友好。
- 存储细节遵循"逻辑路径 + sourceID",不在 biz/前端特判本地 vs 远程。
