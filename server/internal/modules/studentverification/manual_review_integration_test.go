package studentverification

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

type fakeManualReviewMaterialStore struct {
	objects map[string][]byte
}

func newFakeManualReviewMaterialStore() *fakeManualReviewMaterialStore {
	return &fakeManualReviewMaterialStore{objects: make(map[string][]byte)}
}

func (s *fakeManualReviewMaterialStore) PutManualReviewMaterial(
	_ context.Context,
	objectKey string,
	content []byte,
	_ string,
) error {
	s.objects[objectKey] = append([]byte(nil), content...)
	return nil
}

func (s *fakeManualReviewMaterialStore) DeleteManualReviewMaterial(
	_ context.Context,
	objectKey string,
) error {
	delete(s.objects, objectKey)
	return nil
}

func (*fakeManualReviewMaterialStore) GetManualReviewMaterialURL(
	_ context.Context,
	objectKey string,
) (string, error) {
	return "https://objects.example.test/private/" + objectKey, nil
}

func TestLegacyObjectPurgeDeletesPrivateObjectAndQueueRow(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	store := newFakeManualReviewMaterialStore()
	objectKey := "identity/legacy/front.png"
	store.objects[objectKey] = []byte("legacy-private-object")

	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO student_verification_object_purge_queue (
			object_key, source_kind, available_at, created_at, updated_at
		)
		VALUES ($1, 'legacy_identity_photo', $2, $2, $2)
	`, objectKey, now)
	require.NoError(t, err)

	service, err := NewService(
		NewRepository(fixture.DB),
		[]byte("legacy-object-purge-test-key"),
		WithClock(func() time.Time { return now }),
		WithManualReviewMaterialStore(store),
	)
	require.NoError(t, err)
	service.processLegacyObjectPurgeBatch(ctx, "00000000-0000-4000-8000-000000000032")

	assert.NotContains(t, store.objects, objectKey)
	var remaining int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM student_verification_object_purge_queue
		WHERE object_key = $1
	`, objectKey).Scan(&remaining))
	assert.Zero(t, remaining)
}

