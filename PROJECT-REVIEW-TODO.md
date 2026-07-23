# Notes of Ashen 项目审查 TODO 报告

> 审查日期：2026-07-21
> 审查目标：`http://127.0.0.1:1270`
> 项目形态：Go + go-zero 后端、React + TypeScript + Vite 前端、Docker Compose 部署
> 报告性质：基于当前代码、配置、文档、测试和本机运行态的工程审查，不是安全认证、压力测试或形式化正确性证明。

## 1. 结论摘要

当前部署可以正常提供服务，未发现有证据支持的 P0 级安全、数据破坏或生产无法启动问题。Docker 中 API、Web、MySQL、Redis、RabbitMQ、Meilisearch 均处于 healthy，公开健康探针返回 `200`，数据库 schema readiness 正常；Go 后端和前端静态检查、单元测试、生产构建均通过。

当前最值得优先处理的是部署升级链路，而不是页面功能：数据库存在多份增量 SQL，但 Compose 只会在 MySQL 数据卷首次创建时自动执行 `schema.sql`。新旧版本之间的迁移依赖人工操作，容易造成升级后 API readiness 失败或版本不一致。其次是 Redis 密码与本地 Compose 服务端配置没有形成闭环，以及可选 RabbitMQ/Meilisearch 的开关与 Compose profile 缺少自动一致性校验。

### 优先级总览

| 优先级 | TODO 数量 | 核心风险 |
| --- | ---: | --- |
| P0 | 0 | 本次未发现明确的立即性安全或数据破坏问题 |
| P1 | 2 | 数据库升级不可自动完成；Redis 认证配置可能导致部署不可用 |
| P2 | 6 | 接口契约、数据完整性、可选依赖降级和统计口径存在维护或一致性风险 |
| P3 | 3 | 移动端验证、首屏资源体积和长期文档/测试自动化仍可加强 |

## 2. 审查范围与方法

### 2.1 静态审查范围

- 后端入口、配置、服务上下文、HTTP handler、middleware、logic、model、缓存、搜索、邮件、消息队列和备份恢复代码。
- API 描述、后端 `internal/types`、前端 API 封装、前端类型和页面调用链。
- `docker-compose.yml`、`.env.example`、`etc/notes-of-ashen.yaml`、MySQL 初始化/增量 SQL、README 和 API/测试文档。
- Go 单元测试、SQL mock、集成测试脚本、前端单元测试、Playwright 配置和 E2E 用例清单。

### 2.2 运行态审查范围

- `docker compose ps` 和 `docker compose config --quiet`。
- `GET /healthz`、`GET /api/v1/site/settings`。
- 浏览器访问首页、归档、搜索、文章详情、注册、登录、管理后台和系统工具页面。
- 搜索关键词 `Go` 的建议与结果展示。
- 浏览器控制台错误/警告观察。

### 2.3 有意未执行的操作

- 未读取真实 `.env`，也未在报告中记录数据库密码、JWT secret、AI API Key 或其他敏感配置。
- 未对现有部署执行注册、登录、文章编辑、备份恢复等会写入数据的 E2E 操作。
- 未把当前运行中的 Docker 数据卷用于故障注入、删除、恢复或压力测试。
- 浏览器检查覆盖了桌面流程；当前 Playwright 配置只有 `Desktop Chrome` 项目，不能替代真实移动设备验证。

## 3. 已验证的当前状态

### 3.1 部署与接口

- `docker compose ps`：API、Web、MySQL、Redis、RabbitMQ、Meilisearch 均为 `Up ... (healthy)`。
- `docker compose config --quiet`：通过。
- `GET http://127.0.0.1:1270/healthz`：HTTP `200`。
- 健康响应为：

  ```json
  {"status":"ok","checks":{"db":{"status":"up"},"redis":{"status":"up"},"schema":{"status":"up"}}}
  ```

- `GET /api/v1/site/settings`：HTTP `200`，统一响应包含 `code`、`message`、`data`。
- Web 入口返回了 `X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、CSP、`Cache-Control: no-store` 等响应头；本报告未继续进行完整安全基线或渗透测试。

### 3.2 自动化验证

| 验证 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go build ./...` | 通过 |
| `frontend/pnpm test` | 76 个测试通过，0 failed |
| `frontend/pnpm lint` | 通过 |
| `frontend/pnpm type-check` | 通过 |
| `frontend/pnpm build` | 通过，包含 lint、type-check、Vite build、bundle size check |
| `docker compose config --quiet` | 通过 |
| Playwright 真机/移动端 E2E | 本次未执行 |

