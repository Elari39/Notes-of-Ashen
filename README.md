# Notes of Ashen

`Notes of Ashen` 是一个前后端分离的个人博客系统。后端使用 Go 与 go-zero 风格组织代码，前端使用 React、TypeScript、Vite 与 Tailwind CSS 构建页面，部署侧提供 Docker Compose、Nginx 与 1Panel 友好的运行方案。

本机 Docker 默认访问地址：

```text
http://127.0.0.1:1270
```

## 快速开始

适合第一次拉取项目后直接在本机用 Docker 跑起来：

```powershell
Copy-Item .env.example .env
docker compose config --quiet
docker compose up -d --build
```

Linux / macOS：

```bash
cp .env.example .env
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
- 文章管理：创建、编辑、删除、发布、归档、草稿预览、版本查看、版本恢复、Markdown 导入/导出、AI 辅助创作、置顶与显示优先级。
- 内容展示：公开文章列表、文章详情、归档、Meilisearch 全文搜索、Markdown 渲染、代码高亮、LaTeX 数学公式、文章目录和点赞反馈。
- 简历与作品集：结构化简历时间轴、技能树、知识图谱、前端 PDF 导出、作品集画廊和项目标签。
- 分类与标签：公开读取，后台可创建、更新和删除。
- 管理后台：用户管理、用户状态管理、站点设置、简历与项目管理、操作日志查看、访问趋势、来源统计和地理分布。
- 站点能力：RSS、Sitemap、站点标题、描述、关键词、Prerender.io 预渲染配置等。
- 流量统计：公开页面自动上报 PV、UV、来源与可选 GeoIP 地理位置，后台展示最近 30 天趋势。
- 异步日志：通过 RabbitMQ 投递操作事件，并写入 `operation_logs`。
- 统一响应：接口成功时返回 `{ "code": 0, "message": "success", "data": ... }`。

## 技术栈

- 后端：Go 1.25、go-zero REST、MySQL 8.4、Redis 7.4、Meilisearch 1.13、RabbitMQ 4、JWT、bcrypt。
- 前端：React 18、TypeScript、Vite 5、Tailwind CSS 3、Zustand、Axios、Framer Motion、Recharts、React Markdown。
- 部署：Docker、Docker Compose、Nginx、1Panel。
- 文档与脚本：API 文档位于 [docs/API.md](docs/API.md)，数据库脚本位于 [deploy/mysql](deploy/mysql)。

## 项目结构

```text
.
├── api/                    # go-zero API 描述文件
├── cmd/notes-of-ashen/     # 后端服务入口
├── deploy/
│   ├── mysql/              # MySQL 初始化与增量脚本
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

项目使用仓库根目录的 `.env` 为 Docker Compose 提供部署配置。`.env` 包含数据库密码、Redis 密码、RabbitMQ 密码和 JWT 密钥，不应提交到 Git。

从模板复制：

```bash
cp .env.example .env
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
```

至少需要检查并替换这些值：

- `APP_DISPLAY_NAME`：站点对外展示名称，默认 `Notes of Ashen`。
- `APP_AUTH_ACCESS_SECRET`：JWT 签名密钥，生产环境必须替换为足够长的随机字符串。
- `MYSQL_ROOT_PASSWORD`：项目内部 MySQL root 密码。
- `MYSQL_USER`：项目内部 MySQL 普通用户。
- `MYSQL_PASSWORD`：项目内部 MySQL 普通用户密码。
- `REDIS_PASSWORD`：Redis 密码。
- `RABBITMQ_DEFAULT_USER`：RabbitMQ 用户名。
- `RABBITMQ_DEFAULT_PASS`：RabbitMQ 密码。
- `APP_SEARCH_ENABLED`：是否启用 Meilisearch 全文搜索，默认 `false`，关闭时自动回退 MySQL 查询。
- `APP_MEILISEARCH_HOST`：API 访问 Meilisearch 的地址，Docker 部署默认 `http://meilisearch:7700`。
- `APP_MEILISEARCH_API_KEY`：Meilisearch API Key；Docker 部署时也作为 Meilisearch Master Key。
- `APP_MEILISEARCH_INDEX`：文章索引名，默认 `articles`。
- `APP_EMAIL_ENABLED`：是否启用邮箱验证码，使用 QQ 邮箱时设置为 `true`。
- `APP_EMAIL_SMTP_USERNAME`：QQ 邮箱账号，例如 `yourname@qq.com`。
- `APP_EMAIL_SMTP_PASSWORD`：QQ 邮箱 SMTP 授权码，不是 QQ 登录密码。
- `APP_EMAIL_FROM`：发件邮箱，通常和 `APP_EMAIL_SMTP_USERNAME` 一致；留空时后端会回退使用 SMTP 用户名。
- `APP_EMAIL_FROM_NAME`：发件人名称。
- `APP_AI_ENABLED`：是否启用文章 AI 辅助创作。
- `APP_AI_BASE_URL`：兼容 OpenAI Chat Completions 的接口基础地址。
- `APP_AI_API_KEY`：AI 服务 API Key，只能写入真实 `.env` 或受控环境变量。
- `APP_AI_MODEL`：AI 辅助使用的模型名称。
- `APP_AI_TIMEOUT_SECONDS`：AI 非流式请求兼容超时时间，默认 `600` 秒。
- `APP_GEOIP_DATABASE_PATH`：API 容器内 GeoIP/GeoLite2 City `.mmdb` 文件路径，Docker 部署默认 `/data/GeoLite2-City.mmdb`。
- `MAXMIND_ACCOUNT_ID`：可选 MaxMind Account ID，用于 Docker 部署自动下载 `GeoLite2-City.mmdb`。
- `MAXMIND_LICENSE_KEY`：可选 MaxMind License Key，用于 Docker 部署自动下载 `GeoLite2-City.mmdb`，只能写入真实 `.env` 或受控环境变量。
- `PRERENDER_ENABLED`：是否启用 Prerender.io crawler 预渲染，`0` 关闭，`1` 启用。
- `PRERENDER_SERVICE_URL`：Prerender.io 服务地址，默认 `https://service.prerender.io`。
- `PRERENDER_TOKEN`：Prerender.io Token，只能写入真实 `.env` 或受控环境变量。
- `APP_GITHUB_TOKEN`：可选 GitHub Token；留空时使用公开匿名额度，不要提交真实 Token。
- `WEB_PORT`：本机 Web 访问端口，默认 `1270`。

