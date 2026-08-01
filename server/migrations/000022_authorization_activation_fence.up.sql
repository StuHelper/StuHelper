-- projection_status describes projection health. activated_at is the durable
-- activation fence: a new/restored grant stays inactive until its first
-- verified OpenFGA projection, while reconciliation of an already-active grant
-- may become pending/failed without temporarily locking every administrator
-- out of the DB-backed control plane.
DROP INDEX IF EXISTS public.authorization_grants_active_subject_idx;

CREATE INDEX authorization_grants_active_subject_idx
    ON public.authorization_grants (subject_user_id, role, school_id, section_id)
    WHERE desired_state = 'granted' AND activated_at IS NOT NULL;
