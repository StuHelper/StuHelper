ALTER TABLE public.authorization_grants
    DROP CONSTRAINT IF EXISTS chk_authorization_grants_source;

ALTER TABLE public.authorization_grants
    DROP COLUMN IF EXISTS source;
