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
└─────────────┘       └──────┬──────┘       └─────────────┘
                             │
                             │ 1:N
                             ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│    Term     │◄──────│   Review    │───────►│    Vote     │
│  (学期)     │       │ (测评/JSON) │       │  (点赞/踩)  │
└─────────────┘       └──────┬──────┘       └─────────────┘
                             │
                             │ 统计聚合
                             ▼
                      ┌─────────────┐
                      │ RatingStats │
                      │ (评分统计)   │
                      └─────────────┘
```

## 核心表结构

### 1. rating_dimensions (评分维度配置表)

存储可配置的评分维度，支持动态增删改。

```sql
CREATE TABLE rating_dimensions (
    id          SERIAL PRIMARY KEY,
    school_id   INTEGER NOT NULL DEFAULT 1,
    key         VARCHAR(50) NOT NULL,           -- 维度标识符
    name        VARCHAR(100) NOT NULL,          -- 显示名称
    description VARCHAR(500),                   -- 维度说明
    sort_order  INTEGER DEFAULT 0,              -- 排序权重
    is_active   BOOLEAN DEFAULT TRUE,           -- 是否启用
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(school_id, key)
);

CREATE INDEX idx_rating_dimensions_school ON rating_dimensions(school_id);
CREATE INDEX idx_rating_dimensions_active ON rating_dimensions(is_active);
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
| `pro` | 专业课 | 各学院 |

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

### 6. reviews (测评表)

核心表，存储用户发布的课程测评。评分使用 JSON 存储，支持动态维度。

```sql
CREATE TABLE reviews (
    id              VARCHAR(20) PRIMARY KEY,  -- 短ID
    course_id       INTEGER NOT NULL REFERENCES courses(id),
    teacher_id      INTEGER REFERENCES teachers(id),
    term_id         VARCHAR(20) REFERENCES terms(id),
    user_id       VARCHAR(64) NOT NULL,     -- 用户标识哈希（匿名）
    title           VARCHAR(200),
    content         TEXT NOT NULL,
    grade           VARCHAR(20),              -- 成绩
    ratings         JSONB NOT NULL,           -- 动态评分 {"overall":5,"content":4,...}
    like_count      INTEGER DEFAULT 0,
    dislike_count   INTEGER DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'published',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reviews_course ON reviews(course_id);
CREATE INDEX idx_reviews_term ON reviews(term_id);
CREATE INDEX idx_reviews_created ON reviews(created_at DESC);
CREATE INDEX idx_reviews_ratings ON reviews USING gin(ratings);
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

| 值 | 含义 | 显示 |
|----|------|------|
| 5 | 非常好 | ★★★★★ |
| 4 | 好 | ★★★★☆ |
| 3 | 一般 | ★★★☆☆ |
| 2 | 不好 | ★★☆☆☆ |
| 1 | 很差 | ★☆☆☆☆ |

### 7. review_votes (点赞表)

```sql
CREATE TABLE review_votes (
    id          SERIAL PRIMARY KEY,
    review_id   VARCHAR(20) REFERENCES reviews(id),
    user_id   VARCHAR(64) NOT NULL,  -- 用户标识哈希
    vote_type   SMALLINT NOT NULL,     -- 1=赞, -1=踩
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(review_id, user_id)
);
```

### 8. course_rating_stats (课程评分统计表)

预计算的评分统计，支持按学期和维度查询，用于雷达图展示。

```sql
CREATE TABLE course_rating_stats (
    id              SERIAL PRIMARY KEY,
    course_id       INTEGER NOT NULL REFERENCES courses(id),
    term_id         VARCHAR(20) REFERENCES terms(id),  -- NULL 表示总体统计
    dimension_key   VARCHAR(50) NOT NULL,              -- 维度标识
    avg_rating      DECIMAL(3,2),                      -- 平均分
    rating_count    INTEGER DEFAULT 0,                 -- 评分数量
    rating_dist     JSONB DEFAULT '{}',                -- 分布 {"1":5,"2":10,...}
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(course_id, term_id, dimension_key)
);

CREATE INDEX idx_course_rating_stats_course ON course_rating_stats(course_id);
CREATE INDEX idx_course_rating_stats_term ON course_rating_stats(term_id);
```

> **雷达图数据说明**：雷达图展示该课程所有历史出现过的维度。即使维度配置发生变化，历史评分数据仍然保留并展示。

### 9. course_aliases (课程别名表)

课程的别名，方便在搜索时查找

```sql
CREATE TABLE course_aliases (
    id          SERIAL PRIMARY KEY,
    course_id   INTEGER NOT NULL REFERENCES courses(id),

    alias       VARCHAR(200) NOT NULL,        -- 别名本体
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
);

CREATE INDEX idx_course_aliases_alias ON course_aliases(alias);
CREATE INDEX idx_course_aliases_course_id ON course_aliases(course_id);     
```

`idx_course_aliases_course_id` 用于在课程详情页向新人介绍这门课的简称

### 10. course_teachers (课程-教师关联)

```sql
CREATE TABLE course_teachers (
    course_id   INTEGER NOT NULL REFERENCES courses(id)
    teacher_id  INTEGER NOT NULL REFERENCES teachers(id)
    sort_order  INTEGER DEFAULT 0,        -- 展示顺序
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (course_id, teacher_id)
);

CREATE INDEX idx_course_teachers_course ON course_teachers(course_id);
CREATE INDEX idx_course_teachers_teacher ON course_teachers(teacher_id);
```


## TypeScript 类型定义

```typescript
// 评分值 (1-5 五级制)
type RatingValue = 1 | 2 | 3 | 4 | 5;

// 评分维度配置
interface RatingDimension {
  id: number;
  key: string;
  name: string;
  description?: string;
  sortOrder: number;
  isActive: boolean;
}

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
  termName?: string;
  title: string;
  content: string;
  grade?: string;
  ratings: ReviewRatings;  // 动态评分 JSON
  likeCount: number;
  dislikeCount: number;
  createdAt: string;
}

// 维度评分统计
interface DimensionStats {
  key: string;
  name: string;
  avgRating: number;
  ratingCount: number;
  distribution: Record<RatingValue, number>;
}

// 课程评分统计（用于雷达图）
interface CourseRatingStats {
  courseId: number;
  overall: {
    dimensions: DimensionStats[];
  };
  byTerm: {
    termId: string;
    termName: string;
    dimensions: DimensionStats[];
  }[];
  // 所有历史出现过的维度（合并）
  allDimensionKeys: string[];
}
```
