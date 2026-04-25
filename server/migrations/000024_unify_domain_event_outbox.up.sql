BEGIN;

CREATE TABLE domain_event_outbox (
    id BIGSERIAL PRIMARY KEY,
    stream TEXT NOT NULL,
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
    CONSTRAINT chk_domain_event_outbox_stream CHECK (stream <> ''),
    CONSTRAINT chk_domain_event_outbox_status CHECK (
        status IN ('pending', 'processing', 'completed', 'failed')
    )
);

CREATE UNIQUE INDEX domain_event_outbox_stream_dedupe_idx
    ON domain_event_outbox(stream, dedupe_key);

CREATE INDEX domain_event_outbox_stream_pending_idx
    ON domain_event_outbox(stream, status, available_at, id)
    WHERE status IN ('pending', 'failed');

INSERT INTO domain_event_outbox (
    stream, job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
)
SELECT
    'user_external_sync', job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
FROM user_external_sync_outbox
ON CONFLICT (stream, dedupe_key) DO NOTHING;

INSERT INTO domain_event_outbox (
    stream, job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
)
SELECT
    'review_fga_sync', job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
FROM review_fga_sync_outbox
ON CONFLICT (stream, dedupe_key) DO NOTHING;

INSERT INTO audit_events (
    id, category, event_type, actor_type, actor_user_id, actor_username,
    action, resource_type, resource_id, scope_school_id,
    before_data, after_data, result, reason, trace_id, request_id,
    ip_address, user_agent, details, created_at
)
SELECT
    id,
    'admin_operation',
    CASE
        WHEN LOWER(action) LIKE '%delete%' OR LOWER(action) LIKE '%remove%' THEN 'data.delete'
        WHEN LOWER(action) LIKE '%create%' OR LOWER(action) LIKE '%add%' THEN 'data.create'
        ELSE 'data.update'
    END,
    'admin',
    COALESCE(admin_user_id, ''),
    COALESCE(admin_username, ''),
    COALESCE(action, 'unknown'),
    COALESCE(resource_type, 'admin_operation'),
    COALESCE(resource_id, ''),
    NULL,
    old_value,
    new_value,
    'success',
    NULL,
    NULL,
    NULL,
    ip_address,
    user_agent,
    '{}'::jsonb,
    created_at
FROM admin_operation_logs
ON CONFLICT (id) DO NOTHING;

DROP TABLE IF EXISTS user_external_sync_outbox;
DROP TABLE IF EXISTS review_fga_sync_outbox;
DROP TABLE IF EXISTS admin_operation_logs;

COMMIT;
