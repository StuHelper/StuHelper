# 数据模型设计

本文档定义评课社区模块的数据库表结构和实体关系。

## 实体关系图

```
┌───────────────────┐
│ RatingDimension   │
│ (评分维度配置)     │
└─────────┬─────────┘
          │ 配置
          ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│  Department │       │   Course    │       │   Teacher   │
│  (院系)     │◄──────│   (课程)    │───────►│   (教师)    │
└─────────────┘       └──────┬──────┘       └──────┬──────┘
                             │                     │
                      ┌──────┼──────┐              │
                      │      │      │              │
                      ▼      │      ▼              │
               ┌──────────┐  │  ┌──────────┐       │
               │  Alias   │  │  │ Course   │◄──────┘
               │ (别名)   │  │  │ Teachers │
               └──────────┘  │  └──────────┘
                             │ 1:N
                             ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│    Term     │◄──────│   Review    │───────►│    Vote     │
│  (学期)     │       │ (测评/JSON) │       │  (点赞/踩)  │
└─────────────┘       └──────┬──────┘       └─────────────┘
                             │
                      ┌──────┼──────┐
                      │             │
                      ▼             ▼
               ┌─────────────┐ ┌─────────────┐
               │ RatingStats │ │   Report    │
               │ (评分统计)   │ │  (举报)     │
               └─────────────┘ └─────────────┘
```

## 核心表结构

### 1. rating_dimensions (评分维度配置表)

存储可配置的评分维度，支持动态增删改。

```sql
CREATE TABLE rating_dimensions (
    id          SERIAL PRIMARY KEY,
    school_id   INTEGER NOT NULL DEFAULT 1,
    key         VARCHAR(50) NOT NULL,
    name        VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    sort_order  INTEGER DEFAULT 0,
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(school_id, key)
);
```

**默认维度**：

| key | name | description |
|-----|------|-------------|
| `overall` | 总体评价 | 对课程的整体评价 |
| `content` | 内容质量 | 课程内容的深度和实用性 |
| `workload` | 工作量 | 作业、项目等课业负担 |
| `grading` | 考核/给分 | 考核方式和给分情况 |
| `attendance` | 考勤 | 点名、签到等考勤要求 |

### 2. departments (院系表)

```sql
CREATE TABLE departments (
    id          SERIAL PRIMARY KEY,
    school_id   INTEGER NOT NULL DEFAULT 1,
    name        VARCHAR(100) NOT NULL,
    short_name  VARCHAR(20),
    category    VARCHAR(20) NOT NULL DEFAULT 'school',
    sort_order  INTEGER DEFAULT 0,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**分类说明**：

| category | 说明 |
|----------|------|
| `school` | 全校院系 |
| `elective` | 通选课 |
| `pe` | 体育课 |
| `english` | 英语课 |
| `pols` | 思政课 |

### 3. courses (课程表)

```sql
CREATE TABLE courses (
    id              SERIAL PRIMARY KEY,
    school_id       INTEGER NOT NULL DEFAULT 1,
    department_id   INTEGER REFERENCES departments(id),
    code            VARCHAR(20),
    name            VARCHAR(200) NOT NULL,
    name_pinyin     VARCHAR(500),
    credits         DECIMAL(3,1),
    review_count    INTEGER DEFAULT 0,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 4. reviews (测评表)

核心表，存储用户发布的课程测评。评分使用 JSON 存储，支持动态维度。

```sql
CREATE TABLE reviews (
    id              VARCHAR(36) PRIMARY KEY,
    course_id       INTEGER NOT NULL REFERENCES courses(id),
    teacher_id      INTEGER REFERENCES teachers(id),
    term_id         VARCHAR(20) REFERENCES terms(id),
    user_hash       VARCHAR(64) NOT NULL,
    title           VARCHAR(200),
    content         TEXT NOT NULL,
    grade           VARCHAR(20),
    ratings         JSONB NOT NULL,
    like_count      INTEGER DEFAULT 0,
    dislike_count   INTEGER DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'published',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**ratings 字段示例**：

```json
{
  "overall": 4,
  "content": 5,
  "workload": 3,
  "grading": 4,
  "attendance": 2
}
```

**评分等级说明**（五级制 1-5）：

| 值 | 含义 |
|----|------|
| 5 | 非常好 |
| 4 | 好 |
| 3 | 一般 |
| 2 | 不好 |
| 1 | 很差 |

### 5. 其他表

- **review_votes**: 点赞/踩记录
- **review_reports**: 举报记录
- **course_rating_stats**: 课程评分统计（雷达图数据）
- **course_favorites**: 课程收藏
- **review_drafts**: 评论草稿
- **review_replies**: 评论回复
- **notifications**: 通知
- **sensitive_words**: 敏感词
- **admin_operation_logs**: 管理操作日志

## TypeScript 类型定义

```typescript
// 评分值 (1-5 五级制)
type RatingValue = 1 | 2 | 3 | 4 | 5;

// 动态评分 (key -> value)
type ReviewRatings = Record<string, RatingValue>;

// 课程信息
interface Course {
  id: number;
  name: string;
  code?: string;
  credits?: number;
  departmentId: number;
  departmentName: string;
  reviewCount: number;
}

// 测评信息
interface Review {
  id: string;
  courseId: number;
  courseName?: string;
  teacherName?: string;
  termId?: string;
  title: string;
  content: string;
  grade?: string;
  ratings: ReviewRatings;
  likeCount: number;
  dislikeCount: number;
  createdAt: string;
}
```
