# 06e · 逐页规格 · 认证 / 初始化 / Public / OIDC / 异常

> 格式同 `06a`。这些页多为 **P7(独立无外壳)** 或 **P8(异常)**。

---

## 认证

### 登录 `login` · P7 · 优先级:高(Phase 3 早期,门面)
- **文件**:`views/login/index.vue` + `accountLogin/register/forgotPassword`;`api/login`。
- **数据**:账号登录、注册、找回、刷新令牌、OIDC 登录入口。**保留逻辑**。
- **实现口径(已落地)**:居中 hairline 卡 + 冷灰页底 + 角标语言/主题；主色落在按钮与焦点。**不做**半屏大面积品牌渐变。
- **移动**(M-5):表单全宽居中;字段大号触控友好。
- **状态**:提交 loading 在按钮;错误 toast(`feedback`)+ 字段级校验。

### 注册 / 找回:随 `login` 一并重做(同容器子视图),不单列页。

---

## 初始化向导 `initialize` · P3(向导) · 优先级:高
- **文件**:`views/initialize/index.vue`;`api/initialize`。
- **说明**:首次部署引导(建管理员、基础配置)。独立页、无外壳。
- **实现口径**:居中卡 + 紧凑步骤条 + `AppSelect` 库类型 + 令牌表单;确认摘要 + 结果态;业务契约 status/test/confirm 保留。

---

## Public(免登录,独立页)

> 面向终端用户,**移动优先**、少 chrome、独立品牌轻量外壳(非管理外壳)。

### 分享落地 `public/share/:token` · P7 · 优先级:中
- **文件**:`views/public/share/index.vue`;`api/share`。
- **实现**:去 EP;文件模块 `fm-*` 主题;预览 FileViewerDialog/Monaco;**`7ae41db`**。

### OIDC 授权 `oidc/authorize` · P7(authOnly) · 优先级:中
- **文件**:`views/public/oidc/authorize/index.vue`;`api/oidc`。
- **实现**:居中令牌卡 + scope 友好列表 + UButton;全量去 EP;**`7ae41db`**。

### Sub2API Public:`public/sub2api/home` `.../stats` `.../imagine` `.../lottery` · P7
- **说明**:门户/统计/绘图/抽奖。**可被 iframe 嵌入**(`ui_mode=embedded`)——**保留** bootstrap query 与 theme 分支。
- **实现**:home 拆接口渐进(`ea7c1d4`);stats 明确 loading;imagine/lottery 去 EP(`9c963f8`)。
- **imagine**:AppSelect + mode-seg + FormDialog×3 + lightbox;禁 `el-*`/`v-loading`。
- **lottery**:顶栏 `AppIconButton`;StatusPill + EmptyState + FormDialog 规则。

---

## 异常 / 工具页

### 403 / 404 · P8
- 克制居中:大号码 + 标题 + 说明 + 回首页/上一页;无大插画。

### redirect · 工具页
- 令牌页底 + logo 脉冲占位;保留重定向逻辑。

---

## 清理状态(已完成,不再待决)

> DFAN 模板演示页与假仪表盘 **已删除**(见 `08-progress` / `6398d24` / 菜单 `1dbebed`)。

- ~~`views/demo/vxeTable/*`~~ 已删
- ~~`views/extended/*`~~ 已删
- ~~`views/dashboard/analysis/*`~~ 已删
- EP/VXE 已从 package 卸载

---

## 本组验收要点
- 登录/初始化是门面;主色克制。
- Public/OIDC 移动优先;**保留** iframe 嵌入主题联动。
- toast 须压过弹窗(`App.vue` toaster `z-[10000]`)。
