-- Remove two ad hoc production backup tables that are not part of the
-- canonical StuHelper schema.  Their exact names, repository reference count,
-- row count and dependency graph were verified before this migration was
-- introduced.  Do not broaden this cleanup with patterns or dynamic SQL; an
-- unexpected dependency must fail the migration for review.

DROP TABLE IF EXISTS public.group_admission_policies_backup_20260608_743policy;
DROP TABLE IF EXISTS public.group_admission_policies_backup_20260608_targets;
