-- 002_create_identities_table.sql
-- 创建身份认证表

CREATE TABLE IF NOT EXISTS identities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL,
    status SMALLINT DEFAULT 0,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- type: ldap, id_card, manual
-- status: 0=pending, 1=verified, 2=rejected

CREATE INDEX idx_identities_user_id ON identities(user_id);
CREATE INDEX idx_identities_type ON identities(type);
