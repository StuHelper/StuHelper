WITH ranked AS (
    SELECT
        id,
        row_number() OVER (
            PARTITION BY platform, guild_id, qq_id
            ORDER BY created_at DESC, updated_at DESC, id DESC
        ) AS rn
    FROM public.group_admission_sessions
    WHERE status IN ('joined_muted', 'linked', 'material_submitted')
)
UPDATE public.group_admission_sessions session
SET status = 'cancelled',
    cancelled_at = COALESCE(session.cancelled_at, now()),
    next_reminder_at = NULL,
    last_bot_error = NULL,
    updated_at = now()
FROM ranked
WHERE session.id = ranked.id
  AND ranked.rn > 1;

DROP INDEX IF EXISTS public.group_admission_sessions_active_qq_idx;

CREATE UNIQUE INDEX group_admission_sessions_active_qq_idx
    ON public.group_admission_sessions USING btree (platform, guild_id, qq_id)
    WHERE status IN ('joined_muted', 'linked', 'material_submitted');
