-- Independent student-verification domain.
--
-- The target schema is created before migration 000032 irreversibly purges the
-- legacy profile, identity, freshman, and roster data. No shadow-read or
-- compatibility credential is created.

CREATE TABLE public.school_verification_profiles (
    school_id bigint PRIMARY KEY REFERENCES public.schools(id) ON DELETE RESTRICT,
    adapter_id text NOT NULL,
    adapter_version text NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    validation_status text NOT NULL DEFAULT 'pending',
    validation_code text,
    email_domains text[] NOT NULL DEFAULT '{}',
    student_id_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
    name_match_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
    enrollment_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
    manual_form_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
    snapshot_sync_interval_seconds integer NOT NULL DEFAULT 21600,
    snapshot_warning_after_seconds integer NOT NULL DEFAULT 43200,
    snapshot_hard_expiry_seconds integer NOT NULL DEFAULT 172800,
    snapshot_grace_seconds integer NOT NULL DEFAULT 0,
    config_revision bigint NOT NULL DEFAULT 1,
    validated_at timestamp with time zone,
    validated_by_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_school_verification_profiles_adapter
        CHECK (
            adapter_id ~ '^[a-z][a-z0-9_]{1,63}$'
            AND char_length(adapter_version) BETWEEN 1 AND 64
        ),
    CONSTRAINT chk_school_verification_profiles_validation
        CHECK (validation_status IN ('pending', 'valid', 'invalid')),
    CONSTRAINT chk_school_verification_profiles_enabled
        CHECK (NOT enabled OR validation_status = 'valid'),
    CONSTRAINT chk_school_verification_profiles_json
        CHECK (
            jsonb_typeof(student_id_policy) = 'object'
            AND jsonb_typeof(name_match_policy) = 'object'
            AND jsonb_typeof(enrollment_policy) = 'object'
            AND jsonb_typeof(manual_form_schema) = 'object'
        ),
    CONSTRAINT chk_school_verification_profiles_freshness
        CHECK (
            snapshot_sync_interval_seconds > 0
            AND snapshot_warning_after_seconds >= snapshot_sync_interval_seconds
            AND snapshot_hard_expiry_seconds > snapshot_warning_after_seconds
            AND snapshot_grace_seconds >= 0
        ),
    CONSTRAINT chk_school_verification_profiles_revision
        CHECK (config_revision > 0),
    CONSTRAINT chk_school_verification_profiles_validation_metadata
        CHECK (
            (validation_status = 'valid' AND validated_at IS NOT NULL)
            OR validation_status <> 'valid'
        )
);

COMMENT ON TABLE public.school_verification_profiles IS
    'Validated school-level verification policy; a school directory row alone is never an authentication allowlist';
COMMENT ON COLUMN public.school_verification_profiles.student_id_policy IS
    'Declarative, non-executable student identifier policy interpreted by the selected school adapter';
COMMENT ON COLUMN public.school_verification_profiles.manual_form_schema IS
    'Restricted form schema; executable scripts, SQL, and template expressions are forbidden';

