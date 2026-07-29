ALTER TABLE public.domain_event_outbox
    ADD COLUMN revision bigint DEFAULT 1 NOT NULL,
    ADD COLUMN locked_revision bigint;

ALTER TABLE public.domain_event_outbox
    ADD CONSTRAINT chk_domain_event_outbox_revision
        CHECK (revision >= 1),
    ADD CONSTRAINT chk_domain_event_outbox_locked_revision
        CHECK (locked_revision IS NULL OR locked_revision >= 1);

CREATE INDEX domain_event_outbox_processing_lease_idx
    ON public.domain_event_outbox (stream, locked_at, id)
    WHERE status = 'processing';
