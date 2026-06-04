UPDATE public.user_mfa_enrollment
SET methods = array_remove(methods, 'sms'::text),
    active = CASE
        WHEN cardinality(array_remove(methods, 'sms'::text)) = 0 THEN false
        ELSE active
    END,
    updated_at = now()
WHERE 'sms'::text = ANY(methods);

ALTER TABLE public.user_mfa_enrollment
    DROP CONSTRAINT IF EXISTS chk_user_mfa_methods_allowed;

ALTER TABLE public.user_mfa_enrollment
    ADD CONSTRAINT chk_user_mfa_methods_allowed
    CHECK (methods <@ ARRAY['totp'::text, 'webauthn'::text]);
