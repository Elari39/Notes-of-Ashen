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
  - Docker Compose 默认启动本地 MySQL / Redis / RabbitMQ；Meilisearch 可通过 Compose 的 `search` profile 按需启动。
- 前端：React 18、TypeScript、Vite 5、Tailwind CSS 3、Zustand、Axios、Framer Motion、ECharts、React Markdown。
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
- `APP_MYSQL_DATABASE`：本地 Compose 初始化数据库名，默认 `notes_of_ashen`。
- `APP_MYSQL_USER`：本地 Compose MySQL 应用用户，默认 `notes_user`。
- `APP_MYSQL_PASSWORD`：本地 Compose MySQL 应用用户密码，需和 `APP_DATABASE_DSN` 中的密码保持一致。
- `APP_DATABASE_DSN`：API 使用的 MySQL 连接串，Compose 默认连接本地服务：`notes_user:password@tcp(mysql:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local`。如需固定 MySQL 会话时区，可追加 `&time_zone='%2B08:00'`（URL 转义后为 `%27%2B08%3A00%27`）。
- `APP_DATABASE_MAX_OPEN_CONNS`：MySQL 最大打开连接数。
- `APP_DATABASE_MAX_IDLE_CONNS`：MySQL 最大空闲连接数。
- `APP_REDIS_ADDR`：Redis 地址，Compose 默认 `redis:6379`。
- `APP_REDIS_PASSWORD`：Redis 密码；无密码时留空。
- `APP_REDIS_DB`：Redis DB 编号，默认 `0`。
- `APP_RABBITMQ_USER`：本地 Compose RabbitMQ 用户，默认 `notes_user`。
- `APP_RABBITMQ_PASSWORD`：本地 Compose RabbitMQ 密码，需和 `APP_RABBITMQ_URL` 中的密码保持一致。
- `APP_RABBITMQ_ENABLED`：是否启用 RabbitMQ 异步日志，Compose 默认 `true`。
- `APP_RABBITMQ_URL`：RabbitMQ AMQP 地址，Compose 默认连接本地服务：`amqp://notes_user:password@rabbitmq:5672/`。
- `APP_RABBITMQ_EXCHANGE`：RabbitMQ 交换器名，默认 `notes-of-ashen.events`，通常无需修改。
- `APP_RABBITMQ_QUEUE`：RabbitMQ 队列名，默认 `notes-of-ashen.operation_logs`，通常无需修改。
- `APP_RABBITMQ_ROUTING_KEY`：RabbitMQ 路由键，默认 `operation.log`，通常无需修改。
- `APP_SEARCH_ENABLED`：是否启用 Meilisearch 全文搜索，默认 `false`，关闭时自动回退 MySQL 查询。
- `APP_MEILISEARCH_HOST`：API 访问 Meilisearch 的地址，Docker 部署默认 `http://meilisearch:7700`。
- `APP_MEILISEARCH_API_KEY`：Meilisearch API Key；Docker 部署时也作为 Meilisearch Master Key。启用搜索和 Compose `search` profile 时，请在 `.env` 中填写强随机字符串。
- `APP_MEILISEARCH_INDEX`：文章索引名，默认 `articles`。
- `APP_EMAIL_ENABLED`：是否启用邮箱验证码，使用 QQ 邮箱时设置为 `true`。
- `APP_EMAIL_SMTP_HOST`：SMTP 服务器地址，默认 `smtp.qq.com`。
- `APP_EMAIL_SMTP_PORT`：SMTP 端口，默认 `465`（隐式 TLS）；使用 587 时需配合 `APP_EMAIL_TLS_MODE=starttls`。
- `APP_EMAIL_TLS_MODE`：TLS 模式，`implicit`（465 隐式 TLS，默认）、`starttls`（587 STARTTLS）、`none`（明文，仅内网测试）。
- `APP_EMAIL_SMTP_USERNAME`：QQ 邮箱账号，例如 `yourname@qq.com`。
- `APP_EMAIL_SMTP_PASSWORD`：QQ 邮箱 SMTP 授权码，不是 QQ 登录密码。
- `APP_EMAIL_FROM`：发件邮箱，通常和 `APP_EMAIL_SMTP_USERNAME` 一致；留空时后端会回退使用 SMTP 用户名。
- `APP_EMAIL_FROM_NAME`：发件人名称。
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

当前 `docker-compose.yml` 默认启动本地 MySQL、Redis 和 RabbitMQ，API 容器通过 `.env` 中的 `mysql`、`redis`、`rabbitmq` 服务名访问它们。首次启动新的 MySQL 数据卷时会自动执行 [deploy/mysql/schema.sql](deploy/mysql/schema.sql) 初始化数据库和表结构。

如果你要改用远程 MySQL，请先完成：

