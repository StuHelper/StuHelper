-- Revert M15: 敏感词 review 三态

-- 先把 pending_review 状态的数据转为 hidden，防止回滚后违反约束
UPDATE reviews SET status = 'hidden' WHERE status = 'pending_review';
UPDATE review_replies SET status = 'hidden' WHERE status = 'pending_review';

-- 恢复原 CHECK 约束
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_status_check;
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS chk_reviews_status;
ALTER TABLE reviews ADD CONSTRAINT reviews_status_check
    CHECK (status IN ('published', 'hidden', 'deleted'));

ALTER TABLE review_replies DROP CONSTRAINT IF EXISTS review_replies_status_check;
ALTER TABLE review_replies DROP CONSTRAINT IF EXISTS chk_review_replies_status;
ALTER TABLE review_replies ADD CONSTRAINT review_replies_status_check
    CHECK (status IN ('published', 'hidden', 'deleted'));

-- 清理 content_flag = 'review' → 改为 'warn'
UPDATE reviews SET content_flag = 'warn' WHERE content_flag = 'review';

COMMENT ON COLUMN reviews.content_flag IS NULL;

-- 删除索引
DROP INDEX IF EXISTS idx_reviews_pending_review;
DROP INDEX IF EXISTS idx_review_replies_pending_review;
