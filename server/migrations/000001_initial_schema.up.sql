--
-- PostgreSQL database dump
--


-- Dumped from database version 18.3
-- Dumped by pg_dump version 18.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: academic; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA academic;


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: buaa_students; Type: TABLE; Schema: academic; Owner: -
--

CREATE TABLE academic.buaa_students (
    xh character varying(50) NOT NULL,
    xm character varying(100),
    sfzjlxdm character varying(10),
    sfzjh_enc bytea,
    yxdm character varying(20),
    zydm character varying(20),
    bjdm character varying(20),
    xznj character varying(10),
    rxnj character varying(10),
    pyccdm character varying(10),
    xslbdm character varying(10),
    sjh character varying(20),
    dzxx character varying(100),
    xjztdm character varying(10),
    sfzx character varying(10),
    sfzj character varying(10),
    synced_at timestamp with time zone DEFAULT now() NOT NULL,
    sfzjh_hash character varying(64),
    CONSTRAINT chk_buaa_students_sfzjh_envelope_v1 CHECK (((sfzjh_enc IS NULL) OR (octet_length(sfzjh_enc) = 0) OR (get_byte(sfzjh_enc, 0) = 1))),
    CONSTRAINT chk_buaa_students_sfzjh_hash_format CHECK (((sfzjh_hash IS NULL) OR ((sfzjh_hash)::text ~ '^[0-9a-f]{64}$'::text))),
    CONSTRAINT chk_buaa_students_sfzjh_secure_pair CHECK ((((sfzjh_enc IS NULL) AND (sfzjh_hash IS NULL)) OR ((sfzjh_enc IS NOT NULL) AND (sfzjh_hash IS NOT NULL))))
);


--
-- Name: COLUMN buaa_students.xh; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.xh IS '学号 (student number)';


--
-- Name: COLUMN buaa_students.xm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.xm IS '姓名 (full name)';


--
-- Name: COLUMN buaa_students.sfzjlxdm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.sfzjlxdm IS '身份证件类型代码 (ID document type code)';


--
-- Name: COLUMN buaa_students.sfzjh_enc; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.sfzjh_enc IS '身份证件号（AES-GCM envelope v1 加密存储）';


--
-- Name: COLUMN buaa_students.yxdm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.yxdm IS '院系代码 (department code)';


--
-- Name: COLUMN buaa_students.zydm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.zydm IS '专业代码 (major code)';


--
-- Name: COLUMN buaa_students.bjdm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.bjdm IS '班级代码 (class code)';


--
-- Name: COLUMN buaa_students.xznj; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.xznj IS '学制年限/学制年级 (program duration / grade system code)';


--
-- Name: COLUMN buaa_students.rxnj; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.rxnj IS '入学年级 (enrollment grade)';


--
-- Name: COLUMN buaa_students.pyccdm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.pyccdm IS '培养层次代码 (education level code)';


--
-- Name: COLUMN buaa_students.xslbdm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.xslbdm IS '学生类别代码 (student category code)';


--
-- Name: COLUMN buaa_students.sjh; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.sjh IS '手机号 (mobile phone number)';


--
-- Name: COLUMN buaa_students.dzxx; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.dzxx IS '电子邮箱 (email address)';


--
-- Name: COLUMN buaa_students.xjztdm; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.xjztdm IS '学籍状态代码 (student status code)';


--
-- Name: COLUMN buaa_students.sfzx; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.sfzx IS '是否在校 (on-campus status flag)';


--
-- Name: COLUMN buaa_students.sfzj; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.sfzj IS '是否在籍 (registered status flag)';


--
-- Name: COLUMN buaa_students.sfzjh_hash; Type: COMMENT; Schema: academic; Owner: -
--

COMMENT ON COLUMN academic.buaa_students.sfzjh_hash IS '身份证件号 HMAC-SHA256 哈希（用于查找）';


--
-- Name: academic_courses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_courses (
    id bigint NOT NULL,
    source_id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    external_id text NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    department_code text,
    department_name text,
    credit numeric(4,1),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: academic_courses_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_courses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_courses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_courses_id_seq OWNED BY public.academic_courses.id;


--
-- Name: academic_import_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_import_jobs (
    id bigint NOT NULL,
    source_id bigint NOT NULL,
    provider text NOT NULL,
    trigger_mode text DEFAULT 'manual'::text NOT NULL,
    status text NOT NULL,
    requested_by_user_id text,
    stats jsonb DEFAULT '{}'::jsonb NOT NULL,
    error_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT academic_import_jobs_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'running'::text, 'succeeded'::text, 'failed'::text])))
);


--
-- Name: academic_import_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_import_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_import_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_import_jobs_id_seq OWNED BY public.academic_import_jobs.id;


--
-- Name: academic_memberships; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_memberships (
    id bigint NOT NULL,
    source_id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    offering_id bigint NOT NULL,
    external_user_id text NOT NULL,
    student_id text,
    role text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT academic_memberships_role_check CHECK ((role = ANY (ARRAY['student'::text, 'teacher'::text, 'assistant'::text])))
);


--
-- Name: academic_memberships_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_memberships_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_memberships_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_memberships_id_seq OWNED BY public.academic_memberships.id;


--
-- Name: academic_offering_teachers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_offering_teachers (
    offering_id bigint NOT NULL,
    teacher_id bigint NOT NULL
);


