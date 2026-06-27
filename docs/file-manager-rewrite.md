# 文件管理「完全重写」— 新会话交接（唯一事实来源）

> ⚠️ 本文件取代旧的 `docs/file-manager-redesign.md`（那份是旧实现的修补日志，已过时，**新会话不要参考它，也不要参考任何已删除的旧前端文件**——用户多次强调：参考旧代码会让你变成「修改」而不是「重写」）。

## 0. 用户的硬性要求（已重复申明 5+ 次，必须遵守）
1. **完全重写，不是修补**。为强制重写，**旧的文件管理前端代码文件已被物理删除**（见 §4）。
2. **禁止参考已删除的旧代码**（连 `git show` 翻旧实现也尽量不要——除非是 proto/生成类型/通用基础设施这类「契约/底座」）。从**参考图 + proto 契约**出发做 clean-room 重写。
3. **必须长得像参考图**：`docs/design/file-list-target.png`（列表页）、`docs/design/file-editor-target.png`（编辑器）。两张图都是**浅色(LIGHT)干净设计**。当前 app 默认是**暗色紫主题**，所以文件模块若只是「继承 app 主题」就永远不像图——**文件模块要有自成体系的浅色设计**（像图一样），不要被 app 的暗/紫主题牵着走。
4. **不许用「把主题色固定成蓝」这种偷懒法**（即不要只 override `--el-color-primary`）。要建立**独立设计令牌**，逐元素显式上色：结构/悬停=中性灰，蓝色(`#1677ff`)只用于主按钮/选中等**有意为之**处，删除=红，文件夹图标=琥珀。
5. **图标必须统一**（之前编辑器用 Element-Plus 实心、列表用 Heroicons 细线，混搭=「不伦不类」）。统一用 **Heroicons v2 `@heroicons/vue/24/outline`（细线）+ 两个自绘 SVG**：保存=软盘、全屏=四角括号。
6. **后端**：用 `make api` 由 proto 生成 TS 类型（已能跑，exit 0）。判定「后端是否完美适配」要看：**①现有接口是否完美契合新需求 ②API 设计是否简洁/高效/高性能 ③是否被正确充分使用**——**不是**用蹩脚 API 去硬凑新需求。若不完美就**破坏性重写后端(proto/biz/service)**，让后端贴合新前端，禁止让新需求迁就旧 API。（保留底层存储引擎 `FileOper`/上传管线/任务队列/分享存储——那是引擎不是包袱。）
7. 左侧系统菜单栏不要动。

## 1. 设计规格（从参考图逐像素核对得出）
**调色板（自成体系，浅色默认；建议也给暗色变体）**
- 主面 `#ffffff`；页底/表头 `#f5f7fa`/`#f6f8fa`；文本 `#1f2328`/次 `#57606a`/弱 `#8c959f`；边框 `#e4e8ee`；悬停底 `#f3f5f8`。
- 强调蓝 `--accent:#1677ff`（暗色 `#3b82f6`）；删除红 `--el-color-danger`；文件夹琥珀 `#e3a008`。
- 小圆角 0.4~0.5rem；软分隔线；紧凑小图标。

**A. 列表页**（参考 `file-list-target.png`，浅色）
- 顶部导航：返回/前进/上级（图标按钮）+ 面包屑地址（可点进出目录；点击可编辑成输入框直接跳路径）。
- 工具栏：`上传文件`(蓝实心) `新建文件夹`(白底细灰边) `下载`(白底) `删除`(白底+红字红图标) `更多操作▾`(下拉) ……右侧：`搜索文件`输入框(放大镜) + `漏斗筛选`图标按钮(popover 含「包含子目录」)。
- 表格列：复选框 / `文件名`(图标+名，文件夹琥珀图标) / `类型`(文件夹 / `{EXT} 图片|视频|音频|文件`) / `大小` / `路径` / `修改时间` / `操作`(图标：编辑(铅笔)/下载/删除(红)/更多(竖⋮下拉))。表头浅灰底、行细分隔、hover 浅灰。
- 底部：`共 X 个目录, Y 个文件`(左) + 分页(右)：`共 N 条` · `N条/页▾` · `‹ 1 2 3 4 ›`(当前页蓝底白字) · `前往 _ 页`。
- **分页必须自绘**：`TablePagination`(`src/components/pagination/TablePagination.vue`) 被 docker/instance/node/system 等 10+ 页公用，**不能动它**；文件页要自己的 pager 组件（显式蓝色当前页）。

