DROP TRIGGER IF EXISTS departments_teacher_public_stats_update ON public.departments;
DROP TRIGGER IF EXISTS departments_teacher_public_stats_delete_truncate ON public.departments;
DROP TRIGGER IF EXISTS teachers_public_stats_update ON public.teachers;
DROP TRIGGER IF EXISTS teachers_public_stats_insert_delete_truncate ON public.teachers;
DROP TRIGGER IF EXISTS reviews_teacher_public_stats_update ON public.reviews;
DROP TRIGGER IF EXISTS reviews_teacher_public_stats_insert_delete_truncate ON public.reviews;
DROP FUNCTION IF EXISTS public.enqueue_teacher_public_stats_refresh();
