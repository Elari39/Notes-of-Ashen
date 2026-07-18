# Notes of Ashen 项目问题评审报告

> 评审日期：2026-07-17  
> 评审范围：后端 Go/go-zero、前端 React/TypeScript/Vite、MySQL schema 与增量脚本、Docker Compose、Nginx、接口文档和现有测试。  
> 评审方式：静态阅读代码与配置、追踪主要调用链、执行现有测试、检查 Docker Compose 运行状态和容器内探针。  
> 评审结论：当前默认 Docker 部署可用，服务已运行在 `127.0.0.1:1270`；项目不存在已确认的 P0 级问题，但存在若干部署兼容、生产安全、资源控制和文档一致性问题，建议优先处理 P1/P2 项。

## 1. 执行摘要

### 1.0 本轮实施状态（2026-07-17）

- [x] P1-01：Compose 初始化数据库固定为 `notes_of_ashen`，已移除独立可变数据库名入口。
- [x] P1-02：公开 readiness 已校验当前运行所需的表和字段；缺失时返回 `503`。
- [x] P2-01：文章、分类和标签的文本字节上限、请求体上限与前端计数已补齐。
- [x] P2-02：生产构建不再生成 source map，Nginx 对 `.map` 返回 `404`。
- [x] P2-03：分页统一限制为 `page=1..1000`、`size=1..100`。
- [x] P2-04：`X-Request-Id` 已限制为最多 128 位的安全 ASCII 格式，并记录来源。
- [x] P2-05：备份恢复改为媒体卷内暂存、持久 journal、可回滚发布与启动恢复。
- [x] P2-06：Refresh Token Cookie 示例和后台用户管理路由文档已修正。
- [x] P3-01：真实 HTTP、Compose 和 Chromium E2E 测试编排已完成，core 与 extended 已在干净 Docker 环境中完整通过。
- [x] P3-02：已补 `type-check`、前端 CI，并于 2026-07-18 完成 Tailwind CSS 4 升级。

### 1.1 总体评价

项目已经具备较完整的个人博客产品形态和生产部署基础：

- 后端采用分层结构，Handler、Logic、Model、ServiceContext 边界清晰。
- 认证、角色权限、访问令牌失效、Refresh Token 轮换、验证码、敏感接口限流等安全控制较完整。
- 媒体上传包含 MIME、扩展名、图片格式、尺寸、SHA-256 和原子落盘校验。
- AI 配置使用数据库保存，API Key 使用由认证密钥派生的独立用途密钥加密，且对 AI 出站请求做了公网地址和重定向限制。
- 加密备份包含口令保护、归档路径校验、大小上限、校验和、恢复租约和数据关联校验。
- Compose 只把 Web 暴露到宿主机 `127.0.0.1:1270`，MySQL、Redis、RabbitMQ、Meilisearch 只在 Docker 内部网络暴露。

主要问题集中在“配置可变性和初始化脚本不一致”“就绪探针覆盖不足”“用户输入大小控制不足”“构建产物泄露源码映射”“深分页资源保护不足”和“文档已落后于实现”等方面。

### 1.2 问题统计

| 等级 | 数量 | 结论 |
| --- | ---: | --- |
| P0 | 0 | 未发现立即导致权限绕过、密钥泄露或生产无法启动的确定性问题 |
| P1 | 2 | 配置数据库名时可能初始化错误数据库；旧数据库缺迁移时就绪探针仍判定健康 |
| P2 | 6 | 资源限制、源码映射、深分页、Request ID、备份媒体一致性、文档漂移 |
| P3 | 2 | 测试覆盖和维护性改进 |

等级定义遵循项目 `AGENTS.md`：P0 为安全、数据破坏、权限绕过或生产无法启动；P1 为核心流程或部署可用性问题；P2 为明显的安全/体验/一致性/可测试性问题；P3 为性能和维护性改进。

## 2. 项目结构与运行链路

### 2.1 主要组件

