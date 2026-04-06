-- Remove deprecated raw phone storage and normalize duplicated profile snapshots.
DROP INDEX IF EXISTS idx_users_phone;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_phone_key,
    DROP COLUMN IF EXISTS phone;

UPDATE user_profiles
SET phone = LEFT(phone, 3) || '****' || RIGHT(phone, 4)
WHERE phone ~ '^1[3-9][0-9]{9}$';
