# Todo

本文件记录 Notes of Ashen 项目 P0 清理之后的剩余优化项，按优先级排序。P0 已完成（敏感配置 `.env` 注入与启动校验、Redis 限流原子化、首个 admin 并发保护、前端 refresh timeout/失败去重、迁移脚本目标库）。下列事项为后续待办，实施前请先 `git status --short` 确认工作区状态，按 PR 拆分逐步推进。

---

## P1：性能热点与请求链路治理

### 后端

- [ ] **搜索索引同步从写请求链路解耦**
  - 现状：文章创建/更新/改状态/恢复版本后同步 `syncArticleSearch`，`Upsert` 每次都 `ensureIndex`/`configureIndex`，`submitTask` 最长 `waitTask` 2 分钟。
  - 方案：请求内只完成 DB 事务；索引更新转异步事件/后台任务；index settings 在启动或重建时配置一次；允许搜索短暂最终一致。
  - 关键文件：`internal/logic/article/search.go`、`internal/search/client.go`、`internal/logic/article/article.go`。
  - 验证：模拟 Meili 慢/不可用，写接口不应被拖死；事件重试最终一致。

- [ ] **MQ publish 热路径与 consumer 生命周期**
  - 现状：`internal/mq/event.go` publish 在请求中串行持锁；`internal/svc/servicecontext.go` 启动 consumer 无 stop/cancel，`Close()` 未停止 consumer。
  - 方案（分阶段）：① consumer 接收 context，`ServiceContext.Close()` 先停 consumer 再关 DB/Redis；② 引入 outbox 或后台 worker，请求只落库，异步投递 MQ。
  - 关键文件：`internal/mq/event.go`、`internal/svc/servicecontext.go`。
  - 验证：SIGTERM 后无 panic，consumer 停止后再关依赖；RabbitMQ 不可用时写接口不阻塞。

- [ ] **阅读数写热点**
  - 现状：`internal/logic/article/article.go:128-130` 每次详情请求 `UPDATE articles SET view_count = view_count + 1`，热门文章行锁竞争。
  - 方案：Redis 聚合计数 + 定时批量 flush 到 MySQL，失败低频采样日志。
  - 关键文件：`internal/logic/article/article.go`、`model/article.go`。
  - 验证：压测详情访问，DB update QPS 显著下降。

- [ ] **站点设置多 key 查询**
  - 现状：`model/site_settings.go` `SiteSettings()`/`AISettings()` 逐项 `GetStringSetting`/`GetBoolSetting`。
  - 方案：一次性 `SELECT setting_key, setting_value FROM site_settings WHERE setting_key IN (...)`，内存组装默认值；热路径结合缓存。
  - 关键文件：`model/site_settings.go`、`internal/logic/site/cache.go`。
  - 验证：更新站点设置后读取及时一致。

- [ ] **AI HTTP client 每次创建**
  - 现状：`internal/aiclient/client.go` `Assist` 每次新建 `http.Client`/`Transport`。
  - 方案：在 `ServiceContext` 维护可复用 AI client，仅 request 使用 context timeout。
  - 关键文件：`internal/aiclient/client.go`、`internal/svc/servicecontext.go`。
  - 验证：AI 请求超时快速失败，不耗尽连接。

### 前端

（PR 6 已完成：ArticleDetail 滚动隔离、Markdown 预览 debounce、Zustand 精确 selector、请求 AbortSignal、ArticleDetail headings 单次提取，相关条目已移除。）

---

## P2：部署工程化与稳定性

- [ ] **graceful shutdown 顺序**
  - 现状：`cmd/notes-of-ashen/main.go:31-35` 多 defer，关闭顺序不显式。
  - 方案：显式 signal handling，先停 server，再停 consumer，最后关 DB/Redis/MQ。
  - 关键文件：`cmd/notes-of-ashen/main.go`、`internal/svc/servicecontext.go`。

- [ ] **dotenv 健壮性剩余项**
  - 现状：buffer 与 Setenv error 已修，但 `.env` 无法影响配置文件路径本身，转义/多行语义未支持。
  - 方案：明确 dotenv 支持范围；如需更完整语义补测试或引入成熟解析器。
  - 关键文件：`internal/config/dotenv.go`、`internal/config/dotenv_test.go`。

- [ ] **Docker/Compose/Nginx 生产化**
  - 方案：API/Web 镜像改非 root；增加 healthcheck；compose 加日志轮转（max-size/max-file）、资源限制、`stop_grace_period`；Nginx 加静态资源强缓存（`expires 1y; immutable`）、gzip/brotli、安全响应头（`X-Content-Type-Options`/`Referrer-Policy`/CSP）。
  - 关键文件：`Dockerfile.api`、`Dockerfile.web`、`docker-compose.yml`、`deploy/nginx/default.conf`。
  - 验证：`whoami` 非 root；`docker inspect` 有 healthcheck；curl 检查缓存/压缩/安全头。

- [ ] **README 与部署文档同步（剩余项）**
  - 现状：P0 已补密钥/迁移说明；端口说明与当前外部 MySQL/Redis/RabbitMQ 拓扑、部署陷阱（超时一致性、`WEB_PORT` 仅 127.0.0.1）仍需复核。
  - 方案：补「部署陷阱与超时一致性」段落，更新本地开发/Docker/1Panel 端口与外部依赖说明。
  - 关键文件：`README.md`、`docs/API.md`。

