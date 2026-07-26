# Notes of Ashen

`Notes of Ashen` 是一个前后端分离的个人博客系统。后端使用 Go 与 go-zero 风格组织代码，前端使用 React、TypeScript、Vite 与 Tailwind CSS 构建页面，部署侧提供 Docker Compose、Nginx 与 1Panel 友好的运行方案。

本机 Docker 默认访问地址：

```text
http://127.0.0.1:1270
```

## 快速开始

适合用 Docker / 1Panel 一键拉起 Web、API、本地 MySQL、Redis 和 RabbitMQ：

> ⚠️ 复制 `.env` 后，必须把其中所有 `<REPLACE_…>` 占位符替换为真实强随机值，否则后端启动会拒绝占位配置。

```powershell
Copy-Item .env.example .env
# 编辑 .env，替换所有 <REPLACE_…> 占位符，尤其是 JWT 和本地中间件密码
docker compose config --quiet
docker compose up -d --build
```

Linux / macOS：

```bash
cp .env.example .env
# 编辑 .env，替换所有 <REPLACE_…> 占位符，尤其是 JWT 和本地中间件密码
docker compose config --quiet
docker compose up -d --build
```

启动完成后访问：

```text
http://127.0.0.1:1270
```

首次进入站点后注册第一个用户。默认逻辑下，第一个注册用户会成为管理员；如果邮箱服务保持默认关闭，首个管理员注册会自动跳过邮箱验证码，后续注册仍需要先启用并配置邮箱验证码能力。

阅读路径建议：

- 想快速部署：阅读“配置环境变量”和“本机 Docker 部署”。
- 想本地开发：阅读“本地非 Docker 开发”和 [frontend/README.md](frontend/README.md)。
- 想对接接口：阅读“常用 API”和 [docs/API.md](docs/API.md)。

## 功能概览

- 用户认证：注册、登录、退出、刷新 Token。
- 文章管理：创建、编辑、删除、发布、归档、草稿预览、版本查看、版本恢复、Markdown 导入/导出、AI 一键补全文章与 SEO 信息、AI 辅助创作、置顶与显示优先级。
- 内容展示：公开文章列表、文章详情、字数与预计阅读时间、原生分享/复制链接、按年月日展开的发布归档、搜索建议与本地最近搜索、Meilisearch 全文搜索、Markdown 渲染、代码高亮、LaTeX 数学公式、文章目录和点赞反馈。
- 作品集：作品集画廊和项目标签。
- 分类与标签：公开与后台文章数量展示，后台可创建、更新和删除。
- 媒体库：本地持久化 JPEG/PNG/GIF/WebP，内容哈希去重，文章与作品集可选择封面或插入 Markdown 图片。
- 管理后台：用户管理、用户状态管理、站点设置、项目管理、操作日志、页面/文章内容分析、依赖健康探测、口令加密备份与整站恢复。
- 站点能力：RSS、Sitemap、站点标题、描述、关键词、Prerender.io 预渲染配置等。
- 流量统计：公开页面自动上报 PV、UV 与来源，后台展示最近 30 天趋势。
- 异步日志：通过 RabbitMQ 投递操作事件，并写入 `operation_logs`。
- 统一响应：接口成功时返回 `{ "code": 0, "message": "success", "data": ... }`。

## 技术栈

- 后端：Go 1.25、go-zero REST、MySQL 8.4、Redis 7.4、Meilisearch 1.13、RabbitMQ 4、JWT、bcrypt。
  - Docker Compose 默认启动本地 MySQL / Redis / RabbitMQ / Meilisearch 容器；快速开始的 `.env.example` 会启用 RabbitMQ 异步日志，Compose 代码自身的 `APP_RABBITMQ_ENABLED` 回退值为 `false`。搜索功能默认关闭，关闭时 API 回退到 MySQL 查询。
- 前端：React 18、TypeScript、Vite 5、Tailwind CSS 4、Zustand、Axios、Framer Motion、ECharts、React Markdown。
- 部署：Docker、Docker Compose、Nginx、1Panel。
- 文档与脚本：API 文档位于 [docs/API.md](docs/API.md)，数据库脚本位于 [deploy/mysql](deploy/mysql)。

## 项目结构

```text
.
├── api/                    # go-zero API 描述文件
├── cmd/notes-of-ashen/     # 后端服务入口
├── deploy/
│   ├── mysql/              # 不可变编号数据库迁移
│   └── nginx/              # 前端 Nginx 生产配置
├── docs/                   # API 文档
├── etc/                    # 后端默认配置文件
├── frontend/               # React 前端应用
│   └── README.md           # 前端开发说明
├── internal/               # 后端内部模块
├── model/                  # 数据访问层
├── Dockerfile.api          # Go API 镜像构建文件
├── Dockerfile.web          # 前端 Nginx 镜像构建文件
├── docker-compose.yml      # Docker Compose 编排文件
└── .env.example            # 环境变量模板
```

## 从 GitHub 克隆到本地

### 前置要求

请先安装：

- Git
- Docker Desktop，或 Linux 服务器上的 Docker Engine
- Docker Compose

如果要进行非 Docker 本地开发，还需要：

- Go 1.25 或兼容版本
- Node.js 22 或兼容版本
- pnpm

### Windows PowerShell

```powershell
git clone https://github.com/Elari39/Notes-of-Ashen.git
cd Notes-of-Ashen
```

### Linux / macOS

