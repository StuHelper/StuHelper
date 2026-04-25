BEGIN;

CREATE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id);

ALTER TABLE review_replies DROP CONSTRAINT IF EXISTS chk_review_replies_content_length;
ALTER TABLE review_replies
    ADD CONSTRAINT chk_review_replies_content_length
    CHECK (LENGTH(TRIM(content)) >= 1 AND char_length(content) <= 5000);

COMMENT ON COLUMN academic.buaa_students.xh IS NULL;
COMMENT ON COLUMN academic.buaa_students.xm IS NULL;
COMMENT ON COLUMN academic.buaa_students.sfzjlxdm IS NULL;
COMMENT ON COLUMN academic.buaa_students.sfzjh_enc IS NULL;
COMMENT ON COLUMN academic.buaa_students.yxdm IS NULL;
COMMENT ON COLUMN academic.buaa_students.zydm IS NULL;
COMMENT ON COLUMN academic.buaa_students.bjdm IS NULL;
COMMENT ON COLUMN academic.buaa_students.xznj IS NULL;
COMMENT ON COLUMN academic.buaa_students.rxnj IS NULL;
COMMENT ON COLUMN academic.buaa_students.pyccdm IS NULL;
COMMENT ON COLUMN academic.buaa_students.xslbdm IS NULL;
COMMENT ON COLUMN academic.buaa_students.sjh IS NULL;
COMMENT ON COLUMN academic.buaa_students.dzxx IS NULL;
COMMENT ON COLUMN academic.buaa_students.xjztdm IS NULL;
COMMENT ON COLUMN academic.buaa_students.sfzx IS NULL;
COMMENT ON COLUMN academic.buaa_students.sfzj IS NULL;

COMMIT;
