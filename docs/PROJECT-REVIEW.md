# Notes of Ashen 项目问题评审报告

> 评审日期：2026-07-18（代码修复复核）；宿主机运行、内容分发与修复复核：2026-07-19
>
> 修复版本：P3 修复提交（`fix: 完成项目评审 P3 项`）、既有 P2 修复提交 `1f8f450` 及本次本机部署配置修复
>
> 部署地址：`http://127.0.0.1:1270`
>
> 报告性质：全量评审后的 P2/P3 代码修复复核、宿主机运行配置复核，以及 RSS/Sitemap 等内容分发契约复核；结论按当前代码、Compose 默认值、容器实际注入的非密钥环境变量和本轮真实验证结果更新。
> 改动边界：已更新本机未跟踪 `.env` 的非密钥配置，并仅写入运行数据库的 `site_base_url`；未修改用户账号数据、未删除 Docker 数据卷，也不读取或记录真实密钥。旧库升级仍需显式执行新增增量 SQL。

## 1. 执行摘要

当前版本已完成原 P2-01 至 P2-08 和 P3-01 至 P3-06 的代码修复。后端、前端、静态配置检查以及 core 隔离集成套件均通过；Web 入口 `/healthz` 已返回后端 readiness，`/livez` 独立表示 Nginx 存活。2026-07-19 再次核查并修复宿主机部署后，项目 Compose 的 6 个容器均为 `healthy`，Web 是唯一映射到宿主机的服务端口（`127.0.0.1:1270->8080`）。

本轮未发现权限绕过、数据破坏、生产无法启动或额外数据库/缓存端口暴露的 P0 问题。已按本机 Docker 入口完成 P1-01、P1-02、P2-09 和 P2-10 修复：HTTP Refresh Cookie 不再带 Secure 标志、API 仅信任固定 Web 容器地址、Meilisearch 使用 production 模式、RSS/Sitemap 使用明确的 `siteBaseUrl`。

原报告确认的 8 项 P2 代码修复和 6 项 P3 代码修复仍有可验证实现，本轮未发现回归；本次运行修复的验证证据见第 4 节和第 9.2 节。

| 优先级 | 数量 | 结论 |
| --- | ---: | --- |
| P0 | 0 | 未发现安全灾难、数据破坏或完全权限绕过 |
| P1 | 0 | 当前本机部署的 Refresh Cookie 与可信代理配置已收敛 |
| P2 | 0 | Meilisearch 生产模式与 RSS/Sitemap 规范 URL 已验证 |
| P3 | 0 | 原 P3-01 至 P3-06 已完成修复并纳入构建/CI 门禁；另有若干运维级风险提示见 §5.2 |

问题状态统计：未解决 P0=0、P1=0、P2=0、P3=0；本轮未发现原代码修复回归。

## 2. 评审范围、方法与限制

### 2.1 已审阅范围

- 后端：入口、配置注入、路由、中间件、认证授权、Logic、Model/SQL、缓存、MQ、搜索、邮件、AI、媒体、备份恢复和健康检查。
- 前端：路由、Zustand、Axios Token 刷新、API 封装、TypeScript 类型、后台权限、表单边界、Markdown 渲染、持久化、Loading/Error/Empty State、PWA 和 Vite 拆包。
- 数据与接口：`api/notes-of-ashen.api`、`internal/types`、实际路由、`frontend/src/api`、`frontend/src/types`、`docs/API.md`、MySQL 初始化及增量 SQL。
- 工程：Dockerfile、Compose、Nginx、测试脚本、GitHub Actions、README 和示例配置。

### 2.2 排除范围

- 未读取真实 `.env` 内容，也未在报告中记录任何真实密钥、密码、Token 或私有服务地址。
- 未审阅 `node_modules`、构建产物、测试报告产物、二进制、图片、字体等生成或非文本资产。
- 未对第三方依赖源码做供应链审计；仅检查锁文件、镜像/Action 引用方式和构建告警。
- 未执行破坏性故障演练、清库、`down -v` 或真实生产数据恢复。

### 2.3 结论分类