```bash
git clone https://github.com/Elari39/Notes-of-Ashen.git
cd Notes-of-Ashen
```

## 配置环境变量

项目使用仓库根目录的 `.env` 为 Docker Compose 提供部署配置。`.env` 包含本地中间件密码、连接信息和 JWT 密钥，不应提交到 Git。

从模板复制：

```bash
cp .env.example .env
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
```

至少需要检查并替换这些值。模板中的 `<REPLACE_...>` 只用于提示，不是可用默认值；后端启动时会拒绝空值、示例占位值和过短的 JWT 密钥，请把真实密码/密钥只写入 `.env` 或部署平台环境变量。

- `APP_DISPLAY_NAME`：站点对外展示名称，默认 `Notes of Ashen`（当前版本后端未读取，预留）。
- `APP_AUTH_ACCESS_SECRET`：JWT 签名密钥，生产环境必须替换为足够长的随机字符串，建议至少 32 位；后台保存的 AI API Key 也会使用由它派生的独立用途密钥加密，轮换前请先阅读下方迁移说明。
- `APP_AUTH_COOKIE_SECURE`：refreshToken Cookie 的 Secure 标志，生产 HTTPS 保持 `true`；本机 HTTP 开发需设为 `false`，否则浏览器不会保存 Cookie，刷新页面无法恢复会话。
- `APP_MYSQL_ROOT_PASSWORD`：本地 Compose MySQL root 密码，生产或公网环境必须替换为强随机值。
- `APP_MYSQL_USER`：本地 Compose MySQL 应用用户，默认 `notes_user`。
- `APP_MYSQL_PASSWORD`：本地 Compose MySQL 应用用户密码，需和 `APP_DATABASE_DSN` 中的密码保持一致。
- `APP_DATABASE_DSN`：API 与一次性迁移任务使用的 MySQL 连接串，Compose 默认连接本地服务：`notes_user:password@tcp(mysql:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local`。Compose 初始化库固定为 `notes_of_ashen`，此连接串在 Compose 部署中也必须指向该库，并且账号需具有创建/变更表和索引的权限。如需固定 MySQL 会话时区，可追加 `&time_zone='%2B08:00'`（URL 转义后为 `%27%2B08%3A00%27`）。
- `APP_DATABASE_MAX_OPEN_CONNS`：MySQL 最大打开连接数。
- `APP_DATABASE_MAX_IDLE_CONNS`：MySQL 最大空闲连接数。
- `APP_REDIS_ADDR`：Redis 地址。使用 Compose 内置 Redis 时应为 `redis:6379`；接入外部 Redis 时填写外部服务可达的 `host:port`。
- `APP_REDIS_PASSWORD`：Redis 密码。内置 Redis 留空时不启用认证，非空时 Redis 容器、健康检查和 API 会使用同一密码；接入外部 Redis 时填写该实例要求的密码。
- `APP_REDIS_DB`：Redis DB 编号，默认 `0`。
- `APP_RABBITMQ_USER`：本地 Compose RabbitMQ 用户，默认 `notes_user`。
- `APP_RABBITMQ_PASSWORD`：本地 Compose RabbitMQ 密码，需和 `APP_RABBITMQ_URL` 中的密码保持一致。
- `APP_RABBITMQ_ENABLED`：是否启用 RabbitMQ 异步日志。快速开始模板 `.env.example` 显式设为 `true` 以使用本地 RabbitMQ；若不使用模板且未传该变量，`docker-compose.yml` 的代码回退值为 `false`。
- `APP_RABBITMQ_URL`：RabbitMQ AMQP 地址，Compose 默认连接本地服务：`amqp://notes_user:password@rabbitmq:5672/`。
- `APP_RABBITMQ_EXCHANGE`：RabbitMQ 交换器名，默认 `notes-of-ashen.events`，通常无需修改。
- `APP_RABBITMQ_QUEUE`：RabbitMQ 队列名，默认 `notes-of-ashen.operation_logs`，通常无需修改。
- `APP_RABBITMQ_ROUTING_KEY`：RabbitMQ 路由键，默认 `operation.log`，通常无需修改。
- `APP_SEARCH_ENABLED`：是否启用 Meilisearch 全文搜索，默认 `false`，关闭时自动回退 MySQL 查询。
- `APP_MEILISEARCH_HOST`：API 访问 Meilisearch 的地址，Docker 部署默认 `http://meilisearch:7700`。
- `APP_MEILISEARCH_API_KEY`：Meilisearch API Key；Docker 部署时也作为 Meilisearch Master Key。搜索关闭时保持为空；启用搜索时请在 `.env` 中填写强随机字符串。
- `APP_MEILISEARCH_INDEX`：文章索引名，默认 `articles`。
- `APP_EMAIL_ENABLED`：是否启用邮箱验证码，使用 QQ 邮箱时设置为 `true`。
- `APP_EMAIL_SMTP_HOST`：SMTP 服务器地址，默认 `smtp.qq.com`。
- `APP_EMAIL_SMTP_PORT`：SMTP 端口，默认 `465`（隐式 TLS）；使用 587 时需配合 `APP_EMAIL_TLS_MODE=starttls`。
- `APP_EMAIL_TLS_MODE`：TLS 模式，`implicit`（465 隐式 TLS，默认）、`starttls`（587 STARTTLS）、`none`（明文，仅内网测试）。
- `APP_EMAIL_SMTP_USERNAME`：QQ 邮箱账号，例如 `yourname@qq.com`。
- `APP_EMAIL_SMTP_PASSWORD`：QQ 邮箱 SMTP 授权码，不是 QQ 登录密码。
- `APP_EMAIL_FROM`：发件邮箱，通常和 `APP_EMAIL_SMTP_USERNAME` 一致；留空时后端会回退使用 SMTP 用户名。
- `APP_EMAIL_FROM_NAME`：发件人名称。
- `APP_MEDIA_ROOT`：媒体文件根目录；Compose 固定使用 `/data/media` 并挂载持久化卷，本地直跑未设置时默认 `./data/media`。
- `APP_MEDIA_MAX_UPLOAD_BYTES`：单个媒体文件上限，默认 `10485760`（10 MiB）。
- `APP_BACKUP_MAX_UPLOAD_BYTES`：加密备份上传与展开数据上限，默认 `1073741824`（1 GiB）。
- `WEB_BACKUP_MAX_BODY_SIZE`：Web Nginx 恢复接口请求体上限，默认 `1025m`；调整备份上限时必须同步调整。
- `APP_TRUSTED_PROXY_CIDRS`：可信反向代理 CIDR。Compose 默认仅信任固定 Web 容器地址；本地非 Docker 启动默认留空。
- `PRERENDER_ENABLED`：是否启用 Prerender.io crawler 预渲染，`0` 关闭，`1` 启用。
- `PRERENDER_SERVICE_URL`：Prerender.io 服务地址，默认 `https://service.prerender.io`。
- `PRERENDER_TOKEN`：Prerender.io Token，只能写入真实 `.env` 或受控环境变量。
- `APP_GITHUB_TOKEN`：可选 GitHub Token；留空时使用公开匿名额度，不要提交真实 Token（当前版本后端未读取，预留）。
- `WEB_PORT`：本机 Web 访问端口，默认 `1270`。
- `WEB_TRUSTED_PROXY_CIDR`：Web Nginx 直接上游的可信 CIDR，默认 `172.30.127.1/32`（Compose `app` 网桥默认网关）。
- `APP_DOCKER_SUBNET` / `APP_WEB_IPV4_ADDRESS`：Compose 专用子网和 Web 容器固定地址，默认分别为 `172.30.127.0/24`、`172.30.127.10`。

