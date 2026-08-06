-- Weekly roster defaults and persistent, auditable manual full-snapshot jobs.
--
-- Manual jobs carry no Oracle SQL, credentials, host, port, or arbitrary
-- command.  They can only select an already-approved
-- roster_snapshot_upload operation from the campus-connector registry.

ALTER TABLE public.school_verification_profiles
    ALTER COLUMN snapshot_sync_interval_seconds SET DEFAULT 604800,
    ALTER COLUMN snapshot_warning_after_seconds SET DEFAULT 691200,
    ALTER COLUMN snapshot_hard_expiry_seconds SET DEFAULT 1209600;

-- Only migrate the exact former default tuple.  Explicitly customized school
-- policies remain untouched.
UPDATE public.school_verification_profiles
SET snapshot_sync_interval_seconds = 604800,
    snapshot_warning_after_seconds = 691200,
    snapshot_hard_expiry_seconds = 1209600,
    updated_at = NOW()
WHERE snapshot_sync_interval_seconds = 21600
  AND snapshot_warning_after_seconds = 43200
  AND snapshot_hard_expiry_seconds = 172800;

ALTER TABLE public.campus_connector_requests
    ADD COLUMN actor_user_id bigint REFERENCES public.users(id) ON DELETE RESTRICT,
    ADD COLUMN request_reason text,
    ADD COLUMN claimed_at timestamp with time zone,
    ADD COLUMN claim_attempts integer NOT NULL DEFAULT 0;

ALTER TABLE public.campus_connector_requests
    DROP CONSTRAINT chk_campus_connector_requests_kind,
    DROP CONSTRAINT chk_campus_connector_requests_status,
    DROP CONSTRAINT chk_campus_connector_requests_completion;

ALTER TABLE public.campus_connector_requests
    ADD CONSTRAINT chk_campus_connector_requests_kind
        CHECK (request_kind IN (
            'interactive_school_account',
            'roster_snapshot_push',
            'roster_snapshot_manual'
        )),
    ADD CONSTRAINT chk_campus_connector_requests_status
        CHECK (status IN (
            'pending',
            'started',
            'succeeded',
            'failed',
            'timed_out',
            'cancelled'
        )),
    ADD CONSTRAINT chk_campus_connector_requests_completion
        CHECK (
            (
                status IN ('pending', 'started')
                AND completed_at IS NULL
                AND latency_milliseconds IS NULL
            )
            OR (
                status NOT IN ('pending', 'started')
                AND completed_at IS NOT NULL
                AND latency_milliseconds >= 0
            )
        ),
    ADD CONSTRAINT chk_campus_connector_requests_manual_metadata
        CHECK (
            request_kind <> 'roster_snapshot_manual'
            OR (
                actor_user_id IS NOT NULL
                AND request_reason = btrim(request_reason)
                AND char_length(request_reason) BETWEEN 4 AND 500
            )
        ),
    ADD CONSTRAINT chk_campus_connector_requests_claims
        CHECK (
            claim_attempts BETWEEN 0 AND 100
            AND (claim_attempts = 0 OR claimed_at IS NOT NULL)
        );

CREATE UNIQUE INDEX campus_connector_requests_manual_inflight_uidx
    ON public.campus_connector_requests (school_id, operation_key)
    WHERE request_kind = 'roster_snapshot_manual'
      AND status IN ('pending', 'started');

CREATE INDEX campus_connector_requests_manual_claim_idx
    ON public.campus_connector_requests (node_id, status, updated_at, created_at, id)
    WHERE request_kind = 'roster_snapshot_manual'
      AND status IN ('pending', 'started');

COMMENT ON COLUMN public.campus_connector_requests.request_reason IS
    'Audited operator reason for a manual roster sync; never sent to the campus node';
COMMENT ON COLUMN public.campus_connector_requests.claimed_at IS
    'Most recent delivery claim time for a persistent manual roster command';