- **新增**：旧报告未记录，本轮首次形成有证据的当前问题。
- **遗留**：旧报告已有同类风险，本轮确认仍有剩余范围。
- **回归**：旧报告确认修复但当前版本再次出现；本轮为 0。
- **已修复**：旧报告问题在当前代码和验证中不再成立。
- **风险提示**：当前实现可工作，但在特定部署、规模或后续维护条件下风险上升。

## 3. 架构与主要调用链

### 3.1 运行架构

```text
Browser
  -> 127.0.0.1:1270
  -> Web / Nginx (8080, non-root)
       -> SPA 静态资源与 /media 只读卷
       -> /api/* 反向代理到 API:19000
       -> /rss.xml、/sitemap.xml 反向代理到 API 根路由
  -> API / go-zero REST (19000, non-root)
       -> Handler: 参数解析、认证上下文、统一响应
       -> Logic: 业务规则、权限、事务编排
       -> Model: MySQL
       -> Redis: 验证码、限流、Refresh Token 辅助状态、缓存
       -> RabbitMQ: 可选操作日志异步链路
       -> Meilisearch: 可选搜索链路
       -> SMTP: 可选邮件链路
       -> AI Client: 数据库站点设置、API Key v3 密文、SSRF 防护
       -> Media Volume: API 可写、Web 只读
```

### 3.2 认证链路

1. 登录校验密码并签发 HS256 Access Token；Refresh Token 以随机明文交给客户端、Hash 入库。
2. Refresh Token 通过 `HttpOnly`、`SameSite=Strict` Cookie 使用，刷新时执行撤销和轮换。
3. Bearer 请求解析 Access Token 后，认证中间件再从短缓存/数据库确认用户当前角色和状态。
4. 全局备份恢复可通过 access-token not-before 截止时间批量失效旧 Token。
5. 密码修改、找回密码及管理员角色/状态变更会递增用户 `token_version` 并撤销 Refresh Token；认证中间件按数据库版本立即拒绝旧 Access Token。

### 3.3 内容写入链路

- 普通文章：Handler 请求体限制 -> Logic 校验 -> Model 事务写文章、标签关系、版本和搜索/MQ 后续处理。
- Markdown 导入：解析 Front Matter -> 生成 slug -> 在单一 Store 事务中创建/复用分类和标签、创建文章及关联；失败整体回滚。
- 媒体上传：校验 MIME、扩展名、图片解码、尺寸、SHA-256 -> 写入唯一隐藏暂存 -> 创建数据库记录 -> 原子发布；删除使用隔离文件和数据库失败恢复。
- 备份恢复：staging、journal、marker、租约和数据库事务共同保护发布，已覆盖旧报告指出的主要崩溃恢复缺口。

## 4. 当前问题清单

### 4.1 P0 / P1

本轮没有确认的 P0 问题。以下 P1 已在**当前宿主机运行配置**中修复；结论不自动覆盖未来外部反向代理、历史数据库或自定义环境变量组合。

| 编号 | 状态 | 问题与证据 |
| --- | --- | --- |
| P1-01 | 已修复（当前本机 HTTP 部署） | API 容器实际注入 `APP_AUTH_COOKIE_SECURE=false`，与 `http://127.0.0.1:1270` 匹配；Cookie 与认证相关定向测试通过。生产 HTTPS 必须重新设为 `true`。证据：容器 env（2026-07-19）、`internal/logic/auth/cookie.go`、定向 Go 测试。 |
| P1-02 | 已修复 | API 容器实际注入 `APP_TRUSTED_PROXY_CIDRS=172.30.127.10/32`，精确覆盖固定 Web 容器地址；可信代理解析定向测试通过。Web 前方新增外层代理时必须追加其实际出口 CIDR，禁止使用宽泛公网网段。证据：容器 env（2026-07-19）、`internal/httphelper/helper.go`、`docker-compose.yml`、定向 Go 测试。 |

### 4.2 P2 问题

原报告的 P2-01 至 P2-08 已在提交 `1f8f450` 全部修复；部署配置与内容分发的 P2-09、P2-10 已在当前本机部署完成修复并验证。