不要把真实 `.env` 内容写入 README、Issue、提交记录或截图中。

### 中间件配置

当前 `docker-compose.yml` 默认启动本地 MySQL、Redis 和 RabbitMQ 容器，API 容器通过 `.env` 中的 `mysql`、`redis`、`rabbitmq` 服务名访问它们。RabbitMQ 容器是否启动与 API 是否启用异步日志是两个概念：快速开始模板启用异步日志，而 Compose 在缺少 `APP_RABBITMQ_ENABLED` 时按 `false` 运行。数据库表结构由一次性 `migrate` 服务执行 [deploy/mysql/migrations](deploy/mysql/migrations) 中的编号迁移，成功后 API 才会启动。

使用内置 Redis 时，设置 `APP_REDIS_ADDR=redis:6379`。`APP_REDIS_PASSWORD` 留空会以无认证模式启动；设置非空密码后，Redis 会启用认证，健康检查通过 `REDISCLI_AUTH` 认证探测，API 也使用同一密码连接。Redis 不可访问或认证失败时，API 会按 fail-fast 策略启动失败。

接入外部 Redis 时，应用连接由 `APP_REDIS_ADDR` 和 `APP_REDIS_PASSWORD` 决定，应填写外部实例的地址与密码；不要继续假定外部服务名为 `redis`、端口为 `6379`，也不要以内置 Redis 容器的健康检查代表外部实例可用性。使用仓库提供的 [docker-compose.external-redis.yml](docker-compose.external-redis.yml) 覆盖文件可移除 `api.depends_on.redis`，并使内置 Redis 不会启动；该文件使用 `!override`，需要 Docker Compose v2.24.4+（本项目验证环境为 v5.1.4）。修改 `.env` 后，以相同的双文件命令校验并启动：

```bash
docker compose -f docker-compose.yml -f docker-compose.external-redis.yml config --quiet
docker compose -f docker-compose.yml -f docker-compose.external-redis.yml up -d --build
```

如果你要改用远程 MySQL，请先完成：

- 远程 MySQL 创建数据库 `notes_of_ashen`，创建专用用户并只授权给 1Panel 服务器 IP 或内网网段。
- 使用与 API 相同的镜像和 `APP_DATABASE_DSN` 执行 `-migrate-only`；Compose 部署会自动完成此步骤。数据库账号必须具有创建/变更表和索引的权限。
- 不再手动按文件执行 SQL。迁移器会以 MySQL advisory lock 串行执行 `000001` 至最新版本，记录文件名、SHA-256、耗时和失败信息；已发布的编号文件不可修改，修复只能新增更高版本的前向迁移。
- 旧库首次纳入迁移管理前必须备份。历史链包含清理无效头像、清理孤儿文章版本和删除 `traffic_geo_*` 表等前向破坏性操作；回滚依赖备份恢复。
- 可使用 `docker compose logs migrate` 查看结果；若 API 的 `/healthz` 显示 schema 检查失败，其中会给出缺失版本或校验和漂移信息。
- 改用远程 Redis、RabbitMQ 时，确认防火墙和安全组允许 1Panel 服务器访问，避免对公网裸露。
- 如果 MySQL DSN 或 RabbitMQ URL 的密码包含 `@`、`:`、`/`、`?`、`#` 等 URL 分隔符，请先做 URL 转义，或使用不含这些分隔符的强随机密码。