**B. 编辑器**（参考 `file-editor-target.png`，VS Code 风，浅色）
- 窗口壳：标题 `编辑文件 - {文件名}` + 右上 `最小化`/`关闭`；标题栏可拖拽；**默认大窗居中**(≈min(1360,vw-24)×min(880,vh-64))，不要开局很小。可全屏(最大化)/最小化(收成标题条)；右下角可拖拽改大小；左右树宽可拖。
- 工具栏：`保存`(蓝实心 + **软盘**图标, loading/禁用当未改) · `刷新` · `下载` · `重命名` · `删除`(红) · `主题`(下拉 自动/亮/暗, localStorage 持久, 整窗+Monaco 一起切) · `全屏`(四角括号图标, 切换时用内向四角)。次按钮=白底细灰边。
- 主体两栏：左 `文件目录` 懒加载文件树(文件夹琥珀 Folder/FolderOpen + 展开箭头旋转 + Document；选中文件蓝高亮；当前目录为根) ‖ 分隔条(可拖宽) ‖ 右 Monaco(行号+语法高亮+受支持语言校验；主题随上面的主题开关)。
- 底部状态栏(整宽)：左 `文件路径: {path}`；右 `{语言大写,如 YAML}`(蓝) · `文件编码: UTF-8` · `行尾格式: LF/CRLF`(可点切换改写换行) · `行 X, 列 Y` · `共 N 行`。
- 交互：点树文件→(脏则确认丢弃)读取载入；保存/刷新/下载/重命名/删除针对当前活动文件；重命名/删除后刷新树；媒体/二进制在编辑区给「该文件无法在编辑器中打开」遮罩。
- Monaco worker 用 `monaco-editor/esm/.../*.worker?worker` 注入 `MonacoEnvironment`。

**图标映射（统一 icons 模块，建议 `src/components/file/icons.ts`）**
保存=自绘软盘SVG · 刷新=ArrowPathIcon · 下载=ArrowDownTrayIcon · 上传=ArrowUpTrayIcon · 重命名/编辑=PencilSquareIcon · 删除=TrashIcon · 全屏/退出=自绘四角括号(外/内) · 新建文件夹=FolderPlusIcon · 更多=EllipsisHorizontal/Vertical · 搜索=MagnifyingGlassIcon · 筛选=FunnelIcon · 返回/前进/上级=ArrowLeft/Right/Up Icon · 面包屑分隔=ChevronRightIcon · 树 文件夹/展开/文件=Folder/FolderOpen/Document Icon · 窗口 最小化/关闭=Minus/XMark Icon · 主题 自动/亮/暗=ComputerDesktop/Sun/Moon Icon · loading 用 ArrowPath 旋转。（这些 heroicons 名都已确认存在；项目无软盘/四角图标→自绘 `h('svg',{stroke:currentColor,...})`，用 `el-icon` 包裹自动 1em 上色。）

## 2. 后端 API 契约（来自 proto，**这是允许参考的契约**）
**生成**：改 proto 后 `make api`（根目录），同时出 Go(`api/gen`)+HTTP+gRPC+OpenAPI(`frontend/src/types/openapi.yaml`)+**前端 TS 类型**(`frontend/src/types/v1/*.ts`，ts_proto，仅类型、无 client)。已验证可跑(exit 0)。改完只留实质改动、还原版本号 noise；后端 `go build ./...`。

