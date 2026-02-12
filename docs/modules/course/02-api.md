# API 接口设计

本文档定义评课社区模块的 RESTful API 接口规范。

## 基础信息

- **课程模块 Base URL**: `/api/v1/course`
- **评课子模块 Base URL**: `/api/v1/course/review`
- **认证方式**: JWT Token (HttpOnly Cookie 传递)
- **响应格式**: JSON

## 通用响应格式

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

## 接口列表

### 课程模块 (`/api/v1/course`)

| 接口 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/departments` | GET | 否 | 获取院系列表 |
| `/courses` | GET | 否 | 获取课程列表（分页，支持 department_id 筛选） |
| `/courses/search` | GET | 否 | 搜索课程（keyword + limit） |
| `/courses/:id` | GET | 否 | 获取课程详情 |
| `/categories` | GET | 否 | 获取课程分类列表 |

### 评课子模块 — 公开接口 (`/api/v1/course/review`)

| 接口 | 方法 | 说明 |
|------|------|------|
| `/rating-dimensions` | GET | 获取评分维度配置 |
| `/courses/:id/rating-stats` | GET | 获取课程评分统计（按学期+维度聚合） |
| `/courses/:id/rating-trend` | GET | 获取课程评分趋势 |
| `/courses/:id/reviews` | GET | 获取课程测评列表（分页，支持 sort: time/likes/rating） |
| `/reviews/latest` | GET | 获取最新测评（支持 sort 参数） |
| `/reviews/:id/replies` | GET | 获取测评回复列表 |
| `/stats` | GET | 获取门户统计数据（课程数/评论数/院系数） |
| `/rankings/hot` | GET | 热门课程排行（支持 period + limit） |
| `/teachers/:id/stats` | GET | 教师评分统计（含课程列表、评分趋势） |

### 评课子模块 — 用户接口（需要认证）

| 接口 | 方法 | 说明 |
|------|------|------|
| `/reviews` | POST | 发布测评 |
| `/reviews/:id` | PUT | 编辑测评（仅作者） |
| `/reviews/:id` | DELETE | 删除测评（仅作者） |
| `/reviews/:id/vote` | POST | 点赞/踩（支持切换和取消） |
| `/reviews/:id/report` | POST | 举报测评 |
| `/reviews/:id/replies` | POST | 发布回复 |
| `/replies/:id` | DELETE | 删除回复（仅作者） |
| `/content/check` | POST | 内容检查（敏感词检测 + 质量评估，合并响应） |
| `/courses/:id/favorite` | POST | 收藏课程 |
| `/courses/:id/favorite` | DELETE | 取消收藏 |
| `/drafts` | POST | 保存草稿 |
| `/drafts/:courseId` | GET | 获取草稿 |
| `/drafts/:courseId` | DELETE | 删除草稿 |

### 评课子模块 — 用户中心（需要认证）

| 接口 | 方法 | 说明 |
|------|------|------|
| `/user/reviews` | GET | 获取我的测评（分页） |
| `/user/votes` | GET | 获取我的投票（分页，支持 vote_type 筛选） |
| `/user/favorites` | GET | 获取我的收藏（分页） |

### 评课子模块 — 通知（需要认证）

| 接口 | 方法 | 说明 |
|------|------|------|
| `/notifications` | GET | 获取通知列表（分页） |
| `/notifications/unread-count` | GET | 获取未读通知数 |
| `/notifications/:id/read` | PUT | 标记通知已读 |
| `/notifications/read-all` | PUT | 标记全部已读 |

### 评课子模块 — 管理员（需要认证 + 管理员权限）

| 接口 | 方法 | 说明 |
|------|------|------|
| `/admin/reports` | GET | 获取举报列表（分页，支持 status 筛选） |
| `/admin/reports/:id` | PUT | 处理举报 |
| `/admin/reviews` | GET | 获取所有测评（含隐藏/删除，分页） |
| `/admin/reviews/:id` | PUT | 管理员更新测评状态 |
| `/admin/reviews/batch` | POST | 批量更新测评（最多 100 条） |
| `/admin/stats` | GET | 管理后台统计 |
| `/admin/logs` | GET | 操作日志（分页） |
| `/admin/export` | GET | 导出测评（CSV 流式下载） |

## 限流策略

所有写操作均有 Redis 限流保护：

| 操作类型 | 限制 |
|----------|------|
| 发布测评 | 5 次/分钟 |
| 投票 | 30 次/分钟 |
| 举报 | 10 次/分钟 |
| 回复 | 10 次/分钟 |
| 更新/删除 | 10 次/分钟 |

## 内容检查接口详情

`POST /content/check` 合并了敏感词检测和质量评估，返回综合结果：

```json
{
  "success": true,
  "data": {
    "sensitive": {
      "hasSensitive": false,
      "matchCount": 0
    },
    "quality": {
      "score": 85,
      "suggestions": ["quality_too_short"]
    }
  }
}
```
