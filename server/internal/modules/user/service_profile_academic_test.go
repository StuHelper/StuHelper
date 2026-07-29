package user

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type academicAwareMockRepo struct {
	*mockRepo

	onGetAcademicStudentByXHFromTable          func(ctx context.Context, xh string, tableName string) (*AcademicStudent, error)
	onFindAcademicStudentsByPersonUIDFromTable func(ctx context.Context, sfzjlxdm string, sfzjh string, tableName string) ([]AcademicStudent, error)
}

func (m *academicAwareMockRepo) GetAcademicStudentByXHFromTable(ctx context.Context, xh string, tableName string) (*AcademicStudent, error) {
	if m.onGetAcademicStudentByXHFromTable != nil {
		return m.onGetAcademicStudentByXHFromTable(ctx, xh, tableName)
	}
	return nil, nil
}

type fakeLDAPAuthClient struct{}

type fakeExternalStudentDirectory struct {
	record  *ExternalStudentRecord
	handled bool
}

func (d fakeExternalStudentDirectory) LookupStudent(
	context.Context,
	string,
	string,
) (*ExternalStudentRecord, bool, error) {
	return d.record, d.handled, nil
}

func (f *fakeLDAPAuthClient) Login(context.Context, string, string) (*LDAPLoginResult, error) {
	return &LDAPLoginResult{Authenticated: true}, nil
}

func (f *fakeLDAPAuthClient) QueryUserByUID(context.Context, string) (*LDAPUserInfo, error) {
	return &LDAPUserInfo{}, nil
}

func (m *academicAwareMockRepo) FindAcademicStudentsByPersonUIDFromTable(ctx context.Context, sfzjlxdm string, sfzjh string, tableName string) ([]AcademicStudent, error) {
	if m.onFindAcademicStudentsByPersonUIDFromTable != nil {
		return m.onFindAcademicStudentsByPersonUIDFromTable(ctx, sfzjlxdm, sfzjh, tableName)
	}
	return nil, nil
}

