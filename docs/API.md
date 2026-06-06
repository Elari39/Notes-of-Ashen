# Notes of Ashen API 开放文档

本文档描述 Notes of Ashen 当前实现中的 HTTP API。接口基础地址：

```text
http://127.0.0.1:19000
```

## 通用约定

### 请求格式

除 GET 接口外，请求体默认使用 JSON：

```text
Content-Type: application/json
```

受保护接口需要携带 Access Token：

```text
Authorization: Bearer <accessToken>
```

### 统一响应

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

### 错误码

| code | HTTP 状态码 | 含义 |
| --- | --- | --- |
| 40000 | 400 | 请求参数错误 |
| 40100 | 401 | 未登录、Token 缺失、Token 无效或过期 |
| 40300 | 403 | 权限不足或用户被禁用 |
| 40400 | 404 | 资源不存在 |
| 40900 | 409 | 资源冲突，例如账号、邮箱、slug 重复 |
| 50000 | 500 | 服务内部错误 |

### 分页规则

列表接口统一支持：

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

### 时间格式

时间字段由 Go JSON 编码输出，格式通常为 RFC3339：

```text
2026-06-05T20:00:00+08:00
```

## 数据模型

### TokenPair

```json
{
  "accessToken": "jwt",
  "refreshToken": "refresh-token",
  "tokenType": "Bearer",
  "expiresIn": 7200
}
```

### User

```json
{
  "id": 1,
  "account": "admin",
  "email": "admin@example.com",
  "avatarUrl": "https://example.com/avatar.png",
  "nickname": "Admin",
  "role": "admin",
  "status": "active",
  "createdAt": "2026-06-05T20:00:00+08:00",
  "updatedAt": "2026-06-05T20:00:00+08:00"
}
```

### Article

```json
{
  "id": 1,
  "authorId": 1,
  "categoryId": 1,
  "title": "Notes of Ashen 使用指南",
  "slug": "notes-of-ashen-guide",
  "summary": "一篇示例文章",
  "content": "# Markdown 内容",
  "coverUrl": "https://example.com/cover.png",
  "status": "published",
  "viewCount": 1,
  "publishedAt": "2026-06-05T20:00:00+08:00",
  "createdAt": "2026-06-05T20:00:00+08:00",
  "updatedAt": "2026-06-05T20:00:00+08:00",
  "tags": [],
  "category": null
}
```

文章状态：

| 值 | 说明 |
| --- | --- |
| draft | 草稿 |
| published | 已发布 |
| archived | 归档 |

### Category

```json
{
  "id": 1,
  "name": "技术",
  "slug": "tech",
  "description": "技术文章",
  "createdBy": 1,
  "createdAt": "2026-06-05T20:00:00+08:00",
  "updatedAt": "2026-06-05T20:00:00+08:00"
}
```

### Tag

```json
{
  "id": 1,
  "name": "Go",
  "slug": "go",
  "description": "Go 语言",
  "createdBy": 1,
  "createdAt": "2026-06-05T20:00:00+08:00",
  "updatedAt": "2026-06-05T20:00:00+08:00"
}
```

## 认证接口

### 注册

```text
POST /api/v1/auth/register
```

权限：公开。

说明：首个注册用户自动成为 `admin`，后续用户默认为 `user`。

请求参数：

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

响应示例：

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

常见错误：`40000` 参数格式错误，`40900` 账号或邮箱已存在，`50000` Redis/MySQL 写入失败。

### 登录

```text
POST /api/v1/auth/login
```

权限：公开。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| account | string | 是 | 账号或邮箱 |
| password | string | 是 | 密码 |

请求示例：

```json
{
  "account": "admin",
  "password": "Password123!"
}
```

响应示例：

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

常见错误：`40100` 账号或密码错误，`40300` 用户被禁用。

### 刷新 Token

```text
POST /api/v1/auth/refresh
```

权限：公开。

说明：刷新成功后，旧 Refresh Token 会被撤销，并返回新的 Access Token 和 Refresh Token。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| refreshToken | string | 是 | 登录或注册返回的 Refresh Token |

请求示例：

```json
{
  "refreshToken": "refresh-token"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "new-jwt",
    "refreshToken": "new-refresh-token",
    "tokenType": "Bearer",
    "expiresIn": 7200
  }
}
```

常见错误：`40100` Refresh Token 无效、过期或已撤销，`40300` 用户被禁用。

### 退出登录

```text
POST /api/v1/auth/logout
```

权限：登录用户。

请求头：

```text
Authorization: Bearer <accessToken>
```

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| refreshToken | string | 是 | 要撤销的 Refresh Token |

请求示例：

