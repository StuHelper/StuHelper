-- Restricted campus-network connector control plane.
-- This registry never carries arbitrary network routes, SQL, shell commands,
-- end-user passwords, service-account secrets, or private keys.

CREATE TABLE public.campus_connector_nodes (
    id character varying(36) PRIMARY KEY,
    display_name text NOT NULL,
    status text NOT NULL DEFAULT 'registered',
    protocol_version text NOT NULL,
    software_version text NOT NULL,
    certificate_fingerprint character varying(64) NOT NULL,
    signing_key_id text NOT NULL,
    signing_public_key bytea NOT NULL,
    max_concurrency integer NOT NULL DEFAULT 4,
    heartbeat_interval_seconds integer NOT NULL DEFAULT 30,
    last_heartbeat_at timestamp with time zone,
    last_health_code text,
    certificate_not_after timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_campus_connector_nodes_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_campus_connector_nodes_display_name
        CHECK (char_length(display_name) BETWEEN 1 AND 100 AND display_name = btrim(display_name)),
    CONSTRAINT chk_campus_connector_nodes_status
        CHECK (status IN ('registered', 'active', 'degraded', 'offline', 'revoked')),
    CONSTRAINT chk_campus_connector_nodes_versions
        CHECK (
            char_length(protocol_version) BETWEEN 1 AND 64
            AND char_length(software_version) BETWEEN 1 AND 128
        ),
    CONSTRAINT chk_campus_connector_nodes_certificate
        CHECK (
            certificate_fingerprint ~ '^[0-9a-f]{64}$'
            AND octet_length(signing_public_key) > 0
            AND char_length(signing_key_id) BETWEEN 1 AND 128
        ),
    CONSTRAINT chk_campus_connector_nodes_limits
        CHECK (
            max_concurrency BETWEEN 1 AND 128
            AND heartbeat_interval_seconds BETWEEN 5 AND 3600
        ),
    CONSTRAINT chk_campus_connector_nodes_revocation
        CHECK (
            (status = 'revoked' AND revoked_at IS NOT NULL)
            OR (status <> 'revoked' AND revoked_at IS NULL)
        ),
    CONSTRAINT chk_campus_connector_nodes_revision
        CHECK (revision > 0)
);

CREATE UNIQUE INDEX campus_connector_nodes_certificate_uidx
    ON public.campus_connector_nodes (certificate_fingerprint);
CREATE INDEX campus_connector_nodes_health_idx
    ON public.campus_connector_nodes (status, last_heartbeat_at, id);

COMMENT ON TABLE public.campus_connector_nodes IS
    'Approved outbound connector identities; private keys and upstream secrets remain in the connector environment';

