# 数据模型设计

本文档定义评课社区模块的数据库表结构和实体关系。

## 实体关系图

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│  Department │       │   Course    │       │   Teacher   │
│  (院系)     │◄──────│   (课程)    │───────►│   (教师)    │
└─────────────┘       └──────┬──────┘       └─────────────┘
                             │
                             │ 1:N
                             ▼
                      ┌─────────────┐
                      │   Review    │
                      │   (测评)    │
                      └──────┬──────┘
                             │
                             │ 1:N
                             ▼
                      ┌─────────────┐
                      │    Vote     │
                      │  (点赞/踩)  │
                      └─────────────┘
```

## 核心表结构

### 1. departments (院系表)

存储学校的院系信息。

```sql
CREATE TABLE departments (
    id          SERIAL PRIMARY KEY,
    school_id   INTEGER NOT NULL DEFAULT 1,     -- 学校ID，支持多校
    name        VARCHAR(100) NOT NULL,          -- 院系名称
    short_name  VARCHAR(20),                    -- 简称
    category    VARCHAR(20) NOT NULL DEFAULT 'school',  -- 分类
    sort_order  INTEGER DEFAULT 0,              -- 排序权重
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_departments_school ON departments(school_id);
CREATE INDEX idx_departments_category ON departments(category);
```

**分类说明**：

| category | 说明 | 示例 |
|----------|------|------|
| `school` | 全校院系 | 数学科学学院、物理学院 |
| `elective` | 通选课 | 通识教育课程 |
| `pe` | 体育课 | 体育教研部 |
| `english` | 英语课 | 外国语学院 |
| `pols` | 思政课 | 马克思主义学院 |

### 2. courses (课程表)

存储课程基本信息。

```sql
CREATE TABLE courses (
    id              SERIAL PRIMARY KEY,
    school_id       INTEGER NOT NULL DEFAULT 1,
    department_id   INTEGER REFERENCES departments(id),
    code            VARCHAR(20),                -- 课程编号
    name            VARCHAR(200) NOT NULL,      -- 课程名称
    name_pinyin     VARCHAR(500),               -- 拼音 (搜索用)
    credits         DECIMAL(3,1),               -- 学分
    review_count    INTEGER DEFAULT 0,          -- 测评数量
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_courses_department ON courses(department_id);
CREATE INDEX idx_courses_name_fts ON courses
    USING gin(to_tsvector('zhparser', name));
```

### 3. teachers (教师表)

```sql
CREATE TABLE teachers (
    id          SERIAL PRIMARY KEY,
    school_id   INTEGER NOT NULL DEFAULT 1,
    name        VARCHAR(50) NOT NULL,
    department_id INTEGER REFERENCES departments(id),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_teachers_name ON teachers(name);
```

### 4. terms (学期表)

```sql
CREATE TABLE terms (
    id          VARCHAR(20) PRIMARY KEY,  -- 如: "25-26-1"
    school_id   INTEGER NOT NULL DEFAULT 1,
    name        VARCHAR(50) NOT NULL,     -- 如: "2025-2026学年第一学期"
    start_date  DATE,
    end_date    DATE,
    is_current  BOOLEAN DEFAULT FALSE
);
```

### 5. reviews (测评表)

核心表，存储用户发布的课程测评。

```sql
CREATE TABLE reviews (
    id              VARCHAR(20) PRIMARY KEY,  -- 短ID
    course_id       INTEGER NOT NULL REFERENCES courses(id),
    teacher_id      INTEGER REFERENCES teachers(id),
    term_id         VARCHAR(20) REFERENCES terms(id),
    title           VARCHAR(200),
    content         TEXT NOT NULL,
    grade           VARCHAR(20),              -- 成绩
    rating_recommend SMALLINT CHECK (rating_recommend BETWEEN -2 AND 2),
    rating_content   SMALLINT CHECK (rating_content BETWEEN -2 AND 2),
    rating_workload  SMALLINT CHECK (rating_workload BETWEEN -2 AND 2),
    rating_exam      SMALLINT CHECK (rating_exam BETWEEN -2 AND 2),
    like_count      INTEGER DEFAULT 0,
    dislike_count   INTEGER DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'published',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reviews_course ON reviews(course_id);
CREATE INDEX idx_reviews_created ON reviews(created_at DESC);
```

**四维评分说明**：

| 值 | 含义 | 图标 |
|----|------|------|
| 2 | 非常好 | 😍 |
| 1 | 好 | 🙂 |
| 0 | 一般 | 😐 |
| -1 | 不好 | 🙁 |
| -2 | 很差 | 😭 |

### 6. review_votes (点赞表)

```sql
CREATE TABLE review_votes (
    id          SERIAL PRIMARY KEY,
    review_id   VARCHAR(20) REFERENCES reviews(id),
    user_hash   VARCHAR(64) NOT NULL,  -- 用户标识哈希
    vote_type   SMALLINT NOT NULL,     -- 1=赞, -1=踩
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(review_id, user_hash)
);
```

## TypeScript 类型定义

```typescript
// 评分等级
type RatingLevel = -2 | -1 | 0 | 1 | 2;

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
  courseName: string;
  teacherName: string;
  termId: string;
  title: string;
  content: string;
  grade?: string;
  ratings: {
    recommend: RatingLevel;
    content: RatingLevel;
    workload: RatingLevel;
    exam: RatingLevel;
  };
  likeCount: number;
  dislikeCount: number;
  createdAt: string;
}
```
