ALTER TABLE public.group_admission_policies
    ADD COLUMN IF NOT EXISTS join_handling_strategy text DEFAULT 'post_join_guard' NOT NULL,
    ADD COLUMN IF NOT EXISTS unverified_join_reject_reason text DEFAULT '请先完成 StuHelper 学生认证后再申请入群。' NOT NULL;

ALTER TABLE public.group_admission_policies
    DROP CONSTRAINT IF EXISTS chk_group_admission_policies_join_handling_strategy,
    ADD CONSTRAINT chk_group_admission_policies_join_handling_strategy
        CHECK (join_handling_strategy IN ('post_join_guard', 'join_request_review'));
