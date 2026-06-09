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
- `pnpm build`：先执行 `tsc` 类型检查，再执行 Vite 生产构建。
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
