DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.student_manual_review_cases LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.student_manual_review_materials LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.student_manual_camera_handoffs LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.student_manual_review_events LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.school_verification_suggestions LIMIT 1)
    THEN
        RAISE EXCEPTION
            'refusing destructive rollback: student manual review domain contains applications, evidence or audit facts';
    END IF;
END $$;

DROP TABLE IF EXISTS public.school_verification_suggestions;
DROP TABLE IF EXISTS public.student_manual_review_events;
DROP INDEX IF EXISTS public.student_manual_camera_handoffs_active_case_uidx;
DROP TABLE IF EXISTS public.student_manual_camera_handoffs;
DROP TABLE IF EXISTS public.student_manual_review_materials;
DROP TABLE IF EXISTS public.student_manual_review_cases;
