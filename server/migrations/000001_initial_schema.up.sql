-- StuHelper baseline schema migration
-- Source snapshot: server/scripts/init.sql
-- Note: Production schema evolution must go through numbered migrations in this directory.

-- StuHelper 数据库初始化脚本（全量）
-- 使用方法: psql -U stuhelper -d stuhelper -f init.sql
-- 包含: 用户表、课程表、评课社区全部业务表
-- ID 策略: 用户可见实体 (courses/teachers/departments) 使用 BIGSERIAL，内部业务表使用 UUIDv7 (VARCHAR(36))

BEGIN;

-- ============================================
-- 1. 创建 users 表（用户）
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    external_id VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- ============================================
-- 2. 创建 departments 表（院系）
-- ============================================
CREATE TABLE IF NOT EXISTS departments (
    id BIGSERIAL PRIMARY KEY,
    school_id BIGINT NOT NULL DEFAULT 1,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),
    category VARCHAR(50) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_departments_school_id ON departments(school_id);
CREATE INDEX IF NOT EXISTS idx_departments_category ON departments(category);

-- ============================================
-- 3. 创建 teachers 表（教师）
-- ============================================
CREATE TABLE IF NOT EXISTS teachers (
    id BIGSERIAL PRIMARY KEY,
    school_id BIGINT NOT NULL DEFAULT 1,
    name VARCHAR(255) NOT NULL,
    department_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_teachers_department FOREIGN KEY (department_id) REFERENCES departments(id)
);

CREATE INDEX IF NOT EXISTS idx_teachers_school_id ON teachers(school_id);
CREATE INDEX IF NOT EXISTS idx_teachers_department_id ON teachers(department_id);
CREATE INDEX IF NOT EXISTS idx_teachers_name ON teachers(name);

-- ============================================
-- 3b. 创建 terms 表（学期）
-- ============================================
CREATE TABLE IF NOT EXISTS terms (
    id VARCHAR(20) PRIMARY KEY,
    school_id BIGINT NOT NULL DEFAULT 1,
    name VARCHAR(100) NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO terms (id, name, is_current) VALUES
    ('2025-1', '2025 春', false),
    ('2025-2', '2025 秋', true)
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 4. 创建 courses 表（课程）
-- ============================================
CREATE TABLE IF NOT EXISTS courses (
    id BIGSERIAL PRIMARY KEY,
    school_id BIGINT NOT NULL DEFAULT 1,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50),
    department_id BIGINT,
    credits DECIMAL(4,1),
    category VARCHAR(50) NOT NULL DEFAULT '',
    description TEXT,
    review_count INT NOT NULL DEFAULT 0 CHECK (review_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_courses_department FOREIGN KEY (department_id) REFERENCES departments(id)
);

CREATE INDEX IF NOT EXISTS idx_courses_name ON courses(name);
CREATE INDEX IF NOT EXISTS idx_courses_department_id ON courses(department_id);
CREATE INDEX IF NOT EXISTS idx_courses_category ON courses(category);

-- ============================================
-- 4b. 创建 course_categories 表（课程分类配置）
-- ============================================
CREATE TABLE IF NOT EXISTS course_categories (
    id BIGSERIAL PRIMARY KEY,
    school_id BIGINT NOT NULL DEFAULT 1,
    name VARCHAR(50) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name)
);

INSERT INTO course_categories (name, sort_order) VALUES
    ('通识', 0), ('体育', 1), ('英语', 2), ('思政', 3)
ON CONFLICT (school_id, name) DO NOTHING;