func TestManualReviewRequiresIndependentReviewerAndCreatesExpiringCredential(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-manual-review-key")
	applicantID := seedVerificationUser(t, fixture, "manual-applicant")
	reviewerID := seedVerificationUser(t, fixture, "manual-reviewer")
	configureManualReviewMethod(t, fixture, now, false)
	store := newFakeManualReviewMaterialStore()
	service := newManualReviewTestService(t, fixture, key, now, store)

	application, err := service.CreateApplication(ctx, CreateApplicationInput{
		UserID: applicantID, SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	reviewCase, err := service.UpsertManualReview(ctx, UpsertManualReviewInput{
		UserID: applicantID, ApplicationID: application.ID,
		MaterialType: ManualMaterialStudentCard,
		FormValues: map[string]string{
			"department": "计算机学院",
			"studentID":  "20990001",
			"name":       "张三",
			"email":      "20990001@buaa.edu.cn",
		},
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)
	assert.Equal(t, ManualReviewDraft, reviewCase.Status)
	assert.Equal(t, "20****01", reviewCase.StudentIDMasked)
	assert.NotContains(t, reviewCase.ApplicantNameMasked, "张三")

	reviewCase, err = service.UploadManualCameraCapture(ctx, ManualCameraCaptureInput{
		UserID: applicantID, ApplicationID: application.ID,
		ContentType: "image/png", ImageBase64: manualTestImageBase64(t),
		CaptureSource: "web_camera", RequestedFacingMode: "environment",
	})
	require.NoError(t, err)
	require.Len(t, reviewCase.Materials, 1)
	assert.NotEmpty(t, store.objects)
	serializedCase, err := json.Marshal(reviewCase)
	require.NoError(t, err)
	assert.NotContains(t, string(serializedCase), "student-verification/manual/",
		"object key must not appear in serialized/applicant view")

	reviewCase, err = service.SubmitManualReview(ctx, applicantID, application.ID, true)
	require.NoError(t, err)
	assert.Equal(t, ManualReviewPending, reviewCase.Status)

	_, err = service.DecideManualReview(ctx, ManualReviewDecisionInput{
		CaseID: reviewCase.ID, ReviewerUserID: applicantID,
		Action: ManualDecisionApprove, UserVisibleReason: "材料清晰且有效",
		ExpiresInDays: intPointer(90),
	})
	require.ErrorIs(t, err, ErrManualReviewSelfDecision)

	reviewCase, err = service.DecideManualReview(ctx, ManualReviewDecisionInput{
		CaseID: reviewCase.ID, ReviewerUserID: reviewerID,
		Action: ManualDecisionApprove, UserVisibleReason: "材料清晰且有效",
		InternalRiskNote: "仅供本次测试的受保护备注",
		ExpiresInDays:    intPointer(90),
	})
	require.NoError(t, err)
	assert.Equal(t, ManualReviewApproved, reviewCase.Status)
	assert.Equal(t, "formal_student", *reviewCase.CredentialClass)
	require.NotNil(t, reviewCase.CredentialExpiresAt)
	assert.True(t, reviewCase.CredentialExpiresAt.Equal(now.AddDate(0, 0, 90)))

	eligibility, err := service.GetEligibility(ctx, applicantID, testSchoolCode)
	require.NoError(t, err)
	assert.True(t, eligibility.Eligible)
	assert.Equal(t, []Method{MethodManualMaterialReview}, eligibility.CredentialMethods)

	access, schoolCode, err := service.GetManualMaterialAccess(
		ctx, reviewCase.ID, reviewCase.Materials[0].ID, reviewerID,
	)
	require.NoError(t, err)
	assert.Equal(t, testSchoolCode, schoolCode)
	assert.Contains(t, access.URL, "/private/student-verification/manual/")
	assert.Equal(t, now.Add(5*time.Minute), access.ExpiresAt)

	var encryptedForm []byte
	var internalNote []byte
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT form_data_enc, internal_risk_note_enc
		FROM student_manual_review_cases WHERE id = $1
	`, reviewCase.ID).Scan(&encryptedForm, &internalNote))
	assert.NotContains(t, string(encryptedForm), "20990001")
	assert.NotContains(t, string(encryptedForm), "张三")
	assert.NotContains(t, string(internalNote), "受保护备注")

	var accessEvents int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_manual_review_events
		WHERE case_id = $1 AND action = 'material_accessed'
	`, reviewCase.ID).Scan(&accessEvents))
	assert.Equal(t, 1, accessEvents)

	_, err = fixture.Pool.Exec(ctx, `
		UPDATE student_manual_review_materials
		SET created_at = $2, retention_until = $3
		WHERE id = $1
	`, reviewCase.Materials[0].ID, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	require.NoError(t, err)
	deleted, err := service.CleanupExpiredManualReviewMaterials(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	assert.Empty(t, store.objects)
	var materialStatus string
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT status FROM student_manual_review_materials WHERE id = $1
	`, reviewCase.Materials[0].ID).Scan(&materialStatus))
	assert.Equal(t, "deleted", materialStatus)
}

func TestManualCameraHandoffIsApplicationBoundAndContinuationIsSingleChoice(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-manual-handoff-key")
	applicantID := seedVerificationUser(t, fixture, "manual-handoff")
	configureManualReviewMethod(t, fixture, now, false)
	store := newFakeManualReviewMaterialStore()
	service := newManualReviewTestService(t, fixture, key, now, store)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{
		UserID: applicantID, SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	_, err = service.UpsertManualReview(ctx, UpsertManualReviewInput{
		UserID: applicantID, ApplicationID: application.ID,
		MaterialType: ManualMaterialAdmissionNotice,
		FormValues: map[string]string{
			"department": "计算机学院", "studentID": "20990001",
			"name": "张三", "email": "20990001@buaa.edu.cn",
		},
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)

	handoff, err := service.CreateManualCameraHandoff(ctx, applicantID, application.ID)
	require.NoError(t, err)
	assert.Equal(t, ManualHandoffPending, handoff.Status)
	const prefix = "https://stuhelper.example.test/student-verification/manual-camera/"
	require.True(t, strings.HasPrefix(handoff.MobileURL, prefix))
	token := strings.TrimPrefix(handoff.MobileURL, prefix)
	require.NotEmpty(t, token)

	uploaded, err := service.UploadManualHandoffCameraCapture(ctx, ManualCameraCaptureInput{
		Token: token, ContentType: "image/png", ImageBase64: manualTestImageBase64(t),
		CaptureSource: "web_camera", RequestedFacingMode: "environment",
	})
	require.NoError(t, err)
	assert.Equal(t, ManualHandoffUploaded, uploaded.Status)
	require.NotNil(t, uploaded.Material)

	replayed, err := service.UploadManualHandoffCameraCapture(ctx, ManualCameraCaptureInput{
		Token: token, ContentType: "image/png", ImageBase64: manualTestImageBase64(t),
		CaptureSource: "web_camera", RequestedFacingMode: "environment",
	})
	require.NoError(t, err)
	assert.Equal(t, uploaded.Material.ID, replayed.Material.ID)
	assert.Len(t, store.objects, 1)

	chosen, err := service.ChooseManualCameraContinuation(ctx, token, "mobile")
	require.NoError(t, err)
	assert.Equal(t, ManualHandoffLocked, chosen.Status)
	assert.Equal(t, "mobile", *chosen.ContinueOn)
	resumed, err := service.ResumeManualCameraHandoff(ctx, applicantID, token)
	require.NoError(t, err)
	assert.Equal(t, application.ID, resumed.ID)
	otherUserID := seedVerificationUser(t, fixture, "manual-handoff-other-user")
	_, err = service.ResumeManualCameraHandoff(ctx, otherUserID, token)
	require.ErrorIs(t, err, ErrManualHandoffNotFound)
	_, err = service.ChooseManualCameraContinuation(ctx, token, "desktop")
	require.ErrorIs(t, err, ErrManualHandoffState)
	_, err = service.PreviewManualCameraHandoff(ctx, token)
	require.NoError(t, err)
}

func TestManualReviewEmailOTPIsAuxiliaryAndNeverAutoApproves(t *testing.T) {
	fixture := postgresfixture.Start(t)
	redisServer := redisfixture.Start(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	key := []byte("student-verification-manual-email-key")
	applicantID := seedVerificationUser(t, fixture, "manual-email")
	configureManualReviewMethod(t, fixture, now, true)
	store := newFakeManualReviewMaterialStore()
	sender := &capturingStudentEmailSender{}
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithRosterCipher(newPhoneTestCipher(t), 1),
		WithManualReviewMaterialStore(store),
		WithManualReviewPublicBaseURL("https://stuhelper.example.test"),
		WithManualReviewMaterialAccessTTL(5*time.Minute),
		WithRedisClient(redisServer.Client),
		WithStudentEmailSender(sender),
		WithOTPGenerator(func() (string, error) { return "246810", nil }),
	)
	require.NoError(t, err)
	application, err := service.CreateApplication(ctx, CreateApplicationInput{
		UserID: applicantID, SchoolCode: testSchoolCode,
	})
	require.NoError(t, err)
	_, err = service.UpsertManualReview(ctx, UpsertManualReviewInput{
		UserID: applicantID, ApplicationID: application.ID,
		MaterialType: ManualMaterialStudentCard,
		FormValues: map[string]string{
			"department": "计算机学院", "studentID": "20990001",
			"name": "张三", "email": "20990001@buaa.edu.cn",
		},
		PrivacyNoticeVersion: "2026-08-05", SensitiveDataConsent: true,
	})
	require.NoError(t, err)

	challenge, err := service.RequestManualReviewEmailOTP(ctx, applicantID, application.ID)
	require.NoError(t, err)
	assert.Equal(t, "20990001@buaa.edu.cn", sender.email)
	assert.Equal(t, "246810", sender.code)
	assert.Equal(t, "20****01@buaa.edu.cn", challenge.MaskedEmail)
	reviewCase, err := service.VerifyManualReviewEmailOTP(
		ctx, applicantID, application.ID, "246810",
	)
	require.NoError(t, err)
	assert.True(t, reviewCase.EmailVerified)
	assert.True(t, reviewCase.EmailVerificationRequired)
	assert.Equal(t, ManualReviewDraft, reviewCase.Status)

	credentials, err := service.ListCredentials(ctx, applicantID)
	require.NoError(t, err)
	assert.Empty(t, credentials)
	eligibility, err := service.GetEligibility(ctx, applicantID, testSchoolCode)
	require.NoError(t, err)
	assert.False(t, eligibility.Eligible)
}

func newManualReviewTestService(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	key []byte,
	now time.Time,
	store ManualReviewMaterialStore,
) *Service {
	t.Helper()
	service, err := NewService(
		NewRepository(fixture.DB), key,
		WithClock(func() time.Time { return now }),
		WithRosterCipher(newPhoneTestCipher(t), 1),
		WithManualReviewMaterialStore(store),
		WithManualReviewPublicBaseURL("https://stuhelper.example.test"),
		WithManualReviewMaterialAccessTTL(5*time.Minute),
	)
	require.NoError(t, err)
	return service
}

func configureManualReviewMethod(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	now time.Time,
	requireEmail bool,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_profiles
		SET enabled = true, validation_status = 'valid', validated_at = $2,
		    updated_at = $2
		WHERE school_id = $1
	`, testSchoolID, now)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		UPDATE school_verification_methods
		SET enabled = true, validation_status = 'valid', validated_at = $2,
		    health_status = 'healthy', health_checked_at = $2,
		    privacy_notice_version = '2026-08-05',
		    privacy_notice = '{
		      "title":"人工材料审核",
		      "summary":"授权审核员将查看提交的学生材料",
		      "dataCategories":["学校表单","学生材料"],
		      "retentionSummary":"材料按学校策略限期保留"
		    }'::jsonb,
		    risk_policy = jsonb_build_object(
		      'maxMaterialBytes', 10485760,
		      'maxMaterials', 3,
		      'materialRetentionDays', 180,
		      'handoffTTLSeconds', 1800,
		      'reviewWindowSeconds', 604800,
		      'requireEmailVerification', $3::boolean,
		      'admissionNoticeMaxCredentialDays', 180
		    ),
		    updated_at = $2
		WHERE school_id = $1 AND method = 'manual_material_review'
	`, testSchoolID, now, requireEmail)
	require.NoError(t, err)
}

func manualTestImageBase64(t *testing.T) string {
	t.Helper()
	imageValue := image.NewRGBA(image.Rect(0, 0, 320, 320))
	for y := 0; y < 320; y++ {
		for x := 0; x < 320; x++ {
			imageValue.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, imageValue))
	return base64.StdEncoding.EncodeToString(encoded.Bytes())
}

func intPointer(value int) *int { return &value }
