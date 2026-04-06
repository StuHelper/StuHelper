-- TD-02: 评课内容审核流水线 — warn 状态持久化
-- 新增 content_flag 字段，记录敏感词检查结果
-- 值：null（无标记）、warn（触发警告级敏感词）、cleared（人工复核通过）
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS content_flag VARCHAR(20);

-- 新增 content_flag_cleared_at 和 content_flag_cleared_by
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS content_flag_cleared_at TIMESTAMPTZ;
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS content_flag_cleared_by VARCHAR(255);

-- 索引：管理员查询待复核评课
CREATE INDEX IF NOT EXISTS idx_reviews_content_flag ON reviews (content_flag) WHERE content_flag IS NOT NULL;
