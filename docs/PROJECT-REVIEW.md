# Notes of Ashen 项目审计报告

## 1. 审计结论

**审计日期：** 2026-07-26

**审计对象：** `Notes of Ashen` 前后端分离博客系统

**审计基线：** 当前工作区代码（`c715c1a`）与本地 Docker Compose 实例，Web 入口为 `http://127.0.0.1:1270`

**总体结论：** 当前实例的 Web、API、数据库、缓存和已启用的可选服务均健康，迁移与配置检查已成功完成；认证、权限、媒体、备份恢复和代理边界具备较完整的现有保护。未发现可直接确认的 P0 问题。

上一版报告中的本机 HTTP Secure Cookie 风险和应用镜像 `latest` 风险不适用于当前运行实例：运行态 `APP_AUTH_COOKIE_SECURE=false`，应用镜像使用不可变版本 tag。当前最需要优先处理的是发布回滚未校验数据库 schema 向后兼容性；此外仍存在 Markdown 导入资源消耗、会话退出一致性、跨账号草稿隔离、站点基址、部署文档和 Refresh Token 数据生命周期问题。

本报告仅记录审计结果，不修改业务代码、接口、配置或部署文件。

## 2. 范围与方法

### 2.1 检查范围

- Go 后端：API 描述、Handler、Logic、认证、权限、输入限制、数据访问、迁移、备份恢复和 AI 出站请求保护。
- React/TypeScript 前端：认证状态、退出流程、浏览器存储、文章编辑草稿、API 调用、类型与页面错误状态。
- 部署与文档：Docker Compose、Dockerfile、Nginx、迁移器、发布与备份脚本、`.env.example`、README、CI 配置。
- 运行态：Docker 服务与一次性任务状态、健康检查、公开接口、未授权后台接口、RSS/Sitemap、响应头和本地备份产物。

### 2.2 边界与限制

- 未读取或披露真实 `.env`、API Key、Token、密码、备份内容或其他敏感配置。唯一读取的运行态环境值为非敏感布尔项 `APP_AUTH_COOKIE_SECURE`。
- 本次验证针对本机 HTTP 入口，不等同于生产 HTTPS、正式域名、可信代理 CIDR、外部对象存储或异地故障恢复验证。
- 未执行真实登录/注销、恢复导入、完整备份导出或隔离 Docker E2E，以避免改变当前运行实例的数据和容器状态。
- 未执行真实生产回滚；回滚结论来自发布脚本与迁移器状态校验逻辑的静态审计。

## 3. 运行态验证证据

| 检查项 | 结果 |
| --- | --- |
| Docker 服务 | `web`、`api`、`mysql`、`redis`、已启用的 `rabbitmq`、`meilisearch` 均为 healthy |
| 初始化任务 | `config-check`、`migrate` 均以 Exit 0 结束；迁移日志确认所有内置迁移已应用 |
| 存活与就绪 | `GET /livez` 返回 204；`GET /healthz` 返回 200，`db`、`redis`、`schema` 均为 `up` |
| 公开与权限接口 | 首页、文章列表、站点设置、RSS、Sitemap 均可访问；未鉴权 `GET /api/v1/admin/stats` 返回 401 |
| 安全响应头 | Web 与 API 响应包含 CSP、`X-Content-Type-Options: nosniff`、`X-Frame-Options: SAMEORIGIN` 和 `Referrer-Policy` |
| 本机 HTTP Cookie | API 容器的 `APP_AUTH_COOKIE_SECURE=false`，与 `http://127.0.0.1:1270` 本机访问方式匹配 |
| 镜像可追溯性 | `web`、`api`、`migrate`、`config-check` 使用 `v20260726-1646-61020d0`；当前运行实例不使用 `latest` |
| 本地备份产物 | `backups/20260726-165238/` 包含 MySQL 压缩导出、媒体归档和 `SHA256SUMS.txt` |
| 站点基址 | `/api/v1/site/settings` 返回 `siteBaseUrl: ""`；RSS 与 Sitemap 当前生成 `http://127.0.0.1:1270` 绝对链接 |

## 4. 审计发现