| 组件 | 实现 | 运行方式 |
| --- | --- | --- |
| Web | React 18、TypeScript、Vite、Tailwind CSS 4、Zustand、Axios | Nginx 非 root 镜像，容器端口 8080 |
| API | Go 1.25、go-zero REST、JWT、bcrypt | 非 root Alpine 容器，容器端口 19000 |
| 数据库 | MySQL 8.4 | 持久化 volume，首次创建时挂载 `deploy/mysql/schema.sql` |
| 缓存/限流 | Redis 7.4 Alpine | AOF 持久化，内部网络访问 |
| 异步审计 | RabbitMQ 4 Management | 事件发布失败时降级同步写库 |
| 全文搜索 | Meilisearch 1.13 | 可选，故障时回退 MySQL |
| 媒体 | API 可写、Web 只读共享 volume | `/data/media` 与 `/usr/share/nginx/media` |

### 2.2 请求链路

```text
浏览器
  -> 127.0.0.1:1270
  -> Web Nginx
      -> 静态 React 页面
      -> /api/* -> api:19000
      -> /media/* -> 共享媒体卷
  -> API
      -> MySQL：业务数据
      -> Redis：认证缓存、Refresh Token 辅助缓存、验证码、限流、流量缓存
      -> RabbitMQ：异步操作日志
      -> Meilisearch：可选全文搜索
```

## 3. 问题清单

### [x] P1-01：可配置数据库名与初始化 SQL 固定数据库名不一致

### 位置

- `docker-compose.yml:124`：`MYSQL_DATABASE` 使用 `APP_MYSQL_DATABASE`，支持自定义数据库名。
- `docker-compose.yml:133`：首次初始化挂载 `deploy/mysql/schema.sql`。
- `deploy/mysql/schema.sql:3`：固定执行 `USE notes_of_ashen;`。

### 证据

Compose 允许：

```yaml
MYSQL_DATABASE: "${APP_MYSQL_DATABASE:-notes_of_ashen}"
```

但 schema 文件固定：

```sql
USE notes_of_ashen;
```

当用户把 `APP_MYSQL_DATABASE` 改为其他值时，MySQL 入口会创建自定义数据库，但初始化脚本仍把表建到 `notes_of_ashen`。API 若通过 DSN 连接自定义数据库，可能启动成功完成基础连接检查，但随后出现表不存在、配置不存在或功能请求失败。

### 影响

- 自定义数据库名的 Docker 部署无法可靠初始化。
- 错误发生在运行期，启动日志不一定直接说明“schema 初始化到了另一个数据库”。
- 数据库可能同时存在两个名字相近但结构不一致的库，增加运维误判和数据操作风险。

### 建议

优先选择一种单一事实源：

1. 固定 Compose 数据库名为 `notes_of_ashen`，移除可变数据库名；或
2. 将 schema 改为不依赖固定 `USE`，由入口数据库变量决定目标库，并在初始化前验证 DSN、`MYSQL_DATABASE` 和 schema 目标一致。

同时增加一次性初始化测试：自定义数据库名启动全新 MySQL volume，验证核心表、默认设置和 API 首次健康检查。

### 优先级

P1。默认值不会触发，但项目明确暴露了该可配置项，因此属于已确认的部署兼容缺陷。

### [x] P1-02：Compose 就绪探针未验证完整数据库 schema 和迁移状态

### 位置

- `docker-compose.yml:98-113`：API 依赖 MySQL/Redis healthy 后启动。
- `docker-compose.yml:136-143`：MySQL healthcheck 只执行 `mysqladmin ping`。
- `internal/logic/site/health.go:29-43`：`/healthz` 只探测 DB Ping 和 Redis Ping。
- `internal/logic/system/health.go:92`：`backup_schema` 只在管理员系统健康接口中检查。
- `internal/handler/routes.go:52`、`107`：公开 `/healthz` 与管理员健康接口是两条不同路径。

### 证据

当前 API `/healthz` 返回结构仅包含：

```json
{"status":"ok","checks":{"db":{"status":"up"},"redis":{"status":"up"}}}
```

旧 MySQL volume 如果没有执行 `deploy/mysql` 下的新增迁移，MySQL Ping 仍然成功，API 也会被 Docker 判定为 healthy；但媒体、内容分析、备份、AI 设置或新字段相关接口可能在实际调用时失败。项目文档已经说明增量 SQL 需要手动执行，但编排层没有把这一状态显式纳入 readiness。

### 影响

