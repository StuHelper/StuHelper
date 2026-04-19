CREATE TABLE academic_sources (
    id BIGSERIAL PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE academic_import_jobs (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES academic_sources (id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    trigger_mode TEXT NOT NULL DEFAULT 'manual',
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
    requested_by_user_id TEXT,
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE academic_terms (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES academic_sources (id) ON DELETE CASCADE,
    import_job_id BIGINT NOT NULL REFERENCES academic_import_jobs (id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    start_date DATE,
    end_date DATE,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE academic_courses (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES academic_sources (id) ON DELETE CASCADE,
    import_job_id BIGINT NOT NULL REFERENCES academic_import_jobs (id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    department_code TEXT,
    department_name TEXT,
    credit NUMERIC(4,1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE academic_teachers (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES academic_sources (id) ON DELETE CASCADE,
    import_job_id BIGINT NOT NULL REFERENCES academic_import_jobs (id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    department_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE academic_offerings (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES academic_sources (id) ON DELETE CASCADE,
    import_job_id BIGINT NOT NULL REFERENCES academic_import_jobs (id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    term_id BIGINT NOT NULL REFERENCES academic_terms (id) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES academic_courses (id) ON DELETE CASCADE,
    section_code TEXT NOT NULL,
    school_name TEXT,
    department_name TEXT,
    campus TEXT,
    enrollment_limit INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE academic_offering_teachers (
    offering_id BIGINT NOT NULL REFERENCES academic_offerings (id) ON DELETE CASCADE,
    teacher_id BIGINT NOT NULL REFERENCES academic_teachers (id) ON DELETE CASCADE,
    PRIMARY KEY (offering_id, teacher_id)
);

CREATE TABLE academic_schedules (
    id BIGSERIAL PRIMARY KEY,
    offering_id BIGINT NOT NULL REFERENCES academic_offerings (id) ON DELETE CASCADE,
    weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 1 AND 7),
    start_period SMALLINT NOT NULL CHECK (start_period >= 1),
    end_period SMALLINT NOT NULL CHECK (end_period >= start_period),
    location TEXT NOT NULL,
    building TEXT,
    weeks_text TEXT NOT NULL
);

CREATE TABLE academic_memberships (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES academic_sources (id) ON DELETE CASCADE,
    import_job_id BIGINT NOT NULL REFERENCES academic_import_jobs (id) ON DELETE CASCADE,
    offering_id BIGINT NOT NULL REFERENCES academic_offerings (id) ON DELETE CASCADE,
    external_user_id TEXT NOT NULL,
    student_id TEXT,
    role TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'assistant')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, offering_id, external_user_id, role)
);

CREATE INDEX academic_import_jobs_source_created_idx
    ON academic_import_jobs (source_id, created_at DESC);
CREATE INDEX academic_terms_source_code_idx
    ON academic_terms (source_id, code);
CREATE INDEX academic_courses_source_code_idx
    ON academic_courses (source_id, code);
CREATE INDEX academic_courses_name_idx
    ON academic_courses (name);
CREATE INDEX academic_teachers_name_idx
    ON academic_teachers (name);
CREATE INDEX academic_offerings_term_course_idx
    ON academic_offerings (term_id, course_id);
CREATE INDEX academic_memberships_user_idx
    ON academic_memberships (external_user_id, role);

INSERT INTO academic_sources (key, name, provider, config)
VALUES (
    'buaa-fixture',
    'BUAA Demo Fixture',
    'fixture',
    '{"fixture":"buaa-default"}'::jsonb
)
ON CONFLICT (key) DO NOTHING;
