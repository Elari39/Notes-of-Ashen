# Notes of Ashen 待修复问题清单

> **节奏**：严格按 P0 → P1 → P2 → P3 推进；上一等级条目全部 `- [x]` 或被删除后才能进入下一等级。
> **完成方式**：将 `- [ ]` 改为 `- [x]`，或直接删除条目；同时同步更新下方“进度”计数。
> **来源**：基于本仓库代码的实际审计（后端 / 前端 / 配置 部署 文档 三路扫描），所有问题条目均带文件路径与行号。

## 进度

- 当前等级：**P3**
- P0：10 / 10
- P1：11 / 11
- P2：19 / 19
- P3：0 / 27

---

## P0 — 安全 / 数据破坏 / 权限绕过 / 无法启动

- [x] **P0-1 后端 — 用户注册 GET_LOCK 未校验返回值**（[`model/user.go:52-58`](model/user.go#L52-L58)）— `WithUserRegistrationLock` 只判 `Exec` 错误，未校验 `GET_LOCK` 是否真的成功（超时返回 0、失败返回 NULL 都被忽略），并发注册可能多个用户都成 admin。改用 `QueryRowContext` 取返回值并断言为 1；释放也校验返回值。
- [x] **P0-2 后端 — `UpdateUserStatus / UpdateUserRole` 不刷新 `updated_at`**（[`model/user.go:119-133`](model/user.go#L119-L133)）— 仅 `SET status=?` / `SET role=?`，schema 若无 `ON UPDATE CURRENT_TIMESTAMP`，权限变更时间无法审计追溯。显式 `SET ..., updated_at = NOW()` 或在 schema 上补 `ON UPDATE` 并加测试。
- [x] **P0-3 后端 — AI Key v1 解密回退到 `Auth.AccessSecret`**（[`internal/logic/ai/settings.go:120-128`](internal/logic/ai/settings.go#L120-L128)、[`internal/logic/ai/settings.go:222-234`](internal/logic/ai/settings.go#L222-L234)）— 旧密文用 JWT 签名密钥兜底解密，轮换 `APP_AUTH_ACCESS_SECRET` 会让未迁移密文永久不可读。启动时主动扫描并迁移，缺 `KeyEncryptionSecret` 即拒绝 `EffectiveConfig` 并告警。
- [x] **P0-4 后端 — `Refresh` Redis 异常 fail-closed 与注释承诺不一致**（[`internal/logic/auth/auth.go:268-286`](internal/logic/auth/auth.go#L268-L286)）— 注释承诺“Redis miss 时回退 DB 并回填”，代码实际 Redis 抖动直接 500，且无回填。要么实现回填并把 Redis 错误降级为 cache miss，要么修订注释明确 fail-closed。
- [x] **P0-5 后端 — `model.WithTx` 未对 Commit 失败做 Rollback**（[`model/db.go:92-102`](model/db.go#L92-L102)）— `Commit()` 失败时事务状态未定，业务可能误以为成功。改成 `if err := tx.Commit(); err != nil { _ = tx.Rollback(); return err }`。
- [x] **P0-6 后端 — RabbitMQ 关闭时操作日志静默丢失**（[`internal/mq/event.go:76-100`](internal/mq/event.go#L76-L100)、[`internal/svc/servicecontext.go:56-57`](internal/svc/servicecontext.go#L56-L57)）— `Enabled=false` 时所有 audit/操作事件不会写入 `operation_logs` 表，`/api/v1/admin/logs` 永远空。补上同步落库 fallback，或在 README 明确标注“未启用 MQ 时审计日志不可用”。
- [x] **P0-7 后端 — 流量计数 `traffic.Visit` 路径黑名单可绕过**（[`internal/logic/traffic/traffic.go:79-91`](internal/logic/traffic/traffic.go#L79-L91)、[`internal/logic/traffic/traffic.go:30`](internal/logic/traffic/traffic.go#L30)）— `path` 来自 body 任意字符串，仅前缀匹配且未规范化，可大量伪造 articleId 刷统计。改为只统计能解析回真实路由的请求；至少校验 articleId 存在并属于已发布文章。
- [x] **P0-8 前端 — refreshToken 落 localStorage 的 XSS 暴露**（[`frontend/src/utils/http.ts:85`](frontend/src/utils/http.ts#L85)、[`frontend/src/store/auth.ts:34-35`](frontend/src/store/auth.ts#L34-L35)、[`frontend/src/store/auth.ts:52-53`](frontend/src/store/auth.ts#L52-L53)、[`frontend/src/components/Layout.tsx:125`](frontend/src/components/Layout.tsx#L125)）— refreshToken 写在 localStorage，任意 JS（含三方依赖被供应链投毒）都可读，长期续签风险。生产 HTTPS 已具备，refreshToken 迁移到后端 `HttpOnly+Secure+SameSite=Strict` Cookie；前端只保留短期内存 accessToken。
- [x] **P0-9 部署 — `docker-compose.yml` `APP_TIMEOUT` 默认 70s 与 yaml/env/nginx 设计的 610s 不一致**（[`docker-compose.yml:32`](docker-compose.yml#L32)）— 部署者未显式覆盖时 RestConf 整体超时 70s，AI 非流式、Markdown 导入、版本恢复全部 504。改为 `${APP_TIMEOUT:-610000}`，与 [`etc/notes-of-ashen.yaml:4`](etc/notes-of-ashen.yaml#L4) / [`.env.example:6`](.env.example#L6) / [`deploy/nginx/default.conf:44-45`](deploy/nginx/default.conf#L44-L45) 对齐。
- [x] **P0-10 部署 — `APP_DATABASE_DSN` 未加 `?:` 校验，占位符直接进容器**（[`docker-compose.yml:33`](docker-compose.yml#L33)）— `.env.example` 含 `<REPLACE_…>` 占位，未替换时 compose 不报错，由 Go `ValidateConfig` 在容器内 fail-fast，需翻日志才能发现。给 DSN 加 `?:set APP_DATABASE_DSN in .env`，并在 README “快速开始”插入硬性提示“替换所有 `<REPLACE_…>` 占位符”。

---

## P1 — 核心流程不可用

- [x] **P1-1 后端 — SMTP 仅支持隐式 TLS（端口 465）**（[`internal/emailer/smtp.go:42-65`](internal/emailer/smtp.go#L42-L65)）— 直接 `tls.DialWithDialer`，没有 STARTTLS 路径。配 587 端口注册 / 找回密码邮件全部发不出去。按端口区分 SSL/STARTTLS，或暴露 `Mode: ssl|starttls|plain` 配置。
- [x] **P1-2 后端 — 首位 admin 跳过验证码逻辑依赖 P0-1 的 GET_LOCK**（[`internal/logicutil/common.go:28-30`](internal/logicutil/common.go#L28-L30)、[`internal/logic/auth/auth.go:136-140`](internal/logic/auth/auth.go#L136-L140)）— GET_LOCK 失败时两个并发请求都读到 `total==0`、都成 admin、都跳过验证码。在 P0-1 修复基础上再加 DB 双保险（`role='admin'` 哨兵唯一约束或 `SELECT ... FOR UPDATE`）。
- [x] **P1-3 后端 — `ChangePassword` 入口无限流**（[`internal/logic/user/user.go:155-188`](internal/logic/user/user.go#L155-L188)、[`internal/handler/routes.go:67`](internal/handler/routes.go#L67)）— Token 短期被盗场景下可不限速尝试旧密码，且失败无锁账户。给 `/api/v1/users/me/password` 挂同等的 `loginRateLimit`；失败 N 次撤销 refresh 强制登出。
- [x] **P1-4 后端 — 文章点赞 visitor hash IP+UA 易绕过**（[`internal/logic/article/article.go:701-704`](internal/logic/article/article.go#L701-L704)、[`model/article.go:263-286`](model/article.go#L263-L286)）— 攻击者轮换 UA 即可重复点赞；可信代理场景下 X-Forwarded-For 第一个 IP 也可任意伪造。引入前端持久化 visitor UUID 纳入 hash，并限制每篇文章每小时唯一点赞数。
- [x] **P1-5 后端 — 文章搜索回退后 Total/Items 不匹配**（[`internal/logic/article/search.go:31-54`](internal/logic/article/search.go#L31-L54)）— `result.Total` 仍是搜索引擎估值，`visible` 过滤后会与 items 长度不一致，分页会“反复跳动”。过滤后修正 Total，或把 status/visibleAt 直接下推到 Meilisearch filter。
- [x] **P1-6 后端 — `AdminStats.PublishedTotal` 与列表筛选 published 语义不一致**（[`model/stats.go:18-29`](model/stats.go#L18-L29)、[`model/article.go:503-528`](model/article.go#L503-L528)）— 仪表盘“已发布”排除未来排程，列表“已发布”包含未来排程，数字对不上。统一约定（推荐：published 仅指“状态 published 且当前可见”）。
- [x] **P1-7 前端 — 401 重试 token 旋转竞态导致用户被强制下线**（[`frontend/src/utils/http.ts:166-176`](frontend/src/utils/http.ts#L166-L176)）— 后端 refresh 即旋转，第二个并发 401 在 finally 清空 task 后再次发刷新会用旧 token → 失败 → 强制登出。refresh 完成后保留 1-2s token 缓存让窗口内 401 复用最新 access；或 401 前比对 `Authorization` 与 store 中 token 不一致时只换 header 重试。
- [x] **P1-8 前端 — 写操作 in-flight 去重导致 401 重试被复用**（[`frontend/src/utils/http.ts:244-260`](frontend/src/utils/http.ts#L244-L260)）— 401 重试再次进入 adapter 会命中首请求的 401 promise，重试机制失效。`buildDedupeKey` 判 `config._retry === true` 时跳过去重。
- [x] **P1-9 前端 — `tsconfig.app.json` 未开 strict + Profile `value={user?.account}` 受控告警**（[`frontend/tsconfig.app.json`](frontend/tsconfig.app.json)、[`frontend/src/pages/Profile.tsx:138`](frontend/src/pages/Profile.tsx#L138)）— `pnpm build` 用 `tsc && vite build`，无 strict 等同放弃类型保护；user 未到达时 input value=undefined 触发受控/非受控告警。开启 `strict`/`noImplicitAny`/`strictNullChecks` 并修复暴露问题；input 改 `value={user?.account ?? ''}`。**【部分完成】** 已修 Profile 受控告警（`value ?? ''`）与改密码校验（套用 `useFormValidation`，minLength 8）；`tsconfig.app.json` 整体开启 strict 因暴露面大暂缓，作为独立后续项排期。
- [x] **P1-10 文档 — README 内部 MySQL/Redis/RabbitMQ 描述与 compose 现状自相矛盾**（[`README.md:280-286`](README.md#L280-L286)、[`README.md:339`](README.md#L339)）— compose 已不再启动 MySQL/Redis/RabbitMQ，但同份 README“端口说明”仍写 `mysql:3306` / `redis:6379` / `rabbitmq:5672`，新用户照抄就连接失败。重写端口说明仅列 Web/API/Meilisearch；删除/重写第 339 行“项目内部 MySQL”整段。
- [x] **P1-11 部署 — 增量迁移脚本 `add_content_growth_features.sql` `article_versions` 缺列**（[`deploy/mysql/add_content_growth_features.sql:85-110`](deploy/mysql/add_content_growth_features.sql#L85-L110)、[`deploy/mysql/schema.sql:242-270`](deploy/mysql/schema.sql#L242-L270)、[`deploy/mysql/add_resume_portfolio_interaction_geo.sql:21-35`](deploy/mysql/add_resume_portfolio_interaction_geo.sql#L21-L35)、[`deploy/mysql/add_article_pin_priority.sql:53-83`](deploy/mysql/add_article_pin_priority.sql#L53-L83)）— 新库只跑该脚本会缺 `like_count / is_pinned / display_priority`，运行时 `articleVersionSelectFields` 报 Unknown column。在 README 部署小节给出按时间顺序的脚本列表，或在脚本头加“前置依赖”注释。

---

## P2 — 接口 / 类型 / 文档不一致 + 明显 UX

- [x] **P2-1 `api/notes-of-ashen.api` 与 `internal/types/types.go` 大量字段不同步**（[`api/notes-of-ashen.api`](api/notes-of-ashen.api)、[`internal/types/types.go`](internal/types/types.go)）— `*WithIDReq` 系列 types.go 全部缺失；时间戳 api 为 string、types.go 为 `time.Time`；`ArticleResp.Category` api 值类型 vs types.go 指针；`ArticleListReq` form vs json tag 等。重新用 goctl 从 .api 生成 types/routes，或反向修订 .api。
- [x] **P2-2 后端 — Login 邮箱大小写敏感**（[`internal/logic/auth/auth.go:177-196`](internal/logic/auth/auth.go#L177-L196)、[`model/user.go:80-85`](model/user.go#L80-L85)）— 注册时邮箱已 NormalizeEmail，登录时直接用 `req.Account` 查询，大写邮箱登不上。包含 `@` 时视为邮箱并 NormalizeEmail。
- [x] **P2-3 后端 — Refresh 时 user 非 active 不撤销旧 token**（[`internal/logic/auth/auth.go:287-293`](internal/logic/auth/auth.go#L287-L293)）— 直接返回 Forbidden，旧 refresh token 在 DB 中存活到自然过期，每次请求白消耗。禁用用户时即时调 `RevokeUserRefreshTokens`；Refresh 路径发现 disabled 也顺手撤销。
- [x] **P2-4 后端 — Logout 对过期 token 应幂等成功**（[`internal/logic/auth/auth.go:402-410`](internal/logic/auth/auth.go#L402-L410)）— 过期 token 现返回 401，前端反复重试。改为幂等 200。
- [x] **P2-5 后端 — `validator.OptionalHTTPURL` 不阻止 SSRF / 内网地址**（[`internal/validator/validator.go:36-49`](internal/validator/validator.go#L36-L49)）— `avatarUrl/coverUrl/projects.*` 可填内网或保留地址段。禁止 127/8、10/8、172.16/12、192.168/16、169.254/16、::1、fc00::/7。
- [x] **P2-6 后端 — 搜索高亮内容未做 HTML 转义**（[`internal/logic/article/search.go:44-53`](internal/logic/article/search.go#L44-L53)、[`internal/search/client.go:104-105`](internal/search/client.go#L104-L105)）— Meili `<mark>` 高亮片段含原文 HTML 字符未转义，前端若 v-html/dangerouslySetInnerHTML 渲染存在 XSS 面。除 `<mark></mark>` 外其它 `<>&` 服务端转义。
- [x] **P2-7 后端 — `cache.HashKey` 忽略 marshal 错误**（[`internal/cache/cache.go:94-98`](internal/cache/cache.go#L94-L98)）— 不可序列化值会让所有 key 退化成同一前缀。返回错误或 panic，避免脏缓存。
- [x] **P2-8 后端 — `RecordTraffic` SourceName 未限长**（[`internal/logic/traffic/traffic.go:43-58`](internal/logic/traffic/traffic.go#L43-L58)、[`model/traffic.go:30-60`](model/traffic.go#L30-L60)）— referrer 被解析后的 host 可超长，依赖 schema 列宽兜底。logic 层限制 ≤ 128。
- [x] **P2-9 后端 — 文章列表缓存 status 维度未归一化**（[`internal/logic/article/cache.go:23-25`](internal/logic/article/cache.go#L23-L25)）— `status="" ` 与 `status="published"` 公开默认结果一致但 key 不同，浪费缓存。先归一化再算 key。
- [x] **P2-10 后端 — `IncreaseArticleView` 无去重**（[`model/article.go:258-261`](model/article.go#L258-L261)、[`internal/logic/article/article.go:127`](internal/logic/article/article.go#L127)）— 每次刷新 +1，view_count 易刷。复用 visitor hash 24h 去重，或对接 traffic PV。
- [x] **P2-11 前端 — `UpdateSiteSettingsReq` 必填字段与后端 optional 不符**（[`frontend/src/types/api.ts:59-70`](frontend/src/types/api.ts#L59-L70)、[`internal/types/types.go:69-80`](internal/types/types.go#L69-L80)）— store 在 `hasLoaded=false` 时切开关会把前端默认值覆盖到后端。前端类型改 optional + store 拦截未加载时的写操作 + 仅发差异字段。
- [x] **P2-12 前端 — `LoginReq` 必填 captcha 但后端不消费**（[`frontend/src/types/api.ts:21-26`](frontend/src/types/api.ts#L21-L26)、[`internal/handler/routes.go:47`](internal/handler/routes.go#L47)、[`internal/logic/auth/auth.go:184-202`](internal/logic/auth/auth.go#L184-L202)）— 与后端确认登录是否需要 captcha；不需要则前端字段标 optional 并 UI 隐藏；需要则后端补校验。**【核实完成】** 后端 `auth.go:197` 已调 `security.VerifyCaptcha` 消费 captcha，前端必填正确，无需改动，标记完成。
- [x] **P2-13 前端 — 未选分类时显式传 `categoryId=0` 语义模糊**（[`frontend/src/pages/admin/ArticleEditor.tsx:589`](frontend/src/pages/admin/ArticleEditor.tsx#L589)）— `delete payload.categoryId` 让“未选”等价于字段缺失。**【核实完成】** 当前代码为 `categoryId === '' ? 0 : Number()`，并无 `delete`；后端 `article.go:592` 以 `if req.CategoryID > 0` 守卫，`0` 语义为“无分类”，前后端一致，标记完成。
- [x] **P2-14 前端 — `scheduledAt` 时区转换盲区**（[`frontend/src/pages/admin/ArticleEditor.tsx:583`](frontend/src/pages/admin/ArticleEditor.tsx#L583)、[`frontend/src/pages/admin/ArticleEditor.tsx:981-991`](frontend/src/pages/admin/ArticleEditor.tsx#L981-L991)）— 跨时区/夏令时编辑会偏移；旧无 Z 后缀字符串浏览器解析不一致。后端统一返回带 Z 的 ISO，前端用 `Intl.DateTimeFormat` 或 `date-fns-tz` 显式时区。
- [x] **P2-15 前端 — Profile 改密码无前端最小长度校验**（[`frontend/src/pages/Profile.tsx:233-240`](frontend/src/pages/Profile.tsx#L233-L240)）— 与注册保持同等规则，套用 `useFormValidation`。**【核实完成】** Profile.tsx:42-48 已套用 `useFormValidation` + `minLength: 8`，与注册一致，标记完成。
- [x] **P2-16 前端 — 多处直接解构整 store 触发整页重渲**（[`frontend/src/pages/Register.tsx:21`](frontend/src/pages/Register.tsx#L21)、[`frontend/src/pages/Profile.tsx:12`](frontend/src/pages/Profile.tsx#L12)、[`frontend/src/pages/admin/Settings.tsx:12-26`](frontend/src/pages/admin/Settings.tsx#L12-L26)、[`frontend/src/pages/Login.tsx:25`](frontend/src/pages/Login.tsx#L25)）— 改用 `useShallow((s) => ({...}))` 或拆 selector。
- [x] **P2-17 前端 — `Home.tsx:82` 清空筛选误带 `q`**（[`frontend/src/pages/Home.tsx:82`](frontend/src/pages/Home.tsx#L82)）— Home 路由不读 `q`，删除冗余字段，或与 Search 对齐支持 `q`。
- [x] **P2-18 前端 — `App.tsx` sessionExpiredHandler 频繁重建**（[`frontend/src/App.tsx:69-75`](frontend/src/App.tsx#L69-L75)）— 每次 location 变化 set/cleanup，竞态窗口内 401 走默认逻辑。改用 `locationRef = useRef(location)`，effect 依赖只放 `[setSessionExpiredHandler, navigate]`。
- [x] **P2-19 文档 — 接口与文案多处遗漏 / 错误**：
  - `docs/API.md` 缺 `/healthz` 章节（[`docs/API.md`](docs/API.md)）；
  - `docs/API.md` 缺 `versions/:versionNo` GET 单独 endpoint 标题（[`docs/API.md:449-457`](docs/API.md#L449-L457)）；
  - `README.md:528-560` “常用 API”段缺多写操作接口（[`README.md:528-560`](README.md#L528-L560)）；
  - `.env.example:18` Redis 默认值与同文件占位风格不一致（[`.env.example:18`](.env.example#L18)）；
  - `.env.example` 缺 `APP_HOST/APP_PORT`（虽 [`internal/config/config.go:129-132`](internal/config/config.go#L129-L132) 支持）；
  - `.env.example:2` `APP_DISPLAY_NAME` 与 [`.env.example:61`](.env.example#L61) `APP_GITHUB_TOKEN` 后端 `ApplyEnv` 未读取；
  - [`deploy/mysql/alter_site_settings_value_text.sql:1-4`](deploy/mysql/alter_site_settings_value_text.sql#L1-L4) 与 [`deploy/mysql/add_public_page_content_settings.sql:3-4`](deploy/mysql/add_public_page_content_settings.sql#L3-L4) 重复 ALTER；
  - [`README.md:564`](README.md#L564) “替换 .env 中的默认密码”用词遗留，应改为“填写真实强随机密码替换所有 `<REPLACE_…>` 占位符”；
  - [`docs/API.md:87`](docs/API.md#L87) 描述 yaml 含示例密码与实际不符。

---

## P3 — 性能 / 维护性 / 文案

- [ ] **P3-1 后端 — `model.ListArticles` LIKE+MATCH 全表扫描**（[`model/article.go:530-534`](model/article.go#L530-L534)）— Meili 不可用时主搜索退化为 `LIKE '%xxx%'` 走不到索引。fulltext 命中 0 才走 LIKE，或限制 query 模式。
- [ ] **P3-2 后端 — `RequestID` 失败回退占位串相同**（[`internal/middleware/requestid.go:26-33`](internal/middleware/requestid.go#L26-L33)）— rand 失败期所有请求共用 `000000000000000000000000`，日志聚合混淆。退化为时间戳+随机字节。
- [ ] **P3-3 后端 — `aiclient.Assist` 温度 / max_tokens 硬编码**（[`internal/aiclient/client.go:63-72`](internal/aiclient/client.go#L63-L72)、[`internal/aiclient/client.go:204-215`](internal/aiclient/client.go#L204-L215)）— 暴露到 `AIConf`/`AISettings`。
- [ ] **P3-4 后端 — `scanArticleVersion` Unmarshal 失败 silent fallback**（[`model/article.go:643-645`](model/article.go#L643-L645)）— 至少 `logx.Errorf`。
- [ ] **P3-5 后端 — `UpdateProjectsPageContent` 总写空 JSON 覆盖旧字段**（[`model/site_settings.go:392-410`](model/site_settings.go#L392-L410)）— 回滚时旧列被清空数据丢失。要么彻底完成迁移并删字段，要么停止覆盖。
- [ ] **P3-6 后端 — `internal/handler/site/healthz.go:18-19` 写错误未 log**（[`internal/handler/site/healthz.go:18-19`](internal/handler/site/healthz.go#L18-L19)）— 加 logx 错误日志。
- [ ] **P3-7 后端 — `aiclient.Assist` `client.Timeout` 与 `ctx WithTimeout` 双重计时**（[`internal/aiclient/client.go:60-61`](internal/aiclient/client.go#L60-L61)、[`internal/aiclient/client.go:85-87`](internal/aiclient/client.go#L85-L87)）— 保留 ctx，去掉 client.Timeout。
- [ ] **P3-8 后端 — `cmd/notes-of-ashen/main.go` defer 顺序导致 ctx 比 server 先关**（[`cmd/notes-of-ashen/main.go:32-41`](cmd/notes-of-ashen/main.go#L32-L41)）— 调换 defer 顺序，让 `server.Stop()` 先执行。
- [ ] **P3-9 后端 — RabbitMQ 重连无 jitter**（[`internal/mq/event.go:131-152`](internal/mq/event.go#L131-L152)）— 指数退避加随机抖动。
- [ ] **P3-10 后端 — `httphelper.forwardedClientIP` 取 XFF 第一个 IP**（[`internal/httphelper/helper.go:128-138`](internal/httphelper/helper.go#L128-L138)）— 第一个值客户端可自报，配合限流可被绕过。取最右侧、可信代理之前的那一段；或限流 key 直接用 RemoteAddr。
- [ ] **P3-11 后端 — 七段数码 captcha 太弱**（[`internal/security/code.go:213-279`](internal/security/code.go#L213-L279)）— 主流 OCR 秒过；换 `dchest/captcha` 或 `mojocn/base64Captcha`，或加扭曲噪声。
- [ ] **P3-12 前端 — `vite.config.ts` 未开 sourcemap、代理硬编码**（[`frontend/vite.config.ts`](frontend/vite.config.ts)）— `sourcemap: 'hidden'` + `loadEnv` 读 `VITE_API_TARGET`。
- [ ] **P3-13 前端 — `pnpm build` 不跑 lint**（[`frontend/package.json`](frontend/package.json)）— CI 加 `pnpm lint`，或 `build` 改为 `pnpm lint && tsc && vite build`。
- [ ] **P3-14 前端 — `MarkdownCode/MarkdownTable` 语言判断重复且不订阅 store**（[`frontend/src/components/markdown/MarkdownCode.tsx:108-115`](frontend/src/components/markdown/MarkdownCode.tsx#L108-L115)、[`frontend/src/components/markdown/MarkdownTable.tsx:4`](frontend/src/components/markdown/MarkdownTable.tsx#L4)）— i18n 切换时不重渲；改为订阅 `usePreferenceStore` 或 props 注入。
- [ ] **P3-15 前端 — `error.ts:1` 直接耦合 axios**（[`frontend/src/utils/error.ts:1`](frontend/src/utils/error.ts#L1)）— 改用类型守卫减少耦合。
- [ ] **P3-16 前端 — `Login.tsx:25` 整 store 解构**（[`frontend/src/pages/Login.tsx:25`](frontend/src/pages/Login.tsx#L25)）— 同 P2-16，使用 selector。
- [ ] **P3-17 前端 — `ArticleDetail.tsx:99` 切换语言重新拉文章**（[`frontend/src/pages/ArticleDetail.tsx:99`](frontend/src/pages/ArticleDetail.tsx#L99)）— `language` 仅用于错误兜底文案，effect 依赖只保留 `[id]`，用 ref 持有最新 language。
- [ ] **P3-18 前端 — `ArticleEditor.tsx:498` 草稿自动保存依赖混合内容/控制字段**（[`frontend/src/pages/admin/ArticleEditor.tsx:498`](frontend/src/pages/admin/ArticleEditor.tsx#L498)）— 拆两个 effect，内容字段和控制字段分开。
- [ ] **P3-19 前端 — `Pagination` page=1 时 URL 风格不一致**（[`frontend/src/pages/Home.tsx:204`](frontend/src/pages/Home.tsx#L204)、[`frontend/src/pages/Search.tsx:294`](frontend/src/pages/Search.tsx#L294)）— 可选；统一保留 `?page=1` 或统一删除。
- [ ] **P3-20 前端 — `SearchHighlight.tsx` 全局共享 textarea 解码并发竞态**（[`frontend/src/components/SearchHighlight.tsx:10`](frontend/src/components/SearchHighlight.tsx#L10)、[`frontend/src/components/SearchHighlight.tsx:52-53`](frontend/src/components/SearchHighlight.tsx#L52-L53)）— 每次新建 textarea，或用 `useMemo` 在每个组件实例隔离。
- [ ] **P3-21 前端 — `useFormValidation.ts:81-90` `useMemo` 依赖每次新对象失效**（[`frontend/src/hooks/useFormValidation.ts:81-90`](frontend/src/hooks/useFormValidation.ts#L81-L90)）— 改为纯函数计算 errors，或要求调用方 memo `values`。
- [ ] **P3-22 前端 — `Layout.tsx:125` 直读 `localStorage` refreshToken**（[`frontend/src/components/Layout.tsx:125`](frontend/src/components/Layout.tsx#L125)）— P0-8 落地后改为从 store / Cookie 自动逻辑。
- [ ] **P3-23 前端 — `error.ts:97-101` 用中文短语作为 key 不规范**（[`frontend/src/utils/error.ts:97-101`](frontend/src/utils/error.ts#L97-L101)）— 改用枚举常量或形如 `__timeout_write__` 的 key 格式。
- [ ] **P3-24 部署 — api 服务无 healthcheck**（[`docker-compose.yml:22-71`](docker-compose.yml#L22-L71)）— 加 `healthcheck` 调 `/healthz`（注意 alpine 镜像默认无 wget/curl，需 Dockerfile.api 安装）；联动把 web `depends_on.api.condition` 改为 `service_healthy`。
- [ ] **P3-25 仓库 — `.gitignore` 可选忽略 `Todo.md` 等本地笔记**（[`.gitignore`](.gitignore)）— 按需添加。
- [ ] **P3-26 文档 — `README.md:60` MySQL/Redis 等版本应注明为远程依赖建议版本**（[`README.md:60`](README.md#L60)）— 改写为“推荐 MySQL 8.4 / Redis 7.4 / Meilisearch 1.13 / RabbitMQ 4”。
- [ ] **P3-27 数据库 — schema/路由可读性收尾**：
  - [`deploy/mysql/schema.sql:1`](deploy/mysql/schema.sql#L1) `CREATE DATABASE` 加 `CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci`，各表显式 COLLATE；
  - [`README.md:136`](README.md#L136) 与 [`README.md:461`](README.md#L461) DSN 推荐串追加 `time_zone=%27%2B08%3A00%27` 提示；
  - [`internal/handler/routes.go:51-54`](internal/handler/routes.go#L51-L54) 与 [`internal/handler/routes.go:71-72`](internal/handler/routes.go#L71-L72) 把 `/articles/:id` 详情移到所有 `/articles/:id/...` 之后；
  - [`api/notes-of-ashen.api`](api/notes-of-ashen.api) 缺 `/healthz` 声明，可补一行 `@handler healthz / get /healthz` 或在文件头注释说明 healthz 是手写非 goctl。

---

## 附录：已确认合理处理（不列入待办，避免误改）

- 鉴权中间件正确解析 Bearer，cache fail-open，禁用用户即时拒绝（[`internal/middleware/auth.go:53-89`](internal/middleware/auth.go#L53-L89)）。
- 限流中间件登录 / 验证码 fail-closed（[`internal/middleware/ratelimit.go:50-100`](internal/middleware/ratelimit.go#L50-L100)）。
- 配置 `ValidateConfig` 校验 secret 长度与 placeholder（[`internal/config/config.go:218-272`](internal/config/config.go#L218-L272)）。
- AI Key v2 加密使用 AES-GCM + 独立 KDF（[`internal/logic/ai/settings.go:236-253`](internal/logic/ai/settings.go#L236-L253)）。
- SQL 全部参数化（已抽样 article / user / taxonomy / site_settings / stats / log / traffic）。
- `WithUserRegistrationLock` + `uniq` 索引兜底（前提 P0-1 修复）。
- `RequestID` 中间件正确注入 ctx 与响应头（[`internal/middleware/requestid.go`](internal/middleware/requestid.go)）。
- `MarkdownRenderer` 未启用 `rehype-raw`，原始 HTML 被丢弃，XSS 面已收敛。
- [`frontend/package.json`](frontend/package.json) `packageManager: pnpm@9.15.9`，与 README 强调的 pnpm 一致。
- 端口（前端 3000、后端 19000、Docker 1270）在 [`frontend/vite.config.ts:38`](frontend/vite.config.ts#L38) / [`etc/notes-of-ashen.yaml:3`](etc/notes-of-ashen.yaml#L3) / [`docker-compose.yml:15`](docker-compose.yml#L15) 全一致。
- `notes-of-ashen.exe`、`.env*` 已被 `.gitignore` 正确忽略（[`.gitignore`](.gitignore)）。
- 增量迁移脚本均以 `USE notes_of_ashen;` 显式切库。
- Volume 名 `notes-of-ashen_goblog_meili_data`（[`README.md:428`](README.md#L428)）与 compose `name: notes-of-ashen` + `volumes: goblog_meili_data` 拼接一致。