CREATE TABLE public.campus_connector_school_operations (
    node_id character varying(36) NOT NULL
        REFERENCES public.campus_connector_nodes(id) ON DELETE CASCADE,
    school_id bigint NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    operation_key text NOT NULL,
    operation_type text NOT NULL,
    adapter_id text NOT NULL,
    adapter_version text NOT NULL,
    upstream_protocol text NOT NULL,
    target_host text NOT NULL,
    target_port integer NOT NULL,
    target_tls_server_name text,
    allowlisted_attributes text[] NOT NULL DEFAULT '{}',
    timeout_milliseconds integer NOT NULL DEFAULT 5000,
    max_concurrency integer NOT NULL DEFAULT 2,
    rate_limit_per_minute integer NOT NULL DEFAULT 30,
    enabled boolean NOT NULL DEFAULT false,
    validation_status text NOT NULL DEFAULT 'pending',
    health_status text NOT NULL DEFAULT 'unknown',
    health_code text,
    health_checked_at timestamp with time zone,
    config_revision bigint NOT NULL DEFAULT 1,
    validated_at timestamp with time zone,
    validated_by_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (node_id, school_id, operation_key),
    CONSTRAINT chk_campus_connector_operations_key
        CHECK (operation_key ~ '^[a-z][a-z0-9_.-]{1,127}$'),
    CONSTRAINT chk_campus_connector_operations_type
        CHECK (operation_type IN ('school_account_authenticate', 'roster_snapshot_upload')),
    CONSTRAINT chk_campus_connector_operations_adapter
        CHECK (
            adapter_id ~ '^[a-z][a-z0-9_]{1,63}$'
            AND char_length(adapter_version) BETWEEN 1 AND 64
        ),
    CONSTRAINT chk_campus_connector_operations_protocol
        CHECK (upstream_protocol IN (
            'ldaps', 'ldap_starttls', 'ldap_plain_private_network',
            'oracle_tls', 'oracle_ssh_tunnel', 'https'
        )),
    CONSTRAINT chk_campus_connector_operations_target
        CHECK (
            char_length(target_host) BETWEEN 1 AND 255
            AND target_host = btrim(target_host)
            AND target_port BETWEEN 1 AND 65535
            AND (
                (
                    upstream_protocol IN ('ldaps', 'ldap_starttls', 'oracle_tls', 'https')
                    AND target_tls_server_name IS NOT NULL
                    AND char_length(target_tls_server_name) BETWEEN 1 AND 255
                )
                OR (
                    upstream_protocol IN ('ldap_plain_private_network', 'oracle_ssh_tunnel')
                    AND target_tls_server_name IS NULL
                )
            )
        ),
    CONSTRAINT chk_campus_connector_operations_limits
        CHECK (
            timeout_milliseconds BETWEEN 100 AND 120000
            AND max_concurrency BETWEEN 1 AND 64
            AND rate_limit_per_minute BETWEEN 1 AND 10000
        ),
    CONSTRAINT chk_campus_connector_operations_validation
        CHECK (validation_status IN ('pending', 'valid', 'invalid')),
    CONSTRAINT chk_campus_connector_operations_health
        CHECK (health_status IN ('unknown', 'healthy', 'degraded', 'unavailable')),
    CONSTRAINT chk_campus_connector_operations_enabled
        CHECK (
            NOT enabled
            OR (
                validation_status = 'valid'
                AND upstream_protocol IN (
                    'ldaps', 'ldap_starttls', 'ldap_plain_private_network',
                    'oracle_tls', 'oracle_ssh_tunnel', 'https'
                )
            )
        ),
    CONSTRAINT chk_campus_connector_operations_revision
        CHECK (config_revision > 0)
);

CREATE INDEX campus_connector_school_operations_available_idx
    ON public.campus_connector_school_operations (school_id, operation_key, node_id)
    WHERE enabled AND validation_status = 'valid' AND health_status IN ('healthy', 'degraded');

COMMENT ON TABLE public.campus_connector_school_operations IS
    'Exact school, operation, protocol, host, and port allowlist; no arbitrary URL, TCP forwarding, proxy, SQL, or shell operation is representable';