```json
{
  "refreshToken": "refresh-token"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success"
}
```

常见错误：`40100` Access Token 缺失或无效，`40000` Refresh Token 缺失。

## 用户接口

### 获取当前用户

```text
GET /api/v1/users/me
```

权限：登录用户。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "account": "admin",
    "email": "admin@example.com",
    "avatarUrl": "https://example.com/avatar.png",
    "nickname": "站长",
    "role": "admin",
    "status": "active",
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:00:00+08:00"
  }
}
```

常见错误：`40100` Access Token 缺失或无效，`40400` 用户不存在。

### 更新当前用户资料

```text
PUT /api/v1/users/me
```

权限：登录用户。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 否 | 邮箱；为空时保留原邮箱 |
| avatarUrl | string | 否 | 头像 URL；为空表示不显示头像，非空必须为 `http://` 或 `https://` URL |
| nickname | string | 否 | 昵称，非空时长度 1 到 64 |

请求示例：

```json
{
  "email": "new-admin@example.com",
  "avatarUrl": "https://example.com/new-avatar.png",
  "nickname": "新站长"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "account": "admin",
    "email": "new-admin@example.com",
    "avatarUrl": "https://example.com/new-avatar.png",
    "nickname": "新站长",
    "role": "admin",
    "status": "active",
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:10:00+08:00"
  }
}
```

常见错误：`40000` 邮箱格式错误或昵称长度错误，`40900` 邮箱已存在。

### 修改密码

```text
PUT /api/v1/users/me/password
```

权限：登录用户。

说明：修改成功后，用户已有 Refresh Token 会被撤销。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| oldPassword | string | 是 | 当前密码 |
| newPassword | string | 是 | 新密码，长度 8 到 128 |

请求示例：

```json
{
  "oldPassword": "Password123!",
  "newPassword": "NewPassword123!"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success"
}
```

常见错误：`40000` 新密码长度错误，`40100` 原密码错误或 Token 无效。

## 文章接口

### 文章列表

```text
GET /api/v1/articles
```

权限：公开。

说明：当前公开列表只返回 `published` 状态文章。`status` 参数会被校验，但不会让匿名用户看到草稿或归档文章。

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| page | int | 否 | 页码 |
| size | int | 否 | 每页数量 |
| status | string | 否 | 可选值：`draft`、`published`、`archived` |

请求示例：

```text
GET /api/v1/articles?page=1&size=10
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "authorId": 1,
        "categoryId": 1,
        "title": "Notes of Ashen 使用指南",
        "slug": "notes-of-ashen-guide",
        "summary": "一篇示例文章",
        "coverUrl": "https://example.com/cover.png",
        "status": "published",
        "viewCount": 0,
        "publishedAt": "2026-06-05T20:00:00+08:00",
        "createdAt": "2026-06-05T20:00:00+08:00",
        "updatedAt": "2026-06-05T20:00:00+08:00",
        "tags": [],
        "category": null
      }
    ],
    "total": 1,
    "page": 1,
    "size": 10
  }
}
```

常见错误：`40000` status 非法，`50000` 查询失败。

### 文章详情

```text
GET /api/v1/articles/:id
```

权限：公开。

说明：只允许访问 `published` 状态文章，访问成功会增加浏览量。

