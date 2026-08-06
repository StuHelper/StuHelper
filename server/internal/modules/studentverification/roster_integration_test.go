package studentverification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestFullRosterImportStagesEncryptedVersionAndIsIdempotent(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-roster-integration-key")
	configureRosterImport(t, fixture, now)
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithRosterCipher(newPhoneTestCipher(t), 1),
	)
	require.NoError(t, err)

	input := validRosterImportInput(now)
	snapshot, err := service.ImportFullRoster(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "ready", snapshot.Status)
	assert.False(t, snapshot.IsCurrent)
	assert.Equal(t, int64(2), snapshot.RowCount)
	assert.Equal(t, int64(1), snapshot.EligibleRowCount)
	assert.NotEmpty(t, snapshot.Checksum)
	assert.NotEmpty(t, snapshot.QualityChecks)

	var (
		studentCipher []byte
		nameCipher    []byte
		docCipher     []byte
		phoneCipher   []byte
		studentHash   string
		nameHash      string
		docHash       string
		phoneHash     string
	)
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT student_id_enc, name_enc, document_number_enc, phone_enc,
		       student_id_hash, name_hash, document_number_hash, phone_hash
		FROM academic.student_roster_records
		WHERE snapshot_id = $1 AND eligibility_status = 'eligible'
	`, snapshot.ID).Scan(
		&studentCipher, &nameCipher, &docCipher, &phoneCipher,
		&studentHash, &nameHash, &docHash, &phoneHash,
	))
	assert.NotContains(t, string(studentCipher), "20990001")
	assert.NotContains(t, string(nameCipher), "张三")
	assert.NotContains(t, string(docCipher), "11010519491231002X")
	assert.NotContains(t, string(phoneCipher), "13800138000")
	assert.Len(t, studentHash, 64)
	assert.Len(t, nameHash, 64)
	assert.Len(t, docHash, 64)
	assert.Len(t, phoneHash, 64)

	replayed, err := service.ImportFullRoster(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, snapshot.ID, replayed.ID)
	var snapshotCount int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM academic.student_roster_snapshots
		WHERE school_id = $1 AND source_kind = $2 AND source_version = $3
	`, testSchoolID, input.SourceKind, input.SourceVersion).Scan(&snapshotCount))
	assert.Equal(t, 1, snapshotCount)

	changed := input
	changed.Records = append([]RosterImportRecord(nil), input.Records...)
	changed.Records[0].Name = "李四"
	_, err = service.ImportFullRoster(ctx, changed)
	assert.ErrorIs(t, err, ErrRosterSourceConflict)
}

func TestFullRosterImportQualityFailurePreservesNoQueryableRows(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-roster-integration-key")
	configureRosterImport(t, fixture, now)
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithRosterCipher(newPhoneTestCipher(t), 1),
	)
	require.NoError(t, err)

	input := validRosterImportInput(now)
	input.SourceVersion = "duplicate-student-id"
	input.Records[1].StudentID = input.Records[0].StudentID
	snapshot, err := service.ImportFullRoster(ctx, input)
	require.ErrorIs(t, err, ErrRosterQualityFailed)
	require.NotNil(t, snapshot)
	assert.Equal(t, "failed", snapshot.Status)
	assert.Equal(t, int64(0), snapshot.RowCount)
	require.Len(t, snapshot.QualityChecks, 1)
	assert.Equal(t, "student_id.unique", snapshot.QualityChecks[0].CheckKey)
	assert.Equal(t, "failed", snapshot.QualityChecks[0].Status)

	var activeCount int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM academic.student_roster_active WHERE school_id = $1
	`, testSchoolID).Scan(&activeCount))
	assert.Zero(t, activeCount)
}

func TestRosterAutoActivationIsExplicitAndUsesSystemActor(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	configureRosterImport(t, fixture, now)
	service, err := NewService(
		NewRepository(fixture.DB),
		[]byte("student-verification-roster-auto-activation-key"),
		WithClock(func() time.Time { return now }),
		WithRosterCipher(newPhoneTestCipher(t), 1),
	)
	require.NoError(t, err)

	snapshot, err := service.ImportFullRoster(ctx, validRosterImportInput(now))
	require.NoError(t, err)

	// Default is fail closed: connector delivery succeeds, but the ready
	// snapshot remains isolated until the owner explicitly enables automation.
	snapshot, err = service.AutoActivateImportedRosterSnapshot(ctx, testSchoolCode, snapshot.ID)
	require.NoError(t, err)
	assert.Equal(t, "ready", snapshot.Status)
	assert.False(t, snapshot.IsCurrent)

	_, err = fixture.Pool.Exec(ctx, `
		UPDATE school_verification_profiles
		SET snapshot_auto_activate = true
		WHERE school_id = $1
	`, testSchoolID)
	require.NoError(t, err)

	snapshot, err = service.AutoActivateImportedRosterSnapshot(ctx, testSchoolCode, snapshot.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", snapshot.Status)
	assert.True(t, snapshot.IsCurrent)

	var actorType, actorUserID string
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT actor_type, actor_user_id
		FROM audit_events
		WHERE resource_type = 'student_roster_snapshot'
		  AND resource_id = $1
	`, snapshot.ID).Scan(&actorType, &actorUserID))
	assert.Equal(t, "system", actorType)
	assert.Empty(t, actorUserID)
}

