# Notes of Ashen 项目审计报告

## 1. 审计结论

**审计日期：** 2026-07-27

**审计对象：** `Notes of Ashen` 前后端分离博客系统

**审计基线：** 提交 `eb0357e`（`fix: address project audit findings`）与本地 Docker Compose 实例，Web 入口为 `http://127.0.0.1:1270`

**总体结论：** 当前 Web、API、MySQL、Redis 以及已启用的 RabbitMQ、Meilisearch 服务均处于健康状态；迁移和配置检查成功，Go 与前端构建验证通过，未发现可直接确认的 P0 或 P1 问题。上一版报告中的 Markdown 导入权限、备份恢复清理、草稿隔离、Refresh Token 索引与清理、发布脚本默认 schema 兼容性检查以及 README 默认 profile 说明已完成修复。

当前最需要处理的是 Access Token 过期时的注销语义：注销接口被 Access Token 中间件拦截后，Refresh Token Cookie 可能仍然有效。生产环境还必须设置正式的 HTTPS `siteBaseUrl`，否则 RSS、Sitemap 和分享链接会回退到请求基址；此外，迁移器对高于当前镜像的数据库版本、`.env.example` 的可变镜像 tag 和 README 的 AVIF 说明仍有残余风险。

本报告仅记录审计结果，不修改业务代码、接口、配置、数据库或运行中的 Docker 实例。

## 2. 范围与方法

### 2.1 检查范围

- Go 后端：API、Handler、Logic、认证、权限、输入限制、数据访问、迁移、备份恢复、媒体和 AI 出站请求保护。
- React/TypeScript 前端：认证状态、注销流程、浏览器存储、文章编辑草稿、API 调用、类型和页面状态。
- 部署与文档：Docker Compose、Dockerfile、Nginx、迁移器、发布与备份脚本、`.env.example`、README 和 CI 配置。
- 运行态：Docker 服务和一次性任务、健康检查、公开接口、未授权后台接口、RSS/Sitemap、响应头和当前镜像 tag。

### 2.2 边界与限制

- 未读取或披露真实 `.env`、API Key、Token、密码、备份内容或其他敏感配置。
- 本次验证针对本机 HTTP 入口，不等同于生产 HTTPS、正式域名、可信代理 CIDR、外部依赖或异地故障恢复验证。
- 未执行真实登录、注销、备份恢复和跨账号隔离 E2E，以避免修改当前运行实例的数据和容器状态。
- 未执行真实生产回滚；回滚结论来自发布脚本和迁移器状态校验逻辑的静态审计。

## 3. 运行态验证证据

| 检查项 | 结果 |
| --- | --- |
| Docker 服务 | `web`、`api`、`mysql`、`redis`、`rabbitmq`、`meilisearch` 均为 healthy |
| 初始化任务 | `config-check`、`migrate` 均 Exit 0；迁移任务完成内置迁移检查 |
| 存活与就绪 | `GET /livez` 返回 204；`GET /healthz` 返回 200，`db`、`redis`、`schema` 均为 `up` |
| 公开与权限接口 | 首页、归档、搜索、登录页和站点公开接口正常加载；未授权 `GET /api/v1/admin/stats` 返回 401 |
| 安全响应头 | Web/API 响应包含 CSP、`X-Content-Type-Options: nosniff`、`X-Frame-Options: SAMEORIGIN` 和 `Referrer-Policy` |
| 本机 HTTP Cookie | 当前 API 运行态 `APP_AUTH_COOKIE_SECURE=false`，与本机 HTTP 入口匹配 |
| 镜像可追溯性 | `web`、`api`、`migrate`、`config-check` 使用 `v20260726-1646-61020d0`，当前实例不使用 `latest` |
| 站点基址 | `GET /api/v1/site/settings` 返回 `siteBaseUrl: ""`；RSS/Sitemap 当前使用 `http://127.0.0.1:1270` 绝对链接 |

### 3.1 已登录浏览器检查

- 首页顶部显示当前登录用户、`管理` 入口和 `退出` 操作；首页内容正常渲染。
- 管理总览正常加载，显示文章总数 5、已发布 5、草稿 0、用户 1、分类 3、标签 12。
- 站点设置页面正常加载，`站点地址` 输入为空，保存和取消按钮在未修改状态下均为禁用，印证了运行态 `siteBaseUrl` 为空的结论。
- 媒体页面正常加载，界面明确显示支持 JPG/JPEG、PNG、GIF、WebP、AVIF，并展示现有媒体条目。
- 系统工具页面的依赖健康检查显示整体正常，MySQL、Redis、Meilisearch、RabbitMQ、SMTP、媒体和 `backup_schema` 均为正常；备份导出/恢复按钮在未输入管理员密码时保持禁用。
- 本次浏览器检查期间未捕获 `error` 或 `warn` 级别控制台日志；未执行保存、退出、导入、恢复或删除操作。

## 4. 已修复问题

以下问题已在当前提交或前序修复中得到代码或文档层面的处理，不再作为当前未解决发现列出：

