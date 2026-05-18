-- StuHelper consolidated schema rollback.
-- WARNING: destructive. Local reset only.

BEGIN;

DROP MATERIALIZED VIEW IF EXISTS public.mv_teacher_public_stats CASCADE;
DROP SCHEMA IF EXISTS academic CASCADE;

DROP TABLE IF EXISTS public.user_verification_credentials CASCADE;
DROP TABLE IF EXISTS public.user_qq_bindings CASCADE;
DROP TABLE IF EXISTS public.user_qq_binding_codes CASCADE;
DROP TABLE IF EXISTS public.user_profiles CASCADE;
DROP TABLE IF EXISTS public.user_mfa_recovery_codes CASCADE;
DROP TABLE IF EXISTS public.user_mfa_enrollment CASCADE;
DROP TABLE IF EXISTS public.user_identities CASCADE;
DROP TABLE IF EXISTS public.open_platform_audit_events CASCADE;
DROP TABLE IF EXISTS public.open_platform_user_consents CASCADE;
DROP TABLE IF EXISTS public.open_platform_approved_scopes CASCADE;
DROP TABLE IF EXISTS public.open_platform_scope_requests CASCADE;
DROP TABLE IF EXISTS public.open_platform_apps CASCADE;
DROP TABLE IF EXISTS public.users CASCADE;
DROP TABLE IF EXISTS public.terms CASCADE;
DROP TABLE IF EXISTS public.teachers CASCADE;
DROP TABLE IF EXISTS public.teacher_rating_stats CASCADE;
DROP TABLE IF EXISTS public.system_configs CASCADE;
DROP TABLE IF EXISTS public.storage_mounts CASCADE;
DROP TABLE IF EXISTS public.sensitive_words CASCADE;
DROP TABLE IF EXISTS public.schools CASCADE;
DROP TABLE IF EXISTS public.school_configs CASCADE;
DROP TABLE IF EXISTS public.review_votes CASCADE;
DROP TABLE IF EXISTS public.review_reports CASCADE;
DROP TABLE IF EXISTS public.review_replies CASCADE;
DROP TABLE IF EXISTS public.review_drafts CASCADE;
DROP TABLE IF EXISTS public.resource_versions CASCADE;
DROP TABLE IF EXISTS public.resource_tags CASCADE;
DROP TABLE IF EXISTS public.resource_items CASCADE;
DROP TABLE IF EXISTS public.resource_bindings CASCADE;
DROP TABLE IF EXISTS public.rating_dimensions CASCADE;
DROP TABLE IF EXISTS public.notifications CASCADE;
DROP TABLE IF EXISTS public.notification_preferences CASCADE;
DROP TABLE IF EXISTS public.reviews CASCADE;
DROP TABLE IF EXISTS public.group_admission_sessions CASCADE;
DROP TABLE IF EXISTS public.group_admission_policies CASCADE;
DROP TABLE IF EXISTS public.group_admission_failures CASCADE;
DROP TABLE IF EXISTS public.member_blacklist_entries CASCADE;
DROP TABLE IF EXISTS public.freshman_verification_materials CASCADE;
DROP TABLE IF EXISTS public.freshman_verification_applications CASCADE;
DROP TABLE IF EXISTS public.domain_event_outbox CASCADE;
DROP TABLE IF EXISTS public.departments CASCADE;
DROP TABLE IF EXISTS public.courses CASCADE;
DROP TABLE IF EXISTS public.course_rating_stats CASCADE;
DROP TABLE IF EXISTS public.course_favorites CASCADE;
DROP TABLE IF EXISTS public.course_categories CASCADE;
DROP TABLE IF EXISTS public.bot_service_credentials CASCADE;
DROP TABLE IF EXISTS public.audit_events CASCADE;
DROP TABLE IF EXISTS public.academic_terms CASCADE;
DROP TABLE IF EXISTS public.academic_teachers CASCADE;
DROP TABLE IF EXISTS public.academic_sources CASCADE;
DROP TABLE IF EXISTS public.academic_schedules CASCADE;
DROP TABLE IF EXISTS public.academic_offerings CASCADE;
DROP TABLE IF EXISTS public.academic_offering_teachers CASCADE;
DROP TABLE IF EXISTS public.academic_memberships CASCADE;
DROP TABLE IF EXISTS public.academic_import_jobs CASCADE;
DROP TABLE IF EXISTS public.academic_courses CASCADE;

DROP EXTENSION IF EXISTS pg_trgm CASCADE;

COMMIT;
