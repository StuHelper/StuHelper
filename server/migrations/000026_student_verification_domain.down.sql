-- Do not silently destroy facts written through the new verification model.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.student_verification_applications LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.student_verification_attempts LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.student_enrollment_subjects LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.student_subject_conflicts LIMIT 1)
       OR EXISTS (
            SELECT 1
            FROM public.user_verification_credentials
            WHERE kind IN (
                'real_name_identity_check',
                'student_email_outbound_otp',
                'student_email_inbound_challenge',
                'manual_material_review'
            )
               OR verification_application_id IS NOT NULL
               OR enrollment_subject_id IS NOT NULL
       )
    THEN
        RAISE EXCEPTION
            'refusing destructive rollback: student verification domain contains new business facts';
    END IF;
END $$;

DROP INDEX IF EXISTS public.user_verification_credentials_active_method_uidx;
DROP INDEX IF EXISTS public.user_verification_credentials_subject_idx;
DROP INDEX IF EXISTS public.user_verification_credentials_application_idx;

ALTER TABLE public.user_verification_credentials
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_revision,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_manual_expiry,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_state_timestamps,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_adapter_pair,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_assurance,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_roster_dependency,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_class,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_status,
    DROP CONSTRAINT IF EXISTS chk_user_verification_credentials_kind,
    DROP CONSTRAINT IF EXISTS fk_user_verification_credentials_enrollment_subject,
    DROP CONSTRAINT IF EXISTS fk_user_verification_credentials_application;

ALTER TABLE public.user_verification_credentials
    DROP COLUMN IF EXISTS last_evaluated_at,
    DROP COLUMN IF EXISTS revision,
    DROP COLUMN IF EXISTS revoked_reason,
    DROP COLUMN IF EXISTS review_required_reason,
    DROP COLUMN IF EXISTS review_required_at,
    DROP COLUMN IF EXISTS activated_at,
    DROP COLUMN IF EXISTS adapter_version,
    DROP COLUMN IF EXISTS adapter_id,
    DROP COLUMN IF EXISTS assurance,
    DROP COLUMN IF EXISTS roster_dependency,
    DROP COLUMN IF EXISTS credential_class,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS enrollment_subject_id,
    DROP COLUMN IF EXISTS verification_application_id;

ALTER TABLE public.user_verification_credentials
    ADD CONSTRAINT chk_user_verification_credentials_kind
        CHECK (kind IN ('school_sso', 'school_email_otp', 'freshman_material_manual')),
    ADD CONSTRAINT chk_user_verification_credentials_freshman_expiry
        CHECK (kind <> 'freshman_material_manual' OR expires_at IS NOT NULL);

DROP TABLE IF EXISTS public.student_verification_event_outbox;
DROP TABLE IF EXISTS public.student_eligibility_revisions;
DROP TABLE IF EXISTS public.student_subject_conflicts;
DROP TABLE IF EXISTS public.student_enrollment_subjects;
DROP TABLE IF EXISTS public.student_verification_attempts;
DROP TABLE IF EXISTS public.student_verification_applications;
DROP TABLE IF EXISTS public.school_verification_methods;
DROP TABLE IF EXISTS public.school_verification_profiles;