--
-- Name: academic_offerings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_offerings (
    id bigint NOT NULL,
    source_id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    external_id text NOT NULL,
    term_id bigint NOT NULL,
    course_id bigint NOT NULL,
    section_code text NOT NULL,
    school_name text,
    department_name text,
    campus text,
    enrollment_limit integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: academic_offerings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_offerings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_offerings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_offerings_id_seq OWNED BY public.academic_offerings.id;


--
-- Name: academic_schedules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_schedules (
    id bigint NOT NULL,
    offering_id bigint NOT NULL,
    weekday smallint NOT NULL,
    start_period smallint NOT NULL,
    end_period smallint NOT NULL,
    location text NOT NULL,
    building text,
    weeks_text text NOT NULL,
    CONSTRAINT academic_schedules_check CHECK ((end_period >= start_period)),
    CONSTRAINT academic_schedules_start_period_check CHECK ((start_period >= 1)),
    CONSTRAINT academic_schedules_weekday_check CHECK (((weekday >= 1) AND (weekday <= 7)))
);


--
-- Name: academic_schedules_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_schedules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_schedules_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_schedules_id_seq OWNED BY public.academic_schedules.id;


--
-- Name: academic_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_sources (
    id bigint NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    provider text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: academic_sources_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_sources_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_sources_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_sources_id_seq OWNED BY public.academic_sources.id;


--
-- Name: academic_teachers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_teachers (
    id bigint NOT NULL,
    source_id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    external_id text NOT NULL,
    name text NOT NULL,
    department_name text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: academic_teachers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_teachers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_teachers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_teachers_id_seq OWNED BY public.academic_teachers.id;


--
-- Name: academic_terms; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_terms (
    id bigint NOT NULL,
    source_id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    external_id text NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    start_date date,
    end_date date,
    is_current boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: academic_terms_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_terms_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_terms_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_terms_id_seq OWNED BY public.academic_terms.id;


--
-- Name: audit_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_events (
    id character varying(36) NOT NULL,
    category text DEFAULT 'audit'::text NOT NULL,
    event_type text NOT NULL,
    actor_type text DEFAULT 'user'::text NOT NULL,
    actor_user_id text DEFAULT ''::text NOT NULL,
    actor_username text DEFAULT ''::text NOT NULL,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id text DEFAULT ''::text NOT NULL,
    scope_school_id text,
    before_data jsonb,
    after_data jsonb,
    result text DEFAULT 'success'::text NOT NULL,
    reason text,
    trace_id character varying(32),
    request_id character varying(128),
    ip_address character varying(45),
    user_agent character varying(512),
    details jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_audit_events_actor_type CHECK ((actor_type = ANY (ARRAY['user'::text, 'admin'::text, 'system'::text, 'service_account'::text]))),
    CONSTRAINT chk_audit_events_category CHECK ((category = ANY (ARRAY['audit'::text, 'admin_operation'::text, 'domain_event'::text]))),
    CONSTRAINT chk_audit_events_result CHECK ((result = ANY (ARRAY['success'::text, 'failure'::text, 'pending'::text])))
);


--
-- Name: bot_service_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bot_service_credentials (
    id bigint NOT NULL,
    name text NOT NULL,
    token_hash text NOT NULL,
    audience text[] NOT NULL,
    scopes text[] NOT NULL,
    expires_at timestamp with time zone,
    rotated_at timestamp with time zone,
    revoked_at timestamp with time zone,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_bot_service_credentials_audience_nonempty CHECK ((cardinality(audience) > 0)),
    CONSTRAINT chk_bot_service_credentials_name_nonempty CHECK ((length(TRIM(BOTH FROM name)) > 0)),
    CONSTRAINT chk_bot_service_credentials_scopes_nonempty CHECK ((cardinality(scopes) > 0)),
    CONSTRAINT chk_bot_service_credentials_token_hash CHECK ((token_hash ~ '^[0-9a-f]{64}$'::text))
);


--
-- Name: bot_service_credentials_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.bot_service_credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: bot_service_credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.bot_service_credentials_id_seq OWNED BY public.bot_service_credentials.id;


--
-- Name: course_categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.course_categories (
    id bigint NOT NULL,
    school_id bigint NOT NULL,
    name character varying(50) NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: course_categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.course_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: course_categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.course_categories_id_seq OWNED BY public.course_categories.id;


--
-- Name: course_favorites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.course_favorites (
    id character varying(36) NOT NULL,
    user_hash character varying(64) NOT NULL,
    course_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: course_rating_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.course_rating_stats (
    id character varying(36) NOT NULL,
    course_id bigint NOT NULL,
    term_id character varying(20),
    dimension_key character varying(50) NOT NULL,
    avg_rating numeric(3,2) DEFAULT 0 NOT NULL,
    rating_count integer DEFAULT 0 NOT NULL,
    rating_dist jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT course_rating_stats_avg_rating_check CHECK (((avg_rating >= (0)::numeric) AND (avg_rating <= (5)::numeric))),
    CONSTRAINT course_rating_stats_rating_count_check CHECK ((rating_count >= 0))
);


--
-- Name: courses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.courses (
    id bigint NOT NULL,
    school_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    code character varying(50),
    department_id bigint,
    credits numeric(4,1),
    category character varying(50) DEFAULT ''::character varying NOT NULL,
    description text,
    review_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT courses_review_count_check CHECK ((review_count >= 0))
);


--
-- Name: courses_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.courses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: courses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.courses_id_seq OWNED BY public.courses.id;


--
-- Name: departments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.departments (
    id bigint NOT NULL,
    school_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    short_name character varying(50),
    category character varying(50) DEFAULT ''::character varying NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: departments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.departments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: departments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.departments_id_seq OWNED BY public.departments.id;


--
-- Name: domain_event_outbox; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.domain_event_outbox (
    id bigint NOT NULL,
    stream text NOT NULL,
    job_type text NOT NULL,
    dedupe_key text NOT NULL,
    payload jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempt_count integer DEFAULT 0 NOT NULL,
    available_at timestamp with time zone DEFAULT now() NOT NULL,
    locked_at timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_domain_event_outbox_status CHECK ((status = ANY (ARRAY['pending'::text, 'processing'::text, 'completed'::text, 'failed'::text, 'dead_letter'::text]))),
    CONSTRAINT chk_domain_event_outbox_stream CHECK ((stream <> ''::text))
);


--
-- Name: domain_event_outbox_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.domain_event_outbox_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: domain_event_outbox_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.domain_event_outbox_id_seq OWNED BY public.domain_event_outbox.id;


--
-- Name: freshman_verification_applications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.freshman_verification_applications (
    id character varying(36) NOT NULL,
    user_id bigint NOT NULL,
    school_id bigint NOT NULL,
    admission_session_id character varying(36),
    status text DEFAULT 'pending'::text NOT NULL,
    applicant_name text NOT NULL,
    applicant_name_masked text CONSTRAINT freshman_verification_applicatio_applicant_name_masked_not_null NOT NULL,
    department_or_major text,
    material_type text NOT NULL,
    provisional_expires_at timestamp with time zone,
    review_reason text,
    reviewed_by_user_id bigint,
    reviewed_by_operator_qq_id text,
    reviewed_at timestamp with time zone,
    forwarded_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_freshman_verification_applications_material_type CHECK ((material_type = ANY (ARRAY['admission_notice'::text, 'admission_certificate'::text]))),
    CONSTRAINT chk_freshman_verification_applications_status CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text])))
);


--
-- Name: freshman_verification_materials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.freshman_verification_materials (
    id character varying(36) NOT NULL,
    application_id character varying(36) NOT NULL,
    object_key text NOT NULL,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL,
    sha256 character varying(64) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_freshman_verification_materials_content_type CHECK ((content_type = ANY (ARRAY['image/jpeg'::text, 'image/png'::text, 'image/webp'::text]))),
    CONSTRAINT chk_freshman_verification_materials_sha256 CHECK ((char_length((sha256)::text) = 64)),
    CONSTRAINT chk_freshman_verification_materials_size CHECK ((size_bytes > 0))
);


--
-- Name: group_admission_failures; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_admission_failures (
    platform text NOT NULL,
    guild_id text NOT NULL,
    qq_id text NOT NULL,
    failure_count integer DEFAULT 0 NOT NULL,
    last_failure_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_group_admission_failures_count CHECK ((failure_count >= 0))
);


--
-- Name: member_blacklist_entries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.member_blacklist_entries (
    id character varying(36) NOT NULL,
    platform text NOT NULL,
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    scope_type text NOT NULL,
    guild_id text,
    source text NOT NULL,
    reason_code text NOT NULL,
    reason_text text NOT NULL,
    created_by_type text NOT NULL,
    created_by_id text NOT NULL,
    created_from text NOT NULL,
    expires_at timestamp with time zone,
    released_at timestamp with time zone,
    released_by_type text,
    released_by_id text,
    release_reason_code text,
    release_reason text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_member_blacklist_entries_actor_type CHECK ((created_by_type = ANY (ARRAY['system'::text, 'admin_user'::text, 'qq_operator'::text, 'service_account'::text]))),
    CONSTRAINT chk_member_blacklist_entries_created_from CHECK ((created_from = ANY (ARRAY['admission_worker'::text, 'qq_command'::text, 'koishi_console'::text, 'admin_console'::text, 'moderation_review'::text, 'migration_script'::text]))),
    CONSTRAINT chk_member_blacklist_entries_reason_code CHECK ((reason_code = ANY (ARRAY['admission_timeout_limit'::text, 'manual_blacklist'::text, 'manual_kick_blacklist'::text, 'violation_review_blacklist'::text, 'legacy_koishi_blacklist'::text, 'legacy_admission_blacklist'::text]))),
    CONSTRAINT chk_member_blacklist_entries_release_actor_type CHECK (((released_by_type IS NULL) OR (released_by_type = ANY (ARRAY['system'::text, 'admin_user'::text, 'qq_operator'::text, 'service_account'::text])))),
    CONSTRAINT chk_member_blacklist_entries_release_reason_code CHECK (((release_reason_code IS NULL) OR (release_reason_code = ANY (ARRAY['manual_pardon'::text, 'release_only'::text, 'policy_expired_auto'::text, 'admission_appeal_passed'::text, 'migration_inverse'::text])))),
    CONSTRAINT chk_member_blacklist_entries_scope CHECK ((((scope_type = 'global'::text) AND (guild_id IS NULL)) OR ((scope_type = 'guild'::text) AND (guild_id IS NOT NULL)))),
    CONSTRAINT chk_member_blacklist_entries_source CHECK ((source = ANY (ARRAY['admission_failure'::text, 'manual_admin'::text, 'kick_blacklist'::text, 'moderation_action'::text, 'migration_legacy_koishi'::text, 'migration_admission_failure'::text]))),
    CONSTRAINT chk_member_blacklist_entries_subject_type CHECK ((subject_type = ANY (ARRAY['qq_user'::text])))
);


--
-- Name: group_admission_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_admission_policies (
    id character varying(36) NOT NULL,
    platform text NOT NULL,
    guild_id text NOT NULL,
    school_id bigint NOT NULL,
    auto_approve_join boolean DEFAULT true NOT NULL,
    initial_mute_duration_seconds integer DEFAULT 2592000 NOT NULL,
    link_wait_seconds integer DEFAULT 3600 NOT NULL,
    submission_wait_seconds integer DEFAULT 86400 NOT NULL,
    manual_review_timeout_seconds integer DEFAULT 86400 NOT NULL,
    reminder_interval_seconds integer DEFAULT 900 NOT NULL,
    failed_join_limit integer DEFAULT 3 NOT NULL,
    blacklist_duration_seconds integer,
    freshman_channel_enabled boolean DEFAULT true NOT NULL,
    freshman_channel_closes_at timestamp with time zone NOT NULL,
    freshman_default_expires_at timestamp with time zone NOT NULL,
    forward_raw_material_to_qq boolean DEFAULT false NOT NULL,
    management_guild_ids text[] DEFAULT '{}'::text[] NOT NULL,
    max_material_bytes bigint DEFAULT 10485760 NOT NULL,
    max_extension_days integer DEFAULT 90 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_group_admission_policies_positive_limits CHECK (((failed_join_limit > 0) AND (max_material_bytes > 0) AND (max_extension_days > 0))),
    CONSTRAINT chk_group_admission_policies_positive_seconds CHECK (((initial_mute_duration_seconds > 0) AND (link_wait_seconds > 0) AND (submission_wait_seconds > 0) AND (manual_review_timeout_seconds > 0) AND (reminder_interval_seconds > 0)))
);


--
-- Name: group_admission_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_admission_sessions (
    id character varying(36) NOT NULL,
    platform text NOT NULL,
    bot_self_id text DEFAULT ''::text NOT NULL,
    guild_id text NOT NULL,
    channel_id text NOT NULL,
    qq_id text NOT NULL,
    user_id bigint,
    token_hash character varying(128) NOT NULL,
    auth_url text DEFAULT ''::text NOT NULL,
    token_expires_at timestamp with time zone NOT NULL,
    token_consumed_at timestamp with time zone,
    status text DEFAULT 'joined_muted'::text NOT NULL,
    link_wait_deadline_at timestamp with time zone NOT NULL,
    submission_wait_deadline_at timestamp with time zone NOT NULL,
    manual_review_deadline_at timestamp with time zone,
    initial_mute_until timestamp with time zone NOT NULL,
    last_reminded_at timestamp with time zone,
    next_reminder_at timestamp with time zone,
    verified_at timestamp with time zone,
    cancelled_at timestamp with time zone,
    last_bot_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_group_admission_sessions_status CHECK ((status = ANY (ARRAY['joined_muted'::text, 'linked'::text, 'material_submitted'::text, 'verified'::text, 'expired_kicked'::text, 'cancelled'::text])))
);


--
-- Name: reviews; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reviews (
    id character varying(36) NOT NULL,
    course_id bigint NOT NULL,
    school_id bigint NOT NULL,
    teacher_id bigint,
    term_id character varying(20) DEFAULT NULL::character varying,
    user_hash character varying(64) NOT NULL,
    title character varying(200) NOT NULL,
    content text NOT NULL,
    grade character varying(5),
    ratings jsonb DEFAULT '{}'::jsonb NOT NULL,
    avg_rating numeric(3,2) DEFAULT 0 NOT NULL,
    like_count integer DEFAULT 0 NOT NULL,
    dislike_count integer DEFAULT 0 NOT NULL,
    reply_count integer DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'published'::character varying NOT NULL,
    moderation_reason text,
    original_content text,
    original_title character varying(200),
    moderated_by character varying(255),
    moderated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    content_flag character varying(20),
    content_flag_cleared_at timestamp with time zone,
    content_flag_cleared_by character varying(255),
    CONSTRAINT chk_reviews_content_flag CHECK (((content_flag IS NULL) OR ((content_flag)::text = ANY ((ARRAY['warn'::character varying, 'review'::character varying, 'cleared'::character varying])::text[])))),
    CONSTRAINT chk_reviews_title_length CHECK (((char_length(btrim((title)::text)) >= 1) AND (char_length((title)::text) <= 200))),
    CONSTRAINT reviews_avg_rating_check CHECK (((avg_rating >= (0)::numeric) AND (avg_rating <= (5)::numeric))),
    CONSTRAINT reviews_dislike_count_check CHECK ((dislike_count >= 0)),
    CONSTRAINT reviews_like_count_check CHECK ((like_count >= 0)),
    CONSTRAINT reviews_reply_count_check CHECK ((reply_count >= 0)),
    CONSTRAINT reviews_status_check CHECK (((status)::text = ANY ((ARRAY['published'::character varying, 'hidden'::character varying, 'deleted'::character varying, 'pending_review'::character varying])::text[])))
);


--
-- Name: COLUMN reviews.content_flag; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.reviews.content_flag IS 'null=无标记, warn=警告级敏感词, review=审核级敏感词, cleared=人工复核通过';


--
-- Name: teachers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teachers (
    id bigint NOT NULL,
    school_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    department_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: mv_teacher_public_stats; Type: MATERIALIZED VIEW; Schema: public; Owner: -
--

CREATE MATERIALIZED VIEW public.mv_teacher_public_stats AS
 SELECT t.id AS teacher_id,
    t.name AS teacher_name,
    COALESCE(d.name, ''::character varying) AS department_name,
    t.department_id,
    avg(r.avg_rating) AS avg_rating,
    count(DISTINCT r.course_id) AS course_count,
    count(r.id) AS review_count
   FROM ((public.teachers t
     LEFT JOIN public.departments d ON ((d.id = t.department_id)))
     LEFT JOIN public.reviews r ON (((r.teacher_id = t.id) AND ((r.status)::text = 'published'::text))))
  GROUP BY t.id, t.name, d.name, t.department_id
  WITH DATA;


--
-- Name: notification_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_preferences (
    user_id bigint NOT NULL,
    type character varying(50) NOT NULL,
    enabled boolean DEFAULT true NOT NULL
);


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id character varying(36) NOT NULL,
    type character varying(50) NOT NULL,
    title character varying(200) NOT NULL,
    is_read boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id bigint NOT NULL,
    body text DEFAULT ''::text NOT NULL,
    source_module character varying(50) DEFAULT ''::character varying NOT NULL,
    source_id character varying(100) DEFAULT ''::character varying NOT NULL,
    source_url text,
    source_course_id bigint,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: rating_dimensions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rating_dimensions (
    id character varying(36) NOT NULL,
    school_id bigint NOT NULL,
    key character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    sort_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: resource_bindings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resource_bindings (
    resource_id bigint NOT NULL,
    binding_type text NOT NULL,
    binding_value text NOT NULL
);


--
-- Name: resource_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resource_items (
    id bigint NOT NULL,
    owner_user_id text NOT NULL,
    title text NOT NULL,
    description text,
    category text,
    visibility text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT resource_items_visibility_check CHECK ((visibility = ANY (ARRAY['public'::text, 'private'::text])))
);


--
-- Name: resource_items_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.resource_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: resource_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.resource_items_id_seq OWNED BY public.resource_items.id;


--
-- Name: resource_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resource_tags (
    resource_id bigint NOT NULL,
    tag text NOT NULL
);


--
-- Name: resource_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resource_versions (
    id bigint NOT NULL,
    resource_id bigint NOT NULL,
    version_no integer NOT NULL,
    mount_id bigint NOT NULL,
    object_key text NOT NULL,
    filename text NOT NULL,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: resource_versions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.resource_versions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: resource_versions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.resource_versions_id_seq OWNED BY public.resource_versions.id;


--
-- Name: review_drafts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.review_drafts (
    id character varying(36) NOT NULL,
    user_hash character varying(64) NOT NULL,
    course_id bigint,
    teacher_id bigint,
    term_id character varying(20),
    title character varying(200),
    content text,
    grade character varying(5),
    ratings jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: review_replies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.review_replies (
    id character varying(36) NOT NULL,
    review_id character varying(36) NOT NULL,
    parent_id character varying(36),
    user_hash character varying(64) NOT NULL,
    content text NOT NULL,
    like_count integer DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'published'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_review_replies_content_length CHECK (((char_length(btrim(content)) >= 1) AND (char_length(content) <= 5000))),
    CONSTRAINT review_replies_like_count_check CHECK ((like_count >= 0)),
    CONSTRAINT review_replies_status_check CHECK (((status)::text = ANY ((ARRAY['published'::character varying, 'hidden'::character varying, 'deleted'::character varying, 'pending_review'::character varying])::text[])))
);


--
-- Name: review_reports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.review_reports (
    id character varying(36) NOT NULL,
    review_id character varying(36) NOT NULL,
    school_id bigint NOT NULL,
    reporter_hash character varying(64) NOT NULL,
    reason character varying(50) NOT NULL,
    description text,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    resolved_by character varying(255),
    resolved_at timestamp with time zone,
    resolution_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_review_reports_status CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'resolved'::character varying, 'rejected'::character varying])::text[])))
);


--
-- Name: review_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.review_votes (
    id character varying(36) NOT NULL,
    review_id character varying(36) NOT NULL,
    user_hash character varying(64) NOT NULL,
    vote_type character varying(10) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_review_votes_type CHECK (((vote_type)::text = ANY ((ARRAY['like'::character varying, 'dislike'::character varying])::text[])))
);


--
-- Name: school_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.school_configs (
    school_name character varying(100) NOT NULL,
    verification_method character varying(20) DEFAULT 'manual'::character varying NOT NULL,
    ldap_config bytea,
    academic_db_table character varying(100),
    consent_text text,
    manual_form_fields jsonb,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    approval_policy character varying(20) DEFAULT 'auto'::character varying NOT NULL,
    school_id bigint CONSTRAINT school_configs_school_id_not_null1 NOT NULL,
    CONSTRAINT chk_approval_policy CHECK (((approval_policy)::text = ANY ((ARRAY['auto'::character varying, 'manual'::character varying])::text[]))),
    CONSTRAINT chk_school_configs_method CHECK (((verification_method)::text = ANY ((ARRAY['ldap'::character varying, 'manual'::character varying])::text[])))
);


--
-- Name: schools; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schools (
    id bigint NOT NULL,
    code character varying(10) NOT NULL,
    name character varying(100) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: schools_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.schools_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: schools_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.schools_id_seq OWNED BY public.schools.id;


--
-- Name: sensitive_words; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sensitive_words (
    id character varying(36) NOT NULL,
    word character varying(100) NOT NULL,
    category character varying(50) DEFAULT 'general'::character varying NOT NULL,
    level character varying(20) DEFAULT 'block'::character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_sensitive_words_level CHECK (((level)::text = ANY ((ARRAY['block'::character varying, 'warn'::character varying, 'review'::character varying])::text[])))
);


--
-- Name: storage_mounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.storage_mounts (
    id bigint NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    driver text NOT NULL,
    bucket text,
    base_path text DEFAULT ''::text NOT NULL,
    credential_source text DEFAULT 'runtime_default_object_storage'::text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    last_health_status text,
    last_health_error text,
    last_health_checked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: storage_mounts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.storage_mounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: storage_mounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.storage_mounts_id_seq OWNED BY public.storage_mounts.id;


--
-- Name: system_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_configs (
    key character varying(100) NOT NULL,
    value text NOT NULL,
    description character varying(500),
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: teacher_rating_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teacher_rating_stats (
    id character varying(36) NOT NULL,
    teacher_id bigint NOT NULL,
    term_id character varying(20),
    dimension_key character varying(50) NOT NULL,
    avg_rating numeric(3,2) DEFAULT 0 NOT NULL,
    rating_count integer DEFAULT 0 NOT NULL,
    rating_dist jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT teacher_rating_stats_avg_rating_check CHECK (((avg_rating >= (0)::numeric) AND (avg_rating <= (5)::numeric))),
    CONSTRAINT teacher_rating_stats_rating_count_check CHECK ((rating_count >= 0))
);


--
-- Name: teachers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.teachers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: teachers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.teachers_id_seq OWNED BY public.teachers.id;


--
-- Name: terms; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.terms (
    id character varying(20) NOT NULL,
    school_id bigint NOT NULL,
    name character varying(100) NOT NULL,
    is_current boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_identities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_identities (
    user_id bigint NOT NULL,
    doc_type character varying(20) NOT NULL,
    doc_number_enc bytea NOT NULL,
    person_uid character varying(64) NOT NULL,
    real_name character varying(100) NOT NULL,
    verified boolean DEFAULT false NOT NULL,
    verify_method character varying(20),
    reviewed_at timestamp with time zone,
    verified_at timestamp with time zone,
    doc_photo_front text,
    doc_photo_back text,
    doc_photo_selfie text,
    rejection_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_user_identities_doc_type CHECK (((doc_type)::text = ANY ((ARRAY['MAINLAND_ID'::character varying, 'HK_MACAU'::character varying, 'TW'::character varying, 'PASSPORT'::character varying])::text[]))),
    CONSTRAINT chk_user_identities_verify_method CHECK (((verify_method IS NULL) OR ((verify_method)::text = ANY ((ARRAY['academic_db_match'::character varying, 'tencent_cloud'::character varying, 'manual'::character varying])::text[]))))
);


--
-- Name: user_mfa_enrollment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_mfa_enrollment (
    user_id bigint NOT NULL,
    active boolean DEFAULT false NOT NULL,
    methods text[] DEFAULT '{}'::text[] NOT NULL,
    recovery_codes_issued_at timestamp with time zone,
    reset_required boolean DEFAULT false NOT NULL,
    last_enrolled_at timestamp with time zone,
    last_disabled_at timestamp with time zone,
    last_reset_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_user_mfa_active_requires_method CHECK (((active = false) OR (cardinality(methods) > 0))),
    CONSTRAINT chk_user_mfa_methods_allowed CHECK ((methods <@ ARRAY['totp'::text, 'webauthn'::text, 'sms'::text]))
);


--
-- Name: user_mfa_recovery_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_mfa_recovery_codes (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    code_hash text NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_mfa_recovery_codes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_mfa_recovery_codes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_mfa_recovery_codes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_mfa_recovery_codes_id_seq OWNED BY public.user_mfa_recovery_codes.id;


--
-- Name: user_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_profiles (
    user_id bigint NOT NULL,
    student_ids jsonb,
    active_student_id character varying(50),
    manual_form_data jsonb,
    verification_status character varying(20) DEFAULT 'unverified'::character varying NOT NULL,
    verification_method character varying(20),
    rejection_reason text,
    reviewed_at timestamp with time zone,
    consent_given_at timestamp with time zone,
    verified_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    school_id bigint,
    CONSTRAINT chk_user_profiles_method CHECK (((verification_method IS NULL) OR ((verification_method)::text = ANY ((ARRAY['ldap'::character varying, 'manual'::character varying])::text[])))),
    CONSTRAINT chk_user_profiles_status CHECK (((verification_status)::text = ANY ((ARRAY['unverified'::character varying, 'pending'::character varying, 'verified'::character varying, 'rejected'::character varying])::text[])))
);


--
-- Name: user_qq_binding_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_qq_binding_codes (
    user_id bigint NOT NULL,
    code_hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_qq_bindings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_qq_bindings (
    user_id bigint NOT NULL,
    qq_id character varying(64) NOT NULL,
    bound_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_verification_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_verification_credentials (
    id character varying(36) NOT NULL,
    user_id bigint NOT NULL,
    school_id bigint NOT NULL,
    kind text NOT NULL,
    subject_hash character varying(128) NOT NULL,
    subject_display text NOT NULL,
    source_application_id character varying(36),
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    verified_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone,
    revoked_at timestamp with time zone,
    expiry_processed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_user_verification_credentials_freshman_expiry CHECK (((kind <> 'freshman_material_manual'::text) OR (expires_at IS NOT NULL))),
    CONSTRAINT chk_user_verification_credentials_kind CHECK ((kind = ANY (ARRAY['school_sso'::text, 'school_email_otp'::text, 'freshman_material_manual'::text])))
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    casdoor_subject character varying(255) NOT NULL,
    username character varying(100) NOT NULL,
    email character varying(255),
    avatar_url text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    phone_enc bytea,
    phone_hash character varying(64),
    user_hash character varying(64),
    CONSTRAINT chk_users_phone_hash_format CHECK (((phone_hash IS NULL) OR ((phone_hash)::text ~ '^[0-9a-f]{64}$'::text))),
    CONSTRAINT chk_users_phone_secure_pair CHECK ((((phone_enc IS NULL) AND (phone_hash IS NULL)) OR ((phone_enc IS NOT NULL) AND (phone_hash IS NOT NULL))))
);


--
-- Name: open_platform_apps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.open_platform_apps (
    id bigint NOT NULL,
    casdoor_application_name text NOT NULL,
    owner_user_id bigint NOT NULL,
    client_id text NOT NULL,
    client_secret_hash character varying(64) NOT NULL,
    display_name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    homepage_url text NOT NULL,
    privacy_policy_url text NOT NULL,
    redirect_uris jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_open_platform_apps_secret_hash CHECK (((client_secret_hash = ''::text) OR ((client_secret_hash)::text ~ '^[0-9a-f]{64}$'::text))),
    CONSTRAINT chk_open_platform_apps_status CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'suspended'::text, 'revoked'::text]))),
    CONSTRAINT chk_open_platform_apps_redirect_uris_array CHECK (jsonb_typeof(redirect_uris) = 'array'::text)
);


--
-- Name: open_platform_apps_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.open_platform_apps_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: open_platform_apps_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.open_platform_apps_id_seq OWNED BY public.open_platform_apps.id;


--
-- Name: open_platform_scope_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.open_platform_scope_requests (
    id bigint NOT NULL,
    app_id bigint NOT NULL,
    scope text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    reviewer_user_id bigint,
    reviewed_at timestamp with time zone,
    decision_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_open_platform_scope_requests_status CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'withdrawn'::text])))
);


--
-- Name: open_platform_scope_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.open_platform_scope_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: open_platform_scope_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.open_platform_scope_requests_id_seq OWNED BY public.open_platform_scope_requests.id;


--
-- Name: open_platform_redirect_uri_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.open_platform_redirect_uri_requests (
    id bigint NOT NULL,
    app_id bigint NOT NULL,
    redirect_uris jsonb NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    reviewer_user_id bigint,
    reviewed_at timestamp with time zone,
    decision_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_open_platform_redirect_uri_requests_status CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'withdrawn'::text]))),
    CONSTRAINT chk_open_platform_redirect_uri_requests_redirect_uris_array CHECK (jsonb_typeof(redirect_uris) = 'array'::text)
);


