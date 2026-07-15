# Notes of Ashen API 文档

本文档描述 Notes of Ashen 当前实现的 HTTP API。默认开发服务地址：

```text
http://127.0.0.1:19000
```

## 通用约定

除文件上传、Markdown 导出、RSS 和 Sitemap 外，请求体默认使用 JSON：

```text
Content-Type: application/json
```

受保护接口需要携带 Access Token：

```text
Authorization: Bearer <accessToken>
```

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

无数据成功响应：

```json
{
  "code": 0,
  "message": "success"
}
```

失败响应：

```json
{
  "code": 40000,
  "message": "field is required"
}
```

错误码：

| code | HTTP 状态码 | 含义 |
| --- | --- | --- |
| 40000 | 400 | 请求参数错误 |
| 40100 | 401 | 未登录、Token 缺失、Token 无效或过期，或登录凭据不正确 |
| 40300 | 403 | 权限不足、注册关闭、用户被禁用或功能未启用 |
| 40400 | 404 | 资源不存在 |
| 40900 | 409 | 资源冲突，例如账号、邮箱、slug 重复 |
| 42900 | 429 | 请求过于频繁 |
| 50000 | 500 | 服务内部错误 |
| 50300 | 503 | 限流器等依赖故障导致服务暂时不可用 |

列表接口统一支持分页：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| page | int | 1 | 页码，小于 1 时按 1 处理 |
| size | int | 10 | 每页数量，小于 1 时按 10 处理，最大 100 |

分页响应：

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "size": 10
}
```

时间字段由 Go JSON 编码输出，通常为 RFC3339：

```text
2026-06-05T20:00:00+08:00
```

`etc/notes-of-ashen.yaml` 是本地开发默认配置，敏感字段（数据库密码、Redis 密码、JWT Secret、RabbitMQ 地址、Meilisearch 配置等）默认留空或为占位，需通过受控配置或环境变量注入。AI 连接配置统一由管理员保存到数据库 `site_settings`，不从 YAML 或环境变量读取。生产环境部署前必须填入真实值，不要提交密钥到仓库。

默认不信任客户端传入的 `X-Forwarded-*` / `X-Real-IP`。只有 `RemoteAddr` 命中 `APP_TRUSTED_PROXY_CIDRS` 时，后端才会使用这些转发头参与限流、操作日志、流量统计和 RSS/Sitemap 基础 URL 生成。

后台新保存的 AI API Key 使用 `v3:` 密文，密钥由 `APP_AUTH_ACCESS_SECRET` 通过独立用途派生。`v2:` 密文升级后不可继续使用，设置响应会返回 `apiKeyNeedsUpdate = true`，需要管理员重新录入；无版本前缀的旧密文仍兼容读取，并会在后续保存时迁移为 `v3:`。轮换 `APP_AUTH_ACCESS_SECRET` 会使原 `v3:` 密文不可解密，必须同时安排重新录入 AI API Key。

API 访问日志仅记录方法、路径、状态码、耗时、可信客户端 IP 和 Request ID，不记录查询串、请求头、Cookie 或请求正文。上游 AI 错误在写入诊断日志前会对当前 API Key 脱敏，且不会向调用方透传上游响应正文。

## 健康检查

### 健康探针

```text
GET /healthz
```

无需鉴权。返回 JSON 格式的健康报告，`Content-Type: application/json`。全部依赖正常时返回 `200 OK`，响应体 `{"status":"ok","checks":{"db":{"status":"up"},"redis":{"status":"up"},...}}`；存在依赖不可用时返回 `503 Service Unavailable`，`status` 为 `"degraded"`，对应 `checks` 条目中 `status` 为 `"down"` 并附带 `error` 字段。用于容器 healthcheck / 负载均衡探活。注意 `/healthz` 为手写 handler，不使用统一 `{ code, message, data }` 响应封装，也未在 `api/notes-of-ashen.api` 中声明（详见该文件头注释）。

## 认证接口

### 获取图片验证码

```text
POST /api/v1/auth/captcha
```

权限：公开。用于登录、注册、找回密码、修改密码和修改邮箱前的人机校验。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| purpose | string | 否 | `login`、`register`、`reset_password`、`change_password`、`update_email`；为空时默认 `login` |

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| captchaId | string | 图片验证码 ID |
| imageData | string | 可直接用于 `<img src>` 的 PNG base64 Data URL |
| expiresIn | int64 | 有效秒数，默认 300 |

### 发送公开邮箱验证码

```text
POST /api/v1/auth/verify-code/send
```

权限：公开。用于注册和找回密码。发送前必须提交同用途图片验证码，同 IP 1 分钟最多 5 次。

当 `purpose = reset_password` 时，不存在、已禁用和正常邮箱对外返回完全一致的成功响应，避免泄露账号状态；仅正常可用账户会实际生成并发送邮箱验证码。Redis、SMTP 等服务故障仍会返回服务错误。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 是 | 接收验证码的邮箱 |
| purpose | string | 是 | `register` 或 `reset_password` |
| captchaId | string | 是 | 图片验证码 ID |
| captchaCode | string | 是 | 图片验证码答案 |

### 注册

```text
POST /api/v1/auth/register
```

权限：公开。第一个注册用户自动成为 `admin`，后续用户默认为 `user`。若站点注册开关关闭，后续注册会被拒绝。若当前没有任何用户且邮箱服务未启用，第一个管理员注册可不传邮箱验证码，保证默认 Docker 快速开始路径可用。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| account | string | 是 | 账号，长度 3 到 64 |
| password | string | 是 | 密码，长度 8 到 128 |
| email | string | 是 | 邮箱，需符合邮箱格式 |
| emailCode | string | 条件必填 | `register` 用途邮箱验证码；仅当当前没有任何用户且邮箱服务未启用时可省略 |
| nickname | string | 否 | 昵称，非空时长度 1 到 64 |
| avatarUrl | string | 否 | 头像 URL；为空表示不显示头像，非空必须为 `http://` 或 `https://` URL |

