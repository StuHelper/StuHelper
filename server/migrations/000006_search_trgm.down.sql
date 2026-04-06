DROP INDEX IF EXISTS idx_teachers_name_trgm;
DROP INDEX IF EXISTS idx_courses_code_trgm;
DROP INDEX IF EXISTS idx_courses_name_trgm;

DROP EXTENSION IF EXISTS pg_trgm;