| 编号 | 状态 | 修复与验证证据 |
| --- | --- | --- |
| P2-01 | 已修复 | `users.token_version`、JWT Claim 和认证数据库校验已联动；改密、重置密码、角色/状态变更在事务内递增版本并撤销 Refresh Token，旧 Access Token 测试通过 |
| P2-02 | 已修复 | `Store.CreateMarkdownArticle` 将分类、标签、文章和关联纳入同一事务；文章插入失败回滚 taxonomy 的 Model 测试通过 |
| P2-03 | 已修复 | 上传采用 `.upload-*` 暂存和数据库失败补偿，删除采用 `.delete-*` 隔离及恢复；缺少 `file` 返回 400，真实媒体集成路径通过 |
| P2-04 | 已修复 | 剩余结构化写接口统一使用 `ParseLimited`，按 16 KiB、64 KiB、12 MiB 分档；Content-Length 与 chunked 超限路径均有测试 |
| P2-05 | 已修复 | Service Worker 仅对 SPA HTML 导航更新 `/index.html`，排除 API、媒体、XML、健康探针和文件路径，并等待 `cache.put` |
| P2-06 | 已修复 | Nginx 精确代理 `/healthz` 到后端 readiness，`/livez` 返回 204；core 隔离集成测试验证 JSON readiness 和独立 liveness |
| P2-07 | 已修复 | 项目写入只接受 `tagIds`，非空旧 `tags` 返回 400；事务内从标签表生成名称并同步 JSON 快照、实体和关系 |
| P2-08 | 已修复 | 项目逐行插入并读取各自 `LastInsertId`，非连续 ID 的 Model 测试通过 |
| P2-09 | 已修复 | Compose、示例配置和实际 Meilisearch 容器均使用 `MEILI_ENV=production`；服务仍未映射宿主机端口，未带 Key 访问 `/indexes` 返回 401，公开搜索建议接口可正常返回结果。 |
| P2-10 | 已修复（当前部署） | 运行数据库已配置 `siteBaseUrl=http://127.0.0.1:1270`；RSS/Sitemap 在正常和伪造 Host 下均输出该规范地址且保留 `:1270`。Nginx 对这两个端点改用 `$http_host` 以兼容非标准端口。未来不得清空 `siteBaseUrl` 后继续依赖请求 Host 生成规范 URL。 |

## 4.3 P3 修复记录

### P3-01 构建镜像和 GitHub Actions 未固定到不可变摘要

- 状态：**已修复**
- 证据：Dockerfile 语法镜像、Go/Node/Alpine/Nginx 基础镜像，以及 Compose 的 MySQL、Redis、RabbitMQ、Meilisearch 均保留可读 tag 并固定 manifest digest；所有 workflow Action 已固定完整 commit SHA。
- 防回归：新增 `.github/dependabot.yml`，每周检查 `docker` 与 `github-actions` 更新。

### P3-02 默认 Compose 总是启动可选 RabbitMQ 与 Meilisearch

- 状态：**已修复**
- 证据：`rabbitmq` 使用 `messaging` profile，`meilisearch` 使用 `search` profile；默认 `docker compose config --services` 仅列出 Web、API、MySQL、Redis。
- 兼容性：README、1Panel 说明和 `.env.example` 使用 `COMPOSE_PROFILES=messaging,search`；已有启用部署需要同步声明对应 profile。

### P3-03 自动化覆盖率仍集中在工具层，且 CI 未单独执行 go vet

- 状态：**已修复（高风险定向覆盖）**
- 证据：push/PR 核心 CI 新增 `go vet ./...`；新增 JWT 解析、Refresh Cookie 回退和 `UserTokenVersion` 成功/未找到/数据库失败的直接测试。
- 边界：不设置脆弱的全局覆盖率百分比门槛；后续新增高风险分支继续要求包内直接测试。

### P3-04 JWT 解析未显式限定 HS256

- 状态：**已修复**
- 证据：`ParseAccessToken` 使用 `jwt.WithValidMethods([]string{"HS256"})`；HS256 正常解析、HS384 和 `none` 算法拒绝测试均已覆盖。

### P3-05 存在未使用的维护代码和常量

- 状态：**已修复**
- 证据：删除未调用的 `pruneMedia`；媒体恢复继续由事务化 staging/journal 发布与旧目录 finalize 完成清理。删除未生效的 `maxPageContentLength`，不再以死常量表达页面限制。

