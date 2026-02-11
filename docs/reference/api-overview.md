# API 设计文档

## 概述

RESTful API 设计规范，基础路径：`/api`

## 1. 认证接口 (Casdoor OAuth2)

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/auth/login` | GET | 获取 SSO 登录 URL | 🟢 已实现 |
| `/auth/signup` | GET | 获取 SSO 注册 URL | 🟢 已实现 |
| `/auth/callback` | GET | OAuth 回调处理 | 🟢 已实现 |
| `/auth/refresh` | POST | 刷新 Token | 🟢 已实现 |
| `/auth/me` | GET | 获取当前用户信息 | 🟢 已实现 |
| `/auth/logout` | POST | 登出当前设备 | 🟢 已实现 |
| `/auth/logout-all` | POST | 登出所有设备 | 🟢 已实现 |

## 2. 评课社区接口

基础路径：`/api/course-review`

| 接口 | 方法 | 说明 | 状态 |
|------|------|------|------|
| `/rating-dimensions` | GET | 获取评分维度配置 | 🟢 已实现 |
| `/departments` | GET | 获取院系列表 | 🟢 已实现 |
| `/courses` | GET | 获取课程列表 | 🟢 已实现 |
| `/courses/search` | GET | 搜索课程 | 🟢 已实现 |
| `/courses/:id` | GET | 获取课程详情 | 🟢 已实现 |
| `/courses/:id/rating-stats` | GET | 获取课程评分统计 | 🟢 已实现 |
| `/courses/:id/reviews` | GET | 获取课程测评列表 | 🟢 已实现 |
| `/reviews/latest` | GET | 获取最新测评 | 🟢 已实现 |
| `/reviews` | POST | 发布测评 | 🟢 已实现 |
| `/reviews/:id/vote` | POST | 点赞/踩 | 🟢 已实现 |
| `/stats` | GET | 获取统计数据 | 🟢 已实现 |

## 3. 通用响应格式

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
  "data": [],
  "meta": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

### 错误响应

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "error description",
    "details": {}
  }
}
```

## 4. 认证方式

- **Cookie 认证**: Access Token 通过 HttpOnly Cookie 传递
- **需要认证的接口**: 发布测评、投票等写操作

## 5. 状态标记说明

- 🟢 **已实现** - 功能已上线
- 🟡 **开发中** - 正在开发
- 🔴 **规划中** - 尚未开始
