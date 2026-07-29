DROP INDEX IF EXISTS public.notifications_idempotency_key_idx;

ALTER TABLE public.notifications
    DROP CONSTRAINT IF EXISTS chk_notifications_idempotency_key,
    DROP COLUMN IF EXISTS idempotency_key;
