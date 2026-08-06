DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM academic.student_roster_snapshots LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.student_email_inbound_challenges LIMIT 1)
       OR EXISTS (
            SELECT 1
            FROM public.user_verification_credentials
            WHERE roster_snapshot_id IS NOT NULL
       )
       OR EXISTS (
            SELECT 1
            FROM public.student_enrollment_subjects
            WHERE roster_snapshot_id IS NOT NULL
       )
    THEN
        RAISE EXCEPTION
            'refusing destructive rollback: versioned roster contains imported or referenced facts';
    END IF;
END $$;

DROP INDEX IF EXISTS public.user_verification_credentials_roster_snapshot_idx;

DROP TABLE IF EXISTS public.student_email_inbound_events;
DROP TABLE IF EXISTS public.student_email_inbound_challenges;

ALTER TABLE public.user_verification_credentials
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_required_roster,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_roster_snapshot_pair,
    DROP CONSTRAINT IF EXISTS fk_user_verification_credentials_roster_snapshot,
    DROP COLUMN IF EXISTS roster_snapshot_revision,
    DROP COLUMN IF EXISTS roster_snapshot_id;

ALTER TABLE public.student_enrollment_subjects
    DROP CONSTRAINT IF EXISTS chk_student_enrollment_subjects_roster_snapshot_pair,
    DROP CONSTRAINT IF EXISTS fk_student_enrollment_subjects_roster_snapshot,
    DROP COLUMN IF EXISTS roster_snapshot_revision,
    DROP COLUMN IF EXISTS roster_snapshot_id;

ALTER TABLE public.student_verification_attempts
    DROP CONSTRAINT IF EXISTS chk_student_verification_attempts_started_snapshot_pair,
    DROP CONSTRAINT IF EXISTS fk_student_verification_attempts_started_snapshot,
    DROP COLUMN IF EXISTS started_roster_revision,
    DROP COLUMN IF EXISTS started_roster_snapshot_id;

DROP TABLE IF EXISTS academic.student_roster_active;
DROP TABLE IF EXISTS academic.student_roster_ingestion_batches;
DROP TABLE IF EXISTS academic.student_roster_quality_checks;
DROP TABLE IF EXISTS academic.student_roster_records;
DROP TABLE IF EXISTS academic.student_roster_snapshots;