func TestGetAcademicStudentByXH_UsesTableAwareRepoWhenAvailable(t *testing.T) {
	const expectedTable = "academic.custom_students"
	const expectedStudentID = "20240001"

	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{},
		onGetAcademicStudentByXHFromTable: func(_ context.Context, xh string, tableName string) (*AcademicStudent, error) {
			assert.Equal(t, expectedStudentID, xh)
			assert.Equal(t, expectedTable, tableName)
			return &AcademicStudent{XH: expectedStudentID}, nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	student, err := service.getAcademicStudentByXH(context.Background(), expectedStudentID, expectedTable)
	require.NoError(t, err)
	require.NotNil(t, student)
	assert.Equal(t, expectedStudentID, student.XH)
}

func TestFindAcademicStudentsByPersonUID_UsesTableAwareRepoWhenAvailable(t *testing.T) {
	const expectedTable = "academic.custom_students"
	const expectedDocID = "110101199001011237"

	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{},
		onFindAcademicStudentsByPersonUIDFromTable: func(_ context.Context, sfzjlxdm string, sfzjh string, tableName string) ([]AcademicStudent, error) {
			assert.Equal(t, "", sfzjlxdm)
			assert.Equal(t, expectedDocID, sfzjh)
			assert.Equal(t, expectedTable, tableName)
			return []AcademicStudent{{XH: "20240001"}, {XH: "20240002"}}, nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	students, err := service.findAcademicStudentsByPersonUID(context.Background(), "", expectedDocID, expectedTable)
	require.NoError(t, err)
	require.Len(t, students, 2)
	assert.Equal(t, "20240001", students[0].XH)
	assert.Equal(t, "20240002", students[1].XH)
}

func TestSubmitIdentity_MainlandIDAutoMatchRequiresVerifiedBoundStudent(t *testing.T) {
	const (
		userID    = int64(7)
		schoolID  = int64(4111010002)
		studentID = "20240002"
		tableName = "academic.school_b_students"
		docNumber = "110101199001011237"
		realName  = "张三"
	)

	var captured *IdentityRecord
	queryCount := 0
	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{
			onGetIdentityStatusByUserID: func(_ context.Context, gotUserID int64) (*IdentityStatus, error) {
				assert.Equal(t, userID, gotUserID)
				if captured == nil {
					return nil, nil
				}
				return &IdentityStatus{
					UserID:       captured.UserID,
					DocType:      captured.DocType,
					RealName:     captured.RealName,
					Verified:     captured.Verified,
					VerifyMethod: captured.VerifyMethod,
					ReviewedAt:   captured.ReviewedAt,
					VerifiedAt:   captured.VerifiedAt,
				}, nil
			},
			onGetProfileByUserID: func(_ context.Context, gotUserID int64) (*Profile, error) {
				assert.Equal(t, userID, gotUserID)
				activeStudentID := studentID
				return &Profile{
					UserID:             userID,
					SchoolID:           int64Ptr(schoolID),
					StudentIDs:         []string{studentID},
					ActiveStudentID:    &activeStudentID,
					VerificationStatus: StatusVerified,
				}, nil
			},
			onGetSchoolConfig: func(_ context.Context, gotSchoolID int64) (*SchoolConfig, error) {
				assert.Equal(t, schoolID, gotSchoolID)
				return &SchoolConfig{
					SchoolID:        schoolID,
					Enabled:         true,
					AcademicDBTable: stringPtr(tableName),
				}, nil
			},
			onCreateIdentity: func(_ context.Context, identity *IdentityRecord) error {
				copied := *identity
				captured = &copied
				return nil
			},
		},
		onFindAcademicStudentsByPersonUIDFromTable: func(_ context.Context, _, gotDocNumber, gotTableName string) ([]AcademicStudent, error) {
			queryCount++
			assert.Equal(t, docNumber, gotDocNumber)
			assert.Equal(t, tableName, gotTableName)
			return []AcademicStudent{
				{XH: "20249999", XM: stringPtr(realName)},
				{XH: studentID, XM: stringPtr(realName)},
			}, nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := service.SubmitIdentity(context.Background(), userID, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: docNumber,
		RealName:  realName,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.NotNil(t, result)

	assert.True(t, captured.Verified)
	require.NotNil(t, captured.VerifyMethod)
	assert.Equal(t, VerifyMethodAcademicDB, *captured.VerifyMethod)
	assert.NotNil(t, captured.ReviewedAt)
	assert.NotNil(t, captured.VerifiedAt)
	assert.True(t, result.Verified)
	assert.Equal(t, 1, queryCount, "自动匹配只能查询当前已认证账号所属学校的学籍表")
}

func TestSubmitIdentity_MainlandIDDoesNotAutoVerifyWithoutBoundStudentProof(t *testing.T) {
	const (
		userID    = int64(7)
		schoolID  = int64(4111010002)
		tableName = "academic.school_b_students"
		docNumber = "110101199001011237"
		realName  = "张三"
	)

	tests := []struct {
		name    string
		profile *Profile
	}{
		{
			name: "student profile is not verified",
			profile: &Profile{
				UserID:             userID,
				SchoolID:           int64Ptr(schoolID),
				StudentIDs:         []string{"20240002"},
				VerificationStatus: StatusPending,
			},
		},
		{
			name: "academic record belongs to a different student id",
			profile: &Profile{
				UserID:             userID,
				SchoolID:           int64Ptr(schoolID),
				StudentIDs:         []string{"20240001"},
				VerificationStatus: StatusVerified,
			},
		},
		{
			name: "verified profile has no bound student id",
			profile: &Profile{
				UserID:             userID,
				SchoolID:           int64Ptr(schoolID),
				VerificationStatus: StatusVerified,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured *IdentityRecord
			repo := &academicAwareMockRepo{
				mockRepo: &mockRepo{
					onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
						if captured == nil {
							return nil, nil
						}
						return &IdentityStatus{
							UserID:   captured.UserID,
							DocType:  captured.DocType,
							RealName: captured.RealName,
							Verified: captured.Verified,
						}, nil
					},
					onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
						return tt.profile, nil
					},
					onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
						return &SchoolConfig{
							SchoolID:        schoolID,
							Enabled:         true,
							AcademicDBTable: stringPtr(tableName),
						}, nil
					},
					onCreateIdentity: func(_ context.Context, identity *IdentityRecord) error {
						copied := *identity
						captured = &copied
						return nil
					},
				},
				onFindAcademicStudentsByPersonUIDFromTable: func(
					_ context.Context,
					_,
					_ string,
					_ string,
				) ([]AcademicStudent, error) {
					return []AcademicStudent{{XH: "20240002", XM: stringPtr(realName)}}, nil
				},
			}

			store := &fakeIdentityPhotoStore{
				presignURL: "https://storage.example.test/identity/photo.png",
			}
			service, err := NewService(
				repo,
				[]byte("test-hmac-key-at-least-32-chars!"),
				&fakeEncryptor{},
				WithIdentityPhotoStore(store),
			)
			require.NoError(t, err)

			front := "identities/7/2026/04/1777777777777777001-front.png"
			selfie := "identities/7/2026/04/1777777777777777002-selfie.png"
			result, err := service.SubmitIdentity(context.Background(), userID, SubmitIdentityRequest{
				DocType:        DocTypeMainlandID,
				DocNumber:      docNumber,
				RealName:       realName,
				DocPhotoFront:  &front,
				DocPhotoSelfie: &selfie,
			})
			require.NoError(t, err)
			require.NotNil(t, captured)
			require.NotNil(t, result)
			assert.False(t, captured.Verified)
			assert.Nil(t, captured.VerifyMethod)
			assert.Nil(t, captured.ReviewedAt)
			assert.Nil(t, captured.VerifiedAt)
			assert.False(t, result.Verified)
		})
	}
}

func TestSubmitIdentity_AcademicAutoMatchIsRevalidatedInsideTransaction(t *testing.T) {
	const (
		userID    = int64(7)
		schoolID  = int64(4111010002)
		studentID = "20240002"
		tableName = "academic.school_b_students"
	)

	var captured *IdentityRecord
	initialProfile := &Profile{
		UserID:             userID,
		SchoolID:           int64Ptr(schoolID),
		StudentIDs:         []string{studentID},
		VerificationStatus: StatusVerified,
	}
	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{
			onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
				if captured == nil {
					return nil, nil
				}
				return &IdentityStatus{UserID: userID, Verified: captured.Verified}, nil
			},
			onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
				return initialProfile, nil
			},
			onGetProfileByUserIDForUpdateTx: func(_ context.Context, _ pgx.Tx, _ int64) (*Profile, error) {
				return &Profile{
					UserID:             userID,
					SchoolID:           int64Ptr(schoolID),
					StudentIDs:         []string{studentID},
					VerificationStatus: StatusRejected,
				}, nil
			},
			onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
				return &SchoolConfig{
					SchoolID:        schoolID,
					Enabled:         true,
					AcademicDBTable: stringPtr(tableName),
				}, nil
			},
			onCreateIdentityTx: func(_ context.Context, _ pgx.Tx, identity *IdentityRecord) error {
				copied := *identity
				captured = &copied
				return nil
			},
		},
		onFindAcademicStudentsByPersonUIDFromTable: func(
			_ context.Context,
			_,
			_ string,
			_ string,
		) ([]AcademicStudent, error) {
			return []AcademicStudent{{XH: studentID, XM: stringPtr("张三")}}, nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := service.SubmitIdentity(context.Background(), userID, SubmitIdentityRequest{
		DocType:   DocTypeMainlandID,
		DocNumber: "110101199001011237",
		RealName:  "张三",
	})
	require.ErrorIs(t, err, ErrPhotoRequired)
	assert.Nil(t, result)
	assert.Nil(t, captured, "stale automatic proof must not create a pending record without manual evidence")
}

