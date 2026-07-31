-- Intentionally irreversible: rolling application code back must not silently
-- restore Casdoor as a StuHelper business-authorization authority. A rollback
-- requires an explicit, separately reviewed migration and incident procedure.
SELECT 1;