func TestRosterActivationAndRollbackAtomicallyReevaluateDependentCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-roster-integration-key")
	userID := seedVerificationUser(t, fixture, "roster-switch")
	configureRealNameMethod(t, fixture, now)
	configureRosterImport(t, fixture, now)
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithRosterCipher(newPhoneTestCipher(t), 1),
	)
	require.NoError(t, err)

	first, err := service.ImportFullRoster(ctx, validRosterImportInput(now))
	require.NoError(t, err)
	first, err = service.ActivateRosterSnapshot(ctx, RosterSnapshotSwitchInput{
		SchoolCode: testSchoolCode, SnapshotID: first.ID, ActorUserID: userID,
		Reason: "initial verified roster activation",
	})
	require.NoError(t, err)
	assert.Equal(t, "active", first.Status)
	assert.True(t, first.IsCurrent)
	require.NotNil(t, first.ActivationRevision)
	assert.Equal(t, int64(1), *first.ActivationRevision)

	application, err := service.CreateApplication(ctx, CreateApplicationInput{
		UserID: userID, SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	application, err = service.VerifyRealName(ctx, VerifyRealNameInput{
		UserID: userID, ApplicationID: application.ID,
		StudentID: "20990001", Name: "张三", DocumentNumber: "11010519491231002X",
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, application.Credential)

	secondInput := validRosterImportInput(now.Add(time.Minute))
	secondInput.SourceVersion = "fixture-2026-08-05T00:01:00Z"
	secondInput.Records = secondInput.Records[1:]
	second, err := service.ImportFullRoster(ctx, secondInput)
	require.NoError(t, err)
	assert.Equal(t, int64(1), second.DeletedRowCount)
	second, err = service.ActivateRosterSnapshot(ctx, RosterSnapshotSwitchInput{
		SchoolCode: testSchoolCode, SnapshotID: second.ID, ActorUserID: userID,
		Reason: "activate next complete roster",
	})
	require.NoError(t, err)
	assert.True(t, second.IsCurrent)
	require.NotNil(t, second.ActivationRevision)
	assert.Equal(t, int64(2), *second.ActivationRevision)

	credential, err := service.GetApplication(ctx, userID, application.ID)
	require.NoError(t, err)
	require.NotNil(t, credential.Credential)
	assert.Equal(t, CredentialReviewRequired, credential.Credential.Status)
	eligibility, err := service.GetEligibility(ctx, userID, testSchoolCode)
	require.NoError(t, err)
	assert.False(t, eligibility.Eligible)

	rolledBack, err := service.RollbackRosterSnapshot(ctx, RosterSnapshotSwitchInput{
		SchoolCode: testSchoolCode, SnapshotID: first.ID, ActorUserID: userID,
		Reason: "restore prior roster after incident",
	})
	require.NoError(t, err)
	assert.Equal(t, "active", rolledBack.Status)
	assert.True(t, rolledBack.IsCurrent)
	require.NotNil(t, rolledBack.ActivationRevision)
	assert.Equal(t, int64(3), *rolledBack.ActivationRevision)

	credential, err = service.GetApplication(ctx, userID, application.ID)
	require.NoError(t, err)
	require.NotNil(t, credential.Credential)
	assert.Equal(t, CredentialActive, credential.Credential.Status)
	eligibility, err = service.GetEligibility(ctx, userID, testSchoolCode)
	require.NoError(t, err)
	assert.True(t, eligibility.Eligible)

	var auditCount int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		WHERE resource_type = 'student_roster_snapshot'
		  AND scope_school_id = $1
	`, testSchoolCode).Scan(&auditCount))
	assert.Equal(t, 3, auditCount)
}

func configureRosterImport(t *testing.T, fixture *postgresfixture.Fixture, now time.Time) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_profiles
		SET adapter_id = 'buaa', adapter_version = '1',
		    enabled = true, validation_status = 'valid', validated_at = $2,
		    enrollment_policy = '{
		      "mainlandDocumentTypes":["1"],
		      "rosterKnownEligibilityCodes":["CURRENT","INELIGIBLE"],
		      "rosterEligibleCodes":["CURRENT"],
		      "rosterMinimumRows":1,
		      "rosterMaximumRowDeltaRatio":0.75,
		      "rosterRequireCurrentMarker":true
		    }'::jsonb,
		    updated_at = $2
		WHERE school_id = $1
	`, testSchoolID, now)
	require.NoError(t, err)
}

func validRosterImportInput(now time.Time) FullRosterImportInput {
	current := true
	historical := false
	firstYear := 2023
	secondYear := 2019
	return FullRosterImportInput{
		SchoolCode: testSchoolCode, SourceKind: "fixture",
		SourceVersion: "fixture-2026-08-05T00:00:00Z", MappingVersion: "buaa-test-v1",
		SourceCutoffAt: now,
		Records: []RosterImportRecord{
			{
				StudentID: "20990001", Name: "张三", DocumentType: "1",
				DocumentNumber: "11010519491231002X", Phone: "13800138000",
				StudentStatus: "registered", OnCampusStatus: "on_campus",
				RegistrationStatus: "registered", EducationLevel: "undergraduate",
				StudentCategory: "domestic", EnrollmentYear: &firstYear,
				CurrentMarker: &current, EligibilityCode: "CURRENT", SourceUpdatedAt: &now,
			},
			{
				StudentID: "19370001", Name: "李四", StudentStatus: "graduated",
				EnrollmentYear: &secondYear, CurrentMarker: &historical,
				EligibilityCode: "INELIGIBLE", SourceUpdatedAt: &now,
			},
		},
	}
}