### P1：发布脚本的“回滚”没有阻止数据库 schema 向后不兼容

**证据：**

- `scripts/release.ps1:102-109` 的回滚分支仅切换 `IMAGE_TAG`、执行 `docker compose up -d` 并记录成功，不校验目标镜像与当前数据库 schema 的兼容性。
- `internal/migration/migration.go:328-352` 的 `validateState` 仅遍历当前镜像内置的迁移版本；数据库已存在、但旧镜像中没有的更高迁移版本不会形成错误。
- 迁移链含有前向破坏性或难以逆转的变更，例如删除旧流量表、清理孤儿文章版本和添加关系完整性约束（`deploy/mysql/migrations/000013_drop_traffic_geo.sql`、`000023_cleanup_orphan_article_versions.sql`、`000024_add_relationship_integrity.sql`）。

**影响：** 故障时脚本可能显示“已回滚”，但旧应用实际运行在更新后的数据库 schema 上。遇到不向后兼容的迁移时，应用可能不可用，且仅回退应用镜像无法恢复数据或 schema。

**建议：**

1. 将当前操作明确命名为“代码回退”，避免暗示数据库已完成回滚。
2. 在切换镜像前读取目标镜像的最高迁移版本并与数据库已应用版本比较；数据库版本高于目标镜像时阻止回退。
3. 仅允许通过显式危险确认参数绕过该检查，并要求已验证的备份恢复路径。
4. 增加“数据库版本高于目标镜像”时回滚被拒绝的自动化测试。

**验收标准：** 目标镜像落后于数据库 schema 时，发布脚本不会宣称回滚成功；运维人员得到明确的备份恢复或兼容版本处理指引。

### P2：低权限用户可在权限拒绝前触发 Markdown multipart 解析，并可能遗留临时文件

**证据：**

- `internal/handler/routes.go:82` 仅为 `POST /api/v1/articles/import` 配置登录鉴权。
- `internal/handler/article/article.go:296-320` 在角色校验前执行 `ParseMultipartForm`、读取文件和 Markdown 内容，且成功解析后未调用 `r.MultipartForm.RemoveAll()`。
- 真正的内容管理权限检查位于 `internal/logic/article/markdown.go:53-56`；媒体与备份上传 handler 已分别在解析后清理 multipart 临时文件，备份恢复还在解析前完成管理员鉴权（`internal/handler/media/media.go:45-50`、`internal/handler/backup/backup.go:58-75`）。

**影响：** 普通已登录用户可反复提交接近上限的 Markdown multipart 请求，在收到 403 前占用请求解析、内存和可能的临时磁盘；当 multipart 数据溢出到临时文件时，未主动清理会增加临时盘累积风险。

**建议：**

1. 在 `ImportMarkdownHandler` 的开头先执行 `authutil.RequireContentManager(r.Context())`。
2. `ParseMultipartForm` 成功后，在 `r.MultipartForm != nil` 时立即 `defer r.MultipartForm.RemoveAll()`。
3. 添加低权限请求不会解析正文、multipart 临时文件会被清理的 handler 回归测试。

**验收标准：** 普通用户提交导入请求时不进入 multipart 解析；成功与失败导入均不遗留临时 multipart 文件。

### P2：注销失败后前端仍显示已退出，刷新页面可能恢复原会话

**证据：**

- `frontend/src/components/Layout.tsx:156-164` 忽略所有 `apiLogout` 异常，随后无条件清空内存状态并跳转首页。
- Refresh Token 位于 HttpOnly Cookie，前端无法自行删除；`frontend/src/store/auth.ts:74-84` 在初始化时会使用 Cookie 调用 `/auth/refresh` 恢复会话。
- 网络断开或服务端 5xx 时，服务端可能尚未撤销 Token 或清除 Cookie，但当前页面已经把本地状态当作注销完成处理。

**影响：** 在共享设备或多账户使用场景中，用户会看到“已退出”，但刷新页面后可能恢复此前账号，会造成会话状态与用户预期不一致。

**建议：** 对网络和 5xx 类注销失败保留会话、提示用户重试；仅在服务端确认注销成功或确认 Refresh Token 已失效后清理本地认证状态。为成功、401、网络错误和 5xx 分别补充前端回归测试。

