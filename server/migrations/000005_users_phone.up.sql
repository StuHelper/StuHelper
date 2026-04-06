-- Add secure phone storage columns to users table for phone + OTP login.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone_enc BYTEA,
    ADD COLUMN IF NOT EXISTS phone_hash VARCHAR(64);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_users_phone_secure_pair'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT chk_users_phone_secure_pair
            CHECK (
                (phone_enc IS NULL AND phone_hash IS NULL) OR
                (phone_enc IS NOT NULL AND phone_hash IS NOT NULL)
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_users_phone_hash_format'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT chk_users_phone_hash_format
            CHECK (phone_hash IS NULL OR phone_hash ~ '^[0-9a-f]{64}$');
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_hash
    ON users(phone_hash)
    WHERE phone_hash IS NOT NULL;