- Docker/1Panel 可能在 schema 不完整时把 API 提供给 Web。
- 发布检查只看到 healthy，不能说明当前数据库版本满足当前代码要求。
- 迁移缺失问题会被误判为具体接口或权限问题。

### 建议

- 将关键 schema 版本或必要表检查加入内部 readiness 探针，至少覆盖当前运行代码必需的表和字段。
- 将“数据库连接正常”和“应用 schema 就绪”区分为 liveness/readiness 两类状态。
- 若不希望启动时阻断旧库，至少在 `/healthz` 或启动日志中明确报告迁移缺口，并让 Compose healthcheck 根据 readiness 失败。
- 后续引入显式 migration version 表，避免依赖人工按文件名排序执行。

### 优先级

P1。新库默认部署正常，但存量数据卷升级是实际生产场景，缺迁移会导致功能不可用。

### [x] P2-01：文章正文、摘要和分类描述缺少明确输入大小上限

### 位置

- `internal/logic/article/article.go:592-618`：`validateArticle` 只检查正文非空，没有正文最大长度；摘要也没有长度校验。
- `internal/logic/category/category.go:104-112`：分类校验只限制名称和 slug。
- `internal/logic/tag/tag.go:104-112`：标签校验只限制名称和 slug。
- `deploy/nginx/default.conf:74`：普通客户端请求体上限为 20 MiB。
- `deploy/mysql/schema.sql:80-83`：正文为 `MEDIUMTEXT`，摘要和描述为 `TEXT`。

### 证据

文章校验包含：

```go
if err := validator.Required(req.Content, "content"); err != nil {
    return err
}
```

但没有与数据库字段、前端编辑器或 Nginx 请求体对应的业务长度上限。文章创建/更新请求可以携带接近 Web 层 20 MiB 的大正文；摘要、分类描述、标签描述也没有对应的业务上限。

### 影响

- 编辑器用户或被盗的 editor 账号可以持续提交大请求，增加 API 内存、JSON 解码、MySQL 写入和缓存压力。
- `MEDIUMTEXT` 的数据库上限与 Nginx 20 MiB 请求体上限并不一致，可能产生数据库错误。
- 大摘要和描述会进入列表、缓存、搜索索引或前台响应，放大一次输入的影响范围。

### 建议

- 在 Logic 层对正文、摘要、分类/标签描述建立明确的 UTF-8 字符或字节上限。
- 在 Handler 层使用 `http.MaxBytesReader` 或统一请求体策略，避免 JSON 解码前无限接收。
- 前端同步展示剩余字数，并为超过限制的场景提供明确错误。
- 增加边界测试：空值、刚好达到上限、超过上限、中文多字节字符和超大请求体。

### 优先级

P2。当前有 Nginx 20 MiB 上限和权限控制，风险主要是资源消耗和数据质量，不属于匿名直接利用。

### [x] P2-02：生产前端发布了可访问的 JavaScript source map

### 位置

- `frontend/vite.config.ts:21`：`build.sourcemap = 'hidden'`。
- `Dockerfile.web:13`、`27`：构建后把完整 `dist` 目录复制到 Nginx。
- `deploy/nginx/default.conf:128-136`：通用静态路径会直接尝试读取存在的文件，没有拒绝 `.map`。

### 证据

`hidden` 只会隐藏 JS 文件中的 `sourceMappingURL` 注释，不会阻止 `.map` 文件生成。当前运行中的 Web 容器已经存在多个 map 文件，例如：

```text
/usr/share/nginx/html/assets/index-D3XclD0D.js.map
/usr/share/nginx/html/assets/ArticleEditor-CvKo-hKw.js.map
/usr/share/nginx/html/assets/AISettings-CBI2xxZW.js.map
```

### 影响

- 访问者可以下载源码映射，查看模块名、源码路径、实现细节和可能存在的注释。
- 管理后台的前端实现和内部业务流程更容易被逆向分析。
- 如果未来错误地把非公开配置或调试信息打入前端，source map 会扩大泄露面。

### 建议

生产环境优先设置 `sourcemap: false`。如果必须保留构建映射，应将 map 文件上传到受控错误监控系统，不复制到 Nginx 公共目录；也可以在 Nginx 中明确拒绝 `\.map` 请求作为兜底。

### 优先级

