package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStudentVerificationDomainMigrationKeepsIdentityAndAdmissionDecoupled(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000026_student_verification_domain.up.sql")
	applicationDDL := migrationTableDDL(t, up, "public.student_verification_applications")

	for _, expected := range []string{
		"user_id bigint NOT NULL",
		"school_id bigint NOT NULL",
		"privacy_notice_version text",
		"continuation_hash character varying(64)",
		"revision bigint NOT NULL",
	} {
		assert.Contains(t, applicationDDL, expected)
	}
	for _, forbidden := range []string{"qq_id", "guild_id", "admission_session", "redirect_url"} {
		assert.NotContains(t, applicationDDL, forbidden)
	}

	assert.Contains(t, up, "student_enrollment_subjects_active_subject_uidx")
	assert.Contains(t, up, "student_subject_conflicts")
	assert.Contains(t, up, "student_eligibility_revisions")
	assert.Contains(t, up, "student eligibility is computed from current facts")
	assert.NotContains(t, up, "student_verified boolean")
}

func TestRosterSnapshotMigrationHasAtomicPointerAndSecurePairs(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000027_student_roster_snapshots.up.sql")
	recordDDL := migrationTableDDL(t, up, "academic.student_roster_records")

	for _, expected := range []string{
		"CREATE TABLE academic.student_roster_active",
		"PRIMARY KEY REFERENCES public.schools",
		"activation_revision bigint NOT NULL",
		"student_id_enc bytea NOT NULL",
		"student_id_hash character varying(64) NOT NULL",
		"name_enc bytea NOT NULL",
		"name_hash character varying(64) NOT NULL",
		"document_number_enc bytea",
		"document_number_hash character varying(64)",
		"phone_enc bytea",
		"phone_hash character varying(64)",
		"chk_student_roster_records_document_pair",
		"chk_student_roster_records_phone_pair",
	} {
		assert.Contains(t, up, expected)
	}
	for _, forbidden := range []string{
		"student_id text",
		"student_id character varying",
		"name text",
		"phone text",
		"document_number text",
	} {
		assert.NotContains(t, recordDDL, forbidden)
	}
	assert.Contains(t, up, "UNIQUE (school_id, source_kind, source_version)")
}

func TestRosterAutoActivationDefaultsFailClosed(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000034_roster_snapshot_auto_activation.up.sql")
	assert.Contains(t, up, "snapshot_auto_activate boolean NOT NULL DEFAULT false")
	assert.NotContains(t, up, "DEFAULT true")
}

func TestPhoneDomainMigrationKeepsCasdoorAuthoritativeAndOutboxValueFree(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000028_phone_verification_domain.up.sql")
	outboxDDL := migrationTableDDL(t, up, "public.phone_binding_outbox")

	for _, expected := range []string{
		"phone_projection_state",
		"phone_verification_credentials",
		"school_roster_phone_match",
		"sms_possession",
		"phone_number_claims",
		"phone_eligibility_revisions",
		"payload is resolved by operation ID",
	} {
		assert.Contains(t, up, expected)
	}
	for _, forbidden := range []string{"phone_enc", "phone_hash", "payload", "provider_response"} {
		assert.NotContains(t, outboxDDL, forbidden)
	}
	assert.Contains(t, up, "Casdoor remains the only writable authority")
}

func TestCampusConnectorMigrationCannotRepresentGenericNetworkAccess(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000029_campus_connector_registry.up.sql")
	requestDDL := migrationTableDDL(t, up, "public.campus_connector_requests")

	for _, expected := range []string{
		"school_account_authenticate",
		"roster_snapshot_upload",
		"'ldap_plain_private_network'",
		"'oracle_ssh_tunnel'",
		"target_host text NOT NULL",
		"target_port integer NOT NULL",
		"upstream_protocol IN ('ldaps', 'ldap_starttls', 'oracle_tls', 'https')",
		"AND target_tls_server_name IS NOT NULL",
		"upstream_protocol IN ('ldap_plain_private_network', 'oracle_ssh_tunnel')",
		"AND target_tls_server_name IS NULL",
		"certificate_fingerprint",
		"signing_public_key",
		"Payload-free audit envelope",
	} {
		assert.Contains(t, up, expected)
	}
	for _, forbidden := range []string{"password", "request_payload", "response_payload", "sql text", "shell", "proxy_url"} {
		assert.NotContains(t, requestDDL, forbidden)
	}
}

