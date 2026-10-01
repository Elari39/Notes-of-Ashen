# 站点品牌图标

用户选定导航使用 AI 03「旅巫书签」，浏览器标签页使用 AI 02「月弧羽笔」。两者来自用户提供接口生成的候选图，经本地处理移除白底、裁切留白并按明暗配色导出；本次接入未再次调用生图 API。

- `nav-witch-light.png`、`nav-witch-dark.png`：144×144 透明 PNG，在导航中以 36×36 显示，随站点 effectiveTheme 切换。作为独立 Vite 资源加载，避免内联图片增加首屏 JS。
- `public/favicon.svg`：内嵌月弧羽笔的两套 PNG，依据浏览器 `prefers-color-scheme` 切换。
- `public/favicon.png`：32×32 兼容版本，浅色细轮廓帮助其在不同标签栏背景中显示。
- `index.html` 与 Service Worker 预缓存使用 `moon-quill-1` 查询版本，避免继续命中旧 favicon 缓存。

图标旁已有站名文字，导航图片使用空 alt 与 aria-hidden，避免读屏重复朗读。PWA 安装图标未在本次浏览器标签图标修改范围内。

本次接入验证：pnpm test（115 项）、pnpm lint、pnpm build（含类型检查与体积预算）均通过；六套风格在 Chromium、移动 Chromium、移动 WebKit 的 18 项明暗切换测试通过。另行检查导航截图与 favicon 明暗媒体规则。未重跑完整 Docker core 集成；原有 Vite 大 chunk 警告仍存在。
