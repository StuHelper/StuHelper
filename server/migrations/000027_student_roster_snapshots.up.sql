-- Versioned, atomic, privacy-preserving local academic roster snapshots.

CREATE TABLE academic.student_roster_snapshots (
    id character varying(36) PRIMARY KEY,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    source_kind text NOT NULL,
    source_version text NOT NULL,
    import_mode text NOT NULL DEFAULT 'full',
    schema_version integer NOT NULL,
    mapping_version text NOT NULL,
    status text NOT NULL DEFAULT 'staging',
    source_started_at timestamp with time zone,
    source_cutoff_at timestamp with time zone NOT NULL,
    import_started_at timestamp with time zone NOT NULL DEFAULT NOW(),
    import_completed_at timestamp with time zone,
    activated_at timestamp with time zone,
    row_count bigint NOT NULL DEFAULT 0,
    eligible_row_count bigint NOT NULL DEFAULT 0,
    deleted_row_count bigint NOT NULL DEFAULT 0,
    checksum character varying(64),
    encryption_key_version integer NOT NULL,
    hmac_key_version integer NOT NULL,
    signature_algorithm text,
    signature_key_id text,
    snapshot_signature bytea,
    connector_node_id character varying(36),
    cursor_from text,
    cursor_to text,
    failure_code text,
    activation_authorized_by_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_student_roster_snapshots_id_school UNIQUE (id, school_id),
    CONSTRAINT uq_student_roster_snapshots_source UNIQUE (school_id, source_kind, source_version),
    CONSTRAINT chk_student_roster_snapshots_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_roster_snapshots_source_kind
        CHECK (source_kind IN ('campus_connector', 'isolated_oracle_worker', 'fixture')),
    CONSTRAINT chk_student_roster_snapshots_import_mode
        CHECK (import_mode IN ('full', 'incremental', 'reconciled_full')),
    CONSTRAINT chk_student_roster_snapshots_version
        CHECK (
            schema_version > 0
            AND char_length(mapping_version) BETWEEN 1 AND 64
            AND char_length(source_version) BETWEEN 1 AND 255
        ),
    CONSTRAINT chk_student_roster_snapshots_status
        CHECK (status IN (
            'staging',
            'validating',
            'ready',
            'active',
            'superseded',
            'failed',
            'rolled_back'
        )),
    CONSTRAINT chk_student_roster_snapshots_counts
        CHECK (
            row_count >= 0
            AND eligible_row_count >= 0
            AND eligible_row_count <= row_count
            AND deleted_row_count >= 0
        ),
    CONSTRAINT chk_student_roster_snapshots_checksum
        CHECK (checksum IS NULL OR checksum ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_student_roster_snapshots_key_versions
        CHECK (encryption_key_version > 0 AND hmac_key_version > 0),
    CONSTRAINT chk_student_roster_snapshots_signature
        CHECK (
            (
                signature_algorithm IS NULL
                AND signature_key_id IS NULL
                AND snapshot_signature IS NULL
            )
            OR (
                signature_algorithm IS NOT NULL
                AND signature_key_id IS NOT NULL
                AND snapshot_signature IS NOT NULL
                AND octet_length(snapshot_signature) > 0
            )
        ),
    CONSTRAINT chk_student_roster_snapshots_cursor
        CHECK (
            (import_mode <> 'incremental' AND cursor_from IS NULL AND cursor_to IS NULL)
            OR (import_mode = 'incremental' AND cursor_from IS NOT NULL AND cursor_to IS NOT NULL)
        ),
    CONSTRAINT chk_student_roster_snapshots_completion
        CHECK (
            (
                status IN ('ready', 'active', 'superseded', 'rolled_back')
                AND import_completed_at IS NOT NULL
                AND checksum IS NOT NULL
                AND failure_code IS NULL
            )
            OR (
                status = 'failed'
                AND import_completed_at IS NOT NULL
                AND failure_code IS NOT NULL
            )
            OR status IN ('staging', 'validating')
        ),
    CONSTRAINT chk_student_roster_snapshots_activation
        CHECK (
            (status = 'active' AND activated_at IS NOT NULL)
            OR status <> 'active'
        )
);

CREATE INDEX student_roster_snapshots_school_status_idx
    ON academic.student_roster_snapshots (school_id, status, source_cutoff_at DESC, id);
CREATE INDEX student_roster_snapshots_retention_idx
    ON academic.student_roster_snapshots (school_id, activated_at DESC, id)
    WHERE status IN ('active', 'superseded', 'rolled_back');
CREATE UNIQUE INDEX student_roster_snapshots_active_school_uidx
    ON academic.student_roster_snapshots (school_id)
    WHERE status = 'active';

COMMENT ON TABLE academic.student_roster_snapshots IS
    'Immutable roster version metadata; online reads use academic.student_roster_active as the single authority pointer';
COMMENT ON COLUMN academic.student_roster_snapshots.connector_node_id IS
    'Non-secret connector node identifier; populated only for a verified connector upload';

CREATE TABLE academic.student_roster_records (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    snapshot_id character varying(36) NOT NULL,
    school_id bigint NOT NULL,
    source_record_key_hash character varying(64) NOT NULL,
    student_id_enc bytea NOT NULL,
    student_id_hash character varying(64) NOT NULL,
    name_enc bytea NOT NULL,
    name_hash character varying(64) NOT NULL,
    document_type text,
    document_number_enc bytea,
    document_number_hash character varying(64),
    phone_enc bytea,
    phone_hash character varying(64),
    encryption_key_version integer NOT NULL,
    hmac_key_version integer NOT NULL,
    student_status text,
    on_campus_status text,
    registration_status text,
    education_level text,
    student_category text,
    enrollment_year integer,
    valid_from date,
    valid_until date,
    current_marker boolean,
    eligibility_status text NOT NULL DEFAULT 'unknown',
    eligibility_code text NOT NULL,
    source_updated_at timestamp with time zone,
    record_checksum character varying(64) NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_student_roster_records_snapshot
        FOREIGN KEY (snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE CASCADE,
    CONSTRAINT uq_student_roster_records_student UNIQUE (snapshot_id, student_id_hash),
    CONSTRAINT uq_student_roster_records_source_key UNIQUE (snapshot_id, source_record_key_hash),
    CONSTRAINT chk_student_roster_records_source_key_hash
        CHECK (source_record_key_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_student_roster_records_required_envelopes
        CHECK (
            octet_length(student_id_enc) > 0
            AND get_byte(student_id_enc, 0) = 1
            AND octet_length(name_enc) > 0
            AND get_byte(name_enc, 0) = 1
        ),
    CONSTRAINT chk_student_roster_records_required_hashes
        CHECK (
            student_id_hash ~ '^[0-9a-f]{64}$'
            AND name_hash ~ '^[0-9a-f]{64}$'
        ),
    CONSTRAINT chk_student_roster_records_document_pair
        CHECK (
            (
                document_number_enc IS NULL
                AND document_number_hash IS NULL
            )
            OR (
                document_number_enc IS NOT NULL
                AND octet_length(document_number_enc) > 0
                AND get_byte(document_number_enc, 0) = 1
                AND document_number_hash ~ '^[0-9a-f]{64}$'
            )
        ),
    CONSTRAINT chk_student_roster_records_document_type
        CHECK (
            (document_number_enc IS NULL AND document_type IS NULL)
            OR (document_number_enc IS NOT NULL AND document_type IS NOT NULL)
        ),
    CONSTRAINT chk_student_roster_records_phone_pair
        CHECK (
            (phone_enc IS NULL AND phone_hash IS NULL)
            OR (
                phone_enc IS NOT NULL
                AND octet_length(phone_enc) > 0
                AND get_byte(phone_enc, 0) = 1
                AND phone_hash ~ '^[0-9a-f]{64}$'
            )
        ),
    CONSTRAINT chk_student_roster_records_key_versions
        CHECK (encryption_key_version > 0 AND hmac_key_version > 0),
    CONSTRAINT chk_student_roster_records_enrollment_year
        CHECK (enrollment_year IS NULL OR enrollment_year BETWEEN 1900 AND 3000),
    CONSTRAINT chk_student_roster_records_validity
        CHECK (valid_until IS NULL OR valid_from IS NULL OR valid_until >= valid_from),
    CONSTRAINT chk_student_roster_records_eligibility
        CHECK (
            eligibility_status IN ('eligible', 'ineligible', 'unknown')
            AND char_length(eligibility_code) BETWEEN 1 AND 100
        ),
    CONSTRAINT chk_student_roster_records_checksum
        CHECK (record_checksum ~ '^[0-9a-f]{64}$')
);

CREATE INDEX student_roster_records_lookup_idx
    ON academic.student_roster_records (school_id, snapshot_id, student_id_hash);
CREATE INDEX student_roster_records_document_lookup_idx
    ON academic.student_roster_records (school_id, snapshot_id, document_number_hash)
    WHERE document_number_hash IS NOT NULL;
CREATE INDEX student_roster_records_phone_lookup_idx
    ON academic.student_roster_records (school_id, snapshot_id, phone_hash)
    WHERE phone_hash IS NOT NULL;
CREATE INDEX student_roster_records_eligibility_idx
    ON academic.student_roster_records (school_id, snapshot_id, eligibility_status);

COMMENT ON TABLE academic.student_roster_records IS
    'Snapshot-scoped student records; direct identifiers are stored only as envelope ciphertext plus keyed HMAC blind indexes';
COMMENT ON COLUMN academic.student_roster_records.document_number_hash IS
    'Keyed deterministic HMAC used for constant-time equality checks; never a plain SHA-256 digest';

CREATE TABLE academic.student_roster_quality_checks (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    snapshot_id character varying(36) NOT NULL
        REFERENCES academic.student_roster_snapshots(id) ON DELETE CASCADE,
    check_key text NOT NULL,
    status text NOT NULL,
    measured jsonb NOT NULL DEFAULT '{}'::jsonb,
    threshold jsonb NOT NULL DEFAULT '{}'::jsonb,
    detail_code text,
    checked_at timestamp with time zone NOT NULL DEFAULT NOW(),
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_student_roster_quality_checks UNIQUE (snapshot_id, check_key),
    CONSTRAINT chk_student_roster_quality_checks_key
        CHECK (check_key ~ '^[a-z][a-z0-9_.-]{1,99}$'),
    CONSTRAINT chk_student_roster_quality_checks_status
        CHECK (status IN ('pending', 'passed', 'warning', 'failed')),
    CONSTRAINT chk_student_roster_quality_checks_json
        CHECK (jsonb_typeof(measured) = 'object' AND jsonb_typeof(threshold) = 'object')
);

COMMENT ON TABLE academic.student_roster_quality_checks IS
    'Aggregate-only snapshot activation gates; values must not contain row-level student data';

CREATE TABLE academic.student_roster_ingestion_batches (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    snapshot_id character varying(36) NOT NULL
        REFERENCES academic.student_roster_snapshots(id) ON DELETE CASCADE,
    batch_number integer NOT NULL,
    replay_key character varying(64) NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    cursor_from text,
    cursor_to text,
    received_count bigint NOT NULL DEFAULT 0,
    inserted_count bigint NOT NULL DEFAULT 0,
    deleted_count bigint NOT NULL DEFAULT 0,
    checksum character varying(64),
    started_at timestamp with time zone NOT NULL DEFAULT NOW(),
    completed_at timestamp with time zone,
    failure_code text,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_student_roster_ingestion_batches_number UNIQUE (snapshot_id, batch_number),
    CONSTRAINT uq_student_roster_ingestion_batches_replay UNIQUE (snapshot_id, replay_key),
    CONSTRAINT chk_student_roster_ingestion_batches_number
        CHECK (batch_number > 0),
    CONSTRAINT chk_student_roster_ingestion_batches_replay_key
        CHECK (replay_key ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_student_roster_ingestion_batches_status
        CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT chk_student_roster_ingestion_batches_counts
        CHECK (
            received_count >= 0
            AND inserted_count >= 0
            AND deleted_count >= 0
            AND inserted_count + deleted_count <= received_count
        ),
    CONSTRAINT chk_student_roster_ingestion_batches_checksum
        CHECK (checksum IS NULL OR checksum ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_student_roster_ingestion_batches_completion
        CHECK (
            (
                status = 'completed'
                AND completed_at IS NOT NULL
                AND checksum IS NOT NULL
                AND failure_code IS NULL
            )
            OR (
                status = 'failed'
                AND completed_at IS NOT NULL
                AND failure_code IS NOT NULL
            )
            OR status IN ('pending', 'processing')
        )
);

CREATE INDEX student_roster_ingestion_batches_resume_idx
    ON academic.student_roster_ingestion_batches (snapshot_id, status, batch_number);

CREATE TABLE academic.student_roster_active (
    school_id bigint PRIMARY KEY REFERENCES public.schools(id) ON DELETE RESTRICT,
    snapshot_id character varying(36) NOT NULL,
    previous_snapshot_id character varying(36),
    activation_revision bigint NOT NULL DEFAULT 1,
    activated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    activated_by_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_student_roster_active_snapshot
        FOREIGN KEY (snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE RESTRICT,
    CONSTRAINT fk_student_roster_active_previous
        FOREIGN KEY (previous_snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE RESTRICT,
    CONSTRAINT chk_student_roster_active_revision
        CHECK (activation_revision > 0),
    CONSTRAINT chk_student_roster_active_distinct
        CHECK (previous_snapshot_id IS NULL OR previous_snapshot_id <> snapshot_id)
);

COMMENT ON TABLE academic.student_roster_active IS
    'Single atomic authority pointer per school; online verification must never infer the active snapshot from timestamps';

ALTER TABLE public.student_verification_attempts
    ADD COLUMN started_roster_snapshot_id character varying(36),
    ADD COLUMN started_roster_revision bigint,
    ADD CONSTRAINT fk_student_verification_attempts_started_snapshot
        FOREIGN KEY (started_roster_snapshot_id)
        REFERENCES academic.student_roster_snapshots(id) ON DELETE SET NULL,
    ADD CONSTRAINT chk_student_verification_attempts_started_snapshot_pair
        CHECK (
            (started_roster_snapshot_id IS NULL AND started_roster_revision IS NULL)
            OR (started_roster_snapshot_id IS NOT NULL AND started_roster_revision > 0)
        );

ALTER TABLE public.student_enrollment_subjects
    ADD COLUMN roster_snapshot_id character varying(36),
    ADD COLUMN roster_snapshot_revision bigint,
    ADD CONSTRAINT fk_student_enrollment_subjects_roster_snapshot
        FOREIGN KEY (roster_snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_student_enrollment_subjects_roster_snapshot_pair
        CHECK (
            (roster_snapshot_id IS NULL AND roster_snapshot_revision IS NULL)
            OR (roster_snapshot_id IS NOT NULL AND roster_snapshot_revision > 0)
        );

ALTER TABLE public.user_verification_credentials
    ADD COLUMN roster_snapshot_id character varying(36),
    ADD COLUMN roster_snapshot_revision bigint,
    ADD CONSTRAINT fk_user_verification_credentials_roster_snapshot
        FOREIGN KEY (roster_snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_user_verification_credentials_roster_snapshot_pair
        CHECK (
            (roster_snapshot_id IS NULL AND roster_snapshot_revision IS NULL)
            OR (roster_snapshot_id IS NOT NULL AND roster_snapshot_revision > 0)
        ),
    ADD CONSTRAINT chk_user_verification_credentials_required_roster
        CHECK (roster_dependency <> 'required' OR roster_snapshot_id IS NOT NULL);

CREATE INDEX user_verification_credentials_roster_snapshot_idx
    ON public.user_verification_credentials (roster_snapshot_id, status)
    WHERE roster_snapshot_id IS NOT NULL;

CREATE TABLE public.student_email_inbound_challenges (
    id character varying(36) PRIMARY KEY,
    application_id character varying(36) NOT NULL
        REFERENCES public.student_verification_applications(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'waiting',
    target_address text NOT NULL,
    expected_sender_hash character varying(64) NOT NULL,
    expected_sender_display text NOT NULL,
    routing_subject text NOT NULL UNIQUE,
    challenge_value_enc bytea NOT NULL,
    challenge_value_hash character varying(64) NOT NULL,
    encryption_key_version integer NOT NULL,
    hmac_key_version integer NOT NULL,
    student_id_hash character varying(64) NOT NULL,
    name_hash character varying(64) NOT NULL,
    enrollment_subject_hash character varying(128) NOT NULL,
    student_id_display text NOT NULL,
    roster_snapshot_id character varying(36) NOT NULL,
    roster_snapshot_revision bigint NOT NULL,
    privacy_notice_version text NOT NULL,
    message_reference_hash character varying(64),
    expires_at timestamp with time zone NOT NULL,
    verified_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_student_email_inbound_challenges_snapshot
        FOREIGN KEY (roster_snapshot_id, school_id)
        REFERENCES academic.student_roster_snapshots(id, school_id) ON DELETE RESTRICT,
    CONSTRAINT chk_student_email_inbound_challenges_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_student_email_inbound_challenges_status
        CHECK (status IN ('waiting', 'verified', 'expired', 'cancelled')),
    CONSTRAINT chk_student_email_inbound_challenges_addresses
        CHECK (
            char_length(target_address) BETWEEN 3 AND 320
            AND expected_sender_hash ~ '^[0-9a-f]{64}$'
            AND char_length(expected_sender_display) BETWEEN 3 AND 320
        ),
    CONSTRAINT chk_student_email_inbound_challenges_routing
        CHECK (
            char_length(routing_subject) BETWEEN 10 AND 200
            AND challenge_value_hash ~ '^[0-9a-f]{64}$'
            AND octet_length(challenge_value_enc) > 0
            AND get_byte(challenge_value_enc, 0) = 1
        ),
    CONSTRAINT chk_student_email_inbound_challenges_keys
        CHECK (encryption_key_version > 0 AND hmac_key_version > 0),
    CONSTRAINT chk_student_email_inbound_challenges_identity
        CHECK (
            student_id_hash ~ '^[0-9a-f]{64}$'
            AND name_hash ~ '^[0-9a-f]{64}$'
            AND char_length(enrollment_subject_hash) BETWEEN 32 AND 128
            AND char_length(student_id_display) BETWEEN 1 AND 64
            AND roster_snapshot_revision > 0
        ),
    CONSTRAINT chk_student_email_inbound_challenges_message
        CHECK (
            message_reference_hash IS NULL
            OR message_reference_hash ~ '^[0-9a-f]{64}$'
        ),
    CONSTRAINT chk_student_email_inbound_challenges_expiry
        CHECK (expires_at > created_at),
    CONSTRAINT chk_student_email_inbound_challenges_completion
        CHECK (
            (status = 'verified' AND verified_at IS NOT NULL AND message_reference_hash IS NOT NULL)
            OR (status <> 'verified' AND verified_at IS NULL)
        )
);

CREATE INDEX student_email_inbound_challenges_waiting_idx
    ON public.student_email_inbound_challenges (routing_subject, expires_at, id)
    WHERE status = 'waiting';
CREATE UNIQUE INDEX student_email_inbound_challenges_waiting_application_uidx
    ON public.student_email_inbound_challenges (application_id)
    WHERE status = 'waiting';
CREATE UNIQUE INDEX student_email_inbound_challenges_message_uidx
    ON public.student_email_inbound_challenges (message_reference_hash)
    WHERE message_reference_hash IS NOT NULL;

COMMENT ON TABLE public.student_email_inbound_challenges IS
    'Short-lived inbound school-email challenges; exact sender values are HMAC-only and challenge values are encrypted';

CREATE TABLE public.student_email_inbound_events (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    event_reference_hash character varying(64) NOT NULL UNIQUE,
    challenge_id character varying(36)
        REFERENCES public.student_email_inbound_challenges(id) ON DELETE SET NULL,
    signature_verified boolean NOT NULL,
    sender_alignment_verified boolean NOT NULL DEFAULT false,
    mail_authentication_verified boolean NOT NULL DEFAULT false,
    challenge_verified boolean NOT NULL DEFAULT false,
    result_code text NOT NULL,
    received_at timestamp with time zone NOT NULL,
    processed_at timestamp with time zone NOT NULL DEFAULT NOW(),
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_student_email_inbound_events_reference
        CHECK (event_reference_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_student_email_inbound_events_result
        CHECK (result_code IN (
            'verified',
            'challenge_not_found',
            'challenge_expired',
            'sender_mismatch',
            'mail_authentication_failed',
            'challenge_mismatch',
            'application_state_changed',
            'roster_changed',
            'replayed'
        )),
    CONSTRAINT chk_student_email_inbound_events_verified
        CHECK (
            result_code <> 'verified'
            OR (
                signature_verified
                AND sender_alignment_verified
                AND mail_authentication_verified
                AND challenge_verified
                AND challenge_id IS NOT NULL
            )
        )
);

COMMENT ON TABLE public.student_email_inbound_events IS
    'Payload-free inbound provider audit facts; raw headers, bodies, addresses, and provider payloads are never persisted';
