# Notes of Ashen API 文档

本文档描述 Notes of Ashen 当前实现的 HTTP API。默认服务地址：

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

请求示例：

```json
{
  "account": "admin",
  "password": "Password123!",
  "email": "admin@example.com",
  "nickname": "站长",
  "avatarUrl": "https://example.com/avatar.png"
}
```

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

权限：公开。匿名访问只返回 `published` 状态文章。

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| page | int | 否 | 页码 |
| size | int | 否 | 每页数量 |
| status | string | 否 | `draft`、`published`、`archived` |
| q | string | 否 | 关键词搜索，匹配标题、摘要、正文 |
| categoryId | uint64 | 否 | 按分类 ID 筛选 |
| tagId | uint64 | 否 | 按标签 ID 筛选 |

响应中的 `categoryId`、`publishedAt`、`tags`、`category` 可能被省略。

### 文章详情

```text
GET /api/v1/articles/:id
```

权限：公开。只允许访问 `published` 状态文章，访问成功会增加浏览量。

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
| tagIds | uint64[] | 否 | 标签 ID 列表，传入时必须存在 |

### 更新文章

```text
PUT /api/v1/articles/:id
```

权限：文章作者或管理员。请求体使用完整文章字段，普通用户只能更新自己的文章。

### 删除文章

```text
DELETE /api/v1/articles/:id
```

权限：文章作者或管理员。

### 更新文章状态

```text
PATCH /api/v1/articles/:id/status
```

权限：文章作者或管理员。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| status | string | 是 | `draft`、`published`、`archived` |

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

权限：管理员。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 名称，长度 1 到 64 |
| slug | string | 是 | 唯一路径，长度 1 到 96 |
| description | string | 否 | 描述 |

### 更新分类

```text
PUT /api/v1/categories/:id
```

权限：管理员。

### 删除分类

```text
DELETE /api/v1/categories/:id
```

权限：管理员。若分类仍被文章引用，删除可能失败。

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

权限：管理员。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 名称，长度 1 到 64 |
| slug | string | 是 | 唯一路径，长度 1 到 96 |
| description | string | 否 | 描述 |

### 更新标签

```text
PUT /api/v1/tags/:id
```

权限：管理员。

### 删除标签

```text
DELETE /api/v1/tags/:id
```

权限：管理员。

## 管理接口

### 用户列表

```text
GET /api/v1/admin/users
```

权限：管理员。支持 `page`、`size`。

### 修改用户状态

```text
PATCH /api/v1/admin/users/:id/status
```

权限：管理员。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| status | string | 是 | `active` 或 `disabled` |

### 操作日志列表

```text
GET /api/v1/admin/logs
```

权限：管理员。支持 `page`、`size`。响应中的 `userId`、`resourceId`、`metadata` 可能被省略。

### 后台文章列表

```text
GET /api/v1/admin/articles
```

权限：管理员。支持 `page`、`size`、`status`、`q`、`categoryId`、`tagId`。

### 获取站点设置

```text
GET /api/v1/site/settings
```

权限：公开。返回当前游客是否可以注册账号。若用户表为空，即使后台开关保存为关闭，也会返回 `registrationEnabled = true`，确保首个注册用户仍可成为管理员。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "registrationEnabled": true
  }
}
```

### 更新站点设置

```text
PUT /api/v1/admin/site/settings
```

权限：管理员。用于开启或关闭后续账号注册。

请求示例：

```json
{
  "registrationEnabled": false
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
