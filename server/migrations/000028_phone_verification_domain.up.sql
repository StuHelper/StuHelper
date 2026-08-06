-- Account-level phone verification and recoverable Casdoor projection flow.
-- Casdoor remains the only writable authority for the current phone number.

ALTER TABLE public.users
    ADD COLUMN phone_masked character varying(32),
    ADD COLUMN phone_projection_state text,
    ADD COLUMN phone_projection_revision bigint NOT NULL DEFAULT 1,
    ADD COLUMN phone_projection_synced_at timestamp with time zone,
    ADD COLUMN phone_encryption_key_version integer,
    ADD COLUMN phone_hmac_key_version integer;

UPDATE public.users
SET phone_projection_state = CASE
        WHEN phone_enc IS NULL THEN 'absent'
        ELSE 'legacy_unreconciled'
    END;

ALTER TABLE public.users
    ALTER COLUMN phone_projection_state SET NOT NULL,
    ALTER COLUMN phone_projection_state SET DEFAULT 'absent',
    ADD CONSTRAINT chk_users_phone_projection_state
        CHECK (phone_projection_state IN (
            'absent',
            'legacy_unreconciled',
            'syncing',
            'synced',
            'error'
        )),
    ADD CONSTRAINT chk_users_phone_masked
        CHECK (
            phone_masked IS NULL
            OR phone_masked ~ '^\+?86[ *0-9-]{7,24}$'
        ),
    ADD CONSTRAINT chk_users_phone_projection_revision
        CHECK (phone_projection_revision > 0),
    ADD CONSTRAINT chk_users_phone_projection_consistency
        CHECK (
            (
                phone_projection_state = 'absent'
                AND phone_enc IS NULL
                AND phone_hash IS NULL
                AND phone_masked IS NULL
                AND phone_projection_synced_at IS NULL
            )
            OR (
                phone_projection_state = 'synced'
                AND phone_enc IS NOT NULL
                AND phone_hash IS NOT NULL
                AND phone_masked IS NOT NULL
                AND phone_projection_synced_at IS NOT NULL
                AND phone_encryption_key_version > 0
                AND phone_hmac_key_version > 0
            )
            OR phone_projection_state IN ('legacy_unreconciled', 'syncing', 'error')
        );

COMMENT ON COLUMN public.users.phone_enc IS
    'Encrypted full normalized Casdoor phone projection; never a locally writable authority or a masked-number ciphertext';
COMMENT ON COLUMN public.users.phone_projection_state IS
    'Casdoor projection reconciliation state; only synced projections may be used by authorized local business reads';

