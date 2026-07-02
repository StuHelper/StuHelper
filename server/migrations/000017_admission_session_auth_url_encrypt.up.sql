-- F062: auth_url 此前以明文存储完整 join URL（含可用 token），与 token_hash 同行，
-- DB 泄露即可把任意 QQ 绑定到攻击者账号，token_hash 防护形同虚设。
-- 改为 bytea，由应用层用 PII Cipher（AES-256-GCM 版本化信封）加密后写入。
-- active-dev 无需保留历史明文（且历史明文恰是要消除的风险），直接 drop+add。
ALTER TABLE public.group_admission_sessions DROP COLUMN auth_url;
ALTER TABLE public.group_admission_sessions ADD COLUMN auth_url bytea;