- 远程 MySQL 创建数据库 `notes_of_ashen`，创建专用用户并只授权给 1Panel 服务器 IP 或内网网段。
- 在远程 MySQL 执行 [deploy/mysql/schema.sql](deploy/mysql/schema.sql) 初始化表结构；旧库迁移前先备份，再按实际版本补执行 [deploy/mysql](deploy/mysql) 下的增量脚本。增量脚本应在 `notes_of_ashen` 库中执行；现有脚本均显式 `USE notes_of_ashen;`，不要在其他库中直接运行。新库可直接用 `schema.sql` 一步到位，无需补跑增量脚本。

  增量脚本按以下时间顺序执行（已在 `schema.sql` 基础上）：

  1. `add_site_settings.sql` — 站点设置表
  2. `add_content_growth_features.sql` — 文章排程字段、文章版本表 `article_versions`（仅基础列）
  3. `add_article_pin_priority.sql` — 补 `article_versions.is_pinned` / `display_priority`
  4. `add_resume_portfolio_interaction_geo.sql` — 补 `article_versions.like_count`，简历/作品集/点赞表
  5. `alter_site_settings_value_text.sql` — 站点设置 value 列改 MEDIUMTEXT
  6. `add_traffic_ai_import_features.sql` — 流量/AI/导入相关字段
  7. `add_ai_settings.sql` — AI 设置表
  8. `add_public_page_content_settings.sql`、`add_public_page_visibility_settings.sql` — 公开页内容/可见性设置
  9. `add_article_fulltext_index.sql` — 文章全文索引
  10. `drop_traffic_geo.sql` — 移除流量地理表（`traffic_geo_*`）
  11. `cleanup_invalid_avatar_url.sql` — 清理无效头像 URL
  12. `add_article_tags_tag_index.sql` — 文章标签索引
  13. `add_article_category_author_index.sql` — 文章分类/作者索引
  14. `add_operation_logs_index.sql` — operation_logs 表 created_at / user_id 索引（幂等）
  15. `add_users_admin_state_index.sql` — 用户角色/状态复合索引，缩小管理员并发保护的锁定扫描范围（幂等）
  16. `add_operation_logs_filter_indexes.sql` — operation_logs 表事件/来源 IP 与时间复合索引（幂等）
  17. `add_ai_api_format_setting.sql` — 为 AI 设置补充 `apiFormat`，默认使用 `openai`（幂等，不删除旧设置键）

  > 注意：`article_versions` 表的 `like_count` / `is_pinned` / `display_priority` 三列分别由第 3、4 步脚本补齐，必须在 `add_content_growth_features.sql`（第 2 步）之后执行，否则 `model/article.go` 的 `articleVersionSelectFields` 查询会因缺列报 `Unknown column`。
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

后台文章编辑页提供摘要/SEO 元数据生成、纠错和润色能力。AI 配置不再从环境变量或 YAML 读取，统一由管理员在后台 AI 设置页填写并保存到数据库 `site_settings`。可选择 `openai` 或 `anthropic` API 格式，填写服务基础地址和 API Key 后，先获取模型列表、选择模型并测试连接，最后保存并启用。当前 AI 调用均为非流式请求，配置项只保留首字等待和请求总超时。

`openai` 格式可填写兼容 OpenAI API 的基础地址，例如 `https://api.example.com/v1`，也可填写完整的 `/chat/completions` 地址；`anthropic` 格式使用 Messages API。出于 SSRF 防护，Base URL 必须解析到公网地址，不支持本机或内网模型服务。获取模型和测试模型接口都接受尚未保存的草稿连接配置，便于保存前验证。不要把真实 API Key 写入 README、Issue、提交记录或截图中。

新保存的 API Key 使用 `v3:` 密文，密钥由 `APP_AUTH_ACCESS_SECRET` 通过独立用途派生。`v2:` 密文使用的是已移除的独立加密密钥，升级后会在设置响应中标记 `apiKeyNeedsUpdate = true`，管理员必须重新填写 API Key；无版本前缀的旧密文仍兼容读取，建议尽快重新保存以迁移到 `v3:`。由于 `v3:` 密钥依赖 `APP_AUTH_ACCESS_SECRET`，轮换认证密钥时必须同时安排重新录入 AI API Key，否则原密文将不可解密。

### 可信反向代理配置

后端默认不信任客户端传入的 `X-Forwarded-*` / `X-Real-IP` 请求头，限流、操作日志、流量统计和 RSS/Sitemap 基础 URL 都会优先使用直连信息。Compose 通过专用子网给 Web 容器分配固定地址，API 默认仅信任该地址（`172.30.127.10/32`）。若 Web 前方还有可信 Nginx、1Panel 等外层代理，或 API 会直接接受其他可信代理转发，还需把这些代理的出口 CIDR 追加到真实 `.env`，后端才能从右向左跳过完整可信代理链；多个网段用逗号分隔：

```env
APP_TRUSTED_PROXY_CIDRS=172.30.127.10/32,10.0.0.0/24
```