路径参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| id | uint64 | 文章 ID |

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "authorId": 1,
    "categoryId": 1,
    "title": "Notes of Ashen 使用指南",
    "slug": "notes-of-ashen-guide",
    "summary": "一篇示例文章",
    "content": "# Markdown 内容",
    "coverUrl": "https://example.com/cover.png",
    "status": "published",
    "viewCount": 1,
    "publishedAt": "2026-06-05T20:00:00+08:00",
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:00:00+08:00",
    "tags": [],
    "category": null
  }
}
```

常见错误：`40000` ID 非法，`40400` 文章不存在或未发布。

### 创建文章

```text
POST /api/v1/articles
```

权限：登录用户。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| categoryId | uint64 | 否 | 分类 ID，传入时必须存在 |
| title | string | 是 | 标题，长度 1 到 160 |
| slug | string | 是 | 唯一路径，长度 1 到 180，会转为小写并去除首尾空格 |
| summary | string | 否 | 摘要 |
| content | string | 是 | Markdown 内容 |
| coverUrl | string | 否 | 封面 URL |
| status | string | 否 | `draft`、`published`、`archived`，默认 `draft` |
| tagIds | uint64[] | 否 | 标签 ID 列表，传入时必须存在 |

请求示例：

```json
{
  "categoryId": 1,
  "title": "Notes of Ashen 使用指南",
  "slug": "notes-of-ashen-guide",
  "summary": "一篇示例文章",
  "content": "# Markdown 内容",
  "coverUrl": "https://example.com/cover.png",
  "status": "published",
  "tagIds": [1]
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "authorId": 1,
    "categoryId": 1,
    "title": "Notes of Ashen 使用指南",
    "slug": "notes-of-ashen-guide",
    "summary": "一篇示例文章",
    "content": "# Markdown 内容",
    "coverUrl": "https://example.com/cover.png",
    "status": "published",
    "viewCount": 0,
    "publishedAt": "2026-06-05T20:00:00+08:00",
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:00:00+08:00",
    "tags": [],
    "category": null
  }
}
```

常见错误：`40000` 参数校验失败，`40100` Token 无效，`40400` 分类或标签不存在，`40900` slug 已存在。

### 更新文章

```text
PUT /api/v1/articles/:id
```

权限：文章作者或管理员。

说明：请求体使用完整文章字段；普通用户只能更新自己的文章。

请求示例：

```json
{
  "categoryId": 1,
  "title": "Notes of Ashen 使用指南 v2",
  "slug": "notes-of-ashen-guide-v2",
  "summary": "更新后的摘要",
  "content": "# 更新后的 Markdown 内容",
  "coverUrl": "https://example.com/cover-v2.png",
  "status": "published",
  "tagIds": [1]
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "authorId": 1,
    "categoryId": 1,
    "title": "Notes of Ashen 使用指南 v2",
    "slug": "notes-of-ashen-guide-v2",
    "summary": "更新后的摘要",
    "content": "# 更新后的 Markdown 内容",
    "coverUrl": "https://example.com/cover-v2.png",
    "status": "published",
    "viewCount": 1,
    "publishedAt": "2026-06-05T20:00:00+08:00",
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:10:00+08:00",
    "tags": [],
    "category": null
  }
}
```

常见错误：`40100` Token 无效，`40300` 修改他人文章，`40400` 文章/分类/标签不存在，`40900` slug 已存在。

### 删除文章

```text
DELETE /api/v1/articles/:id
```

权限：文章作者或管理员。

响应示例：

```json
{
  "code": 0,
  "message": "success"
}
```

常见错误：`40100` Token 无效，`40300` 删除他人文章，`40400` 文章不存在。

### 更新文章状态

```text
PATCH /api/v1/articles/:id/status
```

权限：文章作者或管理员。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| status | string | 是 | `draft`、`published`、`archived` |

请求示例：

```json
{
  "status": "published"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "authorId": 1,
    "title": "Notes of Ashen 使用指南",
    "slug": "notes-of-ashen-guide",
    "summary": "一篇示例文章",
    "content": "# Markdown 内容",
    "coverUrl": "https://example.com/cover.png",
    "status": "published",
    "viewCount": 0,
    "publishedAt": "2026-06-05T20:00:00+08:00",
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:00:00+08:00"
  }
}
```

常见错误：`40000` status 非法，`40300` 修改他人文章，`40400` 文章不存在。

## 分类接口

### 分类列表

```text
GET /api/v1/categories
```

权限：公开。

查询参数：`page`、`size`。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "技术",
        "slug": "tech",
        "description": "技术文章",
        "createdBy": 1,
        "createdAt": "2026-06-05T20:00:00+08:00",
        "updatedAt": "2026-06-05T20:00:00+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "size": 10
  }
}
```

常见错误：`50000` 查询失败。

### 创建分类

```text
POST /api/v1/categories
```

权限：管理员。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 名称，长度 1 到 64 |
| slug | string | 是 | 唯一路径，长度 1 到 96 |
| description | string | 否 | 描述 |

请求示例：

```json
{
  "name": "技术",
  "slug": "tech",
  "description": "技术文章"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "技术",
    "slug": "tech",
    "description": "技术文章",
    "createdBy": 1,
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:00:00+08:00"
  }
}
```

常见错误：`40000` 参数校验失败，`40100` Token 无效，`40300` 非管理员，`40900` 分类名称或 slug 已存在。

### 更新分类

```text
PUT /api/v1/categories/:id
```

权限：管理员。

请求示例：

```json
{
  "name": "后端技术",
  "slug": "backend",
  "description": "后端相关文章"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "后端技术",
    "slug": "backend",
    "description": "后端相关文章",
    "createdBy": 1,
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:10:00+08:00"
  }
}
```

常见错误：`40300` 非管理员，`40400` 分类不存在，`40900` 名称或 slug 已存在。

### 删除分类

```text
DELETE /api/v1/categories/:id
```

权限：管理员。

响应示例：

```json
{
  "code": 0,
  "message": "success"
}
```

常见错误：`40300` 非管理员，`40400` 分类不存在，`50000` 分类仍被文章引用导致删除失败。