构建产物中 `echarts`、Markdown 和语法高亮相关 chunk 较大，但项目自带 `check:bundle-size` 已通过；这属于后续性能 TODO，不是当前构建失败。

## 4. P1 TODO：近期必须处理

### TODO-P1-01 建立正式的数据库迁移流程

- [x] **优先级**：P1；**负责人建议**：后端 + 运维；**预估**：中等。
- **问题**：数据库有多份增量脚本，但 Compose 只挂载初始 schema。已有数据卷不会因为 API 镜像升级而自动执行新增表、字段或索引变更。
- **证据**：
  - `docker-compose.yml:131-133` 只把 `deploy/mysql/schema.sql` 挂载为 MySQL 初始化脚本。
  - `deploy/mysql/` 另有多份 `add_*`、`alter_*`、`cleanup_*` SQL，例如 `add_content_growth_features.sql`、`add_media_content_analytics.sql`、`add_user_token_version.sql` 等。
  - `README.md:517-519` 明确写明后续增量 SQL 需要手动执行或通过迁移流程处理。
  - `internal/logic/site/health.go:80-93` 只能检测 schema 是否满足运行时要求，不能执行迁移、记录版本或协调并发部署。
- **影响**：升级路径依赖操作者记忆和顺序；漏执行脚本时 API 会因 readiness 失败而不可用。手动 SQL 与代码版本错配时，可能出现半升级状态。健康检查能阻止服务误上线，但不能代替迁移编排。
- **建议动作**：
  - 选定一个正式迁移机制，或将现有 SQL 统一转为带唯一版本号的迁移目录。
  - 增加迁移版本表、执行日志、MySQL advisory lock/等价分布式锁和幂等策略。
  - 明确迁移由独立一次性 job 执行，还是由 API 启动前执行；不要让多副本 API 同时竞争迁移。
  - 为破坏性变更定义备份、回滚/前向修复和兼容窗口；涉及大表索引时记录耗时和锁影响。
  - 将“全新数据库初始化”和“从上一生产版本升级”加入 CI，至少覆盖当前全部增量脚本。
- **验收标准**：
  - 新卷可从零创建并完成最新 schema，旧版本数据卷可通过一次部署升级到最新版本。
  - 两个 API 副本并发启动时只有一个迁移执行者，另一个等待并在完成后正常启动。
  - 漏迁移时 readiness 返回可定位的版本差异；迁移完成后自动恢复 healthy。
  - CI 能验证迁移顺序、重复执行幂等性和至少一条失败恢复路径。

### TODO-P1-02 统一 Redis 密码的 Compose 服务端与 API 客户端配置

- [x] **优先级**：P1；**负责人建议**：运维 + 配置维护；**预估**：小到中等。
- **问题**：API 接收 `APP_REDIS_PASSWORD`，但本地 Compose 启动 Redis 的 command 没有配置 `requirepass`；配置文件只保证客户端一侧可能携带密码。
- **证据**：
  - `docker-compose.yml:69-71` 将 `APP_REDIS_PASSWORD` 注入 API。
  - `docker-compose.yml:156-165` 的 Redis command 只有 `--appendonly yes`，healthcheck 也直接执行未认证的 `redis-cli ping`。
  - `.env.example:39-42` 暴露了 Redis 地址、密码和 DB 配置，但没有说明本地 Compose 需要如何同步 Redis 服务端认证。
- **影响**：如果部署者为 API 设置非空 `APP_REDIS_PASSWORD`，而继续使用当前 Compose Redis 容器，API 的 Redis `PING` 和业务请求可能认证失败，最终 `/healthz` 返回 `503`。反过来，如果只修改 Redis 服务端而没有同步 API，也会造成同样问题。
- **建议动作**：
  - 明确 Compose 内置 Redis 与外部 Redis 两种模式；内置模式下用同一变量生成 `--requirepass` 和 healthcheck 的 `-a` 参数。
  - 对密码为空的开发模式与非空的生产模式分别给出显式配置示例，避免 `redis-cli` 把认证配置误判为健康。
  - 启动前校验地址是否指向 Compose 内置服务；外部 Redis 模式不要复用内置服务的隐含假设。
  - 在 CI 中测试空密码、正确密码、错误密码和 Redis 重启后的重连。
