-- Add user_hash column for reverse lookup from review/reply user_hash to internal user_id
ALTER TABLE users ADD COLUMN IF NOT EXISTS user_hash VARCHAR(64);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_user_hash ON users(user_hash) WHERE user_hash IS NOT NULL;

-- Backfill is handled by application startup (HMAC requires app-level key).
