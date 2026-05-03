BEGIN;

DROP INDEX IF EXISTS user_verification_credentials_expiry_idx;
DROP INDEX IF EXISTS user_verification_credentials_user_idx;
DROP INDEX IF EXISTS group_admission_failures_count_idx;
DROP INDEX IF EXISTS freshman_verification_materials_application_idx;
DROP INDEX IF EXISTS freshman_verification_applications_pending_user_school_idx;
DROP INDEX IF EXISTS freshman_verification_applications_pending_idx;
DROP INDEX IF EXISTS group_admission_sessions_user_idx;
DROP INDEX IF EXISTS group_admission_sessions_deadline_idx;
DROP INDEX IF EXISTS group_admission_sessions_active_qq_idx;

ALTER TABLE user_verification_credentials
    DROP CONSTRAINT IF EXISTS fk_user_verification_credentials_source_application;

DROP TABLE IF EXISTS group_admission_failures;
DROP TABLE IF EXISTS freshman_verification_materials;
DROP TABLE IF EXISTS freshman_verification_applications;
DROP TABLE IF EXISTS group_admission_sessions;
DROP TABLE IF EXISTS group_admission_policies;
DROP TABLE IF EXISTS user_verification_credentials;

COMMIT;
