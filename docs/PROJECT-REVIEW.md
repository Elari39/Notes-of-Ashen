# Notes of Ashen 项目审计报告

## 1. 审计结论

**审计日期：** 2026-07-26

**审计对象：** `Notes of Ashen` 前后端分离博客系统

**运行环境：** 本地 Docker Compose，Web 入口为 `http://127.0.0.1:1270`

**总体结论：** 当前版本具备较完整的认证、权限、数据迁移、媒体和恢复保护，容器与基础依赖运行正常，核心页面和公开接口可用。审计未发现可直接确认的 P0 问题，但本地 HTTP 部署存在一个会影响会话续期的 P1 配置风险；生产上线前还需要补齐正式站点基址、应用镜像不可变版本和备份运维流程。

本报告只记录审计结果，不修改业务代码、配置或部署文件。

## 2. 范围与方法

### 2.1 检查范围

- Go 后端：API 描述、Handler、Logic、认证、权限、配置、缓存、搜索、媒体、备份恢复和数据库迁移。
- React/TypeScript 前端：路由、HTTP 封装、认证状态、页面加载状态、构建产物和后台功能入口。
- 部署：`docker-compose.yml`、Dockerfile、Nginx 配置、健康检查、数据卷和迁移/配置检查任务。
- 运行态：容器健康状态、公开接口、未授权访问、浏览器页面和控制台输出。

### 2.2 边界与限制

- 未读取或披露真实 `.env`、API Key、Token、密码和其他敏感配置；配置结论只使用脱敏后的 Compose 配置、公开配置文件和运行态观察结果。
- 当前验证针对本地 HTTP 入口，不等同于生产 HTTPS、正式域名、外部备份存储或异地故障恢复验证。
- 浏览器打开文章详情会触发站点已有的阅读计数逻辑，因此该检查可能改变测试数据中的阅读计数。

## 3. 运行态验证证据

| 检查项 | 结果 |
| --- | --- |
| Docker 服务 | `web`、`api`、`mysql`、`redis`、`rabbitmq`、`meilisearch` 均为 healthy |
| 初始化任务 | `config-check`、`migrate` 均正常退出，Exit 0 |
| 存活/就绪接口 | `/healthz`、`/livez` 可访问 |
| 公开 API | 文章列表、站点设置、RSS、Sitemap 可访问 |
| 权限边界 | 未鉴权访问 `/api/v1/admin/stats` 返回 401 |
| 浏览器页面 | 首页、归档、搜索、登录、文章详情最终渲染正常；控制台无 error/warning |
| 文章详情 | 首次加载约 1.5 秒后正常显示；访问可能增加阅读计数 |
| 运行态关键配置 | `WEB_PORT=1270`，`APP_AUTH_COOKIE_SECURE=true`；站点设置中的 `siteBaseUrl` 为空 |

## 4. 审计发现

### P1：本地 HTTP 部署启用 Secure refresh Cookie

**证据：**

- `docker-compose.yml:72` 默认将 `APP_AUTH_COOKIE_SECURE` 设为 `true`。
- `docker-compose.yml:17` 的 Web 端口绑定为 `127.0.0.1:${WEB_PORT:-1270}:8080`，当前入口为 HTTP。
- `internal/logic/auth/cookie.go:21-27` 根据配置写入 `HttpOnly`、`SameSite=Strict` 和 `Secure` Cookie。
- `internal/handler/auth/auth.go:17-23` 会从响应体移除 refresh token，长期凭证只通过 Cookie 传递。

**影响：** 浏览器在 HTTP 页面下不会发送或持久化带 `Secure` 属性的 Cookie。登录后短期 access token 可能仍然可用，但刷新页面或 access token 过期时无法通过 refresh 接口恢复会话，表现为被迫重新登录。

**建议：**

1. 本地 HTTP 环境显式设置 `APP_AUTH_COOKIE_SECURE=false`。
2. 生产 HTTPS 环境保持 `APP_AUTH_COOKIE_SECURE=true`，并在部署检查中验证 HTTPS、Cookie 属性和 refresh 流程。
3. 增加端到端验收：登录后等待 access token 过期或主动刷新页面，确认 Cookie refresh 可以换取新的 access token。

**验收标准：** 本地 HTTP 登录后刷新页面仍保持登录；生产 HTTPS 响应包含 `Secure; HttpOnly; SameSite=Strict`，且 refresh 成功。

### P2：生产站点基址为空时，RSS/Sitemap 回退到请求 Host

**证据：**

- 当前 `/api/v1/site/settings` 返回 `siteBaseUrl: ""`。
- `internal/logic/site/feed.go:150-155` 在站点基址为空时使用请求基址生成绝对 URL。

**影响：** 本地访问时 RSS/Sitemap 使用 `http://127.0.0.1:1270` 尚可接受；生产若未配置正式 HTTPS 域名，搜索引擎和订阅客户端会收到错误、不可公开访问或不稳定的绝对链接。

**建议：** 生产部署前在站点设置中配置正式的 `https://` 站点地址，并把 RSS、Sitemap 和文章分享链接作为发布验收项。可在部署检查中加入“生产环境禁止空 `siteBaseUrl`”的策略。

**验收标准：** 生产 `/rss.xml`、`/sitemap.xml` 和文章链接全部使用正式 HTTPS 域名，不依赖请求头中的 Host 推断。

### P2：应用镜像使用 latest 标签，发布不可复现

**证据：** `docker-compose.yml:5,45,128,157` 中 `web`、`api`、`migrate`、`config-check` 使用 `${IMAGE_TAG:-latest}`；当前运行镜像为 `notes-of-ashen-api:latest`、`notes-of-ashen-web:latest`。MySQL、Redis、RabbitMQ 和 Meilisearch 则已使用固定版本及 digest。