--
-- Name: open_platform_redirect_uri_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.open_platform_redirect_uri_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: open_platform_redirect_uri_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.open_platform_redirect_uri_requests_id_seq OWNED BY public.open_platform_redirect_uri_requests.id;


--
-- Name: open_platform_approved_scopes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.open_platform_approved_scopes (
    app_id bigint NOT NULL,
    scope text NOT NULL,
    approved_at timestamp with time zone DEFAULT now() NOT NULL,
    approved_by bigint NOT NULL
);


--
-- Name: open_platform_user_consents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.open_platform_user_consents (
    app_id bigint NOT NULL,
    user_id bigint NOT NULL,
    scope text NOT NULL,
    granted_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone,
    grant_source text DEFAULT 'web'::text NOT NULL,
    request_id text DEFAULT ''::text NOT NULL
);


--
-- Name: open_platform_audit_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.open_platform_audit_events (
    id bigint NOT NULL,
    app_id bigint,
    user_id bigint,
    event_type text NOT NULL,
    scope text,
    request_id text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: open_platform_audit_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.open_platform_audit_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: open_platform_audit_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.open_platform_audit_events_id_seq OWNED BY public.open_platform_audit_events.id;


--
-- Name: open_platform_token_probe_evidence; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.open_platform_token_probe_evidence (
    id bigint NOT NULL,
    app_id bigint NOT NULL,
    reviewer_user_id bigint,
    request_id text,
    casdoor_application_name text NOT NULL,
    client_id text NOT NULL,
    redirect_uri text NOT NULL,
    probe_method text NOT NULL,
    result text NOT NULL,
    inspected_claims jsonb NOT NULL,
    business_claims jsonb NOT NULL,
    token_claims jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    error text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_open_platform_token_probe_evidence_result CHECK ((result = ANY (ARRAY['passed'::text, 'failed'::text]))),
    CONSTRAINT chk_open_platform_token_probe_evidence_inspected_claims_array CHECK (jsonb_typeof(inspected_claims) = 'array'::text),
    CONSTRAINT chk_open_platform_token_probe_evidence_business_claims_array CHECK (jsonb_typeof(business_claims) = 'array'::text),
    CONSTRAINT chk_open_platform_token_probe_evidence_token_claims_object CHECK (jsonb_typeof(token_claims) = 'object'::text),
    CONSTRAINT chk_open_platform_token_probe_evidence_metadata_object CHECK (jsonb_typeof(metadata) = 'object'::text)
);


--
-- Name: open_platform_token_probe_evidence_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.open_platform_token_probe_evidence_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: open_platform_token_probe_evidence_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.open_platform_token_probe_evidence_id_seq OWNED BY public.open_platform_token_probe_evidence.id;


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: academic_courses id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_courses ALTER COLUMN id SET DEFAULT nextval('public.academic_courses_id_seq'::regclass);


--
-- Name: academic_import_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_import_jobs ALTER COLUMN id SET DEFAULT nextval('public.academic_import_jobs_id_seq'::regclass);


--
-- Name: academic_memberships id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_memberships ALTER COLUMN id SET DEFAULT nextval('public.academic_memberships_id_seq'::regclass);


--
-- Name: academic_offerings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offerings ALTER COLUMN id SET DEFAULT nextval('public.academic_offerings_id_seq'::regclass);


--
-- Name: academic_schedules id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_schedules ALTER COLUMN id SET DEFAULT nextval('public.academic_schedules_id_seq'::regclass);


--
-- Name: academic_sources id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_sources ALTER COLUMN id SET DEFAULT nextval('public.academic_sources_id_seq'::regclass);


--
-- Name: academic_teachers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_teachers ALTER COLUMN id SET DEFAULT nextval('public.academic_teachers_id_seq'::regclass);


--
-- Name: academic_terms id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_terms ALTER COLUMN id SET DEFAULT nextval('public.academic_terms_id_seq'::regclass);


--
-- Name: bot_service_credentials id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_service_credentials ALTER COLUMN id SET DEFAULT nextval('public.bot_service_credentials_id_seq'::regclass);


--
-- Name: course_categories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_categories ALTER COLUMN id SET DEFAULT nextval('public.course_categories_id_seq'::regclass);


--
-- Name: courses id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.courses ALTER COLUMN id SET DEFAULT nextval('public.courses_id_seq'::regclass);


--
-- Name: departments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.departments ALTER COLUMN id SET DEFAULT nextval('public.departments_id_seq'::regclass);


--
-- Name: domain_event_outbox id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.domain_event_outbox ALTER COLUMN id SET DEFAULT nextval('public.domain_event_outbox_id_seq'::regclass);


--
-- Name: resource_items id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_items ALTER COLUMN id SET DEFAULT nextval('public.resource_items_id_seq'::regclass);


--
-- Name: resource_versions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_versions ALTER COLUMN id SET DEFAULT nextval('public.resource_versions_id_seq'::regclass);


--
-- Name: schools id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schools ALTER COLUMN id SET DEFAULT nextval('public.schools_id_seq'::regclass);


--
-- Name: storage_mounts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.storage_mounts ALTER COLUMN id SET DEFAULT nextval('public.storage_mounts_id_seq'::regclass);


--
-- Name: teachers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teachers ALTER COLUMN id SET DEFAULT nextval('public.teachers_id_seq'::regclass);


--
-- Name: user_mfa_recovery_codes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_recovery_codes ALTER COLUMN id SET DEFAULT nextval('public.user_mfa_recovery_codes_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: open_platform_apps id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_apps ALTER COLUMN id SET DEFAULT nextval('public.open_platform_apps_id_seq'::regclass);


--
-- Name: open_platform_scope_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_scope_requests ALTER COLUMN id SET DEFAULT nextval('public.open_platform_scope_requests_id_seq'::regclass);


--
-- Name: open_platform_redirect_uri_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_redirect_uri_requests ALTER COLUMN id SET DEFAULT nextval('public.open_platform_redirect_uri_requests_id_seq'::regclass);


--
-- Name: open_platform_audit_events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_audit_events ALTER COLUMN id SET DEFAULT nextval('public.open_platform_audit_events_id_seq'::regclass);


--
-- Name: open_platform_token_probe_evidence id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_token_probe_evidence ALTER COLUMN id SET DEFAULT nextval('public.open_platform_token_probe_evidence_id_seq'::regclass);


--
-- Name: buaa_students buaa_students_pkey; Type: CONSTRAINT; Schema: academic; Owner: -
--

ALTER TABLE ONLY academic.buaa_students
    ADD CONSTRAINT buaa_students_pkey PRIMARY KEY (xh);


--
-- Name: academic_courses academic_courses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_courses
    ADD CONSTRAINT academic_courses_pkey PRIMARY KEY (id);


--
-- Name: academic_courses academic_courses_source_id_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_courses
    ADD CONSTRAINT academic_courses_source_id_external_id_key UNIQUE (source_id, external_id);


--
-- Name: academic_import_jobs academic_import_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_import_jobs
    ADD CONSTRAINT academic_import_jobs_pkey PRIMARY KEY (id);


--
-- Name: academic_memberships academic_memberships_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_memberships
    ADD CONSTRAINT academic_memberships_pkey PRIMARY KEY (id);


--
-- Name: academic_memberships academic_memberships_source_id_offering_id_external_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_memberships
    ADD CONSTRAINT academic_memberships_source_id_offering_id_external_user_id_key UNIQUE (source_id, offering_id, external_user_id, role);


--
-- Name: academic_offering_teachers academic_offering_teachers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offering_teachers
    ADD CONSTRAINT academic_offering_teachers_pkey PRIMARY KEY (offering_id, teacher_id);


--
-- Name: academic_offerings academic_offerings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offerings
    ADD CONSTRAINT academic_offerings_pkey PRIMARY KEY (id);


--
-- Name: academic_offerings academic_offerings_source_id_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offerings
    ADD CONSTRAINT academic_offerings_source_id_external_id_key UNIQUE (source_id, external_id);


--
-- Name: academic_schedules academic_schedules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_schedules
    ADD CONSTRAINT academic_schedules_pkey PRIMARY KEY (id);


--
-- Name: academic_sources academic_sources_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_sources
    ADD CONSTRAINT academic_sources_key_key UNIQUE (key);


--
-- Name: academic_sources academic_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_sources
    ADD CONSTRAINT academic_sources_pkey PRIMARY KEY (id);


--
-- Name: academic_teachers academic_teachers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_teachers
    ADD CONSTRAINT academic_teachers_pkey PRIMARY KEY (id);


--
-- Name: academic_teachers academic_teachers_source_id_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_teachers
    ADD CONSTRAINT academic_teachers_source_id_external_id_key UNIQUE (source_id, external_id);


--
-- Name: academic_terms academic_terms_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_terms
    ADD CONSTRAINT academic_terms_pkey PRIMARY KEY (id);


--
-- Name: academic_terms academic_terms_source_id_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_terms
    ADD CONSTRAINT academic_terms_source_id_external_id_key UNIQUE (source_id, external_id);


--
-- Name: audit_events audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_events
    ADD CONSTRAINT audit_events_pkey PRIMARY KEY (id);


--
-- Name: bot_service_credentials bot_service_credentials_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_service_credentials
    ADD CONSTRAINT bot_service_credentials_name_key UNIQUE (name);


--
-- Name: bot_service_credentials bot_service_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_service_credentials
    ADD CONSTRAINT bot_service_credentials_pkey PRIMARY KEY (id);


--
-- Name: bot_service_credentials bot_service_credentials_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_service_credentials
    ADD CONSTRAINT bot_service_credentials_token_hash_key UNIQUE (token_hash);


--
-- Name: course_categories course_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_categories
    ADD CONSTRAINT course_categories_pkey PRIMARY KEY (id);


--
-- Name: course_categories course_categories_school_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_categories
    ADD CONSTRAINT course_categories_school_id_name_key UNIQUE (school_id, name);


--
-- Name: course_favorites course_favorites_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_favorites
    ADD CONSTRAINT course_favorites_pkey PRIMARY KEY (id);


--
-- Name: course_rating_stats course_rating_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_rating_stats
    ADD CONSTRAINT course_rating_stats_pkey PRIMARY KEY (id);


--
-- Name: courses courses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT courses_pkey PRIMARY KEY (id);


--
-- Name: departments departments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT departments_pkey PRIMARY KEY (id);


--
-- Name: domain_event_outbox domain_event_outbox_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.domain_event_outbox
    ADD CONSTRAINT domain_event_outbox_pkey PRIMARY KEY (id);


--
-- Name: freshman_verification_applications freshman_verification_applications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_applications
    ADD CONSTRAINT freshman_verification_applications_pkey PRIMARY KEY (id);


--
-- Name: freshman_verification_materials freshman_verification_materials_application_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_materials
    ADD CONSTRAINT freshman_verification_materials_application_id_key UNIQUE (application_id);


--
-- Name: freshman_verification_materials freshman_verification_materials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_materials
    ADD CONSTRAINT freshman_verification_materials_pkey PRIMARY KEY (id);


--
-- Name: group_admission_failures group_admission_failures_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_admission_failures
    ADD CONSTRAINT group_admission_failures_pkey PRIMARY KEY (platform, guild_id, qq_id);


--
-- Name: member_blacklist_entries member_blacklist_entries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.member_blacklist_entries
    ADD CONSTRAINT member_blacklist_entries_pkey PRIMARY KEY (id);


--
-- Name: group_admission_policies group_admission_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_admission_policies
    ADD CONSTRAINT group_admission_policies_pkey PRIMARY KEY (id);


--
-- Name: group_admission_policies group_admission_policies_platform_guild_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_admission_policies
    ADD CONSTRAINT group_admission_policies_platform_guild_id_key UNIQUE (platform, guild_id);


--
-- Name: group_admission_sessions group_admission_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_admission_sessions
    ADD CONSTRAINT group_admission_sessions_pkey PRIMARY KEY (id);


--
-- Name: group_admission_sessions group_admission_sessions_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_admission_sessions
    ADD CONSTRAINT group_admission_sessions_token_hash_key UNIQUE (token_hash);


--
-- Name: notification_preferences notification_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT notification_preferences_pkey PRIMARY KEY (user_id, type);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: rating_dimensions rating_dimensions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rating_dimensions
    ADD CONSTRAINT rating_dimensions_pkey PRIMARY KEY (id);


--
-- Name: resource_bindings resource_bindings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_bindings
    ADD CONSTRAINT resource_bindings_pkey PRIMARY KEY (resource_id, binding_type, binding_value);


--
-- Name: resource_items resource_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_items
    ADD CONSTRAINT resource_items_pkey PRIMARY KEY (id);


--
-- Name: resource_tags resource_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_tags
    ADD CONSTRAINT resource_tags_pkey PRIMARY KEY (resource_id, tag);


--
-- Name: resource_versions resource_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_versions
    ADD CONSTRAINT resource_versions_pkey PRIMARY KEY (id);


--
-- Name: resource_versions resource_versions_resource_id_version_no_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_versions
    ADD CONSTRAINT resource_versions_resource_id_version_no_key UNIQUE (resource_id, version_no);


--
-- Name: review_drafts review_drafts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_drafts
    ADD CONSTRAINT review_drafts_pkey PRIMARY KEY (id);


--
-- Name: review_replies review_replies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_replies
    ADD CONSTRAINT review_replies_pkey PRIMARY KEY (id);


--
-- Name: review_reports review_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_reports
    ADD CONSTRAINT review_reports_pkey PRIMARY KEY (id);


--
-- Name: review_votes review_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_votes
    ADD CONSTRAINT review_votes_pkey PRIMARY KEY (id);


--
-- Name: reviews reviews_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_pkey PRIMARY KEY (id);


--
-- Name: school_configs school_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.school_configs
    ADD CONSTRAINT school_configs_pkey PRIMARY KEY (school_id);


--
-- Name: schools schools_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schools
    ADD CONSTRAINT schools_code_key UNIQUE (code);


--
-- Name: schools schools_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schools
    ADD CONSTRAINT schools_pkey PRIMARY KEY (id);


--
-- Name: sensitive_words sensitive_words_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sensitive_words
    ADD CONSTRAINT sensitive_words_pkey PRIMARY KEY (id);


--
-- Name: storage_mounts storage_mounts_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.storage_mounts
    ADD CONSTRAINT storage_mounts_key_key UNIQUE (key);


--
-- Name: storage_mounts storage_mounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.storage_mounts
    ADD CONSTRAINT storage_mounts_pkey PRIMARY KEY (id);


--
-- Name: system_configs system_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_configs
    ADD CONSTRAINT system_configs_pkey PRIMARY KEY (key);


--
-- Name: teacher_rating_stats teacher_rating_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teacher_rating_stats
    ADD CONSTRAINT teacher_rating_stats_pkey PRIMARY KEY (id);


--
-- Name: teachers teachers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teachers
    ADD CONSTRAINT teachers_pkey PRIMARY KEY (id);


--
-- Name: terms terms_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.terms
    ADD CONSTRAINT terms_pkey PRIMARY KEY (id);


--
-- Name: course_favorites uq_course_favorites_user_course; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_favorites
    ADD CONSTRAINT uq_course_favorites_user_course UNIQUE (user_hash, course_id);


--
-- Name: course_rating_stats uq_course_rating_stats; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_rating_stats
    ADD CONSTRAINT uq_course_rating_stats UNIQUE (course_id, term_id, dimension_key);


--
-- Name: rating_dimensions uq_rating_dimensions_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rating_dimensions
    ADD CONSTRAINT uq_rating_dimensions_key UNIQUE (key);


--
-- Name: review_drafts uq_review_drafts_user; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_drafts
    ADD CONSTRAINT uq_review_drafts_user UNIQUE (user_hash);


--
-- Name: review_reports uq_review_reports_user; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_reports
    ADD CONSTRAINT uq_review_reports_user UNIQUE (review_id, reporter_hash);


--
-- Name: review_votes uq_review_votes_user; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_votes
    ADD CONSTRAINT uq_review_votes_user UNIQUE (review_id, user_hash);


--
-- Name: sensitive_words uq_sensitive_words_word; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sensitive_words
    ADD CONSTRAINT uq_sensitive_words_word UNIQUE (word);


--
-- Name: teacher_rating_stats uq_teacher_rating_stats; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teacher_rating_stats
    ADD CONSTRAINT uq_teacher_rating_stats UNIQUE (teacher_id, term_id, dimension_key);


--
-- Name: user_mfa_recovery_codes uq_user_mfa_recovery_code_hash; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_recovery_codes
    ADD CONSTRAINT uq_user_mfa_recovery_code_hash UNIQUE (user_id, code_hash);


--
-- Name: user_identities user_identities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_pkey PRIMARY KEY (user_id);


--
-- Name: user_mfa_enrollment user_mfa_enrollment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_enrollment
    ADD CONSTRAINT user_mfa_enrollment_pkey PRIMARY KEY (user_id);


--
-- Name: user_mfa_recovery_codes user_mfa_recovery_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_recovery_codes
    ADD CONSTRAINT user_mfa_recovery_codes_pkey PRIMARY KEY (id);


--
-- Name: user_profiles user_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_profiles
    ADD CONSTRAINT user_profiles_pkey PRIMARY KEY (user_id);


--
-- Name: user_qq_binding_codes user_qq_binding_codes_code_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_qq_binding_codes
    ADD CONSTRAINT user_qq_binding_codes_code_hash_key UNIQUE (code_hash);


--
-- Name: user_qq_binding_codes user_qq_binding_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_qq_binding_codes
    ADD CONSTRAINT user_qq_binding_codes_pkey PRIMARY KEY (user_id);


--
-- Name: user_qq_bindings user_qq_bindings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_qq_bindings
    ADD CONSTRAINT user_qq_bindings_pkey PRIMARY KEY (user_id);


--
-- Name: user_qq_bindings user_qq_bindings_qq_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_qq_bindings
    ADD CONSTRAINT user_qq_bindings_qq_id_key UNIQUE (qq_id);


--
-- Name: user_verification_credentials user_verification_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_verification_credentials
    ADD CONSTRAINT user_verification_credentials_pkey PRIMARY KEY (id);


--
-- Name: users users_casdoor_subject_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_casdoor_subject_key UNIQUE (casdoor_subject);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: open_platform_apps open_platform_apps_client_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_apps
    ADD CONSTRAINT open_platform_apps_client_id_key UNIQUE (client_id);


--
-- Name: open_platform_apps open_platform_apps_casdoor_application_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_apps
    ADD CONSTRAINT open_platform_apps_casdoor_application_name_key UNIQUE (casdoor_application_name);


--
-- Name: open_platform_apps open_platform_apps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_apps
    ADD CONSTRAINT open_platform_apps_pkey PRIMARY KEY (id);


--
-- Name: open_platform_scope_requests open_platform_scope_requests_app_scope_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_scope_requests
    ADD CONSTRAINT open_platform_scope_requests_app_scope_key UNIQUE (app_id, scope);


--
-- Name: open_platform_scope_requests open_platform_scope_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_scope_requests
    ADD CONSTRAINT open_platform_scope_requests_pkey PRIMARY KEY (id);


--
-- Name: open_platform_redirect_uri_requests open_platform_redirect_uri_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_redirect_uri_requests
    ADD CONSTRAINT open_platform_redirect_uri_requests_pkey PRIMARY KEY (id);


--
-- Name: open_platform_approved_scopes open_platform_approved_scopes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_approved_scopes
    ADD CONSTRAINT open_platform_approved_scopes_pkey PRIMARY KEY (app_id, scope);


--
-- Name: open_platform_user_consents open_platform_user_consents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_user_consents
    ADD CONSTRAINT open_platform_user_consents_pkey PRIMARY KEY (app_id, user_id, scope);


--
-- Name: open_platform_audit_events open_platform_audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_audit_events
    ADD CONSTRAINT open_platform_audit_events_pkey PRIMARY KEY (id);


--
-- Name: open_platform_token_probe_evidence open_platform_token_probe_evidence_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_token_probe_evidence
    ADD CONSTRAINT open_platform_token_probe_evidence_pkey PRIMARY KEY (id);


--
-- Name: idx_buaa_students_rxnj; Type: INDEX; Schema: academic; Owner: -
--

CREATE INDEX idx_buaa_students_rxnj ON academic.buaa_students USING btree (rxnj);


--
-- Name: idx_buaa_students_sfzjh_hash; Type: INDEX; Schema: academic; Owner: -
--

CREATE INDEX idx_buaa_students_sfzjh_hash ON academic.buaa_students USING btree (sfzjh_hash);


--
-- Name: idx_buaa_students_yxdm; Type: INDEX; Schema: academic; Owner: -
--

CREATE INDEX idx_buaa_students_yxdm ON academic.buaa_students USING btree (yxdm);


--
-- Name: academic_courses_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX academic_courses_name_idx ON public.academic_courses USING btree (name);


--
-- Name: academic_courses_source_code_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX academic_courses_source_code_idx ON public.academic_courses USING btree (source_id, code);


--
-- Name: academic_import_jobs_source_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX academic_import_jobs_source_created_idx ON public.academic_import_jobs USING btree (source_id, created_at DESC);


--
-- Name: academic_memberships_user_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX academic_memberships_user_idx ON public.academic_memberships USING btree (external_user_id, role);


--
-- Name: academic_offerings_term_course_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX academic_offerings_term_course_idx ON public.academic_offerings USING btree (term_id, course_id);


--
-- Name: academic_teachers_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX academic_teachers_name_idx ON public.academic_teachers USING btree (name);


--
-- Name: academic_terms_source_code_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX academic_terms_source_code_idx ON public.academic_terms USING btree (source_id, code);


--
-- Name: audit_events_action_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX audit_events_action_idx ON public.audit_events USING btree (action, created_at DESC);


--
-- Name: audit_events_actor_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX audit_events_actor_idx ON public.audit_events USING btree (actor_type, actor_user_id, created_at DESC);


--
-- Name: audit_events_category_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX audit_events_category_created_idx ON public.audit_events USING btree (category, created_at DESC);


--
-- Name: audit_events_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX audit_events_created_idx ON public.audit_events USING btree (created_at DESC);


--
-- Name: audit_events_resource_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX audit_events_resource_idx ON public.audit_events USING btree (resource_type, resource_id, created_at DESC);


--
-- Name: domain_event_outbox_stream_dedupe_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX domain_event_outbox_stream_dedupe_idx ON public.domain_event_outbox USING btree (stream, dedupe_key);


--
-- Name: domain_event_outbox_stream_pending_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX domain_event_outbox_stream_pending_idx ON public.domain_event_outbox USING btree (stream, status, available_at, id) WHERE (status = ANY (ARRAY['pending'::text, 'failed'::text]));


--
-- Name: freshman_verification_applications_pending_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX freshman_verification_applications_pending_idx ON public.freshman_verification_applications USING btree (school_id, created_at DESC) WHERE (status = 'pending'::text);


--
-- Name: freshman_verification_applications_pending_user_school_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX freshman_verification_applications_pending_user_school_idx ON public.freshman_verification_applications USING btree (user_id, school_id) WHERE (status = 'pending'::text);


--
-- Name: freshman_verification_materials_application_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX freshman_verification_materials_application_idx ON public.freshman_verification_materials USING btree (application_id);


--
-- Name: group_admission_failures_count_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX group_admission_failures_count_idx ON public.group_admission_failures USING btree (platform, guild_id, failure_count DESC);


--
-- Name: member_blacklist_access_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX member_blacklist_access_idx ON public.member_blacklist_entries USING btree (platform, subject_type, subject_id, scope_type, guild_id) WHERE (released_at IS NULL);


--
-- Name: member_blacklist_global_active_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX member_blacklist_global_active_key ON public.member_blacklist_entries USING btree (platform, subject_type, subject_id) WHERE ((released_at IS NULL) AND (scope_type = 'global'::text));


--
-- Name: member_blacklist_guild_active_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX member_blacklist_guild_active_key ON public.member_blacklist_entries USING btree (platform, subject_type, subject_id, guild_id) WHERE ((released_at IS NULL) AND (scope_type = 'guild'::text));


--
-- Name: group_admission_sessions_active_qq_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX group_admission_sessions_active_qq_idx ON public.group_admission_sessions USING btree (platform, guild_id, qq_id) WHERE (status = ANY (ARRAY['joined_muted'::text, 'linked'::text, 'material_submitted'::text]));


--
-- Name: group_admission_sessions_deadline_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX group_admission_sessions_deadline_idx ON public.group_admission_sessions USING btree (status, link_wait_deadline_at, submission_wait_deadline_at);


--
-- Name: group_admission_sessions_pending_action_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX group_admission_sessions_pending_action_idx ON public.group_admission_sessions USING btree (platform, bot_self_id, status, updated_at, id) WHERE (status = ANY (ARRAY['joined_muted'::text, 'linked'::text, 'material_submitted'::text, 'verified'::text]));


--
-- Name: group_admission_sessions_user_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX group_admission_sessions_user_idx ON public.group_admission_sessions USING btree (user_id, created_at DESC);


--
-- Name: idx_bot_service_credentials_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_bot_service_credentials_active ON public.bot_service_credentials USING btree (revoked_at, last_used_at DESC);


--
-- Name: idx_course_favorites_course_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_course_favorites_course_id ON public.course_favorites USING btree (course_id);


--
-- Name: idx_course_favorites_user_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_course_favorites_user_hash ON public.course_favorites USING btree (user_hash);


--
-- Name: idx_course_rating_stats_course; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_course_rating_stats_course ON public.course_rating_stats USING btree (course_id);


--
-- Name: idx_course_rating_stats_term; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_course_rating_stats_term ON public.course_rating_stats USING btree (course_id, term_id);


--
-- Name: idx_courses_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_courses_category ON public.courses USING btree (category);


--
-- Name: idx_courses_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_courses_code ON public.courses USING btree (code);


--
-- Name: idx_courses_code_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_courses_code_trgm ON public.courses USING gin (code public.gin_trgm_ops);


--
-- Name: idx_courses_department_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_courses_department_id ON public.courses USING btree (department_id);


--
-- Name: idx_courses_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_courses_name ON public.courses USING btree (name);


--
-- Name: idx_courses_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_courses_name_trgm ON public.courses USING gin (name public.gin_trgm_ops);


--
-- Name: idx_departments_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_departments_category ON public.departments USING btree (category);


--
-- Name: idx_departments_school_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_departments_school_id ON public.departments USING btree (school_id);


--
-- Name: idx_mv_teacher_public_stats_dept; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mv_teacher_public_stats_dept ON public.mv_teacher_public_stats USING btree (department_id);


--
-- Name: idx_mv_teacher_public_stats_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_mv_teacher_public_stats_id ON public.mv_teacher_public_stats USING btree (teacher_id);


--
-- Name: idx_mv_teacher_public_stats_rating; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mv_teacher_public_stats_rating ON public.mv_teacher_public_stats USING btree (avg_rating DESC NULLS LAST);


--
-- Name: idx_mv_teacher_public_stats_reviews; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_mv_teacher_public_stats_reviews ON public.mv_teacher_public_stats USING btree (review_count DESC);


--
-- Name: mv_teacher_public_stats; Type: OWNER; Schema: public; Owner: stuhelper_app
--

ALTER MATERIALIZED VIEW public.mv_teacher_public_stats OWNER TO stuhelper_app;


--
-- Name: mv_teacher_public_stats dependencies; Type: ACL; Schema: public; Owner: -
--

GRANT SELECT ON TABLE public.teachers TO stuhelper_app;
GRANT SELECT ON TABLE public.departments TO stuhelper_app;
GRANT SELECT ON TABLE public.reviews TO stuhelper_app;

--
-- Name: academic runtime lookup; Type: ACL; Schema: academic; Owner: -
--

GRANT USAGE ON SCHEMA academic TO stuhelper_app;
GRANT SELECT ON TABLE academic.buaa_students TO stuhelper_app;


--
-- Name: idx_notifications_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_created_at ON public.notifications USING btree (created_at DESC);


--
-- Name: idx_notifications_source_course_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_source_course_id ON public.notifications USING btree (source_course_id) WHERE (source_course_id IS NOT NULL);


--
-- Name: idx_notifications_user_id_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_id_created ON public.notifications USING btree (user_id, created_at DESC);


--
-- Name: idx_notifications_user_id_unread; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_id_unread ON public.notifications USING btree (user_id) WHERE (is_read = false);


--
-- Name: idx_rating_dimensions_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rating_dimensions_active ON public.rating_dimensions USING btree (is_active);


--
-- Name: idx_rating_dimensions_school; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rating_dimensions_school ON public.rating_dimensions USING btree (school_id);


--
-- Name: idx_review_drafts_teacher_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_drafts_teacher_id ON public.review_drafts USING btree (teacher_id) WHERE (teacher_id IS NOT NULL);


--
-- Name: idx_review_replies_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_replies_created_at ON public.review_replies USING btree (review_id, created_at DESC);


--
-- Name: idx_review_replies_parent_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_replies_parent_id ON public.review_replies USING btree (parent_id);


--
-- Name: idx_review_replies_pending_review; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_replies_pending_review ON public.review_replies USING btree (created_at DESC) WHERE ((status)::text = 'pending_review'::text);


--
-- Name: idx_review_replies_review_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_replies_review_id ON public.review_replies USING btree (review_id);


--
-- Name: idx_review_replies_user_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_replies_user_hash ON public.review_replies USING btree (user_hash);


--
-- Name: idx_review_reports_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_reports_created_at ON public.review_reports USING btree (created_at DESC);


--
-- Name: idx_review_reports_school_status_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_reports_school_status_created ON public.review_reports USING btree (school_id, status, created_at DESC);


--
-- Name: idx_review_reports_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_reports_status ON public.review_reports USING btree (status);


--
-- Name: idx_review_reports_status_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_reports_status_created ON public.review_reports USING btree (status, created_at DESC);


--
-- Name: idx_review_votes_user_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_review_votes_user_hash ON public.review_votes USING btree (user_hash, vote_type);


--
-- Name: idx_reviews_avg_rating; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_avg_rating ON public.reviews USING btree (avg_rating DESC);


--
-- Name: idx_reviews_content_flag; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_content_flag ON public.reviews USING btree (content_flag) WHERE (content_flag IS NOT NULL);


--
-- Name: idx_reviews_course_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_course_id ON public.reviews USING btree (course_id);


--
-- Name: idx_reviews_course_status_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_course_status_created ON public.reviews USING btree (course_id, status, created_at DESC);


--
-- Name: idx_reviews_course_teacher_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_course_teacher_status ON public.reviews USING btree (course_id, teacher_id, status);


--
-- Name: idx_reviews_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_created_at ON public.reviews USING btree (created_at DESC);


--
-- Name: idx_reviews_pending_review; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_pending_review ON public.reviews USING btree (created_at DESC) WHERE ((status)::text = 'pending_review'::text);


--
-- Name: idx_reviews_school_status_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_school_status_created ON public.reviews USING btree (school_id, status, created_at DESC);


--
-- Name: idx_reviews_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_status ON public.reviews USING btree (status);


--
-- Name: idx_reviews_teacher_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_teacher_id ON public.reviews USING btree (teacher_id);


--
-- Name: idx_reviews_term_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_term_id ON public.reviews USING btree (term_id);


--
-- Name: idx_reviews_user_course; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_reviews_user_course ON public.reviews USING btree (user_hash, course_id) WHERE ((status)::text <> 'deleted'::text);


--
-- Name: idx_reviews_user_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_user_hash ON public.reviews USING btree (user_hash);


--
-- Name: idx_reviews_user_hash_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reviews_user_hash_created ON public.reviews USING btree (user_hash, created_at DESC);


--
-- Name: idx_sensitive_words_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sensitive_words_active ON public.sensitive_words USING btree (is_active);


--
-- Name: idx_sensitive_words_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sensitive_words_category ON public.sensitive_words USING btree (category);


--
-- Name: idx_teacher_rating_stats_teacher; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_teacher_rating_stats_teacher ON public.teacher_rating_stats USING btree (teacher_id);


--
-- Name: idx_teacher_rating_stats_term; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_teacher_rating_stats_term ON public.teacher_rating_stats USING btree (teacher_id, term_id);


--
-- Name: idx_teachers_department_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_teachers_department_id ON public.teachers USING btree (department_id);


--
-- Name: idx_teachers_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_teachers_name ON public.teachers USING btree (name);


--
-- Name: idx_teachers_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_teachers_name_trgm ON public.teachers USING gin (name public.gin_trgm_ops);


--
-- Name: idx_teachers_school_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_teachers_school_id ON public.teachers USING btree (school_id);


--
-- Name: idx_user_identities_person_uid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_user_identities_person_uid ON public.user_identities USING btree (person_uid);


--
-- Name: idx_user_mfa_enrollment_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_mfa_enrollment_active ON public.user_mfa_enrollment USING btree (active, updated_at DESC);


--
-- Name: idx_user_mfa_recovery_codes_unused; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_mfa_recovery_codes_unused ON public.user_mfa_recovery_codes USING btree (user_id, created_at DESC) WHERE (used_at IS NULL);


--
-- Name: idx_user_profiles_school; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_profiles_school ON public.user_profiles USING btree (school_id);


--
-- Name: idx_user_profiles_school_student; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_user_profiles_school_student ON public.user_profiles USING btree (school_id, active_student_id) WHERE (active_student_id IS NOT NULL);


--
-- Name: idx_user_profiles_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_profiles_status ON public.user_profiles USING btree (verification_status);


--
-- Name: idx_user_qq_binding_codes_code_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_qq_binding_codes_code_hash ON public.user_qq_binding_codes USING btree (code_hash);


--
-- Name: idx_user_qq_binding_codes_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_qq_binding_codes_expires_at ON public.user_qq_binding_codes USING btree (expires_at);


--
-- Name: idx_user_qq_bindings_qq_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_qq_bindings_qq_id ON public.user_qq_bindings USING btree (qq_id);


--
-- Name: idx_users_phone_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_phone_hash ON public.users USING btree (phone_hash) WHERE (phone_hash IS NOT NULL);


--
-- Name: idx_users_user_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_user_hash ON public.users USING btree (user_hash) WHERE (user_hash IS NOT NULL);


--
-- Name: idx_users_username; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_username ON public.users USING btree (username);


--
-- Name: idx_open_platform_apps_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_apps_owner ON public.open_platform_apps USING btree (owner_user_id);


--
-- Name: idx_open_platform_apps_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_apps_status ON public.open_platform_apps USING btree (status);


--
-- Name: idx_open_platform_scope_requests_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_scope_requests_status ON public.open_platform_scope_requests USING btree (status);


--
-- Name: idx_open_platform_redirect_uri_requests_app_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_redirect_uri_requests_app_created ON public.open_platform_redirect_uri_requests USING btree (app_id, created_at DESC);


--
-- Name: idx_open_platform_redirect_uri_requests_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_redirect_uri_requests_status ON public.open_platform_redirect_uri_requests USING btree (status);


--
-- Name: idx_open_platform_redirect_uri_requests_pending_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_open_platform_redirect_uri_requests_pending_unique ON public.open_platform_redirect_uri_requests USING btree (app_id) WHERE (status = 'pending'::text);


--
-- Name: idx_open_platform_user_consents_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_user_consents_user ON public.open_platform_user_consents USING btree (user_id);


--
-- Name: idx_open_platform_user_consents_active_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_user_consents_active_user ON public.open_platform_user_consents USING btree (user_id, app_id, scope) WHERE (revoked_at IS NULL);


--
-- Name: idx_open_platform_user_consents_active_app; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_user_consents_active_app ON public.open_platform_user_consents USING btree (app_id, user_id, scope) WHERE (revoked_at IS NULL);


--
-- Name: idx_open_platform_audit_events_app_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_audit_events_app_created ON public.open_platform_audit_events USING btree (app_id, created_at DESC);


--
-- Name: idx_open_platform_audit_events_type_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_audit_events_type_created ON public.open_platform_audit_events USING btree (event_type, created_at DESC);


--
-- Name: idx_open_platform_audit_events_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_audit_events_user_created ON public.open_platform_audit_events USING btree (user_id, created_at DESC);


--
-- Name: idx_open_platform_audit_events_disclosure_usage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_audit_events_disclosure_usage ON public.open_platform_audit_events USING btree (app_id, user_id, created_at DESC, id DESC) WHERE (event_type = 'open_platform.disclosure.granted'::text);


--
-- Name: idx_open_platform_token_probe_evidence_app_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_token_probe_evidence_app_created ON public.open_platform_token_probe_evidence USING btree (app_id, created_at DESC);


--
-- Name: idx_open_platform_token_probe_evidence_result_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_open_platform_token_probe_evidence_result_created ON public.open_platform_token_probe_evidence USING btree (result, created_at DESC);


--
-- Name: resource_bindings_lookup_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX resource_bindings_lookup_idx ON public.resource_bindings USING btree (binding_type, binding_value);


--
-- Name: open_platform_apps open_platform_apps_owner_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_apps
    ADD CONSTRAINT open_platform_apps_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: open_platform_scope_requests open_platform_scope_requests_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_scope_requests
    ADD CONSTRAINT open_platform_scope_requests_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.open_platform_apps(id) ON DELETE CASCADE;


--
-- Name: open_platform_scope_requests open_platform_scope_requests_reviewer_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_scope_requests
    ADD CONSTRAINT open_platform_scope_requests_reviewer_user_id_fkey FOREIGN KEY (reviewer_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: open_platform_redirect_uri_requests open_platform_redirect_uri_requests_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_redirect_uri_requests
    ADD CONSTRAINT open_platform_redirect_uri_requests_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.open_platform_apps(id) ON DELETE CASCADE;


--
-- Name: open_platform_redirect_uri_requests open_platform_redirect_uri_requests_reviewer_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_redirect_uri_requests
    ADD CONSTRAINT open_platform_redirect_uri_requests_reviewer_user_id_fkey FOREIGN KEY (reviewer_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: open_platform_approved_scopes open_platform_approved_scopes_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_approved_scopes
    ADD CONSTRAINT open_platform_approved_scopes_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.open_platform_apps(id) ON DELETE CASCADE;


--
-- Name: open_platform_approved_scopes open_platform_approved_scopes_approved_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_approved_scopes
    ADD CONSTRAINT open_platform_approved_scopes_approved_by_fkey FOREIGN KEY (approved_by) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: open_platform_user_consents open_platform_user_consents_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_user_consents
    ADD CONSTRAINT open_platform_user_consents_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.open_platform_apps(id) ON DELETE CASCADE;


--
-- Name: open_platform_user_consents open_platform_user_consents_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_user_consents
    ADD CONSTRAINT open_platform_user_consents_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: open_platform_audit_events open_platform_audit_events_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_audit_events
    ADD CONSTRAINT open_platform_audit_events_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.open_platform_apps(id) ON DELETE SET NULL;


--
-- Name: open_platform_audit_events open_platform_audit_events_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_audit_events
    ADD CONSTRAINT open_platform_audit_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: open_platform_token_probe_evidence open_platform_token_probe_evidence_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_token_probe_evidence
    ADD CONSTRAINT open_platform_token_probe_evidence_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.open_platform_apps(id) ON DELETE CASCADE;


--
-- Name: open_platform_token_probe_evidence open_platform_token_probe_evidence_reviewer_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.open_platform_token_probe_evidence
    ADD CONSTRAINT open_platform_token_probe_evidence_reviewer_user_id_fkey FOREIGN KEY (reviewer_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: resource_items_visibility_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX resource_items_visibility_created_idx ON public.resource_items USING btree (visibility, created_at DESC);


--
-- Name: resource_tags_tag_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX resource_tags_tag_idx ON public.resource_tags USING btree (tag);


--
-- Name: resource_versions_resource_version_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX resource_versions_resource_version_idx ON public.resource_versions USING btree (resource_id, version_no DESC);


--
-- Name: storage_mounts_enabled_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX storage_mounts_enabled_idx ON public.storage_mounts USING btree (enabled, key);


--
-- Name: user_verification_credentials_expiry_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_verification_credentials_expiry_idx ON public.user_verification_credentials USING btree (expires_at) WHERE ((expires_at IS NOT NULL) AND (revoked_at IS NULL) AND (expiry_processed_at IS NULL));


--
-- Name: user_verification_credentials_user_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_verification_credentials_user_idx ON public.user_verification_credentials USING btree (user_id, school_id, kind);


--
-- Name: academic_courses academic_courses_import_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_courses
    ADD CONSTRAINT academic_courses_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.academic_import_jobs(id) ON DELETE CASCADE;


--
-- Name: academic_courses academic_courses_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_courses
    ADD CONSTRAINT academic_courses_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.academic_sources(id) ON DELETE CASCADE;


--
-- Name: academic_import_jobs academic_import_jobs_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_import_jobs
    ADD CONSTRAINT academic_import_jobs_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.academic_sources(id) ON DELETE CASCADE;


--
-- Name: academic_memberships academic_memberships_import_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_memberships
    ADD CONSTRAINT academic_memberships_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.academic_import_jobs(id) ON DELETE CASCADE;


--
-- Name: academic_memberships academic_memberships_offering_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_memberships
    ADD CONSTRAINT academic_memberships_offering_id_fkey FOREIGN KEY (offering_id) REFERENCES public.academic_offerings(id) ON DELETE CASCADE;


--
-- Name: academic_memberships academic_memberships_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_memberships
    ADD CONSTRAINT academic_memberships_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.academic_sources(id) ON DELETE CASCADE;


--
-- Name: academic_offering_teachers academic_offering_teachers_offering_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offering_teachers
    ADD CONSTRAINT academic_offering_teachers_offering_id_fkey FOREIGN KEY (offering_id) REFERENCES public.academic_offerings(id) ON DELETE CASCADE;


--
-- Name: academic_offering_teachers academic_offering_teachers_teacher_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offering_teachers
    ADD CONSTRAINT academic_offering_teachers_teacher_id_fkey FOREIGN KEY (teacher_id) REFERENCES public.academic_teachers(id) ON DELETE CASCADE;


--
-- Name: academic_offerings academic_offerings_course_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offerings
    ADD CONSTRAINT academic_offerings_course_id_fkey FOREIGN KEY (course_id) REFERENCES public.academic_courses(id) ON DELETE CASCADE;


--
-- Name: academic_offerings academic_offerings_import_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offerings
    ADD CONSTRAINT academic_offerings_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.academic_import_jobs(id) ON DELETE CASCADE;


--
-- Name: academic_offerings academic_offerings_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offerings
    ADD CONSTRAINT academic_offerings_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.academic_sources(id) ON DELETE CASCADE;


--
-- Name: academic_offerings academic_offerings_term_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_offerings
    ADD CONSTRAINT academic_offerings_term_id_fkey FOREIGN KEY (term_id) REFERENCES public.academic_terms(id) ON DELETE CASCADE;


--
-- Name: academic_schedules academic_schedules_offering_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_schedules
    ADD CONSTRAINT academic_schedules_offering_id_fkey FOREIGN KEY (offering_id) REFERENCES public.academic_offerings(id) ON DELETE CASCADE;


--
-- Name: academic_teachers academic_teachers_import_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_teachers
    ADD CONSTRAINT academic_teachers_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.academic_import_jobs(id) ON DELETE CASCADE;


--
-- Name: academic_teachers academic_teachers_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_teachers
    ADD CONSTRAINT academic_teachers_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.academic_sources(id) ON DELETE CASCADE;


--
-- Name: academic_terms academic_terms_import_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_terms
    ADD CONSTRAINT academic_terms_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.academic_import_jobs(id) ON DELETE CASCADE;


--
-- Name: academic_terms academic_terms_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_terms
    ADD CONSTRAINT academic_terms_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.academic_sources(id) ON DELETE CASCADE;


--
-- Name: course_categories fk_course_categories_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_categories
    ADD CONSTRAINT fk_course_categories_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: course_favorites fk_course_favorites_course; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_favorites
    ADD CONSTRAINT fk_course_favorites_course FOREIGN KEY (course_id) REFERENCES public.courses(id) ON DELETE CASCADE;


--
-- Name: course_rating_stats fk_course_rating_stats_course; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_rating_stats
    ADD CONSTRAINT fk_course_rating_stats_course FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: course_rating_stats fk_course_rating_stats_dimension; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.course_rating_stats
    ADD CONSTRAINT fk_course_rating_stats_dimension FOREIGN KEY (dimension_key) REFERENCES public.rating_dimensions(key) ON DELETE CASCADE;


--
-- Name: courses fk_courses_department; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT fk_courses_department FOREIGN KEY (department_id) REFERENCES public.departments(id);


--
-- Name: courses fk_courses_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT fk_courses_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: departments fk_departments_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT fk_departments_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: notification_preferences fk_notification_preferences_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT fk_notification_preferences_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: notifications fk_notifications_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: review_drafts fk_review_drafts_course; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_drafts
    ADD CONSTRAINT fk_review_drafts_course FOREIGN KEY (course_id) REFERENCES public.courses(id) ON DELETE CASCADE;


--
-- Name: review_drafts fk_review_drafts_teacher; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_drafts
    ADD CONSTRAINT fk_review_drafts_teacher FOREIGN KEY (teacher_id) REFERENCES public.teachers(id) ON DELETE SET NULL;


--
-- Name: review_drafts fk_review_drafts_term; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_drafts
    ADD CONSTRAINT fk_review_drafts_term FOREIGN KEY (term_id) REFERENCES public.terms(id) ON DELETE RESTRICT;


--
-- Name: review_replies fk_review_replies_parent; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_replies
    ADD CONSTRAINT fk_review_replies_parent FOREIGN KEY (parent_id) REFERENCES public.review_replies(id) ON DELETE CASCADE;


--
-- Name: review_replies fk_review_replies_review; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_replies
    ADD CONSTRAINT fk_review_replies_review FOREIGN KEY (review_id) REFERENCES public.reviews(id) ON DELETE CASCADE;


--
-- Name: review_reports fk_review_reports_review; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_reports
    ADD CONSTRAINT fk_review_reports_review FOREIGN KEY (review_id) REFERENCES public.reviews(id) ON DELETE CASCADE;


--
-- Name: review_reports fk_review_reports_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_reports
    ADD CONSTRAINT fk_review_reports_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: review_votes fk_review_votes_review; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_votes
    ADD CONSTRAINT fk_review_votes_review FOREIGN KEY (review_id) REFERENCES public.reviews(id) ON DELETE CASCADE;


--
-- Name: reviews fk_reviews_course; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT fk_reviews_course FOREIGN KEY (course_id) REFERENCES public.courses(id) ON DELETE RESTRICT;


--
-- Name: reviews fk_reviews_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT fk_reviews_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: reviews fk_reviews_teacher; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT fk_reviews_teacher FOREIGN KEY (teacher_id) REFERENCES public.teachers(id) ON DELETE SET NULL;


--
-- Name: reviews fk_reviews_term; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT fk_reviews_term FOREIGN KEY (term_id) REFERENCES public.terms(id) ON DELETE RESTRICT;


--
-- Name: school_configs fk_school_configs_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.school_configs
    ADD CONSTRAINT fk_school_configs_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: teacher_rating_stats fk_teacher_rating_stats_dimension; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teacher_rating_stats
    ADD CONSTRAINT fk_teacher_rating_stats_dimension FOREIGN KEY (dimension_key) REFERENCES public.rating_dimensions(key) ON DELETE CASCADE;


--
-- Name: teacher_rating_stats fk_teacher_rating_stats_teacher; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teacher_rating_stats
    ADD CONSTRAINT fk_teacher_rating_stats_teacher FOREIGN KEY (teacher_id) REFERENCES public.teachers(id) ON DELETE CASCADE;


--
-- Name: teachers fk_teachers_department; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teachers
    ADD CONSTRAINT fk_teachers_department FOREIGN KEY (department_id) REFERENCES public.departments(id);


--
-- Name: teachers fk_teachers_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teachers
    ADD CONSTRAINT fk_teachers_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: terms fk_terms_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.terms
    ADD CONSTRAINT fk_terms_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: user_profiles fk_user_profiles_school; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_profiles
    ADD CONSTRAINT fk_user_profiles_school FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: user_verification_credentials fk_user_verification_credentials_source_application; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_verification_credentials
    ADD CONSTRAINT fk_user_verification_credentials_source_application FOREIGN KEY (source_application_id) REFERENCES public.freshman_verification_applications(id) ON DELETE SET NULL;


--
-- Name: freshman_verification_applications freshman_verification_applications_admission_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_applications
    ADD CONSTRAINT freshman_verification_applications_admission_session_id_fkey FOREIGN KEY (admission_session_id) REFERENCES public.group_admission_sessions(id) ON DELETE SET NULL;


--
-- Name: freshman_verification_applications freshman_verification_applications_reviewed_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_applications
    ADD CONSTRAINT freshman_verification_applications_reviewed_by_user_id_fkey FOREIGN KEY (reviewed_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: freshman_verification_applications freshman_verification_applications_school_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_applications
    ADD CONSTRAINT freshman_verification_applications_school_id_fkey FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: freshman_verification_applications freshman_verification_applications_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_applications
    ADD CONSTRAINT freshman_verification_applications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: freshman_verification_materials freshman_verification_materials_application_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.freshman_verification_materials
    ADD CONSTRAINT freshman_verification_materials_application_id_fkey FOREIGN KEY (application_id) REFERENCES public.freshman_verification_applications(id) ON DELETE CASCADE;


--
-- Name: group_admission_policies group_admission_policies_school_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_admission_policies
    ADD CONSTRAINT group_admission_policies_school_id_fkey FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: group_admission_sessions group_admission_sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_admission_sessions
    ADD CONSTRAINT group_admission_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: resource_bindings resource_bindings_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_bindings
    ADD CONSTRAINT resource_bindings_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.resource_items(id) ON DELETE CASCADE;


--
-- Name: resource_tags resource_tags_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_tags
    ADD CONSTRAINT resource_tags_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.resource_items(id) ON DELETE CASCADE;


--
-- Name: resource_versions resource_versions_mount_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_versions
    ADD CONSTRAINT resource_versions_mount_id_fkey FOREIGN KEY (mount_id) REFERENCES public.storage_mounts(id) ON DELETE RESTRICT;


--
-- Name: resource_versions resource_versions_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_versions
    ADD CONSTRAINT resource_versions_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.resource_items(id) ON DELETE CASCADE;


--
-- Name: user_identities user_identities_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_mfa_enrollment user_mfa_enrollment_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_enrollment
    ADD CONSTRAINT user_mfa_enrollment_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_mfa_recovery_codes user_mfa_recovery_codes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_recovery_codes
    ADD CONSTRAINT user_mfa_recovery_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_profiles user_profiles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_profiles
    ADD CONSTRAINT user_profiles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_qq_binding_codes user_qq_binding_codes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_qq_binding_codes
    ADD CONSTRAINT user_qq_binding_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_qq_bindings user_qq_bindings_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_qq_bindings
    ADD CONSTRAINT user_qq_bindings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_verification_credentials user_verification_credentials_school_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_verification_credentials
    ADD CONSTRAINT user_verification_credentials_school_id_fkey FOREIGN KEY (school_id) REFERENCES public.schools(id) ON DELETE RESTRICT;


--
-- Name: user_verification_credentials user_verification_credentials_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_verification_credentials
    ADD CONSTRAINT user_verification_credentials_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--



--
-- Baseline seed data for a fresh StuHelper database.
--

INSERT INTO public.schools (id, code, name)
VALUES
    (4111010006, '4111010006', '北京航空航天大学');

SELECT pg_catalog.setval('public.schools_id_seq', 4111010006, true);

INSERT INTO public.terms (id, school_id, name, is_current)
VALUES
    ('2025-1', 4111010006, '2025 春', false),
    ('2025-2', 4111010006, '2025 秋', true);

INSERT INTO public.course_categories (school_id, name, sort_order)
VALUES
    (4111010006, '通识', 0),
    (4111010006, '体育', 1),
    (4111010006, '英语', 2),
    (4111010006, '思政', 3);

INSERT INTO public.rating_dimensions (id, school_id, key, name, description, sort_order)
VALUES
    ('01973860-0001-7000-8000-000000000001', 4111010006, 'difficulty', '课程难度', '课程内容的难易程度', 1),
    ('01973860-0002-7000-8000-000000000002', 4111010006, 'workload', '作业量', '课程作业和任务的工作量', 2),
    ('01973860-0003-7000-8000-000000000003', 4111010006, 'usefulness', '实用性', '课程内容对未来学习或工作的帮助程度', 3),
    ('01973860-0004-7000-8000-000000000004', 4111010006, 'teaching', '教学质量', '教师的授课水平和教学效果', 4),
    ('01973860-0005-7000-8000-000000000005', 4111010006, 'grading', '给分情况', '课程的评分标准和给分宽松程度', 5);

INSERT INTO public.school_configs (
    school_id, school_name, verification_method, academic_db_table, consent_text, enabled, approval_policy
)
VALUES (
    4111010006,
    '北京航空航天大学',
    'ldap',
    'academic.buaa_students',
    '本功能将使用您提供的学号和密码通过学校统一身份认证系统验证您的学生身份。验证成功后，系统将读取您的姓名、院系、年级、手机号等学籍信息用于平台服务。您的密码不会被存储。',
    false,
    'auto'
);

INSERT INTO public.system_configs (key, value, description)
VALUES
    ('review_access_school_ids', '["4111010006"]', '允许查看完整评课和发布评课的学校 ID 列表（JSON 数组；留空则回退到已启用学校）'),
    ('review_preview_title_chars', '24', '评课标题预览最大字符数'),
    ('review_preview_content_chars', '120', '评课正文预览最大字符数'),
    ('review_preview_content_percent', '100', '评课正文预览最大展示比例（1-100）'),
    ('auth_access_token_ttl_seconds', '300', 'Access Token 有效期（秒）；可在管理后台系统设置中热更新'),
    ('email.delivery_policy', '{"mode":"priority","maxAttempts":2,"providers":[{"name":"tencent_ses","enabled":true,"priority":10,"weight":100},{"name":"resend","enabled":true,"priority":20,"weight":100}]}', '邮件发送提供商策略；mode=priority/weighted，priority 越小越优先，weight 用于同优先级负载均衡');

INSERT INTO public.academic_sources (key, name, provider, config)
VALUES ('buaa-fixture', 'BUAA Demo Fixture', 'fixture', '{"fixture":"buaa-default"}'::jsonb);

INSERT INTO public.storage_mounts (key, name, driver, credential_source)
VALUES ('default-s3', 'Default S3 Mount', 's3', 'runtime_default_object_storage');
