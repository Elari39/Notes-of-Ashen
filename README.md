# Notes of Ashen

`Notes of Ashen` 是一个前后端分离的个人博客系统。后端使用 Go 与 go-zero 风格组织代码，前端使用 React、TypeScript、Vite 与 Tailwind CSS 构建页面，部署侧提供 Docker Compose、Nginx 与 1Panel 友好的运行方案。

本机 Docker 默认访问地址：

```text
http://127.0.0.1:1270
```

## 快速开始

适合已经准备好远程 MySQL、Redis 和 RabbitMQ 后，用 Docker / 1Panel 跑起来：

> ⚠️ 复制 `.env` 后，必须把其中所有 `<REPLACE_…>` 占位符替换为真实强随机值，否则后端启动或 `docker compose config` 会直接失败。

```powershell
Copy-Item .env.example .env
# 编辑 .env，替换所有 <REPLACE_…> 占位符，填入远程 MySQL、Redis、RabbitMQ 连接信息
docker compose config --quiet
docker compose up -d --build
```

Linux / macOS：

```bash
cp .env.example .env
# 编辑 .env，替换所有 <REPLACE_…> 占位符，填入远程 MySQL、Redis、RabbitMQ 连接信息
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
- 管理后台：用户管理、用户状态管理、站点设置、简历与项目管理、操作日志查看、访问趋势和来源统计。
- 站点能力：RSS、Sitemap、站点标题、描述、关键词、Prerender.io 预渲染配置等。
- 流量统计：公开页面自动上报 PV、UV 与来源，后台展示最近 30 天趋势。
- 异步日志：通过 RabbitMQ 投递操作事件，并写入 `operation_logs`。
- 统一响应：接口成功时返回 `{ "code": 0, "message": "success", "data": ... }`。

## 技术栈

- 后端：Go 1.25、go-zero REST、MySQL 8.4、Redis 7.4、Meilisearch 1.13、RabbitMQ 4、JWT、bcrypt。
  - 上述 MySQL / Redis / Meilisearch / RabbitMQ 版本为远程依赖的**推荐版本**，需由外部服务提供（compose 不再启动它们）。
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

项目使用仓库根目录的 `.env` 为 Docker Compose 提供部署配置。`.env` 包含远程 MySQL、Redis、RabbitMQ 连接信息和 JWT 密钥，不应提交到 Git。

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
- `APP_AUTH_ACCESS_SECRET`：JWT 签名密钥，生产环境必须替换为足够长的随机字符串，建议至少 32 位。
- `APP_DATABASE_DSN`：远程 MySQL 连接串，例如 `notes_user:password@tcp(mysql.example.com:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local`。如需固定 MySQL 会话时区，可追加 `&time_zone='%2B08:00'`（URL 转义后为 `%27%2B08%3A00%27`）。
- `APP_DATABASE_MAX_OPEN_CONNS`：MySQL 最大打开连接数。
- `APP_DATABASE_MAX_IDLE_CONNS`：MySQL 最大空闲连接数。
- `APP_REDIS_ADDR`：远程 Redis 地址，例如 `redis.example.com:6379`。
- `APP_REDIS_PASSWORD`：Redis 密码；无密码时留空。
- `APP_REDIS_DB`：Redis DB 编号，默认 `0`。
- `APP_RABBITMQ_ENABLED`：是否启用 RabbitMQ 异步日志，默认 `true`。
- `APP_RABBITMQ_URL`：远程 RabbitMQ AMQP 地址，例如 `amqp://rabbit_user:password@rabbitmq.example.com:5672/`。
- `APP_SEARCH_ENABLED`：是否启用 Meilisearch 全文搜索，默认 `false`，关闭时自动回退 MySQL 查询。
- `APP_MEILISEARCH_HOST`：API 访问 Meilisearch 的地址，Docker 部署默认 `http://meilisearch:7700`。
- `APP_MEILISEARCH_API_KEY`：Meilisearch API Key；Docker 部署时也作为 Meilisearch Master Key。即使搜索默认关闭，Compose 中的 Meilisearch 容器也需要该值，请在 `.env` 中填写强随机字符串。
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
- `APP_AI_KEY_ENCRYPTION_SECRET`：后台保存 AI API Key 时使用的加密密钥，生产环境应设置为不同于 `APP_AUTH_ACCESS_SECRET` 的长随机值。
- `APP_AI_TIMEOUT_SECONDS`：AI 请求兼容超时时间，默认 `600` 秒（已由 `APP_AI_FIRST_BYTE_TIMEOUT_SECONDS`/`APP_AI_STREAM_TIMEOUT_SECONDS`/`APP_AI_NON_STREAM_TIMEOUT_SECONDS` 三段细化覆盖，本项保留作兜底）。
- `APP_TRUSTED_PROXY_CIDRS`：可信反向代理 CIDR，默认留空表示不信任客户端传入的 `X-Forwarded-*` / `X-Real-IP`。
- `PRERENDER_ENABLED`：是否启用 Prerender.io crawler 预渲染，`0` 关闭，`1` 启用。
- `PRERENDER_SERVICE_URL`：Prerender.io 服务地址，默认 `https://service.prerender.io`。
- `PRERENDER_TOKEN`：Prerender.io Token，只能写入真实 `.env` 或受控环境变量。
- `APP_GITHUB_TOKEN`：可选 GitHub Token；留空时使用公开匿名额度，不要提交真实 Token（当前版本后端未读取，预留）。
- `WEB_PORT`：本机 Web 访问端口，默认 `1270`。

不要把真实 `.env` 内容写入 README、Issue、提交记录或截图中。

### 1Panel 远程中间件配置

当前 `docker-compose.yml` 不再启动内置 MySQL、Redis 和 RabbitMQ，API 容器会直接读取 `.env` 中的远程连接配置。1Panel 部署前请先完成：

- 远程 MySQL 创建数据库 `notes_of_ashen`，创建专用用户并只授权给 1Panel 服务器 IP 或内网网段。
- 在远程 MySQL 执行 [deploy/mysql/schema.sql](deploy/mysql/schema.sql) 初始化表结构；旧库迁移前先备份，再按实际版本补执行 [deploy/mysql](deploy/mysql) 下的增量脚本。增量脚本应在 `notes_of_ashen` 库中执行；现有脚本均显式 `USE notes_of_ashen;`，不要在其他库中直接运行。新库可直接用 `schema.sql` 一步到位，无需补跑增量脚本。

  增量脚本按以下时间顺序执行（已在 `schema.sql` 基础上）：

  1. `add_site_settings.sql` — 站点设置表
  2. `add_content_growth_features.sql` — 文章排程字段、文章版本表 `article_versions`（仅基础列）
  3. `add_article_pin_priority.sql` — 补 `article_versions.is_pinned` / `display_priority`
  4. `add_resume_portfolio_interaction_geo.sql` — 补 `article_versions.like_count`，简历/作品集/点赞表
  5. `alter_site_settings_value_text.sql` — 站点设置 value 列改 TEXT
  6. `add_traffic_ai_import_features.sql` — 流量/AI/导入相关字段
  7. `add_ai_settings.sql` — AI 设置表
  8. `add_public_page_content_settings.sql`、`add_public_page_visibility_settings.sql` — 公开页内容/可见性设置
  9. `add_article_fulltext_index.sql` — 文章全文索引
  10. `drop_traffic_geo.sql` — 移除流量地理表（`traffic_geo_*`）
  11. `cleanup_invalid_avatar_url.sql` — 清理无效头像 URL
  12. `add_article_tags_tag_index.sql` — 文章标签索引

  > 注意：`article_versions` 表的 `like_count` / `is_pinned` / `display_priority` 三列分别由第 3、4 步脚本补齐，必须在 `add_content_growth_features.sql`（第 2 步）之后执行，否则 `model/article.go` 的 `articleVersionSelectFields` 查询会因缺列报 `Unknown column`。
- 确认远程 Redis、RabbitMQ 防火墙和安全组允许 1Panel 服务器访问，避免对公网裸露。
- 如果 MySQL DSN 或 RabbitMQ URL 的密码包含 `@`、`:`、`/`、`?`、`#` 等 URL 分隔符，请先做 URL 转义，或使用不含这些分隔符的强随机密码。

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
APP_AI_KEY_ENCRYPTION_SECRET=replace-with-a-different-long-random-secret
APP_AI_TIMEOUT_SECONDS=600
```

`APP_AI_BASE_URL` 可以填写兼容 OpenAI Chat Completions 的基础地址，例如 `https://api.example.com/v1`；如果服务商只提供完整端点，也可以填写到 `/chat/completions`。不要把真实 API Key 写入 README、Issue、提交记录或截图中。

`APP_AI_KEY_ENCRYPTION_SECRET` 只用于加密后台保存的 AI API Key。旧版本已保存的密文仍可读取；管理员再次保存 AI 设置时会写入新格式密文。

### 可信反向代理配置

后端默认不信任客户端传入的 `X-Forwarded-*` / `X-Real-IP` 请求头，限流、操作日志、流量统计和 RSS/Sitemap 基础 URL 都会优先使用直连信息。只有在确认 API 只接受可信 Nginx、1Panel 或其他反向代理转发时，才在真实 `.env` 中配置代理出口地址或网段：

```env
APP_TRUSTED_PROXY_CIDRS=172.18.0.0/16
```

不要在 API 可被公网或不可信客户端直连时配置过宽的网段。

### 全文搜索配置

Docker Compose 已包含 Meilisearch 服务，但搜索默认关闭，主站启动不依赖 Meilisearch 健康状态。需要启用全文搜索时，在真实 `.env` 中设置：

```env
APP_SEARCH_ENABLED=true
APP_MEILISEARCH_HOST=http://meilisearch:7700
APP_MEILISEARCH_API_KEY=replace-with-a-long-random-key
APP_MEILISEARCH_INDEX=articles
```

重新创建服务后，使用 `editor` 或 `admin` 登录后台，并调用 `POST /api/v1/admin/search/reindex` 全量重建文章索引。Meilisearch 不可用时，公开文章搜索会回退到 MySQL 查询。

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

- Web：`127.0.0.1:1270 -> 80`（唯一暴露到宿主机的端口）
- API：容器内部 `api:19000`（仅 Docker 内部网络可达）
- Meilisearch：容器内部 `meilisearch:7700`（仅 Docker 内部网络可达）

MySQL、Redis、RabbitMQ 由远程服务提供（见 1Panel 远程中间件配置），不在 `docker-compose.yml` 内启动，也不会映射端口到宿主机。Web 通过 Nginx 反向代理访问 API。

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
4. 在 1Panel 项目目录创建 `.env`，内容可从 `.env.example` 复制后修改。
5. 启动 Compose 项目。
6. 在 1Panel 网站反向代理中配置：

   ```text
   http://127.0.0.1:1270
   ```

7. 绑定域名并开启 HTTPS。

项目使用远程 MySQL/Redis/RabbitMQ，不会在宿主机映射 `3306`/`6379`/`5672` 端口，因此不会与 1Panel 已有中间件端口冲突。Go 后端通过 `.env` 中的 DSN/URL 连接远程服务。

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

Docker 部署只会为本地 Meilisearch 使用独立 volume：

- `notes-of-ashen_goblog_meili_data`

MySQL、Redis 和 RabbitMQ 数据由远程服务自行管理。首次部署前需要在远程 MySQL 执行 [deploy/mysql/schema.sql](deploy/mysql/schema.sql) 初始化数据库和表结构。

注意：远程服务的账号、密码或访问白名单变更后，需要同步更新 `.env` 并重新创建 API 容器。

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

确认 `.env` 中的 `APP_DATABASE_DSN` 指向远程 MySQL：

```text
notes_user:password@tcp(mysql.example.com:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local
```

如需固定 MySQL 会话时区，可在 DSN 末尾追加 `&time_zone='%2B08:00'`（URL 转义后为 `%27%2B08%3A00%27`）。

不要在容器内使用 `127.0.0.1:3306` 连接远程 MySQL，因为容器里的 `127.0.0.1` 只代表 API 容器自己。还要确认远程 MySQL 已执行初始化 SQL，并允许 1Panel 服务器访问。

查看日志：

```bash
docker compose logs -f api
```

### Redis 认证失败

确认 `.env` 中的 `APP_REDIS_ADDR`、`APP_REDIS_PASSWORD` 和远程 Redis 实际配置一致，并允许 1Panel 服务器访问。

查看日志：

```bash
docker compose logs -f api
```

### RabbitMQ 不可用

查看日志：

```bash
docker compose logs -f api
```

确认 `.env` 中的 `APP_RABBITMQ_URL` 指向远程 RabbitMQ AMQP 地址，并且账号有声明 exchange、queue 和 bind 的权限。

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
- `GET /api/v1/site/resume`
- `GET /api/v1/site/projects`
- `GET /api/v1/users/me`
- `PUT /api/v1/users/me`
- `PUT /api/v1/users/me/password`
- `POST /api/v1/users/me/verify-code/send`
- 管理后台（需管理员鉴权）：
  - `POST/PUT/DELETE /api/v1/articles`、`PUT /api/v1/articles/:id`、`PATCH /api/v1/articles/:id/status`
  - `GET /api/v1/admin/articles`、`GET /api/v1/admin/stats`、`GET /api/v1/admin/logs`
  - `GET/PUT /api/v1/admin/users`、`PATCH /api/v1/admin/users/:id/status`、`PATCH /api/v1/admin/users/:id/role`
  - `GET/PUT /api/v1/admin/site/settings`、`GET/PUT /api/v1/admin/site/resume`、`GET/PUT /api/v1/admin/site/projects`
  - `GET/PUT /api/v1/admin/ai/settings`、`POST /api/v1/admin/search/reindex`

完整说明见 [docs/API.md](docs/API.md)。

## 维护建议

- 生产环境务必填写真实强随机密码替换所有 `<REPLACE_…>` 占位符，并设置 `APP_AUTH_ACCESS_SECRET` 和 `APP_AI_KEY_ENCRYPTION_SECRET`。
- 前端依赖管理统一使用 `pnpm`，不要混用 `npm` 或 `yarn`。
- 不要提交 `.env`、数据库备份、日志文件或任何真实密钥；数据库备份建议放在仓库目录外。
- 升级前先备份远程 MySQL 数据，并确认远程 Redis、RabbitMQ 的持久化策略符合预期。
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
