-- Data-preserving rollback: target states are mapped to their legacy serving
-- equivalents, while revision columns are intentionally retained so a later
-- roll-forward does not erase security/audit facts created during migration.

DROP INDEX IF EXISTS public.admission_bot_action_outbox_release_revision_idx;
ALTER TABLE public.admission_bot_action_outbox
    DROP CONSTRAINT IF EXISTS chk_admission_release_action_revision,
    DROP CONSTRAINT IF EXISTS chk_admission_bot_action_eligibility_revision;

DROP INDEX IF EXISTS public.student_verification_event_outbox_claim_lease_idx;
ALTER TABLE public.student_verification_event_outbox
    DROP CONSTRAINT IF EXISTS chk_student_verification_event_outbox_claim_owner,
    DROP CONSTRAINT IF EXISTS chk_student_verification_event_outbox_state;
UPDATE public.student_verification_event_outbox
SET status = 'pending',
    claimed_at = NULL,
    claim_owner = NULL,
    available_at = LEAST(available_at, NOW()),
    updated_at = NOW()
WHERE status = 'claimed';
ALTER TABLE public.student_verification_event_outbox
    ADD CONSTRAINT chk_student_verification_event_outbox_state
        CHECK (
            (status = 'pending' AND claimed_at IS NULL AND published_at IS NULL)
            OR (status = 'claimed' AND claimed_at IS NOT NULL AND published_at IS NULL)
            OR (status = 'published' AND published_at IS NOT NULL)
            OR status = 'dead_letter'
        );
ALTER TABLE public.student_verification_event_outbox
    DROP COLUMN IF EXISTS claim_owner;

DROP INDEX IF EXISTS public.group_admission_sessions_active_qq_idx;
DROP INDEX IF EXISTS public.group_admission_sessions_pending_action_idx;

ALTER TABLE public.group_admission_sessions
    DROP CONSTRAINT IF EXISTS chk_group_admission_sessions_requirements_status,
    DROP CONSTRAINT IF EXISTS chk_group_admission_sessions_action_revision,
    DROP CONSTRAINT IF EXISTS chk_group_admission_sessions_eligibility_revision,
    DROP CONSTRAINT IF EXISTS chk_group_admission_sessions_status;

UPDATE public.group_admission_sessions
SET status = CASE
        WHEN status IN ('created', 'awaiting_account_link') THEN 'joined_muted'
        WHEN status IN ('awaiting_requirements', 'eligible') THEN 'linked'
        WHEN status = 'pending_manual_review' THEN 'material_submitted'
        WHEN status IN ('action_pending', 'admitted', 'released') THEN 'verified'
        WHEN status = 'rejected' THEN 'cancelled'
        WHEN status = 'expired' THEN 'expired_kicked'
        ELSE status
    END,
    cancelled_at = CASE
        WHEN status IN ('admitted', 'released') THEN COALESCE(cancelled_at, NOW())
        ELSE cancelled_at
    END,
    updated_at = NOW()
WHERE status IN (
    'created', 'awaiting_account_link', 'awaiting_requirements',
    'pending_manual_review', 'eligible', 'action_pending',
    'admitted', 'released', 'rejected', 'expired'
);

WITH ranked AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY platform, guild_id, qq_id
               ORDER BY created_at DESC, updated_at DESC, id DESC
           ) AS row_number
    FROM public.group_admission_sessions
    WHERE status IN ('joined_muted', 'linked', 'material_submitted')
)
UPDATE public.group_admission_sessions AS session
SET status = 'cancelled',
    cancelled_at = COALESCE(session.cancelled_at, NOW()),
    next_reminder_at = NULL,
    updated_at = NOW()
FROM ranked
WHERE session.id = ranked.id
  AND ranked.row_number > 1;

ALTER TABLE public.group_admission_sessions
    ADD CONSTRAINT chk_group_admission_sessions_status
        CHECK (status IN (
            'joined_muted', 'linked', 'material_submitted',
            'verified', 'expired_kicked', 'cancelled'
        ));

CREATE UNIQUE INDEX group_admission_sessions_active_qq_idx
    ON public.group_admission_sessions (platform, guild_id, qq_id)
    WHERE status IN ('joined_muted', 'linked', 'material_submitted');

CREATE INDEX group_admission_sessions_pending_action_idx
    ON public.group_admission_sessions (platform, bot_self_id, status, updated_at, id)
    WHERE status IN ('joined_muted', 'linked', 'material_submitted', 'verified');

COMMENT ON COLUMN public.group_admission_sessions.eligibility_revision IS
    'Retained across rollback to preserve migration-period eligibility fences';
COMMENT ON COLUMN public.admission_bot_action_outbox.eligibility_revision IS
    'Retained across rollback to preserve migration-period action audit facts';