CREATE TABLE public.school_verification_methods (
    school_id bigint NOT NULL REFERENCES public.school_verification_profiles(school_id) ON DELETE CASCADE,
    method text NOT NULL,
    display_name text NOT NULL,
    description text NOT NULL DEFAULT '',
    adapter_id text NOT NULL,
    adapter_version text NOT NULL,
    roster_dependency text NOT NULL,
    conditional_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
    public_form_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
    risk_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
    credential_ttl_seconds integer,
    connector_operation_key text,
    secret_reference text,
    privacy_notice_version text,
    privacy_notice jsonb NOT NULL DEFAULT '{}'::jsonb,
    enabled boolean NOT NULL DEFAULT false,
    validation_status text NOT NULL DEFAULT 'pending',
    validation_code text,
    health_status text NOT NULL DEFAULT 'unknown',
    health_code text,
    health_checked_at timestamp with time zone,
    config_revision bigint NOT NULL DEFAULT 1,
    validated_at timestamp with time zone,
    validated_by_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (school_id, method),
    CONSTRAINT chk_school_verification_methods_method
        CHECK (method IN (
            'real_name_identity_check',
            'school_sso',
            'student_email_outbound_otp',
            'student_email_inbound_challenge',
            'manual_material_review'
        )),
    CONSTRAINT chk_school_verification_methods_adapter
        CHECK (
            adapter_id ~ '^[a-z][a-z0-9_]{1,63}$'
            AND char_length(adapter_version) BETWEEN 1 AND 64
        ),
    CONSTRAINT chk_school_verification_methods_roster_dependency
        CHECK (roster_dependency IN ('required', 'independent', 'conditional')),
    CONSTRAINT chk_school_verification_methods_conditional_policy
        CHECK (
            jsonb_typeof(conditional_policy) = 'object'
            AND (
                roster_dependency <> 'conditional'
                OR conditional_policy <> '{}'::jsonb
            )
        ),
    CONSTRAINT chk_school_verification_methods_json
        CHECK (
            jsonb_typeof(public_form_schema) = 'object'
            AND jsonb_typeof(risk_policy) = 'object'
            AND jsonb_typeof(privacy_notice) = 'object'
        ),
    CONSTRAINT chk_school_verification_methods_ttl
        CHECK (
            credential_ttl_seconds IS NULL
            OR credential_ttl_seconds BETWEEN 60 AND 157680000
        ),
    CONSTRAINT chk_school_verification_methods_manual_expiry
        CHECK (
            method <> 'manual_material_review'
            OR credential_ttl_seconds IS NOT NULL
        ),
    CONSTRAINT chk_school_verification_methods_connector_key
        CHECK (
            connector_operation_key IS NULL
            OR connector_operation_key ~ '^[a-z][a-z0-9_.-]{1,127}$'
        ),
    CONSTRAINT chk_school_verification_methods_secret_reference
        CHECK (
            secret_reference IS NULL
            OR (
                char_length(secret_reference) BETWEEN 3 AND 255
                AND secret_reference = btrim(secret_reference)
            )
        ),
    CONSTRAINT chk_school_verification_methods_validation
        CHECK (validation_status IN ('pending', 'valid', 'invalid')),
    CONSTRAINT chk_school_verification_methods_health
        CHECK (health_status IN ('unknown', 'healthy', 'degraded', 'unavailable')),
    CONSTRAINT chk_school_verification_methods_enabled
        CHECK (
            NOT enabled
            OR (
                validation_status = 'valid'
                AND privacy_notice_version IS NOT NULL
                AND privacy_notice <> '{}'::jsonb
            )
        ),
    CONSTRAINT chk_school_verification_methods_revision
        CHECK (config_revision > 0)
);

CREATE INDEX school_verification_methods_available_idx
    ON public.school_verification_methods (school_id, method)
    WHERE enabled AND validation_status = 'valid' AND health_status IN ('healthy', 'degraded');

COMMENT ON TABLE public.school_verification_methods IS
    'Per-school capability registry consumed by generic verification clients and services';
COMMENT ON COLUMN public.school_verification_methods.secret_reference IS
    'Reference name only; never contains a password, token, private key, or connection credential';

