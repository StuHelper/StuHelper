BEGIN;

DROP INDEX IF EXISTS idx_users_external_id;

ALTER TABLE review_replies DROP CONSTRAINT IF EXISTS chk_review_replies_content_length;
ALTER TABLE review_replies
    ADD CONSTRAINT chk_review_replies_content_length
    CHECK (char_length(btrim(content)) >= 1 AND char_length(content) <= 5000);

COMMENT ON COLUMN academic.buaa_students.xh IS '学号 (student number)';
COMMENT ON COLUMN academic.buaa_students.xm IS '姓名 (full name)';
COMMENT ON COLUMN academic.buaa_students.sfzjlxdm IS '身份证件类型代码 (ID document type code)';
COMMENT ON COLUMN academic.buaa_students.sfzjh_enc IS '身份证件号（加密存储）(ID document number, encrypted)';
COMMENT ON COLUMN academic.buaa_students.yxdm IS '院系代码 (department code)';
COMMENT ON COLUMN academic.buaa_students.zydm IS '专业代码 (major code)';
COMMENT ON COLUMN academic.buaa_students.bjdm IS '班级代码 (class code)';
COMMENT ON COLUMN academic.buaa_students.xznj IS '学制年限/学制年级 (program duration / grade system code)';
COMMENT ON COLUMN academic.buaa_students.rxnj IS '入学年级 (enrollment grade)';
COMMENT ON COLUMN academic.buaa_students.pyccdm IS '培养层次代码 (education level code)';
COMMENT ON COLUMN academic.buaa_students.xslbdm IS '学生类别代码 (student category code)';
COMMENT ON COLUMN academic.buaa_students.sjh IS '手机号 (mobile phone number)';
COMMENT ON COLUMN academic.buaa_students.dzxx IS '电子邮箱 (email address)';
COMMENT ON COLUMN academic.buaa_students.xjztdm IS '学籍状态代码 (student status code)';
COMMENT ON COLUMN academic.buaa_students.sfzx IS '是否在校 (on-campus status flag)';
COMMENT ON COLUMN academic.buaa_students.sfzj IS '是否在籍 (registered status flag)';

COMMIT;
