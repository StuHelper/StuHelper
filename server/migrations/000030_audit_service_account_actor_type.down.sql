UPDATE audit_events
SET actor_type = 'system'
WHERE actor_type = 'service_account';

ALTER TABLE audit_events
    DROP CONSTRAINT IF EXISTS chk_audit_events_actor_type;

ALTER TABLE audit_events
    ADD CONSTRAINT chk_audit_events_actor_type
    CHECK (actor_type IN ('user', 'admin', 'system'));