P2。不会直接泄露后端密钥，但属于可避免的生产源码暴露。

### [x] P2-03：分页只限制 size，不限制 page，存在深分页资源消耗

### 位置

- `internal/httphelper/helper.go:50-63`：解析 HTTP 查询参数时只把 size 限制到 100。
- `internal/logicutil/common.go:16-27`：业务层同样没有 page 上限。
- 多个 Model 使用 `(page - 1) * size` 计算 offset。

### 证据

```go
if size > 100 {
    size = 100
}
return page, size
```

`page` 只做了小于 1 的归一化，没有最大值、最大 offset 或游标分页策略。公开文章、分类、标签、搜索建议以外的部分列表接口没有统一 IP 限流，攻击者可以反复请求非常大的 page 值。

### 影响

- MySQL 可能执行高成本的深 offset 扫描。
- 极大 page 在乘法计算时可能产生整数溢出或非法 offset，造成 500。
- 在数据规模增长后，简单的列表接口可能成为低成本资源消耗入口。

### 建议

- 为 page 设置合理上限，超出时返回 400 或归一化到最后一页。
- 对公共列表优先考虑基于 ID/时间的 cursor pagination。
- 为高流量列表增加缓存、索引检查和慢查询监控。
- 增加 page=1、极大 page、极大 size、负数和非法字符测试。

### 优先级

P2。当前数据规模较小时影响有限，但属于可预防的性能和稳定性问题。

### [x] P2-04：`X-Request-Id` 可由客户端任意注入且没有长度/字符限制

### 位置

- `internal/middleware/requestid.go:18-23`。
- `internal/middleware/accesslog.go:43-57`。

### 证据

```go
id := r.Header.Get("X-Request-Id")
if id == "" {
    id = newRequestID()
}
w.Header().Set("X-Request-Id", id)
```

客户端提供的值会被直接写回响应并进入访问日志上下文。虽然当前日志框架会对 JSON 内容进行编码，降低换行注入的实际影响，但代码本身没有限制长度、控制字符或非法格式。

### 影响

- 超长 Request ID 会放大响应头和日志体积。
- 控制字符、换行或特殊格式会增加日志检索和审计风险。
- 外部调用方可以伪造链路 ID，影响日志关联可信度。

### 建议

- 只接受固定长度的 ASCII 字母、数字、短横线或下划线。
- 超长、包含控制字符或格式非法时重新生成服务端 ID。
- 保留一个内部字段标识“客户端提供”还是“服务端生成”，避免审计歧义。
- 增加控制字符和超长值测试。

### 优先级

P2。当前不是认证绕过，但属于日志和运维安全边界缺口。

### [x] P2-05：备份恢复的媒体落盘与数据库事务不是完全原子操作

### 位置

- `internal/logic/backup/backup.go:252-260`：先恢复媒体文件，再执行数据库 `RestoreBackup`。
- `internal/logic/backup/backup.go:405-433`：媒体直接写入正式媒体根目录。
- `model/backup.go:205-337`：数据库替换在单独的 SQL 事务中完成。

### 证据

恢复流程顺序为：

1. 解密并校验归档。
2. 将归档媒体写入正式媒体目录。
3. 执行数据库整站替换事务。
4. 事务成功后再清理旧媒体文件。

如果数据库事务失败，事务会回滚，但已经写入的媒体文件不会自动回滚；如果媒体清理失败，数据库已经切换完成但旧文件会残留。

### 影响

- 数据库恢复失败时可能留下孤立媒体文件，持续占用磁盘。
- 恢复过程中断或进程异常退出时，媒体目录和数据库状态可能暂时不一致。
- 长期多次恢复后，残留文件会增加备份和磁盘运维成本。

### 现有缓解措施

- 媒体 storage key 为内容 SHA-256，原子写入可以避免部分文件覆盖。
- 恢复有本地和 Redis 分布式租约，避免多实例并发恢复。
- 数据库替换使用 Serializable 事务，归档在落库前完整校验。

### 建议

- 先写入按恢复任务隔离的 staging 目录。
- 数据库事务成功后，再通过目录 rename 或受控复制将 staging 切换为正式目录。
- 失败时删除 staging；进程重启时清理超时 staging 目录。
- 为“媒体写入成功、数据库失败”和“数据库成功、清理失败”增加故障注入测试。

