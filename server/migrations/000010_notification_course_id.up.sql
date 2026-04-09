-- 通知表增加 source_course_id，支持精准跳转到课程页
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS source_course_id BIGINT;

-- 为按课程过滤通知建索引
CREATE INDEX IF NOT EXISTS idx_notifications_source_course_id ON notifications (source_course_id) WHERE source_course_id IS NOT NULL;