**系统级**(`api/proto/v1/file.proto`，service `FileManager`)——HTTP 前缀 `/api/v1`：
- `GET /file/system` GetFileSystemList(path,page,page_size,keywords?,include_sub_dir?,sort_field,is_desc) → {directory:FileDirectoryInfo, items:FileEntryInfo[], page,page_size,total}
- `GET /file/system/tree` GetFileSystemTree(path) → {path, nodes:FileTreeNode[]}（懒加载，目录在前）
- `POST /file/system/deletes` BatchDeleteFileSystem(paths[]) → {items:FileOperationResult[]}
- `POST /file/system/create` CreateFileSystem(info:FileCreateItem{path,is_dir,content bytes})
- `POST /file/system/rename` RenameFileSystem(path,new_name) → {path}
- `POST /file/system/copy` / `/cut` → {task:FileTaskInfo}（异步，轮询 task）
- `GET /file/task/{task_id}` GetFileTask → {task:FileTaskInfo(status:FileTaskStatus,...)}
- `POST /file/system/compress` BatchCompress(paths[],target_path?) → {output_path}
- `POST /file/system/unzip` Unzip(path,target_path?) → {output_path}
- `POST /file/system/open/file` OpenFileSystemFile(path) → {info: bytes}  ← **打开正文走内联 bytes(base64 over JSON)**
- `POST /file/system/edit/file` EditFileSystemFile(path,content bytes)
- `GET /file/system/file/pre-sign` FileSystemPreSign(path) → {download_url_path}（下载/媒体预览用，配 `resolveStaticResourceUrl`）
- `GET /file/system/file/upload/pre-sign` PreSignUpload(path,file_name,file_size,hash,part_size?) → {info:UploadInfo(分片上传)}
- `GET /file/upload/status` · `POST /file/upload/complete` · `POST /file/upload/cancel`
- 分享：`POST /file/share/create` · `GET /file/share/list` · `POST /file/share/update` · `POST /file/share/delete`
- 公开分享：`GET /public/share/meta`(token) · `POST /public/share/list`(token,code,sub_path)

**实例级**(`api/proto/v1/instance.proto`，service 内，与系统级**几乎一一镜像**，多个 `{id}`)——`/api/v1`：
`GET /instance/file/{id}`(list) · `GET /instance/file/tree/{id}` · `POST /instance/file/create/{id}` · `/rename/{id}` · `/copy/{id}` · `/cut/{id}` · `/deletes/{id}` · `/compress/{id}` · `/unzip/{id}` · `/open/{id}` · `/edit/{id}` · `GET /instance/file/pre-sign/{id}` · `GET /instance/file/upload/pre-sign/{id}`。复用 file.proto 的 `FileEntryInfo/FileDirectoryInfo/FileTreeNode/FileTaskInfo/UploadInfo/FileOperationResult` 等。

**关键类型**(`frontend/src/types/v1/file.ts` 等，已生成)：`FileEntryInfo{name,path,is_dir→isDir,permission,userName,userId,groupName,groupId,size,updateTime}`、`FileDirectoryInfo{name,path,parentPath,dirCount,fileCount,itemCount}`、`FileTreeNode{name,path,isDir}`、`FileSortField`(enum, stringEnums)、`FileTaskStatus`(enum)、`ShareInfo`、`UploadInfo` 等。bytes 字段在 TS 里是 `Uint8Array`，但 JSON 传输需 base64。

**⚠️ 后端「是否完美」评估点（用户特别强调，决定要不要破坏性重写后端）**：
- **系统级 + 实例级是两套近乎重复的 RPC/消息** → 不够简洁。可考虑统一成一套(scope=system|instance:{id})，或共享全部 message。**若统一会改路由→牵动 RBAC + 整个前端，但更干净**。
- **打开/编辑正文走内联 bytes(base64 over JSON)** → 大文件低效；下载已有 pre-sign 流式。可评估编辑器读写是否也该走 pre-sign/流式，或 open 是否该带「大小/是否二进制」标记避免前端先全量拉。
- 这些是「不蹩脚」的判断依据；按用户意图：**该重写就重写后端，别让前端凑合**。

