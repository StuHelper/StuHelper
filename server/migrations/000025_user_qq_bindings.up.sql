BEGIN;

CREATE TABLE IF NOT EXISTS user_qq_bindings (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    qq_id VARCHAR(64) NOT NULL UNIQUE,
    qq_nickname VARCHAR(255),
    bound_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_qq_bindings_qq_id
    ON user_qq_bindings (qq_id);

CREATE TABLE IF NOT EXISTS user_qq_binding_codes (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_qq_binding_codes_code_hash
    ON user_qq_binding_codes (code_hash);

CREATE INDEX IF NOT EXISTS idx_user_qq_binding_codes_expires_at
    ON user_qq_binding_codes (expires_at);

COMMIT;
