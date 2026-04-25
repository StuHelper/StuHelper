DROP INDEX IF EXISTS idx_notifications_source_course_id;
ALTER TABLE notifications DROP COLUMN IF EXISTS source_course_id;
