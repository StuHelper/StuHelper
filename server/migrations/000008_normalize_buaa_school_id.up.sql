-- Normalize BUAA's school primary key to its Ministry of Education code.
-- Existing environments may still have the legacy five-digit key. New
-- installs already use the MOE code from the baseline seed.

DO $$
DECLARE
  legacy_school_id bigint := power(10::numeric, 4)::bigint + 6;
  buaa_school_id bigint := 4111010006;
BEGIN
  IF EXISTS (SELECT 1 FROM public.schools WHERE id = legacy_school_id) THEN
    UPDATE public.schools
    SET code = 'legacy-buaa'
    WHERE id = legacy_school_id
      AND code = buaa_school_id::text;
  END IF;

  INSERT INTO public.schools (
    id, code, name, authority, location, education_level, remark, source_name, source_as_of
  )
  SELECT
    buaa_school_id,
    buaa_school_id::text,
    name,
    authority,
    location,
    education_level,
    remark,
    source_name,
    source_as_of
  FROM public.schools
  WHERE id = legacy_school_id
  ON CONFLICT (id) DO UPDATE
  SET code = EXCLUDED.code,
      name = EXCLUDED.name,
      authority = EXCLUDED.authority,
      location = EXCLUDED.location,
      education_level = EXCLUDED.education_level,
      remark = EXCLUDED.remark,
      source_name = EXCLUDED.source_name,
      source_as_of = EXCLUDED.source_as_of;

  INSERT INTO public.schools (
    id, code, name, authority, location, education_level, remark, source_name, source_as_of
  )
  VALUES (
    buaa_school_id,
    buaa_school_id::text,
    '北京航空航天大学',
    '工业和信息化部',
    '北京市',
    '本科',
    NULL,
    '全国普通高等学校名单',
    DATE '2025-06-20'
  )
  ON CONFLICT (id) DO UPDATE
  SET code = EXCLUDED.code,
      name = EXCLUDED.name,
      authority = COALESCE(public.schools.authority, EXCLUDED.authority),
      location = COALESCE(public.schools.location, EXCLUDED.location),
      education_level = COALESCE(public.schools.education_level, EXCLUDED.education_level),
      source_name = COALESCE(public.schools.source_name, EXCLUDED.source_name),
      source_as_of = COALESCE(public.schools.source_as_of, EXCLUDED.source_as_of);

  DELETE FROM public.course_categories old_category
  WHERE old_category.school_id = legacy_school_id
    AND EXISTS (
      SELECT 1
      FROM public.course_categories new_category
      WHERE new_category.school_id = buaa_school_id
        AND new_category.name = old_category.name
    );

  UPDATE public.school_configs new_config
  SET school_name = old_config.school_name,
      verification_method = old_config.verification_method,
      ldap_config = old_config.ldap_config,
      academic_db_table = old_config.academic_db_table,
      consent_text = old_config.consent_text,
      manual_form_fields = old_config.manual_form_fields,
      enabled = old_config.enabled,
      approval_policy = old_config.approval_policy,
      updated_at = now()
  FROM public.school_configs old_config
  WHERE old_config.school_id = legacy_school_id
    AND new_config.school_id = buaa_school_id;

  DELETE FROM public.school_configs
  WHERE school_id = legacy_school_id
    AND EXISTS (
      SELECT 1
      FROM public.school_configs
      WHERE school_id = buaa_school_id
    );

  UPDATE public.freshman_verification_applications old_application
  SET status = 'rejected',
      review_reason = 'superseded by MOE school code migration',
      updated_at = now()
  WHERE old_application.school_id = legacy_school_id
    AND old_application.status = 'pending'
    AND EXISTS (
      SELECT 1
      FROM public.freshman_verification_applications new_application
      WHERE new_application.user_id = old_application.user_id
        AND new_application.school_id = buaa_school_id
        AND new_application.status = 'pending'
    );

  UPDATE public.user_profiles old_profile
  SET active_student_id = NULL,
      verification_status = 'unverified',
      verification_method = NULL,
      updated_at = now()
  WHERE old_profile.school_id = legacy_school_id
    AND old_profile.active_student_id IS NOT NULL
    AND EXISTS (
      SELECT 1
      FROM public.user_profiles new_profile
      WHERE new_profile.school_id = buaa_school_id
        AND new_profile.active_student_id = old_profile.active_student_id
    );

  UPDATE public.course_categories SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.courses SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.departments SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.freshman_verification_applications SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.group_admission_policies SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.rating_dimensions SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.review_reports SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.reviews SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.school_configs SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.teachers SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.terms SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.user_profiles SET school_id = buaa_school_id WHERE school_id = legacy_school_id;
  UPDATE public.user_verification_credentials SET school_id = buaa_school_id WHERE school_id = legacy_school_id;

  UPDATE public.system_configs
  SET value = (
    SELECT jsonb_agg(
      CASE
        WHEN elem.value = legacy_school_id::text THEN buaa_school_id::text
        ELSE elem.value
      END
    )::text
    FROM jsonb_array_elements_text(public.system_configs.value::jsonb) AS elem(value)
  )
  WHERE key = 'review_access_school_ids'
    AND value::jsonb ? legacy_school_id::text;

  DELETE FROM public.schools
  WHERE id = legacy_school_id;

  PERFORM setval(
    'public.schools_id_seq',
    GREATEST(
      buaa_school_id,
      COALESCE((SELECT max(id) FROM public.schools), buaa_school_id),
      (SELECT last_value FROM public.schools_id_seq)
    ),
    true
  );
END $$;
