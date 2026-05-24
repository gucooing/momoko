# 系统初始化前端适配文档

后端在未初始化时不会连接业务数据库，只启动一个只接受初始化接口的 HTTP 服务。前端在访问根路径时应先检查初始化状态，未初始化则跳转初始化页面。

## 接口

所有响应仍使用统一包装：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

### 获取初始化状态

`GET /api/v1/system/initialize/status`

响应 `data`：

```ts
import type { DatabaseType } from '@/types/v1/initialize'

interface InitializeStatusResponse {
  initialized: boolean
  supportedDatabaseTypes: DatabaseType[]
}
```

`supportedDatabaseTypes` 当前可能值：

```ts
DatabaseType.DatabaseType_SQLite
DatabaseType.DatabaseType_MySQL
DatabaseType.DatabaseType_PostgreSQL
```

### 测试数据库连接

`POST /api/v1/system/initialize/database/test`

请求中的数据库配置不再填写原始连接串。SQLite 只填文件路径；MySQL/PostgreSQL 填连接地址、账号、密码和数据库名。

字段说明：

| 字段 | SQLite | MySQL/PostgreSQL |
| --- | --- | --- |
| `type` | 必填 | 必填 |
| `sqlitePath` | 必填，可为空使用默认 `./data/momoko.db` | 置空 |
| `address` | 置空 | 必填，支持 `host` 或 `host:port` |
| `username` | 置空 | 必填 |
| `password` | 置空 | 按数据库配置填写 |
| `databaseName` | 置空 | 必填 |

当 `address` 不包含端口时，MySQL 默认补 `3306`，PostgreSQL 默认补 `5432`。

SQLite 示例：

```ts
import { DatabaseType } from '@/types/v1/initialize'
import type { TestInitializeDatabaseRequest } from '@/types/v1/initialize'

const payload: TestInitializeDatabaseRequest = {
  database: {
    type: DatabaseType.DatabaseType_SQLite,
    sqlitePath: './data/momoko.db',
    address: '',
    username: '',
    password: '',
    databaseName: '',
  },
}
```

MySQL 示例：

```ts
const payload: TestInitializeDatabaseRequest = {
  database: {
    type: DatabaseType.DatabaseType_MySQL,
    address: '127.0.0.1:3306',
    username: 'momoko',
    password: 'password',
    databaseName: 'momoko',
    sqlitePath: '',
  },
}
```

PostgreSQL 示例：

```ts
import { DatabaseType } from '@/types/v1/initialize'
import type { TestInitializeDatabaseRequest } from '@/types/v1/initialize'

const payload: TestInitializeDatabaseRequest = {
  database: {
    type: DatabaseType.DatabaseType_PostgreSQL,
    address: '127.0.0.1:5432',
    username: 'momoko',
    password: 'password',
    databaseName: 'momoko',
    sqlitePath: '',
  },
}
```

响应 `data`：

```ts
interface TestInitializeDatabaseResponse {
  success: boolean
}
```

该接口只测试连接，不会建表、不会写入管理员、也不会写入 `data/initialized.json`。

### 确认初始化

`POST /api/v1/system/initialize/confirm`

请求：

```ts
import { DatabaseType } from '@/types/v1/initialize'
import type { ConfirmInitializeRequest } from '@/types/v1/initialize'

const payload: ConfirmInitializeRequest = {
  database: {
    type: DatabaseType.DatabaseType_SQLite,
    sqlitePath: './data/momoko.db',
    address: '',
    username: '',
    password: '',
    databaseName: '',
  },
  admin: {
    username: 'admin',
    password: 'your-password',
    email: 'admin@example.com',
    name: '超级管理员',
  },
}
```

响应 `data`：

```ts
interface ConfirmInitializeResponse {
  initialized: boolean
  restartRequired: boolean
}
```

该接口成功后，后端会自动从初始化 HTTP 服务切换到完整业务服务。切换期间可能出现短暂连接中断，前端可在收到成功响应后轮询初始化状态接口可用性。

## 前端流程建议

1. 根目录请求 `GET /system/initialize/status`。
2. `initialized === false` 时跳转 `/initialize`，并禁止进入登录页和业务页面。
3. `/initialize` 页面分三步：
   - 选择数据库类型并填写数据库路径或连接信息。
   - 创建超级管理员。
   - 确认提交，提示用户服务需要重启。
4. `POST /system/initialize/confirm` 成功后，后端会完成建表、写入内置菜单/角色、创建超级管理员、更新 `configs/config.yaml` 中数据库配置，并写入 `data/initialized.json` 标记。
5. 收到 `restartRequired: true` 后展示完成状态，提示服务正在自动重启；轮询状态接口返回 `initialized: true` 或登录接口恢复后，再进入登录页。

初始化接口免登录。确认初始化后不要立即自动登录，等待后端自动重启并启动完整业务服务后再进入登录流程。
