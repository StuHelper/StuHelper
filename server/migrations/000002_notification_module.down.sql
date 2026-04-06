-- 回滚通知模块独立化

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS user_hash VARCHAR(255);
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS content TEXT;
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS related_type VARCHAR(50);
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS related_id VARCHAR(100);

-- 回填旧字段
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
            SET user_hash = u.user_hash
            FROM users u
            WHERE n.user_id = u.id
              AND n.user_hash IS NULL
        $sql$;
    END IF;

    -- 当前 schema 已保证 external_id 唯一，降级回滚时用它作为兼容映射。
    UPDATE notifications n
    SET user_hash = u.external_id
    FROM users u
    WHERE n.user_id = u.id
      AND n.user_hash IS NULL;
END $$;

UPDATE notifications SET content = body WHERE content IS NULL;
UPDATE notifications SET related_type = source_module WHERE related_type IS NULL;
UPDATE notifications SET related_id = source_id WHERE related_id IS NULL;

DROP INDEX IF EXISTS idx_notifications_user_id_created;
DROP INDEX IF EXISTS idx_notifications_user_id_unread;

ALTER TABLE notifications DROP COLUMN IF EXISTS user_id;
ALTER TABLE notifications DROP COLUMN IF EXISTS body;
ALTER TABLE notifications DROP COLUMN IF EXISTS source_module;
ALTER TABLE notifications DROP COLUMN IF EXISTS source_id;
ALTER TABLE notifications DROP COLUMN IF EXISTS source_url;