CREATE TABLE public.student_verification_applications (
    id character varying(36) PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'created',
    current_method text,
    privacy_notice_version text,
    consented_at timestamp with time zone,
    continuation_hash character varying(64),
    continuation_expires_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    terminal_code text,
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_verification_applications_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_verification_applications_status
        CHECK (status IN (
            'created',
            'in_progress',
            'pending_manual_review',
            'approved',
            'rejected',
            'cancelled',
            'expired'
        )),
    CONSTRAINT chk_student_verification_applications_method
        CHECK (
            current_method IS NULL
            OR current_method IN (
                'real_name_identity_check',
                'school_sso',
                'student_email_outbound_otp',
                'student_email_inbound_challenge',
                'manual_material_review'
            )
        ),
    CONSTRAINT chk_student_verification_applications_consent
        CHECK (
            (privacy_notice_version IS NULL AND consented_at IS NULL)
            OR (privacy_notice_version IS NOT NULL AND consented_at IS NOT NULL)
        ),
    CONSTRAINT chk_student_verification_applications_continuation
        CHECK (
            (continuation_hash IS NULL AND continuation_expires_at IS NULL)
            OR (
                continuation_hash ~ '^[0-9a-f]{64}$'
                AND continuation_expires_at IS NOT NULL
            )
        ),
    CONSTRAINT chk_student_verification_applications_expiry
        CHECK (expires_at > created_at),
    CONSTRAINT chk_student_verification_applications_terminal
        CHECK (
            (status IN ('approved', 'rejected', 'cancelled', 'expired') AND completed_at IS NOT NULL)
            OR (status NOT IN ('approved', 'rejected', 'cancelled', 'expired') AND completed_at IS NULL)
        ),
    CONSTRAINT chk_student_verification_applications_revision
        CHECK (revision > 0)
);

CREATE INDEX student_verification_applications_user_idx
    ON public.student_verification_applications (user_id, created_at DESC, id);
CREATE INDEX student_verification_applications_pending_idx
    ON public.student_verification_applications (school_id, status, updated_at, id)
    WHERE status IN ('created', 'in_progress', 'pending_manual_review');
CREATE UNIQUE INDEX student_verification_applications_active_user_school_uidx
    ON public.student_verification_applications (user_id, school_id)
    WHERE status IN ('created', 'in_progress', 'pending_manual_review');

COMMENT ON TABLE public.student_verification_applications IS
    'Student verification workflow independent from QQ bindings and group-admission sessions';
COMMENT ON COLUMN public.student_verification_applications.continuation_hash IS
    'Hash of an allowlisted short-lived continuation token; never stores a browser redirect URL';

CREATE TABLE public.student_verification_attempts (
    id character varying(36) PRIMARY KEY,
    application_id character varying(36) NOT NULL
        REFERENCES public.student_verification_applications(id) ON DELETE CASCADE,
    attempt_number integer NOT NULL,
    method text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    result_code text,
    adapter_id text NOT NULL,
    adapter_version text NOT NULL,
    evidence_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    started_at timestamp with time zone NOT NULL DEFAULT NOW(),
    completed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_verification_attempts_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_verification_attempts_number
        CHECK (attempt_number > 0),
    CONSTRAINT chk_student_verification_attempts_method
        CHECK (method IN (
            'real_name_identity_check',
            'school_sso',
            'student_email_outbound_otp',
            'student_email_inbound_challenge',
            'manual_material_review'
        )),
    CONSTRAINT chk_student_verification_attempts_status
        CHECK (status IN ('pending', 'succeeded', 'failed', 'unavailable', 'cancelled', 'expired')),
    CONSTRAINT chk_student_verification_attempts_adapter
        CHECK (
            adapter_id ~ '^[a-z][a-z0-9_]{1,63}$'
            AND char_length(adapter_version) BETWEEN 1 AND 64
        ),
    CONSTRAINT chk_student_verification_attempts_metadata
        CHECK (jsonb_typeof(evidence_metadata) = 'object'),
    CONSTRAINT chk_student_verification_attempts_completion
        CHECK (
            (status = 'pending' AND completed_at IS NULL)
            OR (status <> 'pending' AND completed_at IS NOT NULL)
        ),
    UNIQUE (application_id, attempt_number)
);

CREATE INDEX student_verification_attempts_application_idx
    ON public.student_verification_attempts (application_id, attempt_number DESC);

COMMENT ON COLUMN public.student_verification_attempts.evidence_metadata IS
    'Allowlisted non-secret evidence metadata only; raw names, identifiers, passwords, OTPs, and provider payloads are forbidden';

