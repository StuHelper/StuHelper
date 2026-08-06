DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM public.campus_connector_requests
        WHERE request_kind = 'roster_snapshot_manual'
    ) THEN
        RAISE EXCEPTION
            'refusing destructive rollback: manual roster sync audit facts exist';
    END IF;
END $$;

DROP INDEX IF EXISTS public.campus_connector_requests_manual_claim_idx;
DROP INDEX IF EXISTS public.campus_connector_requests_manual_inflight_uidx;

ALTER TABLE public.campus_connector_requests
    DROP CONSTRAINT chk_campus_connector_requests_claims,
    DROP CONSTRAINT chk_campus_connector_requests_manual_metadata,
    DROP CONSTRAINT chk_campus_connector_requests_completion,
    DROP CONSTRAINT chk_campus_connector_requests_status,
    DROP CONSTRAINT chk_campus_connector_requests_kind;

ALTER TABLE public.campus_connector_requests
    ADD CONSTRAINT chk_campus_connector_requests_kind
        CHECK (request_kind IN ('interactive_school_account', 'roster_snapshot_push')),
    ADD CONSTRAINT chk_campus_connector_requests_status
        CHECK (status IN ('started', 'succeeded', 'failed', 'timed_out', 'cancelled')),
    ADD CONSTRAINT chk_campus_connector_requests_completion
        CHECK (
            (status = 'started' AND completed_at IS NULL AND latency_milliseconds IS NULL)
            OR (
                status <> 'started'
                AND completed_at IS NOT NULL
                AND latency_milliseconds >= 0
            )
        );

ALTER TABLE public.campus_connector_requests
    DROP COLUMN claim_attempts,
    DROP COLUMN claimed_at,
    DROP COLUMN request_reason,
    DROP COLUMN actor_user_id;

-- Rolling back code must not rewrite an operator's current school policy.
-- Restore only the defaults used for newly-created rows by the old version.
ALTER TABLE public.school_verification_profiles
    ALTER COLUMN snapshot_sync_interval_seconds SET DEFAULT 21600,
    ALTER COLUMN snapshot_warning_after_seconds SET DEFAULT 43200,
    ALTER COLUMN snapshot_hard_expiry_seconds SET DEFAULT 172800;
