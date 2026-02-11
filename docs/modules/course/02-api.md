# API 接口设计

本文档定义评课社区模块的 RESTful API 接口规范。

## 基础信息

- **课程模块 Base URL**: `/api/v1/course`
- **评课子模块 Base URL**: `/api/v1/course/review`
- **认证方式**: JWT Token (Cookie 传递)
- **响应格式**: JSON

## 通用响应格式

### 成功响应

```json
{
  "success": true,
  "data": {}
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

| 接口 | 方法 | 说明 |
|------|------|------|
| `/departments` | GET | 获取院系列表 |
| `/courses` | GET | 获取课程列表 |
| `/courses/search` | GET | 搜索课程 |
| `/courses/:id` | GET | 获取课程详情 |

### 评课子模块 (`/api/v1/course/review`)

| 接口 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/rating-dimensions` | GET | 否 | 获取评分维度配置 |
| `/courses/:id/rating-stats` | GET | 否 | 获取评分统计 |
| `/courses/:id/reviews` | GET | 否 | 获取课程测评列表 |
| `/reviews/latest` | GET | 否 | 获取最新测评 |
| `/reviews` | POST | 是 | 发布测评 |
| `/reviews/:id` | PUT | 是 | 编辑测评 |
| `/reviews/:id` | DELETE | 是 | 删除测评 |
| `/reviews/:id/vote` | POST | 是 | 点赞/踩 |
| `/reviews/:id/report` | POST | 是 | 举报测评 |
| `/rankings/hot` | GET | 否 | 热门课程排行 |
| `/teachers/:id/stats` | GET | 否 | 教师评分统计 |
