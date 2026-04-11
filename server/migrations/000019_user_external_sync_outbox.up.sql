BEGIN;

CREATE TABLE IF NOT EXISTS user_external_sync_outbox (
    id BIGSERIAL PRIMARY KEY,
    job_type TEXT NOT NULL,
    dedupe_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_external_sync_outbox_job_type CHECK (
        job_type IN ('verified_student_role', 'user_profile_projection')
    ),
    CONSTRAINT chk_user_external_sync_outbox_status CHECK (
        status IN ('pending', 'processing', 'completed', 'failed')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_external_sync_outbox_dedupe_key
    ON user_external_sync_outbox(dedupe_key);

CREATE INDEX IF NOT EXISTS idx_user_external_sync_outbox_pending
    ON user_external_sync_outbox(status, available_at, id)
    WHERE status IN ('pending', 'failed');

COMMIT;
