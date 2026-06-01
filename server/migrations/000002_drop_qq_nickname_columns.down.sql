ALTER TABLE IF EXISTS public.user_qq_bindings
    ADD COLUMN IF NOT EXISTS qq_nickname character varying(255);

ALTER TABLE IF EXISTS public.group_admission_sessions
    ADD COLUMN IF NOT EXISTS qq_nickname text;
