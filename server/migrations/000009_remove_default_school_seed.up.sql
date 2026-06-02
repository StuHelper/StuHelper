-- Remove the legacy non-MOE default school seed. StuHelper school primary
-- identifiers are Ministry of Education 10-digit school codes.

ALTER TABLE public.course_categories ALTER COLUMN school_id DROP DEFAULT;
ALTER TABLE public.courses ALTER COLUMN school_id DROP DEFAULT;
ALTER TABLE public.departments ALTER COLUMN school_id DROP DEFAULT;
ALTER TABLE public.teachers ALTER COLUMN school_id DROP DEFAULT;
ALTER TABLE public.rating_dimensions ALTER COLUMN school_id DROP DEFAULT;
ALTER TABLE public.terms ALTER COLUMN school_id DROP DEFAULT;

DELETE FROM public.school_configs
WHERE school_id = 1;

DELETE FROM public.review_reports
WHERE school_id = 1;

DELETE FROM public.reviews
WHERE school_id = 1;

DELETE FROM public.freshman_verification_applications
WHERE school_id = 1;

DELETE FROM public.user_verification_credentials
WHERE school_id = 1;

DELETE FROM public.group_admission_policies
WHERE school_id = 1;

DELETE FROM public.course_categories
WHERE school_id = 1;

DELETE FROM public.courses
WHERE school_id = 1;

DELETE FROM public.teachers
WHERE school_id = 1;

DELETE FROM public.rating_dimensions
WHERE school_id = 1;

DELETE FROM public.departments
WHERE school_id = 1;

DELETE FROM public.terms
WHERE school_id = 1;

DELETE FROM public.user_profiles
WHERE school_id = 1;

DELETE FROM public.schools
WHERE id = 1
  AND code = 'default';
