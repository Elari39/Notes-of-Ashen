# 测试说明

## P3-01 真实集成测试

完整集成测试使用真实的 Nginx、API、MySQL 和 Redis。运行前需要 Docker Compose、Go、pnpm，以及已安装前端依赖；脚本不会读取仓库或用户目录中的 `.env`。

```powershell
pwsh -File scripts/test-integration.ps1 -Suite core
pwsh -File scripts/test-integration.ps1 -Suite extended
```

`core` 依次执行两个彼此隔离的 Compose 生命周期：

- Go 黑盒 HTTP 集成测试（真实 API、MySQL、Redis 与 Web Nginx）。
- Chromium 浏览器 E2E（首次注册、刷新 Cookie、文章编辑、媒体与角色边界）。

`extended` 在上述两个阶段后，使用第三个全新 Compose 生命周期执行并发、恢复失败注入与 Redis fail-closed 测试；Redis 故障用例固定最后执行，避免其停止/重启过程影响其他需要宿主 Redis 端口的断言。

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

首次在新机器执行时，脚本会安装 Playwright Chromium。CI 已预装浏览器时，可在调用前设置 `E2E_SKIP_BROWSER_INSTALL=1`。

## CI

`.github/workflows/integration-e2e.yml` 在推送和拉取请求时运行 `core`；每日 `Asia/Shanghai` 02:00（UTC 18:00）及手动触发运行 `extended`。工作流失败时上传 Playwright 产物和脚本保存的 Compose 日志。
