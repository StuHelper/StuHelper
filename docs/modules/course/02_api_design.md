# API 接口设计

本文档定义评课社区模块的 RESTful API 接口规范。

## 基础信息

- **Base URL**: `/api/v1/course-review`
- **认证方式**: JWT Token (可选，部分接口需要)
- **响应格式**: JSON

## 通用响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

## 接口列表

| 模块 | 接口 | 方法 | 说明 |
|------|------|------|------|
| 配置 | `/rating-dimensions` | GET | 获取评分维度配置 |
| 院系 | `/departments` | GET | 获取院系列表 |
| 课程 | `/courses` | GET | 获取课程列表 |
| 课程 | `/courses/:id` | GET | 获取课程详情 |
| 课程 | `/courses/search` | GET | 搜索课程 |
| 课程 | `/courses/:id/rating-stats` | GET | 获取评分统计（雷达图） |
| 测评 | `/courses/:id/reviews` | GET | 获取课程测评列表 |
| 测评 | `/reviews/latest` | GET | 获取最新测评 |
| 测评 | `/reviews` | POST | 发布测评 |
| 测评 | `/reviews/:id/vote` | POST | 点赞/踩 |
| 统计 | `/stats` | GET | 获取统计数据 |

## 课程接口

### GET /courses

获取课程列表（按院系分组）。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 否 | 分类: school/elective/pe/english/pols |
| department_id | number | 否 | 院系ID |

**响应示例**：

```json
{
  "code": 0,
  "data": {
    "departments": [
      {
        "id": 1,
        "name": "数学科学学院",
        "courses": [
          {
            "id": 1033,
            "name": "偏微分方程",
            "credits": 3,
            "reviewCount": 30
          }
        ]
      }
    ]
  }
}
```

### GET /courses/:id

获取课程详情及评分统计。

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| id | number | 课程ID |

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| teacher | string | 否 | 教师ID筛选 |

### GET /courses/search

搜索课程（支持模糊匹配）。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 是 | 搜索关键词 |
| limit | number | 否 | 返回数量，默认10 |

## 测评接口

### GET /reviews/latest

获取最新测评列表。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | number | 否 | 页码，默认1 |
| page_size | number | 否 | 每页数量，默认20 |

### POST /reviews

发布测评。

**请求体**：

```json
{
  "course_id": 1033,
  "teacher_id": 123,
  "term_id": "25-26-1",
  "title": "简单好学",
  "content": "课程听感:...",
  "grade": "99",
  "ratings": {
    "overall": 5,
    "content": 4,
    "workload": 3,
    "grading": 4,
    "attendance": 2
  }
}
```

**字段说明**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| course_id | number | 是 | 课程ID |
| teacher_id | number | 否 | 教师ID |
| term_id | string | 否 | 学期ID |
| title | string | 否 | 测评标题 |
| content | string | 是 | 测评内容（至少10字） |
| grade | string | 否 | 成绩 |
| ratings | object | 是 | 评分（动态维度，1-5分） |

> **注意**：`ratings` 字段的 key 应与 `/rating-dimensions` 返回的维度 key 对应。

### POST /reviews/:id/vote

点赞或踩测评。

**请求体**：

```json
{
  "vote_type": 1
}
```

| vote_type | 说明 |
|-----------|------|
| 1 | 点赞 |
| -1 | 踩 |
