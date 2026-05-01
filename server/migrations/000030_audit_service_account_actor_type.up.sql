ALTER TABLE audit_events
    DROP CONSTRAINT IF EXISTS chk_audit_events_actor_type;

ALTER TABLE audit_events
    ADD CONSTRAINT chk_audit_events_actor_type
    CHECK (actor_type IN ('user', 'admin', 'system', 'service_account'));