- **Markdown 导入权限与临时文件：** `internal/handler/article/article.go` 在 multipart 解析前校验内容管理权限，并在成功与失败路径清理 multipart 临时文件。
- **备份恢复权限与清理：** `internal/handler/backup/backup.go` 在解析前完成管理员鉴权，并清理恢复请求产生的 multipart 临时资源。
- **文章草稿隔离：** 前端文章编辑草稿键已包含用户维度，避免同一浏览器不同账户互相恢复本地草稿。
- **Refresh Token 生命周期基础能力：** 已增加过期/撤销记录的清理机制和索引，降低长期运行时令牌表持续增长的风险。
- **发布脚本回退保护：** `scripts/release.ps1` 默认比较目标镜像内置迁移版本和数据库版本，不兼容时拒绝代码回退；绕过检查需要显式危险参数。
- **Compose profile 文档：** README 已说明默认只启动 MySQL/Redis，可选 RabbitMQ、Meilisearch 需要对应 profile、功能开关和凭据。
- **前端注销失败策略：** 已禁止注销请求自动 refresh，并对 401、网络错误和 5xx 进行区分处理；相关前端测试已补充。

## 5. 当前审计发现

### P2：Access Token 过期时注销可能无法撤销 Refresh Token Cookie

**证据：**

- `internal/handler/routes.go` 将 `POST /api/v1/auth/logout` 注册为 `authRequired(authhandler.LogoutHandler(...))`。
- `internal/handler/auth/auth.go` 只有进入 `LogoutHandler` 后才解析 Refresh Token、调用注销逻辑并清除 Cookie。
- `frontend/src/utils/http.ts` 明确禁止注销请求自动触发 refresh；`frontend/src/store/logoutPolicy.ts` 将 401 视为可清理本地会话。

**影响：** 当 Access Token 已过期但 HttpOnly Refresh Token Cookie 仍有效时，注销请求会先被认证中间件返回 401，后端不会进入注销逻辑，也不会撤销 Refresh Token 或清除 Cookie。前端清空本地状态后，刷新页面仍可能通过 `/auth/refresh` 恢复原会话。

**建议：**

1. 让注销接口以 Refresh Token Cookie 为主要凭据，即使 Access Token 已过期也能撤销会话并清除 Cookie；或在前端注销前先刷新 Access Token，再调用注销接口。
2. 将“收到 401”与“Refresh Token 已失效”区分处理，避免仅凭 Access Token 认证失败就宣称服务端会话已撤销。
3. 增加真实会话测试，覆盖 Access Token 过期、Refresh Token 有效、Refresh Token 已失效、网络失败和服务端 5xx 场景。

**验收标准：** Access Token 过期时执行注销，Refresh Token 不得继续恢复会话；注销响应、Cookie 状态和前端认证状态保持一致。

### P2（生产配置）/P3（默认行为）：站点基址为空时生成本机绝对链接

**证据：**

- 当前 `GET /api/v1/site/settings` 返回 `siteBaseUrl: ""`。
- 当前 `GET /rss.xml` 和 `GET /sitemap.xml` 的绝对链接以 `http://127.0.0.1:1270` 开头。
- `internal/logic/site/feed.go` 在站点基址为空时回退到请求基址，并记录生产环境配置提示；前端文章分享链接也在空值时回退到当前页面 origin。

**影响：** 本地部署访问正常，但若生产环境沿用空值，搜索引擎、RSS 阅读器和分享链接会得到不可公开访问或不稳定的本机地址。

**建议：** 在生产站点设置中填写正式 `https://` 域名，将 RSS、Sitemap 和文章分享链接纳入发布验收；不要依赖请求 Host 推断规范站点地址。

**验收标准：** 生产环境所有公开绝对链接均使用正式 HTTPS 域名，并且站点设置、RSS、Sitemap 和分享链接保持一致。

### P2：迁移器不会拒绝数据库版本高于当前镜像的情况

**证据：**

- `scripts/release.ps1` 默认已增加目标镜像与数据库迁移版本兼容性检查，但允许通过 `-AllowIncompatibleSchema` 显式绕过。
- `internal/migration/migration.go` 的 `validateState` 主要遍历当前镜像已知迁移；当数据库包含当前旧镜像未内置的更高版本时，不会自动形成“未知高版本”错误。

**影响：** 正常发布脚本路径已有保护，但手工使用旧镜像、直接执行 Compose，或显式绕过回退检查时，可能出现应用代码低于数据库 schema 的状态。仅切换应用镜像不会回滚数据库，也不能保证旧代码兼容新结构。

**建议：** 让迁移状态检查显式拒绝数据库中高于当前镜像的迁移版本；保留显式危险绕过能力时，应强制输出“仅代码回退、数据库未回退”并要求已验证的备份恢复路径。补充旧镜像面对高版本 schema 的失败测试。

**验收标准：** 手工启动旧镜像或执行迁移检查时，只要数据库版本高于镜像内置版本就明确失败；代码回退不会被表述为数据库回滚。

### P3：`.env.example` 默认使用可变的 `IMAGE_TAG=latest`

