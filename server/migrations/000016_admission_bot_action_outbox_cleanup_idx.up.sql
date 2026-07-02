-- F058: 支撑 admission_bot_action_outbox 终态行的定期清理。
-- 终态（succeeded/dead_letter/stale）行不再参与派发，但此前无任何删除/归档，表无限膨胀。
-- 该分区索引让 "DELETE ... WHERE status IN (终态) AND updated_at < cutoff" 走索引扫描。
CREATE INDEX IF NOT EXISTS admission_bot_action_outbox_terminal_cleanup_idx
  ON public.admission_bot_action_outbox (updated_at)
  WHERE status IN ('succeeded', 'dead_letter', 'stale');