### 登录

```text
POST /api/v1/auth/login
```

权限：公开。登录接口有 IP 限流保护。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| account | string | 是 | 账号或邮箱 |
| password | string | 是 | 密码 |
| captchaId | string | 是 | `login` 用途图片验证码 ID |
| captchaCode | string | 是 | 图片验证码答案 |

### 找回密码

```text
POST /api/v1/auth/password/reset
```

权限：公开。服务端先原子验证并消费验证码，再查询账号；无效和过期验证码统一返回 `400 email code is invalid or expired`，不会通过 `404/403/400` 差异泄露邮箱是否注册或账号状态。重置成功后会撤销该用户已有 Refresh Token，需要重新登录。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 是 | 账号绑定邮箱 |
| emailCode | string | 是 | `reset_password` 用途邮箱验证码 |
| newPassword | string | 是 | 新密码，长度 8 到 128 |

### 刷新 Token

```text
POST /api/v1/auth/refresh
```

权限：公开。刷新成功后，旧 Refresh Token 会被撤销，并返回新的 Access Token。

> Refresh Token 默认通过后端下发的 `HttpOnly` + `SameSite=Strict` Cookie（`noa_refresh_token`）携带，请求需带凭证（`withCredentials`）；Body 中的 `refreshToken` 字段可选，仅用于直接调用 API 的客户端兜底。响应中的 `refreshToken` 字段已置空，前端不应再依赖。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| refreshToken | string | 否 | 默认由 Cookie 携带；Body 兜底传入旧 Refresh Token |