- **验收标准**：
  - 使用 `.env.example` 的内置服务配置能稳定 healthy。
  - 设置非空密码后，Redis 服务端、healthcheck、API 客户端三处认证一致。
  - 错误密码能在启动日志和 `/healthz` 中明确定位，不出现无限重试但无结论的状态。

## 5. P2 TODO：应进入近期迭代

### TODO-P2-01 增加可选依赖开关与 Compose profile 的一致性校验

- [x] **优先级**：P2；**负责人建议**：运维 + 后端；**预估**：小到中等。
- **问题**：RabbitMQ 与 Meilisearch 使用 Compose profile 控制容器是否创建，API 又分别使用 `APP_RABBITMQ_ENABLED`、`APP_SEARCH_ENABLED` 控制能力；当前主要依赖 README 和操作者手工保持一致。
- **证据**：
  - `docker-compose.yml:72-80` 注入搜索与 RabbitMQ 开关/连接配置。
  - `docker-compose.yml:184-220` 将 RabbitMQ 放在 `messaging` profile、Meilisearch 放在 `search` profile。
  - `.env.example:29` 提供 `COMPOSE_PROFILES`，`.env.example:44-54` 通过注释说明启用 RabbitMQ/搜索时需同步 profile。
  - `internal/mq/event.go:54-60、92-113、153-179` 对 RabbitMQ 连接失败有重连和同步落库降级；搜索逻辑也有 MySQL fallback，因此错误配置未必立即暴露为页面错误，但可能持续刷连接错误日志并增加延迟。
- **影响**：用户打开功能开关却忘记 profile 时，服务行为依赖降级路径；可选能力“看起来已打开”，实际使用的却是 fallback。RabbitMQ profile 开启但未配置强密码/URL 时也可能启动失败或连接失败。
- **建议动作**：
  - 在 Compose 校验脚本中检查：能力开关、`COMPOSE_PROFILES`、连接 URL、凭据是否成组出现。
  - API 启动时记录明确的 `enabled / disabled / unavailable` 状态，并在管理端系统工具中区分“关闭”和“配置错误”。
  - README、`.env.example`、部署脚本保持同一组可复制命令；当前文档已有说明，但缺少自动化防错。
- **验收标准**：
  - `APP_SEARCH_ENABLED=true` 未启用 `search` profile 时，部署前检查直接给出可操作错误，或 API 明确降级且状态可见。
  - RabbitMQ 的 profile、URL、用户、密码缺一项时不进入“半启用”状态。
  - 可选服务故障时，文章搜索和操作日志的 fallback 行为分别有集成测试和告警指标。

### TODO-P2-02 统一 API 描述、后端类型和 Cookie-only Refresh Token 契约

- [x] **优先级**：P2；**负责人建议**：后端 + 前端；**预估**：小。
- **问题**：Refresh Token 实际由 HttpOnly Cookie 携带，handler 会把响应体中的 `refreshToken` 清空，但 API 描述中的 `TokenPair.refreshToken` 仍是普通必填字段。
- **证据**：
  - `api/notes-of-ashen.api:98-102` 将 `TokenPair.refreshToken` 定义为非 optional。
  - `internal/handler/auth/auth.go:17-23` 写 Cookie 后将 `resp.RefreshToken` 设为空字符串。
  - `frontend/src/types/index.ts:19-25` 已将前端 `refreshToken` 设为可选。
  - `docs/API.md:211-228` 已注明 Cookie-only 语义和空响应字段。
- **影响**：依赖 API 描述生成客户端、SDK 或接口校验器时，会误以为响应必然有可用的 Refresh Token；不同调用方可能错误地把空字符串当作长期凭证，或重新把长期凭证暴露到 JS 可见响应体。
- **建议动作**：
  - 选择一个正式契约：将响应字段标记为 optional/明确为空，或拆分 `CookieTokenResp` 与 `TokenPair`，避免名称暗示响应体始终包含 refresh token。
  - 将 API 描述、`internal/types`、前端类型和 `docs/API.md` 放入同一份契约检查。
  - 保留对 Cookie 缺失、body 兜底、登出幂等和 `withCredentials` 的测试。
- **验收标准**：
  - 新客户端根据契约不会依赖响应体中的 Refresh Token。
  - 注册、登录、刷新、登出四条路径的 Cookie 属性、响应 JSON 和文档一致。
  - API 契约检查能阻止未来将 `refreshToken` 改回必填明文响应。

