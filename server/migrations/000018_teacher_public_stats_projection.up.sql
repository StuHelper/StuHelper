CREATE OR REPLACE FUNCTION public.enqueue_teacher_public_stats_refresh()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO public.domain_event_outbox AS current_job (
        stream,
        job_type,
        dedupe_key,
        payload,
        status,
        attempt_count,
        available_at,
        locked_at,
        last_error,
        created_at,
        updated_at
    )
    VALUES (
        'review_projection',
        'teacher_public_stats_refresh',
        'teacher_public_stats',
        '{}'::jsonb,
        'pending',
        0,
        NOW(),
        NULL,
        NULL,
        NOW(),
        NOW()
    )
    ON CONFLICT (stream, dedupe_key)
    DO UPDATE SET
        job_type = EXCLUDED.job_type,
        payload = EXCLUDED.payload,
        status = CASE
            WHEN current_job.status = 'processing' THEN 'processing'
            ELSE 'pending'
        END,
        attempt_count = 0,
        available_at = NOW(),
        locked_at = CASE
            WHEN current_job.status = 'processing' THEN current_job.locked_at
            ELSE NULL
        END,
        locked_revision = CASE
            WHEN current_job.status = 'processing' THEN current_job.locked_revision
            ELSE NULL
        END,
        revision = current_job.revision + 1,
        last_error = NULL,
        updated_at = NOW();

    RETURN NULL;
END;
$$;

REVOKE ALL ON FUNCTION public.enqueue_teacher_public_stats_refresh() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.enqueue_teacher_public_stats_refresh() TO stuhelper_app;

CREATE TRIGGER reviews_teacher_public_stats_insert_delete_truncate
AFTER INSERT OR DELETE OR TRUNCATE ON public.reviews
FOR EACH STATEMENT
EXECUTE FUNCTION public.enqueue_teacher_public_stats_refresh();

CREATE TRIGGER reviews_teacher_public_stats_update
AFTER UPDATE OF teacher_id, course_id, avg_rating, status ON public.reviews
FOR EACH STATEMENT
EXECUTE FUNCTION public.enqueue_teacher_public_stats_refresh();

CREATE TRIGGER teachers_public_stats_insert_delete_truncate
AFTER INSERT OR DELETE OR TRUNCATE ON public.teachers
FOR EACH STATEMENT
EXECUTE FUNCTION public.enqueue_teacher_public_stats_refresh();

CREATE TRIGGER teachers_public_stats_update
AFTER UPDATE OF name, department_id ON public.teachers
FOR EACH STATEMENT
EXECUTE FUNCTION public.enqueue_teacher_public_stats_refresh();

CREATE TRIGGER departments_teacher_public_stats_delete_truncate
AFTER DELETE OR TRUNCATE ON public.departments
FOR EACH STATEMENT
EXECUTE FUNCTION public.enqueue_teacher_public_stats_refresh();

CREATE TRIGGER departments_teacher_public_stats_update
AFTER UPDATE OF name ON public.departments
FOR EACH STATEMENT
EXECUTE FUNCTION public.enqueue_teacher_public_stats_refresh();