### QQ 邮箱验证码配置

在 QQ 邮箱开启 SMTP 服务后，将授权码写入真实 `.env`：

```env
APP_EMAIL_ENABLED=true
APP_EMAIL_SMTP_HOST=smtp.qq.com
APP_EMAIL_SMTP_PORT=465
# TLS 模式：implicit(465 隐式 TLS) | starttls(587 STARTTLS) | none(明文,仅内网测试)
APP_EMAIL_TLS_MODE=implicit
APP_EMAIL_SMTP_USERNAME=yourname@qq.com
APP_EMAIL_SMTP_PASSWORD=your-qq-mail-auth-code
APP_EMAIL_FROM=yourname@qq.com
APP_EMAIL_FROM_NAME="Notes of Ashen"
```

修改 `.env` 后重新创建 API 容器：

```bash
docker compose up -d api
```

### AI 辅助创作配置

后台文章编辑页提供文章信息一键补全、保存时自动摘要、纠错、润色、扩写、缩写和翻译能力。一键补全会生成文章标题、slug、摘要、SEO 标题、SEO 描述、SEO 关键词及分类/标签建议，只填充当前为空的文本字段；分类和标签仅展示文字建议，不会自动创建或选择。AI 配置不再从环境变量或 YAML 读取，统一由管理员在后台 AI 设置页填写并保存到数据库 `site_settings`。可选择 `openai` 或 `anthropic` API 格式，填写服务基础地址和 API Key 后，先获取模型列表、选择模型并测试连接，最后保存并启用。当前 AI 调用均为非流式请求，配置项只保留首字等待和请求总超时，默认分别为 60 秒和 600 秒；文章补全、模型列表和模型测试均由这两项服务端配置控制，不再使用前端固定 AI 超时。AI Base URL 在每次新建连接时重新解析并固定到已校验的公网 IP，拒绝私网、保留地址和公私混合结果，且不使用环境 HTTP 代理、不跟随重定向。本机 DNS 若由 Clash 等透明代理返回 `198.18.0.0/15` Fake-IP，HTTPS 默认端口 `443` 的域名会保留原主机名交给透明代理连接；直接填写该网段 IP、本机或内网模型服务仍不会被接受。

`openai` 格式可填写兼容 OpenAI API 的基础地址，例如 `https://api.example.com/v1`，也可填写完整的 `/chat/completions` 地址；`anthropic` 格式使用 Messages API，可填写主机地址或带服务前缀的基础地址，例如 `https://api.example.com`、`https://api.example.com/anthropic`。当 Anthropic 基础地址未包含 `/v1` 且不是完整的 `/messages` 或 `/models` 端点时，后端会按标准 SDK 语义补全 `/v1/messages` 或 `/v1/models`；显式完整端点和查询参数保持不变。出于 SSRF 防护，Base URL 的实际连接只允许公网地址，或 HTTPS 默认端口 `443` 下由透明代理生成的 `198.18.0.0/15` Fake-IP，不支持本机或内网模型服务。获取模型和测试模型接口都接受尚未保存的草稿连接配置，便于保存前验证；模型测试使用固定 JSON 探针并预留 512 个输出 token，以兼容先生成思考块的模型。不要把真实 API Key 写入 README、Issue、提交记录或截图中。

新保存的 API Key 使用 `v3:` 密文，密钥由 `APP_AUTH_ACCESS_SECRET` 通过独立用途派生。`v2:` 密文使用的是已移除的独立加密密钥，升级后会在设置响应中标记 `apiKeyNeedsUpdate = true`，管理员必须重新填写 API Key；无版本前缀的旧密文仍兼容读取，建议尽快重新保存以迁移到 `v3:`。由于 `v3:` 密钥依赖 `APP_AUTH_ACCESS_SECRET`，轮换认证密钥时必须同时安排重新录入 AI API Key，否则原密文将不可解密。

API 访问日志只记录请求方法、路径、状态码、耗时、可信客户端 IP 和 Request ID，不记录查询串、请求头、Cookie 或请求正文；上游 AI 诊断错误也会对 API Key 脱敏。若凭证曾进入历史日志，应立即轮换对应密钥、撤销相关登录会话，并按部署环境的日志保留策略清理历史副本。

### 可信反向代理配置

后端默认不信任客户端传入的 `X-Forwarded-*` / `X-Real-IP` 请求头，限流、操作日志、流量统计和 RSS/Sitemap 基础 URL 都会优先使用直连信息。Compose 通过专用子网给 Web 容器分配固定地址，API 默认仅信任该地址（`172.30.127.10/32`）。若 Web 前方还有可信 Nginx、1Panel 等外层代理，或 API 会直接接受其他可信代理转发，还需把这些代理的出口 CIDR 追加到真实 `.env`，后端才能从右向左跳过完整可信代理链；多个网段用逗号分隔：

```env
APP_TRUSTED_PROXY_CIDRS=172.30.127.10/32,10.0.0.0/24
```

