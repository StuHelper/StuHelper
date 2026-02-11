-- +goose Up
ALTER TABLE reviews ADD COLUMN avg_rating DECIMAL(3,2) NOT NULL DEFAULT 0;
CREATE INDEX idx_reviews_avg_rating ON reviews(avg_rating DESC);

-- 回填已有数据
UPDATE reviews SET avg_rating = COALESCE(
    (SELECT AVG(value::numeric) FROM jsonb_each_text(ratings) WHERE value ~ '^\d+(\.\d+)?$'),
    0
);

-- +goose Down
DROP INDEX IF EXISTS idx_reviews_avg_rating;
ALTER TABLE reviews DROP COLUMN IF EXISTS avg_rating;