func TestUpdateSchoolConfig_RejectsInvalidAcademicTable(t *testing.T) {
	existingConfig := &SchoolConfig{
		SchoolID:           4111010006,
		SchoolName:         "BUAA",
		VerificationMethod: VerifyMethodManual,
		Enabled:            true,
	}
	updatedCalled := false
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			assert.Equal(t, int64(4111010006), schoolID)
			configCopy := *existingConfig
			return &configCopy, nil
		},
		onUpdateSchoolConfig: func(_ context.Context, config *SchoolConfig) error {
			updatedCalled = true
			return nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	invalidTable := "academic.students;drop table users;"
	err = service.UpdateSchoolConfig(context.Background(), 4111010006, UpdateSchoolConfigInput{
		AcademicDBTable: &invalidTable,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAcademicDBTable)
	assert.False(t, updatedCalled)
}

func TestGetAcademicInfo_UsesSchoolConfiguredAcademicTable(t *testing.T) {
	const (
		schoolID        = int64(4111010006)
		expectedTable   = "academic.custom_students"
		expectedStudent = "20240001"
	)

	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{
			onGetSchoolConfig: func(_ context.Context, gotSchoolID int64) (*SchoolConfig, error) {
				assert.Equal(t, schoolID, gotSchoolID)
				return &SchoolConfig{
					SchoolID:        schoolID,
					AcademicDBTable: stringPtr(expectedTable),
					Enabled:         true,
				}, nil
			},
		},
		onGetAcademicStudentByXHFromTable: func(_ context.Context, xh string, tableName string) (*AcademicStudent, error) {
			assert.Equal(t, expectedStudent, xh)
			assert.Equal(t, expectedTable, tableName)
			return &AcademicStudent{XH: expectedStudent}, nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	student, err := service.GetAcademicInfo(context.Background(), schoolID, expectedStudent)
	require.NoError(t, err)
	require.NotNil(t, student)
	assert.Equal(t, expectedStudent, student.XH)
}

func TestGetAcademicInfo_UsesExternalStudentDirectoryWhenConfigured(t *testing.T) {
	const (
		schoolID   = int64(4111010006)
		schoolCode = "4111010006"
		studentID  = "20250001"
	)
	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{
			onGetSchoolConfig: func(_ context.Context, gotSchoolID int64) (*SchoolConfig, error) {
				assert.Equal(t, schoolID, gotSchoolID)
				return &SchoolConfig{
					SchoolID:   schoolID,
					SchoolCode: schoolCode,
					Enabled:    true,
				}, nil
			},
		},
		onGetAcademicStudentByXHFromTable: func(context.Context, string, string) (*AcademicStudent, error) {
			t.Fatal("local academic table should not be queried when external source handles the school")
			return nil, nil
		},
	}

	service, err := NewService(
		repo,
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithExternalStudentDirectory(fakeExternalStudentDirectory{
			handled: true,
			record: &ExternalStudentRecord{
				SchoolCode:  schoolCode,
				StudentID:   studentID,
				StudentName: "张三",
			},
		}),
	)
	require.NoError(t, err)

	student, err := service.GetAcademicInfo(context.Background(), schoolID, studentID)
	require.NoError(t, err)
	require.NotNil(t, student)
	assert.Equal(t, studentID, student.XH)
	require.NotNil(t, student.XM)
	assert.Equal(t, "张三", *student.XM)
}

func TestGetAcademicInfo_FailsWhenAcademicTableMissing(t *testing.T) {
	const (
		schoolID        = int64(4111010006)
		expectedStudent = "20240001"
	)

	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID: schoolID,
				Enabled:  true,
			}, nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = service.GetAcademicInfo(context.Background(), schoolID, expectedStudent)
	assert.ErrorIs(t, err, ErrAcademicTableNotConfigured)
}

