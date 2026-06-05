package user

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

type mobileLDAPAuthClient struct {
	mobile string
}

func (c mobileLDAPAuthClient) Login(context.Context, string, string) (*LDAPLoginResult, error) {
	return &LDAPLoginResult{Authenticated: true}, nil
}

func (c mobileLDAPAuthClient) QueryUserByUID(context.Context, string) (*LDAPUserInfo, error) {
	return &LDAPUserInfo{Mobile: c.mobile}, nil
}

func TestValidateStudentVerificationTransition(t *testing.T) {
	t.Run("nil profile allowed", func(t *testing.T) {
		require.NoError(t, validateStudentVerificationTransition(nil))
	})

	t.Run("verified blocked", func(t *testing.T) {
		err := validateStudentVerificationTransition(&Profile{VerificationStatus: StatusVerified})
		require.ErrorIs(t, err, ErrProfileAlreadyVerified)
	})

	t.Run("pending blocked", func(t *testing.T) {
		err := validateStudentVerificationTransition(&Profile{VerificationStatus: StatusPending})
		require.ErrorIs(t, err, ErrProfilePendingReview)
	})

	t.Run("unverified allowed", func(t *testing.T) {
		require.NoError(t, validateStudentVerificationTransition(&Profile{VerificationStatus: StatusUnverified}))
	})
}

func TestVerifyStudent_ManualAllowsEmptyCredentialsAndPersistsManualData(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(4111010006), schoolID)
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "manual",
				ManualFormFields: json.RawMessage(`[
					{"key":"studentID","label":"学号","type":"text","required":true},
					{"key":"department","label":"院系","type":"text","required":false}
				]`),
				Enabled: true,
			}, nil
		},
		onCreateProfile: func(_ context.Context, profile *Profile) error {
			captured = profile
			return nil
		},
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			if captured == nil {
				return nil, nil
			}
			return captured, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	profile, err := svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: 4111010006,
		ManualFormData: map[string]any{
			"studentID":  "20240001",
			"department": "计算机学院",
		},
		Consent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.NotNil(t, captured)

	require.NotNil(t, captured.VerificationMethod)
	assert.Equal(t, VerifyMethodManual, *captured.VerificationMethod)
	assert.Equal(t, StatusPending, captured.VerificationStatus)
	assert.Equal(t, []string{"20240001"}, captured.StudentIDs)
	require.NotNil(t, captured.ActiveStudentID)
	assert.Equal(t, "20240001", *captured.ActiveStudentID)
	assert.JSONEq(t, `{"department":"计算机学院","studentID":"20240001"}`, string(captured.ManualFormData))
}

func TestVerifyStudent_DoesNotRequireIdentityVerification(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			t.Fatal("student verification must not query identity verification status")
			return nil, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(4111010006), schoolID)
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "manual",
				ManualFormFields: json.RawMessage(`[
					{"key":"studentID","label":"学号","type":"text","required":true}
				]`),
				Enabled: true,
			}, nil
		},
		onCreateProfile: func(_ context.Context, profile *Profile) error {
			captured = profile
			return nil
		},
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			if captured == nil {
				return nil, nil
			}
			return captured, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	profile, err := svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: 4111010006,
		ManualFormData: map[string]any{
			"studentID": "20240001",
		},
		Consent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.NotNil(t, captured)
	assert.Equal(t, StatusPending, captured.VerificationStatus)
}

func TestVerifyStudent_ManualWithoutStudentIDDoesNotPersistBlankIdentifiers(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           4111010001,
				SchoolName:         "人工审核学校",
				VerificationMethod: VerifyMethodManual,
				Enabled:            true,
			}, nil
		},
		onCreateProfile: func(_ context.Context, profile *Profile) error {
			captured = profile
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: 4111010001,
		Consent:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Nil(t, captured.StudentIDs)
	assert.Nil(t, captured.ActiveStudentID)
	assert.Nil(t, captured.ManualFormData)
}

func TestVerifyStudent_LDAPRequiresStudentID(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           4111010002,
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID: 4111010002,
		Password: "secret",
		Consent:  true,
	})
	assert.ErrorIs(t, err, ErrStudentIDRequired)
}

func TestVerifyStudent_LDAPRequiresPassword(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           4111010002,
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID:  4111010002,
		StudentID: "20240001",
		Consent:   true,
	})
	assert.ErrorIs(t, err, ErrPasswordRequired)
}

