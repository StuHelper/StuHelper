-- Keep main-site student profiles consistent with admission credentials.
-- Admission verification credentials are the authoritative proof created by join flows;
-- user_profiles is the account-center projection used by stuhelper.com pages.

ALTER TABLE public.user_profiles DROP CONSTRAINT IF EXISTS chk_user_profiles_method;
ALTER TABLE public.user_profiles
  ADD CONSTRAINT chk_user_profiles_method
  CHECK (
    verification_method IS NULL
    OR verification_method::text = ANY (
      ARRAY[
        'ldap'::varchar,
        'manual'::varchar,
        'school_email_otp'::varchar,
        'school_sso'::varchar
      ]::text[]
    )
  );

WITH latest_active_credentials AS (
  SELECT DISTINCT ON (user_id)
    user_id,
    school_id,
    kind,
    subject_display,
    verified_at,
    CASE
      WHEN kind = 'school_email_otp' THEN 'school_email_otp'
      WHEN kind = 'school_sso' THEN 'school_sso'
      ELSE 'manual'
    END AS verification_method
  FROM public.user_verification_credentials
  WHERE revoked_at IS NULL
    AND (expires_at IS NULL OR expires_at > NOW())
  ORDER BY user_id, verified_at DESC, created_at DESC, id DESC
),
upserted_profiles AS (
  INSERT INTO public.user_profiles (
    user_id, school_id, student_ids, active_student_id, manual_form_data,
    verification_status, verification_method, rejection_reason, reviewed_at,
    consent_given_at, verified_at, created_at, updated_at
  )
  SELECT
    user_id,
    school_id,
    '[]'::jsonb,
    NULL,
    jsonb_build_object(
      'source', 'admission_credential',
      'credentialKind', kind,
      'subjectDisplay', subject_display
    ),
    'verified',
    verification_method,
    NULL,
    verified_at,
    verified_at,
    verified_at,
    NOW(),
    NOW()
  FROM latest_active_credentials
  ON CONFLICT (user_id)
  DO UPDATE SET
    school_id = EXCLUDED.school_id,
    verification_status = 'verified',
    verification_method = EXCLUDED.verification_method,
    rejection_reason = NULL,
    reviewed_at = EXCLUDED.reviewed_at,
    consent_given_at = COALESCE(public.user_profiles.consent_given_at, EXCLUDED.consent_given_at),
    verified_at = EXCLUDED.verified_at,
    manual_form_data = CASE
      WHEN public.user_profiles.manual_form_data IS NULL
        OR public.user_profiles.manual_form_data = '{}'::jsonb
        THEN EXCLUDED.manual_form_data
      ELSE public.user_profiles.manual_form_data
    END,
    updated_at = NOW()
  RETURNING user_id
),
profile_projection_jobs AS (
  INSERT INTO public.domain_event_outbox (
    stream, job_type, dedupe_key, payload, status, attempt_count,
    available_at, locked_at, last_error, created_at, updated_at
  )
  SELECT
    'iam_openfga_tuple_sync',
    'user_profile_projection',
    'user-profile-projection:' || user_id,
    jsonb_build_object('userID', user_id, 'approved', true),
    'pending',
    0,
    NOW(),
    NULL,
    NULL,
    NOW(),
    NOW()
  FROM upserted_profiles
  ON CONFLICT (stream, dedupe_key)
  DO UPDATE SET
    job_type = EXCLUDED.job_type,
    payload = EXCLUDED.payload,
    status = 'pending',
    attempt_count = 0,
    available_at = NOW(),
    locked_at = NULL,
    last_error = NULL,
    updated_at = NOW()
)
INSERT INTO public.domain_event_outbox (
  stream, job_type, dedupe_key, payload, status, attempt_count,
  available_at, locked_at, last_error, created_at, updated_at
)
SELECT
  'iam_casdoor_role_sync',
  'verified_student_role',
  'verified-student-role:' || user_id,
  jsonb_build_object('userID', user_id, 'role', 'verified_student', 'approved', true),
  'pending',
  0,
  NOW(),
  NULL,
  NULL,
  NOW(),
  NOW()
FROM upserted_profiles
ON CONFLICT (stream, dedupe_key)
DO UPDATE SET
  job_type = EXCLUDED.job_type,
  payload = EXCLUDED.payload,
  status = 'pending',
  attempt_count = 0,
  available_at = NOW(),
  locked_at = NULL,
  last_error = NULL,
  updated_at = NOW();
