-- Independent, reusable manual student-verification workflow.
--
-- Raw form values are encrypted by the application before persistence. Lists
-- expose only the dedicated masked projections below, and object keys never
-- leave the service/repository boundary.

CREATE TABLE public.student_manual_review_cases (
    id character varying(36) PRIMARY KEY,
    application_id character varying(36) NOT NULL UNIQUE
        REFERENCES public.student_verification_applications(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'draft',
    material_type text NOT NULL,
    form_data_enc bytea NOT NULL,
    form_digest character varying(64) NOT NULL,
    encryption_key_version integer NOT NULL,
    student_id_hash character varying(64) NOT NULL,
    student_id_display text NOT NULL,
    applicant_name_masked text NOT NULL,
    email_hash character varying(64),
    email_display text,
    email_verified_at timestamp with time zone,
    email_verification_source text,
    privacy_notice_version text NOT NULL,
    consented_at timestamp with time zone NOT NULL,
    submitted_at timestamp with time zone,
    reviewed_by_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    reviewed_at timestamp with time zone,
    user_visible_reason text,
    internal_risk_note_enc bytea,
    credential_class text,
    credential_expires_at timestamp with time zone,
    credential_id character varying(36)
        REFERENCES public.user_verification_credentials(id) ON DELETE SET NULL,
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_manual_review_cases_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_manual_review_cases_status
        CHECK (status IN (
            'draft', 'pending', 'supplement_required', 'approved',
            'rejected', 'cancelled', 'expired'
        )),
    CONSTRAINT chk_student_manual_review_cases_material_type
        CHECK (material_type IN (
            'campus_card', 'student_card', 'admission_notice', 'other_approved'
        )),
    CONSTRAINT chk_student_manual_review_cases_secure_form
        CHECK (
            octet_length(form_data_enc) BETWEEN 1 AND 131072
            AND form_digest ~ '^[0-9a-f]{64}$'
            AND encryption_key_version > 0
            AND student_id_hash ~ '^[0-9a-f]{64}$'
            AND char_length(student_id_display) BETWEEN 1 AND 64
            AND char_length(applicant_name_masked) BETWEEN 1 AND 100
            AND (
                (email_hash IS NULL AND email_display IS NULL)
                OR (
                    email_hash ~ '^[0-9a-f]{64}$'
                    AND char_length(email_display) BETWEEN 3 AND 320
                )
            )
        ),
    CONSTRAINT chk_student_manual_review_cases_email_verification
        CHECK (
            (email_verified_at IS NULL AND email_verification_source IS NULL)
            OR (
                email_hash IS NOT NULL
                AND email_verified_at IS NOT NULL
                AND email_verification_source = 'outbound_otp'
            )
        ),
    CONSTRAINT chk_student_manual_review_cases_review_state
        CHECK (
            (
                status IN ('approved', 'rejected')
                AND reviewed_by_user_id IS NOT NULL
                AND reviewed_at IS NOT NULL
            )
            OR (
                status NOT IN ('approved', 'rejected')
                AND reviewed_at IS NULL
            )
        ),
    CONSTRAINT chk_student_manual_review_cases_submission
        CHECK (
            (status = 'draft' AND submitted_at IS NULL)
            OR (status <> 'draft' AND submitted_at IS NOT NULL)
        ),
    CONSTRAINT chk_student_manual_review_cases_credential
        CHECK (
            (
                status = 'approved'
                AND credential_class IN ('formal_student', 'temporary_freshman')
                AND credential_expires_at IS NOT NULL
                AND credential_id IS NOT NULL
            )
            OR (
                status <> 'approved'
                AND credential_class IS NULL
                AND credential_expires_at IS NULL
                AND credential_id IS NULL
            )
        ),
    CONSTRAINT chk_student_manual_review_cases_revision CHECK (revision > 0)
);

CREATE INDEX student_manual_review_cases_user_idx
    ON public.student_manual_review_cases (user_id, created_at DESC, id);
CREATE INDEX student_manual_review_cases_queue_idx
    ON public.student_manual_review_cases (school_id, submitted_at, id)
    WHERE status IN ('pending', 'supplement_required');

COMMENT ON TABLE public.student_manual_review_cases IS
    'School-configured manual verification cases independent from QQ and group-admission sessions';
COMMENT ON COLUMN public.student_manual_review_cases.form_data_enc IS
    'Encrypted allowlisted form values; plaintext values are never stored in this table';
COMMENT ON COLUMN public.student_manual_review_cases.internal_risk_note_enc IS
    'Encrypted reviewer-only note; never returned to applicants or list endpoints';

CREATE TABLE public.student_manual_review_materials (
    id character varying(36) PRIMARY KEY,
    case_id character varying(36) NOT NULL
        REFERENCES public.student_manual_review_cases(id) ON DELETE CASCADE,
    object_key text NOT NULL UNIQUE,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL,
    sha256 character varying(64) NOT NULL,
    width integer NOT NULL,
    height integer NOT NULL,
    capture_source text NOT NULL DEFAULT 'web_camera',
    requested_facing_mode text NOT NULL DEFAULT 'environment',
    status text NOT NULL DEFAULT 'active',
    retention_until timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_manual_review_materials_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_manual_review_materials_object_key
        CHECK (
            object_key ~ '^student-verification/manual/[0-9a-f-]{36}/[0-9a-f-]{36}\.(jpg|png|webp)$'
            AND char_length(object_key) <= 255
        ),
    CONSTRAINT chk_student_manual_review_materials_content
        CHECK (
            content_type IN ('image/jpeg', 'image/png', 'image/webp')
            AND size_bytes BETWEEN 1 AND 20971520
            AND sha256 ~ '^[0-9a-f]{64}$'
            AND width BETWEEN 320 AND 12000
            AND height BETWEEN 320 AND 12000
            AND (width::bigint * height::bigint) <= 40000000
        ),
    CONSTRAINT chk_student_manual_review_materials_capture
        CHECK (
            capture_source = 'web_camera'
            AND requested_facing_mode IN ('environment', 'unknown')
        ),
    CONSTRAINT chk_student_manual_review_materials_status
        CHECK (status IN ('active', 'deleted')),
    CONSTRAINT chk_student_manual_review_materials_retention
        CHECK (retention_until > created_at),
    CONSTRAINT chk_student_manual_review_materials_deletion
        CHECK (
            (status = 'deleted' AND deleted_at IS NOT NULL)
            OR (status = 'active' AND deleted_at IS NULL)
        )
);

CREATE INDEX student_manual_review_materials_case_idx
    ON public.student_manual_review_materials (case_id, created_at, id)
    WHERE status = 'active';
CREATE INDEX student_manual_review_materials_retention_idx
    ON public.student_manual_review_materials (retention_until, id)
    WHERE status = 'active';

COMMENT ON TABLE public.student_manual_review_materials IS
    'Private camera-captured review evidence; object keys are repository-internal and are never list API fields';
COMMENT ON COLUMN public.student_manual_review_materials.requested_facing_mode IS
    'Browser request intent only; it is not device attestation and must never be described as proof of a rear camera';

CREATE TABLE public.student_manual_camera_handoffs (
    id character varying(36) PRIMARY KEY,
    case_id character varying(36) NOT NULL
        REFERENCES public.student_manual_review_cases(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash character varying(64) NOT NULL UNIQUE,
    token_enc bytea,
    encryption_key_version integer,
    status text NOT NULL DEFAULT 'pending',
    material_id character varying(36)
        REFERENCES public.student_manual_review_materials(id) ON DELETE SET NULL,
    continue_on text,
    expires_at timestamp with time zone NOT NULL,
    uploaded_at timestamp with time zone,
    chosen_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_manual_camera_handoffs_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_manual_camera_handoffs_token
        CHECK (
            token_hash ~ '^[0-9a-f]{64}$'
            AND (
                (
                    status IN ('pending', 'uploaded')
                    AND token_enc IS NOT NULL
                    AND octet_length(token_enc) BETWEEN 1 AND 1024
                    AND encryption_key_version > 0
                )
                OR (
                    status IN ('locked', 'expired')
                    AND token_enc IS NULL
                    AND encryption_key_version IS NULL
                )
            )
        ),
    CONSTRAINT chk_student_manual_camera_handoffs_status
        CHECK (status IN ('pending', 'uploaded', 'locked', 'expired')),
    CONSTRAINT chk_student_manual_camera_handoffs_continuation
        CHECK (continue_on IS NULL OR continue_on IN ('desktop', 'mobile')),
    CONSTRAINT chk_student_manual_camera_handoffs_expiry
        CHECK (expires_at > created_at),
    CONSTRAINT chk_student_manual_camera_handoffs_upload
        CHECK (
            (
                status IN ('uploaded', 'locked')
                AND material_id IS NOT NULL
                AND uploaded_at IS NOT NULL
            )
            OR (
                status IN ('pending', 'expired')
                AND material_id IS NULL
                AND uploaded_at IS NULL
            )
        ),
    CONSTRAINT chk_student_manual_camera_handoffs_choice
        CHECK (
            (continue_on IS NULL AND chosen_at IS NULL)
            OR (continue_on IS NOT NULL AND chosen_at IS NOT NULL)
        )
);

CREATE UNIQUE INDEX student_manual_camera_handoffs_active_case_uidx
    ON public.student_manual_camera_handoffs (case_id)
    WHERE status IN ('pending', 'uploaded');
CREATE INDEX student_manual_camera_handoffs_expiry_idx
    ON public.student_manual_camera_handoffs (expires_at, id)
    WHERE status IN ('pending', 'uploaded');

CREATE TABLE public.student_manual_review_events (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    case_id character varying(36) NOT NULL
        REFERENCES public.student_manual_review_cases(id) ON DELETE CASCADE,
    case_revision bigint NOT NULL,
    actor_type text NOT NULL,
    actor_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    action text NOT NULL,
    reason_code text,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_manual_review_events_revision CHECK (case_revision > 0),
    CONSTRAINT chk_student_manual_review_events_actor
        CHECK (actor_type IN ('applicant', 'reviewer', 'system')),
    CONSTRAINT chk_student_manual_review_events_action
        CHECK (action IN (
            'draft_saved', 'material_added', 'submitted', 'supplement_requested',
            'approved', 'rejected', 'cancelled', 'expired', 'material_accessed',
            'material_access_requested', 'material_access_failed',
            'material_access_denied', 'material_deleted', 'credential_revoked'
        )),
    CONSTRAINT chk_student_manual_review_events_reason
        CHECK (reason_code IS NULL OR char_length(reason_code) BETWEEN 1 AND 100)
);

CREATE INDEX student_manual_review_events_case_idx
    ON public.student_manual_review_events (case_id, created_at, id);

COMMENT ON TABLE public.student_manual_review_events IS
    'Payload-free workflow history; form values, material keys, URLs and reviewer notes are prohibited';

CREATE TABLE public.school_verification_suggestions (
    id character varying(36) PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    school_name text NOT NULL,
    school_location text,
    status text NOT NULL DEFAULT 'pending',
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_school_verification_suggestions_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_school_verification_suggestions_name
        CHECK (char_length(btrim(school_name)) BETWEEN 2 AND 100),
    CONSTRAINT chk_school_verification_suggestions_location
        CHECK (school_location IS NULL OR char_length(btrim(school_location)) BETWEEN 2 AND 100),
    CONSTRAINT chk_school_verification_suggestions_status
        CHECK (status IN ('pending', 'accepted', 'rejected', 'duplicate'))
);

CREATE INDEX school_verification_suggestions_pending_idx
    ON public.school_verification_suggestions (status, created_at, id)
    WHERE status = 'pending';
CREATE INDEX school_verification_suggestions_user_idx
    ON public.school_verification_suggestions (user_id, created_at DESC, id);
