# 数据库设计文档

## 概述

系统采用 PostgreSQL 作为核心数据库，Redis 作为缓存层。

---

## 1. 用户相关表

### 1.1 users 表

```sql
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    nickname        VARCHAR(50),
    student_id_cipher BYTEA,          -- 加密存储
    id_card_cipher  BYTEA,            -- 加密存储
    phone_cipher    BYTEA,            -- 加密存储
    school_id       INT REFERENCES schools(id),
    identity_verified BOOLEAN DEFAULT FALSE,
    points          INT DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'active',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### 1.2 oauth_bindings 表

```sql
CREATE TABLE oauth_bindings (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id),
    provider    VARCHAR(20),  -- qq, wechat
    open_id     VARCHAR(100),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, open_id)
);
```

---

## 2. 课程相关表

### 2.1 courses 表

```sql
CREATE TABLE courses (
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(20),
    name          VARCHAR(100) NOT NULL,
    teacher       VARCHAR(50),
    dept_name     VARCHAR(50),
    credits       DECIMAL(3,1),
    search_vector TSVECTOR,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_courses_search ON courses USING GIN(search_vector);
```

### 2.2 reviews 表

```sql
CREATE TABLE reviews (
    id          BIGSERIAL PRIMARY KEY,
    course_id   BIGINT REFERENCES courses(id),
    user_id     BIGINT REFERENCES users(id),
    rating      SMALLINT CHECK (rating BETWEEN 1 AND 5),
    difficulty  SMALLINT,
    workload    SMALLINT,
    content     TEXT,
    is_anonymous BOOLEAN DEFAULT TRUE,
    status      VARCHAR(20) DEFAULT 'pending',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(course_id, user_id)
);
```

### 2.3 resources 表

```sql
CREATE TABLE resources (
    id          BIGSERIAL PRIMARY KEY,
    course_id   BIGINT REFERENCES courses(id),
    uploader_id BIGINT REFERENCES users(id),
    file_url    VARCHAR(500),
    file_type   VARCHAR(20),
    status      VARCHAR(20) DEFAULT 'pending',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 3. 日志表

```sql
CREATE UNLOGGED TABLE logs (
    id          BIGSERIAL PRIMARY KEY,
    log_type    VARCHAR(20),
    user_id     BIGINT,
    action      VARCHAR(50),
    details     JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```
