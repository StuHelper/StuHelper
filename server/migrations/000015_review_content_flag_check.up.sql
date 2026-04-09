-- Add CHECK constraint for reviews.content_flag
-- Valid values: NULL (no flag), 'warn' (warning-level sensitive word), 'review' (review-level sensitive word), 'cleared' (admin cleared)
ALTER TABLE reviews ADD CONSTRAINT chk_reviews_content_flag
    CHECK (content_flag IS NULL OR content_flag IN ('warn', 'review', 'cleared'));