### P3-06 前端大模块仍有首访性能压力

- 状态：**已修复（构建体积门禁）**
- 证据：Markdown 渲染器在实际内容出现后加载，语法高亮仅在代码块出现后加载，ECharts 在图表区域有数据时动态加载；Vite 告警阈值降为 550 KiB。
- 防回归：`pnpm build` 自动执行无依赖体积检查，限制初始入口并要求 Markdown、语法高亮、ECharts 保持独立、不得写入 `index.html` 初始资源。当前检查值为入口 294.57 KiB、Markdown 423.48 KiB、语法高亮 95.69 KiB、ECharts 500.81 KiB。
- 边界：本轮未采集真实用户 Web Vitals，避免增加用户性能数据收集与存储范围。

## 5. 安全审计结论

### 5.1 已确认有效的安全控制

- 首个用户管理员逻辑、注册验证码例外条件和后续注册校验路径一致，未发现可重复获取管理员权限的绕过。
- Access Token 中角色不被直接信任；认证中间件会回查短缓存/数据库中的当前角色和状态，禁用用户会被拒绝。
- Refresh Token 使用随机值、Hash 持久化、轮换和撤销；Cookie 使用 `HttpOnly`、`SameSite=Strict`，生产默认 Secure。
- 登录、注册、验证码、密码重置等敏感入口有 Redis 限流；关键路径在 Redis 故障时采用保守策略。
- 默认不信任 `X-Forwarded-*`/`X-Real-IP`；只有 `RemoteAddr` 命中 `APP_TRUSTED_PROXY_CIDRS` 时才采纳转发头。
- AI API Key 使用 `v3:` 密文和独立用途派生密钥；旧版本密文兼容策略明确。
- AI URL 校验覆盖私网/保留地址、DNS 解析、重定向和代理绕过，HTTP Client 禁止环境代理并限制重定向。
- 媒体上传校验 MIME、扩展名、图片解码、尺寸和 SHA-256，storage key 不取信用户文件名。
- Markdown 使用 `react-markdown` 安全默认链路，未启用 raw HTML/`rehype-raw`；外链和图片 URL 经过限制。
- 备份恢复要求管理员身份、当前密码、口令和确认文本，并使用 staging、journal、marker、租约、路径/大小/Hash 校验和故障恢复流程。
- Docker API/Web 以非 root 用户运行；MySQL、Redis、RabbitMQ、Meilisearch 均未映射宿主机端口；Web 只绑定 loopback。
- Nginx 配置安全头、API 限流、source map 404、媒体恢复临时路径拒绝和备份专用请求体上限。

### 5.2 尚需处理的安全边界

- 用户级 Access Token 已通过 `token_version` 支持立即撤销；改密后当前及其他会话均退出。
- 结构化写接口已增加端点级请求体限制，chunked 请求也由 `MaxBytesReader` 约束。
- JWT 解析已显式限定 HS256，并由错误算法拒绝测试保护，见 P3-04。
- 外部 readiness 和 Web liveness 已分离；后续仍应按实际部署环境配置监控来源和告警策略。
- 当前本机 HTTP 入口已使用 `APP_AUTH_COOKIE_SECURE=false`；切换 HTTPS 时必须恢复为 `true`（P1-01）。
- 当前 API 仅信任固定 Web 容器 `172.30.127.10/32`；新增可信代理时按实际链路追加精确 CIDR，禁止使用宽泛公网网段（P1-02）。
- Meilisearch 已运行于 `production` 模式，仍未映射宿主机端口，且 Master Key 鉴权保持生效（P2-09）。
- `siteBaseUrl` 已固定为 `http://127.0.0.1:1270`，RSS/Sitemap 不再依赖请求 Host；Nginx 同时使用 `$http_host` 保留非标准端口。若未来清空该设置，仍应避免将不可信请求 Host 作为规范 URL（P2-10）。
- **风险提示（不计入未解决 P3 计数）**：
  - Nginx CSP 含 `upgrade-insecure-requests`；生产应走 HTTPS 反代，纯 HTTP 本机入口需知悉浏览器可能升级部分请求。
  - Redis 当前无密码且仅暴露在 Compose 内网；未映射宿主机，但 Docker 网络横向失陷时可读取验证码/限流等缓存键。
  - API Prometheus `/metrics` 监听容器内 `0.0.0.0:9101`（compose `expose` 不映射宿主机）；编排切勿误映射到公网。
  - 修改密码/邮箱路径强依赖邮箱验证码；当前运行已启用邮件，若关闭邮件服务则改密会被验证码消费阻断（设计约束）。