func TestVerifyStudent_LDAPRequiresAcademicStudentRecord(t *testing.T) {
	academicTable := "academic.students"
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 1, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           4111010002,
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				LDAPConfig:         json.RawMessage(`{"url":"ldaps://ldap.example:636","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":true,"insecureSkipVerify":false}`),
				AcademicDBTable:    &academicTable,
				Enabled:            true,
			}, nil
		},
		onGetAcademicStudentByXHFromTable: func(_ context.Context, xh string, tableName string) (*AcademicStudent, error) {
			assert.Equal(t, "20240001", xh)
			assert.Equal(t, academicTable, tableName)
			return nil, nil
		},
	}

	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithLDAPClientFactory(func(_ LDAPConfig) (LDAPAuthClient, error) {
			return &fakeLDAPAuthClient{}, nil
		}),
	)
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), 1, VerifyStudentRequest{
		SchoolID:  4111010002,
		StudentID: "20240001",
		Password:  "secret",
		Consent:   true,
	})

	assert.ErrorIs(t, err, ErrStudentNotFound)
}

func TestVerifyStudent_LDAPPhoneSyncRunsAfterProfileTransactionAndPersistsProjection(t *testing.T) {
	const (
		userID        = int64(1)
		studentID     = "20240001"
		academicTable = "academic.students"
		rawPhone      = "13800138000"
	)

	var (
		capturedProfile *Profile
		phoneProjection []byte
		events          []string
		syncedSubject   string
		syncedPhone     string
	)
	hmacKey := []byte("test-hmac-key-at-least-32-chars!")
	expectedPhoneHash, err := phoneutil.HashLookupWithKey(rawPhone, hmacKey)
	require.NoError(t, err)

	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{
			onGetProfileByUserID: func(_ context.Context, gotUserID int64) (*Profile, error) {
				assert.Equal(t, userID, gotUserID)
				if capturedProfile == nil {
					return nil, nil
				}
				profile := *capturedProfile
				if len(phoneProjection) > 0 {
					profile.PhoneEnc = append([]byte(nil), phoneProjection...)
				}
				return &profile, nil
			},
			onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
				require.Equal(t, int64(4111010002), schoolID)
				return &SchoolConfig{
					SchoolID:           4111010002,
					SchoolName:         "LDAP 学校",
					VerificationMethod: VerifyMethodLDAP,
					LDAPConfig:         json.RawMessage(`{"url":"ldaps://ldap.example:636","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":true,"insecureSkipVerify":false}`),
					AcademicDBTable:    stringPtr(academicTable),
					ApprovalPolicy:     "auto",
					Enabled:            true,
				}, nil
			},
			onWithTx: func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
				events = append(events, "tx_begin")
				if err := fn(ctx, nil); err != nil {
					return err
				}
				events = append(events, "tx_commit")
				return nil
			},
			onCreateProfileTx: func(_ context.Context, _ pgx.Tx, profile *Profile) error {
				profileCopy := *profile
				capturedProfile = &profileCopy
				events = append(events, "create_profile")
				return nil
			},
			onGetCasdoorSubject: func(_ context.Context, gotUserID int64) (string, error) {
				assert.Equal(t, userID, gotUserID)
				return "casdoor-subject-1", nil
			},
			onSetUserPhone: func(_ context.Context, gotUserID int64, phoneEnc []byte, phoneHash string) error {
				assert.Equal(t, userID, gotUserID)
				assert.Equal(t, "encrypted:138****8000", string(phoneEnc))
				assert.Equal(t, expectedPhoneHash, phoneHash)
				phoneProjection = append([]byte(nil), phoneEnc...)
				events = append(events, "phone_projection")
				return nil
			},
		},
		onGetAcademicStudentByXHFromTable: func(_ context.Context, xh string, tableName string) (*AcademicStudent, error) {
			assert.Equal(t, studentID, xh)
			assert.Equal(t, academicTable, tableName)
			return &AcademicStudent{XH: studentID}, nil
		},
	}

	gateway := profileIdentitySyncFunc(func(_ context.Context, subject, phone string) error {
		syncedSubject = subject
		syncedPhone = phone
		events = append(events, "identity_phone")
		return nil
	})

	svc, err := NewService(
		repo,
		hmacKey,
		&fakeEncryptor{},
		WithLDAPClientFactory(func(_ LDAPConfig) (LDAPAuthClient, error) {
			return mobileLDAPAuthClient{mobile: rawPhone}, nil
		}),
		WithProfileIdentitySyncGateway(gateway),
	)
	require.NoError(t, err)

	profile, err := svc.VerifyStudent(context.Background(), userID, VerifyStudentRequest{
		SchoolID:  4111010002,
		StudentID: studentID,
		Password:  "secret",
		Consent:   true,
	})
	require.NoError(t, err)
	require.NotNil(t, profile)

	assert.Equal(t, "casdoor-subject-1", syncedSubject)
	assert.Equal(t, "+8613800138000", syncedPhone)
	assert.True(t, profile.PhoneVerified)
	assert.Equal(t, []string{"tx_begin", "create_profile", "tx_commit", "identity_phone", "phone_projection"}, events)
}

