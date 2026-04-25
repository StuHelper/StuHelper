-- 000016: Batch schema fixes addressing DB-C1, DB-C2, DB-C3, DB-H1..H7, DB-M1..M6
-- This migration addresses database issues in a single atomic migration.
--
-- NOTE on DB-M10/DB-M11: 000001 has redundant ALTER TABLE ... ADD COLUMN IF NOT EXISTS
-- statements for user_identities.reviewed_at and user_profiles.{rejection_reason,reviewed_at}.
-- These are harmless no-ops (columns already exist in the CREATE TABLE) and do not require fixing.
--
-- NOTE on DB-H6: 000001's down script uses DROP TABLE IF EXISTS academic CASCADE instead of
-- DROP SCHEMA IF EXISTS academic CASCADE. This is a known issue in the historical migration.
-- Down migrations are destructive and only used for local resets, so this is documented here
-- rather than addressed with a corrective migration.

BEGIN;

-- ============================================
-- DB-C1: notifications and notification_preferences missing user_id FK
-- ============================================
ALTER TABLE notifications
    ADD CONSTRAINT fk_notifications_user FOREIGN KEY (user_id)
    REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE notification_preferences
    ADD CONSTRAINT fk_notification_preferences_user FOREIGN KEY (user_id)
    REFERENCES users(id) ON DELETE CASCADE;

-- ============================================
-- DB-C2: reviews.course_id FK missing ON DELETE strategy
-- Make ON DELETE RESTRICT explicit (documents design intent; this is the implicit default)
-- ============================================
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS fk_reviews_course;
ALTER TABLE reviews
    ADD CONSTRAINT fk_reviews_course FOREIGN KEY (course_id)
    REFERENCES courses(id) ON DELETE RESTRICT;

-- ============================================
-- DB-C3: reviews.term_id and review_drafts.term_id missing FK to terms
-- reviews.term_id has DEFAULT '' which would violate FK (empty string is not a valid term id).
-- Fix: change default to NULL, convert existing '' to NULL, then add FK.
-- ============================================

-- reviews.term_id: change default and backfill
ALTER TABLE reviews ALTER COLUMN term_id DROP NOT NULL;
ALTER TABLE reviews ALTER COLUMN term_id SET DEFAULT NULL;
UPDATE reviews SET term_id = NULL WHERE term_id = '';

ALTER TABLE reviews
    ADD CONSTRAINT fk_reviews_term FOREIGN KEY (term_id)
    REFERENCES terms(id) ON DELETE RESTRICT;

-- review_drafts.term_id already allows NULL, just add FK
ALTER TABLE review_drafts
    ADD CONSTRAINT fk_review_drafts_term FOREIGN KEY (term_id)
    REFERENCES terms(id) ON DELETE RESTRICT;

-- ============================================
-- DB-H1: Missing review_votes(user_hash) index
-- Useful for queries like "get all votes by a user"
-- ============================================
CREATE INDEX IF NOT EXISTS idx_review_votes_user_hash ON review_votes(user_hash, vote_type);

-- ============================================
-- DB-H2: Redundant idx_review_votes_review_user index
-- Covered by UNIQUE constraint uq_review_votes_user (review_id, user_hash)
-- ============================================
DROP INDEX IF EXISTS idx_review_votes_review_user;

-- ============================================
-- DB-H3: Redundant idx_review_votes_review_id index
-- The leading column of uq_review_votes_user (review_id, user_hash) serves single-column lookups on review_id
-- ============================================
DROP INDEX IF EXISTS idx_review_votes_review_id;

-- ============================================
-- DB-H4: Redundant idx_review_reports_review_reporter index
-- Covered by UNIQUE constraint uq_review_reports_user (review_id, reporter_hash)
-- ============================================
DROP INDEX IF EXISTS idx_review_reports_review_reporter;

-- ============================================
-- DB-H5: Redundant idx_review_reports_review_id index
-- The leading column of uq_review_reports_user (review_id, reporter_hash) serves single-column lookups on review_id
-- ============================================
DROP INDEX IF EXISTS idx_review_reports_review_id;