## 6. 旧报告问题重新验证

以下结论均按当前代码和本轮运行结果重新检查，不直接继承旧报告的完成标记。

| 历史问题 | 当前状态 | 重新验证结论 |
| --- | --- | --- |
| 数据库名配置与初始化不一致 | 已修复 | Compose 和示例配置固定使用 `notes_of_ashen`，本轮服务正常连接现有卷 |
| readiness 只检查 DB/Redis、不检查 schema | 已修复 | 容器内 `/healthz` 返回 `db`、`redis`、`schema` 均为 up |
| 文章/分类/标签缺少长度和请求体限制 | 已修复 | 所有剩余结构化写接口也已按业务容量使用 `ParseLimited` |
| 生产 source map 可公开访问 | 已修复 | Vite `sourcemap: false`，Nginx 对 `.map` 拒绝 |
| 深分页缺少 page 上限 | 已修复 | 当前分页校验包含 page/size 边界 |
| Request ID 长度和字符缺少限制 | 已修复 | 中间件已校验格式与长度 |
| 备份恢复媒体与数据库发布缺少崩溃恢复 | 已修复 | staging/journal/marker/租约及 extended 故障注入测试通过 |
| API 文档与错误码/字段明显漂移 | 已修复 | 项目写入统一为 `tagIds`，API、Go/TS 类型、前端请求和文档已同步 |
| 前端检查因环境错误无法执行 | 已修复（验证条件） | 本轮 `test`、`lint`、`type-check`、`build` 均实际通过 |

本轮未发现上述已修复事项发生回归。

## 7. 前后端、API 与数据库一致性

### 7.1 已确认一致

- `api/notes-of-ashen.api`、`internal/types`、Handler/Logic 和前端 API 封装的主要路由、分页、统一响应结构一致。
- 站点设置可选布尔字段使用指针语义，字段缺失与显式 `false` 可区分。
- 文章、分类、标签、媒体、备份、AI 设置和用户管理的前端请求字段与后端类型总体一致。
- schema readiness 已覆盖 `users.token_version` 等当前必需字段；隔离新库验证通过，既有卷上线前必须执行新增迁移。
- 初始化 SQL 与增量脚本的目标数据库和 Compose DSN 一致。

### 7.2 已确认漂移

- 本轮未确认新的前后端字段、项目标签写入或健康探针契约漂移。
- 旧客户端若继续提交非空 `tags` 会明确收到 400，这是收敛写入契约的预期兼容性行为。
- 内容分发契约：当前 `siteBaseUrl` 固定为 `http://127.0.0.1:1270`，RSS/Sitemap 的绝对 URL 不受请求 Host 影响且保留端口；未来变更域名、协议或端口时必须同步更新该设置，见 P2-10。

### 7.3 数据一致性风险

- Markdown taxonomy、文章及标签关系已使用单事务写入。
- 媒体文件与数据库仍属于不同资源，但已通过唯一暂存、隔离删除和可恢复补偿关闭已知双向窗口。
- 项目标签关系不再依赖 MySQL 自增步长，并与 JSON 快照在同一事务内保持一致。

## 8. 测试、CI 与构建结果

### 8.1 后端

| 命令 | 结果 | 说明 |
| --- | --- | --- |
| `go test ./...` | 通过 | 全部 Go 包通过，包含本轮新增 Token、事务、媒体、项目和 Handler 测试 |
| `go vet ./...` | 通过 | 无静态检查错误 |
| `go build -o bin/notes-of-ashen.exe ./cmd/notes-of-ashen` | 通过 | 后端可执行文件构建成功 |

原评审记录的代表性覆盖率（本轮未重新统计）：

