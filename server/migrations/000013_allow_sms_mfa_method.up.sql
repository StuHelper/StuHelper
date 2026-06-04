ALTER TABLE public.user_mfa_enrollment
    DROP CONSTRAINT IF EXISTS chk_user_mfa_methods_allowed;

ALTER TABLE public.user_mfa_enrollment
    ADD CONSTRAINT chk_user_mfa_methods_allowed
    CHECK (methods <@ ARRAY['totp'::text, 'webauthn'::text, 'sms'::text]);