Token 响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "jwt",
    "refreshToken": "",
    "tokenType": "Bearer",
    "expiresIn": 7200
  }
}
```

### 退出登录

```text
POST /api/v1/auth/logout
```

权限：登录用户。撤销当前 Refresh Token 并清除 `noa_refresh_token` Cookie；Cookie 缺失时幂等返回成功。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| refreshToken | string | 否 | 默认由 Cookie 携带；Body 兜底传入要撤销的 Refresh Token |

## 用户接口

### 获取当前用户

```text
GET /api/v1/users/me
```

权限：登录用户。

### 发送当前用户邮箱验证码

```text
POST /api/v1/users/me/verify-code/send
```

权限：登录用户。用于修改密码和修改邮箱，发送前必须提交同用途图片验证码，同 IP 1 分钟最多 5 次。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| purpose | string | 是 | `change_password` 或 `update_email` |
| email | string | 条件必填 | `update_email` 时为目标新邮箱；`change_password` 时不用传 |
| captchaId | string | 是 | 图片验证码 ID |
| captchaCode | string | 是 | 图片验证码答案 |

### 更新当前用户资料

```text
PUT /api/v1/users/me
```

权限：登录用户。修改邮箱时必须先通过 `update_email` 用途邮箱验证码。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 否 | 邮箱；字段缺失或为空时保留原邮箱 |
| emailCode | string | 条件必填 | 仅当 `email` 变更时必填 |
| avatarUrl | string | 否 | 字段缺失时保留原值；显式空字符串表示不显示头像 |
| nickname | string | 否 | 字段缺失时保留原值；显式空字符串表示清空，非空时长度 1 到 64 |

### 修改密码

```text
PUT /api/v1/users/me/password
```

权限：登录用户。修改成功后，用户已有 Refresh Token 会被撤销。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| oldPassword | string | 是 | 当前密码 |
| newPassword | string | 是 | 新密码，长度 8 到 128 |
| emailCode | string | 是 | 当前邮箱收到的 `change_password` 用途验证码 |

## 文章接口

### 文章列表

```text
GET /api/v1/articles
```

权限：公开。匿名访问只返回已发布且未到未来发布时间的文章。列表按置顶、显示优先级、发布时间和 ID 倒序排列。

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| page | int | 否 | 页码 |
| size | int | 否 | 每页数量 |
| status | string | 否 | `draft`、`published`、`archived`、`scheduled`；公开接口匿名访问仍只返回可见文章 |
| q | string | 否 | 关键词搜索，启用 Meilisearch 时优先返回高亮搜索结果，失败时回退 MySQL FULLTEXT / LIKE |
| categoryId | uint64 | 否 | 按分类 ID 筛选 |
| tagId | uint64 | 否 | 按标签 ID 筛选 |

`scheduled` 是列表筛选语义，表示 `status = published` 且 `scheduledAt` 在未来。文章创建和更新时仍使用 `published` 状态加未来 `scheduledAt` 表示定时发布。

### 文章详情

```text
GET /api/v1/articles/:id
```

权限：公开。只允许访问已公开可见文章，访问成功会增加浏览量。

文章响应主要字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uint64 | 文章 ID |
| authorId | uint64 | 作者 ID |
| categoryId | uint64 | 分类 ID，可能省略 |
| title | string | 标题 |
| slug | string | 唯一路径 |
| summary | string | 摘要 |
| content | string | 正文，仅详情、预览和版本详情返回 |
| coverUrl | string | 封面 URL |
| status | string | `draft`、`published` 或 `archived` |
| viewCount | uint64 | 浏览量 |
| likeCount | uint64 | 点赞数 |
| scheduledAt | string | 定时发布时间，可能省略 |
| publishedAt | string | 发布时间，可能省略 |
| isPinned | bool | 是否置顶 |
| displayPriority | int | 显示优先级，0 到 9999，数值越大越靠前 |
| seoTitle | string | SEO 标题 |
| seoDescription | string | SEO 描述 |
| seoKeywords | string | SEO 关键词 |
| tags | Tag[] | 标签列表，可能省略 |
| category | Category | 分类信息，可能省略 |
| searchHighlights | object | 搜索列表命中时可能返回，包含 `title`、`summary`、`content` 高亮片段 |

### 文章上下文

```text
GET /api/v1/articles/:id/context
```

权限：公开。返回上一篇、下一篇和相关文章，只针对公开可见文章，并遵循置顶与优先级排序。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "related": []
  }
}
```

### 创建文章

```text
POST /api/v1/articles
```

权限：`editor` 或 `admin`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| categoryId | uint64 | 否 | 分类 ID，传入时必须存在 |
| title | string | 是 | 标题，长度 1 到 160 |
| slug | string | 是 | 唯一路径，长度 1 到 180，会转为小写并去除首尾空格 |
| summary | string | 否 | 摘要 |
| content | string | 是 | Markdown 内容 |
| coverUrl | string | 否 | 封面 URL，非空必须为 `http://` 或 `https://` URL |
| status | string | 否 | `draft`、`published`、`archived`，默认 `draft` |
| scheduledAt | string | 否 | 定时发布时间；配合 `published` 且时间在未来时表现为定时发布 |
| isPinned | bool | 否 | 是否置顶，默认 `false` |
| displayPriority | int | 否 | 显示优先级，0 到 9999，默认 0 |
| seoTitle | string | 否 | SEO 标题，非空时最长 160 |
| seoDescription | string | 否 | SEO 描述，非空时最长 255 |
| seoKeywords | string | 否 | SEO 关键词，非空时最长 255 |
| tagIds | uint64[] | 否 | 标签 ID 列表，传入时必须存在 |