CREATE TABLE public.student_enrollment_subjects (
    id character varying(36) PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    subject_hash character varying(128) NOT NULL,
    person_link_hash character varying(128),
    student_id_hash character varying(64) NOT NULL,
    student_id_display text NOT NULL,
    source_method text NOT NULL,
    binding_status text NOT NULL DEFAULT 'active',
    valid_from date,
    valid_until date,
    activated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    review_required_at timestamp with time zone,
    review_required_reason text,
    revoked_at timestamp with time zone,
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_enrollment_subjects_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_enrollment_subjects_subject_hash
        CHECK (
            char_length(subject_hash) BETWEEN 32 AND 128
            AND (person_link_hash IS NULL OR char_length(person_link_hash) BETWEEN 32 AND 128)
            AND student_id_hash ~ '^[0-9a-f]{64}$'
        ),
    CONSTRAINT chk_student_enrollment_subjects_display
        CHECK (char_length(student_id_display) BETWEEN 1 AND 64),
    CONSTRAINT chk_student_enrollment_subjects_source_method
        CHECK (source_method IN (
            'real_name_identity_check',
            'school_sso',
            'student_email_outbound_otp',
            'student_email_inbound_challenge',
            'manual_material_review'
        )),
    CONSTRAINT chk_student_enrollment_subjects_status
        CHECK (binding_status IN ('active', 'historical', 'review_required', 'revoked')),
    CONSTRAINT chk_student_enrollment_subjects_validity
        CHECK (valid_until IS NULL OR valid_from IS NULL OR valid_until >= valid_from),
    CONSTRAINT chk_student_enrollment_subjects_state_timestamps
        CHECK (
            (
                binding_status = 'review_required'
                AND review_required_at IS NOT NULL
                AND review_required_reason IS NOT NULL
            )
            OR (
                binding_status <> 'review_required'
                AND review_required_at IS NULL
                AND review_required_reason IS NULL
            )
        ),
    CONSTRAINT chk_student_enrollment_subjects_revocation
        CHECK (binding_status <> 'revoked' OR revoked_at IS NOT NULL),
    CONSTRAINT chk_student_enrollment_subjects_revision
        CHECK (revision > 0)
);

CREATE INDEX student_enrollment_subjects_user_idx
    ON public.student_enrollment_subjects (user_id, school_id, binding_status, activated_at DESC);
CREATE UNIQUE INDEX student_enrollment_subjects_active_subject_uidx
    ON public.student_enrollment_subjects (school_id, subject_hash)
    WHERE binding_status IN ('active', 'review_required');

COMMENT ON COLUMN public.student_enrollment_subjects.subject_hash IS
    'School-scoped stable enrollment subject, normally derived from the canonical student identifier';
COMMENT ON COLUMN public.student_enrollment_subjects.person_link_hash IS
    'Optional high-sensitivity signal for controlled same-person review; never authorizes automatic account merging';

CREATE TABLE public.student_subject_conflicts (
    id character varying(36) PRIMARY KEY,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    subject_hash character varying(128) NOT NULL,
    claimant_user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE RESTRICT,
    incumbent_user_id bigint REFERENCES public.users(id) ON DELETE RESTRICT,
    application_id character varying(36)
        REFERENCES public.student_verification_applications(id) ON DELETE SET NULL,
    status text NOT NULL DEFAULT 'open',
    reason_code text NOT NULL,
    resolution_code text,
    resolved_by_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    resolved_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_subject_conflicts_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_subject_conflicts_subject_hash
        CHECK (char_length(subject_hash) BETWEEN 32 AND 128),
    CONSTRAINT chk_student_subject_conflicts_status
        CHECK (status IN ('open', 'under_review', 'resolved', 'dismissed')),
    CONSTRAINT chk_student_subject_conflicts_resolution
        CHECK (
            (status IN ('resolved', 'dismissed') AND resolved_at IS NOT NULL AND resolution_code IS NOT NULL)
            OR (status IN ('open', 'under_review') AND resolved_at IS NULL)
        ),
    CONSTRAINT chk_student_subject_conflicts_distinct_users
        CHECK (incumbent_user_id IS NULL OR incumbent_user_id <> claimant_user_id)
);

