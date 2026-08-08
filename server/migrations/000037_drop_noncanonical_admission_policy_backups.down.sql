-- Intentionally irreversible.  These tables were noncanonical production
-- drift, not application state.  Recreating empty lookalikes during rollback
-- would fabricate backup evidence and make the database less trustworthy.

SELECT 1;
