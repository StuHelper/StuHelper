BEGIN;

CREATE TABLE user_verification_credentials (
    id VARCHAR(36) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL,
    subject_hash VARCHAR(128) NOT NULL,
    subject_display TEXT NOT NULL,
    source_application_id VARCHAR(36),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    expiry_processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_verification_credentials_kind CHECK (
        kind IN ('school_sso', 'school_email_otp', 'freshman_material_manual')
    ),
    CONSTRAINT chk_user_verification_credentials_freshman_expiry CHECK (
        kind <> 'freshman_material_manual' OR expires_at IS NOT NULL
    )
);

CREATE TABLE group_admission_policies (
    id VARCHAR(36) PRIMARY KEY,
    platform TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    auto_approve_join BOOLEAN NOT NULL DEFAULT TRUE,
    initial_mute_duration_seconds INTEGER NOT NULL DEFAULT 2592000,
    link_wait_seconds INTEGER NOT NULL DEFAULT 3600,
    submission_wait_seconds INTEGER NOT NULL DEFAULT 86400,
    manual_review_timeout_seconds INTEGER NOT NULL DEFAULT 86400,
    reminder_interval_seconds INTEGER NOT NULL DEFAULT 900,
    failed_join_limit INTEGER NOT NULL DEFAULT 3,
    blacklist_duration_seconds INTEGER,
    freshman_channel_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    freshman_channel_closes_at TIMESTAMPTZ NOT NULL,
    freshman_default_expires_at TIMESTAMPTZ NOT NULL,
    forward_raw_material_to_qq BOOLEAN NOT NULL DEFAULT FALSE,
    management_guild_ids TEXT[] NOT NULL DEFAULT '{}',
    max_material_bytes BIGINT NOT NULL DEFAULT 10485760,
    max_extension_days INTEGER NOT NULL DEFAULT 90,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (platform, guild_id),
    CONSTRAINT chk_group_admission_policies_positive_seconds CHECK (
        initial_mute_duration_seconds > 0
        AND link_wait_seconds > 0
        AND submission_wait_seconds > 0
        AND manual_review_timeout_seconds > 0
        AND reminder_interval_seconds > 0
    ),
    CONSTRAINT chk_group_admission_policies_positive_limits CHECK (
        failed_join_limit > 0 AND max_material_bytes > 0 AND max_extension_days > 0
    )
);

CREATE TABLE group_admission_sessions (
    id VARCHAR(36) PRIMARY KEY,
    platform TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    qq_id TEXT NOT NULL,
    qq_nickname TEXT,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    token_hash VARCHAR(128) NOT NULL UNIQUE,
    token_expires_at TIMESTAMPTZ NOT NULL,
    token_consumed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'joined_muted',
    link_wait_deadline_at TIMESTAMPTZ NOT NULL,
    submission_wait_deadline_at TIMESTAMPTZ NOT NULL,
    manual_review_deadline_at TIMESTAMPTZ,
    initial_mute_until TIMESTAMPTZ NOT NULL,
    last_reminded_at TIMESTAMPTZ,
    next_reminder_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    last_bot_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_group_admission_sessions_status CHECK (
        status IN ('joined_muted', 'linked', 'material_submitted', 'verified', 'expired_kicked', 'cancelled')
    )
);

CREATE TABLE freshman_verification_applications (
    id VARCHAR(36) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    admission_session_id VARCHAR(36) REFERENCES group_admission_sessions(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    applicant_name TEXT NOT NULL,
    applicant_name_masked TEXT NOT NULL,
    department_or_major TEXT,
    material_type TEXT NOT NULL,
    provisional_expires_at TIMESTAMPTZ,
    review_reason TEXT,
    reviewed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_by_operator_qq_id TEXT,
    reviewed_at TIMESTAMPTZ,
    forwarded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_freshman_verification_applications_status CHECK (
        status IN ('pending', 'approved', 'rejected')
    ),
    CONSTRAINT chk_freshman_verification_applications_material_type CHECK (
        material_type IN ('admission_notice', 'admission_certificate')
    )
);

ALTER TABLE user_verification_credentials
    ADD CONSTRAINT fk_user_verification_credentials_source_application
    FOREIGN KEY (source_application_id)
    REFERENCES freshman_verification_applications(id)
    ON DELETE SET NULL;

CREATE TABLE freshman_verification_materials (
    id VARCHAR(36) PRIMARY KEY,
    application_id VARCHAR(36) NOT NULL REFERENCES freshman_verification_applications(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (application_id),
    CONSTRAINT chk_freshman_verification_materials_content_type CHECK (
        content_type IN ('image/jpeg', 'image/png', 'image/webp')
    ),
    CONSTRAINT chk_freshman_verification_materials_size CHECK (size_bytes > 0),
    CONSTRAINT chk_freshman_verification_materials_sha256 CHECK (char_length(sha256) = 64)
);

CREATE TABLE group_admission_failures (
    platform TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    qq_id TEXT NOT NULL,
    failure_count INTEGER NOT NULL DEFAULT 0,
    blacklisted_at TIMESTAMPTZ,
    blacklist_expires_at TIMESTAMPTZ,
    last_failure_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (platform, guild_id, qq_id),
    CONSTRAINT chk_group_admission_failures_count CHECK (failure_count >= 0)
);

CREATE UNIQUE INDEX group_admission_sessions_active_qq_idx
    ON group_admission_sessions (platform, guild_id, qq_id)
    WHERE status IN ('joined_muted', 'linked');

CREATE INDEX group_admission_sessions_deadline_idx
    ON group_admission_sessions (status, link_wait_deadline_at, submission_wait_deadline_at);

CREATE INDEX group_admission_sessions_user_idx
    ON group_admission_sessions (user_id, created_at DESC);

CREATE INDEX freshman_verification_applications_pending_idx
    ON freshman_verification_applications (school_id, created_at DESC)
    WHERE status = 'pending';

CREATE UNIQUE INDEX freshman_verification_applications_pending_user_school_idx
    ON freshman_verification_applications (user_id, school_id)
    WHERE status = 'pending';

CREATE INDEX freshman_verification_materials_application_idx
    ON freshman_verification_materials (application_id);

CREATE INDEX group_admission_failures_count_idx
    ON group_admission_failures (platform, guild_id, failure_count DESC);

CREATE INDEX user_verification_credentials_user_idx
    ON user_verification_credentials (user_id, school_id, kind);

CREATE INDEX user_verification_credentials_expiry_idx
    ON user_verification_credentials (expires_at)
    WHERE expires_at IS NOT NULL AND revoked_at IS NULL AND expiry_processed_at IS NULL;

COMMIT;