## 标签接口

### 标签列表

```text
GET /api/v1/tags
```

权限：公开。

查询参数：`page`、`size`。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "Go",
        "slug": "go",
        "description": "Go 语言",
        "createdBy": 1,
        "createdAt": "2026-06-05T20:00:00+08:00",
        "updatedAt": "2026-06-05T20:00:00+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "size": 10
  }
}
```

常见错误：`50000` 查询失败。

### 创建标签

```text
POST /api/v1/tags
```

权限：管理员。

请求示例：

```json
{
  "name": "Go",
  "slug": "go",
  "description": "Go 语言"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "Go",
    "slug": "go",
    "description": "Go 语言",
    "createdBy": 1,
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:00:00+08:00"
  }
}
```

常见错误：`40000` 参数校验失败，`40300` 非管理员，`40900` 标签名称或 slug 已存在。

### 更新标签

```text
PUT /api/v1/tags/:id
```

权限：管理员。

请求示例：

```json
{
  "name": "Golang",
  "slug": "golang",
  "description": "Go/Golang"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "Golang",
    "slug": "golang",
    "description": "Go/Golang",
    "createdBy": 1,
    "createdAt": "2026-06-05T20:00:00+08:00",
    "updatedAt": "2026-06-05T20:10:00+08:00"
  }
}
```

常见错误：`40300` 非管理员，`40400` 标签不存在，`40900` 名称或 slug 已存在。

### 删除标签

```text
DELETE /api/v1/tags/:id
```

权限：管理员。

响应示例：

```json
{
  "code": 0,
  "message": "success"
}
```

常见错误：`40300` 非管理员，`40400` 标签不存在。

## 管理接口

### 用户列表

```text
GET /api/v1/admin/users
```

权限：管理员。

查询参数：`page`、`size`。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "account": "admin",
        "email": "admin@example.com",
        "avatarUrl": "https://example.com/avatar.png",
        "nickname": "站长",
        "role": "admin",
        "status": "active",
        "createdAt": "2026-06-05T20:00:00+08:00",
        "updatedAt": "2026-06-05T20:00:00+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "size": 10
  }
}
```

常见错误：`40100` Token 无效，`40300` 非管理员。

### 修改用户状态

```text
PATCH /api/v1/admin/users/:id/status
```

权限：管理员。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| status | string | 是 | `active` 或 `disabled` |

请求示例：

```json
{
  "status": "disabled"
}
```

响应示例：

```json
{
  "code": 0,
  "message": "success"
}
```

常见错误：`40000` status 非法，`40300` 非管理员，`40400` 用户不存在。

### 操作日志列表

```text
GET /api/v1/admin/logs
```

权限：管理员。

查询参数：`page`、`size`。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "userId": 1,
        "eventType": "user.logged_in",
        "resourceType": "user",
        "resourceId": 1,
        "metadata": "{}",
        "ip": "127.0.0.1",
        "userAgent": "curl/8.0.0",
        "createdAt": "2026-06-05T20:00:00+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "size": 10
  }
}
```

常见错误：`40100` Token 无效，`40300` 非管理员，`50000` 查询失败。

## 推荐调用流程

1. 注册第一个用户，获得管理员身份。
2. 使用注册返回的 `accessToken` 调用分类、标签创建接口。
3. 创建文章并设置 `status = published`。
4. 匿名调用文章列表和文章详情接口验证公开访问。
5. 使用 `refreshToken` 调用刷新接口，拿到新的 Token。

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

## 文章搜索与集合筛选补充

### 公开文章列表筛选

```text
GET /api/v1/articles
```

新增查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| q | string | 否 | 关键词搜索，匹配标题、摘要、正文 |
| categoryId | uint64 | 否 | 按分类 ID 筛选 |
| tagId | uint64 | 否 | 按标签 ID 筛选 |

该接口仍为公开接口，匿名访问只返回 `published` 文章。参数可以与 `page`、`size`、`status` 组合使用。

示例：

```text
GET /api/v1/articles?q=go&categoryId=1&tagId=2&page=1&size=10
```

### 后台文章列表

```text
GET /api/v1/admin/articles
```

权限：管理员。

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| page | int | 否 | 页码 |
| size | int | 否 | 每页数量 |
| status | string | 否 | `draft`、`published`、`archived` |
| q | string | 否 | 关键词搜索，匹配标题、摘要、正文 |
| categoryId | uint64 | 否 | 按分类 ID 筛选 |
| tagId | uint64 | 否 | 按标签 ID 筛选 |

该接口用于后台文章管理，可以查看管理员权限下的全量文章集合；非管理员访问返回 `40300`。
