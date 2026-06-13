# AGENTS.md

本文件是 `Notes of Ashen` 项目的 Codex 项目级说明。后续在本仓库工作时，优先遵守这里的项目上下文、约定和验证方式，避免重复读取基础信息。

## 语言与沟通

- 所有说明、分析、计划、总结、测试结果使用中文。
- 优先中文提问、中文回复。
- 代码、库名、接口字段名、命令和路径可保留英文。
- 最终回复使用以下结构：
  - `### 改动概述`
  - `### 修改文件`
  - `### 关键实现`
  - `### 验证结果`
  - `### 风险说明`

## 启动检查清单

每次开始任务先完成这些只读检查，再决定是否修改：

- 读取本文件，确认当前约定是否变化。
- 执行 `git status --short`，识别用户已有未提交改动。
- 修改前阅读相关代码、配置、类型定义和调用链。
- 若遇到已修改文件，先判断是否与任务相关；相关则顺着现状改，不相关则不碰。
- 不覆盖、不回滚用户未提交改动，除非用户明确要求。

## 项目概况

- 项目名：`Notes of Ashen`。
- 类型：前后端分离的个人博客系统。
- 后端：Go + go-zero REST 风格。
- 前端：React + TypeScript + Vite + Tailwind CSS + Zustand + Axios。
- 部署：Docker Compose、Nginx、1Panel 友好方案。
- 本机 Docker 默认访问地址：`http://127.0.0.1:1270`。
- 后端本地默认端口：`19000`。
- 前端开发服务默认端口：`3000`，并将 `/api` 代理到 `http://127.0.0.1:19000`。

## 项目已知约束

- 首个注册用户自动成为 `admin`。
- 当用户表为空且邮箱服务关闭时，首个管理员注册可跳过邮箱验证码；后续注册仍需要 `register` 用途邮箱验证码。
- 邮箱、AI、Meilisearch、RabbitMQ、GeoIP、Prerender 均为可配置能力；修改时必须同时核对默认配置、环境变量、Docker Compose、README 和实际降级路径。
- 站点设置更新中的可选布尔字段使用指针语义：字段缺失表示保留当前值，显式 `false` 表示关闭。
- AI 设置中的 API Key 由 `APP_AUTH_ACCESS_SECRET` 派生密钥加密；调整密钥轮换逻辑时必须考虑已保存密文迁移。
- 直连 API 时 `X-Forwarded-*` 请求头不可天然视为可信；涉及 IP、限流、来源统计和链接生成时要明确代理信任边界。

## 重要目录

- `api/notes-of-ashen.api`：go-zero API 描述文件，是接口结构的重要来源。
- `cmd/notes-of-ashen/main.go`：后端服务入口。
- `etc/notes-of-ashen.yaml`：后端默认配置。
- `internal/config`：配置结构与环境变量覆盖逻辑。
- `internal/handler`：HTTP Handler 和路由注册，保持轻量。
- `internal/logic`：业务逻辑主要位置。
- `internal/svc`：服务上下文和依赖初始化。
- `internal/response`：统一响应封装。
- `model`：数据访问层。
- `deploy/mysql`：数据库初始化与增量脚本。
- `docs/API.md`：API 文档。
- `frontend/`：React 前端应用。
- `frontend/src/api`：前端接口请求封装。
- `frontend/src/types`：前端 TypeScript 类型。
- `frontend/src/pages`：页面与后台页面。
- `frontend/src/store`：Zustand 状态。
- `frontend/src/utils/http.ts`：Axios 实例、统一响应处理和 Token 刷新逻辑。

## 后端约定

- Go 版本以 `go.mod` 为准，当前为 Go `1.25.0`。
- 模块名：`notes-of-ashen`。
- 使用 go-zero REST，入口加载 `etc/notes-of-ashen.yaml` 后调用 `Config.ApplyEnv()` 覆盖环境变量。
- Handler 保持简洁，只做请求解析、鉴权上下文读取、调用 logic、返回响应。
- 业务逻辑放在 `internal/logic`，通用辅助能力放到已有 util/cache/search/email/mq 等模块。
- 数据访问优先复用 `model.Store` 和既有 model 方法。
- 错误必须明确处理，不忽略返回值。
- 对外响应统一使用 `internal/response`：
  - 成功：`{ "code": 0, "message": "success", "data": ... }`
  - 无数据成功：`{ "code": 0, "message": "success" }`
  - 业务错误优先使用项目已有错误类型。