### 更新文章

```text
PUT /api/v1/articles/:id
```

权限：文章作者、`editor` 或 `admin`。请求体使用完整文章字段。更新前会自动保存一份文章版本。

### 文章预览

```text
GET /api/v1/articles/:id/preview
```

权限：文章作者、`editor` 或 `admin`。可查看草稿、归档和定时发布文章的完整内容。

### 删除文章

```text
DELETE /api/v1/articles/:id
```

权限：文章作者、`editor` 或 `admin`。

### 更新文章状态

```text
PATCH /api/v1/articles/:id/status
```

权限：文章作者、`editor` 或 `admin`。更新前会自动保存一份文章版本。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| status | string | 是 | `draft`、`published`、`archived` |

### AI 辅助创作

```text
POST /api/v1/articles/ai/assist
```

权限：`editor` 或 `admin`。需要管理员先在后台 AI 设置中保存并启用有效的 API 格式、基础地址、API Key 和模型；运行配置只从数据库读取。接口有 IP 限流保护。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| action | string | 是 | `complete`、`metadata`、`proofread`、`polish`、`expand`、`shorten` 或 `translate` |
| title | string | 否 | 文章标题，最长 160 |
| content | string | 是 | Markdown 正文，最长 30000 |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| title | string | `complete` 时可能返回的文章标题，最长 160 |
| slug | string | `complete` 时可能返回的小写 ASCII kebab-case slug，最长 180 |
| summary | string | `complete` 或 `metadata` 时可能返回的摘要 |
| seoTitle | string | `complete` 时可能返回的 SEO 标题 |
| seoDescription | string | `complete` 或 `metadata` 时可能返回的 SEO 描述 |
| seoKeywords | string | `complete` 或 `metadata` 时可能返回的 SEO 关键词 |
| categorySuggestion | string | `complete` 时可能返回的分类文字建议，不会自动创建或选择分类 |
| tagSuggestions | string[] | `complete` 时可能返回的标签文字建议，去重后最多 5 项，不会自动创建或选择标签 |
| revisedContent | string | `proofread`、`polish`、`expand`、`shorten`、`translate` 返回的修订正文 |
| suggestions | string[] | 修改建议 |

`complete` 会同时尝试返回标题、slug、摘要、SEO 标题、SEO 描述、SEO 关键词及分类/标签建议。后台文章编辑页只把有效结果写入当前为空的文本字段，不覆盖人工内容；分类和标签仅显示在独立建议卡片中。`metadata` 继续用于保存文章时的轻量摘要/SEO 补全。

文章补全使用后台已保存的 `firstByteTimeoutSeconds` 和 `nonStreamTimeoutSeconds`，前端不再设置独立的固定 AI 请求时限；上游超时返回 `504`，同时仍受服务端和反向代理安全上限约束。

### Markdown 导入

```text
POST /api/v1/articles/import
```

权限：`editor` 或 `admin`。请求体为 `multipart/form-data`，文件字段名为 `file`，仅支持 `.md`，最大 2 MiB。导入成功后创建草稿文章，若分类或标签不存在会自动创建。

支持的 Front Matter 字段包括：`title`、`slug`、`summary`、`category`、`tags`、`date`、`cover_url`、`pinned`、`priority`、`seo_title`、`seo_description`、`seo_keywords`。若没有 `title`，会尝试使用正文第一个 H1 或文件名。

### Markdown 导出

```text
GET /api/v1/articles/:id/export
```

权限：文章作者、`editor` 或 `admin`。响应为 `text/markdown; charset=utf-8` 文件流，不使用统一 JSON 响应包装。

### 文章版本

#### 文章版本列表

```text
GET /api/v1/articles/:id/versions
```

#### 单个文章版本详情

```text
GET /api/v1/articles/:id/versions/:versionNo
```

#### 恢复文章版本

```text
POST /api/v1/articles/:id/versions/:versionNo/restore
```

权限：文章作者、`editor` 或 `admin`。版本列表支持 `page`、`size`；列表不返回版本正文，详情返回完整内容。恢复版本前会先保存当前文章为新版本。

