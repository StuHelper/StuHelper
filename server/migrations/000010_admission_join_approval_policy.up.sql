ALTER TABLE public.group_admission_policies
    ADD COLUMN IF NOT EXISTS auto_approve_verified_join boolean DEFAULT true NOT NULL,
    ADD COLUMN IF NOT EXISTS auto_approve_unverified_join boolean DEFAULT true NOT NULL;

UPDATE public.group_admission_policies
SET auto_approve_verified_join = auto_approve_join,
    auto_approve_unverified_join = auto_approve_join;