CREATE TABLE public.phone_binding_operations (
    id character varying(36) PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    operation_kind text NOT NULL,
    status text NOT NULL DEFAULT 'pending_verification',
    verification_method text,
    target_phone_enc bytea,
    target_phone_hash character varying(64),
    target_phone_masked character varying(32),
    encryption_key_version integer,
    hmac_key_version integer,
    casdoor_expected_revision text,
    casdoor_result_revision text,
    failure_code text,
    attempt_count integer NOT NULL DEFAULT 0,
    sms_resend_available_at timestamp with time zone,
    revision bigint NOT NULL DEFAULT 1,
    expires_at timestamp with time zone NOT NULL,
    verified_at timestamp with time zone,
    casdoor_updated_at timestamp with time zone,
    projection_synced_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_phone_binding_operations_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_phone_binding_operations_kind
        CHECK (operation_kind IN ('bind', 'change', 'unbind', 'admin_correction', 'reconcile')),
    CONSTRAINT chk_phone_binding_operations_status
        CHECK (status IN (
            'pending_verification',
            'verification_succeeded',
            'casdoor_update_pending',
            'casdoor_updated',
            'projection_sync_pending',
            'completed',
            'failed',
            'cancelled',
            'expired'
        )),
    CONSTRAINT chk_phone_binding_operations_method
        CHECK (
            verification_method IS NULL
            OR verification_method IN ('school_roster_phone_match', 'sms_possession', 'step_up_mfa', 'reconciliation')
        ),
    CONSTRAINT chk_phone_binding_operations_target
        CHECK (
            (
                operation_kind IN ('unbind', 'reconcile')
                AND target_phone_enc IS NULL
                AND target_phone_hash IS NULL
                AND target_phone_masked IS NULL
                AND encryption_key_version IS NULL
                AND hmac_key_version IS NULL
            )
            OR (
                operation_kind NOT IN ('unbind', 'reconcile')
                AND target_phone_hash ~ '^[0-9a-f]{64}$'
                AND target_phone_masked IS NOT NULL
                AND hmac_key_version > 0
                AND (
                    (
                        target_phone_enc IS NOT NULL
                        AND octet_length(target_phone_enc) > 0
                        AND get_byte(target_phone_enc, 0) = 1
                        AND encryption_key_version > 0
                    )
                    OR (
                        status IN ('completed', 'failed', 'cancelled', 'expired')
                        AND target_phone_enc IS NULL
                        AND encryption_key_version IS NULL
                    )
                )
            )
        ),
    CONSTRAINT chk_phone_binding_operations_masked
        CHECK (
            target_phone_masked IS NULL
            OR target_phone_masked ~ '^\+?86[ *0-9-]{7,24}$'
        ),
    CONSTRAINT chk_phone_binding_operations_attempts
        CHECK (attempt_count >= 0),
    CONSTRAINT chk_phone_binding_operations_revision
        CHECK (revision > 0),
    CONSTRAINT chk_phone_binding_operations_expiry
        CHECK (expires_at > created_at),
    CONSTRAINT chk_phone_binding_operations_progress
        CHECK (
            (status NOT IN (
                'verification_succeeded',
                'casdoor_update_pending',
                'casdoor_updated',
                'projection_sync_pending',
                'completed'
            ) OR verified_at IS NOT NULL)
            AND (status NOT IN ('casdoor_updated', 'projection_sync_pending', 'completed') OR casdoor_updated_at IS NOT NULL)
            AND (status <> 'completed' OR (projection_synced_at IS NOT NULL AND completed_at IS NOT NULL))
            AND (status NOT IN ('failed', 'cancelled', 'expired') OR completed_at IS NOT NULL)
        )
);

CREATE INDEX phone_binding_operations_user_idx
    ON public.phone_binding_operations (user_id, created_at DESC, id);
CREATE INDEX phone_binding_operations_pending_idx
    ON public.phone_binding_operations (status, updated_at, id)
    WHERE status IN (
        'pending_verification',
        'verification_succeeded',
        'casdoor_update_pending',
        'casdoor_updated',
        'projection_sync_pending'
    );
CREATE UNIQUE INDEX phone_binding_operations_active_user_uidx
    ON public.phone_binding_operations (user_id)
    WHERE status IN (
        'pending_verification',
        'verification_succeeded',
        'casdoor_update_pending',
        'casdoor_updated',
        'projection_sync_pending'
    );
CREATE UNIQUE INDEX phone_binding_operations_active_number_uidx
    ON public.phone_binding_operations (target_phone_hash)
    WHERE target_phone_hash IS NOT NULL
      AND status IN (
        'pending_verification',
        'verification_succeeded',
        'casdoor_update_pending',
        'casdoor_updated',
        'projection_sync_pending'
      );

COMMENT ON TABLE public.phone_binding_operations IS
    'Recoverable orchestration around Casdoor updates; rows never grant a phone-gated capability by themselves';
COMMENT ON COLUMN public.phone_binding_operations.target_phone_enc IS
    'Short-lived encrypted target number; purge after completion or terminal failure';

