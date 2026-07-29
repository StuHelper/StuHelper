DROP INDEX IF EXISTS public.domain_event_outbox_processing_lease_idx;

ALTER TABLE public.domain_event_outbox
    DROP CONSTRAINT IF EXISTS chk_domain_event_outbox_locked_revision,
    DROP CONSTRAINT IF EXISTS chk_domain_event_outbox_revision,
    DROP COLUMN IF EXISTS locked_revision,
    DROP COLUMN IF EXISTS revision;
