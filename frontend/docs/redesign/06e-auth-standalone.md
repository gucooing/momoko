# 06e · 逐页规格 · 认证 / 初始化 / Public / OIDC / 异常 / 待清理

> 格式同 `06a`。这些页多为 **P7(独立无外壳)** 或 **P8(异常)**。

---

## 认证

### 登录 `login` · P7 · 优先级:高(Phase 3 早期,门面)
- **文件**:`views/login/index.vue` + `accountLogin/register/forgotPassword`;`api/login`。
- **数据**:账号登录、注册、找回、刷新令牌、OIDC 登录入口。**保留逻辑**。
- **桌面**:两栏——左**品牌区**(低饱和主色渐变,唯一允许大面积主色处;logo + 名 + 一句 slogan + 克制装饰)+ 右**表单卡**(`UForm`:账号/密码 + 记住我 + 登录 + 找回/注册切换 + 第三方/OIDC)。
- **切换**:登录/注册/找回同一容器内切换(`accountLogin/register/forgotPassword` 子视图),过渡轻。
- **移动**(M-5):品牌区收为顶部小标识;表单全宽居中;字段大号触控友好;`< md` 隐藏装饰。
- **状态**:提交 loading 在按钮;错误 toast + 字段级校验;验证码/锁定(若有)保留。

### 注册 / 找回:随 `login` 一并重做(同容器子视图),不单列页。

---

## 初始化向导 `initialize` · P3(向导) · 优先级:高
- **文件**:`views/initialize/index.vue`;`api/initialize`。
- **说明**:首次部署引导(建管理员、基础配置)。独立页、无外壳。
- **桌面**:居中容器 + 顶部**步骤条**;一屏一步(欢迎 → 管理员账号 → 基础/站点配置 → 完成);吸底"下一步/完成"。
- **移动**(M-3):步骤条紧凑点状;一屏一步;吸底操作;表单标签顶对齐。
- **状态**:每步校验;完成后跳登录/工作台。**保留**初始化判定与接口。

---

## Public(免登录,独立页)

> 面向终端用户,**移动优先**、少 chrome、独立品牌轻量外壳(非管理外壳)。

### 分享落地 `public/share/:token` · P7 · 优先级:中
- **文件**:`views/public/share/index.vue`;`api/share`。
- **桌面/移动**:居中卡:文件/文件夹信息(名/大小/类型/来源可混合)+ 预览(图片/媒体/文本 Monaco 只读)+ 下载;若有密码→先输密码;过期→友好提示页。移动优先单列。
- **保留** 分享校验/多源/密码逻辑。

### OIDC 授权 `oidc/authorize` · P7(authOnly) · 优先级:中
- **文件**:`views/public/oidc/authorize/index.vue`;`api/oidc`。
- **结构**:居中授权卡:应用信息 + 请求的 scope 列表(清晰可读)+ 账号 + `授权/拒绝`。移动全宽。
- **保留** 授权流与回调逻辑。

### Sub2API Public:`public/sub2api/home` `.../stats` `.../imagine` · P7 · 优先级:中
- **文件**:对应 `views/public/sub2api/*`;`stores/sub2api/*`。
- **说明**:面向终端用户的 Sub2API 门户/统计/绘图。**可被 iframe 嵌入**(`ui_mode=embedded`,父级下发主题)——**保留** `stores/theme` 里的 embedded 主题读取与 postMessage 逻辑。
- **结构**:轻量独立外壳(顶部小标题 + 内容);home=入口/说明,stats=用量图表 + 指标带,imagine=绘图交互。移动优先单列。
- **注意**:嵌入态主题跟随父级,不写 localStorage;保留该分支。

---

## 异常 / 工具页

### 403 `exception/403` · P8 · 优先级:低
- 居中:大字/插画 + "无权限访问" + 说明 + `返回首页`/`联系管理员`。移动全宽。

### 404 `exception/404` · P8 · 优先级:低
- 居中:404 + 说明 + `返回首页`。也是 `menuToRoute` 缺失视图的兜底。

### redirect `redirect/index` · 工具页 · 优先级:低
- 无 UI(仅做刷新/重定向);保留逻辑,给一个居中 loading 占位。

---

## 待清理 / 待决(Phase 4,默认删除,删前确认)

> 这些是 **DFAN 模板遗留的演示页**,非 Momoko 业务。删除前:确认后端菜单不再下发、无路由引用。记入 `08` 待决区。

- `views/demo/vxeTable/*`(`index.vue`/`index copy.vue`/`create.vue`)——VXE 演示,**随 VXE 下线删除**。
- `views/extended/*`(`button/dialog/hoverAnimation/iconSelector/textEllipsis/transitionAnimation`)——组件演示页,删除。
- `views/dashboard/analysis/*`——假数据分析页(见 `06a`),删除或改造为真实分析。
- 相关菜单项、路由、i18n、以及仅被它们使用的组件(如 `HoverAnimateWrapper` 若无其他引用)一并清理。

---

## 本组验收要点
- 登录/初始化是门面,精修优先级高;品牌区是**唯一**允许大面积主色处,仍需克制。
- Public/OIDC 面向外部用户,移动优先;**保留** iframe 嵌入主题联动逻辑。
- 演示页统一登记待决,不在其上花精力;随 EP/VXE 下线一起清理。