func TestManualReviewMigrationKeepsEvidencePrivateAndWorkflowIndependent(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000030_student_manual_review.up.sql")
	caseDDL := migrationTableDDL(t, up, "public.student_manual_review_cases")
	materialDDL := migrationTableDDL(t, up, "public.student_manual_review_materials")
	eventDDL := migrationTableDDL(t, up, "public.student_manual_review_events")

	for _, expected := range []string{
		"form_data_enc bytea NOT NULL",
		"form_digest character varying(64) NOT NULL",
		"student_id_hash character varying(64) NOT NULL",
		"internal_risk_note_enc bytea",
		"capture_source text NOT NULL DEFAULT 'web_camera'",
		"requested_facing_mode text NOT NULL DEFAULT 'environment'",
		"token_hash character varying(64) NOT NULL UNIQUE",
		"token_enc bytea",
		"Payload-free workflow history",
	} {
		assert.Contains(t, up, expected)
	}
	for _, forbidden := range []string{"qq_id", "guild_id", "group_admission", "phone"} {
		assert.NotContains(t, caseDDL, forbidden)
	}
	for _, forbidden := range []string{"public_url", "presigned_url", "file_name", "device_attested"} {
		assert.NotContains(t, materialDDL, forbidden)
	}
	for _, forbidden := range []string{"form_data", "object_key", "material_url", "internal_risk_note"} {
		assert.NotContains(t, eventDDL, forbidden)
	}
}

func TestVerificationMigrationsRefuseDestructiveRollbackAfterCutoverFacts(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"000026_student_verification_domain.down.sql",
		"000027_student_roster_snapshots.down.sql",
		"000028_phone_verification_domain.down.sql",
		"000029_campus_connector_registry.down.sql",
		"000030_student_manual_review.down.sql",
	} {
		down := readServerFile(t, "migrations", name)
		assert.Contains(t, down, "refusing destructive rollback", name)
	}
}

func TestAdmissionEligibilityMigrationFencesReleaseAndRebuildsActiveUniqueness(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000031_admission_eligibility_revision_fence.up.sql")
	down := readServerFile(t, "migrations", "000031_admission_eligibility_revision_fence.down.sql")

	for _, expected := range []string{
		"eligibility_revision bigint",
		"requirements_status text",
		"claim_owner character varying(36)",
		"chk_admission_release_action_revision",
		"status = 'stale'",
		"CREATE UNIQUE INDEX group_admission_sessions_active_qq_idx",
		"'awaiting_account_link'",
		"'awaiting_requirements'",
		"'pending_manual_review'",
		"'eligible'",
		"'action_pending'",
		"'admitted'",
	} {
		assert.Contains(t, up, expected)
	}
	assert.NotContains(t, up, "student_verified boolean")
	assert.Contains(t, down, "revision columns are intentionally retained")
	assert.NotContains(t, down, "DROP COLUMN IF EXISTS eligibility_revision")
}

func TestLegacyStudentVerificationPurgeIsExplicitAndPreservesSeparateDomains(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000032_purge_legacy_student_verification.up.sql")
	down := readServerFile(t, "migrations", "000032_purge_legacy_student_verification.down.sql")

	for _, expected := range []string{
		"LOCK TABLE",
		"IN ACCESS EXCLUSIVE MODE",
		"student_verification_object_purge_queue",
		"Transient deletion queue only",
		"DELETE FROM public.user_verification_credentials",
		"DELETE FROM public.student_email_inbound_events",
		"DELETE FROM public.student_manual_review_materials",
		"DELETE FROM public.student_verification_attempts",
		"DELETE FROM public.school_verification_suggestions",
		"DELETE FROM public.freshman_camera_handoffs",
		"DELETE FROM public.freshman_verification_materials",
		"DELETE FROM public.freshman_verification_applications",
		"DELETE FROM public.user_identities",
		"DELETE FROM public.user_profiles",
		"TRUNCATE TABLE academic.buaa_students",
		"'freshman_near_expiry'",
		"status = 'stale'",
		"status = 'awaiting_requirements'",
		"reject_retired_student_verification_write",
		"reject_user_profiles_legacy_write",
		"reject_user_identities_legacy_write",
		"reject_freshman_applications_legacy_write",
		"reject_buaa_upsert_roster_legacy_write",
		"reject_unscoped_student_credential_legacy_write",
		"WHEN (NEW.verification_application_id IS NULL)",
	} {
		assert.Contains(t, up, expected)
	}
	for _, forbidden := range []string{
		"DELETE FROM public.users",
		"DELETE FROM public.user_qq_bindings",
		"DELETE FROM public.phone_verification_credentials",
		"TRUNCATE TABLE public.users",
		"TRUNCATE TABLE public.user_qq_bindings",
	} {
		assert.NotContains(t, up, forbidden)
	}
	assert.Contains(t, down, "cannot be restored")
}

func TestCurrentStudentProjectionRejectsUnscopedLegacyCredentials(t *testing.T) {
	t.Parallel()

	up := readServerFile(t, "migrations", "000033_current_student_eligibility_projection.up.sql")
	assert.Contains(t, up, "credential.verification_application_id IS NOT NULL")
}

func migrationTableDDL(t *testing.T, migration string, table string) string {
	t.Helper()
	startMarker := "CREATE TABLE " + table + " ("
	start := strings.Index(migration, startMarker)
	require.NotEqual(t, -1, start, "missing table %s", table)
	rest := migration[start:]
	end := strings.Index(rest, "\n);\n")
	require.NotEqual(t, -1, end, "unterminated table %s", table)
	return rest[:end+4]
}