CREATE INDEX student_subject_conflicts_open_idx
    ON public.student_subject_conflicts (school_id, status, created_at, id)
    WHERE status IN ('open', 'under_review');
CREATE UNIQUE INDEX student_subject_conflicts_open_claim_uidx
    ON public.student_subject_conflicts (school_id, subject_hash, claimant_user_id)
    WHERE status IN ('open', 'under_review');

CREATE TABLE public.student_eligibility_revisions (
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    revision bigint NOT NULL DEFAULT 1,
    reason_code text NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, school_id),
    CONSTRAINT chk_student_eligibility_revisions_revision
        CHECK (revision > 0),
    CONSTRAINT chk_student_eligibility_revisions_reason
        CHECK (char_length(reason_code) BETWEEN 1 AND 100)
);

COMMENT ON TABLE public.student_eligibility_revisions IS
    'Monotonic invalidation fence only; student eligibility is computed from current facts and is not stored as a permanent boolean';

CREATE TABLE public.student_verification_event_outbox (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    event_id character varying(36) NOT NULL UNIQUE,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    event_type text NOT NULL,
    aggregate_revision bigint NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'pending',
    available_at timestamp with time zone NOT NULL DEFAULT NOW(),
    claimed_at timestamp with time zone,
    published_at timestamp with time zone,
    attempts integer NOT NULL DEFAULT 0,
    last_error_code text,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_verification_event_outbox_event_id
        CHECK (event_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_verification_event_outbox_aggregate
        CHECK (
            aggregate_type IN ('student_eligibility')
            AND char_length(aggregate_id) BETWEEN 1 AND 200
            AND aggregate_revision > 0
        ),
    CONSTRAINT chk_student_verification_event_outbox_event_type
        CHECK (event_type IN (
            'student_eligibility.changed',
            'student_credential.activated',
            'student_credential.revoked',
            'student_credential.review_required'
        )),
    CONSTRAINT chk_student_verification_event_outbox_payload
        CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT chk_student_verification_event_outbox_status
        CHECK (status IN ('pending', 'claimed', 'published', 'dead_letter')),
    CONSTRAINT chk_student_verification_event_outbox_attempts
        CHECK (attempts >= 0),
    CONSTRAINT chk_student_verification_event_outbox_state
        CHECK (
            (status = 'pending' AND claimed_at IS NULL AND published_at IS NULL)
            OR (status = 'claimed' AND claimed_at IS NOT NULL AND published_at IS NULL)
            OR (status = 'published' AND published_at IS NOT NULL)
            OR status = 'dead_letter'
        ),
    UNIQUE (aggregate_type, aggregate_id, aggregate_revision, event_type)
);

CREATE INDEX student_verification_event_outbox_pending_idx
    ON public.student_verification_event_outbox (available_at, id)
    WHERE status = 'pending';

COMMENT ON TABLE public.student_verification_event_outbox IS
    'Durable, idempotent student eligibility invalidation events; payload contains no identity evidence or raw identifiers';

ALTER TABLE public.user_verification_credentials
    DROP CONSTRAINT chk_user_verification_credentials_kind;
ALTER TABLE public.user_verification_credentials
    DROP CONSTRAINT chk_user_verification_credentials_freshman_expiry;

ALTER TABLE public.user_verification_credentials
    ADD COLUMN verification_application_id character varying(36),
    ADD COLUMN enrollment_subject_id character varying(36),
    ADD COLUMN status text,
    ADD COLUMN credential_class text NOT NULL DEFAULT 'formal_student',
    ADD COLUMN roster_dependency text NOT NULL DEFAULT 'independent',
    ADD COLUMN assurance text NOT NULL DEFAULT 'standard',
    ADD COLUMN adapter_id text,
    ADD COLUMN adapter_version text,
    ADD COLUMN activated_at timestamp with time zone DEFAULT NOW(),
    ADD COLUMN review_required_at timestamp with time zone,
    ADD COLUMN review_required_reason text,
    ADD COLUMN revoked_reason text,
    ADD COLUMN revision bigint NOT NULL DEFAULT 1,
    ADD COLUMN last_evaluated_at timestamp with time zone;

UPDATE public.user_verification_credentials
SET status = CASE
        WHEN revoked_at IS NOT NULL THEN 'revoked'
        WHEN expires_at IS NOT NULL AND expires_at <= NOW() THEN 'expired'
        ELSE 'active'
    END,
    credential_class = CASE
        WHEN kind = 'freshman_material_manual' THEN 'temporary_freshman'
        ELSE 'formal_student'
    END,
    roster_dependency = CASE
        WHEN kind = 'freshman_material_manual' THEN 'independent'
        ELSE 'conditional'
    END,
    assurance = CASE
        WHEN kind = 'freshman_material_manual' THEN 'reviewed'
        ELSE 'standard'
    END,
    activated_at = verified_at,
    last_evaluated_at = COALESCE(updated_at, verified_at);

ALTER TABLE public.user_verification_credentials
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN status SET DEFAULT 'active';

ALTER TABLE public.user_verification_credentials
    ADD CONSTRAINT fk_user_verification_credentials_application
        FOREIGN KEY (verification_application_id)
        REFERENCES public.student_verification_applications(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_user_verification_credentials_enrollment_subject
        FOREIGN KEY (enrollment_subject_id)
        REFERENCES public.student_enrollment_subjects(id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_user_verification_credentials_kind
        CHECK (kind IN (
            'school_sso',
            'school_email_otp',
            'freshman_material_manual',
            'real_name_identity_check',
            'student_email_outbound_otp',
            'student_email_inbound_challenge',
            'manual_material_review'
        )),
    ADD CONSTRAINT chk_user_verification_credentials_status
        CHECK (status IN ('pending', 'active', 'review_required', 'expired', 'revoked', 'rejected')),
    ADD CONSTRAINT chk_user_verification_credentials_class
        CHECK (credential_class IN ('formal_student', 'temporary_freshman')),
    ADD CONSTRAINT chk_user_verification_credentials_roster_dependency
        CHECK (roster_dependency IN ('required', 'independent', 'conditional')),
    ADD CONSTRAINT chk_user_verification_credentials_assurance
        CHECK (assurance IN ('standard', 'reviewed')),
    ADD CONSTRAINT chk_user_verification_credentials_adapter_pair
        CHECK (
            (adapter_id IS NULL AND adapter_version IS NULL)
            OR (
                adapter_id ~ '^[a-z][a-z0-9_]{1,63}$'
                AND char_length(adapter_version) BETWEEN 1 AND 64
            )
        ),
    ADD CONSTRAINT chk_user_verification_credentials_state_timestamps
        CHECK (
            (status NOT IN ('active', 'review_required') OR activated_at IS NOT NULL)
            AND (
                (
                    status = 'review_required'
                    AND review_required_at IS NOT NULL
                    AND review_required_reason IS NOT NULL
                )
                OR (
                    status <> 'review_required'
                    AND review_required_at IS NULL
                    AND review_required_reason IS NULL
                )
            )
            AND (status <> 'revoked' OR revoked_at IS NOT NULL)
        ),
    ADD CONSTRAINT chk_user_verification_credentials_manual_expiry
        CHECK (
            kind NOT IN ('freshman_material_manual', 'manual_material_review')
            OR expires_at IS NOT NULL
        ),
    ADD CONSTRAINT chk_user_verification_credentials_revision
        CHECK (revision > 0);

CREATE INDEX user_verification_credentials_application_idx
    ON public.user_verification_credentials (verification_application_id)
    WHERE verification_application_id IS NOT NULL;
CREATE INDEX user_verification_credentials_subject_idx
    ON public.user_verification_credentials (enrollment_subject_id, status, expires_at)
    WHERE enrollment_subject_id IS NOT NULL;
CREATE UNIQUE INDEX user_verification_credentials_active_method_uidx
    ON public.user_verification_credentials (user_id, school_id, kind, subject_hash)
    WHERE verification_application_id IS NOT NULL
      AND status IN ('active', 'review_required')
      AND revoked_at IS NULL;

-- Bootstrap the target BUAA capability registry in a fail-closed state. The
-- configuration must be validated, receive a privacy notice version, and pass
-- health checks before any method can be enabled.
INSERT INTO public.school_verification_profiles (
    school_id,
    adapter_id,
    adapter_version,
    enabled,
    validation_status,
    email_domains,
    student_id_policy,
    name_match_policy,
    enrollment_policy,
    manual_form_schema
)
SELECT
    id,
    'buaa',
    '1',
    false,
    'pending',
    ARRAY['buaa.edu.cn']::text[],
    '{"strategy":"adapter"}'::jsonb,
    '{"strategy":"adapter"}'::jsonb,
    '{"strategy":"adapter","statusMappingRequired":true}'::jsonb,
    '{}'::jsonb
FROM public.schools
WHERE id = 4111010006
ON CONFLICT (school_id) DO NOTHING;

INSERT INTO public.school_verification_methods (
    school_id,
    method,
    display_name,
    description,
    adapter_id,
    adapter_version,
    roster_dependency,
    conditional_policy,
    public_form_schema,
    risk_policy,
    credential_ttl_seconds,
    connector_operation_key,
    enabled,
    validation_status,
    health_status
)
SELECT
    4111010006,
    values.method,
    values.display_name,
    values.description,
    values.adapter_id,
    '1',
    values.roster_dependency,
    values.conditional_policy,
    CASE WHEN values.method = 'manual_material_review' THEN
        '{"fields":[
          {"key":"department","label":"学院或院系","inputType":"text","required":true,"maxLength":100},
          {"key":"studentID","label":"学号或录取编号","inputType":"text","required":true,"maxLength":64},
          {"key":"name","label":"姓名","inputType":"text","required":true,"maxLength":100},
          {"key":"email","label":"学校邮箱","inputType":"email","required":true,"maxLength":320}
        ]}'::jsonb
    ELSE '{}'::jsonb END,
    CASE WHEN values.method = 'manual_material_review' THEN
        '{"maxMaterialBytes":10485760,"maxMaterials":3,"materialRetentionDays":180,"handoffTTLSeconds":1800,"requireEmailVerification":true}'::jsonb
    ELSE '{}'::jsonb END,
    values.credential_ttl_seconds,
    values.connector_operation_key,
    false,
    'pending',
    'unknown'
FROM (VALUES
    (
        'real_name_identity_check',
        '实名信息校验',
        '使用实名信息完成一次性身份校验',
        'buaa',
        'required',
        '{}'::jsonb,
        NULL::integer,
        NULL::text
    ),
    (
        'school_sso',
        '统一身份认证验证',
        '使用学校统一身份认证账号完成一次性校验',
        'buaa_ldap_bind',
        'conditional',
        '{"type":"adapter_assertion","requiredAttribute":"current_student"}'::jsonb,
        NULL::integer,
        'buaa.ldap.authenticate'
    ),
    (
        'student_email_outbound_otp',
        '学校邮箱接收验证码',
        '向规范学号邮箱发送一次性验证码',
        'buaa',
        'required',
        '{}'::jsonb,
        NULL::integer,
        NULL::text
    ),
    (
        'student_email_inbound_challenge',
        '从学校邮箱发送验证邮件',
        '从规范学号邮箱发送一次性挑战邮件',
        'buaa',
        'required',
        '{}'::jsonb,
        NULL::integer,
        NULL::text
    ),
    (
        'manual_material_review',
        '人工材料审核',
        '拍摄并提交学校批准的学生材料',
        'shared_manual_review',
        'independent',
        '{}'::jsonb,
        31536000,
        NULL::text
    )
) AS values(
    method,
    display_name,
    description,
    adapter_id,
    roster_dependency,
    conditional_policy,
    credential_ttl_seconds,
    connector_operation_key
)
WHERE EXISTS (
    SELECT 1
    FROM public.school_verification_profiles
    WHERE school_id = 4111010006
)
ON CONFLICT (school_id, method) DO NOTHING;
