DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.phone_binding_operations LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.phone_verification_credentials LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.phone_number_claims LIMIT 1)
       OR EXISTS (
            SELECT 1
            FROM public.users
            WHERE phone_projection_state NOT IN ('absent', 'legacy_unreconciled')
       )
    THEN
        RAISE EXCEPTION
            'refusing destructive rollback: phone verification domain contains new business facts';
    END IF;
END $$;

DROP TABLE IF EXISTS public.phone_eligibility_event_outbox;
DROP TABLE IF EXISTS public.phone_binding_outbox;
DROP TABLE IF EXISTS public.phone_eligibility_revisions;
DROP TABLE IF EXISTS public.phone_number_claims;
DROP TABLE IF EXISTS public.phone_verification_credentials;
DROP TABLE IF EXISTS public.phone_verification_attempts;
DROP TABLE IF EXISTS public.phone_binding_operations;

ALTER TABLE public.users
    DROP CONSTRAINT IF EXISTS chk_users_phone_projection_consistency,
    DROP CONSTRAINT IF EXISTS chk_users_phone_projection_revision,
    DROP CONSTRAINT IF EXISTS chk_users_phone_masked,
    DROP CONSTRAINT IF EXISTS chk_users_phone_projection_state,
    DROP COLUMN IF EXISTS phone_hmac_key_version,
    DROP COLUMN IF EXISTS phone_encryption_key_version,
    DROP COLUMN IF EXISTS phone_projection_synced_at,
    DROP COLUMN IF EXISTS phone_projection_revision,
    DROP COLUMN IF EXISTS phone_projection_state,
    DROP COLUMN IF EXISTS phone_masked;