Web Nginx 使用 `$binary_remote_addr` 做每 IP 限流，并仅接受 `WEB_TRUSTED_PROXY_CIDR` 指定的直接上游提供的 `X-Forwarded-For`。默认值 `172.30.127.1/32` 是 Compose `app` 网桥的默认网关，适用于宿主机上的 1Panel 反向代理转发到 `127.0.0.1:1270` 的部署方式。该值必须保持为实际直接上游的精确 CIDR，禁止填写 `0.0.0.0/0` 等宽泛公网网段，否则客户端可伪造来源 IP 绕过按 IP 限流。

不要在 API 可被公网或不可信客户端直连时配置过宽的网段。若默认 `172.30.127.0/24` 与宿主机或现有 Docker 网络冲突，请同时修改 `APP_DOCKER_SUBNET`、`APP_WEB_IPV4_ADDRESS`、`WEB_TRUSTED_PROXY_CIDR`（新子网的网桥网关 `/32`），并将 `APP_TRUSTED_PROXY_CIDRS` 中的 Web `/32` 更新为相同地址。

### 全文搜索配置

Docker Compose 已包含 Meilisearch 服务，但它位于 `search` profile，搜索默认关闭。需要启用全文搜索时，在真实 `.env` 中设置：

```env
APP_SEARCH_ENABLED=true
APP_MEILISEARCH_HOST=http://meilisearch:7700
APP_MEILISEARCH_API_KEY=replace-with-a-long-random-key
APP_MEILISEARCH_INDEX=articles
```

然后使用 profile 启动（也可设置环境变量 `COMPOSE_PROFILES=search` 后执行普通 Compose 命令）：

```bash
docker compose --profile search up -d --build
```

重新创建服务后，使用 `editor` 或 `admin` 登录后台，并调用 `POST /api/v1/admin/search/reindex` 全量重建文章索引。Meilisearch 初始化或运行中不可用时，API 不会因此退出，公开文章搜索会回退到 MySQL；后端会在后台重试索引初始化，恢复后重新启用 Meilisearch。

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

- Web：`127.0.0.1:1270 -> 8080`（唯一暴露到宿主机的端口）
- API：容器内部 `api:19000`（仅 Docker 内部网络可达）
- MySQL：容器内部 `mysql:3306`（仅 Docker 内部网络可达）
- Redis：容器内部 `redis:6379`（仅 Docker 内部网络可达）
- RabbitMQ：容器内部 `rabbitmq:5672`，管理端 `rabbitmq:15672`（均仅 Docker 内部网络可达）
- Meilisearch：容器内部 `meilisearch:7700`（仅 Docker 内部网络可达）

MySQL、Redis、RabbitMQ 默认由 `docker-compose.yml` 在内部网络启动，不会映射端口到宿主机。Web 通过 Nginx 反向代理访问 API。

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
4. 在 1Panel 项目目录创建 `.env`，内容可从 `.env.example` 复制后修改。
5. 启动 Compose 项目。
6. 在 1Panel 网站反向代理中配置：

   ```text
   http://127.0.0.1:1270
   ```

7. 绑定域名并开启 HTTPS。

1Panel 作为宿主机上的直接上游时，Web 容器默认只信任 Compose `app` 网桥网关 `172.30.127.1/32` 提供的 `X-Forwarded-For`。若修改 `APP_DOCKER_SUBNET`，必须同步把 `WEB_TRUSTED_PROXY_CIDR` 改为新子网网桥网关的 `/32`；禁止配置 `0.0.0.0/0` 等宽泛公网网段。

项目默认使用 Compose 内部 MySQL/Redis/RabbitMQ，不会在宿主机映射 `3306`/`6379`/`5672`/`15672` 端口，因此不会与 1Panel 已有中间件端口冲突。Go 后端通过 `.env` 中的 DSN/URL 连接这些内部服务。

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

Docker 部署会为本地中间件使用独立 volume：

- `notes-of-ashen_goblog_mysql_data`
- `notes-of-ashen_goblog_redis_data`
- `notes-of-ashen_goblog_rabbitmq_data`
- `notes-of-ashen_goblog_meili_data`

MySQL 的数据卷首次创建时会自动执行 [deploy/mysql/schema.sql](deploy/mysql/schema.sql) 初始化数据库和表结构。后续增量 SQL 需要按版本变更手动执行或通过迁移流程处理。

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

不要在容器内使用 `127.0.0.1:3306` 连接 MySQL，因为容器里的 `127.0.0.1` 只代表 API 容器自己。还要确认 `mysql` 容器健康，且首次初始化日志没有 SQL 执行失败。

查看日志：

```bash
docker compose logs -f api
```

### Redis 认证失败

确认 `.env` 中的 `APP_REDIS_ADDR=redis:6379`，且 `redis` 容器健康。

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
  - `GET/PUT /api/v1/admin/ai/settings`、`POST /api/v1/admin/ai/models`、`POST /api/v1/admin/ai/test`、`POST /api/v1/admin/search/reindex`

完整说明见 [docs/API.md](docs/API.md)。

## 维护建议

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
  pnpm build
  docker compose config --quiet
  ```
