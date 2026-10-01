# 测试说明

## P3-01 真实集成测试

完整集成测试使用真实的 Nginx、API、MySQL 和 Redis。运行前需要 Docker Compose、Go、pnpm，以及已安装前端依赖；脚本不会读取仓库或用户目录中的 `.env`。

Go 工具链应与 `go.mod` 声明的 1.27.1 保持一致，Docker API 构建也固定此版本及镜像摘要。在 PowerShell 中可先设置 `$env:GOTOOLCHAIN='go1.27.1'`，再执行单元测试、`go vet` 和集成脚本。gRPC 和 `golang.org/x/net` 已随安全更新同步调整；升级工具链仍需单独验证，不应通过关闭检查或修改依赖缓存绕过编译失败。

```powershell
pwsh -File scripts/test-integration.ps1 -Suite core
pwsh -File scripts/test-integration.ps1 -Suite extended
```

`core` 在彼此隔离的 Compose 生命周期中执行：

- Go 黑盒 HTTP 集成测试（真实 API、MySQL、Redis 与 Web Nginx）。
- 真实 MySQL 的首次管理员并发注册和最后管理员保护测试，同时断言禁用后的令牌撤销。runner 自动注入 `APP_TEST_DATABASE_DSN`；单独运行 `go test ./model -run '^TestMySQL' -count=1` 时，必须提供以 `_test` 结尾的入口库 DSN，并允许创建/删除本轮随机测试库。夹具通过生产迁移建库，不重置入口库。
- 历史 schema 自动迁移、并发 migrate、Redis 认证与错误密码 fail-fast。
- Chromium 桌面、Pixel 7 Chromium、iPhone 13 WebKit 浏览器 E2E；三个浏览器项目各使用全新的 Compose 数据库，覆盖首次注册、刷新 Cookie、文章编辑、媒体、角色边界，以及移动端首页、搜索、文章详情、登录和后台文章表格的触控与横向溢出断言。

`extended` 在 `core` 后使用全新的 Compose 生命周期执行其余并发与恢复失败注入测试，并恢复原生产 Nginx 10r/s、burst 20 配置验证 45 请求突发的 429 语义；Redis 故障用例固定在浏览器阶段前执行，避免其停止/重启过程影响其他需要宿主 Redis 端口的断言。

每个阶段都会生成新的 Docker 项目名、卷、随机数据库/认证凭据、Docker 子网和 loopback 随机端口。测试结束会执行 `docker compose down --volumes --remove-orphans --rmi local`；成功时会删除临时目录和凭据。失败时会保留不含临时环境文件的 Compose 日志，路径会输出到控制台；GitHub Actions 会上传该目录以及 Playwright 报告、trace 和截图。

测试进程可读取下列由脚本注入的变量，禁止把它们写入生产配置：

| 变量 | 用途 |
| --- | --- |
| `E2E_WEB_BASE_URL` | 随机 loopback Web/Nginx 地址。 |
| `E2E_API_BASE_URL` | 随机 loopback API 根地址，不含 `/api/v1`。 |
| `E2E_REDIS_URL` | 随机 loopback Redis 地址，仅用于验证码和故障断言。 |
| `E2E_MYSQL_DSN` | 测试业务账户 MySQL DSN。 |
| `E2E_MYSQL_ROOT_DSN` | 测试 MySQL root DSN，仅在测试进程生命周期内有效。 |
| `E2E_COMPOSE_PROJECT` | 当前阶段隔离的 Compose 项目名。 |
| `E2E_REDIS_CONTAINER_ID` / `E2E_MYSQL_CONTAINER_ID` | 扩展故障注入所需的测试容器 ID。 |
| `E2E_ARTIFACT_DIR` | 当前阶段的失败产物目录。 |

测试环境固定关闭邮件、RabbitMQ、Meilisearch 与 Prerender；仅在该环境设置 `APP_AUTH_COOKIE_SECURE=false`，使 HTTP loopback Chromium 能验证 Refresh Token Cookie。生产 Compose 配置不受影响。

会话失效回跳场景需要两次真实登录。移动端串行用例会先读取隔离 Redis 的登录计数，并在必要时等待前序场景的 5 次/分钟窗口自然释放；不会清除计数、放宽阈值或重试失败的登录。Nginx 生产限流有独立突发断言。

首次在新机器执行时，脚本会安装 Playwright Chromium 与 WebKit。CI 已预装浏览器时，可在调用前设置 `E2E_SKIP_BROWSER_INSTALL=1`。

本地与 CI 的 Playwright 重试次数均为 0；首次失败会使门禁失败。已知上游问题也不通过跳过测试或吞异常记为通过。

当前 WebKit 的 `setOffline(true)` 会拒绝由 Service Worker 提供的导航，相关上游问题为 [microsoft/playwright#42775](https://github.com/microsoft/playwright/issues/42775)。本项目保留“真实删除文章后，读者离线刷新不能恢复已撤回内容”的原断言；此用例受阻时完整套件仍失败。关闭源站的对照试验不能代替该离线模拟用例，也不能据此宣称完整 E2E 通过。

## 主题专项回归

`frontend/e2e/theme-preferences.spec.ts` 在三个浏览器项目中验证四种风格的明暗切换、刷新持久化、默认色与自定义色、系统明暗变化、异常或不可用存储、中英文界面、320px 窄屏布局，以及加载／失败／重试／空状态。三种新主题的默认文字与常用背景还需满足 4.5:1 对比度。

已有本地 Web 服务时，可在 `frontend/` 目录单独运行：

```powershell
$env:E2E_WEB_BASE_URL='http://127.0.0.1:1270'
$env:E2E_ARTIFACT_DIR='../tmp/theme-e2e'
pnpm exec playwright test e2e/theme-preferences.spec.ts
```

专项用例拦截公共 API，使用固定的长标题与 Markdown 内容，不写入预览站点数据；其结果不等同于后端集成测试。完整 `core` / `extended` 脚本会同时运行专项用例和 `critical-path.spec.ts` 中的真实认证、发布、媒体及权限链路，不能以专项用例替代现有集成检查。桌面 Chromium 按既有规则跳过移动端专用用例，该用例在两个移动浏览器项目执行。

## 前端构建与包体积

在 `frontend/` 目录执行：

```powershell
pnpm lint
pnpm type-check
pnpm test
pnpm build
pnpm check:bundle-size
```

`check:bundle-size` 对初始 JavaScript、Markdown、语法高亮与 ECharts chunk 分别输出并限制 raw、gzip、brotli 三种大小；Markdown 代码语言语法按需动态加载，未知或加载失败的语言会渲染为普通代码块。

如需调整包体积预算，提交或 PR 必须说明调整原因，并给出调整前后的三种大小基线；不得仅为通过检查而抬高阈值。

## CI

`.github/workflows/integration-e2e.yml` 在任意 push 和拉取请求时运行 `core`；`extended` 作为发布门禁在 `main` push 时运行，并继续在每日 `Asia/Shanghai` 02:00（UTC 18:00）和手动触发时运行。成功测试会删除临时日志、环境文件和凭据；失败时会上传 Playwright 报告、trace、截图/视频和脚本保留的无凭据 Compose 日志。
