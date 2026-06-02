DROP INDEX IF EXISTS public.group_admission_sessions_active_qq_idx;

CREATE UNIQUE INDEX group_admission_sessions_active_qq_idx
    ON public.group_admission_sessions USING btree (platform, guild_id, qq_id)
    WHERE status IN ('joined_muted', 'linked');