### 优先级

P2。当前更可能造成磁盘残留，不会直接覆盖不同 hash 的媒体内容。

### [x] P2-06：README 和 API 调用示例与当前实现存在文档漂移

### 位置

- `docs/API.md:199` 已说明 Refresh Token 响应字段会置空。
- `docs/API.md:990` 仍然把 `$register.data.refreshToken` 赋给 PowerShell 变量。
- `README.md:631` 使用 `GET/PUT /api/v1/admin/users`，而实际接口是 `GET /api/v1/admin/users` 和 `PATCH /api/v1/admin/users/:id/status|role`。

### 影响

- 直接复制 PowerShell 示例的 API 使用者会拿到空 Refresh Token，误以为登录或刷新流程异常。
- 后台用户管理接口的 README 路径和方法描述不准确。
- 文档与前端当前基于 HttpOnly Cookie 的认证策略不完全一致。

### 建议

- 删除示例中对响应体 Refresh Token 的依赖，改为说明 Cookie 需要客户端 Cookie 容器保存；如果是无 Cookie API 客户端，明确要求使用兼容 body 方式并说明当前响应不回显新 token。
- 统一 README、`docs/API.md`、`api/notes-of-ashen.api`、前端 API 封装和实际路由的自动检查。
- 将路由清单作为 CI 文档漂移检查输入，而不是依赖人工维护。

### 优先级

P2。不会影响前端同源浏览器主流程，但会影响 API 使用者和运维排查。

### [x] P3-01：真实 HTTP/数据库/Compose 集成覆盖已补齐

### 现状证据

`go test -cover ./...` 成功，但覆盖分布不均：

| 包 | 覆盖率 |
| --- | ---: |
| `internal/contentstats` | 100.0% |
| `internal/aiclient` | 77.8% |
| `internal/config` | 70.7% |
| `internal/logic/auth` | 10.0% |
| `internal/logic/article` | 20.5% |
| `internal/logic/backup` | 27.6% |
| `internal/logic/media` | 28.8% |
| `model` | 20.2% |
| 多个 Handler/response/types 包 | 0.0% |

当前测试大量使用单元测试和 sqlmock，没有覆盖完整的“HTTP 路由 -> Middleware -> Logic -> MySQL/Redis”链路，也没有前端浏览器端到端测试。

### 建议

- 增加 API 集成测试：注册、登录、刷新、权限、文章发布、媒体上传、备份恢复。
- 使用临时 MySQL/Redis 或 Docker Compose 测试 profile 验证真实 SQL、迁移和健康检查。
- 增加前端关键路径 E2E：首次注册、登录刷新、文章编辑、媒体插入、后台权限。
- 对备份恢复、并发注册、Refresh Token 旋转和限流故障增加故障注入测试。

### 当前实施

- 新增 `scripts/test-integration.ps1 -Suite core|extended`，每个 HTTP、浏览器和扩展阶段均使用新的随机 Compose 项目、卷、凭据、网络与 loopback 端口，且不读取开发 `.env`。
- 新增测试 Compose 覆盖，仅启动 Web、API、MySQL 与 Redis；测试环境关闭邮件、RabbitMQ、Meilisearch 与 Prerender，并只在该环境关闭 Refresh Cookie 的 Secure 属性。
- `core` 覆盖真实 HTTP/数据库链路和 Chromium E2E；`extended` 在独立生命周期中覆盖并发、Redis 故障及恢复失败注入。详细命令见 `docs/TESTING.md`。
- GitHub Actions 在 push/PR 运行 core，并在每日上海时间 02:00 与手动触发时运行 extended；失败时保存 Compose 与 Playwright 产物。

`core` 与 `extended` 已在干净 Docker 环境中完整通过；测试项目、卷、镜像和临时凭据均已清理。

### 优先级

P3，若项目进入多人协作或频繁发布阶段，建议提升到发布门禁项目。

### [x] P3-02：前端构建工具链和项目约定已统一

### 原现状

- `frontend/package.json` 实际使用 `tailwindcss ^3.3.5`。
- 根目录说明与项目约定对 Tailwind 版本的描述并不统一。
- `package.json` 没有独立的 `type-check` 脚本，类型检查由 `pnpm build` 内部的 `tsc` 完成。

