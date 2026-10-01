# Notes of Ashen Frontend

这是 `Notes of Ashen` 的前端子项目，基于 React 18、TypeScript、Vite、Tailwind CSS、Zustand 与 Axios 构建。

根目录 README 负责完整部署说明；这里仅记录前端本地开发和验证命令。

## 本地开发

请在 `frontend/` 目录中执行命令，并统一使用 `pnpm`。

```bash
pnpm install
pnpm dev
```

开发服务默认地址：

```text
http://127.0.0.1:3000
```

Vite 会将 `/api` 代理到：

```text
http://127.0.0.1:19000
```

如需调整代理目标，请修改 [vite.config.ts](vite.config.ts)。

## 常用脚本

```bash
pnpm lint
pnpm build
pnpm preview
```

- `pnpm lint`：执行 ESLint。
- `pnpm build`：执行零警告 lint、类型检查、Vite 生产构建与构建体积预算检查。
- `pnpm preview`：预览生产构建结果。

## 目录说明

```text
frontend/
├── public/              # 静态公共资源
├── src/
│   ├── api/             # 接口请求封装
│   ├── assets/          # 前端资源
│   ├── components/      # 复用组件
│   ├── pages/           # 页面与后台页面
│   ├── store/           # Zustand 状态
│   ├── types/           # TypeScript 类型
│   └── utils/           # 通用工具
├── index.html
├── package.json
├── tailwind.config.js
└── vite.config.ts
```

## 注意事项

- 不要混用 `npm` 或 `yarn`。
- 不要把真实 Token、密钥或 `.env` 内容写入前端代码。
- 接口字段变更时同步检查 `src/api/`、`src/types/` 和相关页面展示。


## 主题与页面布局

在导航栏的「偏好」中选择原有刊物、柔和新粗野、日式纸本、瑞士平面、经典新粗野（Neo-brutalism）或暗色书房（Dark Academia）。六套风格均支持浅色、深色和跟随系统，原有刊物仍为默认风格。

- 经典新粗野：黄色主推区、蓝红装饰色、粗描边与硬阴影；深色版保留亮色装饰，正文使用中性色面板。
- 暗色书房：浅色为羊皮纸，深色为胡桃木与黄铜色，使用现有衬线字体及中文字体回退。
- 风格与明暗分别保存在浏览器中；切换风格不会覆盖明暗偏好或自定义强调色。重置强调色恢复当前风格、当前明暗下的默认值。
- 主题覆盖后台和公共页面。图表会在风格、明暗或强调色改变时重新应用颜色。

首页只在无筛选第一页展示主推文章，使用服务端返回的第一篇并从下方列表按 ID 去重。每页仍按接口的 10 篇计算，置顶排序、分类／标签 URL、标准／交错布局和底部入口隐藏设置不变。只有一篇文章时不显示空的后续列表；坏封面降级为文字卡片。

主题注册位于 `src/store/themeStyles.ts`；现有四套风格在 `src/themes.css`，新增两套在 `src/themes-new.css`；公共排版及后台工作区布局在 `src/page-layout.css`。装饰色和文本强调色分离，新主题默认正文／常用背景对比度至少为 4.5:1。用户自选强调色可能降低链接与背景的对比度，主按钮文字会按强调色自动选择深浅色。

## UI 验证

```powershell
pnpm test
pnpm lint
pnpm build
# 已启动前端服务后，主题夹具测试不需要写入后端数据
$env:E2E_WEB_BASE_URL = 'http://127.0.0.1:3000'
pnpm exec playwright test e2e/theme-preferences.spec.ts
# 在仓库根目录执行真实隔离 Compose 集成测试
.\scripts\test-integration.ps1 -Suite core
```

主题测试覆盖六套风格的明暗切换、存储降级、中英文窄屏、首页分页与去重、坏封面、失败重试、对比度以及截图。真实集成测试独立验证认证、权限、文章发布、媒体、会话和离线撤回，并检查新主题下的后台页面及 Mermaid。浏览器项目保持 Chromium、移动 Chromium 与移动 WebKit，失败重试为零；截图检查不替代行为断言。Vite 大 chunk 警告需如实记录，不能通过放宽预算或删除断言让验证通过。

本轮实际执行结果、失败记录与环境复现说明见 [UI_VERIFICATION.md](UI_VERIFICATION.md)。
