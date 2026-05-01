CREATE TABLE IF NOT EXISTS bot_service_credentials (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL UNIQUE,
    audience TEXT[] NOT NULL,
    scopes TEXT[] NOT NULL,
    expires_at TIMESTAMPTZ,
    rotated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_bot_service_credentials_name_nonempty
        CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_bot_service_credentials_token_hash
        CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_bot_service_credentials_audience_nonempty
        CHECK (cardinality(audience) > 0),
    CONSTRAINT chk_bot_service_credentials_scopes_nonempty
        CHECK (cardinality(scopes) > 0)
);

CREATE INDEX IF NOT EXISTS idx_bot_service_credentials_active
    ON bot_service_credentials (revoked_at, last_used_at DESC);
