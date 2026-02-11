-- 评课社区模块数据库初始化脚本
-- 使用方法: psql -U stuhelper -d stuhelper -f init_reviews.sql

BEGIN;

-- ============================================
-- 1. 确保 courses 表有 review_count 字段
-- ============================================
ALTER TABLE courses ADD COLUMN IF NOT EXISTS review_count INT NOT NULL DEFAULT 0;

-- ============================================
-- 2. 创建 reviews 表（评论主表）
-- ============================================
CREATE TABLE IF NOT EXISTS reviews (
    id VARCHAR(36) PRIMARY KEY,
    course_id BIGINT NOT NULL,
    teacher_id BIGINT,
    term_id VARCHAR(20),
    user_hash VARCHAR(64) NOT NULL,
    title VARCHAR(200),
    content TEXT NOT NULL,
    grade VARCHAR(5),
    ratings JSONB NOT NULL DEFAULT '{}',
    avg_rating DECIMAL(3,2) NOT NULL DEFAULT 0,
    like_count INT NOT NULL DEFAULT 0,
    dislike_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_reviews_course FOREIGN KEY (course_id) REFERENCES courses(id)
);

-- reviews 表索引
CREATE INDEX IF NOT EXISTS idx_reviews_course_id ON reviews(course_id);
CREATE INDEX IF NOT EXISTS idx_reviews_user_hash ON reviews(user_hash);
CREATE INDEX IF NOT EXISTS idx_reviews_status ON reviews(status);
CREATE INDEX IF NOT EXISTS idx_reviews_created_at ON reviews(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reviews_avg_rating ON reviews(avg_rating DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_user_course ON reviews(user_hash, course_id)
    WHERE status != 'deleted';

-- ============================================
-- 3. 创建 review_votes 表（投票记录）
-- ============================================
CREATE TABLE IF NOT EXISTS review_votes (
    id BIGSERIAL PRIMARY KEY,
    review_id VARCHAR(36) NOT NULL,
    user_hash VARCHAR(64) NOT NULL,
    vote_type VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_review_votes_review FOREIGN KEY (review_id) REFERENCES reviews(id) ON DELETE CASCADE,
    CONSTRAINT uq_review_votes_user UNIQUE (review_id, user_hash)
);

CREATE INDEX IF NOT EXISTS idx_review_votes_review_id ON review_votes(review_id);

-- ============================================
-- 4. 创建 rating_dimensions 表（评分维度配置）
-- ============================================
CREATE TABLE IF NOT EXISTS rating_dimensions (
    id BIGSERIAL PRIMARY KEY,
    school_id BIGINT NOT NULL DEFAULT 1,
    key VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_rating_dimensions_key UNIQUE (school_id, key)
);

CREATE INDEX IF NOT EXISTS idx_rating_dimensions_school ON rating_dimensions(school_id);
CREATE INDEX IF NOT EXISTS idx_rating_dimensions_active ON rating_dimensions(is_active);

-- ============================================
-- 5. 创建 course_rating_stats 表（课程评分统计）
-- ============================================
CREATE TABLE IF NOT EXISTS course_rating_stats (
    id BIGSERIAL PRIMARY KEY,
    course_id BIGINT NOT NULL,
    term_id VARCHAR(20),
    dimension_key VARCHAR(50) NOT NULL,
    avg_rating DECIMAL(3,2) NOT NULL DEFAULT 0,
    rating_count INT NOT NULL DEFAULT 0,
    rating_dist JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_course_rating_stats_course FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT uq_course_rating_stats UNIQUE (course_id, term_id, dimension_key)
);

CREATE INDEX IF NOT EXISTS idx_course_rating_stats_course ON course_rating_stats(course_id);

-- ============================================
-- 6. 创建 review_reports 表（举报记录）
-- ============================================
CREATE TABLE IF NOT EXISTS review_reports (
    id BIGSERIAL PRIMARY KEY,
    review_id VARCHAR(36) NOT NULL,
    reporter_hash VARCHAR(64) NOT NULL,
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    resolved_by VARCHAR(255),
    resolved_at TIMESTAMPTZ,
    resolution_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_review_reports_review FOREIGN KEY (review_id) REFERENCES reviews(id) ON DELETE CASCADE,
    CONSTRAINT uq_review_reports_user UNIQUE (review_id, reporter_hash)
);

CREATE INDEX IF NOT EXISTS idx_review_reports_review_id ON review_reports(review_id);
CREATE INDEX IF NOT EXISTS idx_review_reports_status ON review_reports(status);
CREATE INDEX IF NOT EXISTS idx_review_reports_created_at ON review_reports(created_at DESC);

-- ============================================
-- 7. 创建 course_favorites 表（课程收藏）
-- ============================================
CREATE TABLE IF NOT EXISTS course_favorites (
    id BIGSERIAL PRIMARY KEY,
    user_hash VARCHAR(64) NOT NULL,
    course_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_course_favorites_course FOREIGN KEY (course_id)
        REFERENCES courses(id) ON DELETE CASCADE,
    CONSTRAINT uq_course_favorites_user_course UNIQUE (user_hash, course_id)
);

CREATE INDEX IF NOT EXISTS idx_course_favorites_user_hash ON course_favorites(user_hash);
CREATE INDEX IF NOT EXISTS idx_course_favorites_course_id ON course_favorites(course_id);

-- ============================================
-- 8. 创建 review_drafts 表（评论草稿）
-- ============================================
CREATE TABLE IF NOT EXISTS review_drafts (
    id BIGSERIAL PRIMARY KEY,
    user_hash VARCHAR(64) NOT NULL,
    course_id BIGINT NOT NULL,
    teacher_id BIGINT,
    term_id VARCHAR(20),
    title VARCHAR(200),
    content TEXT,
    grade VARCHAR(5),
    ratings JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_review_drafts_course FOREIGN KEY (course_id)
        REFERENCES courses(id) ON DELETE CASCADE,
    CONSTRAINT uq_review_drafts_user_course UNIQUE (user_hash, course_id)
);

CREATE INDEX IF NOT EXISTS idx_review_drafts_user_hash ON review_drafts(user_hash);

-- ============================================
-- 10. 创建 review_replies 表（评论回复）
-- ============================================
CREATE TABLE IF NOT EXISTS review_replies (
    id BIGSERIAL PRIMARY KEY,
    review_id VARCHAR(36) NOT NULL,
    parent_id BIGINT,
    user_hash VARCHAR(64) NOT NULL,
    content TEXT NOT NULL,
    like_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_review_replies_review FOREIGN KEY (review_id)
        REFERENCES reviews(id) ON DELETE CASCADE,
    CONSTRAINT fk_review_replies_parent FOREIGN KEY (parent_id)
        REFERENCES review_replies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_review_replies_review_id ON review_replies(review_id);
CREATE INDEX IF NOT EXISTS idx_review_replies_parent_id ON review_replies(parent_id);
CREATE INDEX IF NOT EXISTS idx_review_replies_user_hash ON review_replies(user_hash);

-- 添加 reviews 表的 reply_count 字段
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS reply_count INT NOT NULL DEFAULT 0;

-- ============================================
-- 11. 创建 notifications 表（通知）
-- ============================================
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_hash VARCHAR(64) NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    related_type VARCHAR(50),
    related_id VARCHAR(64),
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_hash ON notifications(user_hash);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(user_hash, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

-- ============================================
-- 12. 创建 teacher_rating_stats 表（教师评分统计）
-- ============================================
CREATE TABLE IF NOT EXISTS teacher_rating_stats (
    id BIGSERIAL PRIMARY KEY,
    teacher_id BIGINT NOT NULL,
    term_id VARCHAR(20),
    dimension_key VARCHAR(50) NOT NULL,
    avg_rating DECIMAL(3,2) NOT NULL DEFAULT 0,
    rating_count INT NOT NULL DEFAULT 0,
    rating_dist JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_teacher_rating_stats_teacher FOREIGN KEY (teacher_id)
        REFERENCES teachers(id) ON DELETE CASCADE,
    CONSTRAINT uq_teacher_rating_stats UNIQUE (teacher_id, term_id, dimension_key)
);

CREATE INDEX IF NOT EXISTS idx_teacher_rating_stats_teacher ON teacher_rating_stats(teacher_id);
CREATE INDEX IF NOT EXISTS idx_teacher_rating_stats_term ON teacher_rating_stats(teacher_id, term_id);

-- ============================================
-- 13. 创建 sensitive_words 表（敏感词）
-- ============================================
CREATE TABLE IF NOT EXISTS sensitive_words (
    id BIGSERIAL PRIMARY KEY,
    word VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    level VARCHAR(20) NOT NULL DEFAULT 'block',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sensitive_words_word UNIQUE (word)
);

CREATE INDEX IF NOT EXISTS idx_sensitive_words_category ON sensitive_words(category);
CREATE INDEX IF NOT EXISTS idx_sensitive_words_active ON sensitive_words(is_active);

-- ============================================
-- 14. 创建 admin_operation_logs 表（管理操作日志）
-- ============================================
CREATE TABLE IF NOT EXISTS admin_operation_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_username VARCHAR(255) NOT NULL,
    admin_user_id VARCHAR(100) NOT NULL DEFAULT '',
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_logs_admin ON admin_operation_logs(admin_username);
CREATE INDEX IF NOT EXISTS idx_admin_logs_user_id ON admin_operation_logs(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_action ON admin_operation_logs(action);
CREATE INDEX IF NOT EXISTS idx_admin_logs_resource ON admin_operation_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created ON admin_operation_logs(created_at DESC);

-- ============================================
-- 15. 插入默认评分维度数据
-- ============================================
INSERT INTO rating_dimensions (school_id, key, name, description, sort_order) VALUES
    (1, 'difficulty', '课程难度', '课程内容的难易程度', 1),
    (1, 'workload', '作业量', '课程作业和任务的工作量', 2),
    (1, 'usefulness', '实用性', '课程内容对未来学习或工作的帮助程度', 3),
    (1, 'teaching', '教学质量', '教师的授课水平和教学效果', 4),
    (1, 'grading', '给分情况', '课程的评分标准和给分宽松程度', 5)
ON CONFLICT (school_id, key) DO NOTHING;

COMMIT;
