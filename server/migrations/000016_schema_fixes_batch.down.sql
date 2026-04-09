-- Reverse all changes from 000016_schema_fixes_batch.up.sql

BEGIN;

-- ============================================
-- Reverse DB-M6: Drop materialized view
-- ============================================
DROP MATERIALIZED VIEW IF EXISTS mv_teacher_public_stats;

-- ============================================
-- Reverse DB-M5: Restore sfzjh_enc back to sfzjh VARCHAR(50)
-- ============================================
DROP INDEX IF EXISTS academic.idx_buaa_students_sfzjh_hash;
ALTER TABLE academic.buaa_students ALTER COLUMN sfzjh_enc TYPE VARCHAR(50) USING convert_from(sfzjh_enc, 'UTF8');
ALTER TABLE academic.buaa_students RENAME COLUMN sfzjh_enc TO sfzjh;
ALTER TABLE academic.buaa_students DROP COLUMN IF EXISTS sfzjh_hash;
CREATE INDEX IF NOT EXISTS idx_buaa_students_sfzjh ON academic.buaa_students(sfzjh);
COMMENT ON COLUMN academic.buaa_students.sfzjh IS '身份证件号 (ID document number)';

-- ============================================
-- Reverse DB-M3: Restore both unique constraints
-- ============================================
ALTER TABLE rating_dimensions RENAME CONSTRAINT uq_rating_dimensions_key TO uq_rating_dimensions_key_global;
ALTER TABLE rating_dimensions ADD CONSTRAINT uq_rating_dimensions_key UNIQUE (school_id, key);

-- ============================================
-- Reverse DB-M2: Drop idx_courses_code
-- ============================================
DROP INDEX IF EXISTS idx_courses_code;

-- ============================================
-- Reverse DB-M1: Restore school_configs and user_profiles to VARCHAR(10), drop schools table and FKs
-- ============================================

-- Drop all school FKs added in the up migration
ALTER TABLE course_categories DROP CONSTRAINT IF EXISTS fk_course_categories_school;
ALTER TABLE terms DROP CONSTRAINT IF EXISTS fk_terms_school;
ALTER TABLE courses DROP CONSTRAINT IF EXISTS fk_courses_school;
ALTER TABLE teachers DROP CONSTRAINT IF EXISTS fk_teachers_school;
ALTER TABLE departments DROP CONSTRAINT IF EXISTS fk_departments_school;
ALTER TABLE user_profiles DROP CONSTRAINT IF EXISTS fk_user_profiles_school;
ALTER TABLE school_configs DROP CONSTRAINT IF EXISTS fk_school_configs_school;

-- Restore user_profiles.school_id to VARCHAR(10)
DROP INDEX IF EXISTS idx_user_profiles_school;
DROP INDEX IF EXISTS idx_user_profiles_school_student;
ALTER TABLE user_profiles RENAME COLUMN school_id TO school_id_new;
ALTER TABLE user_profiles ADD COLUMN school_id VARCHAR(10);
UPDATE user_profiles SET school_id = school_id_new::TEXT WHERE school_id_new IS NOT NULL;
ALTER TABLE user_profiles DROP COLUMN school_id_new;
CREATE INDEX IF NOT EXISTS idx_user_profiles_school ON user_profiles(school_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_profiles_school_student ON user_profiles(school_id, active_student_id) WHERE active_student_id IS NOT NULL;

-- Restore school_configs.school_id to VARCHAR(10)
ALTER TABLE school_configs DROP CONSTRAINT IF EXISTS school_configs_pkey;
ALTER TABLE school_configs RENAME COLUMN school_id TO school_id_new;
ALTER TABLE school_configs ADD COLUMN school_id VARCHAR(10);
UPDATE school_configs SET school_id = school_id_new::TEXT;
ALTER TABLE school_configs ALTER COLUMN school_id SET NOT NULL;
ALTER TABLE school_configs ADD PRIMARY KEY (school_id);
ALTER TABLE school_configs DROP COLUMN school_id_new;

-- Restore FK from user_profiles to school_configs
ALTER TABLE user_profiles
    ADD CONSTRAINT fk_user_profiles_school FOREIGN KEY (school_id)
    REFERENCES school_configs(school_id);

-- Drop schools table
DROP TABLE IF EXISTS schools;

-- ============================================
-- Reverse DB-H7: Restore user_profiles.phone and phone_verified
-- ============================================
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS phone_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- ============================================
-- Reverse DB-H5: Restore idx_review_reports_review_id
-- ============================================
CREATE INDEX IF NOT EXISTS idx_review_reports_review_id ON review_reports(review_id);

-- ============================================
-- Reverse DB-H4: Restore idx_review_reports_review_reporter
-- ============================================
CREATE INDEX IF NOT EXISTS idx_review_reports_review_reporter ON review_reports(review_id, reporter_hash);

-- ============================================
-- Reverse DB-H3: Restore idx_review_votes_review_id
-- ============================================
CREATE INDEX IF NOT EXISTS idx_review_votes_review_id ON review_votes(review_id);

-- ============================================
-- Reverse DB-H2: Restore idx_review_votes_review_user
-- ============================================
CREATE INDEX IF NOT EXISTS idx_review_votes_review_user ON review_votes(review_id, user_hash);

-- ============================================
-- Reverse DB-H1: Drop review_votes user_hash index
-- ============================================
DROP INDEX IF EXISTS idx_review_votes_user_hash;

-- ============================================
-- Reverse DB-C3: Drop term FKs, restore reviews.term_id to NOT NULL DEFAULT ''
-- ============================================
ALTER TABLE review_drafts DROP CONSTRAINT IF EXISTS fk_review_drafts_term;
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS fk_reviews_term;
UPDATE reviews SET term_id = '' WHERE term_id IS NULL;
ALTER TABLE reviews ALTER COLUMN term_id SET NOT NULL;
ALTER TABLE reviews ALTER COLUMN term_id SET DEFAULT '';

-- ============================================
-- Reverse DB-C2: Restore fk_reviews_course without explicit ON DELETE
-- ============================================
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS fk_reviews_course;
ALTER TABLE reviews
    ADD CONSTRAINT fk_reviews_course FOREIGN KEY (course_id) REFERENCES courses(id);

-- ============================================
-- Reverse DB-C1: Drop user_id FKs from notifications and notification_preferences
-- ============================================
ALTER TABLE notification_preferences DROP CONSTRAINT IF EXISTS fk_notification_preferences_user;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS fk_notifications_user;

COMMIT;