func TestGetAcademicInfo_FailsWhenSchoolDisabled(t *testing.T) {
	const (
		schoolID        = int64(4111010006)
		expectedStudent = "20240001"
	)

	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			table := "academic.custom_students"
			return &SchoolConfig{
				SchoolID:        schoolID,
				AcademicDBTable: &table,
				Enabled:         false,
			}, nil
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = service.GetAcademicInfo(context.Background(), schoolID, expectedStudent)
	assert.ErrorIs(t, err, ErrSchoolDisabled)
}

func stringPtr(value string) *string {
	return &value
}

func TestParseSchoolLDAPConfig(t *testing.T) {
	raw := json.RawMessage(`{
		"url": "ldap://ldap.example.com:636",
		"baseDN": "cn=users,dc=example,dc=com",
		"systemBindDN": "cn=system,dc=example,dc=com",
		"systemBindPassword": "secret",
		"useTLS": true
	}`)
	cfg, err := parseSchoolLDAPConfig(raw)
	require.NoError(t, err)
	assert.Equal(t, "ldap://ldap.example.com:636", cfg.URL)
	assert.Equal(t, "cn=users,dc=example,dc=com", cfg.BaseDN)
	assert.Equal(t, "cn=system,dc=example,dc=com", cfg.SystemBindDN)
	assert.Equal(t, "secret", cfg.SystemBindPassword)
	assert.True(t, cfg.UseTLS)
}

func TestParseSchoolLDAPConfig_EmptyFails(t *testing.T) {
	_, err := parseSchoolLDAPConfig(nil)
	assert.ErrorIs(t, err, ErrSchoolLDAPConfigMissing)
}

func TestEnsureLDAPClientForSchool_UsesSchoolConfig(t *testing.T) {
	service, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	var captured LDAPConfig
	expectedClient := &fakeLDAPAuthClient{}
	service.ldapClientFactory = func(cfg LDAPConfig) (LDAPAuthClient, error) {
		captured = cfg
		return expectedClient, nil
	}

	client, err := service.ensureLDAPClientForSchool(&SchoolConfig{
		LDAPConfig: json.RawMessage(`{
			"url":"ldaps://ldap.school.example:636",
			"baseDN":"ou=users,dc=example,dc=com",
			"systemBindDN":"cn=system,dc=example,dc=com",
			"systemBindPassword":"secret",
			"useTLS":true,
			"insecureSkipVerify":false
		}`),
	})
	require.NoError(t, err)
	assert.Same(t, expectedClient, client)
	assert.Equal(t, "ldaps://ldap.school.example:636", captured.URL)
	assert.Equal(t, "ou=users,dc=example,dc=com", captured.BaseDN)
}

func TestEnsureLDAPClientForSchool_MissingConfigFailsEvenWhenDefaultClientExists(t *testing.T) {
	service, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	_, err = service.ensureLDAPClientForSchool(&SchoolConfig{})
	assert.ErrorIs(t, err, ErrSchoolLDAPConfigMissing)
}
