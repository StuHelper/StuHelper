DROP INDEX IF EXISTS idx_reviews_content_flag;
ALTER TABLE reviews DROP COLUMN IF EXISTS content_flag_cleared_by;
ALTER TABLE reviews DROP COLUMN IF EXISTS content_flag_cleared_at;
ALTER TABLE reviews DROP COLUMN IF EXISTS content_flag;
