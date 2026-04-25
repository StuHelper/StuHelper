BEGIN;

ALTER TABLE academic.buaa_students DROP CONSTRAINT IF EXISTS chk_buaa_students_sfzjh_secure_pair;
ALTER TABLE academic.buaa_students DROP CONSTRAINT IF EXISTS chk_buaa_students_sfzjh_envelope_v1;
ALTER TABLE academic.buaa_students DROP CONSTRAINT IF EXISTS chk_buaa_students_sfzjh_hash_format;

COMMENT ON COLUMN academic.buaa_students.sfzjh_enc IS '身份证件号（加密存储）(ID document number, encrypted)';
COMMENT ON COLUMN academic.buaa_students.sfzjh_hash IS '身份证件号哈希（用于查找）(ID document number hash, for lookup)';

COMMIT;
