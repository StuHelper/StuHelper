CREATE TABLE IF NOT EXISTS public.admission_bot_action_outbox (
  id BIGSERIAL PRIMARY KEY,
  action_key TEXT NOT NULL,
  session_id TEXT NOT NULL REFERENCES public.group_admission_sessions(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  platform TEXT NOT NULL,
  bot_self_id TEXT NOT NULL,
  guild_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  qq_id TEXT NOT NULL,
  scheduled_at TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_error TEXT,
  message_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT admission_bot_action_outbox_action_key_unique UNIQUE (action_key),
  CONSTRAINT chk_admission_bot_action_outbox_action CHECK (
    action = ANY (ARRAY['remind'::text, 'release'::text, 'kick'::text, 'blacklist'::text, 'forward'::text])
  ),
  CONSTRAINT chk_admission_bot_action_outbox_status CHECK (
    status = ANY (ARRAY['pending'::text, 'dispatched'::text, 'succeeded'::text, 'failed'::text, 'dead_letter'::text, 'stale'::text])
  ),
  CONSTRAINT chk_admission_bot_action_outbox_attempt_count CHECK (attempt_count >= 0)
);

CREATE INDEX IF NOT EXISTS admission_bot_action_outbox_due_idx
  ON public.admission_bot_action_outbox (platform, bot_self_id, status, scheduled_at, next_attempt_at, id)
  WHERE status IN ('pending', 'failed', 'dispatched');

CREATE INDEX IF NOT EXISTS admission_bot_action_outbox_session_idx
  ON public.admission_bot_action_outbox (session_id, status, id);
