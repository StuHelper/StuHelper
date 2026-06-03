-- Data projection is intentionally not reversed: user_profiles rows may now be
-- the user's current verified student state. Keep rollback limited to the
-- schema constraint, and preserve rows that use school_sso.

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
