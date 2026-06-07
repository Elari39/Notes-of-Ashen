# Notes of Ashen

Notes of Ashen 是一个前后端分离的个人博客项目。后端使用 Go 与 go-zero 风格组织代码，提供用户认证、文章管理、分类标签、后台管理、站点设置和操作日志等能力；前端使用 React、TypeScript、Vite 与 Tailwind CSS 构建博客展示和后台管理界面。

## 功能概览

- 用户认证：注册、登录、退出、刷新 Token。
- 双 Token 机制：Access Token 使用 JWT，Refresh Token 哈希后持久化，并结合 Redis 校验。
- 文章管理：创建、编辑、删除、发布、归档、草稿预览、版本查看、公开列表与公开详情。
- Markdown 渲染：支持 GFM、代码高亮、表格、LaTeX 数学公式和正文图片点击放大。
- 分类与标签：公开读取，后台可创建、更新和删除。
- 后台能力：用户列表、用户状态管理、站点设置和操作日志查看。
- 异步日志：通过 RabbitMQ 投递注册、登录、文章操作等事件，并写入 `operation_logs`。
- 统一响应：接口成功时返回 `{ "code": 0, "message": "success", "data": ... }`。

## 技术栈

- 后端：Go 1.25、go-zero REST、MySQL、Redis、RabbitMQ、JWT、bcrypt。
- 前端：React 18、TypeScript、Vite、Tailwind CSS、Zustand、Axios。
- 文档与脚本：API 文档位于 [docs/API.md](docs/API.md)，数据库脚本位于 [deploy/mysql](deploy/mysql)。

## 项目结构

```text
.
├── api/                    # go-zero API 描述文件
├── cmd/notes-of-ashen/     # 后端服务入口
├── deploy/mysql/           # MySQL 初始化与增量脚本
├── docs/                   # API 文档
├── etc/                    # 后端配置文件
├── frontend/               # React 前端应用
├── internal/               # 后端内部模块
│   ├── authutil/           # JWT 与 Token 工具
│   ├── config/             # 配置结构
│   ├── handler/            # HTTP handler
│   ├── logic/              # 业务逻辑
│   ├── middleware/         # 鉴权中间件
│   ├── mq/                 # RabbitMQ 发布与消费
│   ├── response/           # 统一响应
│   ├── svc/                # 服务依赖上下文
│   └── validator/          # 参数校验
└── model/                  # 数据访问层
```

## 环境依赖

本地开发默认依赖以下服务：

- MySQL 8.0：默认连接 `root / 123456`，端口 `3306`。
- Redis：默认密码 `123456`，端口 `6379`。
- RabbitMQ：默认连接 `guest / guest`，端口 `5672`，管理端口 `15672`。

这些默认账号和密码只适合本地开发，生产环境必须通过安全配置替换。

```powershell
docker ps
```

## 后端配置

配置文件位于 [etc/notes-of-ashen.yaml](etc/notes-of-ashen.yaml)：

```yaml
Name: notes-of-ashen-api
Host: 0.0.0.0
Port: 19000
Timeout: 5000

Database:
  DataSource: "root:123456@tcp(127.0.0.1:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local"
  MaxOpenConns: 20
  MaxIdleConns: 10

Auth:
  AccessSecret: "please-change-this-secret-in-production"
  AccessExpire: 7200
  RefreshExpire: 604800

Redis:
  Addr: "127.0.0.1:6379"
  Password: "123456"
  DB: 0

RabbitMQ:
  Enabled: true
  URL: "amqp://guest:guest@127.0.0.1:5672/"
  Exchange: "notes-of-ashen.events"
  Queue: "notes-of-ashen.operation_logs"
  RoutingKey: "operation.log"
```

关键字段：

- `Port`：后端 HTTP 服务端口，默认 `19000`。
- `Database.DataSource`：MySQL DSN。
- `Auth.AccessSecret`：JWT 签名密钥，生产环境必须修改。
- `Auth.AccessExpire`：Access Token 有效期，单位秒。
- `Auth.RefreshExpire`：Refresh Token 有效期，单位秒。
- `RabbitMQ.Enabled`：是否启用事件发布与消费。

## 数据库初始化

如果 `notes_of_ashen` 数据库尚未创建，可在项目根目录执行：

```powershell
Get-Content deploy\mysql\schema.sql | docker exec -i mysql mysql -uroot -p123456
```