### TODO-P2-03 修正前端统一响应类型的可选 data 语义

- [x] **优先级**：P2；**负责人建议**：前端；**预估**：小。
- **问题**：后端无数据成功响应会省略 `data`，但前端 `BaseResp<T>` 仍把 `data` 定义为必填；代码注释承认了实际差异，并通过调用约定规避了运行时问题。
- **证据**：
  - `frontend/src/types/index.ts:263-269`：`BaseResp<T>.data` 为必填，同时注释说明 NoData 会省略 data。
  - `frontend/src/utils/http.ts` 和多个 mutation API 依赖统一响应处理，但类型没有区分“有数据成功”和“NoData 成功”。
  - `internal/response` 的成功/无数据响应约定见项目 API 文档。
- **影响**：类型不能准确表达 API 契约；新页面可能在 `code === 0` 时无条件读取 `data`，导致运行时异常。当前测试通过只说明已有调用方没有触发该问题，不代表类型安全。
- **建议动作**：
  - 定义 `BaseResp<T> = { code; message; data: T }` 与 `NoDataResp = { code; message; data?: never }`，或把 `data` 改为可选后在请求封装层做类型收窄。
  - 将 mutation API 的返回类型改成 `NoDataResp`，查询 API 保留 `BaseResp<T>`。
  - 增加 `data` 缺失的成功响应测试，覆盖 Axios 统一处理和错误响应解析。
- **验收标准**：
  - TypeScript 能在查询和 mutation 两类 API 中正确提示错误用法。
  - 所有现有调用方通过 type-check，且 NoData 响应不会被误判为错误。

### TODO-P2-04 建立数据库关系完整性策略

- [x] **优先级**：P2；**负责人建议**：数据层；**预估**：中等，需评估兼容性。
- **问题**：`schema.sql` 没有发现 `FOREIGN KEY`、`REFERENCES` 或级联约束；文章、标签、项目、用户、Token 和日志之间的关系主要依靠业务 SQL 手工维护。
- **证据**：
  - `deploy/mysql/schema.sql:106-228` 定义 categories、tags、articles、article_versions、article_tags、article_likes 等关系表，但没有数据库外键声明。
  - `model/taxonomy.go:335-349`、`model/article.go:277-279、770-778` 显示删除分类/标签/文章关联时由代码显式清理。
  - `model/backup.go:274-325` 也依赖固定删除/插入顺序来维持恢复过程的关系一致性。
- **影响**：直接 SQL 操作、异常导入、未来新增调用方或部分事务失败可能产生孤儿关系；统计、备份恢复和后台列表可能出现数量不一致。当前业务路径有不少手工清理测试，但数据库本身无法阻止非法状态。
- **建议动作**：
  - 逐表评估是否能安全增加外键和 `ON DELETE` 策略；文章分类可能需要 `SET NULL`，关联表通常适合 `CASCADE`，审计日志可能适合保留用户删除后的 NULL。
  - 如果因备份/迁移兼容性不能加外键，增加定期 orphan audit、修复命令和部署前一致性检查。
  - 用真实 MySQL 集成测试验证新增约束、删除语义和备份恢复顺序。
- **验收标准**：
  - 生产 schema 对关键关系有明确的数据库约束或明确的巡检/修复机制。
  - 删除分类、标签、文章、用户后的关系行为有书面规则和自动化测试。
  - 备份恢复、重复恢复和异常中止不会留下可见孤儿数据。

### TODO-P2-05 统一 Visitor ID、IP/UA 与 UV 统计口径

- [x] **优先级**：P2；**负责人建议**：后端 + 产品数据；**预估**：中等。
- **问题**：前端为点赞去重发送持久化 `X-Visitor-Id`，但流量 UV 的 visitor hash 仍只由日期、IP、User-Agent 组成；同一 NAT 出口下的多个用户可能被合并。
- **证据**：
  - `frontend/src/utils/http.ts:96-100` 全局注入 `X-Visitor-Id`。
  - `frontend/src/utils/visitor.ts:1-23` 说明该 ID 用于点赞等场景。
  - `internal/logic/traffic/traffic.go:63-70` 记录流量时调用 `visitorDailyHash(date, meta.IP, meta.UserAgent)`。
  - `internal/logic/traffic/traffic.go:159-161` 的 hash 输入没有 Visitor ID。
