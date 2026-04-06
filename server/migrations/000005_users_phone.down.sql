-- Remove secure phone storage columns from users table.
DROP INDEX IF EXISTS idx_users_phone_hash;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS chk_users_phone_hash_format,
    DROP CONSTRAINT IF EXISTS chk_users_phone_secure_pair,
    DROP COLUMN IF EXISTS phone_hash,
    DROP COLUMN IF EXISTS phone_enc;