- 涉及接口变更时，同步检查：
  - `api/notes-of-ashen.api`
  - `internal/types`
  - 对应 handler/logic/model
  - `docs/API.md`
  - 前端 `frontend/src/api`、`frontend/src/types` 和相关页面。

## 接口联动清单

涉及接口字段、状态码、分页、错误结构或权限变化时，至少同步核对：

- API 描述：`api/notes-of-ashen.api`。
- 后端类型与实现：`internal/types`、对应 handler、logic、model。
- 统一响应与错误：`internal/response`、`internal/errors`、前端 `frontend/src/utils/error.ts`。
- 前端调用与类型：`frontend/src/api`、`frontend/src/types`、相关页面和 store。
- 文档与示例：`docs/API.md`、`README.md`、`.env.example`、Docker Compose。
- 测试：优先覆盖权限、可选字段、默认值、失败路径和前后端类型漂移风险。

## 前端约定

- 前端位于 `frontend/`。
- 使用 React 18、TypeScript、Vite 5、Tailwind CSS、Zustand、Axios。
- 严格使用 `pnpm`，不要使用 `npm` 或 `yarn`。
- 优先函数式组件和 Hooks。
- 优先复用现有组件、store、API 封装和工具函数。
- 页面改动必须考虑 Loading、Error、Empty State。
- 接口请求优先通过 `frontend/src/api` 和 `frontend/src/utils/http.ts`。
- API 类型优先维护在 `frontend/src/types`。
- 不要把真实 Token、Key、密码或 `.env` 内容写入前端代码。

## 配置与安全

- 不读取、不泄露真实 `.env` 内容。
- 需要了解配置时优先参考 `.env.example`、`README.md`、`etc/notes-of-ashen.yaml` 和代码中的配置结构。
- 不硬编码 API Key、Token、密码、私有模型路径或私有服务地址。
- AI、邮箱、Meilisearch、Redis、RabbitMQ、GeoIP 等配置都应通过配置文件或环境变量传入。
- 修改配置项时同步检查 Docker Compose、README、后端 config 结构和实际使用点。

## 工作原则

- 先阅读相关代码和配置，再开始修改。
- 优先最小必要改动。
- 优先复用现有实现和项目约定。
- 不修改与任务无关的文件。
- 不进行无必要重构。
- 不新增占位实现。
- 不引入无必要依赖；确需新增前端依赖时使用 `pnpm add xxx@latest` 或 `pnpm add -D xxx@latest`。
- 保持前后端一致，不留下接口字段、响应结构、类型定义或页面展示不匹配的状态。

## 审计优先级

查找 bug 或做 review 时按优先级输出和修复：

- P0：安全问题、数据破坏、权限绕过、生产无法启动。
- P1：核心流程不可用，例如首次部署、注册登录、文章发布、构建失败。
- P2：接口/类型/文档不一致、明显用户体验问题、可测试性缺口。
- P3：性能警告、维护性改进、局部文案或样式 polish。

## 常用命令

后端：

```powershell
go test ./...
go build ./cmd/notes-of-ashen
go run ./cmd/notes-of-ashen -f etc/notes-of-ashen.yaml
```

前端命令需在 `frontend/` 目录执行：

```powershell
pnpm install
pnpm dev
pnpm lint
pnpm build
pnpm preview
```

Docker 本机运行：

```powershell
docker compose config --quiet
docker compose up -d --build
```

## 验证要求

- 完成 Go 后端改动后，尽可能执行 `go test ./...`。
- 完成前端改动后，尽可能在 `frontend/` 执行：
  - `pnpm lint`
  - `pnpm build`
- 完成部署配置改动后，尽可能执行 `docker compose config --quiet`。
- 纯文档改动不需要运行构建或单元测试，但需要检查 diff。
- `pnpm build` 的 Vite chunk size warning 属于性能风险提示，不等同于构建失败；若本次未优化，需要在最终回复风险中说明。
- 未实际执行的验证不能声称已通过。
- 如果无法验证，必须在最终回复中说明原因。

## 终端与编码

- 默认使用 PowerShell / `pwsh` 风格命令。
- 默认使用 UTF-8 编码。
- 注意中文文件、中文路径和中文输出兼容性。
