-- M15: 敏感词 review 三态真正落地
-- review 级别敏感词：不公开发布，进入人工审核队列，审核通过后再可见

-- 1. reviews.status 扩展：新增 pending_review
-- PostgreSQL CHECK 约束需要先删后建
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_status_check;
ALTER TABLE reviews ADD CONSTRAINT reviews_status_check
    CHECK (status IN ('published', 'hidden', 'deleted', 'pending_review'));

-- 2. review_replies.status 扩展：同步支持 pending_review
ALTER TABLE review_replies DROP CONSTRAINT IF EXISTS review_replies_status_check;
ALTER TABLE review_replies ADD CONSTRAINT review_replies_status_check
    CHECK (status IN ('published', 'hidden', 'deleted', 'pending_review'));

-- 3. content_flag 扩展明确支持 review 值
-- 原 migration 000004 未加 CHECK 约束，content_flag 为自由 VARCHAR
-- 加上注释明确合法值
COMMENT ON COLUMN reviews.content_flag IS 'null=无标记, warn=警告级敏感词, review=审核级敏感词, cleared=人工复核通过';

-- 4. 索引：管理员查询待审核评课
CREATE INDEX IF NOT EXISTS idx_reviews_pending_review ON reviews (created_at DESC) WHERE status = 'pending_review';
CREATE INDEX IF NOT EXISTS idx_review_replies_pending_review ON review_replies (created_at DESC) WHERE status = 'pending_review';
