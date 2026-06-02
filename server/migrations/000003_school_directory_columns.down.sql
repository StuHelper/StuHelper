UPDATE public.schools
SET authority = NULL,
    location = NULL,
    education_level = NULL,
    remark = NULL,
    source_name = NULL,
    source_as_of = NULL
WHERE code = '4111010006';

WITH resolved_buaa_school AS (
  SELECT id
  FROM public.schools
  WHERE code = '4111010006'
  LIMIT 1
)
UPDATE public.school_configs sc
SET verification_method = 'ldap',
    academic_db_table = 'academic.buaa_students',
    consent_text = '本功能将使用您提供的学号和密码通过学校统一身份认证系统验证您的学生身份。验证成功后，系统将读取您的姓名、院系、年级、手机号等学籍信息用于平台服务。您的密码不会被存储。',
    manual_form_fields = NULL,
    enabled = false,
    approval_policy = 'auto',
    updated_at = now()
FROM resolved_buaa_school
WHERE sc.school_id = resolved_buaa_school.id;

ALTER TABLE public.user_profiles DROP CONSTRAINT IF EXISTS chk_user_profiles_method;
ALTER TABLE public.user_profiles
  ADD CONSTRAINT chk_user_profiles_method
  CHECK (
    verification_method IS NULL
    OR verification_method::text = ANY (
      ARRAY['ldap'::varchar, 'manual'::varchar]::text[]
    )
  );

ALTER TABLE public.schools DROP COLUMN IF EXISTS source_as_of;
ALTER TABLE public.schools DROP COLUMN IF EXISTS source_name;
ALTER TABLE public.schools DROP COLUMN IF EXISTS remark;
ALTER TABLE public.schools DROP COLUMN IF EXISTS education_level;
ALTER TABLE public.schools DROP COLUMN IF EXISTS location;
ALTER TABLE public.schools DROP COLUMN IF EXISTS authority;
ALTER TABLE public.schools ALTER COLUMN code TYPE character varying(10);