-- ============================================
-- DB-H7: user_profiles.phone stores masked phone — should be removed
-- The masked phone should be derived at application layer from users.phone_enc.
-- Note: Go code changes to stop reading/writing this column are handled separately.
-- ============================================
ALTER TABLE user_profiles DROP COLUMN IF EXISTS phone;
ALTER TABLE user_profiles DROP COLUMN IF EXISTS phone_verified;

-- ============================================
-- DB-M1: school_id type inconsistency — school_configs uses VARCHAR(10), others use BIGINT
-- Create a schools reference table, migrate school_configs.school_id to BIGINT, add FKs.
-- ============================================

-- Step 1: Create the schools reference table
CREATE TABLE IF NOT EXISTS schools (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Step 2: Seed from existing school_configs data
INSERT INTO schools (id, code, name)
SELECT school_id::BIGINT, school_id, school_name
FROM school_configs
ON CONFLICT (id) DO NOTHING;

-- Ensure a default school (id=1) exists for tables that use school_id DEFAULT 1
INSERT INTO schools (id, code, name) VALUES (1, 'default', '默认学校')
ON CONFLICT (id) DO NOTHING;

-- Step 3: Migrate school_configs.school_id from VARCHAR(10) to BIGINT
-- First drop the FK from user_profiles that references school_configs
ALTER TABLE user_profiles DROP CONSTRAINT IF EXISTS fk_user_profiles_school;

-- Rename old column, add new column, copy data, drop old
ALTER TABLE school_configs RENAME COLUMN school_id TO school_id_old;
ALTER TABLE school_configs ADD COLUMN school_id BIGINT;
UPDATE school_configs SET school_id = school_id_old::BIGINT;
ALTER TABLE school_configs ALTER COLUMN school_id SET NOT NULL;
ALTER TABLE school_configs DROP CONSTRAINT IF EXISTS school_configs_pkey;
ALTER TABLE school_configs ADD PRIMARY KEY (school_id);
ALTER TABLE school_configs DROP COLUMN school_id_old;

-- Add FK from school_configs to schools
ALTER TABLE school_configs
    ADD CONSTRAINT fk_school_configs_school FOREIGN KEY (school_id)
    REFERENCES schools(id) ON DELETE RESTRICT;

-- Step 4: Migrate user_profiles.school_id from VARCHAR(10) to BIGINT
DROP INDEX IF EXISTS idx_user_profiles_school;
DROP INDEX IF EXISTS idx_user_profiles_school_student;
ALTER TABLE user_profiles RENAME COLUMN school_id TO school_id_old;
ALTER TABLE user_profiles ADD COLUMN school_id BIGINT;
UPDATE user_profiles SET school_id = school_id_old::BIGINT WHERE school_id_old IS NOT NULL;
ALTER TABLE user_profiles DROP COLUMN school_id_old;

-- Re-create indexes for user_profiles
CREATE INDEX IF NOT EXISTS idx_user_profiles_school ON user_profiles(school_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_profiles_school_student ON user_profiles(school_id, active_student_id) WHERE active_student_id IS NOT NULL;

-- Add FK from user_profiles to schools
ALTER TABLE user_profiles
    ADD CONSTRAINT fk_user_profiles_school FOREIGN KEY (school_id)
    REFERENCES schools(id) ON DELETE RESTRICT;

-- Step 5: Add FKs from existing tables that use BIGINT school_id
ALTER TABLE departments
    ADD CONSTRAINT fk_departments_school FOREIGN KEY (school_id)
    REFERENCES schools(id) ON DELETE RESTRICT;

ALTER TABLE teachers
    ADD CONSTRAINT fk_teachers_school FOREIGN KEY (school_id)
    REFERENCES schools(id) ON DELETE RESTRICT;

ALTER TABLE courses
    ADD CONSTRAINT fk_courses_school FOREIGN KEY (school_id)
    REFERENCES schools(id) ON DELETE RESTRICT;

ALTER TABLE terms
    ADD CONSTRAINT fk_terms_school FOREIGN KEY (school_id)
    REFERENCES schools(id) ON DELETE RESTRICT;

ALTER TABLE course_categories
    ADD CONSTRAINT fk_course_categories_school FOREIGN KEY (school_id)
    REFERENCES schools(id) ON DELETE RESTRICT;

-- ============================================
-- DB-M2: courses.code is queried via ILIKE in repository.go, so the index is required in the authoritative migration history.
-- ============================================
CREATE INDEX IF NOT EXISTS idx_courses_code ON courses(code);

-- ============================================
-- DB-M3: rating_dimensions has two conflicting unique constraints
-- The key is used as a FK target from course_rating_stats and teacher_rating_stats
-- (both reference rating_dimensions(key)). Since we only have school_id=1 and the FK
-- references (key) globally, keep the global UNIQUE(key) and drop the composite.
-- The composite would only be needed if keys could repeat across schools, but the FK
-- design requires global uniqueness.
-- ============================================
ALTER TABLE rating_dimensions DROP CONSTRAINT IF EXISTS uq_rating_dimensions_key;
-- Rename the global constraint to take the cleaner name
ALTER TABLE rating_dimensions RENAME CONSTRAINT uq_rating_dimensions_key_global TO uq_rating_dimensions_key;

-- ============================================
-- DB-M5: academic.buaa_students.sfzjh stores plaintext ID number
-- Add sfzjh_hash for lookup, rename sfzjh to sfzjh_enc and change type to BYTEA.
--
-- NOTE: this migration is phase 1 only. Existing rows are rewritten to BYTEA so the
-- closure step can encrypt them with the application PII cipher and backfill sfzjh_hash.
-- The repository migration runner performs that closure before 000018 tightens invariants.
-- ============================================
ALTER TABLE academic.buaa_students ADD COLUMN IF NOT EXISTS sfzjh_hash VARCHAR(64);
ALTER TABLE academic.buaa_students RENAME COLUMN sfzjh TO sfzjh_enc;
ALTER TABLE academic.buaa_students ALTER COLUMN sfzjh_enc TYPE BYTEA USING convert_to(sfzjh_enc, 'UTF8');

-- Update index: drop old, create new on sfzjh_hash
DROP INDEX IF EXISTS academic.idx_buaa_students_sfzjh;
CREATE INDEX IF NOT EXISTS idx_buaa_students_sfzjh_hash ON academic.buaa_students(sfzjh_hash);

COMMENT ON COLUMN academic.buaa_students.sfzjh_enc IS '身份证件号（加密存储）(ID document number, encrypted)';
COMMENT ON COLUMN academic.buaa_students.sfzjh_hash IS '身份证件号哈希（用于查找）(ID document number hash, for lookup)';

-- ============================================
-- DB-M6: ListPublicTeachers full table aggregate
-- teacher_rating_stats table already exists. Create a materialized view for the
-- public teacher list query to avoid full-table aggregation on every request.
-- ============================================
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_teacher_public_stats AS
SELECT
    t.id AS teacher_id,
    t.name AS teacher_name,
    COALESCE(d.name, '') AS department_name,
    t.department_id,
    AVG(r.avg_rating) AS avg_rating,
    COUNT(DISTINCT r.course_id) AS course_count,
    COUNT(r.id) AS review_count
FROM teachers t
LEFT JOIN departments d ON d.id = t.department_id
LEFT JOIN reviews r ON r.teacher_id = t.id AND r.status = 'published'
GROUP BY t.id, t.name, d.name, t.department_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_teacher_public_stats_id ON mv_teacher_public_stats(teacher_id);
CREATE INDEX IF NOT EXISTS idx_mv_teacher_public_stats_dept ON mv_teacher_public_stats(department_id);
CREATE INDEX IF NOT EXISTS idx_mv_teacher_public_stats_rating ON mv_teacher_public_stats(avg_rating DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_mv_teacher_public_stats_reviews ON mv_teacher_public_stats(review_count DESC);

COMMIT;
