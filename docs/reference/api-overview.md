# API 设计文档

## 概述

RESTful API 设计规范，基础路径：`/api/v1`

> **OpenAPI 规范**: 本项目采用 Spec-First 模式，API 的权威定义在 `server/api/openapi.yaml`。
> 开发环境可访问 http://localhost:8080/docs/ 查看交互式 Swagger UI 文档。
> 以下表格为快速参考，详细的请求/响应 Schema 请查阅 OpenAPI 规范。

## 1. 认证接口 (`/api/v1/auth`)

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/login` | GET | 获取 SSO 登录 URL | 🟢 已实现 |
| `/signup` | GET | 获取 SSO 注册 URL | 🟢 已实现 |
| `/callback` | GET | OAuth 回调处理 | 🟢 已实现 |
| `/refresh` | POST | 刷新 Token | 🟢 已实现 |
| `/me` | GET | 获取当前用户信息 | 🟢 已实现 |
| `/logout` | POST | 登出当前设备 | 🟢 已实现 |
| `/logout-all` | POST | 登出所有设备 | 🟢 已实现 |

## 2. 课程接口 (`/api/v1/course`)

| 接口 | 方法 | 认证 | 说明 | 状态 |
|------|------|------|------|------|
| `/departments` | GET | 否 | 获取院系列表 | 🟢 已实现 |
| `/courses` | GET | 否 | 获取课程列表（分页） | 🟢 已实现 |
| `/courses/search` | GET | 否 | 搜索课程 | 🟢 已实现 |
| `/courses/:id` | GET | 否 | 获取课程详情 | 🟢 已实现 |
| `/categories` | GET | 否 | 获取课程分类列表 | 🟢 已实现 |

## 3. 评课社区接口 (`/api/v1/course/review`)

### 公开接口（无需认证）

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/rating-dimensions` | GET | 获取评分维度配置 | 🟢 已实现 |
| `/courses/:id/rating-stats` | GET | 获取课程评分统计 | 🟢 已实现 |
| `/courses/:id/rating-trend` | GET | 获取课程评分趋势 | 🟢 已实现 |
| `/courses/:id/reviews` | GET | 获取课程测评列表 | 🟢 已实现 |
| `/reviews/latest` | GET | 获取最新测评（支持 sort 参数） | 🟢 已实现 |
| `/reviews/:id/replies` | GET | 获取测评回复列表 | 🟢 已实现 |
| `/stats` | GET | 获取门户统计数据 | 🟢 已实现 |
| `/rankings/hot` | GET | 热门课程排行 | 🟢 已实现 |
| `/teachers/:id/stats` | GET | 教师评分统计 | 🟢 已实现 |

### 用户接口（需要认证）

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/reviews` | POST | 发布测评 | 🟢 已实现 |
| `/reviews/:id` | PUT | 编辑测评 | 🟢 已实现 |
| `/reviews/:id` | DELETE | 删除测评 | 🟢 已实现 |
| `/reviews/:id/vote` | POST | 点赞/踩 | 🟢 已实现 |
| `/reviews/:id/report` | POST | 举报测评 | 🟢 已实现 |
| `/reviews/:id/replies` | POST | 发布回复 | 🟢 已实现 |
| `/replies/:id` | DELETE | 删除回复 | 🟢 已实现 |
| `/content/check` | POST | 内容检查（敏感词+质量） | 🟢 已实现 |
| `/courses/:id/favorite` | POST | 收藏课程 | 🟢 已实现 |
| `/courses/:id/favorite` | DELETE | 取消收藏 | 🟢 已实现 |
| `/drafts` | POST | 保存草稿 | 🟢 已实现 |
| `/drafts/:courseID` | GET | 获取草稿 | 🟢 已实现 |
| `/drafts/:courseID` | DELETE | 删除草稿 | 🟢 已实现 |

### 用户中心接口（需要认证）

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/user/reviews` | GET | 获取我的测评 | 🟢 已实现 |
| `/user/votes` | GET | 获取我的投票 | 🟢 已实现 |
| `/user/favorites` | GET | 获取我的收藏 | 🟢 已实现 |

### 通知接口（需要认证）

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/notifications` | GET | 获取通知列表 | 🟢 已实现 |
| `/notifications/unread-count` | GET | 获取未读通知数 | 🟢 已实现 |
| `/notifications/:id/read` | PUT | 标记通知已读 | 🟢 已实现 |
| `/notifications/read-all` | PUT | 标记全部已读 | 🟢 已实现 |

### 管理员接口（需要认证 + 管理员权限）

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/admin/reports` | GET | 获取举报列表 | 🟢 已实现 |
| `/admin/reports/:id` | PUT | 处理举报 | 🟢 已实现 |
| `/admin/reviews` | GET | 获取所有测评（含隐藏/删除） | 🟢 已实现 |
| `/admin/reviews/:id` | PUT | 管理员更新测评状态 | 🟢 已实现 |
| `/admin/reviews/batch` | POST | 批量更新测评 | 🟢 已实现 |
| `/admin/stats` | GET | 管理后台统计 | 🟢 已实现 |
| `/admin/logs` | GET | 操作日志 | 🟢 已实现 |
| `/admin/export` | GET | 导出测评（CSV） | 🟢 已实现 |

## 4. 通用响应格式

### 成功响应

```json
{
  "success": true,
  "data": {}
}
```

### 分页响应

```json
{
  "success": true,
  "data": {
    "list": [],
    "total": 100
  }
}
```

### 错误响应

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "error description"
  }
}
```

## 5. 认证方式

- Cookie 认证：Access Token 通过 HttpOnly Cookie 传递
- 需要认证的接口：所有写操作 + 用户中心 + 通知
- 管理员接口额外需要 Casdoor 管理员角色

## 6. 限流策略

| 操作类型 | 限制 |
|----------|------|
| 发布测评 | 5 次/分钟 |
| 投票 | 30 次/分钟 |
| 举报 | 10 次/分钟 |
| 回复 | 10 次/分钟 |
| 更新/删除 | 10 次/分钟 |