Web Nginx 使用 `$binary_remote_addr` 做每 IP 限流，并仅接受 `WEB_TRUSTED_PROXY_CIDR` 指定的直接上游提供的 `X-Forwarded-For`。默认值 `172.30.127.1/32` 是 Compose `app` 网桥的默认网关，适用于宿主机上的 1Panel 反向代理转发到 `127.0.0.1:1270` 的部署方式。该值必须保持为实际直接上游的精确 CIDR，禁止填写 `0.0.0.0/0` 等宽泛公网网段，否则客户端可伪造来源 IP 绕过按 IP 限流。

不要在 API 可被公网或不可信客户端直连时配置过宽的网段。若默认 `172.30.127.0/24` 与宿主机或现有 Docker 网络冲突，请同时修改 `APP_DOCKER_SUBNET`、`APP_WEB_IPV4_ADDRESS`、`WEB_TRUSTED_PROXY_CIDR`（新子网的网桥网关 `/32`），并将 `APP_TRUSTED_PROXY_CIDRS` 中的 Web `/32` 更新为相同地址。

### 全文搜索配置

Meilisearch 通过 `search` profile 作为可选服务，默认不会启动；搜索功能默认关闭，`APP_SEARCH_ENABLED=false` 时 API 使用 MySQL 回退，不会调用 Meilisearch。需要启用全文搜索时，在真实 `.env` 中同时设置：

```env
APP_SEARCH_ENABLED=true
COMPOSE_PROFILES=search
APP_MEILISEARCH_HOST=http://meilisearch:7700
APP_MEILISEARCH_API_KEY=replace-with-a-long-random-key
APP_MEILISEARCH_INDEX=articles
```

如果同时启用了 RabbitMQ（`APP_RABBITMQ_ENABLED=true`），请将 `COMPOSE_PROFILES` 改为 `messaging,search`，并确保 RabbitMQ 的密码和 URL 也已配置一致。

启用 `APP_SEARCH_ENABLED=true` 却未在 `COMPOSE_PROFILES` 中加入 `search` 时，`config-check` 会直接失败。配置完成后使用带 profile 的 Compose 命令启动：

```bash
docker compose --profile search up -d --build
```

启用搜索时必须为 `APP_MEILISEARCH_API_KEY` 配置强随机值（同时作为 Meilisearch Master Key）。Compose 在启用搜索时默认使用 `MEILI_ENV=production`；如需临时开发模式，才显式设置 `MEILI_ENV=development`。服务启动后，使用 `editor` 或 `admin` 登录后台，并调用 `POST /api/v1/admin/search/reindex` 全量重建文章索引。Meilisearch 初始化或运行中不可用时，API 不会因此退出，公开文章搜索会回退到 MySQL；后端会在后台重试索引初始化，恢复后重新启用 Meilisearch。

### 媒体、内容分析与系统工具

Docker Compose 使用 `goblog_media_data` 同时挂载到 API 的 `/data/media`（可写）和 Web 的 `/usr/share/nginx/media`（只读）。Nginx 以 `/media/` 提供不可变长期缓存；不要在 Web 容器内直接修改媒体文件。非 Docker 开发使用 `Media.RootDir` 或 `APP_MEDIA_ROOT`，并由 Vite 将 `/media` 代理到 API。

媒体仅接受 JPEG、PNG、GIF 和 WebP，按内容 SHA-256 保存和去重。上传先写入隐藏暂存文件，元数据成功后再原子发布；删除先移入隐藏隔离区，数据库失败时恢复，进程中断残留会在后续媒体操作中按数据库状态恢复或清理。后台 `editor/admin` 可浏览与上传，只有 `admin` 可删除；被文章、历史版本、作品或头像引用的媒体不会被删除。

Web 入口的 `GET /healthz` 代理到 API readiness，会反映数据库、Redis 和 schema 状态；`GET /livez` 仅表示 Nginx 静态入口存活。监控与负载均衡应使用 `/healthz` 判断应用是否可接流量。

“内容分析”从 `traffic_content_daily_stats` 新表部署后开始累计页面和文章级 PV/UV，无法从旧的文章总浏览量可靠回填。用于 UV 去重的访客哈希明细会定期清理，每日聚合长期保留；UV 使用服务端 IP、User-Agent 与校验后的匿名 Visitor ID 组合哈希，同一 NAT 下的不同浏览器可区分，清理/缺失 Visitor ID 时回退到 IP/UA；不保存原始 IP、Visitor ID、设备、浏览器和地域信息。

“系统工具”仅管理员可访问，提供依赖健康探测以及 `.noa-backup` 加密导出/恢复。健康页的 `backup_schema` 项会校验媒体与内容分析表是否齐全；旧 MySQL 数据卷若显示异常，先完成数据库备份并运行一次迁移任务。备份口令不会持久化；归档不包含 AI API Key、Token、日志、流量明细、搜索索引和访客点赞哈希。恢复是破坏性的整站替换，会清空目标会话、日志、流量统计和 AI Key，并强制退出当前登录。执行恢复前仍应保留数据库/媒体卷的基础设施快照，并先在独立实例演练。

### 预渲染配置

SPA SEO 预渲染默认关闭。需要为搜索引擎 crawler 返回预渲染 HTML 时，在真实 `.env` 中设置：

```env
PRERENDER_ENABLED=1
PRERENDER_SERVICE_URL=https://service.prerender.io
PRERENDER_TOKEN=your-prerender-token
```

不要提交 Prerender Token 或任何真实密钥。

## 本机 Docker 部署

先校验 Compose 配置：

