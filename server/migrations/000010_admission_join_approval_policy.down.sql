ALTER TABLE public.group_admission_policies
    DROP COLUMN IF EXISTS auto_approve_unverified_join,
    DROP COLUMN IF EXISTS auto_approve_verified_join;
