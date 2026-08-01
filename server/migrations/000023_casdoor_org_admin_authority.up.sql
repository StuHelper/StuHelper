ALTER TABLE public.authorization_grants
    ADD COLUMN source text NOT NULL DEFAULT 'manual';

-- From this migration onward every platform super-admin grant is a local
-- projection of Casdoor's organization-level isAdmin fact. Existing rows are
-- adopted by that projection and will be verified on the subject's next login,
-- refresh, or privileged request.
UPDATE public.authorization_grants
SET source = 'casdoor_org_admin'
WHERE role = 'super_admin';

ALTER TABLE public.authorization_grants
    ADD CONSTRAINT chk_authorization_grants_source
        CHECK (
            (role = 'super_admin' AND source = 'casdoor_org_admin')
            OR
            (role <> 'super_admin' AND source = 'manual')
        );

COMMENT ON COLUMN public.authorization_grants.source IS
    'Authority source: Casdoor organization isAdmin for super_admin; manual for scoped business roles';