**影响：** 同一 Compose 配置在不同时间可能拉取不同应用内容，难以准确追溯构建来源、复现问题和执行可靠回滚；应用与迁移任务也可能出现版本漂移。

**建议：** CI 构建后生成不可变版本号或 digest，并让 `IMAGE_TAG` 成为发布必填项；保留上一版本镜像和对应迁移记录，发布时记录 Git commit、镜像 digest 与数据库迁移版本。

**验收标准：** 生产 Compose 展开结果不包含 `latest`；同一版本可在另一台环境重建并得到相同镜像 digest，且可按记录回滚。

### P2/P3：备份持久化、异地保存和恢复演练依赖运维流程

**已有控制：**

- 管理员权限与当前密码校验。
- age scrypt 加密、归档路径约束、数量/大小限制、SHA-256 和关联关系校验。
- 媒体目录 journal、staging、rollback 和恢复标记。
- Redis lease 与进程级锁防止并发恢复；恢复后清缓存、失效 Token，并尝试重建搜索索引。
- 数据库迁移具备版本、SHA-256 checksum、advisory lock 和 readiness 检查。

**缺口与影响：** Compose 提供了命名数据卷，但未发现自动定期备份、异地/离线副本、备份保留策略和定期恢复演练的完整运维闭环。主机、卷或密钥同时损坏时，代码层保护不能替代外部备份策略。

**建议：** 建立加密备份任务、异地或对象存储副本、保留周期和告警；至少按月在隔离环境执行一次全量恢复，验证数据库、媒体、搜索索引、AI 设置迁移和管理员登录链路。

**验收标准：** 有可审计的备份成功记录和保留策略；最近一次恢复演练有时间、版本、数据完整性和耗时记录，并能在目标恢复时间内完成。

### P3：前端 bundle 体积偏大

**观察：** 构建门禁通过，但构建产物中 ECharts 约 512 KB raw、Markdown 相关模块约 431 KB raw、初始 JS 约 291 KB raw。

**影响：** 首次访问和低带宽设备上的脚本下载、解析成本较高，后台图表和 Markdown 能力会增加资源占用。

**建议：** 保持 ECharts 等后台能力按需加载，继续拆分 Markdown/语法高亮模块；为初始 chunk 和单 chunk 增加 gzip/brotli 后预算，并通过真实移动网络指标决定是否继续优化。

**验收标准：** 关键公开页面不加载后台专用模块；构建 bundle 检查持续通过，且核心页面的 LCP/JS 下载量不因功能迭代持续恶化。

## 5. 已有安全与可靠性控制

- API 与 Web 容器使用非 root 用户；API 不直接映射宿主机端口，Web 仅绑定本机 `127.0.0.1:1270`。
- refresh token 使用 HttpOnly、SameSite Strict Cookie；JWT 方法限制、token version 和全局 token cutoff 可用于撤销会话。
- 管理员与内容管理角色在 Logic 层再次校验，避免只依赖路由层鉴权。
- Redis 敏感接口限流采用 fail-closed 策略；默认不信任 `X-Forwarded-*`/`X-Real-IP`，仅对命中可信代理 CIDR 的请求使用转发头。
- AI 连接 URL 有 SSRF 防护；媒体上传包含内容检测、扩展名匹配、SHA-256 去重和原子发布。
- 请求日志不记录查询串、Cookie、Header 和正文；Nginx 提供 CSP、nosniff、SAMEORIGIN、Referrer-Policy 和静态资源缓存策略。
- 数据库迁移具备连续版本、checksum 漂移检测、advisory lock、事务记录和 readiness 检查。
- 备份恢复流程具备权限、加密、大小限制、完整性校验、恢复标记和并发恢复保护。

## 6. 测试与构建验证

以下结果已在本次审计前完成并纳入结论：

| 验证 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go build ./...` | 通过 |
| `frontend/pnpm lint` | 通过 |
| `frontend/pnpm type-check` | 通过 |
| `frontend/pnpm build` | 通过；仅需关注 bundle 体积观察项 |
| `go test ./... -cover` | 通过；认证、业务 Logic、Handler 等部分覆盖率偏低 |

覆盖率偏低不代表当前功能失败，但建议针对登录/refresh、权限边界、备份恢复失败路径和生产配置矩阵补充集成或端到端测试。

## 7. 优先级行动清单

### 立即处理（P1）

- 为本地 HTTP 部署设置 `APP_AUTH_COOKIE_SECURE=false`，并验证登录、刷新、登出和页面重载。
- 为生产 HTTPS 保留 `true`，将 Cookie 属性检查加入发布验收。

### 发布前处理（P2）

- 配置正式 HTTPS `siteBaseUrl`，验证 RSS、Sitemap 和分享链接。
- 用不可变镜像 tag/digest 替换应用 `latest`，建立版本、迁移和回滚记录。
- 固化备份加密密钥管理、异地副本、保留周期、告警和恢复演练计划。

### 持续改进（P3）

- 监控初始 bundle、LCP 和移动端资源成本，继续按需拆分图表和 Markdown 依赖。
- 补充关键安全与恢复路径的集成测试，并持续检查覆盖率趋势。

## 8. 剩余风险与假设

- 本报告没有验证真实生产域名、TLS 终止层、可信代理 CIDR 和外部对象存储策略；这些配置需要上线前单独复核。
- 本地 Docker 健康状态只能证明当前实例正常，不能证明升级、节点故障、数据库损坏或跨主机恢复可用。
- 运行态检查没有执行真实备份导出/恢复演练，以避免改变本地数据；相关结论来自代码审计和部署结构检查。
- 未读取真实 `.env`，因此无法对其中的密钥强度、轮换周期和第三方凭证有效性作结论。
