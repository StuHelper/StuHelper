BEGIN;

ALTER TABLE academic.buaa_students
    ADD CONSTRAINT chk_buaa_students_sfzjh_secure_pair CHECK (
        (sfzjh_enc IS NULL AND sfzjh_hash IS NULL) OR
        (sfzjh_enc IS NOT NULL AND sfzjh_hash IS NOT NULL)
    );

ALTER TABLE academic.buaa_students
    ADD CONSTRAINT chk_buaa_students_sfzjh_envelope_v1 CHECK (
        sfzjh_enc IS NULL OR octet_length(sfzjh_enc) = 0 OR get_byte(sfzjh_enc, 0) = 1
    );

ALTER TABLE academic.buaa_students
    ADD CONSTRAINT chk_buaa_students_sfzjh_hash_format CHECK (
        sfzjh_hash IS NULL OR sfzjh_hash ~ '^[0-9a-f]{64}$'
    );

COMMENT ON COLUMN academic.buaa_students.sfzjh_enc IS '身份证件号（AES-GCM envelope v1 加密存储）';
COMMENT ON COLUMN academic.buaa_students.sfzjh_hash IS '身份证件号 HMAC-SHA256 哈希（用于查找）';

COMMIT;
