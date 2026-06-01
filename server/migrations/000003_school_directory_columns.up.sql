-- Add Ministry of Education school directory metadata.
-- The directory itself is imported by infra/ops/import-school-directory.sh.
-- It is not an admission whitelist; school_configs.enabled controls enabled schools.

ALTER TABLE public.schools ALTER COLUMN code TYPE character varying(32);
ALTER TABLE public.schools ADD COLUMN IF NOT EXISTS authority character varying(100);
ALTER TABLE public.schools ADD COLUMN IF NOT EXISTS location character varying(100);
ALTER TABLE public.schools ADD COLUMN IF NOT EXISTS education_level character varying(50);
ALTER TABLE public.schools ADD COLUMN IF NOT EXISTS remark text;
ALTER TABLE public.schools ADD COLUMN IF NOT EXISTS source_name text;
ALTER TABLE public.schools ADD COLUMN IF NOT EXISTS source_as_of date;

UPDATE public.schools
SET code = '4111010006',
    name = '北京航空航天大学',
    authority = '工业和信息化部',
    location = '北京市',
    education_level = '本科',
    remark = NULL,
    source_name = '全国普通高等学校名单',
    source_as_of = DATE '2025-06-20'
WHERE id = 10006;

ALTER TABLE public.user_profiles DROP CONSTRAINT IF EXISTS chk_user_profiles_method;
ALTER TABLE public.user_profiles
  ADD CONSTRAINT chk_user_profiles_method
  CHECK (
    verification_method IS NULL
    OR verification_method::text = ANY (
      ARRAY['ldap'::varchar, 'manual'::varchar, 'school_email_otp'::varchar]::text[]
    )
  );

UPDATE public.school_configs
SET verification_method = 'manual',
    academic_db_table = 'academic.buaa_students',
    manual_form_fields = jsonb_set(
      jsonb_set(
        COALESCE(manual_form_fields, '{}'::jsonb),
        '{admission,emailDomains}',
        '["buaa.edu.cn"]'::jsonb,
        true
      ),
      '{admission,emailIdentityPolicy}',
      '{"type":"academic_student_email","studentIDEmailDomain":"buaa.edu.cn","requireStudentName":true}'::jsonb,
      true
    ),
    enabled = true,
    approval_policy = 'auto',
    updated_at = now()
WHERE school_id = 10006;

UPDATE public.school_configs
SET enabled = false,
    updated_at = now()
WHERE school_id <> 10006
  AND enabled = true;
