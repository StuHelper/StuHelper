package admission

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestFreshmanApplicationRejectsClosedChannelAndDuplicatePending(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-closed")

	svc.now = func() time.Time { return fixedAdmissionNow().AddDate(0, 6, 0) }
	_, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      1,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.ErrorIs(t, err, ErrAdmissionFreshmanChannelClosed)

	svc.now = fixedAdmissionNow
	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      1,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	require.NotEmpty(t, app.ID)

	_, err = svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      1,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.ErrorIs(t, err, ErrAdmissionFreshmanPendingExists)
}

func TestFreshmanCameraCaptureValidatesAndStoresImage(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera")
	app := createFreshmanTestApplication(t, svc, userID)

	_, err := svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "application/pdf",
		ImageBase64:   base64.StdEncoding.EncodeToString([]byte("%PDF")),
	})
	require.ErrorIs(t, err, ErrAdmissionMaterialInvalidType)

	_, err = svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString([]byte("not an image")),
	})
	require.ErrorIs(t, err, ErrAdmissionMaterialInvalidData)

	updated, err := svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)

	assert.Equal(t, FreshmanApplicationPending, updated.Status)
	assert.Equal(t, "image/png", store.contentType)
	assert.NotEmpty(t, store.objectKey)
	assert.NotEmpty(t, store.content)
}

func newFreshmanTestService(t *testing.T, fixture *postgresfixture.Fixture) *Service {
	t.Helper()
	svc := newSessionTestService(t, fixture)
	insertAdmissionSchoolConfig(t, fixture)
	insertAdmissionPolicy(t, fixture)
	return svc
}

func seedLinkedAdmissionUser(t *testing.T, fixture *postgresfixture.Fixture, svc *Service, suffix string) int64 {
	t.Helper()
	userID := seedAdmissionUser(t, fixture, suffix)
	created := createLinkableSession(t, svc)
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:   created.Token,
		QQQuery: "10001",
		UserID:  userID,
	})
	require.NoError(t, err)
	return userID
}

func createFreshmanTestApplication(t *testing.T, svc *Service, userID int64) *FreshmanApplication {
	t.Helper()
	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      1,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	return app
}

type testAdmissionMaterialStore struct {
	objectKey   string
	content     []byte
	contentType string
}

func (s *testAdmissionMaterialStore) PutAdmissionMaterial(
	_ context.Context,
	objectKey string,
	content []byte,
	contentType string,
) error {
	s.objectKey = objectKey
	s.content = content
	s.contentType = contentType
	return nil
}

func (s *testAdmissionMaterialStore) DeleteAdmissionMaterial(_ context.Context, objectKey string) error {
	if s.objectKey == objectKey {
		s.objectKey = ""
		s.content = nil
		s.contentType = ""
	}
	return nil
}

func (s *testAdmissionMaterialStore) GetAdmissionMaterialURL(_ context.Context, objectKey string) (string, error) {
	return "https://materials.example/" + objectKey, nil
}

func insertAdmissionSchoolConfig(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO school_configs (
			school_id, school_name, verification_method, approval_policy, manual_form_fields, enabled
		)
		VALUES (
			1, 'Admission Test University', 'manual', 'auto',
			'{"admission":{"emailDomains":["buaa.edu.cn"],"ssoLoginURL":"https://sso.school.example/login"}}',
			true
		)
		ON CONFLICT (school_id) DO UPDATE
		SET manual_form_fields = EXCLUDED.manual_form_fields,
		    enabled = EXCLUDED.enabled,
		    updated_at = NOW()
	`)
	require.NoError(t, err)
}

func validPNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x62, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
		0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