不要把真实 `.env` 内容写入 README、Issue、提交记录或截图中。

### QQ 邮箱验证码配置

在 QQ 邮箱开启 SMTP 服务后，将授权码写入真实 `.env`：

```env
APP_EMAIL_ENABLED=true
APP_EMAIL_SMTP_HOST=smtp.qq.com
APP_EMAIL_SMTP_PORT=465
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

后台文章编辑页提供摘要/SEO 元数据生成、纠错和润色能力。该功能默认关闭，生产启用时只在真实 `.env` 中填写密钥：

```env
APP_AI_ENABLED=true
APP_AI_BASE_URL=https://api.example.com/v1
APP_AI_API_KEY=your-ai-api-key
APP_AI_MODEL=your-model-name
APP_AI_TIMEOUT_SECONDS=600
```

`APP_AI_BASE_URL` 可以填写兼容 OpenAI Chat Completions 的基础地址，例如 `https://api.example.com/v1`；如果服务商只提供完整端点，也可以填写到 `/chat/completions`。不要把真实 API Key 写入 README、Issue、提交记录或截图中。

### 全文搜索配置

Docker Compose 已包含 Meilisearch 服务，但搜索默认关闭，主站启动不依赖 Meilisearch 健康状态。需要启用全文搜索时，在真实 `.env` 中设置：

```env
APP_SEARCH_ENABLED=true
APP_MEILISEARCH_HOST=http://meilisearch:7700
APP_MEILISEARCH_API_KEY=replace-with-a-long-random-key
APP_MEILISEARCH_INDEX=articles
```

重新创建服务后，使用 `editor` 或 `admin` 登录后台，并调用 `POST /api/v1/admin/search/reindex` 全量重建文章索引。Meilisearch 不可用时，公开文章搜索会回退到 MySQL 查询。

### GeoIP 与预渲染配置

访客地理位置分析使用离线 `.mmdb` 数据库。Docker 部署时，如果真实 `.env` 中填写了 MaxMind 凭据，`geoipupdate` 容器会在 `data/GeoLite2-City.mmdb` 不存在时自动下载；如果文件已存在会直接复用，避免消耗 MaxMind 每日下载额度。

```env
APP_GEOIP_DATABASE_PATH=/data/GeoLite2-City.mmdb
MAXMIND_ACCOUNT_ID=your-maxmind-account-id
MAXMIND_LICENSE_KEY=your-maxmind-license-key
```

然后执行常规部署命令即可：

```bash
docker compose up -d --build
```

查看自动下载日志：

```bash
docker compose logs geoipupdate
```

如果不填写 MaxMind 凭据，或下载失败，API 仍会启动，访客地理位置记录为 `Unknown`。也可以手动下载 `GeoLite2-City.mmdb` 并放到仓库根目录的 `data/GeoLite2-City.mmdb`，Docker 会把它只读挂载到 API 容器的 `/data/GeoLite2-City.mmdb`。

SPA SEO 预渲染默认关闭。需要为搜索引擎 crawler 返回预渲染 HTML 时，在真实 `.env` 中设置：

```env
PRERENDER_ENABLED=1
PRERENDER_SERVICE_URL=https://service.prerender.io
PRERENDER_TOKEN=your-prerender-token
```