版本响应包含文章的主要字段、`tagIds`、`originalCreatedAt`、`originalUpdatedAt`、`isPinned` 和 `displayPriority`。

## 分类与标签接口

### 分类

```text
GET /api/v1/categories
POST /api/v1/categories
PUT /api/v1/categories/:id
DELETE /api/v1/categories/:id
```

读取权限：公开。写入权限：`editor` 或 `admin`。创建和更新字段如下：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 名称，长度 1 到 64 |
| slug | string | 是 | 唯一路径，长度 1 到 96 |
| description | string | 否 | 描述 |

若分类仍被文章引用，删除可能失败。

### 标签

```text
GET /api/v1/tags
POST /api/v1/tags
PUT /api/v1/tags/:id
DELETE /api/v1/tags/:id
```

读取权限：公开。写入权限：`editor` 或 `admin`。创建和更新字段同分类。

## 站点接口

### 获取站点设置

```text
GET /api/v1/site/settings
```

权限：公开。返回当前游客是否可以注册账号、注册是否需要邮箱验证码、首页文章列表布局、前台页面控制和站点 SEO 信息。若用户表为空，即使后台开关保存为关闭，也会返回 `registrationEnabled = true`，确保首个注册用户仍可成为管理员。`registrationEmailCodeRequired` 仅在“用户表为空且邮箱服务未启用”时为 `false`。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "registrationEnabled": true,
    "registrationEmailCodeRequired": false,
    "homeArticleLayout": "standard",
    "siteTitle": "Notes of Ashen",
    "siteDescription": "A personal blog written slowly by the lamp of ink.",
    "siteKeywords": "blog,notes,writing",
    "siteBaseUrl": "",
    "projectsPageEnabled": false,
    "projectsNavHidden": true
  }
}
```

### 文章点赞

```text
POST /api/v1/articles/:id/like
```

权限：公开。仅允许点赞公开可见文章。后端基于访客哈希做幂等限制，不保存原始 IP；前端也会使用 `localStorage` 做本地重复点击限制。

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| liked | bool | 本次请求是否新增点赞 |
| likeCount | uint64 | 当前文章点赞数 |

### 更新站点设置

```text
PUT /api/v1/admin/site/settings
```

权限：`admin`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| registrationEnabled | bool | 否 | 是否允许后续用户注册；不传时保留当前值 |
| homeArticleLayout | string | 否 | `standard` 或 `alternating`；字段缺失或空值时保留当前值 |
| siteTitle | string | 否 | 站点标题；字段缺失或空值时保留当前值 |
| siteDescription | string | 否 | 站点描述；字段缺失或空值时保留当前值 |
| siteKeywords | string | 否 | 站点关键词；字段缺失或空值时保留当前值 |
| siteBaseUrl | string | 否 | 字段缺失时保留当前值；显式空字符串表示清空，非空必须为 `http://` 或 `https://` URL |
| projectsPageEnabled | bool | 否 | 是否启用 `/projects` 项目页面；不传时保留当前值 |
| projectsNavHidden | bool | 否 | 是否在前台导航隐藏项目入口；不传时保留当前值 |

所有可选字段缺失时均保留当前值。可选布尔字段显式传入 `false` 会把对应开关更新为关闭；仅 `siteBaseUrl` 支持用显式空字符串清空。

### 获取项目页面内容

```text
GET /api/v1/site/projects
```

权限：公开。仅当后台启用 `/projects` 页面时可访问，否则返回 `403 feature disabled`。项目数组顺序即前台展示顺序。

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| title | string | 页面标题 |
| subtitle | string | 页面副标题 |
| items | object[] | 项目列表 |

项目字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | string | 项目稳定 ID |
| title | string | 项目标题 |
| summary | string | 项目摘要 |
| role | string | 角色或职责 |
| period | string | 项目周期 |
| tags | string[] | 标签 |
| tagIds | uint64[] | 关联的标签 ID，后台读取时返回 |
| coverUrl | string | 封面 URL，可为空 |
| demoUrl | string | 演示链接，可为空 |
| repoUrl | string | 代码仓库链接，可为空 |
| contentMarkdown | string | 项目详情 Markdown |
| featured | bool | 是否精选 |

### 更新项目页面内容

```text
GET /api/v1/admin/site/projects
PUT /api/v1/admin/site/projects
```

权限：`admin`。后台读取不受公开页面启用状态影响。保存时 `items` 数组顺序会作为展示顺序。

