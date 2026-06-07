# Notes of Ashen API 文档

本文档描述 Notes of Ashen 当前实现的 HTTP API。默认开发服务地址：

```text
http://127.0.0.1:19000
```

## 通用约定

除 `GET` 接口外，请求体默认使用 JSON：

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
| 40100 | 401 | 未登录、Token 缺失、Token 无效或过期 |
| 40300 | 403 | 权限不足、注册关闭或用户被禁用 |
| 40400 | 404 | 资源不存在 |
| 40900 | 409 | 资源冲突，例如账号、邮箱、slug 重复 |
| 50000 | 500 | 服务内部错误 |

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

## 配置安全提示

`etc/notes-of-ashen.yaml` 是本地开发默认配置，包含示例数据库密码、Redis 密码、JWT Secret 和 RabbitMQ 地址。生产环境部署前必须替换这些值，并通过受控配置或环境注入管理敏感信息，不能直接复用仓库中的默认值。

## 认证接口

### 注册

```text
POST /api/v1/auth/register
```

权限：公开。第一个注册用户自动成为 `admin`，后续用户默认为 `user`。若站点注册开关关闭，后续注册会被拒绝。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| account | string | 是 | 账号，长度 3 到 64 |
| password | string | 是 | 密码，长度 8 到 128 |
| email | string | 是 | 邮箱，需符合邮箱格式 |
| nickname | string | 否 | 昵称，非空时长度 1 到 64 |
| avatarUrl | string | 否 | 头像 URL；为空表示不显示头像，非空必须为 `http://` 或 `https://` URL |

### 登录

```text
POST /api/v1/auth/login
```

权限：公开。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| account | string | 是 | 账号或邮箱 |
| password | string | 是 | 密码 |

### 刷新 Token

```text
POST /api/v1/auth/refresh
```

权限：公开。刷新成功后，旧 Refresh Token 会被撤销，并返回新的 Access Token 和 Refresh Token。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| refreshToken | string | 是 | 登录或注册返回的 Refresh Token |

Token 响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "jwt",
    "refreshToken": "refresh-token",
    "tokenType": "Bearer",
    "expiresIn": 7200
  }
}
```

### 退出登录

```text
POST /api/v1/auth/logout
```

权限：登录用户。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| refreshToken | string | 是 | 要撤销的 Refresh Token |

## 用户接口

### 获取当前用户

```text
GET /api/v1/users/me
```

权限：登录用户。

### 更新当前用户资料

```text
PUT /api/v1/users/me
```

权限：登录用户。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 否 | 邮箱；为空时保留原邮箱 |
| avatarUrl | string | 否 | 头像 URL；为空表示不显示头像 |
| nickname | string | 否 | 昵称，非空时长度 1 到 64 |

### 修改密码

```text
PUT /api/v1/users/me/password
```

权限：登录用户。修改成功后，用户已有 Refresh Token 会被撤销。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| oldPassword | string | 是 | 当前密码 |
| newPassword | string | 是 | 新密码，长度 8 到 128 |

## 文章接口

### 文章列表

```text
GET /api/v1/articles
```

权限：公开。匿名访问只返回已发布且未到未来发布时间的文章。

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| page | int | 否 | 页码 |
| size | int | 否 | 每页数量 |
| status | string | 否 | `draft`、`published`、`archived`、`scheduled`；公开接口匿名访问仍只返回可见文章 |
| q | string | 否 | 关键词搜索，匹配标题、摘要、正文 |
| categoryId | uint64 | 否 | 按分类 ID 筛选 |
| tagId | uint64 | 否 | 按标签 ID 筛选 |

`scheduled` 是列表筛选语义，表示 `status = published` 且 `scheduledAt` 在未来。文章创建和更新时仍使用 `published` 状态加未来 `scheduledAt` 表示定时发布。

### 文章详情

```text
GET /api/v1/articles/:id
```

权限：公开。只允许访问已公开可见文章，访问成功会增加浏览量。

响应中的 `content` 仅详情、预览和版本详情接口返回。`categoryId`、`scheduledAt`、`publishedAt`、`tags`、`category` 可能被省略。

### 文章上下文

```text
GET /api/v1/articles/:id/context
```

权限：公开。返回上一篇、下一篇和相关文章，只针对公开可见文章。

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

权限：登录用户。

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

### 文章版本列表

```text
GET /api/v1/articles/:id/versions
```

权限：文章作者、`editor` 或 `admin`。支持 `page`、`size`。列表不返回版本正文。

### 文章版本详情

```text
GET /api/v1/articles/:id/versions/:versionNo
```

权限：文章作者、`editor` 或 `admin`。返回指定版本的完整内容。

### 恢复文章版本

```text
POST /api/v1/articles/:id/versions/:versionNo/restore
```

权限：文章作者、`editor` 或 `admin`。恢复前会先保存当前文章为新版本。

## 分类接口

### 分类列表

```text
GET /api/v1/categories
```

权限：公开。支持 `page`、`size`。

### 创建分类

```text
POST /api/v1/categories
```

权限：`editor` 或 `admin`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 名称，长度 1 到 64 |
| slug | string | 是 | 唯一路径，长度 1 到 96 |
| description | string | 否 | 描述 |

### 更新分类

```text
PUT /api/v1/categories/:id
```

权限：`editor` 或 `admin`。

### 删除分类

```text
DELETE /api/v1/categories/:id
```

权限：`editor` 或 `admin`。若分类仍被文章引用，删除可能失败。

## 标签接口

### 标签列表

```text
GET /api/v1/tags
```

权限：公开。支持 `page`、`size`。

### 创建标签

```text
POST /api/v1/tags
```

权限：`editor` 或 `admin`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 名称，长度 1 到 64 |
| slug | string | 是 | 唯一路径，长度 1 到 96 |
| description | string | 否 | 描述 |

### 更新标签

```text
PUT /api/v1/tags/:id
```

权限：`editor` 或 `admin`。

### 删除标签

```text
DELETE /api/v1/tags/:id
```

权限：`editor` 或 `admin`。

## 站点接口

### 获取站点设置

```text
GET /api/v1/site/settings
```

权限：公开。返回当前游客是否可以注册账号、首页文章列表布局和站点 SEO 信息。若用户表为空，即使后台开关保存为关闭，也会返回 `registrationEnabled = true`，确保首个注册用户仍可成为管理员。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "registrationEnabled": true,
    "homeArticleLayout": "standard",
    "siteTitle": "Notes of Ashen",
    "siteDescription": "A personal blog written slowly by the lamp of ink.",
    "siteKeywords": "blog,notes,writing",
    "siteBaseUrl": ""
  }
}
```

