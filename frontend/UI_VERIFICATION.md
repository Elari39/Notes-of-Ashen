# UI 主题与页面重构验证记录

日期：2026-10-01。结论：实现及主要验证完成，**完整 Docker core 集成门禁未通过**；保留一项可在独立最小页面复现的 Windows WebKit 离线刷新失败。

## 改动与兼容性

新增经典 Neo-brutalism 与 Dark Academia 两套明暗主题，保留原有四套风格、默认风格、自定义强调色与系统明暗偏好。首页采用紧凑刊头、主推文章与去重后的文章流，保留每页 10 篇、服务端排序、筛选 URL、两种列表布局及 CTA 开关。无封面／坏图不打断封面交错顺序。

公共阅读、搜索、表单与弹窗统一排版；后台统一导航分组、标题、表格、设置卡片、编辑器布局与开关表现。图表补充风格变更依赖。未改变后端接口、数据库、依赖版本或业务权限。

## 实际执行结果

| 验证 | 结果 |
| --- | --- |
| 改动前 `pnpm test` | 112 通过，0 失败、0 跳过 |
| 改动前 `pnpm build` | lint、类型检查、构建、体积预算通过；存在大 chunk 警告 |
| 改动后 `pnpm test` | 115 通过，0 失败、0 跳过 |
| 改动后 `pnpm lint` / `pnpm build` | 零警告 lint、类型检查、构建及全部体积预算通过 |
| 最终 Docker Chromium | 49 通过、1 原有条件跳过 |
| 最终 Docker 移动 Chromium | 50 通过、0 跳过 |
| 最终 Docker 移动 WebKit | 49 通过、1 失败 |
| `git diff --check` | 通过 |

最终 Docker 命令：`./scripts/test-integration.ps1 -Suite core`，退出码 **1**。前三列浏览器合计 148 通过、1 失败、1 跳过。唯一跳过是原有“仅移动端运行”的触控测试在桌面 Chromium 中不适用；移动项目中正常执行。脚本在浏览器前完成 HTTP、两种历史迁移、并发迁移与 Redis 认证／故障阶段。

每个浏览器的 26 项主题测试均通过，共 78 项。覆盖六套主题的明暗切换、刷新恢复、系统偏好、存储异常、320px 中英文、桌面／平板截图、首页去重／分页／筛选、默认对比度、自定义色主按钮文字、键盘退出、真实坏图请求以及封面交错行为。真实链路覆盖注册、验证码登录、文章发布、媒体上传、权限降级、会话恢复、新主题后台页面与设置保存。截图已抽查首页和后台，修正了开关轨道被通用高度规则拉伸的问题。

首屏 JavaScript 最终为 raw **307.36 KiB**、gzip **101.42 KiB**、brotli **83.65 KiB**，预算仍为 320 / 110 / 96 KiB。未调整预算或 Vite 警告阈值。

## 未通过项及独立复现

原有用例“真实删除文章后，读者离线刷新不能恢复已撤回内容”在移动 WebKit 中失败：

```text
await reader.setOffline(true);
await page.reload();
Error: page.reload: WebKit encountered an internal error
```

此前删除响应 200、重新请求 404、文章缓存已清除的断言均通过；失败发生于离线导航，后续页面与内容断言未执行，因此不能宣布该链路通过。Chromium 两个项目的相同用例通过。

为区分应用回归与环境问题，另起临时本地 HTTP 服务，返回仅有一个标题和 Service Worker 注册的 HTML；Service Worker 缓存首页并在离线导航时回退缓存。该页面不加载本项目的 JS、CSS 或后端。当前安装的 Playwright WebKit 确认 `controller: true`、`cached: true` 后，执行 `setOffline(true)` 和 `reload()`，同样得到 `WebKit encountered an internal error`。这支持当前 Windows WebKit 离线导航环境问题的判断，但不替代原用例通过。未修改或跳过该断言，未吞掉异常，未增加自动重试。

## 失败历史与证据位置

- 首轮开发服务 WebKit 出现导航挂载时序及运行期间热更新相关的测试超时；辅助函数增加导航可见性等待，矩阵按主题拆分，保持原有 90 秒超时及全部断言。后续生产构建矩阵通过。
- 首轮 Docker 在新增交错封面夹具失败：相对 URL 被项目媒体规则过滤。修正为合法的受拦截 HTTP URL，并加入图片请求计数断言；专项 7 项通过，最终三个浏览器对应测试均通过。
- 最终完整报告：`D:/temp/notes-of-ashen-integration-20261001145432-52173a02/`，各浏览器的 `playwright-report/index.html`、截图和失败 trace 保留在其中。
- 首轮 Docker 失败记录：`D:/temp/notes-of-ashen-integration-20261001134917-55d894bc/`。
- 早期移动端记录：`D:/temp/notes-ui-mobile-verification/`。

隔离测试容器、卷与测试镜像由既有脚本清理；未重建日常站点。`extended` 套件未执行，本次没有后端代码改动，未额外执行全量 `go test ./...`。

## 剩余风险

完整集成门禁仍为失败，需要在可用的 WebKit 离线环境继续验证该用例。Vite 大 chunk 性能警告仍存在。用户自选强调色可能降低链接与背景对比度；默认主题和自定义色主按钮文字已分别验证。中文字体回退可能随操作系统产生外观差异。
