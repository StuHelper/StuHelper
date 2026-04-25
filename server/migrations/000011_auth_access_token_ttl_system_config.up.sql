BEGIN;

INSERT INTO system_configs (key, value, description)
VALUES (
    'auth_access_token_ttl_seconds',
    '300',
    'Access Token 有效期（秒）；可在管理后台系统设置中热更新'
)
ON CONFLICT (key) DO NOTHING;

COMMIT;
