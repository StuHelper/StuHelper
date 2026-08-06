-- Admission consumes student eligibility by decision + monotonic revision.
-- This migration also replaces identity-shaped legacy status names with
-- admission-progress states and rebuilds every dependent partial index.

ALTER TABLE public.group_admission_sessions
    ADD COLUMN IF NOT EXISTS eligibility_revision bigint,
    ADD COLUMN IF NOT EXISTS eligibility_evaluated_at timestamp with time zone,
    ADD COLUMN IF NOT EXISTS requirements_status text;

ALTER TABLE public.admission_bot_action_outbox
    ADD COLUMN IF NOT EXISTS eligibility_revision bigint;

ALTER TABLE public.student_verification_event_outbox
    ADD COLUMN IF NOT EXISTS claim_owner character varying(36);

-- No deployed publisher exists before this migration. Resetting a stranded
-- pre-migration claim is therefore safer than preserving an ownerless lease.
UPDATE public.student_verification_event_outbox
SET status = 'pending',
    claimed_at = NULL,
    claim_owner = NULL,
    available_at = LEAST(available_at, NOW()),
    updated_at = NOW()
WHERE status = 'claimed';

ALTER TABLE public.student_verification_event_outbox
    DROP CONSTRAINT IF EXISTS chk_student_verification_event_outbox_state;
ALTER TABLE public.student_verification_event_outbox
    ADD CONSTRAINT chk_student_verification_event_outbox_state
        CHECK (
            (status = 'pending' AND claimed_at IS NULL AND claim_owner IS NULL AND published_at IS NULL)
            OR (
                status = 'claimed'
                AND claimed_at IS NOT NULL
                AND claim_owner IS NOT NULL
                AND published_at IS NULL
            )
            OR (status = 'published' AND published_at IS NOT NULL AND claim_owner IS NULL)
            OR (status = 'dead_letter' AND claim_owner IS NULL)
        ),
    ADD CONSTRAINT chk_student_verification_event_outbox_claim_owner
        CHECK (
            claim_owner IS NULL
            OR claim_owner ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        );

CREATE INDEX student_verification_event_outbox_claim_lease_idx
    ON public.student_verification_event_outbox (status, claimed_at, available_at, id)
    WHERE status IN ('pending', 'claimed');

-- Preserve the strongest revision that can be reconstructed for legacy
-- action-pending sessions. Rows without a revision are returned to a safe
-- waiting state below; they are never released on a guessed fence.
UPDATE public.group_admission_sessions AS session
SET eligibility_revision = revision.revision,
    eligibility_evaluated_at = COALESCE(session.verified_at, session.updated_at),
    requirements_status = 'awaiting_requirements'
FROM public.group_admission_policies AS policy
JOIN public.student_eligibility_revisions AS revision
  ON revision.school_id = policy.school_id
WHERE session.platform = policy.platform
  AND session.guild_id = policy.guild_id
  AND revision.user_id = session.user_id
  AND session.status = 'verified'
  AND session.cancelled_at IS NULL;

UPDATE public.admission_bot_action_outbox AS action
SET eligibility_revision = session.eligibility_revision
FROM public.group_admission_sessions AS session
WHERE action.session_id = session.id
  AND action.action = 'release'
  AND action.status <> 'succeeded'
  AND session.status = 'verified'
  AND session.cancelled_at IS NULL
  AND session.eligibility_revision IS NOT NULL;

-- An unreleased legacy decision with no reconstructable revision must be
-- reevaluated by the new eligibility consumer before another release action.
UPDATE public.admission_bot_action_outbox AS action
SET status = 'stale',
    last_error = NULL,
    updated_at = NOW()
FROM public.group_admission_sessions AS session
WHERE action.session_id = session.id
  AND action.action = 'release'
  AND action.status IN ('pending', 'failed', 'dispatched', 'dead_letter')
  AND session.status = 'verified'
  AND session.cancelled_at IS NULL
  AND session.eligibility_revision IS NULL;

DROP INDEX IF EXISTS public.group_admission_sessions_active_qq_idx;
DROP INDEX IF EXISTS public.group_admission_sessions_pending_action_idx;

ALTER TABLE public.group_admission_sessions
    DROP CONSTRAINT IF EXISTS chk_group_admission_sessions_status;

UPDATE public.group_admission_sessions
SET status = CASE
        WHEN status = 'joined_muted' THEN 'awaiting_account_link'
        WHEN status = 'linked' THEN 'awaiting_requirements'
        WHEN status = 'material_submitted' THEN 'pending_manual_review'
        WHEN status = 'verified' AND cancelled_at IS NOT NULL THEN 'admitted'
        WHEN status = 'verified' AND eligibility_revision IS NOT NULL THEN 'action_pending'
        WHEN status = 'verified' THEN 'awaiting_requirements'
        WHEN status = 'expired_kicked' THEN 'expired'
        ELSE status
    END,
    verified_at = CASE
        WHEN status = 'verified' AND cancelled_at IS NULL AND eligibility_revision IS NULL THEN NULL
        ELSE verified_at
    END,
    updated_at = NOW()
