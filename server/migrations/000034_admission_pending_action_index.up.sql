BEGIN;

CREATE INDEX group_admission_sessions_pending_action_idx
    ON group_admission_sessions (platform, bot_self_id, status, updated_at, id)
    WHERE status IN ('joined_muted', 'linked', 'material_submitted', 'verified');

COMMIT;