不要提交 `.mmdb` 文件、MaxMind License Key、Prerender Token 或任何真实密钥。

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

查看容器状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs geoipupdate
docker compose logs -f api
docker compose logs -f web
```

### 端口说明

- Web：`127.0.0.1:1270 -> 80`
- API：容器内部 `api:19000`
- MySQL：容器内部 `mysql:3306`
- Redis：容器内部 `redis:6379`
- Meilisearch：容器内部 `meilisearch:7700`
- RabbitMQ：容器内部 `rabbitmq:5672`

只有 Web 会暴露到宿主机的 `1270` 端口。API、MySQL、Redis、Meilisearch、RabbitMQ 默认只在 Docker 内部网络访问。

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

首次进入站点后，注册第一个用户。默认逻辑下，第一个注册用户会成为管理员。

## 1Panel 部署

1. 将项目推送到 GitHub，或直接使用仓库地址：

   ```text
   https://github.com/Elari39/Notes-of-Ashen.git
   ```

2. 在 1Panel 中创建 Docker Compose 项目，选择从 Git 源码构建。
3. Compose 文件选择仓库根目录的 `docker-compose.yml`。
4. 在 1Panel 项目目录创建 `.env`，内容可从 `.env.example` 复制后修改。
5. 启动 Compose 项目。
6. 在 1Panel 网站反向代理中配置：

   ```text
   http://127.0.0.1:1270
   ```

7. 绑定域名并开启 HTTPS。

项目内部 MySQL 不会映射到宿主机 `3306`，因此不会和 1Panel 已有 MySQL 容器端口冲突。Go 后端通过 Docker 内部服务名 `mysql:3306` 连接项目内部 MySQL。

## 本地非 Docker 开发

非 Docker 开发时，需要你自己准备 MySQL、Redis 和 RabbitMQ，并根据 [etc/notes-of-ashen.yaml](etc/notes-of-ashen.yaml) 修改连接信息，或通过环境变量覆盖配置。

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

项目当前没有单独的 `type-check` 脚本，`pnpm build` 会先执行 `tsc`，再执行 Vite 生产构建。

```bash
cd frontend
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

Docker 部署会使用独立 volume 保存数据：

- `notes-of-ashen_goblog_mysql_data`
- `notes-of-ashen_goblog_redis_data`
- `notes-of-ashen_goblog_meili_data`
- `notes-of-ashen_goblog_rabbitmq_data`

GeoIP 数据库使用宿主机目录 `./data/GeoLite2-City.mmdb`，不会写入 Docker volume，也不会进入 Git 或镜像构建上下文。

首次启动时，MySQL 会执行 [deploy/mysql/schema.sql](deploy/mysql/schema.sql) 初始化数据库和表结构。

注意：如果容器已经创建过并且 volume 已存在，之后修改 `.env` 中的数据库密码不会自动修改旧数据库中的用户密码。遇到密码不一致时，应手动调整数据库用户，或在确认不需要旧数据后再清理 volume。

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

Docker 部署时，API 应连接：

```text
mysql:3306
```

不要在容器内使用 `127.0.0.1:3306` 连接 MySQL，因为容器里的 `127.0.0.1` 只代表 API 容器自己。

查看日志：

```bash
docker compose logs -f mysql
docker compose logs -f api
```

### Redis 认证失败

确认 `.env` 中的 `REDIS_PASSWORD` 与 Redis 容器启动配置一致。

查看日志：

```bash
docker compose logs -f redis
docker compose logs -f api
```

### RabbitMQ 不可用

查看日志：

```bash
docker compose logs -f rabbitmq
docker compose logs -f api
```

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
- `GET /api/v1/categories`
- `GET /api/v1/tags`
- `GET /api/v1/site/settings`
- `GET /api/v1/site/resume`
- `GET /api/v1/site/projects`
- `GET /api/v1/users/me`
- `GET /api/v1/admin/articles`
- `GET /api/v1/admin/stats`
- `GET /api/v1/admin/users`
- `GET /api/v1/admin/logs`
- `POST /api/v1/admin/search/reindex`
- `GET /api/v1/admin/site/resume`
- `GET /api/v1/admin/site/projects`

完整说明见 [docs/API.md](docs/API.md)。

## 维护建议

- 生产环境务必替换 `.env` 中的默认密码和 `APP_AUTH_ACCESS_SECRET`。
- 前端依赖管理统一使用 `pnpm`，不要混用 `npm` 或 `yarn`。
- 不要提交 `.env`、数据库备份、日志文件或任何真实密钥；数据库备份建议放在仓库目录外。
- 升级前先备份 Docker volume 中的数据。
- 修改后建议至少运行：

  ```bash
  go test ./...
  go vet ./...
  go build ./...
  cd frontend
  pnpm lint
  pnpm build
  docker compose config --quiet
  ```
