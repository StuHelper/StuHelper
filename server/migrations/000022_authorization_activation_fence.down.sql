DROP INDEX IF EXISTS public.authorization_grants_active_subject_idx;

CREATE INDEX authorization_grants_active_subject_idx
    ON public.authorization_grants (subject_user_id, role, school_id, section_id)
    WHERE desired_state = 'granted' AND projection_status = 'applied';
