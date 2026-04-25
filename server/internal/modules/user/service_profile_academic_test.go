package user

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
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

func (f *fakeLDAPAuthClient) Login(context.Context, string, string) (*ldap.LoginResult, error) {
	return &ldap.LoginResult{Authenticated: true}, nil
}

func (f *fakeLDAPAuthClient) QueryUserByUID(context.Context, string) (*ldap.UserInfo, error) {
	return &ldap.UserInfo{}, nil
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
	const expectedDocID = "110101199001011234"

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

func TestSubmitIdentity_MainlandIDAutoMatchUsesEnabledSchoolTables(t *testing.T) {
	const (
		docNumber = "110101199001011234"
		realName  = "张三"
	)

	var captured *IdentityRecord
	repo := &academicAwareMockRepo{
		mockRepo: &mockRepo{
			onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
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
			onListSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
				tableOne := "academic.school_a_students"
				tableTwo := "academic.school_b_students"
				return []SchoolConfig{
					{SchoolID: 10001, Enabled: true, AcademicDBTable: &tableOne},
					{SchoolID: 10002, Enabled: true, AcademicDBTable: &tableTwo},
				}, nil
			},
			onCreateIdentity: func(_ context.Context, identity *IdentityRecord) error {
				copied := *identity
				captured = &copied
				return nil
			},
		},
		onFindAcademicStudentsByPersonUIDFromTable: func(_ context.Context, _, gotDocNumber, tableName string) ([]AcademicStudent, error) {
			assert.Equal(t, docNumber, gotDocNumber)
			switch tableName {
			case "academic.school_a_students":
				return []AcademicStudent{{XH: "20240001", XM: stringPtr("李四")}}, nil
			case "academic.school_b_students":
				return []AcademicStudent{{XH: "20240002", XM: stringPtr(realName)}}, nil
			default:
				return nil, nil
			}
		},
	}

	service, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	result, err := service.SubmitIdentity(context.Background(), 7, SubmitIdentityRequest{
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
}

func TestUpdateSchoolConfig_RejectsInvalidAcademicTable(t *testing.T) {
	existingConfig := &SchoolConfig{
		SchoolID:           10006,
		SchoolName:         "BUAA",
		VerificationMethod: VerifyMethodManual,
		Enabled:            true,
	}
	updatedCalled := false
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			assert.Equal(t, int64(10006), schoolID)
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
	err = service.UpdateSchoolConfig(context.Background(), 10006, UpdateSchoolConfigInput{
		AcademicDBTable: &invalidTable,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAcademicDBTable)
	assert.False(t, updatedCalled)
}

func TestGetAcademicInfo_UsesSchoolConfiguredAcademicTable(t *testing.T) {
	const (
		schoolID        = int64(10006)
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

func TestGetAcademicInfo_FailsWhenAcademicTableMissing(t *testing.T) {
	const (
		schoolID        = int64(10006)
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
		schoolID        = int64(10006)
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

	var captured ldap.Config
	expectedClient := &fakeLDAPAuthClient{}
	service.ldapClientFactory = func(cfg ldap.Config) (ldapAuthClient, error) {
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