- **影响**：后台今日 UV、趋势 UV 和内容 UV 与产品对“访客”的直觉可能不一致；代理、企业网络、移动网络和 UA 变化场景下误合并或误拆分。若直接信任客户端 Visitor ID，又会引入清缓存/伪造计数的问题，不能简单把 header 直接当唯一事实来源。
- **建议动作**：
  - 先明确三种口径：统计 UV、点赞去重、反滥用识别，不要共用一个未经定义的 ID。
  - 选择服务端可解释的组合策略，例如可信代理 IP + UA + 签名/匿名 cookie，并记录隐私与保留期限。
  - 对 `X-Visitor-Id` 做长度、字符集和来源策略校验；不要把任意客户端值直接作为高信任安全依据。
  - 增加 NAT、多 UA、清 Cookie、代理头不可信、跨日和并发访问测试。
- **验收标准**：
  - 产品文档明确 PV/UV 的定义、时区、去重窗口和代理处理规则。
  - 后台展示值与数据库明细/Redis 聚合口径一致，跨实例结果稳定。
  - 点赞防刷与流量统计互不意外影响，且不记录不必要的可识别信息。

### TODO-P2-06 明确搜索建议的分类/标签来源展示

- [x] **优先级**：P2；**负责人建议**：前端；**预估**：小。
- **问题**：搜索建议 API 已返回 `kind`，但建议列表 UI 对分类和标签只显示同名文字与 `articleCount`，没有显示“分类”或“标签”来源。在实际运行态中，分类 `Go` 与标签 `Go` 都显示为类似 `Go · 2` 的候选项，用户难以区分点击后的筛选含义。
- **证据**：
  - `internal/types/types.go:338-346`、`api/notes-of-ashen.api:448-456` 已定义 `kind` 为 `article/category/tag`。
  - `internal/logic/search/search.go:45-75` 按文章、分类、标签混合返回建议。
  - `frontend/src/pages/Search.tsx:327-340` 只渲染 `item.label` 和数量，没有渲染 `item.kind` 的可见来源标识。
  - `frontend/src/pages/Search.tsx:347-369` 的过滤按钮虽用 `#` 区分标签，但输入建议列表没有同等区分。
- **影响**：同名分类和标签产生重复候选，键盘导航和无障碍用户更难理解结果；点击后分别进入 `categoryId` 或 `tagId`，行为差异不够可见。
- **建议动作**：
  - 在建议项中显示国际化来源标签或使用稳定图标加可访问文本，例如“分类 / 标签 / 文章”。
  - 保持 `kind` 为联合类型，并对未知 kind 做安全 fallback。
  - 增加同名分类/标签、文章同名、键盘选择和屏幕阅读器 label 测试。
- **验收标准**：
  - 同名分类和标签在视觉、ARIA 名称和键盘导航中都可区分。
  - 点击每一种建议后 URL 参数正确，且返回列表标题/筛选状态能说明当前来源。

## 6. P3 TODO：持续改进

### TODO-P3-01 增加真实移动端 Playwright 项目

- [x] **优先级**：P3；**负责人建议**：前端测试；**预估**：小到中等。
- **问题**：`frontend/playwright.config.ts:37-42` 只有 `chromium` + `Desktop Chrome`，没有 `Pixel 7`、`iPhone` 或等价移动 viewport 项目。本次浏览器手工调整 viewport 也未形成可靠的 CSS 移动端验证。
- **影响**：首页、搜索建议、文章详情、后台表格和编辑器可能在窄屏发生横向溢出、按钮换行或触控区域问题，而桌面测试无法覆盖。
- **建议动作**：增加至少一个 Chromium 移动项目和一个 WebKit/Safari 兼容项目，覆盖首页、搜索、文章详情、登录以及后台核心表格；在 CI 保存失败截图和 trace。
- **验收标准**：390px 左右 CSS viewport 下无横向滚动，表单/弹层/建议列表不遮挡内容，关键操作可通过触控语义完成。

### TODO-P3-02 持续治理前端首屏与大资源 chunk

