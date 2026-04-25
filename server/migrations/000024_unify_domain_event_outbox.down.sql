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

CREATE TABLE IF NOT EXISTS review_fga_sync_outbox (
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
    CONSTRAINT chk_review_fga_sync_outbox_job_type CHECK (
        job_type IN ('review_relations', 'report_relations')
    ),
    CONSTRAINT chk_review_fga_sync_outbox_status CHECK (
        status IN ('pending', 'processing', 'completed', 'failed')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_review_fga_sync_outbox_dedupe_key
    ON review_fga_sync_outbox(dedupe_key);

CREATE INDEX IF NOT EXISTS idx_review_fga_sync_outbox_pending
    ON review_fga_sync_outbox(status, available_at, id)
    WHERE status IN ('pending', 'failed');

CREATE TABLE IF NOT EXISTS admin_operation_logs (
    id VARCHAR(36) PRIMARY KEY,
    admin_username VARCHAR(255) NOT NULL,
    admin_user_id VARCHAR(100) NOT NULL DEFAULT '',
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    ip_address VARCHAR(45),
    user_agent VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_logs_admin ON admin_operation_logs(admin_username);
CREATE INDEX IF NOT EXISTS idx_admin_logs_user_id ON admin_operation_logs(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_action ON admin_operation_logs(action);
CREATE INDEX IF NOT EXISTS idx_admin_logs_resource ON admin_operation_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created ON admin_operation_logs(created_at DESC);

INSERT INTO user_external_sync_outbox (
    job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
)
SELECT
    job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
FROM domain_event_outbox
WHERE stream = 'user_external_sync'
ON CONFLICT (dedupe_key) DO NOTHING;

INSERT INTO review_fga_sync_outbox (
    job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
)
SELECT
    job_type, dedupe_key, payload, status,
    attempt_count, available_at, locked_at, last_error, created_at, updated_at
FROM domain_event_outbox
WHERE stream = 'review_fga_sync'
ON CONFLICT (dedupe_key) DO NOTHING;

INSERT INTO admin_operation_logs (
    id, admin_username, admin_user_id, action, resource_type, resource_id,
    old_value, new_value, ip_address, user_agent, created_at
)
SELECT
    id, actor_username, actor_user_id, action, resource_type, resource_id,
    before_data, after_data, ip_address, user_agent, created_at
FROM audit_events
WHERE category = 'admin_operation'
ON CONFLICT (id) DO NOTHING;

DROP TABLE IF EXISTS domain_event_outbox;

COMMIT;