func TestVerifyStudent_LDAPPhoneSyncSkippedWhenProfileTransactionFails(t *testing.T) {
	const (
		userID        = int64(1)
		studentID     = "20240001"
		academicTable = "academic.students"
	)
	txErr := errors.New("commit profile transaction")
	var (
		identitySyncCalls int
		setPhoneCalls     int
	)

	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{
			onGetProfileByUserID: func(_ context.Context, gotUserID int64) (*Profile, error) {
				assert.Equal(t, userID, gotUserID)
				return nil, nil
			},
			onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
				require.Equal(t, int64(4111010002), schoolID)
				return &SchoolConfig{
					SchoolID:           4111010002,
					SchoolName:         "LDAP 学校",
					VerificationMethod: VerifyMethodLDAP,
					LDAPConfig:         json.RawMessage(`{"url":"ldaps://ldap.example:636","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":true,"insecureSkipVerify":false}`),
					AcademicDBTable:    stringPtr(academicTable),
					ApprovalPolicy:     "auto",
					Enabled:            true,
				}, nil
			},
			onWithTx: func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
				if err := fn(ctx, nil); err != nil {
					return err
				}
				return txErr
			},
			onCreateProfileTx: func(context.Context, pgx.Tx, *Profile) error {
				return nil
			},
			onGetCasdoorSubject: func(context.Context, int64) (string, error) {
				t.Fatal("Casdoor subject must not be loaded after a failed profile transaction")
				return "", nil
			},
			onSetUserPhone: func(context.Context, int64, []byte, string) error {
				setPhoneCalls++
				return nil
			},
		},
		onGetAcademicStudentByXHFromTable: func(_ context.Context, xh string, tableName string) (*AcademicStudent, error) {
			assert.Equal(t, studentID, xh)
			assert.Equal(t, academicTable, tableName)
			return &AcademicStudent{XH: studentID}, nil
		},
	}

	gateway := profileIdentitySyncFunc(func(context.Context, string, string) error {
		identitySyncCalls++
		return nil
	})

	svc, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithLDAPClientFactory(func(_ LDAPConfig) (LDAPAuthClient, error) {
			return mobileLDAPAuthClient{mobile: "13800138000"}, nil
		}),
		WithProfileIdentitySyncGateway(gateway),
	)
	require.NoError(t, err)

	_, err = svc.VerifyStudent(context.Background(), userID, VerifyStudentRequest{
		SchoolID:  4111010002,
		StudentID: studentID,
		Password:  "secret",
		Consent:   true,
	})
	require.ErrorIs(t, err, txErr)
	assert.Zero(t, identitySyncCalls)
	assert.Zero(t, setPhoneCalls)
}

// ---------------------------------------------------------------------------
// ReviewStudentVerification
// ---------------------------------------------------------------------------

func TestReviewStudentVerification_RejectionReasonIsOptional(t *testing.T) {
	var captured *Profile

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return &Profile{UserID: 1, VerificationStatus: StatusPending}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			captured = profile
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewStudentVerification(context.Background(), 1, false, "")
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, StatusRejected, captured.VerificationStatus)
	assert.Nil(t, captured.RejectionReason)

	err = svc.ReviewStudentVerification(context.Background(), 1, false, " ")
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Nil(t, captured.RejectionReason)

	reason := "材料不全"
	err = svc.ReviewStudentVerification(context.Background(), 1, false, reason)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.NotNil(t, captured.RejectionReason)
	assert.Equal(t, reason, *captured.RejectionReason)
}

func TestReviewStudentVerification_ApproveFlow(t *testing.T) {
	var capturedProfile *Profile

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return &Profile{UserID: 1, VerificationStatus: StatusPending}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			capturedProfile = profile
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.ReviewStudentVerification(context.Background(), 1, true, "")
	require.NoError(t, err)
	require.NotNil(t, capturedProfile)
	assert.Equal(t, StatusVerified, capturedProfile.VerificationStatus)
	assert.NotNil(t, capturedProfile.VerifiedAt)
}

// ---------------------------------------------------------------------------
// UpdateSchoolConfig
// ---------------------------------------------------------------------------
