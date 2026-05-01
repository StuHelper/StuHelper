CREATE TABLE IF NOT EXISTS user_mfa_enrollment (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    active BOOLEAN NOT NULL DEFAULT false,
    methods TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    recovery_codes_issued_at TIMESTAMPTZ,
    reset_required BOOLEAN NOT NULL DEFAULT false,
    last_enrolled_at TIMESTAMPTZ,
    last_disabled_at TIMESTAMPTZ,
    last_reset_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_mfa_methods_allowed
        CHECK (methods <@ ARRAY['totp', 'webauthn']::TEXT[]),
    CONSTRAINT chk_user_mfa_active_requires_method
        CHECK (active = false OR cardinality(methods) > 0)
);

CREATE INDEX IF NOT EXISTS idx_user_mfa_enrollment_active
    ON user_mfa_enrollment(active, updated_at DESC);