| 包/区域 | 覆盖率 |
| --- | ---: |
| `internal/aiclient` | 78.6% |
| `internal/config` | 70.7% |
| `internal/middleware` | 62.6% |
| `internal/logic/media` | 50.3% |
| `internal/logic/auth` | 10.0% |
| `internal/logic/system` | 12.0% |
| `internal/mq` | 10.5% |
| `model` | 22.4% |

多数 Handler 的深度路径仍主要由集成/E2E 覆盖；本轮针对认证、Refresh Cookie 和用户令牌版本补充直接测试，后续新增高风险分支继续采用定向包内测试策略。

### 8.2 前端

| 命令 | 结果 | 说明 |
| --- | --- | --- |
| `pnpm test` | 通过 | 73 项测试通过 |
| `pnpm lint` | 通过 | 无 lint 错误 |
| `pnpm type-check` | 通过 | TypeScript 类型检查通过 |
| `pnpm build` | 通过 | 生产构建成功，并通过入口/Markdown/高亮/ECharts 体积门禁 |

### 8.3 集成测试

| 命令 | 结果 | 说明 |
| --- | --- | --- |
| `scripts/test-integration.ps1 -Suite core` | 通过 | 隔离环境、后端 HTTP 集成及 4 项 Chromium E2E 通过；覆盖 Web `/healthz` 与 `/livez`，环境自动清理 |
| `scripts/test-integration.ps1 -Suite extended` | 通过 | 并发、Redis 故障、恢复失败注入及 E2E 通过，环境自动清理 |
| `node --check frontend/public/sw.js` | 通过 | Service Worker 语法检查通过 |

### 8.4 CI 审计

- Frontend CI 使用 pnpm 锁文件安装并运行前端测试与构建。
- Integration E2E 在 push/PR 运行前端 lint/type-check/test/build、Go test、`go vet` 和 core 集成套件；extended 在计划任务或手动触发时运行。
- Action 已固定完整 SHA，Docker/Action 更新由 Dependabot 每周创建审查请求，见 P3-01、P3-03。

## 9. Docker 部署与运行核验

### 9.1 执行结果

| 命令/检查 | 结果 |
| --- | --- |
| `docker compose config --quiet` | 通过 |
| core 隔离 Compose 构建与启动 | 通过；测试项目、网络和命名卷在结束后自动清理 |
| Web `GET /healthz` | `200 application/json`，返回后端 DB/Redis/schema readiness |
| Web `GET /livez` | `204 No Content`，仅表示 Nginx 存活 |
| 真实媒体上传与 Web 读取 | 通过；发布文件权限和暂存路径策略均生效 |

### 9.2 本次宿主机容器复核

复核日期：2026-07-19（承接 2026-07-18 复核结论并扩展 feed/sitemap 与运行 env 非密钥字段）。

执行了 `docker ps`、`docker compose ps`、额外容器名称检查、容器网络 IP、非密钥环境变量抽样，以及 `GET http://127.0.0.1:1270/healthz`、`/livez`、公开 settings、RSS/Sitemap。结果如下：

- `mysql8`：未发现。
- `redis7`：未发现。
- 当前运行的 6 个容器均属于 `notes-of-ashen` Compose 项目并处于 `healthy` 状态：`web`、`api`、`mysql`、`redis`、`meilisearch`、`rabbitmq`。
- 只有 `notes-of-ashen-web-1` 映射 `127.0.0.1:1270->8080/tcp`。
- API、MySQL、Redis、RabbitMQ、Meilisearch 均未映射宿主机端口；宿主机探测 `3306`/`6379`/`7700`/`15672`/`9101` 不可达。
- Web 固定地址 `172.30.127.10`；API 实际注入（非密钥）：`APP_AUTH_COOKIE_SECURE=false`、`APP_TRUSTED_PROXY_CIDRS=172.30.127.10/32`、`APP_SEARCH_ENABLED=true`、`APP_RABBITMQ_ENABLED=true`、`APP_EMAIL_ENABLED=true`。
- Meilisearch：`MEILI_ENV=production`；Master Key 已设置（容器内无 key 访问 `/indexes` 返回 401），公开搜索建议接口可正常返回结果。
- `/healthz` 返回 HTTP 200，`db`、`redis`、`schema` 均为 `up`；`/livez` 返回 204。
- 公开 `siteBaseUrl` 为 `http://127.0.0.1:1270`；`/rss.xml`、`/sitemap.xml` 在正常和伪造 Host 下均输出该规范地址（含 `:1270`），见 P2-10。

