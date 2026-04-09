BEGIN;

DELETE FROM system_configs
WHERE key = 'auth_access_token_ttl_seconds';

COMMIT;
