-- Remove CHECK constraint for reviews.content_flag
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS chk_reviews_content_flag;
