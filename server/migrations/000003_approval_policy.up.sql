-- TD-04: 认证方式与审批策略解耦
-- 新增 approval_policy 字段，控制认证结果是否需要人工审核
-- 值：auto（认证通过即批准）、manual（认证通过后仍需人工审核）
ALTER TABLE school_configs ADD COLUMN IF NOT EXISTS approval_policy VARCHAR(20) NOT NULL DEFAULT 'auto';

-- 约束：仅允许 auto / manual
ALTER TABLE school_configs ADD CONSTRAINT chk_approval_policy
  CHECK (approval_policy IN ('auto', 'manual'));

-- 回填：LDAP 学校默认 auto，manual 学校默认 manual
UPDATE school_configs SET approval_policy = 'auto' WHERE verification_method = 'ldap';
UPDATE school_configs SET approval_policy = 'manual' WHERE verification_method = 'manual';
