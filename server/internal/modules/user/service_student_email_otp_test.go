package user

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestStudentEmailOTPDerivesBUAAEmailAndCreatesVerifiedProfile(t *testing.T) {
	redis := redisfixture.Start(t)
	var captured *Profile
	var capturedCredential *VerificationCredentialProjection
	sender := &testStudentEmailSender{}
	academicTable := "academic.buaa_students"
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return captured, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			assert.Equal(t, int64(4111010006), schoolID)
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolCode:         "4111010006",
				SchoolName:         "北京航空航天大学",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "auto",
				AcademicDBTable:    &academicTable,
				Enabled:            true,
				ManualFormFields: json.RawMessage(
					`{"admission":{"emailDomains":["buaa.edu.cn"],"emailIdentityPolicy":{"type":"academic_student_email","studentIDEmailDomain":"buaa.edu.cn","requireStudentName":true}}}`,
				),
			}, nil
		},
		onGetAcademicStudentByXHFromTable: func(_ context.Context, xh string, tableName string) (*AcademicStudent, error) {
			assert.Equal(t, "20250001", xh)
			assert.Equal(t, academicTable, tableName)
			return &AcademicStudent{XH: xh, XM: stringPtr("张三")}, nil
		},
		onCreateProfile: func(_ context.Context, profile *Profile) error {
			copy := *profile
			captured = &copy
			return nil
		},
		onEnsureVerificationCredentialTx: func(_ context.Context, _ pgx.Tx, credential VerificationCredentialProjection) error {
			copy := credential
			capturedCredential = &copy
			return nil
		},
	}

	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithStudentEmailOTP(redis.Client, sender),
	)
	require.NoError(t, err)

	resp, err := svc.RequestStudentEmailOTP(context.Background(), StudentEmailOTPInput{
		UserID:      7,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: " 张 三 ",
	})
	require.NoError(t, err)
	assert.Equal(t, "20250001@buaa.edu.cn", resp.Email)
	assert.Equal(t, "20250001@buaa.edu.cn", sender.email)

	profile, err := svc.VerifyStudentEmailOTP(context.Background(), StudentEmailOTPVerifyInput{
		UserID:   7,
		SchoolID: 4111010006,
		Email:    "20250001@buaa.edu.cn",
		Code:     sender.code,
		Consent:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.NotNil(t, captured)
	assert.Equal(t, StatusVerified, captured.VerificationStatus)
	require.NotNil(t, captured.VerificationMethod)
	assert.Equal(t, VerifyMethodSchoolEmailOTP, *captured.VerificationMethod)
	assert.Equal(t, []string{"20250001"}, captured.StudentIDs)
	assert.JSONEq(t, `{"schoolEmail":"20250001@buaa.edu.cn","studentID":"20250001","studentName":"张三"}`, string(captured.ManualFormData))
	require.NotNil(t, capturedCredential)
	assert.Equal(t, int64(7), capturedCredential.UserID)
	assert.Equal(t, int64(4111010006), capturedCredential.SchoolID)
	assert.Equal(t, userVerificationCredentialKindSchoolEmailOTP, capturedCredential.Kind)
	assert.NotEmpty(t, capturedCredential.SubjectHash)
	assert.Equal(t, "2******1@buaa.edu.cn", capturedCredential.SubjectDisplay)
	require.NotNil(t, captured.VerifiedAt)
	assert.Equal(t, *captured.VerifiedAt, capturedCredential.VerifiedAt)
}

func TestStudentEmailOTPRejectsBUAAStudentNameMismatch(t *testing.T) {
	redis := redisfixture.Start(t)
	svc, err := NewService(
		buaaStudentEmailOTPRepo(t, &AcademicStudent{XH: "20250001", XM: stringPtr("李四")}),
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithStudentEmailOTP(redis.Client, &testStudentEmailSender{}),
	)
	require.NoError(t, err)

	_, err = svc.RequestStudentEmailOTP(context.Background(), StudentEmailOTPInput{
		UserID:      7,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "张三",
	})

	require.ErrorIs(t, err, ErrStudentNameMismatch)
}

func TestStudentEmailOTPRejectsBUAAAliasEmail(t *testing.T) {
	redis := redisfixture.Start(t)
	sender := &testStudentEmailSender{}
	svc, err := NewService(
		buaaStudentEmailOTPRepo(t, &AcademicStudent{XH: "20250001", XM: stringPtr("张三")}),
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithStudentEmailOTP(redis.Client, sender),
	)
	require.NoError(t, err)

	_, err = svc.RequestStudentEmailOTP(context.Background(), StudentEmailOTPInput{
		UserID:      7,
		SchoolID:    4111010006,
		Email:       "alias@buaa.edu.cn",
		StudentID:   "20250001",
		StudentName: "张三",
	})

	require.ErrorIs(t, err, ErrStudentEmailDomainNotAllowed)
	assert.Empty(t, sender.email)
}

func TestRequestStudentEmailOTPSendFailureKeepsCooldown(t *testing.T) {
	ctx := context.Background()
	redis := redisfixture.Start(t)
	sender := &testStudentEmailSender{err: errors.New("email backend unavailable")}
	svc, err := NewService(
		buaaStudentEmailOTPRepo(t, &AcademicStudent{XH: "20250001", XM: stringPtr("张三")}),
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithStudentEmailOTP(redis.Client, sender),
	)
	require.NoError(t, err)
	input := StudentEmailOTPInput{
		UserID:      7,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "张三",
	}

	_, err = svc.RequestStudentEmailOTP(ctx, input)

	require.ErrorContains(t, err, "email backend unavailable")
	assert.Equal(t, 1, sender.calls)
	_, err = svc.loadStudentEmailOTPRecord(ctx, input.UserID, input.SchoolID)
	require.ErrorIs(t, err, ErrStudentEmailOTPExpired)

	_, err = svc.RequestStudentEmailOTP(ctx, input)

	require.ErrorIs(t, err, ErrStudentEmailOTPCooldown)
	assert.Equal(t, 1, sender.calls)
}

func TestRequestStudentEmailOTPSendFailureCleanupSurvivesRequestCancellation(t *testing.T) {
	redis := redisfixture.Start(t)
	sender := &testStudentEmailSender{err: errors.New("email backend unavailable")}
	svc, err := NewService(
		buaaStudentEmailOTPRepo(t, &AcademicStudent{XH: "20250001", XM: stringPtr("张三")}),
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithStudentEmailOTP(redis.Client, sender),
	)
	require.NoError(t, err)
	input := StudentEmailOTPInput{
		UserID:      7,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "张三",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender.onSend = cancel

	_, err = svc.RequestStudentEmailOTP(ctx, input)

	require.ErrorContains(t, err, "email backend unavailable")
	assert.Equal(t, 1, sender.calls)
	_, err = svc.loadStudentEmailOTPRecord(context.Background(), input.UserID, input.SchoolID)
	require.ErrorIs(t, err, ErrStudentEmailOTPExpired)

	_, err = svc.RequestStudentEmailOTP(context.Background(), input)
	require.ErrorIs(t, err, ErrStudentEmailOTPCooldown)
	assert.Equal(t, 1, sender.calls)
}

func TestVerifyStudentEmailOTPKeepsCodeWhenProfileCommitFails(t *testing.T) {
	ctx := context.Background()
	redis := redisfixture.Start(t)
	sender := &testStudentEmailSender{}
	var captured *Profile
	commitErr := errors.New("profile db unavailable")
	repo := buaaStudentEmailOTPRepo(t, &AcademicStudent{XH: "20250001", XM: stringPtr("张三")})
	repo.onGetProfileByUserID = func(_ context.Context, _ int64) (*Profile, error) {
		return captured, nil
	}
	repo.onCreateProfileTx = func(_ context.Context, _ pgx.Tx, profile *Profile) error {
		if commitErr != nil {
			return commitErr
		}
		copy := *profile
		captured = &copy
		return nil
	}
	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithStudentEmailOTP(redis.Client, sender),
	)
	require.NoError(t, err)
	_, err = svc.RequestStudentEmailOTP(ctx, StudentEmailOTPInput{
		UserID:      7,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "张三",
	})
	require.NoError(t, err)

	_, err = svc.VerifyStudentEmailOTP(ctx, StudentEmailOTPVerifyInput{
		UserID:   7,
		SchoolID: 4111010006,
		Email:    sender.email,
		Code:     sender.code,
		Consent:  true,
	})

	require.ErrorContains(t, err, "profile db unavailable")
	record, err := svc.loadStudentEmailOTPRecord(ctx, 7, 4111010006)
	require.NoError(t, err)
	assert.Equal(t, sender.code, record.Code)

	commitErr = nil
	profile, err := svc.VerifyStudentEmailOTP(ctx, StudentEmailOTPVerifyInput{
		UserID:   7,
		SchoolID: 4111010006,
		Email:    sender.email,
		Code:     sender.code,
		Consent:  true,
	})

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, StatusVerified, profile.VerificationStatus)
	_, err = svc.loadStudentEmailOTPRecord(ctx, 7, 4111010006)
	require.ErrorIs(t, err, ErrStudentEmailOTPExpired)
}

