-- Casdoor is an authentication-only IdP. StuHelper business authorization is
-- managed by authorization_grants and projected to OpenFGA. Retire every
-- unfinished legacy business-role projection so an old worker cannot replay it
-- after the cutover.
UPDATE public.domain_event_outbox
SET status = 'completed',
    locked_at = NULL,
    locked_revision = NULL,
    last_error = 'retired: Casdoor business-role projection replaced by PostgreSQL authorization grants',
    updated_at = NOW()
WHERE stream = 'iam_casdoor_role_sync'
  AND status IN ('pending', 'processing', 'failed', 'dead_letter');
