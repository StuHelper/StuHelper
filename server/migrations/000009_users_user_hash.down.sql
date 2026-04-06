DROP INDEX IF EXISTS idx_users_user_hash;
ALTER TABLE users DROP COLUMN IF EXISTS user_hash;
