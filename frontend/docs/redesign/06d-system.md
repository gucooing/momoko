# 06d · 逐页规格 · 系统管理

> 格式同 `06a`。系统管理是 CRUD 最集中的模块,建议作为 **Phase 3 起点**(用户管理已在 Phase 2 做样板)。

---

## 用户管理 `system/user` · P1 · 优先级:高(Phase 2 样板)
- **文件**:`index.vue` + `create.vue`;`api/user`。
- **数据**:`userPage`(分页、按用户名/状态)、创建/编辑、删除、批量删除。**保留逻辑**。
- **桌面**:
  1. `PageHeader`("用户管理" + 一句描述 + `新增用户`)。
  2. `FilterBar`:用户名(`UInput`)+ 状态(`USelect`:启用/禁用)+ 搜索/重置。
  3. `Toolbar`:批量删除(选中后启用,`PERM.USER_DELETE`)。
  4. **卡片流** `EntityCard`(实体型,用卡):头像/首字(`AppAvatar`)+ 用户名 + 状态胶囊 + 姓名·邮箱 + 角色徽标 + 创建时间;底部 `编辑` `删除`;可多选。
     - 备选:提供"表视图"切换到 `DataTable`(列:序号/用户名/姓名/邮箱/角色/状态/创建/操作)。默认卡。
  5. `Pagination`。
- **弹窗**:`create.vue`→`FormDialog`(用户名/姓名/邮箱/角色/密码/状态,schema;编辑时禁改用户名)。
- **移动**(M-1):卡片单列;筛选折叠;`新增`在标题区/FAB;批量走多选 + 底部条;分页简化。
- **状态**:空态引导新增;删除 `AdaptiveConfirm`。
- **权限**:`PERM.USER_ADD/EDIT/DELETE`(`v-permission` / `useButtonPermission`)——**保留**。

## 角色管理 `system/role` · P1 · 优先级:高
- **文件**:`index.vue` + `create.vue`;`api/role`。
- **数据**:角色列表、创建/编辑(含**菜单/按钮权限勾选树**)、删除。**保留权限树逻辑**。
- **桌面**:`PageHeader`(+ `新增角色`)+ `FilterBar`(角色名/状态)+ 卡片流 `EntityCard`:角色名 + 状态 + 描述 + 用户数 + 操作(编辑、分配权限、删除)。
- **弹窗**:`create.vue`→`FormDialog`:基本信息 + **权限树**(`UTree` 或自建树,复选,联动父子)。移动:权限树全屏、可搜索。
- **状态/权限**:内置超管角色(`role_1`)不可删/受限;空态引导。

## 菜单管理 `system/menu` · P1(树表) · 优先级:中
- **文件**:`index.vue` + `create.vue`;`api/menu`。
- **数据**:菜单树(目录/菜单/按钮三型)、图标、路径、权限标识、排序、状态。**保留逻辑**。
- **桌面**:`PageHeader`(+ `新增菜单`)+ **`DataTable` 树表**(强层级):名称(缩进 + `UIcon`)/类型徽标/路径/权限标识/排序/状态/操作(增子项、编辑、删除)。
- **图标选择**:`create.vue` 的图标选择器改为 **Iconify 选择**(存 iconify 名;旧值经 `iconMap` 兜底,见 `03b` §3)。
- **弹窗**:`create.vue`→`FormDialog`(父级/类型/名称/图标/路径/权限标识/排序/状态)。
- **移动**(M-1):树表→可展开卡片层级;操作收 `ActionMenu`。
- **注意**:菜单 path 决定路由映射,**改 path 影响页面加载**,谨慎;规格里提醒。

## OIDC 客户端 `system/oidc` · P1 · 优先级:中
- **文件**:`views/system/oidc/index.vue`;`api/oidc`。
- **数据**:OIDC 客户端(client_id/secret/回调/授权范围/状态)。**保留逻辑**。
- **桌面**:`PageHeader`(+ `新增客户端`)+ 卡片流 `EntityCard`:客户端名 + 状态 + `client_id`(复制)+ 回调域 + 范围;操作:编辑、重置密钥、删除。
- **安全**:secret 一次性明文 + 复制;列表打码。
- **弹窗**:`FormDialog`。移动转卡。授权页见 `06e` `oidc/authorize`。

## 系统设置 `system/settings` · P3(表单) · 优先级:中
- **文件**:`views/system/settings/index.vue`;`api/system`。
- **桌面**:整页 `UTabs` 分组(基础/安全/邮件/存储/更新…),每组 `AppPanel` + `UForm`;`SectionHeader` 分节;敏感字段显隐;"保存"吸顶或分区保存。
- **更新检查**:接 `system:update`(内置超管),保留逻辑。
- **移动**(M-3):Tabs 横滑;全宽表单;吸底保存。

## 定时任务 `system/task` · P1 · 优先级:中
- **文件**:`views/system/task/index.vue`;`api/task`。
- **数据**:任务列表(名称/cron/上次/下次/状态/结果)、启停、立即执行、日志。**保留逻辑**。
- **桌面**:`PageHeader`(+ `新增任务`)+ `FilterBar`(名称/状态)+ **`DataTable`**(强列:名称/cron/上次运行/下次运行/状态/操作)或卡片(任务少)。操作:执行、启停、编辑、删除、查看日志。
- **移动**(M-1)转卡;cron 可读化;日志全屏。

## 操作日志 `system/operation` · P1(表格) · 优先级:中
- **文件**:`views/system/operation/index.vue`;`api`(操作日志)。
- **数据**:日志(用户/操作类型/IP/耗时/结果/时间/详情 JSON/UA)。只读 + 筛选 + 分页。**保留逻辑**。
- **桌面**:`PageHeader` + `FilterBar`(用户/操作类型/结果/路径/时间范围)+ **`DataTable`**(强列):
  用户 · 操作类型(`UBadge`)· IP · 耗时 · 结果(状态胶囊)· 时间 · 操作(查看详情)。
  **详情 JSON / UA 不塞进单元格**——点行/查看 → `UModal` 展示格式化 JSON(可复制)+ UA 解析。
- **移动**(M-1):转卡片(用户 + 类型 + 结果 + 时间;点开看详情);筛选折叠;分页简化。
- **状态**:空态;时间范围默认近 7 天。

---

## 本组验收要点
- 用户/角色用卡片(实体型);菜单/任务/操作日志用 `DataTable`(强列/树)。
- 操作日志**杜绝**把 JSON/UA 塞单元格,详情走弹窗。
- 权限树、图标选择、敏感字段(secret/key)一次性明文——逻辑保留,观感重做。
- 内置超管/角色(`role_1`)的受限规则保留。