验证数据库：

```powershell
docker exec mysql mysql -uroot -p123456 -N -e "SHOW DATABASES LIKE 'notes_of_ashen';"
```

验证数据表：

```powershell
docker exec mysql mysql -uroot -p123456 -D notes_of_ashen -N -e "SHOW TABLES;"
```

## 启动后端

在项目根目录执行：

```powershell
go mod tidy
go run ./cmd/notes-of-ashen -f etc/notes-of-ashen.yaml
```

服务默认监听：

```text
http://127.0.0.1:19000
```

也可以构建后运行：

```powershell
go build -o notes-of-ashen.exe ./cmd/notes-of-ashen
.\notes-of-ashen.exe -f etc\notes-of-ashen.yaml
```

## 启动前端

前端位于 `frontend/`，必须使用 `pnpm` 管理依赖。开发服务器默认监听 `3000`，并将 `/api` 代理到 `http://127.0.0.1:19000`。

```powershell
cd frontend
pnpm install
pnpm dev
```

访问地址：

```text
http://127.0.0.1:3000
```

常用前端命令：

```powershell
pnpm build
pnpm lint
pnpm preview
```

## 常用验证命令

后端测试：

```powershell
go test ./...
```

后端构建：

```powershell
go build ./...
```

Redis 连通性：

```powershell
docker exec redis redis-cli -a 123456 ping
```

公开文章列表接口：

```powershell
Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:19000/api/v1/articles?page=1&size=10"
```

## 认证流程

1. 调用 `POST /api/v1/auth/register` 注册用户。
2. 第一个注册用户自动成为 `admin`，后续注册用户默认为 `user`。
3. 注册或登录成功后返回 `accessToken` 与 `refreshToken`。
4. 访问受保护接口时添加请求头：

```text
Authorization: Bearer <accessToken>
```

5. Access Token 过期后，调用 `POST /api/v1/auth/refresh` 获取新的 Token。
6. 调用 `POST /api/v1/auth/logout` 可撤销当前 Refresh Token。

## 权限说明

- 公开接口：注册、登录、刷新 Token、公开文章列表、公开文章详情、分类列表、标签列表。
- 登录用户：查看和修改个人资料，修改密码，创建文章，管理自己的文章。
- 管理员：管理分类、标签、用户状态、站点设置，并查看操作日志。
- 普通用户不能修改或删除其他用户的文章。

## 文章与用户状态

文章状态：

- `draft`：草稿。
- `published`：已发布，可在公开文章列表和详情页访问。
- `archived`：归档。

用户状态：

- `active`：正常。
- `disabled`：禁用，无法登录或刷新 Token。

## RabbitMQ 行为

服务启动后会创建并绑定：

- Exchange：`notes-of-ashen.events`
- Queue：`notes-of-ashen.operation_logs`
- RoutingKey：`operation.log`

以下事件会投递到 RabbitMQ：

- `user.registered`
- `user.logged_in`
- `user.logged_out`
- `article.created`
- `article.updated`
- `article.deleted`
- `article.status_updated`

消费者收到消息后写入 `operation_logs`。RabbitMQ 发布失败不会阻断主业务，但会记录错误日志。

## API 文档

完整接口说明见 [docs/API.md](docs/API.md)。

常用接口：

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/articles`
- `GET /api/v1/articles/:id`
- `GET /api/v1/categories`
- `GET /api/v1/tags`
- `GET /api/v1/users/me`
- `GET /api/v1/admin/users`
- `GET /api/v1/admin/logs`

## 常见问题

### Redis 返回 NOAUTH

说明 Redis 开启了密码认证，但配置文件密码不正确。先确认 Redis：

```powershell
docker exec redis redis-cli -a 123456 ping
```

再检查 [etc/notes-of-ashen.yaml](etc/notes-of-ashen.yaml)：

```yaml
Redis:
  Password: "123456"
```

### MySQL 连接失败

确认容器运行：

```powershell
docker ps
```

确认配置中的 DSN：

```yaml
Database:
  DataSource: "root:123456@tcp(127.0.0.1:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local"
```

### 管理员接口返回 403

只有 `role = admin` 的用户可以访问管理员接口。默认情况下，第一个注册用户会自动成为管理员。

### 文章详情返回 404

公开文章详情只允许访问 `published` 状态的文章。`draft` 和 `archived` 状态不会对匿名用户开放。