CREATE TABLE public.phone_verification_attempts (
    id character varying(36) PRIMARY KEY,
    operation_id character varying(36) NOT NULL
        REFERENCES public.phone_binding_operations(id) ON DELETE CASCADE,
    attempt_number integer NOT NULL,
    method text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    result_code text,
    provider text,
    challenge_reference_hash character varying(64),
    school_id bigint REFERENCES public.schools(id) ON DELETE RESTRICT,
    enrollment_subject_id character varying(36)
        REFERENCES public.student_enrollment_subjects(id) ON DELETE RESTRICT,
    roster_snapshot_id character varying(36),
    roster_snapshot_revision bigint,
    started_at timestamp with time zone NOT NULL DEFAULT NOW(),
    completed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_phone_verification_attempts_roster_snapshot
        FOREIGN KEY (roster_snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE RESTRICT,
    CONSTRAINT uq_phone_verification_attempts_number UNIQUE (operation_id, attempt_number),
    CONSTRAINT chk_phone_verification_attempts_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_phone_verification_attempts_number
        CHECK (attempt_number > 0),
    CONSTRAINT chk_phone_verification_attempts_method
        CHECK (method IN ('school_roster_phone_match', 'sms_possession', 'step_up_mfa', 'reconciliation')),
    CONSTRAINT chk_phone_verification_attempts_status
        CHECK (status IN ('pending', 'succeeded', 'failed', 'unavailable', 'cancelled', 'expired')),
    CONSTRAINT chk_phone_verification_attempts_challenge
        CHECK (
            challenge_reference_hash IS NULL
            OR challenge_reference_hash ~ '^[0-9a-f]{64}$'
        ),
    CONSTRAINT chk_phone_verification_attempts_roster_evidence
        CHECK (
            (
                method = 'school_roster_phone_match'
                AND school_id IS NOT NULL
                AND enrollment_subject_id IS NOT NULL
                AND roster_snapshot_id IS NOT NULL
                AND roster_snapshot_revision > 0
            )
            OR (
                method <> 'school_roster_phone_match'
                AND school_id IS NULL
                AND enrollment_subject_id IS NULL
                AND roster_snapshot_id IS NULL
                AND roster_snapshot_revision IS NULL
            )
        ),
    CONSTRAINT chk_phone_verification_attempts_completion
        CHECK (
            (status = 'pending' AND completed_at IS NULL)
            OR (status <> 'pending' AND completed_at IS NOT NULL)
        )
);

COMMENT ON COLUMN public.phone_verification_attempts.challenge_reference_hash IS
    'Opaque challenge reference HMAC only; OTP values and provider payloads remain outside PostgreSQL and logs';

CREATE TABLE public.phone_verification_credentials (
    id character varying(36) PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    phone_hash character varying(64) NOT NULL,
    phone_display character varying(32) NOT NULL,
    method text NOT NULL,
    assurance text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    operation_id character varying(36)
        REFERENCES public.phone_binding_operations(id) ON DELETE SET NULL,
    school_id bigint REFERENCES public.schools(id) ON DELETE RESTRICT,
    enrollment_subject_id character varying(36)
        REFERENCES public.student_enrollment_subjects(id) ON DELETE RESTRICT,
    roster_snapshot_id character varying(36),
    roster_snapshot_revision bigint,
    verified_at timestamp with time zone,
    last_confirmed_at timestamp with time zone,
    expires_at timestamp with time zone,
    review_required_at timestamp with time zone,
    review_required_reason text,
    revoked_at timestamp with time zone,
    revoked_reason text,
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_phone_verification_credentials_roster_snapshot
        FOREIGN KEY (roster_snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE RESTRICT,
    CONSTRAINT chk_phone_verification_credentials_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_phone_verification_credentials_hash
        CHECK (phone_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_phone_verification_credentials_display
        CHECK (phone_display ~ '^\+?86[ *0-9-]{7,24}$'),
    CONSTRAINT chk_phone_verification_credentials_method
        CHECK (method IN ('school_roster_phone_match', 'sms_possession')),
    CONSTRAINT chk_phone_verification_credentials_assurance
        CHECK (assurance IN ('school_data_match', 'current_possession')),
    CONSTRAINT chk_phone_verification_credentials_status
        CHECK (status IN ('pending', 'active', 'review_required', 'expired', 'revoked', 'rejected')),
    CONSTRAINT chk_phone_verification_credentials_roster_evidence
        CHECK (
            (
                method = 'school_roster_phone_match'
                AND school_id IS NOT NULL
                AND enrollment_subject_id IS NOT NULL
                AND roster_snapshot_id IS NOT NULL
                AND roster_snapshot_revision > 0
            )
            OR (
                method <> 'school_roster_phone_match'
                AND roster_snapshot_id IS NULL
                AND roster_snapshot_revision IS NULL
            )
        ),
    CONSTRAINT chk_phone_verification_credentials_state_timestamps
        CHECK (
            (status NOT IN ('active', 'review_required', 'expired', 'revoked') OR verified_at IS NOT NULL)
            AND (status NOT IN ('active', 'review_required') OR last_confirmed_at IS NOT NULL)
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
    CONSTRAINT chk_phone_verification_credentials_expiry
        CHECK (expires_at IS NULL OR verified_at IS NULL OR expires_at > verified_at),
    CONSTRAINT chk_phone_verification_credentials_revision
        CHECK (revision > 0)
);

CREATE INDEX phone_verification_credentials_user_idx
    ON public.phone_verification_credentials (user_id, status, verified_at DESC, id);
CREATE UNIQUE INDEX phone_verification_credentials_active_user_uidx
    ON public.phone_verification_credentials (user_id)
    WHERE status IN ('active', 'review_required') AND revoked_at IS NULL;
CREATE UNIQUE INDEX phone_verification_credentials_active_number_uidx
    ON public.phone_verification_credentials (phone_hash)
    WHERE status IN ('active', 'review_required') AND revoked_at IS NULL;

COMMENT ON TABLE public.phone_verification_credentials IS
    'Account-level phone evidence; never a student credential and never inferred from Casdoor Phone being non-empty';

CREATE TABLE public.phone_number_claims (
    phone_hash character varying(64) PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    claim_status text NOT NULL,
    operation_id character varying(36)
        REFERENCES public.phone_binding_operations(id) ON DELETE RESTRICT,
    credential_id character varying(36)
        REFERENCES public.phone_verification_credentials(id) ON DELETE RESTRICT,
    revision bigint NOT NULL DEFAULT 1,
    expires_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_phone_number_claims_hash
        CHECK (phone_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_phone_number_claims_status
        CHECK (claim_status IN ('pending', 'active', 'releasing')),
    CONSTRAINT chk_phone_number_claims_source
        CHECK (
            (claim_status = 'pending' AND operation_id IS NOT NULL AND credential_id IS NULL AND expires_at IS NOT NULL)
            OR (claim_status = 'active' AND credential_id IS NOT NULL AND expires_at IS NULL)
            OR claim_status = 'releasing'
        ),
    CONSTRAINT chk_phone_number_claims_revision
        CHECK (revision > 0)
);

CREATE UNIQUE INDEX phone_number_claims_active_user_uidx
    ON public.phone_number_claims (user_id)
    WHERE claim_status = 'active';
CREATE UNIQUE INDEX phone_number_claims_pending_user_uidx
    ON public.phone_number_claims (user_id)
    WHERE claim_status = 'pending';
CREATE INDEX phone_number_claims_expiry_idx
    ON public.phone_number_claims (expires_at, phone_hash)
    WHERE claim_status = 'pending';

COMMENT ON TABLE public.phone_number_claims IS
    'Cross-phase uniqueness fence for a normalized phone HMAC; conflicts map to a non-enumerable recovery path';

CREATE TABLE public.phone_eligibility_revisions (
    user_id bigint PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    revision bigint NOT NULL DEFAULT 1,
    reason_code text NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_phone_eligibility_revisions_revision
        CHECK (revision > 0),
    CONSTRAINT chk_phone_eligibility_revisions_reason
        CHECK (char_length(reason_code) BETWEEN 1 AND 100)
);

COMMENT ON TABLE public.phone_eligibility_revisions IS
    'Monotonic invalidation fence for account phone gates; no phone_verified boolean is stored';

CREATE TABLE public.phone_binding_outbox (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    operation_id character varying(36) NOT NULL
        REFERENCES public.phone_binding_operations(id) ON DELETE CASCADE,
    action_kind text NOT NULL,
    action_revision bigint NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    attempt_count integer NOT NULL DEFAULT 0,
    available_at timestamp with time zone NOT NULL DEFAULT NOW(),
    lease_owner text,
    lease_expires_at timestamp with time zone,
    last_error_code text,
    completed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_phone_binding_outbox_action UNIQUE (operation_id, action_kind, action_revision),
    CONSTRAINT chk_phone_binding_outbox_action
        CHECK (action_kind IN ('apply_casdoor', 'refresh_projection', 'activate_credential', 'reconcile')),
    CONSTRAINT chk_phone_binding_outbox_revision
        CHECK (action_revision > 0),
    CONSTRAINT chk_phone_binding_outbox_status
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'superseded')),
    CONSTRAINT chk_phone_binding_outbox_attempts
        CHECK (attempt_count >= 0),
    CONSTRAINT chk_phone_binding_outbox_lease
        CHECK (
            (status = 'processing' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
            OR (status <> 'processing' AND lease_owner IS NULL AND lease_expires_at IS NULL)
        ),
    CONSTRAINT chk_phone_binding_outbox_completion
        CHECK (
            (status IN ('completed', 'superseded') AND completed_at IS NOT NULL)
            OR (status NOT IN ('completed', 'superseded') AND completed_at IS NULL)
        )
);

CREATE INDEX phone_binding_outbox_claim_idx
    ON public.phone_binding_outbox (available_at, id)
    WHERE status = 'pending';
CREATE INDEX phone_binding_outbox_lease_idx
    ON public.phone_binding_outbox (lease_expires_at, id)
    WHERE status = 'processing';

COMMENT ON TABLE public.phone_binding_outbox IS
    'Durable idempotent work queue; payload is resolved by operation ID so phone values never enter the outbox';

CREATE TABLE public.phone_eligibility_event_outbox (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    event_id character varying(36) NOT NULL UNIQUE,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    revision bigint NOT NULL,
    event_type text NOT NULL DEFAULT 'phone_eligibility.changed',
    status text NOT NULL DEFAULT 'pending',
    available_at timestamp with time zone NOT NULL DEFAULT NOW(),
    claimed_at timestamp with time zone,
    published_at timestamp with time zone,
    attempts integer NOT NULL DEFAULT 0,
    last_error_code text,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_phone_eligibility_event_outbox_event_id
        CHECK (event_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_phone_eligibility_event_outbox_revision
        CHECK (revision > 0),
    CONSTRAINT chk_phone_eligibility_event_outbox_event_type
        CHECK (event_type = 'phone_eligibility.changed'),
    CONSTRAINT chk_phone_eligibility_event_outbox_status
        CHECK (status IN ('pending', 'claimed', 'published', 'dead_letter')),
    CONSTRAINT chk_phone_eligibility_event_outbox_attempts
        CHECK (attempts >= 0),
    CONSTRAINT chk_phone_eligibility_event_outbox_state
        CHECK (
            (status = 'pending' AND claimed_at IS NULL AND published_at IS NULL)
            OR (status = 'claimed' AND claimed_at IS NOT NULL AND published_at IS NULL)
            OR (status = 'published' AND published_at IS NOT NULL)
            OR status = 'dead_letter'
        ),
    UNIQUE (user_id, revision, event_type)
);

CREATE INDEX phone_eligibility_event_outbox_pending_idx
    ON public.phone_eligibility_event_outbox (available_at, id)
    WHERE status = 'pending';

COMMENT ON TABLE public.phone_eligibility_event_outbox IS
    'Minimal cache-invalidation events containing only internal user ID and monotonic revision';