### 影响

- 新贡献者可能按 Tailwind 4 语法修改，但当前构建链仍是 Tailwind 3。
- CI 若只运行 `pnpm lint`，不会执行 TypeScript 类型检查和 Vite 构建。

### 当前实施

- 前端已升级到 Tailwind CSS 4，并使用官方 `@tailwindcss/vite` 插件接入 Vite。
- 保留现有 JavaScript 主题配置，通过 `@config` 兼容加载颜色、字体、圆角、阴影和 Typography 插件。
- 已移除 Tailwind 3 使用的 PostCSS/Autoprefixer 配置，并迁移旧透明度、渐变、outline 与 blur 工具类。
- 已增加独立 `pnpm type-check`，CI 固定执行 lint、类型检查、构建和前端测试。

### 优先级

P3，属于工程一致性和维护成本问题。

## 4. 已确认的安全与工程优点

### 4.1 认证与权限

- Access Token 使用 HS256 JWT，并通过用户状态和角色再次查询确认，不只信任 Token 内角色。
- Refresh Token 使用随机值、数据库 hash 保存、轮换撤销和 HttpOnly Cookie。
- 密码修改、找回密码和管理员状态变更会撤销或驱逐相关会话缓存。
- 管理员、内容管理员和文章作者权限在 Logic 层再次校验，避免只依赖路由分组。
- 最后一个 active admin 不能被禁用或降级，管理员不能自行降级或禁用自己。
- 登录、注册、验证码、密码重置等敏感接口使用 Redis 限流，关键限流器 Redis 故障时 fail-closed。

### 4.2 SSRF、文件和内容安全

- 头像、封面、项目链接限制为 HTTP/HTTPS，并拒绝本机、私网、链路本地、文档和保留地址。
- AI Base URL 在建立新连接时重新解析并拒绝受限 IP，不使用环境代理，不跟随重定向。
- 媒体上传校验内容类型、扩展名、图片解码和尺寸，文件名不会决定存储路径。
- 媒体文件以 SHA-256 storage key 保存，并使用临时文件 + rename 原子写入。
- Markdown 使用 `react-markdown` 默认安全渲染链，未发现启用原始 HTML 的 `rehype-raw`。
- 搜索高亮结果在后端做 HTML 转义，只保留受控 `<mark>` 标签。

### 4.3 备份与恢复

- 备份使用 age scrypt 口令加密。
- 归档包含路径安全检查、entry 数量限制、展开大小限制、manifest、data 和媒体 SHA-256 校验。
- 备份不包含 AI API Key、Refresh Token、日志、流量明细和访客去重 hash。
- 恢复要求管理员权限、当前密码、口令和 `REPLACE` 确认。
- 恢复使用本地原子状态和 Redis 分布式租约，能够降低并发恢复风险。

### 4.4 Docker 与 Nginx

- Web、API 镜像均以非 root 用户运行。
- API、数据库、Redis、RabbitMQ、Meilisearch 没有映射到宿主机公网端口。
- Web 仅绑定宿主机 loopback：`127.0.0.1:${WEB_PORT:-1270}:8080`。
- Nginx 配置了 SPA 路由回退、备份大请求体专用上限、API 基础限流、安全响应头和媒体只读挂载。
- Compose 服务配置了 restart、healthcheck、资源限制和日志滚动参数。

## 5. 接口与数据一致性审计结论

### 5.1 当前未发现的高风险漂移

- 后端统一响应基本遵循 `{code, message, data}` 约定。
- 前端 Axios 已使用 `withCredentials`，与 Refresh Token HttpOnly Cookie 机制匹配。
- `internal/types`、前端 `src/types`、主要 API 封装和页面字段总体一致。
- 可选布尔字段在站点设置更新逻辑中使用指针语义，显式 `false` 可以关闭开关。
- 搜索故障时可以回退 MySQL，不会因为 Meilisearch 不可用导致 API 进程退出。

### 5.2 需要持续保持的联动约束

