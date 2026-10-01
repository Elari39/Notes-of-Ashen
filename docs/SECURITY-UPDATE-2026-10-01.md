# 2026-10-01 依赖安全更新

本次更新依据项目审计的 F08。历史审计报告描述旧版本，不能作为更新后制品的验证结果。

## 更新范围

- Go 工具链与 API 构建镜像固定为 1.27.1，镜像同时固定 SHA-256 摘要。
- 更新 go-zero、RabbitMQ 客户端、gRPC、OpenTelemetry、压缩库及 `golang.org/x/{crypto,image,net,text,sync,sys}`。RabbitMQ 客户端至少为 1.13.0，处理 GO-2026-6372 的 broker 超大载荷问题。
- gRPC 使用稳定补丁版 1.83.2。GO-2026-6443 的官方受影响区间仍包含 1.84.0，因此不能简单将最高稳定版本等同于已修复版本。
- 更新 Axios 1.20、Mermaid 11.16.1、React Router 7.18.4、react-syntax-highlighter 16.1.1、Vite 6.4.3，并更新锁文件中受公告影响的传递依赖。保留 Node 22 与 pnpm 9.15.9。
- 保持原包体积门禁。删除没有任何 Tooltip 使用者的全局 Provider，使定位库不进入首屏；Tooltip 组件仍可在需要它的功能入口使用。没有用新增初始 chunk 转移超限代码。

## 扫描结果与边界

官方 npm 审计对更新后的冻结锁文件报告 0 项漏洞匹配。Go 源码扫描报告 0 个调用匹配、0 个导入包匹配；`golang.org/x/crypto` 模块仍包含未维护 OpenPGP 的 GO-2026-5932 记录，此公告没有修复版本。

`govulncheck` 二进制模式仍以通配符报告 GO-2026-5932，退出码为 3；不得将该命令记为通过。针对实际 Docker Linux 制品补充核验：

1. `GOOS=linux go list -deps ./cmd/notes-of-ashen` 不包含任何 `golang.org/x/crypto/openpgp` 包。
2. 从实际 ELF 的 `.gopclntab` 解析到 36,302 个 Go 函数，其中 OpenPGP 函数为 0；作为正向校验，实际使用的 bcrypt 函数为 17。
3. 因此该模块级通配符记录不对应本制品链接或调用的 OpenPGP 功能。保留扫描原始结果及核验脚本，不通过忽略公告或移除检查取得成功状态。

上述结果是指定源码与制品的审计，不代表所有依赖没有未知漏洞，也不证明线上已经部署新版本。后续依赖或工具链变化后需重新扫描并执行真实集成测试。

本地证据保存在被 Git 忽略的 `security-audit/2026-10-01/remediation/`：`pnpm-audit-upgraded-v2.json`、`govulncheck-source-final-deps.log`、`govulncheck-binary-upgraded.log`、`api-buildinfo-upgraded.txt`、`linux-package-dependencies.txt`、`binary-package-evidence.log` 和 `check-binary-imports.go`。