- [x] **优先级**：P3；**负责人建议**：前端；**预估**：中等。
- **现状证据**：本次 `pnpm build` 通过，但产物中 `echarts` 约 500.81 KiB、Markdown 约 423.48 KiB、syntax highlighter 约 95.69 KiB；项目的 `check:bundle-size` 当前阈值允许通过。
- **影响**：低带宽设备首次进入后台分析、文章详情或编辑器时可能下载较大的 JS；路由懒加载已经缓解部分问题，但仍应以真实网络指标确认。
- **建议动作**：确认这些 chunk 是否只在对应路由按需加载；对 Markdown/高亮语言按需加载，评估 ECharts 按页面拆分；建立 gzip/brotli 大小预算和 Lighthouse/Web Vitals 监测。
- **验收标准**：公开首页不加载后台分析和编辑器资源；预算有明确阈值、CI 失败信息和例外审批机制；在 4G/低端设备下首屏指标有基线。

### TODO-P3-03 将扩展故障注入纳入发布门禁

- [x] **优先级**：P3；**负责人建议**：测试 + 运维；**预估**：中等。
- **现状证据**：仓库已有 `test/integration/http_integration_test.go`、`scripts/test-integration.ps1`、`frontend/e2e/critical-path.spec.ts`，`docs/TESTING.md` 还描述了 `core`/`extended` 阶段及 Redis 故障注入；本次只执行了本地单元/静态检查和运行中的只读探查，未执行会重建 Compose 生命周期的扩展套件。
- **影响**：Redis 不可用、迁移漏执行、RabbitMQ profile 缺失、媒体目录异常、备份恢复中途退出等生产级故障路径虽然有部分代码和测试基础，但当前审查没有新鲜执行证据。
- **建议动作**：在隔离的临时 Compose 项目和临时 volume 中周期性运行 `core` 与 `extended`；发布前至少运行核心集成测试，夜间运行恢复失败注入和 Redis fail-closed；保留失败日志、trace、截图并清理临时资源。
- **验收标准**：CI 运行结果可追溯到提交；失败时能保留脱敏诊断产物；测试不会触碰 `127.0.0.1:1270` 的现有部署或真实数据卷。

## 7. 已确认的设计优点与无需立即改动项

这些项目在本次范围内表现良好，暂不作为问题 TODO：

- `healthz` 同时检查 DB、Redis 和运行时 schema，schema 缺失时返回 `503`，比只做进程存活检查更适合作为 readiness。
- RabbitMQ 关闭或发布失败时会同步写入 `operation_logs`；搜索服务不可用时会回退 MySQL，关键功能有明确降级路径。
- Refresh Token 已迁移到 HttpOnly Cookie，前端 Axios 设置了 `withCredentials`；需要做的是契约收敛，不是撤销当前安全方向。
- API 描述文件头部明确说明它不是编译源，运行时类型源是 `internal/types`；这降低了误以为 go-zero 会自动生成路由的风险，但仍建议增加契约漂移检查。
- 文章、媒体、备份、AI 配置和权限相关测试覆盖面较广，且本次 `go test ./...` 与前端 76 项测试均通过。
- 当前 Docker 镜像使用了多项固定 digest；这有利于复现性，后续升级时应配合镜像更新评估和 SBOM/漏洞扫描。

## 8. 推荐实施顺序

1. 先落地 TODO-P1-01，明确生产升级时数据库 schema 如何到达代码所需版本。
2. 同时修复 TODO-P1-02，并用干净临时 Compose 验证密码为空/非空两种部署模式。
3. 加入 TODO-P2-01 的配置一致性检查，避免可选依赖“开关已开但容器没起”的半配置状态。
4. 在下一次 API 变更窗口处理 TODO-P2-02 和 TODO-P2-03，统一前后端类型与 Cookie-only 认证契约。
5. 评估 TODO-P2-04 的数据完整性方案，再决定外键还是 orphan audit；不要直接批量添加级联约束而不先验证备份恢复。
6. 处理搜索建议来源标识和 UV 统计口径，补齐对应回归测试。
7. 将移动端 E2E、资源预算和故障注入加入持续集成与发布门禁。

## 9. 审查结论与风险边界

本次结论是“当前实例运行正常，代码质量和测试基础较好，但升级运维链路仍需要工程化”。最现实的上线风险来自数据库增量迁移和配置组合，而不是已观察到的页面崩溃或 API 基本不可用。

本报告没有替代以下工作：真实生产数据恢复演练、带授权边界的完整安全测试、负载/容量测试、浏览器兼容矩阵、MySQL 大表迁移耗时评估，以及对真实 `.env` 的配置审计。完成 P1 项目前，不建议把“当前容器 healthy”解释为“任意历史版本都可无感升级”。