```bash
docker compose config --services
docker compose config --quiet
```

构建并启动：

```bash
docker compose up -d --build
```

### 发布与回滚（不可变镜像 tag）

`docker compose up -d --build` 适合本机开发，但镜像会一直使用 `latest` 标签，发布不可复现（审计 P2）。正式发布请使用发布脚本：

```powershell
# 发布：以当前 git 提交构建镜像，tag 形如 v20260726-1030-3521de6，
# 自动更新 .env 的 IMAGE_TAG 并 docker compose up -d
pwsh scripts/release.ps1

# 只构建与更新 .env，不启动
pwsh scripts/release.ps1 -SkipDeploy

# 回滚到之前发布过的 tag（要求本地镜像仍存在）
pwsh scripts/release.ps1 -Rollback v20260726-1030-3521de6
```

脚本会把每次发布/回滚的时间、tag、git commit 和镜像 digest 追加记录到 `deploy/release-history.local.jsonl`（不入库），用于追溯与回滚。发布验收要求：`docker compose config` 展开结果中的应用镜像不得包含 `latest`。

默认仅启动 Web、API、MySQL 和 Redis。RabbitMQ 与 Meilisearch 是可选服务：启用前在 `.env` 中配置对应能力与凭据，并设置 `COMPOSE_PROFILES`：

```env
# 异步操作日志
COMPOSE_PROFILES=messaging
APP_RABBITMQ_ENABLED=true
APP_RABBITMQ_PASSWORD=<strong-random-password>
APP_RABBITMQ_URL=amqp://notes_user:<strong-random-password>@rabbitmq:5672/

# 搜索；与 messaging 同时启用时写为 messaging,search
# COMPOSE_PROFILES=search
# APP_SEARCH_ENABLED=true
# APP_MEILISEARCH_API_KEY=<strong-random-key>
MEILI_ENV=production
```

也可临时使用 `docker compose --profile messaging up -d --build` 或 `docker compose --profile search up -d --build`；能力开关和凭据仍须在 `.env` 中显式配置。

查看容器状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f api
docker compose logs -f web
```

### 端口说明

- Web：`127.0.0.1:1270 -> 8080`（唯一暴露到宿主机的端口）
- API：容器内部 `api:19000`（仅 Docker 内部网络可达）
- MySQL：容器内部 `mysql:3306`（仅 Docker 内部网络可达）
- Redis：内置模式容器内部 `redis:6379`（仅 Docker 内部网络可达）
- RabbitMQ：启用 `messaging` profile 后，容器内部 `rabbitmq:5672`，管理端 `rabbitmq:15672`（均仅 Docker 内部网络可达）
- Meilisearch：启用 `search` profile 后，容器内部 `meilisearch:7700`（仅 Docker 内部网络可达）

MySQL 与内置 Redis 默认由 `docker-compose.yml` 在内部网络启动，不会映射端口到宿主机。Web 通过 Nginx 反向代理访问 API；RabbitMQ 和 Meilisearch 仅在对应 profile 启用后启动。

Compose 中 API 容器端口固定为 `19000`，`.env` 的 `APP_PORT` 仅影响本地非 Docker 启动。默认专用网络为 `172.30.127.0/24`，如有冲突请按“可信反向代理配置”一节同时调整子网、Web 固定地址、网桥网关可信代理 `/32` 与后端可信代理 `/32`。

### 访问验证

首页：

```text
http://127.0.0.1:1270
```

公开文章接口：

```text
http://127.0.0.1:1270/api/v1/articles?page=1&size=10
```

如果返回类似下面的 JSON，说明 Nginx 到 Go API 的代理正常：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "size": 10
  }
}
```

首次进入站点后，注册第一个用户。默认逻辑下，第一个注册用户会成为管理员；如果邮箱服务保持默认关闭，首个管理员注册会自动跳过邮箱验证码。

## 1Panel 部署

1. 将项目推送到 GitHub，或直接使用仓库地址：

   ```text
   https://github.com/Elari39/Notes-of-Ashen.git
   ```

2. 在 1Panel 中创建 Docker Compose 项目，选择从 Git 源码构建。
3. Compose 文件选择仓库根目录的 `docker-compose.yml`。
4. 在 1Panel 项目目录创建 `.env`，内容可从 `.env.example` 复制后修改。需要 RabbitMQ 或 Meilisearch 时，将 `COMPOSE_PROFILES` 设为 `messaging`、`search` 或 `messaging,search`，并同步打开对应能力开关、填写凭据。
5. 启动 Compose 项目。
6. 在 1Panel 网站反向代理中配置：

   ```text
   http://127.0.0.1:1270
   ```

7. 绑定域名并开启 HTTPS。

1Panel 作为宿主机上的直接上游时，Web 容器默认只信任 Compose `app` 网桥网关 `172.30.127.1/32` 提供的 `X-Forwarded-For`。若修改 `APP_DOCKER_SUBNET`，必须同步把 `WEB_TRUSTED_PROXY_CIDR` 改为新子网网桥网关的 `/32`；禁止配置 `0.0.0.0/0` 等宽泛公网网段。

项目默认使用 Compose 内部 MySQL/Redis；启用 profile 后 RabbitMQ/Meilisearch 也仅在内部网络可达，不会映射 `3306`/`6379`/`5672`/`15672` 端口，因此不会与 1Panel 已有中间件端口冲突。Go 后端通过 `.env` 中的 DSN/URL 连接这些内部服务。