**验收标准：** 失败注销不会伪装为服务端注销成功；用户能明确重试，或在 Token 已无效时安全进入未登录状态。

### P2：文章编辑草稿未按账户隔离，可能跨账号暴露未发布内容

**证据：**

- `frontend/src/pages/admin/ArticleEditor.tsx:555-587` 自动把编辑草稿写入 `localStorage`。
- `frontend/src/pages/admin/ArticleEditor.tsx:1395-1419` 的键仅为 `article-editor:draft:${id}`，不包含当前用户 ID；新文章固定使用 `article-editor:draft:new`。
- 同一页面在读取已有文章和新文章时都会读取该键并提示恢复（`ArticleEditor.tsx:482-486`、`522-526`）。

**影响：** 同一浏览器先后登录不同编辑者或管理员时，后登录者可能看到并恢复前一账户尚未发布的草稿内容，尤其是新文章草稿。

**建议：** 将草稿键按当前用户 ID 隔离，必要时加入站点或租户标识；在注销、账户切换和权限失效时清理当前账户草稿；增加跨账号、新文章与已有文章草稿的回归测试。

**验收标准：** 账户 B 无法读取账户 A 的本地草稿，且同一账户的草稿恢复行为保持不变。

### P2：当前站点基址为空，RSS 与 Sitemap 生成本机绝对链接

**证据：**

- 当前 `GET /api/v1/site/settings` 返回 `siteBaseUrl: ""`。
- `GET /rss.xml` 与 `GET /sitemap.xml` 返回的链接均以 `http://127.0.0.1:1270` 开头。
- `internal/logic/site/feed.go:150-159` 在站点基址为空时回退到请求基址，并记录生产配置提示。

**影响：** 本地访问正常，但若以相同站点设置部署到生产，搜索引擎、RSS 阅读器和分享链接可能得到不可公开访问或不稳定的本机地址。

**建议：** 在生产站点设置中填写正式 `https://` 域名，并将 RSS、Sitemap、文章分享链接作为发布验收项。

**验收标准：** 生产 RSS、Sitemap 和文章绝对链接全部使用正式 HTTPS 域名，不依赖请求 Host 推断。

### P2：README 的默认 Compose 拓扑说明与实际 profile 配置不一致

**证据：**

- `docker-compose.yml:269` 和 `:303` 分别将 RabbitMQ、Meilisearch 放入 `messaging`、`search` profile；`.env.example:32-53` 默认 `COMPOSE_PROFILES=`、关闭 RabbitMQ。
- `README.md:13`、`:64`、`:152` 和 `:186` 仍描述快速开始或默认配置会启动/启用 RabbitMQ、Meilisearch。
- `README.md:330` 之后的发布说明则正确描述默认仅启动 Web、API、MySQL 和 Redis，可选能力需要同时启用 profile、功能开关与凭据。

**影响：** 新部署者可能误判异步日志或全文搜索已运行，配置、排障和容量规划会基于错误的服务拓扑。

**建议：** 统一 README 的快速开始、技术栈和环境变量章节，明确默认服务集；将 RabbitMQ、Meilisearch 的 profile、开关和凭据要求放在同一处说明。

**验收标准：** README 全文对默认 Compose 行为的描述一致，并能让新部署者正确判断哪些可选服务已启用。

### P3：Refresh Token 表缺少过期与已撤销记录的清理策略

**证据：**

- `model/token.go:18-22` 每次签发 Refresh Token 都插入新记录。
- `internal/logic/auth/auth.go:300-335` 刷新时仅撤销旧记录并签发新记录。
- 未发现针对 `refresh_tokens` 中过期或长期已撤销记录的定期删除任务、迁移事件或后台清理入口。

**影响：** 长期运行后令牌表会持续增长，备份体积、索引规模及按 hash 查询的维护成本会逐步上升。

**建议：** 增加定期清理机制，按保留窗口删除过期记录和已撤销一段时间的记录；为清理阈值、活跃会话和异常重试补充测试。

