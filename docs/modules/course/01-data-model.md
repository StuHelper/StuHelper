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
                      ▼      │      ▼              ▼
               ┌──────────┐  │  ┌──────────┐ ┌──────────────┐
               │ Category │  │  │ Favorite │ │ TeacherStats │
               │ (分类)   │  │  │ (收藏)   │ │ (教师统计)   │
               └──────────┘  │  └──────────┘ └──────────────┘
                             │ 1:N
                             ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│    Vote     │◄──────│   Review    │───────►│   Report    │
│  (点赞/踩)  │       │ (测评/JSON) │       │  (举报)     │
└─────────────┘       └──────┬──────┘       └─────────────┘
                             │
                      ┌──────┼──────┐
                      │      │      │
                      ▼      ▼      ▼
               ┌──────────┐ ┌────┐ ┌───────┐
               │  Reply   │ │Draft│ │Notif. │
               │ (回复)   │ │(草稿)│ │(通知) │
               └──────────┘ └────┘ └───────┘
```

## ID 策略

| 场景 | 类型 | 示例表 | 理由 |
| --- | --- | --- | --- |
| 用户可见的实体（URL 中出现） | BIGSERIAL (int64) | courses, teachers, departments | URL 短且直观 |
| 内部业务数据（用户不直接访问） | UUIDv7 (VARCHAR(36)) | reviews, replies, votes, reports, drafts, notifications | 不可预测、分布式友好、自带时序 |

## 核心表结构

### 1. rating_dimensions (评分维度配置表)

存储可配置的评分维度，支持动态增删改。

```sql
CREATE TABLE rating_dimensions (
    id          VARCHAR(36) PRIMARY KEY,
    school_id   BIGINT NOT NULL DEFAULT 1,
    key         VARCHAR(50) NOT NULL,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    sort_order  INT NOT NULL DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_rating_dimensions_key UNIQUE (school_id, key)
);
```

**默认维度**：

| key | name | description |
|-----|------|-------------|
| `difficulty` | 课程难度 | 课程内容的难易程度 |
| `workload` | 作业量 | 课程作业和任务的工作量 |
| `usefulness` | 实用性 | 课程内容对未来学习或工作的帮助程度 |
| `teaching` | 教学质量 | 教师的授课水平和教学效果 |
| `grading` | 给分情况 | 课程的评分标准和给分宽松程度 |

### 2. departments (院系表)

```sql
CREATE TABLE departments (
    id          BIGSERIAL PRIMARY KEY,
    school_id   BIGINT NOT NULL DEFAULT 1,
    name        VARCHAR(255) NOT NULL,
    short_name  VARCHAR(50),
    category    VARCHAR(50) NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 3. courses (课程表)

```sql
CREATE TABLE courses (
    id              BIGSERIAL PRIMARY KEY,
    school_id       BIGINT NOT NULL DEFAULT 1,
    name            VARCHAR(255) NOT NULL,
    code            VARCHAR(50),
    department_id   BIGINT REFERENCES departments(id),
    credits         DECIMAL(3,1),
    category        VARCHAR(50) NOT NULL DEFAULT '',
    description     TEXT,
    review_count    INT NOT NULL DEFAULT 0 CHECK (review_count >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 4. course_categories (课程分类配置表)

```sql
CREATE TABLE course_categories (
    id          BIGSERIAL PRIMARY KEY,
    school_id   BIGINT NOT NULL DEFAULT 1,
    name        VARCHAR(50) NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**默认分类**：通识、体育、英语、思政

### 5. reviews (测评表)

核心表，存储用户发布的课程测评。评分使用 JSONB 存储，支持动态维度。`avg_rating` 为预计算列，INSERT/UPDATE 时自动计算。

```sql
CREATE TABLE reviews (
    id              VARCHAR(36) PRIMARY KEY,
    course_id       BIGINT NOT NULL REFERENCES courses(id),
    teacher_id      BIGINT REFERENCES teachers(id) ON DELETE SET NULL,
    term_id         VARCHAR(20),
    user_hash       VARCHAR(64) NOT NULL,
    title           VARCHAR(200),
    content         TEXT NOT NULL,
    grade           VARCHAR(5),
    ratings         JSONB NOT NULL DEFAULT '{}',
    avg_rating      DECIMAL(3,2) NOT NULL DEFAULT 0 CHECK (avg_rating >= 0 AND avg_rating <= 5),
    like_count      INT NOT NULL DEFAULT 0 CHECK (like_count >= 0),
    dislike_count   INT NOT NULL DEFAULT 0 CHECK (dislike_count >= 0),
    reply_count     INT NOT NULL DEFAULT 0 CHECK (reply_count >= 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'published',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_reviews_status CHECK (status IN ('published', 'hidden', 'deleted')),
    CONSTRAINT chk_reviews_title_length CHECK (title IS NULL OR char_length(title) <= 200)
);
-- 唯一约束：同一用户对同一课程只能发布一条未删除的测评
CREATE UNIQUE INDEX idx_reviews_user_course ON reviews(user_hash, course_id)
    WHERE status != 'deleted';
```

**ratings 字段示例**：

```json
{
  "difficulty": 4,
  "workload": 3,
  "usefulness": 5,
  "teaching": 4,
  "grading": 4
}
```

**评分等级说明**（五级制 1-5，表情评分）：

| 值 | 含义 | 表情 |
|----|------|------|
| 5 | 超赞 | 😍 |
| 4 | 不错 | 😊 |
| 3 | 一般 | 😐 |
| 2 | 较差 | 😟 |
| 1 | 很差 | 😢 |

### 6. review_replies (评论回复表)

```sql
CREATE TABLE review_replies (
    id          VARCHAR(36) PRIMARY KEY,
    review_id   VARCHAR(36) NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    parent_id   VARCHAR(36) REFERENCES review_replies(id) ON DELETE CASCADE,
    user_hash   VARCHAR(64) NOT NULL,
    content     TEXT NOT NULL,
    like_count  INT NOT NULL DEFAULT 0 CHECK (like_count >= 0),
    status      VARCHAR(20) NOT NULL DEFAULT 'published',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_review_replies_status CHECK (status IN ('published', 'hidden', 'deleted')),
    CONSTRAINT chk_review_replies_content_length CHECK (char_length(content) <= 5000)
);
```

### 7. 其他表

| 表名 | 用途 | ID 类型 |
|------|------|---------|
| review_votes | 点赞/踩记录（review_id + user_hash 唯一） | UUIDv7 |
| review_reports | 举报记录（review_id + reporter_hash 唯一） | UUIDv7 |
| course_rating_stats | 课程评分统计（按学期+维度聚合） | UUIDv7 |
| teacher_rating_stats | 教师评分统计（按学期+维度聚合） | UUIDv7 |
| course_favorites | 课程收藏（user_hash + course_id 唯一） | UUIDv7 |
| review_drafts | 评论草稿（user_hash + course_id 唯一） | UUIDv7 |
| notifications | 通知（支持 reply/vote/system 类型） | UUIDv7 |
| sensitive_words | 敏感词（block/warn/review 三级） | UUIDv7 |
| admin_operation_logs | 管理操作日志 | UUIDv7 |

## TypeScript 类型定义

```typescript
// 评分值 (1-5 五级制)
type RatingValue = 1 | 2 | 3 | 4 | 5

// 动态评分 (key -> value)
type ReviewRatings = Record<string, RatingValue>

// 课程信息
interface Course {
  id: number
  name: string
  code?: string
  credits: number
  departmentID: number
  departmentName?: string
  category: string
  reviewCount: number
}

// 课程分类
interface CourseCategory {
  id: number
  schoolID: number
  name: string
  sortOrder: number
}

// 测评信息
interface Review {
  id: string
  courseID: number
  courseName?: string
  teacherName?: string
  termID?: string
  title: string
  content: string
  grade?: string
  ratings: ReviewRatings
  likeCount: number
  dislikeCount: number
  replyCount: number
  status?: 'published' | 'hidden'
  createdAt: string
}
```