- [ ] **CI/CD 与依赖治理**
  - 方案：新增 `.github/workflows/`：`go test ./...`、`go vet`、`govulncheck`、前端 `pnpm lint`/`pnpm build`、Docker build、secret scan；修正 `go.mod` 中未来日期 genproto 伪版本；后续引入迁移验证与依赖更新机器人。
  - 关键文件：`go.mod`、`go.sum`、`frontend/package.json`、新增 `.github/workflows/`。

- [ ] **迁移脚本版本化与回滚**
  - 现状：`deploy/mysql/` 全部 `add_xxx.sql`/`drop_xxx.sql` 字典序混排，无版本前缀，无 down。
  - 方案：重命名 `0001_xxx.up.sql`/`0001_xxx.down.sql`；引入 `golang-migrate`/`goose`；README 列执行顺序。
  - 关键文件：`deploy/mysql/`。

- [ ] **仓库根残留排查**
  - 现状：`notes_of_ashen_backup.sql`、`notes-of-ashen.exe` 未被 Git 跟踪（已在 `.gitignore`），但 Git 历史是否含需核实。
  - 方案：`git log --stat` 核查历史；若已提交用 `git filter-repo` 清理并轮换相关凭据。

---

## P3：体验、可访问性与可维护性

（前端可访问性 7 项已随 PR 6 完成：ImageLightbox 焦点陷阱、CaptchaField 标签关联、TaxonomyCombobox 方向键与 ARIA、Pagination type/aria-current、RequestProgressBar useReducedMotion、useCountdown timer 清理、Register 发码前邮箱校验，相关条目已移除。）

### 类型与组件复用

- [ ] **role/status 收敛 union**：定义 `UserRole`/`UserStatus`/`ArticleStatus` union，API 与页面复用。`frontend/src/types/index.ts`、`frontend/src/types/api.ts`。
- [ ] **Home/Search 文章卡片复用**：抽 `ArticleCard`/`ArticleListItem`，统一封面/分类/置顶/日期/阅读数/预加载。`frontend/src/pages/Home.tsx`、`frontend/src/pages/Search.tsx`。
- [ ] **http 去重 key 纳入 params**：`buildDedupeKey` 加入 `config.params` 稳定序列化；`stableStringify` 特判 Date/FormData 等。`frontend/src/utils/http.ts`。

### 后端可观测性与健壮性

- [ ] **统一错误日志带上下文**：`internal/response/response.go` 500 错误记录 request id/path/method/err；缓存/AI/搜索失败采样记录。
- [ ] **ServiceContext 启动依赖策略**：Redis ping 校验；可选依赖降级、必需 fail-fast；健康检查接口暴露 DB/Redis/MQ/Search 状态。
- [ ] **refresh token 轮换一致性**：以 DB 为准，Redis 仅缓存，删除失败记 warn；并发刷新用唯一约束 + CAS。
- [ ] **认证中间件用户状态缓存**：`internal/middleware/auth.go` 每请求 `FindUserByID`，加短 TTL 缓存，禁用/改角色时失效。
- [ ] **articleTags/articleResp 错误处理**：`internal/logic/article/article.go:614-623` 静默吞错，列表记 warn，详情/管理返回错误。
- [ ] **createArticleVersion 并发**：`model/article.go` `MAX(version_no)+1` 改 `SELECT ... FOR UPDATE` 或维护版本号字段。
- [ ] **RSS/Sitemap 复用缓存**：`internal/logic/site/feed.go` 直接打 DB，复用 `cachedSiteSettings` 等并加短 TTL。
- [ ] **标签反向索引**：`article_tags` 增 `(tag_id, article_id)` 索引；`project_tags` 同步检查。

### 后端安全增强

- [ ] **限流 fail-closed 策略**：登录/验证码等敏感接口 Redis 失败时 fail-closed 或降级（当前 fail-open）。
- [ ] **refresh token 迁移 HttpOnly Cookie**：需后端配合 CSRF/HTTPS/SameSite，与前端、Nginx、域名联动，作为认证专项分阶段实施。

### 文案与文档

- [ ] **i18n 覆盖集中化**：`Layout.tsx`/`ArticleDetail.tsx`/`AdminLayout.tsx`/`Articles.tsx`/`ArticleEditor.tsx` 页面级散落文案逐步迁入 `i18n.ts`。
- [ ] **docs/API.md 与 api 描述一致性**：考虑 `goctl api doc`/`swag` 自动生成，CI 比对。

---

## 建议 PR 拆分（剩余）

1. **PR 2：并发正确性与认证稳定性补充** — 限流 fail-closed、refresh token 一致性测试补全。
2. **PR 4：搜索与 MQ 解耦** — Meili 异步化、consumer 生命周期、graceful shutdown。
3. **PR 5：部署生产化** — Docker 非 root、healthcheck、compose 日志/资源、Nginx 缓存压缩安全头、README 完整部署说明。
4. **PR 7：工程化与维护** — CI/CD、迁移版本化、依赖治理、类型 union、文章卡片复用。

> 已完成（不再列入）：PR 3 文章查询 N+1 修复与批量加载（工作区改动）、PR 6 前端性能与可访问性（本次提交）、useCountdown 清理、SearchHighlight 优化。

## 待确认决策

- 搜索索引是否接受最终一致（发布后短时间搜不到）。
- 阅读数是否允许延迟落库。
- 是否关闭普通用户注册或改邀请制。
- 生产是否具备 HTTPS，能否启用 Secure Cookie（影响 refresh token Cookie 迁移）。
- 是否引入统一监控告警。