更新字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| title | string | 是 | 页面标题，长度 1 到 160 |
| subtitle | string | 否 | 页面副标题，最长 255 |
| items | object[] | 是 | 项目列表，最多 50 个 |

项目更新字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| id | string | 否 | 项目 ID；为空时后端会按顺序生成 |
| title | string | 是 | 项目标题，长度 1 到 120 |
| summary | string | 否 | 摘要，最长 500 |
| role | string | 否 | 角色或职责，最长 80 |
| period | string | 否 | 项目周期，最长 80 |
| tags | string[] | 否 | 最多 12 个，每个最长 32；保存时会去空和去重 |
| tagIds | uint64[] | 否 | 关联现有标签 ID，传入时必须存在 |
| coverUrl | string | 否 | 封面 URL，非空必须为 `http://` 或 `https://` |
| demoUrl | string | 否 | 演示 URL，非空必须为 `http://` 或 `https://` |
| repoUrl | string | 否 | 代码仓库 URL，非空必须为 `http://` 或 `https://` |
| contentMarkdown | string | 否 | 项目详情 Markdown，最长 50000 字符 |
| featured | bool | 否 | 是否精选 |

### RSS 与 Sitemap

```text
GET /rss.xml
GET /sitemap.xml
```

权限：公开。分别返回公开文章的 RSS XML 和站点 XML Sitemap；查询只读取标题、摘要和时间等轻量字段，不读取文章正文。RSS 按发布时间保留最近 50 篇公开文章；Sitemap 在单文件最多 50,000 个 URL 的限制内纳入全部公开文章，超出部分不在本文件中输出。已启用的 `/projects` 页面也会进入 Sitemap，RSS 中会作为静态页面条目输出。

## 流量接口

### 记录访问

```text
POST /api/v1/traffic/visit
```

权限：公开。前端自动上报公开页面访问，接口有 IP 限流保护。后端仅记录公开页面路径、PV/UV 与访问来源，后台、登录、注册、个人资料和找回密码等路径会被忽略。

访客 IP 默认来自直连 `RemoteAddr`；只有请求来自 `APP_TRUSTED_PROXY_CIDRS` 中配置的可信代理时，才会读取 `X-Forwarded-For` 或 `X-Real-IP`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| path | string | 是 | 当前访问路径 |
| routeType | string | 是 | 前端路由类型，例如 `home`、`article`、`archive`、`search` |
| articleId | uint64 | 否 | 文章详情页对应的文章 ID |
| referrer | string | 否 | 浏览器 referrer |

## 管理接口

### 后台统计

```text
GET /api/v1/admin/stats
```

权限：`editor` 或 `admin`。返回文章、用户、分类、标签、访问统计、热门文章、最近文章和最近操作日志。

响应 `data` 主要字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| articleTotal | int64 | 文章总数 |
| publishedTotal | int64 | 已公开发布文章数 |
| draftTotal | int64 | 草稿数 |
| archivedTotal | int64 | 归档数 |
| scheduledTotal | int64 | 定时发布数 |
| viewTotal | uint64 | 文章浏览总量 |
| likeTotal | uint64 | 文章点赞总量 |
| todayPv | int64 | 今日 PV |
| todayUv | int64 | 今日 UV |
| trafficTrend | object[] | 最近 30 天 PV / UV 趋势，字段为 `date`、`pv`、`uv` |
| topReferers | object[] | 最近 30 天访问来源排行，字段为 `sourceType`、`sourceName`、`pv` |
| userTotal | int64 | 用户数 |
| categoryTotal | int64 | 分类数 |
| tagTotal | int64 | 标签数 |
| popularArticles | Article[] | 热门文章 |
| recentArticles | Article[] | 最近文章 |
| recentLogs | OperationLog[] | 最近操作日志 |

### 后台文章列表

```text
GET /api/v1/admin/articles
```

权限：`editor` 或 `admin`。支持 `page`、`size`、`status`、`q`、`categoryId`、`tagId`。内容管理角色可查看全部文章。

### 后台 AI 设置

```text
GET /api/v1/admin/ai/settings
PUT /api/v1/admin/ai/settings
```

权限：`admin`。用于读取和保存数据库中的 AI 辅助创作配置。读取接口不会返回明文 API Key，只返回是否已配置以及密文是否需要更新。保存新 API Key 时使用由 `APP_AUTH_ACCESS_SECRET` 派生的独立用途密钥生成 `v3:` 密文。

