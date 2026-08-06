-- Intentionally irreversible. The product owner explicitly rejected backup,
-- migration, retention and compatibility for legacy student-verification
-- facts. Runtime rollback is limited to application versions that remain
-- compatible with the target schema and keep every retired write path disabled;
-- no down migration may fabricate or restore deleted identity evidence.

DO $$
BEGIN
    RAISE EXCEPTION 'refusing destructive rollback: purged legacy student-verification data cannot be restored';
END
$$;
