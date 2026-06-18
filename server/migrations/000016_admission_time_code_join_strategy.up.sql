UPDATE public.group_admission_policies
SET join_handling_strategy = 'post_join_time_code'
WHERE join_handling_strategy = 'join_request_time_code';

ALTER TABLE public.group_admission_policies
    DROP CONSTRAINT IF EXISTS chk_group_admission_policies_join_handling_strategy,
    ADD CONSTRAINT chk_group_admission_policies_join_handling_strategy
        CHECK (join_handling_strategy IN ('post_join_guard', 'join_request_review', 'post_join_time_code'));
