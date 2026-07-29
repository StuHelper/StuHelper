ALTER TABLE public.notifications
    ADD COLUMN idempotency_key text;

ALTER TABLE public.notifications
    ADD CONSTRAINT chk_notifications_idempotency_key
        CHECK (
            idempotency_key IS NULL
            OR (
                char_length(idempotency_key) BETWEEN 1 AND 255
                AND idempotency_key = btrim(idempotency_key)
            )
        );

CREATE UNIQUE INDEX notifications_idempotency_key_idx
    ON public.notifications (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