本项仅代表本次核查与修复后的宿主机状态；仅重建并重启了 Web、API 和 Meilisearch，未停止或删除数据卷，也未读取真实 `.env` 文件内容。

### 9.3 健康与日志

- core 与 extended 隔离环境未发现 API 启动失败或未清理的测试资源。
- `/healthz` 不再落入 SPA fallback；集成测试确认其返回 JSON 健康报告。
- `/livez` 独立返回 204，不会把 Web 存活误报为后端依赖就绪。

### 9.4 数据卷与暴露面

- 未对当前运行中的用户数据库或 Docker 数据卷执行迁移、清库或删除。
- 集成测试只使用隔离项目和临时命名卷，并在结束后自动清理。
- 本次复核未发现额外的 MySQL/Redis 容器或其宿主机端口映射；API、数据库、缓存、MQ 和搜索服务不映射宿主机端口。
- API/Web 配置资源限制和 json-file 日志滚动；媒体卷在 Web 侧只读挂载。
- 仓库根目录存在本地 `.env`、构建产物 `notes-of-ashen.exe` 与 `*_backup.sql` 等文件，均已被 `.gitignore` 覆盖且未纳入版本跟踪；部署与备份流程仍应避免误提交。

## 10. 后续路线图

原 P2/P3 代码修复阶段已完成；当前运行配置、内容分发与部署状态按以下事项维护：

1. 审查 Dependabot 提交的镜像和 GitHub Action 更新，按发布流程更新固定摘要。
2. 新增认证、数据写入和 Handler 高风险分支时同步补充直接测试。
3. 保持 `pnpm build` 体积门禁；若未来确有性能运营需求，再单独评估 Web Vitals 数据采集与隐私边界。
4. 切换 HTTPS、域名或外部反向代理前，重新核对 `APP_AUTH_COOKIE_SECURE`、`APP_TRUSTED_PROXY_CIDRS` 与完整代理链；禁止配置宽泛公网网段。
5. 搜索启用时持续保持 Meilisearch production 模式、有效 Master Key 和非宿主机端口暴露。
6. 变更域名、协议或端口时同步更新 `siteBaseUrl`，并复核 RSS/Sitemap 的规范 URL；不要在清空该设置后信任请求 Host。
7. 若需内网抓取 metrics，仅在受控编排网络内访问 API `:9101`，不要映射到宿主机公网接口。

## 11. 最终结论

`1f8f450` 已关闭原报告 P2-01 至 P2-08：会话失效、Markdown 事务、媒体补偿、请求体上限、PWA 缓存、公开 readiness、项目标签契约和非连续自增 ID 均已有实现及验证证据。第 4.3 节原 P3-01 至 P3-06 亦已关闭并纳入 CI/构建门禁。

2026-07-19 宿主机复核未发现额外 MySQL/Redis 容器或宿主机数据库/缓存端口暴露；本轮也未发现新的 P0 权限绕过或数据破坏类代码缺陷。P1-01、P1-02、P2-09 和 P2-10 已在当前本机部署完成修复：

- HTTP 本机入口使用非 Secure Refresh Cookie，API 仅信任固定 Web 容器 CIDR；
- Meilisearch 使用 production 模式并保持 Master Key 鉴权与非宿主机端口暴露；
- `siteBaseUrl` 固定为 `http://127.0.0.1:1270`，RSS/Sitemap 在正常与伪造 Host 下均输出该规范地址（含端口）。

当前未关闭问题统计为 P0/P1/P2/P3 均为 0。P2-10 的关闭以前述规范 `siteBaseUrl` 持续存在为前提；未来不得清空该设置后继续信任请求 Host。

上线旧数据库前必须先执行 `deploy/mysql/add_user_token_version.sql`。未迁移数据库会由 readiness 的 schema 检查判定为未就绪，避免在缺失 `users.token_version` 时继续提供服务。后续优先收敛运行配置与 feed URL 契约，再按常规依赖更新、定向测试和构建体积门禁维护。