WHERE status IN ('joined_muted', 'linked', 'material_submitted', 'verified', 'expired_kicked');

-- Older schemas allowed a verified row and another waiting row for the same
-- subject because verified was outside the partial unique index. Keep the
-- most advanced/recent row and cancel the rest before widening the invariant.
WITH ranked AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY platform, guild_id, qq_id
               ORDER BY
                   CASE status
                       WHEN 'action_pending' THEN 5
                       WHEN 'eligible' THEN 4
                       WHEN 'pending_manual_review' THEN 3
                       WHEN 'awaiting_requirements' THEN 2
                       WHEN 'awaiting_account_link' THEN 1
                       ELSE 0
                   END DESC,
                   created_at DESC,
                   updated_at DESC,
                   id DESC
           ) AS row_number
    FROM public.group_admission_sessions
    WHERE status IN (
        'created', 'awaiting_account_link', 'awaiting_requirements',
        'pending_manual_review', 'eligible', 'action_pending'
    )
), cancelled AS (
    UPDATE public.group_admission_sessions AS session
    SET status = 'cancelled',
        cancelled_at = COALESCE(session.cancelled_at, NOW()),
        next_reminder_at = NULL,
        last_bot_error = NULL,
        updated_at = NOW()
    FROM ranked
    WHERE session.id = ranked.id
      AND ranked.row_number > 1
    RETURNING session.id
)
UPDATE public.admission_bot_action_outbox AS action
SET status = 'stale',
    last_error = NULL,
    updated_at = NOW()
FROM cancelled
WHERE action.session_id = cancelled.id
  AND action.status <> 'succeeded';

ALTER TABLE public.group_admission_sessions
    ADD CONSTRAINT chk_group_admission_sessions_status
        CHECK (status IN (
            'created',
            'awaiting_account_link',
            'awaiting_requirements',
            'pending_manual_review',
            'eligible',
            'action_pending',
            'admitted',
            'released',
            'rejected',
            'cancelled',
            'expired'
        )),
    ADD CONSTRAINT chk_group_admission_sessions_eligibility_revision
        CHECK (eligibility_revision IS NULL OR eligibility_revision > 0),
    ADD CONSTRAINT chk_group_admission_sessions_action_revision
        CHECK (status NOT IN ('eligible', 'action_pending') OR eligibility_revision IS NOT NULL),
    ADD CONSTRAINT chk_group_admission_sessions_requirements_status
        CHECK (
            requirements_status IS NULL
            OR requirements_status IN ('awaiting_requirements', 'pending_manual_review')
        );

CREATE UNIQUE INDEX group_admission_sessions_active_qq_idx
    ON public.group_admission_sessions (platform, guild_id, qq_id)
    WHERE status IN (
        'created',
        'awaiting_account_link',
        'awaiting_requirements',
        'pending_manual_review',
        'eligible',
        'action_pending'
    );

CREATE INDEX group_admission_sessions_pending_action_idx
    ON public.group_admission_sessions (platform, bot_self_id, status, updated_at, id)
    WHERE status IN (
        'created',
        'awaiting_account_link',
        'awaiting_requirements',
        'pending_manual_review',
        'eligible',
        'action_pending'
    );

ALTER TABLE public.admission_bot_action_outbox
    ADD CONSTRAINT chk_admission_bot_action_eligibility_revision
        CHECK (eligibility_revision IS NULL OR eligibility_revision > 0),
    ADD CONSTRAINT chk_admission_release_action_revision
        CHECK (
            action <> 'release'
            OR eligibility_revision IS NOT NULL
            OR status IN ('succeeded', 'stale', 'dead_letter')
        );

CREATE INDEX admission_bot_action_outbox_release_revision_idx
    ON public.admission_bot_action_outbox (session_id, eligibility_revision, status, id)
    WHERE action = 'release' AND status IN ('pending', 'failed', 'dispatched');

COMMENT ON COLUMN public.group_admission_sessions.eligibility_revision IS
    'Revision of the derived student eligibility used by the current admission decision; never an identity truth source';
COMMENT ON COLUMN public.group_admission_sessions.requirements_status IS
    'Admission waiting state restored when a not-yet-executed eligibility decision is invalidated';
COMMENT ON COLUMN public.admission_bot_action_outbox.eligibility_revision IS
    'Release-action fence; claim and ACK are valid only while it equals the session eligibility revision';
COMMENT ON COLUMN public.student_verification_event_outbox.claim_owner IS
    'Opaque worker lease owner used to fence stale publishers; never a user or platform identifier';
