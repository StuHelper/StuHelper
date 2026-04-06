ALTER TABLE school_configs DROP CONSTRAINT IF EXISTS chk_approval_policy;
ALTER TABLE school_configs DROP COLUMN IF EXISTS approval_policy;
