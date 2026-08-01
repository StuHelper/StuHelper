CREATE TABLE public.authorization_authority_cutover (
    singleton_id smallint PRIMARY KEY DEFAULT 1,
    status text NOT NULL DEFAULT 'pending',
    source_digest text,
    imported_grant_count integer NOT NULL DEFAULT 0,
    completed_at timestamp with time zone,
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_authorization_authority_cutover_singleton
        CHECK (singleton_id = 1),
    CONSTRAINT chk_authorization_authority_cutover_status
        CHECK (status IN ('pending', 'completed')),
    CONSTRAINT chk_authorization_authority_cutover_count
        CHECK (imported_grant_count >= 0),
    CONSTRAINT chk_authorization_authority_cutover_completion
        CHECK (
            (
                status = 'pending'
                AND source_digest IS NULL
                AND imported_grant_count = 0
                AND completed_at IS NULL
            )
            OR (
                status = 'completed'
                AND source_digest ~ '^[0-9a-f]{64}$'
                AND completed_at IS NOT NULL
            )
        )
);

INSERT INTO public.authorization_authority_cutover (singleton_id)
VALUES (1);

COMMENT ON TABLE public.authorization_authority_cutover IS
    'Fail-closed deployment gate for the one-time Casdoor/OpenFGA to PostgreSQL authorization ledger cutover';
