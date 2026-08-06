ALTER TABLE public.group_admission_policies
    ADD COLUMN allow_temporary_freshman boolean NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN public.group_admission_policies.allow_temporary_freshman IS
    'Whether this group accepts active temporary_freshman credentials; formal_student credentials are always eligible';
