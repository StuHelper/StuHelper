ALTER TABLE IF EXISTS public.group_admission_sessions
    DROP COLUMN IF EXISTS qq_nickname;

ALTER TABLE IF EXISTS public.user_qq_bindings
    DROP COLUMN IF EXISTS qq_nickname;
