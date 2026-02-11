-- +goose Up
ALTER TABLE admin_operation_logs ADD COLUMN admin_user_id VARCHAR(100) NOT NULL DEFAULT '';
CREATE INDEX idx_admin_logs_user_id ON admin_operation_logs(admin_user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_admin_logs_user_id;
ALTER TABLE admin_operation_logs DROP COLUMN IF EXISTS admin_user_id;
