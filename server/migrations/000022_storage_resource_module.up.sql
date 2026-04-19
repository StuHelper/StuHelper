CREATE TABLE storage_mounts (
    id BIGSERIAL PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    driver TEXT NOT NULL,
    bucket TEXT,
    base_path TEXT NOT NULL DEFAULT '',
    credential_source TEXT NOT NULL DEFAULT 'runtime_default_object_storage',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_health_status TEXT,
    last_health_error TEXT,
    last_health_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE resource_items (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    category TEXT,
    visibility TEXT NOT NULL CHECK (visibility IN ('public', 'private')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE resource_versions (
    id BIGSERIAL PRIMARY KEY,
    resource_id BIGINT NOT NULL REFERENCES resource_items (id) ON DELETE CASCADE,
    version_no INTEGER NOT NULL,
    mount_id BIGINT NOT NULL REFERENCES storage_mounts (id) ON DELETE RESTRICT,
    object_key TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (resource_id, version_no)
);

CREATE TABLE resource_bindings (
    resource_id BIGINT NOT NULL REFERENCES resource_items (id) ON DELETE CASCADE,
    binding_type TEXT NOT NULL,
    binding_value TEXT NOT NULL,
    PRIMARY KEY (resource_id, binding_type, binding_value)
);

CREATE TABLE resource_tags (
    resource_id BIGINT NOT NULL REFERENCES resource_items (id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (resource_id, tag)
);

CREATE INDEX storage_mounts_enabled_idx
    ON storage_mounts (enabled, key);
CREATE INDEX resource_items_visibility_created_idx
    ON resource_items (visibility, created_at DESC);
CREATE INDEX resource_versions_resource_version_idx
    ON resource_versions (resource_id, version_no DESC);
CREATE INDEX resource_bindings_lookup_idx
    ON resource_bindings (binding_type, binding_value);
CREATE INDEX resource_tags_tag_idx
    ON resource_tags (tag);

INSERT INTO storage_mounts (key, name, driver, credential_source)
VALUES ('default-s3', 'Default S3 Mount', 's3', 'runtime_default_object_storage')
ON CONFLICT (key) DO NOTHING;