升级兼容规则：`v2:` 密文不能继续解密，需管理员重新录入 API Key；无版本前缀的旧密文仍兼容读取，并会在下一次成功保存设置时迁移到 `v3:`。如果轮换 `APP_AUTH_ACCESS_SECRET`，已有 `v3:` 密文也需要重新录入。

更新字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| enabled | bool | 是 | 是否启用后台 AI 辅助创作 |
| apiFormat | string | 否 | API 格式：`openai` 或 `anthropic`；字段缺失时保留当前值，旧配置缺失时默认为 `openai` |
| baseUrl | string | 否 | 字段缺失时保留当前值；非空必须为解析到公网地址的 `http://` 或 `https://` URL，AI 关闭时可用显式空字符串清空 |
| apiKey | string | 否 | 新 API Key；为空表示保留已保存密钥 |
| clearApiKey | bool | 否 | 是否清空已保存 API Key；传入新 `apiKey` 时应为 `false` |
| model | string | 否 | 字段缺失时保留当前值；模型名称最长 120，AI 关闭时可用显式空字符串清空 |
| firstByteTimeoutSeconds | int | 否 | 字段缺失时保留当前值；显式 `0` 恢复默认 60，其他值范围 1 到 1800 |
| nonStreamTimeoutSeconds | int | 否 | 字段缺失时保留当前值；显式 `0` 恢复默认 600，其他值范围 1 到 1800，不能小于首字等待且不能超过服务端请求超时 |

字段缺失一律保留当前值。切换到非空的新 `apiFormat` 或 `baseUrl` 组合时，若原来已保存 API Key，必须同时传入新 `apiKey` 或设置 `clearApiKey = true`，避免把旧服务商密钥误发给新端点；关闭 AI 并清空地址/模型时可继续保留密钥。启用 AI 时 `baseUrl`、`model` 和可正常解密的 API Key 均为必需项。

AI 出站请求在每次新建连接时重新解析域名，解析结果只要包含私网、本机、链路本地、CGNAT、文档或保留地址就整体拒绝，并直接连接已校验的公网 IP；请求不使用环境 HTTP 代理，也不跟随重定向，以防 DNS 重绑定或代理端重新解析绕过校验。

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| enabled | bool | 当前是否启用 |
| apiFormat | string | 当前 API 格式：`openai` 或 `anthropic` |
| baseUrl | string | 当前基础地址 |
| model | string | 当前模型 |
| apiKeyConfigured | bool | 是否已保存 API Key 密文 |
| apiKeyNeedsUpdate | bool | 已保存密文是否无法继续使用；为 `true` 时必须重新录入 API Key |
| firstByteTimeoutSeconds | int | 首字等待超时 |
| nonStreamTimeoutSeconds | int | 非流式输出总超时 |

### 获取 AI 模型列表

```text
POST /api/v1/admin/ai/models
```

权限：`admin`。使用请求中的草稿连接配置访问上游模型列表，无需先保存设置。若 `apiKey` 留空，仅当 `apiFormat` 和标准化后的 `baseUrl` 与已保存设置一致时才会复用已保存且可解密的 API Key；测试尚未保存的新端点时必须传入 `apiKey`。

`anthropic` 格式遵循标准 SDK 的 Base URL 语义：基础地址未包含 `/v1` 且不是完整的 `/messages` 或 `/models` 端点时，后端会补全 `/v1/models`；测试模型时对应补全 `/v1/messages`。显式完整端点和原有查询参数保持不变。

请求字段校验失败返回 `400`；上游拒绝、协议错误或无效响应统一返回 `502`，上游超时返回 `504`。不会把上游的 `401/403` 原样透传为本站认证失败，也不会回显上游响应正文或 API Key。

请求字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| apiFormat | string | 否 | `openai` 或 `anthropic`，留空按 `openai` 处理 |
| baseUrl | string | 是 | 待连接的上游基础地址，必须为解析到公网地址的 `http://` 或 `https://` URL |
| apiKey | string | 否 | 草稿 API Key；满足上述条件时可复用已保存密钥 |
| firstByteTimeoutSeconds | int | 否 | 首字等待超时，默认 60，范围 1 到 1800 |
| nonStreamTimeoutSeconds | int | 否 | 请求总超时，默认 600，范围 1 到 1800，不能小于首字等待且不能超过服务端请求超时 |

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| models | string[] | 上游返回的模型 ID，去空、去重并按字典序排序 |