- 修改 API 请求字段时必须同步 `api/notes-of-ashen.api`、`internal/types`、前端 `src/api`、前端 `src/types`、页面和 `docs/API.md`。
- 修改数据库字段时必须同步 schema、增量脚本、Model 查询字段、备份导入导出和健康检查。
- 修改 `APP_AUTH_ACCESS_SECRET` 时必须安排已保存 AI API Key 重新录入。
- 修改 Docker 子网时必须同时调整 Web 固定 IP、Web 可信代理 CIDR 和 API 可信代理 CIDR。

## 6. 修复优先级路线图

### 第一阶段：发布前修复

1. 统一 `APP_MYSQL_DATABASE` 与 schema 初始化目标，补充自定义数据库名测试。
2. 增强 readiness，显式检查当前代码需要的数据库 schema/migration 状态。
3. 关闭生产公开 source map。
4. 为文章正文、摘要、分类描述、标签描述增加业务长度上限。

### 第二阶段：稳定性与安全加固

1. 增加 page 上限或引入 cursor pagination。
2. 校验和规范化 `X-Request-Id`。
3. 将备份媒体恢复改为 staging 目录切换。
4. 增加故障注入和真实中间件集成测试。

### 第三阶段：工程维护

1. 修正文档中的 Refresh Token 和后台用户管理接口示例。
2. 增加 `pnpm type-check` 脚本和 CI 门禁。
3. Tailwind CSS 4 升级及版本说明统一已完成。

## 7. 验证记录

### 7.1 后端

执行：

```text
go test ./...
go test -cover ./...
```

结果：通过。

未能在当前 Windows 沙箱中完成 `go vet ./...`，命令启动阶段反复出现：

```text
CreateProcessAsUserW failed: 1312
```

该错误属于执行环境会话创建失败，本报告不将其判定为代码静态检查失败。

### 7.2 前端

目标命令：

```text
pnpm lint
pnpm test
pnpm build
```

当前沙箱在启动 `pnpm` 子进程、ESLint 和 Node test runner 时同样出现 `CreateProcessAsUserW failed: 1312`，因此未声称前端检查通过。Docker Web 镜像已经能够使用当前前端源码完成构建并提供首页静态资源。

### 7.3 Docker

执行或检查：

```text
docker compose config --quiet
docker compose ps
docker compose exec -T web wget -qO- http://127.0.0.1:8080/
docker compose exec -T api wget -qO- http://127.0.0.1:19000/healthz
Test-NetConnection 127.0.0.1 -Port 1270
```

结果：

- Compose 配置校验通过。
- `web`、`api`、`mysql`、`redis`、`rabbitmq`、`meilisearch` 均为 healthy。
- Web 容器首页返回 HTML。
- API 容器 `/healthz` 返回 DB/Redis 均为 up。
- 宿主机 `127.0.0.1:1270` TCP 连通。

### 7.4 密钥与敏感文件

- 未读取或输出真实 `.env` 内容。
- `.env` 被 `.gitignore` 和 Docker ignore 排除。
- 未发现代码中硬编码 API Key、Token 或密码。

## 8. Docker 部署状态

本次部署目标为：

```text
http://127.0.0.1:1270
```

Compose 对外只暴露 Web 端口；API 和中间件保持 Docker 内部网络访问。部署完成后建议继续使用以下命令进行日常检查：

```powershell
docker compose ps
docker compose logs --tail 100 api web
Test-NetConnection 127.0.0.1 -Port 1270
```

生产或 1Panel 场景还应在外层 HTTPS 代理中确认：

- `APP_AUTH_COOKIE_SECURE=true`。
- `APP_TRUSTED_PROXY_CIDRS` 只包含实际可信代理出口网段。
- 若修改 Compose 子网，同步修改可信代理 CIDR 和 Web 固定地址。
- 旧数据库在上线前完成所有增量 SQL，并先执行数据库和媒体卷备份。

## 9. 最终结论

项目当前可以作为个人博客系统运行，默认 Docker 部署链路已经在 `127.0.0.1:1270` 正常工作。安全和业务保护实现明显高于基础 CRUD 项目，但数据库初始化与迁移管理、生产构建产物、输入资源限制和测试门禁仍需要加强。

建议先修复 P1-01、P1-02、P2-01 和 P2-02，再进入正式公网或多人协作发布；其余 P2/P3 项可以纳入后续迭代，但应在数据量和用户量增长前完成分页与集成测试建设。
