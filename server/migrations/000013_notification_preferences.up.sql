-- 通知偏好表：用户可按通知类型开关通知
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id BIGINT NOT NULL,
    type    VARCHAR(50) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY (user_id, type)
);