### 测试 AI 模型

```text
POST /api/v1/admin/ai/test
```

权限：`admin`。使用草稿配置向指定模型发送固定的小型 JSON 探针，并预留 512 个输出 token 以兼容先生成思考块的模型；连接成功且响应格式有效时返回模型名称和完整请求耗时。`apiKey` 的复用规则与模型列表接口相同。

失败状态码与模型列表接口一致；测试不会保存草稿配置、启用 AI 或返回模型生成正文。

请求字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| apiFormat | string | 否 | `openai` 或 `anthropic`，留空按 `openai` 处理 |
| baseUrl | string | 是 | 待测试的上游基础地址，必须为解析到公网地址的 `http://` 或 `https://` URL |
| apiKey | string | 否 | 草稿 API Key；满足复用条件时可留空 |
| model | string | 是 | 待测试模型，长度 1 到 120 |
| firstByteTimeoutSeconds | int | 否 | 首字等待超时，默认 60，范围 1 到 1800 |
| nonStreamTimeoutSeconds | int | 否 | 请求总超时，默认 600，范围 1 到 1800，不能小于首字等待且不能超过服务端请求超时 |

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| model | string | 已通过探针测试的模型名称 |
| latencyMs | int64 | 完整探针请求耗时，单位毫秒，最小返回 1 |

### 重建搜索索引

```text
POST /api/v1/admin/search/reindex
```

权限：`editor` 或 `admin`。将已发布文章全量同步到 Meilisearch。未启用搜索时返回 `enabled = false`，不会修改文章数据。

响应 `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| indexed | int | 本次写入搜索索引的文章数 |
| enabled | bool | 当前是否启用 Meilisearch 搜索 |

### 用户管理

```text
GET /api/v1/admin/users
PATCH /api/v1/admin/users/:id/status
PATCH /api/v1/admin/users/:id/role
```

权限：`admin`。用户列表支持 `page`、`size`。

修改用户状态：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| status | string | 是 | `active` 或 `disabled` |

修改用户角色：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| role | string | 是 | `user`、`editor` 或 `admin` |

不能禁用自己，不能将自己降级，也不能禁用或降级最后一个可用管理员。

### 操作日志列表

```text
GET /api/v1/admin/logs
```

权限：`admin`。

| 查询参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| page | int | 否 | 页码，默认 `1` |
| size | int | 否 | 每页数量，默认 `10`，最大 `100` |
| eventType | string | 否 | 事件代码精确匹配，例如 `article.updated` |
| actor | string | 否 | 纯数字时精确匹配用户 ID，否则按用户账号模糊匹配 |
| ip | string | 否 | IPv4/IPv6 精确匹配，格式无效时返回 `400` |
| startAt | string | 否 | RFC3339 开始时间，包含该时刻 |
| endAt | string | 否 | RFC3339 结束时间，不包含该时刻；与 `startAt` 同时存在时必须更晚 |

筛选条件之间使用 AND 组合。响应中的 `userId`、`userAccount`、`resourceId`、`metadata` 可能被省略；`userAccount` 为该操作发起者的账号（来自 `users` 表，可能因用户被删除而为空）。响应结构与未筛选列表保持一致。

## PowerShell 调用示例

注册前先获取验证码并发送邮箱验证码，下面示例只展示最终注册请求体结构。若当前没有任何用户且邮箱服务未启用，首个管理员注册可省略 `emailCode`。

```powershell
$body = @{
  account = "admin"
  password = "Password123!"
  email = "admin@example.com"
  emailCode = "123456"
  nickname = "站长"
  avatarUrl = "https://example.com/avatar.png"
} | ConvertTo-Json

$register = Invoke-RestMethod -Method Post `
  -Uri "http://127.0.0.1:19000/api/v1/auth/register" `
  -ContentType "application/json" `
  -Body $body

$accessToken = $register.data.accessToken
$refreshToken = $register.data.refreshToken
```

访问受保护接口：

```powershell
$headers = @{ Authorization = "Bearer $accessToken" }

Invoke-RestMethod -Method Get `
  -Uri "http://127.0.0.1:19000/api/v1/users/me" `
  -Headers $headers
```