### RSS

```text
GET /rss.xml
```

权限：公开。返回公开文章的 RSS XML。

### Sitemap

```text
GET /sitemap.xml
```

权限：公开。返回站点 XML Sitemap。

## 管理接口

### 后台统计

```text
GET /api/v1/admin/stats
```

权限：`editor` 或 `admin`。返回文章、用户、分类、标签数量，以及热门文章、最近文章和最近操作日志。

### 后台文章列表

```text
GET /api/v1/admin/articles
```

权限：`editor` 或 `admin`。支持 `page`、`size`、`status`、`q`、`categoryId`、`tagId`。内容管理角色可查看全部文章。

### 用户列表

```text
GET /api/v1/admin/users
```

权限：`admin`。支持 `page`、`size`。

### 修改用户状态

```text
PATCH /api/v1/admin/users/:id/status
```

权限：`admin`。不能禁用自己，也不能禁用最后一个可用管理员。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| status | string | 是 | `active` 或 `disabled` |

### 修改用户角色

```text
PATCH /api/v1/admin/users/:id/role
```

权限：`admin`。不能将自己降级，也不能降级最后一个可用管理员。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| role | string | 是 | `user`、`editor` 或 `admin` |

### 操作日志列表

```text
GET /api/v1/admin/logs
```

权限：`admin`。支持 `page`、`size`。响应中的 `userId`、`resourceId`、`metadata` 可能被省略。

### 更新站点设置

```text
PUT /api/v1/admin/site/settings
```

权限：`admin`。用于开启或关闭后续账号注册，设置首页文章列表布局和站点 SEO 信息。`homeArticleLayout` 可选值为 `standard` 或 `alternating`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| registrationEnabled | bool | 是 | 是否允许后续用户注册 |
| homeArticleLayout | string | 是 | `standard` 或 `alternating` |
| siteTitle | string | 否 | 站点标题，空值表示保留当前值 |
| siteDescription | string | 否 | 站点描述，空值表示保留当前值 |
| siteKeywords | string | 否 | 站点关键词，空值表示保留当前值 |
| siteBaseUrl | string | 否 | 站点公开基础 URL，非空必须为 `http://` 或 `https://` URL |

请求示例：

```json
{
  "registrationEnabled": false,
  "homeArticleLayout": "alternating",
  "siteTitle": "Notes of Ashen",
  "siteDescription": "A personal blog written slowly by the lamp of ink.",
  "siteKeywords": "blog,notes,writing",
  "siteBaseUrl": "https://example.com"
}
```

## PowerShell 调用示例

注册：

```powershell
$body = @{
  account = "admin"
  password = "Password123!"
  email = "admin@example.com"
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

## 邮箱验证码、图片验证码与限流补充

新增错误码：

| code | HTTP 状态码 | 含义 |
| --- | --- | --- |
| 42900 | 429 | 请求过于频繁，常见于登录、发送验证码或重置密码限流 |

### 获取图片验证码

```text
POST /api/v1/auth/captcha
```

请求字段：

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

用于注册和找回密码。发送前必须提交同用途图片验证码，同 IP 1 分钟最多 5 次。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 是 | 接收验证码的邮箱 |
| purpose | string | 是 | `register` 或 `reset_password` |
| captchaId | string | 是 | 图片验证码 ID |
| captchaCode | string | 是 | 图片验证码答案 |

### 注册

`POST /api/v1/auth/register` 新增字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| emailCode | string | 是 | `register` 用途邮箱验证码；第一个管理员账号也必须校验 |

### 登录

`POST /api/v1/auth/login` 新增字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| captchaId | string | 是 | `login` 用途图片验证码 ID |
| captchaCode | string | 是 | 图片验证码答案 |

### 找回密码

```text
POST /api/v1/auth/password/reset
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 是 | 账号绑定邮箱 |
| emailCode | string | 是 | `reset_password` 用途邮箱验证码 |
| newPassword | string | 是 | 新密码，长度 8 到 128 |

重置成功后会撤销该用户已有 Refresh Token，需要重新登录。

### 登录态发送邮箱验证码

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

### 修改当前用户资料

`PUT /api/v1/users/me` 在修改邮箱时新增字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| emailCode | string | 条件必填 | 仅当 `email` 变更时必填，校验 `update_email` 用途验证码 |

### 修改密码

`PUT /api/v1/users/me/password` 新增字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| emailCode | string | 是 | 当前邮箱收到的 `change_password` 用途验证码 |