**证据：** `.env.example` 仍设置 `IMAGE_TAG=latest`，且 `docker-compose.yml` 对应用镜像使用 `${IMAGE_TAG:-latest}`；当前实际运行实例使用不可变 tag，发布脚本也已拒绝正式发布使用 `latest`。

**影响：** 按示例直接部署可能无法稳定复现构建内容，且镜像更新、回滚和审计记录不具备明确版本边界。

**建议：** 将示例中的 tag 改为明确的占位版本，并在部署文档中要求生产环境使用不可变 tag；保留 Compose 默认值时应明确其仅适用于本地开发。

**验收标准：** 生产部署示例和检查流程不会默认解析到 `latest`，每次发布都能关联明确的镜像 tag 和提交。

### P3：README 功能概览遗漏 AVIF 媒体格式

**证据：** 媒体实现和测试已支持 AVIF（`internal/logic/media/media.go`、`internal/logic/media/media_test.go`），前端上传提示也列出 AVIF；README 的媒体功能概览仍只列出 JPEG/PNG/GIF/WebP。

**影响：** 文档用户会误以为 AVIF 不受支持，造成使用和排障信息不一致。

**建议：** 在 README 功能概览中补充 AVIF，并与前端上传提示和后端校验保持同一格式清单。

**验收标准：** README、前端提示、错误文案、后端校验和相关测试对支持的媒体格式描述一致。

## 6. 现有安全与可靠性控制

- API 与 Web 容器以非 root 用户运行；API 不映射宿主机端口，Web 仅绑定本机 `127.0.0.1:1270`。
- Access Token 仅保存在前端内存中；Refresh Token 使用 HttpOnly、SameSite Strict Cookie，并结合 Token version、用户状态和全局 Token cutoff 进行失效控制。
- 管理员和内容管理权限在 Logic 层再次校验；未鉴权后台接口已在运行态返回 401。
- Redis 对敏感接口采用 fail-closed 限流；默认不信任转发头，只有命中可信代理 CIDR 时才使用转发来源。
- AI 出站连接具备 URL、DNS 和建连 SSRF 校验；媒体上传具备内容检测、扩展名匹配、SHA-256 去重和原子发布。
- 请求日志不记录查询串、Cookie、Header 或正文；Nginx 配置 CSP、`nosniff`、SAMEORIGIN、恢复上传限制和媒体内部目录保护。
- 数据库迁移具备固定版本、checksum、advisory lock、事务记录和就绪检查；备份恢复具备管理员校验、大小限制、完整性校验、恢复锁和恢复后 Token 失效处理。
- 前端 API 路径和方法与后端路由/API 描述一致；Markdown 渲染未发现 `dangerouslySetInnerHTML`、`innerHTML` 或原始 HTML 注入路径。

## 7. 测试与构建验证

| 验证 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `frontend/pnpm test` | 通过，91/91 测试通过 |
| `frontend/pnpm build` | 通过，包含 lint、TypeScript、Vite 构建和 bundle size 检查 |
| `docker compose config --quiet` | 通过 |
| 本地运行态检查 | Docker 健康检查、`/healthz`、`/livez`、站点设置、RSS、Sitemap 和未授权后台接口均符合本报告记录 |

`pnpm build` 输出的 Vite chunk size warning 属于性能风险提示，不影响本次构建成功结论。项目已配置隔离 Docker 集成测试与 Chromium/WebKit E2E CI；本次未执行该套件，也未执行真实登录、注销、备份恢复和跨账号隔离验证。

## 8. 优先级行动清单

### 立即处理（P2）

- 修正 Access Token 过期时的注销路径，使 Refresh Token 能被撤销且 Cookie 能被清除；补充真实会话测试。
- 让迁移状态检查拒绝数据库高于当前镜像的版本，并明确区分代码回退与数据库回滚。
- 在生产环境设置正式 HTTPS `siteBaseUrl`，验证 RSS、Sitemap 和文章分享链接。

### 近期处理（P3）

- 将 `.env.example` 的生产部署示例改为不可变镜像 tag，并明确 `latest` 仅适用于本地开发（如仍保留 Compose fallback）。
- 在 README 媒体功能概览中补充 AVIF。
- 围绕注销、迁移兼容性和生产站点基址补充集成/E2E 验收。

## 9. 剩余风险与假设

- 当前健康状态证明本地实例可用，但不能代替版本升级、灾难恢复、数据库损坏、跨主机迁移和生产回滚演练。
- 当前本机 HTTP 使用 `APP_AUTH_COOKIE_SECURE=false` 是本地配置；生产 HTTPS 必须保持 `true`，并实际验证 Cookie 属性与会话续期。
- 当前本地站点基址为空是本机测试状态，不应复制到生产环境。
- 本地 Docker 健康状态不能证明外部邮件、AI、搜索服务、可信代理、异地备份存储或正式 TLS 终止层已验证。
- 当前未执行真实登录、注销、备份恢复和跨账号隔离 E2E，因此注销 Cookie 撤销链路和数据恢复流程仍需在隔离环境完成验收。
