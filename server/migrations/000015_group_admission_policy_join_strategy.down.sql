ALTER TABLE public.group_admission_policies
    DROP CONSTRAINT IF EXISTS chk_group_admission_policies_join_handling_strategy,
    DROP COLUMN IF EXISTS unverified_join_reject_reason,
    DROP COLUMN IF EXISTS join_handling_strategy;