CREATE TABLE public.campus_connector_requests (
    id character varying(36) PRIMARY KEY,
    request_reference_hash character varying(64) NOT NULL,
    node_id character varying(36) NOT NULL,
    school_id bigint NOT NULL,
    operation_key text NOT NULL,
    request_kind text NOT NULL,
    status text NOT NULL DEFAULT 'started',
    result_code text,
    application_id character varying(36)
        REFERENCES public.student_verification_applications(id) ON DELETE SET NULL,
    roster_snapshot_id character varying(36)
        REFERENCES academic.student_roster_snapshots(id) ON DELETE SET NULL,
    deadline_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    latency_milliseconds integer,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_campus_connector_requests_operation
        FOREIGN KEY (node_id, school_id, operation_key)
        REFERENCES public.campus_connector_school_operations(node_id, school_id, operation_key)
        ON DELETE RESTRICT,
    CONSTRAINT chk_campus_connector_requests_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_campus_connector_requests_reference
        CHECK (request_reference_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_campus_connector_requests_kind
        CHECK (request_kind IN ('interactive_school_account', 'roster_snapshot_push')),
    CONSTRAINT chk_campus_connector_requests_status
        CHECK (status IN ('started', 'succeeded', 'failed', 'timed_out', 'cancelled')),
    CONSTRAINT chk_campus_connector_requests_deadline
        CHECK (deadline_at > created_at),
    CONSTRAINT chk_campus_connector_requests_completion
        CHECK (
            (status = 'started' AND completed_at IS NULL AND latency_milliseconds IS NULL)
            OR (
                status <> 'started'
                AND completed_at IS NOT NULL
                AND latency_milliseconds >= 0
            )
        )
);

CREATE UNIQUE INDEX campus_connector_requests_reference_uidx
    ON public.campus_connector_requests (request_reference_hash);
CREATE INDEX campus_connector_requests_inflight_idx
    ON public.campus_connector_requests (node_id, school_id, operation_key, deadline_at, id)
    WHERE status = 'started';
CREATE INDEX campus_connector_requests_audit_idx
    ON public.campus_connector_requests (school_id, operation_key, created_at DESC, id);

COMMENT ON TABLE public.campus_connector_requests IS
    'Payload-free audit envelope; interactive credentials are transmitted only over the live mTLS request and are never persisted or replayed';

CREATE TABLE public.campus_connector_snapshot_uploads (
    id character varying(36) PRIMARY KEY,
    request_id character varying(36) NOT NULL UNIQUE
        REFERENCES public.campus_connector_requests(id) ON DELETE RESTRICT,
    snapshot_id character varying(36) NOT NULL UNIQUE
        REFERENCES academic.student_roster_snapshots(id) ON DELETE RESTRICT,
    manifest_schema_version integer NOT NULL,
    manifest_checksum character varying(64) NOT NULL,
    signature_key_id text NOT NULL,
    signature_verified boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'received',
    failure_code text,
    received_at timestamp with time zone NOT NULL DEFAULT NOW(),
    verified_at timestamp with time zone,
    imported_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_campus_connector_snapshot_uploads_id
        CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    CONSTRAINT chk_campus_connector_snapshot_uploads_manifest
        CHECK (
            manifest_schema_version > 0
            AND manifest_checksum ~ '^[0-9a-f]{64}$'
            AND char_length(signature_key_id) BETWEEN 1 AND 128
        ),
    CONSTRAINT chk_campus_connector_snapshot_uploads_status
        CHECK (status IN ('received', 'verified', 'imported', 'rejected')),
    CONSTRAINT chk_campus_connector_snapshot_uploads_progress
        CHECK (
            (status = 'received' AND verified_at IS NULL AND imported_at IS NULL)
            OR (
                status = 'verified'
                AND signature_verified
                AND verified_at IS NOT NULL
                AND imported_at IS NULL
            )
            OR (
                status = 'imported'
                AND signature_verified
                AND verified_at IS NOT NULL
                AND imported_at IS NOT NULL
            )
            OR (status = 'rejected' AND failure_code IS NOT NULL)
        )
);

CREATE INDEX campus_connector_snapshot_uploads_status_idx
    ON public.campus_connector_snapshot_uploads (status, received_at, id);

CREATE TABLE public.campus_connector_node_events (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    node_id character varying(36) NOT NULL
        REFERENCES public.campus_connector_nodes(id) ON DELETE RESTRICT,
    event_type text NOT NULL,
    event_code text NOT NULL,
    reason text,
    actor_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    revision bigint NOT NULL,
    occurred_at timestamp with time zone NOT NULL DEFAULT NOW(),
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_campus_connector_node_events_type
        CHECK (event_type IN (
            'registered',
            'activated',
            'health_changed',
            'certificate_rotated',
            'revoked',
            'configuration_changed'
        )),
    CONSTRAINT chk_campus_connector_node_events_code
        CHECK (char_length(event_code) BETWEEN 1 AND 100),
    CONSTRAINT chk_campus_connector_node_events_reason
        CHECK (reason IS NULL OR char_length(reason) BETWEEN 4 AND 500),
    CONSTRAINT chk_campus_connector_node_events_revision
        CHECK (revision > 0)
);

CREATE INDEX campus_connector_node_events_node_idx
    ON public.campus_connector_node_events (node_id, occurred_at DESC, id);