-- ============================================
-- 5. 创建 reviews 表（评论主表）
-- ============================================
CREATE TABLE IF NOT EXISTS reviews (
    id VARCHAR(36) PRIMARY KEY,
    course_id BIGINT NOT NULL,
    teacher_id BIGINT,
    term_id VARCHAR(20) NOT NULL DEFAULT '',
    user_hash VARCHAR(64) NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    grade VARCHAR(5),
    ratings JSONB NOT NULL DEFAULT '{}',
    avg_rating DECIMAL(3,2) NOT NULL DEFAULT 0 CHECK (avg_rating >= 0 AND avg_rating <= 5),
    like_count INT NOT NULL DEFAULT 0 CHECK (like_count >= 0),
    dislike_count INT NOT NULL DEFAULT 0 CHECK (dislike_count >= 0),
    reply_count INT NOT NULL DEFAULT 0 CHECK (reply_count >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    moderation_reason TEXT,
    original_content TEXT,
    original_title VARCHAR(200),
    moderated_by VARCHAR(255),
    moderated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_reviews_course FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_reviews_teacher FOREIGN KEY (teacher_id) REFERENCES teachers(id) ON DELETE SET NULL,
    CONSTRAINT chk_reviews_status CHECK (status IN ('published', 'hidden', 'deleted')),
    CONSTRAINT chk_reviews_title_length CHECK (char_length(btrim(title)) >= 1 AND char_length(title) <= 200)
);

CREATE INDEX IF NOT EXISTS idx_reviews_course_id ON reviews(course_id);
CREATE INDEX IF NOT EXISTS idx_reviews_teacher_id ON reviews(teacher_id);
CREATE INDEX IF NOT EXISTS idx_reviews_user_hash ON reviews(user_hash);
CREATE INDEX IF NOT EXISTS idx_reviews_status ON reviews(status);
CREATE INDEX IF NOT EXISTS idx_reviews_term_id ON reviews(term_id);
-- L-58: 此索引可能被 idx_reviews_course_status_created 和 idx_reviews_user_hash_created 覆盖，
-- 仅在全局按时间排序（不带 course_id/user_hash 过滤）时有用。
-- 建议上线后用 EXPLAIN ANALYZE 验证实际查询是否命中此索引，若未命中可考虑移除。
CREATE INDEX IF NOT EXISTS idx_reviews_created_at ON reviews(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reviews_avg_rating ON reviews(avg_rating DESC);
-- M-113: 索引列顺序 (course_id, status, created_at DESC) 经过优化：
-- course_id 等值过滤在前（高选择性），status 等值过滤居中，created_at DESC 用于排序避免 filesort。
-- 此顺序匹配主查询 WHERE course_id=? AND status='published' ORDER BY created_at DESC。
CREATE INDEX IF NOT EXISTS idx_reviews_course_status_created ON reviews(course_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reviews_course_teacher_status ON reviews(course_id, teacher_id, status);
CREATE INDEX IF NOT EXISTS idx_reviews_user_hash_created ON reviews(user_hash, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_user_course ON reviews(user_hash, course_id)
    WHERE status != 'deleted';

-- ============================================
-- 6. 创建 review_votes 表（投票记录）
-- ============================================
CREATE TABLE IF NOT EXISTS review_votes (
    id VARCHAR(36) PRIMARY KEY,
    review_id VARCHAR(36) NOT NULL,
    user_hash VARCHAR(64) NOT NULL,
    vote_type VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_review_votes_review FOREIGN KEY (review_id) REFERENCES reviews(id) ON DELETE CASCADE,
    CONSTRAINT uq_review_votes_user UNIQUE (review_id, user_hash),
    CONSTRAINT chk_review_votes_type CHECK (vote_type IN ('like', 'dislike'))
);

CREATE INDEX IF NOT EXISTS idx_review_votes_review_id ON review_votes(review_id);
CREATE INDEX IF NOT EXISTS idx_review_votes_review_user ON review_votes(review_id, user_hash);

-- ============================================
-- 7. 创建 rating_dimensions 表（评分维度配置）
-- ============================================
CREATE TABLE IF NOT EXISTS rating_dimensions (
    id VARCHAR(36) PRIMARY KEY,
    school_id BIGINT NOT NULL DEFAULT 1,
    key VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_rating_dimensions_key UNIQUE (school_id, key),
    CONSTRAINT uq_rating_dimensions_key_global UNIQUE (key)
);

CREATE INDEX IF NOT EXISTS idx_rating_dimensions_school ON rating_dimensions(school_id);
CREATE INDEX IF NOT EXISTS idx_rating_dimensions_active ON rating_dimensions(is_active);

-- ============================================
-- 8. 创建 course_rating_stats 表（课程评分统计）
-- ============================================
CREATE TABLE IF NOT EXISTS course_rating_stats (
    id VARCHAR(36) PRIMARY KEY,
    course_id BIGINT NOT NULL,
    term_id VARCHAR(20),
    dimension_key VARCHAR(50) NOT NULL,
    avg_rating DECIMAL(3,2) NOT NULL DEFAULT 0 CHECK (avg_rating >= 0 AND avg_rating <= 5),
    rating_count INT NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    rating_dist JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_course_rating_stats_course FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_course_rating_stats_dimension FOREIGN KEY (dimension_key) REFERENCES rating_dimensions(key) ON DELETE CASCADE,
    CONSTRAINT uq_course_rating_stats UNIQUE (course_id, term_id, dimension_key)
);

CREATE INDEX IF NOT EXISTS idx_course_rating_stats_course ON course_rating_stats(course_id);
CREATE INDEX IF NOT EXISTS idx_course_rating_stats_term ON course_rating_stats(course_id, term_id);

-- ============================================
-- 9. 创建 review_reports 表（举报记录）
-- ============================================
CREATE TABLE IF NOT EXISTS review_reports (
    id VARCHAR(36) PRIMARY KEY,
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
    CONSTRAINT uq_review_reports_user UNIQUE (review_id, reporter_hash),
    CONSTRAINT chk_review_reports_status CHECK (status IN ('pending', 'resolved', 'rejected'))
);

CREATE INDEX IF NOT EXISTS idx_review_reports_review_id ON review_reports(review_id);
CREATE INDEX IF NOT EXISTS idx_review_reports_status ON review_reports(status);
CREATE INDEX IF NOT EXISTS idx_review_reports_created_at ON review_reports(created_at DESC);
-- M-35: 支持按 review_id + reporter_hash 快速查询是否已举报（去重检查）
CREATE INDEX IF NOT EXISTS idx_review_reports_review_reporter ON review_reports(review_id, reporter_hash);
-- M-36: 支持管理后台按状态+时间排序查询举报列表
CREATE INDEX IF NOT EXISTS idx_review_reports_status_created ON review_reports(status, created_at DESC);

-- ============================================
-- 10. 创建 course_favorites 表（课程收藏）
-- ============================================
CREATE TABLE IF NOT EXISTS course_favorites (
    id VARCHAR(36) PRIMARY KEY,
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
-- 11. 创建 review_drafts 表（评论草稿）
-- ============================================
CREATE TABLE IF NOT EXISTS review_drafts (
    id VARCHAR(36) PRIMARY KEY,
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
-- 12. 创建 review_replies 表（评论回复）
-- ============================================
CREATE TABLE IF NOT EXISTS review_replies (
    id VARCHAR(36) PRIMARY KEY,
    review_id VARCHAR(36) NOT NULL,
    parent_id VARCHAR(36),
    user_hash VARCHAR(64) NOT NULL,
    content TEXT NOT NULL,
    like_count INT NOT NULL DEFAULT 0 CHECK (like_count >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_review_replies_review FOREIGN KEY (review_id)
        REFERENCES reviews(id) ON DELETE CASCADE,
    CONSTRAINT fk_review_replies_parent FOREIGN KEY (parent_id)
        REFERENCES review_replies(id) ON DELETE CASCADE,
    CONSTRAINT chk_review_replies_status CHECK (status IN ('published', 'hidden', 'deleted')),
    CONSTRAINT chk_review_replies_content_length CHECK (char_length(btrim(content)) >= 1 AND char_length(content) <= 5000)
);

CREATE INDEX IF NOT EXISTS idx_review_replies_review_id ON review_replies(review_id);
CREATE INDEX IF NOT EXISTS idx_review_replies_parent_id ON review_replies(parent_id);
CREATE INDEX IF NOT EXISTS idx_review_replies_user_hash ON review_replies(user_hash);
CREATE INDEX IF NOT EXISTS idx_review_replies_created_at ON review_replies(review_id, created_at DESC);

-- ============================================
-- 13. 创建 notifications 表（通知）
-- ============================================
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(36) PRIMARY KEY,
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
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_hash, created_at DESC);
-- M-38: 支持「未读通知列表」查询 WHERE user_hash=? AND is_read=false ORDER BY created_at DESC
CREATE INDEX IF NOT EXISTS idx_notifications_user_read_created ON notifications(user_hash, is_read, created_at DESC);

-- ============================================
-- 14. 创建 teacher_rating_stats 表（教师评分统计）
-- ============================================
CREATE TABLE IF NOT EXISTS teacher_rating_stats (
    id VARCHAR(36) PRIMARY KEY,
    teacher_id BIGINT NOT NULL,
    term_id VARCHAR(20),
    dimension_key VARCHAR(50) NOT NULL,
    avg_rating DECIMAL(3,2) NOT NULL DEFAULT 0 CHECK (avg_rating >= 0 AND avg_rating <= 5),
    rating_count INT NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    rating_dist JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_teacher_rating_stats_teacher FOREIGN KEY (teacher_id)
        REFERENCES teachers(id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_rating_stats_dimension FOREIGN KEY (dimension_key)
        REFERENCES rating_dimensions(key) ON DELETE CASCADE,
    CONSTRAINT uq_teacher_rating_stats UNIQUE (teacher_id, term_id, dimension_key)
);

CREATE INDEX IF NOT EXISTS idx_teacher_rating_stats_teacher ON teacher_rating_stats(teacher_id);
CREATE INDEX IF NOT EXISTS idx_teacher_rating_stats_term ON teacher_rating_stats(teacher_id, term_id);

-- ============================================
-- 15. 创建 sensitive_words 表（敏感词）
-- ============================================
CREATE TABLE IF NOT EXISTS sensitive_words (
    id VARCHAR(36) PRIMARY KEY,
    word VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    level VARCHAR(20) NOT NULL DEFAULT 'block',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sensitive_words_word UNIQUE (word),
    CONSTRAINT chk_sensitive_words_level CHECK (level IN ('block', 'warn', 'review'))
);

CREATE INDEX IF NOT EXISTS idx_sensitive_words_category ON sensitive_words(category);
CREATE INDEX IF NOT EXISTS idx_sensitive_words_active ON sensitive_words(is_active);

-- ============================================
-- 16. 创建 admin_operation_logs 表（管理操作日志）
-- ============================================
CREATE TABLE IF NOT EXISTS admin_operation_logs (
    id VARCHAR(36) PRIMARY KEY,
    admin_username VARCHAR(255) NOT NULL,
    admin_user_id VARCHAR(100) NOT NULL DEFAULT '',
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    ip_address VARCHAR(45),
    user_agent VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_logs_admin ON admin_operation_logs(admin_username);
CREATE INDEX IF NOT EXISTS idx_admin_logs_user_id ON admin_operation_logs(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_action ON admin_operation_logs(action);
CREATE INDEX IF NOT EXISTS idx_admin_logs_resource ON admin_operation_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created ON admin_operation_logs(created_at DESC);

-- ============================================
-- 17. 创建 school_configs 表（学校认证配置）
-- ============================================
CREATE TABLE IF NOT EXISTS school_configs (
    school_id VARCHAR(10) PRIMARY KEY,
    school_name VARCHAR(100) NOT NULL,
    verification_method VARCHAR(20) NOT NULL DEFAULT 'manual',
    ldap_config BYTEA,
    academic_db_table VARCHAR(100),
    consent_text TEXT,
    manual_form_fields JSONB,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_school_configs_method CHECK (verification_method IN ('ldap', 'manual'))
);

-- ============================================
-- 18. 创建 user_identities 表（实名认证）
-- ============================================
CREATE TABLE IF NOT EXISTS user_identities (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    doc_type VARCHAR(20) NOT NULL,
    doc_number_enc BYTEA NOT NULL,
    person_uid VARCHAR(64) NOT NULL,
    real_name VARCHAR(100) NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verify_method VARCHAR(20),
    reviewed_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    doc_photo_front TEXT,
    doc_photo_back TEXT,
    doc_photo_selfie TEXT,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_user_identities_doc_type CHECK (doc_type IN ('MAINLAND_ID', 'HK_MACAU', 'TW', 'PASSPORT')),
    CONSTRAINT chk_user_identities_verify_method CHECK (verify_method IS NULL OR verify_method IN ('academic_db_match', 'tencent_cloud', 'manual'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_identities_person_uid ON user_identities(person_uid);
ALTER TABLE user_identities ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;
UPDATE user_identities
SET reviewed_at = COALESCE(reviewed_at, verified_at, updated_at)
WHERE reviewed_at IS NULL AND (verified = true OR rejection_reason IS NOT NULL);

-- ============================================
-- 19. 创建 user_profiles 表（学生认证）
-- ============================================
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    school_id VARCHAR(10),
    student_ids JSONB,
    active_student_id VARCHAR(50),
    manual_form_data JSONB,
    verification_status VARCHAR(20) NOT NULL DEFAULT 'unverified',
    verification_method VARCHAR(20),
    rejection_reason TEXT,
    reviewed_at TIMESTAMPTZ,
    phone VARCHAR(20),
    phone_verified BOOLEAN NOT NULL DEFAULT FALSE,
    consent_given_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_user_profiles_school FOREIGN KEY (school_id) REFERENCES school_configs(school_id),
    CONSTRAINT chk_user_profiles_status CHECK (verification_status IN ('unverified', 'pending', 'verified', 'rejected')),
    CONSTRAINT chk_user_profiles_method CHECK (verification_method IS NULL OR verification_method IN ('ldap', 'manual'))
);

CREATE INDEX IF NOT EXISTS idx_user_profiles_school ON user_profiles(school_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_status ON user_profiles(verification_status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_profiles_school_student ON user_profiles(school_id, active_student_id) WHERE active_student_id IS NOT NULL;

ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS rejection_reason TEXT;
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;

-- ============================================
-- 20. 创建系统配置表（角色→能力映射已迁移到 Go 常量 capability.RoleCapabilities）
-- ============================================
CREATE TABLE IF NOT EXISTS system_configs (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description VARCHAR(500),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 24. 创建学籍数据本地表（从 bj1 同步）
-- ============================================
CREATE SCHEMA IF NOT EXISTS academic;

CREATE TABLE IF NOT EXISTS academic.buaa_students (
    xh VARCHAR(50) PRIMARY KEY,
    xm VARCHAR(100),
    sfzjlxdm VARCHAR(10),
    sfzjh VARCHAR(50),
    yxdm VARCHAR(20),
    zydm VARCHAR(20),
    bjdm VARCHAR(20),
    xznj VARCHAR(10),
    rxnj VARCHAR(10),
    pyccdm VARCHAR(10),
    xslbdm VARCHAR(10),
    sjh VARCHAR(20),
    dzxx VARCHAR(100),
    xjztdm VARCHAR(10),
    sfzx VARCHAR(10),
    sfzj VARCHAR(10),
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_buaa_students_sfzjh ON academic.buaa_students(sfzjh);
CREATE INDEX IF NOT EXISTS idx_buaa_students_yxdm ON academic.buaa_students(yxdm);
CREATE INDEX IF NOT EXISTS idx_buaa_students_rxnj ON academic.buaa_students(rxnj);

COMMENT ON COLUMN academic.buaa_students.xh IS '学号 (student number)';
COMMENT ON COLUMN academic.buaa_students.xm IS '姓名 (full name)';
COMMENT ON COLUMN academic.buaa_students.sfzjlxdm IS '身份证件类型代码 (ID document type code)';
COMMENT ON COLUMN academic.buaa_students.sfzjh IS '身份证件号 (ID document number)';
COMMENT ON COLUMN academic.buaa_students.yxdm IS '院系代码 (department code)';
COMMENT ON COLUMN academic.buaa_students.zydm IS '专业代码 (major code)';
COMMENT ON COLUMN academic.buaa_students.bjdm IS '班级代码 (class code)';
COMMENT ON COLUMN academic.buaa_students.xznj IS '学制年限/学制年级 (program duration / grade system code)';
COMMENT ON COLUMN academic.buaa_students.rxnj IS '入学年级 (enrollment grade)';
COMMENT ON COLUMN academic.buaa_students.pyccdm IS '培养层次代码 (education level code)';
COMMENT ON COLUMN academic.buaa_students.xslbdm IS '学生类别代码 (student category code)';
COMMENT ON COLUMN academic.buaa_students.sjh IS '手机号 (mobile phone number)';
COMMENT ON COLUMN academic.buaa_students.dzxx IS '电子邮箱 (email address)';
COMMENT ON COLUMN academic.buaa_students.xjztdm IS '学籍状态代码 (student status code)';
COMMENT ON COLUMN academic.buaa_students.sfzx IS '是否在校 (on-campus status flag)';
COMMENT ON COLUMN academic.buaa_students.sfzj IS '是否在籍 (registered status flag)';

-- ============================================
-- 25. 插入默认评分维度数据
-- ============================================
INSERT INTO rating_dimensions (id, school_id, key, name, description, sort_order) VALUES
    (gen_random_uuid()::text, 1, 'difficulty', '课程难度', '课程内容的难易程度', 1),
    (gen_random_uuid()::text, 1, 'workload', '作业量', '课程作业和任务的工作量', 2),
    (gen_random_uuid()::text, 1, 'usefulness', '实用性', '课程内容对未来学习或工作的帮助程度', 3),
    (gen_random_uuid()::text, 1, 'teaching', '教学质量', '教师的授课水平和教学效果', 4),
    (gen_random_uuid()::text, 1, 'grading', '给分情况', '课程的评分标准和给分宽松程度', 5)
ON CONFLICT (school_id, key) DO NOTHING;

-- ============================================
-- 26. 插入默认学校配置
-- ============================================
INSERT INTO school_configs (school_id, school_name, verification_method, academic_db_table, consent_text, enabled) VALUES
    ('10006', '北京航空航天大学', 'ldap', 'academic.buaa_students', '本功能将使用您提供的学号和密码通过学校统一身份认证系统验证您的学生身份。验证成功后，系统将读取您的姓名、院系、年级、手机号等学籍信息用于平台服务。您的密码不会被存储。', FALSE)
ON CONFLICT (school_id) DO NOTHING;

-- ============================================
-- 27. 插入默认系统配置
-- ============================================
INSERT INTO system_configs (key, value, description) VALUES
    ('review_access_school_ids', '["10006"]', '允许查看完整评课和发布评课的学校 ID 列表（JSON 数组；留空则回退到已启用学校）'),
    ('review_preview_title_chars', '24', '评课标题预览最大字符数'),
    ('review_preview_content_chars', '120', '评课正文预览最大字符数'),
    ('review_preview_content_percent', '100', '评课正文预览最大展示比例（1-100）')
ON CONFLICT (key) DO NOTHING;

COMMIT;
