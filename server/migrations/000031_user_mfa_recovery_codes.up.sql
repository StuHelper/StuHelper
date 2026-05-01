CREATE TABLE IF NOT EXISTS user_mfa_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_mfa_recovery_code_hash UNIQUE (user_id, code_hash)
);

CREATE INDEX IF NOT EXISTS idx_user_mfa_recovery_codes_unused
    ON user_mfa_recovery_codes(user_id, created_at DESC)
    WHERE used_at IS NULL;