func TestStudentEmailAcademicMatchReturnsImmediateResult(t *testing.T) {
	svc, err := NewService(
		buaaStudentEmailOTPRepo(t, &AcademicStudent{XH: "20250001", XM: stringPtr("张三")}),
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
	)
	require.NoError(t, err)

	resp, err := svc.MatchStudentEmailAcademicStudent(context.Background(), StudentEmailAcademicMatchInput{
		UserID:      7,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: " 张 三 ",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Matched)
	assert.Equal(t, "20250001@buaa.edu.cn", resp.Email)
	assert.Equal(t, "20250001", resp.StudentID)
	assert.NotEmpty(t, resp.Message)

	svc, err = NewService(
		buaaStudentEmailOTPRepo(t, &AcademicStudent{XH: "20250001", XM: stringPtr("李四")}),
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
	)
	require.NoError(t, err)
	resp, err = svc.MatchStudentEmailAcademicStudent(context.Background(), StudentEmailAcademicMatchInput{
		UserID:      7,
		SchoolID:    4111010006,
		StudentID:   "20250001",
		StudentName: "张三",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Matched)
	assert.Empty(t, resp.Email)
	assert.NotEmpty(t, resp.Message)
}

func TestNormalizeStudentEmailRejectsEmptyEmailParts(t *testing.T) {
	email, err := normalizeStudentEmail(" Student@BUAA.edu.cn ")
	require.NoError(t, err)
	assert.Equal(t, "student@buaa.edu.cn", email)

	_, err = normalizeStudentEmail("@buaa.edu.cn")
	require.ErrorIs(t, err, ErrStudentEmailDomainNotAllowed)

	_, err = normalizeStudentEmail("student@")
	require.ErrorIs(t, err, ErrStudentEmailDomainNotAllowed)
}

func buaaStudentEmailOTPRepo(t *testing.T, student *AcademicStudent) *mockRepo {
	t.Helper()
	academicTable := "academic.buaa_students"
	return &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return nil, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			assert.Equal(t, int64(4111010006), schoolID)
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolCode:         "4111010006",
				SchoolName:         "北京航空航天大学",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "auto",
				AcademicDBTable:    &academicTable,
				Enabled:            true,
				ManualFormFields: json.RawMessage(
					`{"admission":{"emailDomains":["buaa.edu.cn"],"emailIdentityPolicy":{"type":"academic_student_email","studentIDEmailDomain":"buaa.edu.cn","requireStudentName":true}}}`,
				),
			}, nil
		},
		onGetAcademicStudentByXHFromTable: func(_ context.Context, xh string, tableName string) (*AcademicStudent, error) {
			assert.Equal(t, "20250001", xh)
			assert.Equal(t, academicTable, tableName)
			return student, nil
		},
	}
}

type testStudentEmailSender struct {
	email  string
	code   string
	err    error
	calls  int
	onSend func()
}

func (s *testStudentEmailSender) SendStudentVerificationOTP(_ context.Context, email string, code string) error {
	s.calls++
	s.email = email
	s.code = code
	if s.onSend != nil {
		s.onSend()
	}
	return s.err
}