### 生产发布验收清单

每次生产发布（含首次上线）完成后，按以下清单验收：

1. **Cookie 安全属性**：生产 HTTPS 环境保持 `APP_AUTH_COOKIE_SECURE=true`。登录后在浏览器开发者工具中确认 `noa_refresh_token` Cookie 带有 `Secure; HttpOnly; SameSite=Strict`。
2. **会话续期**：登录后刷新页面，确认通过 `/api/v1/auth/refresh` 换取新 accessToken 且无需重新登录；access token 过期后同样能自动恢复会话。
3. **本机 HTTP 环境例外**：仅通过 `http://127.0.0.1:1270` 访问的本机开发环境必须设置 `APP_AUTH_COOKIE_SECURE=false`，否则浏览器不会保存 refresh Cookie，刷新页面会被迫重新登录。禁止在生产 HTTPS 环境使用 `false`。
4. **站点基址**：生产环境必须在后台站点设置中将 `siteBaseUrl` 配置为正式 HTTPS 域名（如 `https://blog.example.com`）。验收 `/rss.xml`、`/sitemap.xml` 与文章分享链接全部使用该域名，不依赖请求 Host 推断；`siteBaseUrl` 为空时后端日志会输出 `siteBaseUrl is empty` 回退提示。
5. **不可变镜像 tag**：使用 `scripts/release.ps1` 发布，确认 `.env` 的 `IMAGE_TAG` 为本次发布 tag，`docker compose config` 展开结果中的应用镜像不包含 `latest`，且 `deploy/release-history.local.jsonl` 已记录本次发布。

## 本地非 Docker 开发

非 Docker 开发时，需要自行准备 MySQL 和 Redis；仅在启用异步日志或搜索时再准备 RabbitMQ 或 Meilisearch，并根据 [etc/notes-of-ashen.yaml](etc/notes-of-ashen.yaml) 修改连接信息，或通过环境变量覆盖配置。

### 启动后端

```bash
go mod tidy
go run ./cmd/notes-of-ashen -f etc/notes-of-ashen.yaml
```

后端默认监听：

```text
http://127.0.0.1:19000
```

### 启动前端

前端必须使用 `pnpm` 管理依赖。

```bash
cd frontend
pnpm install
pnpm dev
```

开发服务器默认监听：

```text
http://127.0.0.1:3000
```

Vite 开发代理会将 `/api` 转发到 `http://127.0.0.1:19000`。

## 常用验证命令

后端测试：

```bash
go test ./...
```

后端静态检查：

```bash
go vet ./...
```

后端构建：

```bash
go build ./...
```

前端 lint：

```bash
cd frontend
pnpm lint
```

前端类型检查与生产构建：

项目提供独立的 `type-check` 脚本；`pnpm build` 会依次执行 lint、类型检查和 Vite 生产构建。

```bash
cd frontend
pnpm type-check
pnpm build
```

Docker 配置校验：

```bash
docker compose config --quiet
```

Docker 容器状态：

```bash
docker compose ps
```

## 数据持久化

Docker 部署会为本地中间件使用独立 volume：

- `notes-of-ashen_goblog_mysql_data`
- `notes-of-ashen_goblog_redis_data`
- `notes-of-ashen_goblog_rabbitmq_data`（启用 `messaging` profile 时创建）
- `notes-of-ashen_goblog_meili_data`（启用 `search` profile 时创建）
- `notes-of-ashen_goblog_media_data`

数据库结构由 [deploy/mysql/migrations](deploy/mysql/migrations) 中不可变的编号迁移统一管理。`docker compose up -d --build` 会先运行一次性 `migrate` 任务，成功后 API 才会启动；MySQL 容器只负责创建空的 `notes_of_ashen` 数据库。可用 `docker compose logs migrate` 查看版本、校验和与失败诊断。

关系完整性由迁移 `000024_add_relationship_integrity.sql` 在数据库层维护：文章/标签/项目的关联表和点赞记录对父记录使用 `CASCADE`，文章分类和历史版本分类使用 `SET NULL`，用户、文章作者、媒体创建者及历史版本操作者使用 `RESTRICT`，操作日志用户使用 `SET NULL`。迁移只清理可安全丢弃的孤儿关联；必需创建者关系若仍非法会阻止迁移并要求先修复数据。备份恢复按父表到子表的顺序写入，恢复前会先清理子表，因此兼容这些约束。

已有数据卷首次升级会建立迁移记录并按历史顺序执行全部迁移，其中包含清理无效头像、清理孤儿文章版本和删除旧流量表的前向破坏性操作。升级前必须完成可恢复的数据库备份；迁移失败时不要修改已发布的 SQL 文件，应在修复后重新执行或新增前向修复迁移。回滚依赖备份恢复。

注意：修改本地中间件账号、密码或连接串后，需要同步更新 `.env` 并重新创建相关中间件和 API 容器；已有数据卷中的 MySQL/RabbitMQ 初始账号不会因单纯修改 `.env` 自动重置。

## 常见问题

### 1270 端口被占用

修改 `.env` 中的 `WEB_PORT`：

```env
WEB_PORT=1271
```

然后重启：

```bash
docker compose up -d
```

访问地址也要相应改为：

```text
http://127.0.0.1:1271
```

### MySQL 连接失败

确认 `.env` 中的 `APP_DATABASE_DSN` 指向 Compose 内部 MySQL：