## 3. 前端底座（可复用的基础设施，非「旧文件管理代码」）
- 请求：`src/utils/request.ts` 默认导出 `request.{get,post,put,delete}<T>(url,config)`，返回 `AxiosResponse<T>`；响应拦截器把后端 `{code,message,data}` 解包成 `response.data=data`；baseURL=`VITE_API_BASE_URL`。`showRequestError(err,fallback)` 统一报错。
- 类型：`@/types/v1/*`（make 生成）。OpenAPI：`frontend/src/types/openapi.yaml`。
- 主题：`@/stores/theme` `useThemeStore()` 有 `isDarkTheme`、`primaryColor`(项目主色#8B5CF6 紫)。**文件模块别用它的紫**；要自成体系浅色。
- i18n：`vue-i18n`，键加在 `src/locales/messages.ts` 的 `export const messages` 三语言(zh-CN/zh-HK/en-US) 对应 `fileManager.*`/`file.share.*`/`system.common.*`/`common.*`。`translate as t` 可在非组件中用。
- 静态资源：`@/utils/assets` `resolveStaticResourceUrl(path)`（pre-sign 下载路径→可访问 URL）。
- 图标库：`@heroicons/vue/24/outline`（已装）、`@element-plus/icons-vue`（尽量别用，统一 heroicons）。
- 路由：`src/router/route.ts` 静态 import 了 `views/public/share/index.vue` 和 `views/instance/fileManager/index.vue`（**已删→构建会断，必须重建这两个 view 文件**）；系统文件管理页 `views/file/index/index.vue` 走菜单动态路由(`menuToRoute.ts`，缺失有 404 兜底，但要重建)；公开分享路由 `/public/share/:token`。
- 分页共享件 `src/components/pagination/TablePagination.vue`：**勿改**(10+ 页公用)。

**需要重新实现的工具职责**（旧 `utils/filePreview.ts`/`fileTask.ts` 已删；按职责 clean-room 重写，建议 `src/utils/file*.ts`）：
- `encodeBytesToBase64(Uint8Array)→string` / `decodeBase64ToBytes` / `decodeFileContent(bytes|base64)→string`（open 正文解码、edit/create 编码；`api/instance.ts` 也 import 了 `encodeBytesToBase64`，见 §4）。
- `resolveFilePreviewKind(path)→'image'|'audio'|'video'|null`（按扩展名判媒体）。
- `resolvePreSignedFileUrl(download_url_path)`（套 `resolveStaticResourceUrl`）。
- `downloadFileFromUrl(url,name)`（a[download] 触发）/ `copyTextToClipboard(text)`。
- `joinPath(root,segment)`（按 `\`或`/` 分隔符）；`normalizeNumber`、`hasOwnQueryKey`、`normalizeKeywords`、`normalizeIncludeSubDir` 等小工具。
- `waitFileTask(taskId)`：轮询 `GET /file/task/{id}` 直到 SUCCESS/FAILED（copy/cut 异步用）。

## 4. 当前仓库状态（已做的破坏性删除 + 待修断点）
**已 `git rm` 删除（前端）**：
`src/components/fileManager/`(整目录: index.vue/FileManagerPage.vue/FileEditorDialogContent.vue/FileMediaDialogContent.vue/FileUploadDialog.vue/useFileUpload.ts/icons.ts) · `src/views/file/`(index/, share/) · `src/views/instance/fileManager/` · `src/views/public/share/` · `src/stores/file/index.ts` · `src/stores/instance/fileManager.ts` · `src/api/file.ts` · `src/api/share.ts` · `src/utils/filePreview.ts` · `src/utils/fileTask.ts`。

**因删除而断的引用（必须在重写中修复）**：
- `src/api/instance.ts`：`import { encodeBytesToBase64 } from '@/utils/filePreview'`（已删）+ 含全部实例文件 API 函数 + 从 `@/types/v1/instance` import 文件类型。**此文件还含实例管理(start/stop 等)非文件代码，不要整文件删**；重写文件部分、把 encode 换到新工具。
- `src/components/share/ShareFormDialog.vue`：`import {...} from '@/api/share'`（已删）。share 表单弹窗(创建/编辑分享)——重建 api/share 后修复，或一并重写。
- `src/router/route.ts`：静态 import 了已删的 `views/public/share/index.vue`、`views/instance/fileManager/index.vue` → **构建断**，重建这两个 view。
- `src/stores/instance/types.ts`：`import type { FileManagerAction } from '@/components/fileManager/index.vue'`（已删，2 处：import + re-export）→ 把 `FileManagerAction` 类型搬到新位置(如 `src/components/file/types.ts`)。
- 后端 proto 已 `make api` 跑过；`api/gen/**`、`frontend/src/types/v1/*` 可能有版本号 noise，提交前按需 `git checkout` 还原非实质改动。

**未删但相关**：`src/components/share/ShareFormDialog.vue`（保留，但依赖已删 api/share）。`frontend/src/types/v1/{file,instance}.ts` 生成类型保留。

## 5. 建议重写顺序（每步 `cd frontend && npx vue-tsc --build && npx eslint src/...` 须 EXIT 0；后端动了则 `go build ./...`）
1. **底座**：新 `src/components/file/types.ts`(workbench 视图模型 + `FileManagerAction` 等) + 新工具(§3 职责) + 新 `src/components/file/icons.ts`(heroicons + 软盘/四角自绘)。修 `stores/instance/types.ts`、`api/instance.ts` 的断点。
2. **数据层**：新 `api/file.ts`+`api/share.ts`(基于生成类型 + §2 路由 + request 工具，base64 编解码包装 bytes 字段) + 新 store(系统 + 实例；若后端统一则一个 scope 化 store)。
3. **UI(核心，先出可见效果)**：新文件列表组件(自成体系浅色、导航+工具栏+表格+自绘分页) + 新编辑器组件(树+Monaco+工具栏+状态栏+窗口控件+主题开关) + 子弹窗(新建/重命名/删除/压缩/解压/上传/媒体预览)。host views: `views/file/index`、`views/instance/fileManager`。
4. **分享**：`views/file/share`(管理页，重建成新视觉) + `views/public/share`(公开页) + ShareFormDialog。
5. **后端(若评估不完美)**：按 §2 评估点破坏性重写 proto/biz/service(保留 FileOper 引擎)→`make api`→前端切新类型。
6. **i18n** 三语言补齐；浏览器实测。

## 6. 验证（浏览器 MCP，chrome-devtools，dev 跑在 :3007）
登录 `admin/admin`（默认弱口令，见 README）。系统文件页：直接 `http://localhost:3007/file/index?workdir=D:\github\momoko\configs` 可定位到有文件的目录；打开 `config.yaml` 看编辑器。翻页用有 32 项的 `D:\github\momoko`(workdir 切到 momoko)看分页。**逐屏截图与 `docs/design/*.png` 比对，直到一致**。app 当前是暗色——文件模块应呈现自带浅色(像参考图)。

## 7. 备注/坑
- `make`/protoc/kratos/ts_proto/wire 工具链已验证可用。
- ts_proto 配置 `outputServices=none,outputClientImpl=false` → 只生成类型，**HTTP 请求函数要手写**(基于路由+request)。
- Monaco 体积大；worker 用 `?worker` 注入。
- 媒体/二进制：前端按扩展名(媒体)或内容(NUL/非法 UTF-8)判定；媒体走 pre-sign objectUrl 预览，编辑器对其给「无法打开」。
- 用户在用暗色主题且对照浅色参考图——这是反复不满的根因：**文件模块必须自带浅色设计，不可继承 app 暗/紫**。