**验收标准：** 清理任务不会删除仍有效的 Refresh Token，过期与历史撤销记录在既定保留期后可预测地清除。

## 5. 已有安全与可靠性控制

- API 与 Web 容器以非 root 用户运行；API 不映射宿主机端口，Web 仅绑定本机 `127.0.0.1:1270`。
- Access Token 仅保存在前端内存中；Refresh Token 使用 HttpOnly、SameSite Strict Cookie，JWT 算法受限，并结合 Token version、用户状态与全局 Token cutoff 进行失效控制。
- 管理员和内容管理权限在 Logic 层再次校验；未鉴权后台接口已在运行态返回 401。
- Redis 对敏感接口采用 fail-closed 限流；默认不信任转发头，只有命中可信代理 CIDR 时才使用转发来源。
- AI 出站连接具备 URL、DNS 和建连时 SSRF 校验；媒体上传具备内容检测、扩展名匹配、SHA-256 去重和原子发布。
- 请求日志不记录查询串、Cookie、Header 或正文；Nginx 配置 CSP、`nosniff`、SAMEORIGIN、Referrer-Policy、恢复上传限制和媒体内部目录保护。
- 数据库迁移具备固定版本、checksum、advisory lock、事务记录与就绪检查；备份恢复具备管理员校验、口令加密、大小限制、完整性校验、恢复锁和恢复后 Token 失效处理。
- 前端 API 路径和方法与后端路由/API 描述一致；Markdown 渲染未发现 `dangerouslySetInnerHTML`、`innerHTML` 或原始 HTML 注入路径。

## 6. 测试与构建验证

| 验证 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test ./... -cover` | 通过；部分 Handler 和认证 Logic 覆盖率偏低，需优先补充本报告 P1/P2 路径的回归测试 |
| `frontend/pnpm test` | 通过，87/87 测试通过 |
| `frontend/pnpm build` | 通过，包含 lint、TypeScript 类型检查、Vite 构建和 bundle 预算检查 |
| `docker compose config --quiet` | 通过 |
| 本地运行态检查 | `/`、`/healthz`、`/livez`、站点设置、RSS、Sitemap 和未授权后台统计接口均符合本报告记录 |

项目已配置隔离 Docker 集成测试与 Chromium/WebKit E2E CI；本次未在当前机器执行该套件，避免创建额外 Docker 栈、故障注入或测试数据。

## 7. 优先级行动清单

### 立即处理（P1）

- 为 `scripts/release.ps1 -Rollback` 增加目标镜像与数据库迁移版本兼容性检查；不兼容时阻止“回滚成功”状态并指向备份恢复流程。

### 近期处理（P2）

- 将 Markdown 导入的内容管理权限校验前移到 multipart 解析之前，并清理 multipart 临时文件。
- 修正注销失败时的前端会话状态处理，避免刷新后恢复被宣称已退出的会话。
- 以用户 ID 隔离文章编辑草稿，并在注销和账户切换时清理。
- 在生产环境配置正式 HTTPS `siteBaseUrl`，验证 RSS、Sitemap 和分享链接。
- 统一 README 与 Compose/profile/.env 示例的默认服务说明。

### 持续改进（P3）

- 为 `refresh_tokens` 建立过期与已撤销记录的保留、清理和监控策略。
- 围绕发布回滚、导入权限顺序、会话注销、跨账号草稿与生产配置补充单元、handler 和集成测试；持续观察关键模块覆盖率。

## 8. 剩余风险与假设

- 当前本地备份产物只能证明一次本地备份已生成，不能证明异地同步、保留策略告警或恢复演练已完成。
- 本机 HTTP 使用 `APP_AUTH_COOKIE_SECURE=false` 是正确的本地配置；生产 HTTPS 必须保持 `true`，并实际验证 Cookie 属性与会话续期。
- 本报告未验证真实生产域名、TLS 终止层、外部 Redis/MySQL、可信代理 CIDR、第三方邮件/AI/搜索服务或异地备份存储。
- 当前健康状态证明该实例可用，但不能代替版本升级、灾难恢复、数据库损坏、跨主机迁移和生产回滚演练。