```text
notes_user:password@tcp(mysql:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local
```

如需固定 MySQL 会话时区，可在 DSN 末尾追加 `&time_zone='%2B08:00'`（URL 转义后为 `%27%2B08%3A00%27`）。

不要在容器内使用 `127.0.0.1:3306` 连接 MySQL，因为容器里的 `127.0.0.1` 只代表 API 容器自己。还要确认 `mysql` 容器健康，并检查迁移任务是否成功退出。

查看日志：

```bash
docker compose logs -f migrate api
```

### Redis 认证失败

内置 Redis 模式下，确认 `.env` 中的 `APP_REDIS_ADDR=redis:6379`、`APP_REDIS_PASSWORD` 与 Redis 容器密码一致，且 `redis` 容器健康；空密码表示无认证。外部 Redis 模式下，使用 `docker-compose.external-redis.yml` 覆盖文件启动，并确认这两个变量与外部实例一致。Redis 不可用或密码错误会使 API 按 fail-fast 策略启动失败。

查看日志：

```bash
docker compose logs -f api
```

### RabbitMQ 不可用

查看日志：

```bash
docker compose logs -f api
```

确认 `.env` 中的 `APP_RABBITMQ_URL` 指向 `rabbitmq:5672`，且账号密码与 `APP_RABBITMQ_USER`、`APP_RABBITMQ_PASSWORD` 一致。

还要确认 `.env` 的 `COMPOSE_PROFILES` 包含 `messaging`；既有部署升级到当前 Compose 文件后，若继续启用 RabbitMQ，必须补充该 profile。

如果只想临时关闭异步日志队列，可以在 `.env` 中设置：

```env
APP_RABBITMQ_ENABLED=false
```

然后重启 API：

```bash
docker compose up -d api
```

### 前端刷新页面 404

生产环境由 Nginx 提供前端页面，并通过 `try_files` 将 React 路由回退到 `index.html`。如果刷新后台页、登录页或文章页出现 404，请检查：

```text
deploy/nginx/default.conf
```

### `.env` 修改后没有生效

修改 `.env` 后，重新创建相关容器：

```bash
docker compose up -d
```

如果是镜像构建阶段相关变量，还需要重新构建：

```bash
docker compose up -d --build
```

## 常用 API

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/captcha`
- `POST /api/v1/auth/verify-code/send`
- `POST /api/v1/auth/password/reset`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `POST /api/v1/traffic/visit`
- `GET /api/v1/articles`
- `GET /api/v1/articles/:id`
- `GET /api/v1/articles/:id/context`
- `POST /api/v1/articles/:id/like`
- `POST /api/v1/articles/ai/assist`
- `POST /api/v1/articles/import`
- `GET /api/v1/articles/:id/preview`
- `GET /api/v1/articles/:id/export`
- `GET /api/v1/articles/:id/versions`
- `POST /api/v1/articles/:id/versions/:versionNo/restore`
- `GET /api/v1/articles/:id/versions/:versionNo`
- `GET /api/v1/categories`
- `POST /api/v1/categories`
- `PUT /api/v1/categories/:id`
- `DELETE /api/v1/categories/:id`
- `GET /api/v1/tags`
- `POST /api/v1/tags`
- `PUT /api/v1/tags/:id`
- `DELETE /api/v1/tags/:id`
- `GET /api/v1/site/settings`
- `GET /api/v1/site/projects`
- `GET /api/v1/users/me`
- `PUT /api/v1/users/me`
- `PUT /api/v1/users/me/password`
- `POST /api/v1/users/me/verify-code/send`
- 管理后台（需管理员鉴权）：
  - `POST/PUT/DELETE /api/v1/articles`、`PUT /api/v1/articles/:id`、`PATCH /api/v1/articles/:id/status`
  - `GET /api/v1/admin/articles`、`GET /api/v1/admin/stats`、`GET /api/v1/admin/logs`
  - `GET /api/v1/admin/users`、`PATCH /api/v1/admin/users/:id/status`、`PATCH /api/v1/admin/users/:id/role`
  - `GET/PUT /api/v1/admin/site/settings`、`GET/PUT /api/v1/admin/site/projects`
  - `GET/PUT /api/v1/admin/ai/settings`、`POST /api/v1/admin/ai/models`、`POST /api/v1/admin/ai/test`、`POST /api/v1/admin/search/reindex`

完整说明见 [docs/API.md](docs/API.md)。

## 维护建议

- 备份、异地副本、恢复演练与发布回滚的完整运维流程见 [docs/OPERATIONS.md](docs/OPERATIONS.md)；日常备份使用 `pwsh scripts/backup.ps1`。
- 生产环境务必填写真实强随机密码替换所有 `<REPLACE_…>` 占位符，并设置稳定、足够长的 `APP_AUTH_ACCESS_SECRET`。
- 前端依赖管理统一使用 `pnpm`，不要混用 `npm` 或 `yarn`。
- 不要提交 `.env`、数据库备份、日志文件或任何真实密钥；数据库备份建议放在仓库目录外。
- 升级前先备份 MySQL 数据卷或远程 MySQL 数据，并确认 Redis、RabbitMQ 的持久化策略符合预期。
- 修改后建议至少运行：

  ```bash
  go test ./...
  go vet ./...
  go build ./...
  cd frontend
  pnpm lint
  pnpm type-check
  pnpm build
  docker compose config --quiet
  ```
