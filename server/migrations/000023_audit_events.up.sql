CREATE TABLE audit_events (
    id VARCHAR(36) PRIMARY KEY,
    category TEXT NOT NULL DEFAULT 'audit',
    event_type TEXT NOT NULL,
    actor_type TEXT NOT NULL DEFAULT 'user',
    actor_user_id TEXT NOT NULL DEFAULT '',
    actor_username TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    scope_school_id TEXT,
    before_data JSONB,
    after_data JSONB,
    result TEXT NOT NULL DEFAULT 'success',
    reason TEXT,
    trace_id VARCHAR(32),
    request_id VARCHAR(128),
    ip_address VARCHAR(45),
    user_agent VARCHAR(512),
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_audit_events_category CHECK (category IN ('audit', 'admin_operation', 'domain_event')),
    CONSTRAINT chk_audit_events_actor_type CHECK (actor_type IN ('user', 'admin', 'system')),
    CONSTRAINT chk_audit_events_result CHECK (result IN ('success', 'failure', 'pending'))
);

CREATE INDEX audit_events_created_idx
    ON audit_events (created_at DESC);

CREATE INDEX audit_events_actor_idx
    ON audit_events (actor_type, actor_user_id, created_at DESC);

CREATE INDEX audit_events_category_created_idx
    ON audit_events (category, created_at DESC);

CREATE INDEX audit_events_resource_idx
    ON audit_events (resource_type, resource_id, created_at DESC);

CREATE INDEX audit_events_action_idx
    ON audit_events (action, created_at DESC);
