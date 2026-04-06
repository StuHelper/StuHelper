-- 通知模块独立化：从 user_hash 切换到 user_id，新增 source 字段

-- 添加新列
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS user_id BIGINT;
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS body TEXT NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS source_module VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS source_id VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS source_url TEXT;

-- 从 users 表回填 user_id（兼容历史上可能存在的不同 users 标识列）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'users'
          AND column_name = 'user_hash'
    ) THEN
        EXECUTE $sql$
            UPDATE notifications n
            SET user_id = u.id
            FROM users u
            WHERE n.user_hash = u.user_hash
              AND n.user_id IS NULL
        $sql$;
    END IF;

    -- 某些旧环境 notifications.user_hash 直接存 external_id，而不是 users.user_hash
    EXECUTE $sql$
        UPDATE notifications n
        SET user_id = u.id
        FROM users u
        WHERE n.user_hash = u.external_id
          AND n.user_id IS NULL
    $sql$;

    IF EXISTS (
        SELECT 1
        FROM notifications
        WHERE user_hash IS NOT NULL
          AND user_id IS NULL
    ) THEN
        RAISE EXCEPTION 'notification migration requires manual user_id backfill for legacy rows';
    END IF;
END $$;

-- 迁移旧字段到新字段
UPDATE notifications SET body = content WHERE body = '' AND content IS NOT NULL AND content != '';
UPDATE notifications SET source_module = 'review' WHERE source_module = '' AND related_type IS NOT NULL AND related_type != '';
UPDATE notifications SET source_id = related_id WHERE source_id = '' AND related_id IS NOT NULL AND related_id != '';

-- user_id 设为 NOT NULL（所有行已回填）
ALTER TABLE notifications ALTER COLUMN user_id SET NOT NULL;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_created ON notifications (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_unread ON notifications (user_id) WHERE is_read = false;

-- 删除旧列（user_hash 不再需要，由 user_id 替代）
ALTER TABLE notifications DROP COLUMN IF EXISTS user_hash;
ALTER TABLE notifications DROP COLUMN IF EXISTS content;
ALTER TABLE notifications DROP COLUMN IF EXISTS related_type;
ALTER TABLE notifications DROP COLUMN IF EXISTS related_id;
