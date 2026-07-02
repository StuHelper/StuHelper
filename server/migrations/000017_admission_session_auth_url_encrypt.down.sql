ALTER TABLE public.group_admission_sessions DROP COLUMN auth_url;
ALTER TABLE public.group_admission_sessions ADD COLUMN auth_url text NOT NULL DEFAULT ''::text;
