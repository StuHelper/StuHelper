package user

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

func TestUpdateSchoolConfig_MergesPartialUpdateAndPreservesUnspecifiedFields(t *testing.T) {
	var captured *SchoolConfig

	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(4111010006), schoolID)
			academicTable := "academic.buaa_students"
			consentText := "原始授权文本"
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "旧学校名",
				VerificationMethod: VerifyMethodLDAP,
				LDAPConfig:         json.RawMessage(`{"url":"ldap://ldap.old:389","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":false,"insecureSkipVerify":false}`),
				AcademicDBTable:    &academicTable,
				ConsentText:        &consentText,
				ManualFormFields:   json.RawMessage(`[{"key":"idCard","label":"身份证号","type":"text","required":true}]`),
				Enabled:            true,
			}, nil
		},
		onUpdateSchoolConfig: func(_ context.Context, config *SchoolConfig) error {
			captured = config
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	schoolName := "新学校名"
	enabled := false
	ldapURL := "ldaps://ldap.new:636"
	systemBindPassword := "new-secret"
	useTLS := true

	err = svc.UpdateSchoolConfig(context.Background(), 4111010006, UpdateSchoolConfigInput{
		SchoolName: &schoolName,
		LDAPConfig: &SchoolLDAPConfigInput{
			URL:                &ldapURL,
			SystemBindPassword: &systemBindPassword,
			UseTLS:             &useTLS,
		},
		Enabled: &enabled,
	})
	require.NoError(t, err)
	require.NotNil(t, captured)

	assert.Equal(t, int64(4111010006), captured.SchoolID)
	assert.Equal(t, "新学校名", captured.SchoolName)
	assert.Equal(t, VerifyMethodLDAP, captured.VerificationMethod, "未提供的 verificationMethod 应保留原值")
	assert.JSONEq(t, `{"baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"new-secret","url":"ldaps://ldap.new:636","useTLS":true}`, string(captured.LDAPConfig))
	require.NotNil(t, captured.AcademicDBTable)
	assert.Equal(t, "academic.buaa_students", *captured.AcademicDBTable, "未提供的 academicDbTable 应保留原值")
	require.NotNil(t, captured.ConsentText)
	assert.Equal(t, "原始授权文本", *captured.ConsentText, "未提供的 consentText 应保留原值")
	assert.JSONEq(t, `[{"key":"idCard","label":"身份证号","type":"text","required":true}]`, string(captured.ManualFormFields), "未提供的 manualFormFields 应保留原值")
	assert.False(t, captured.Enabled)
}

func TestUpdateSchoolConfig_PreservesExistingLDAPPasswordWhenOmitted(t *testing.T) {
	var captured *SchoolConfig

	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			require.Equal(t, int64(4111010006), schoolID)
			table := "academic.buaa_students"
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodLDAP,
				LDAPConfig:         json.RawMessage(`{"url":"ldap://ldap.old:389","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":false,"insecureSkipVerify":false}`),
				AcademicDBTable:    &table,
				Enabled:            true,
			}, nil
		},
		onUpdateSchoolConfig: func(_ context.Context, config *SchoolConfig) error {
			copied := *config
			captured = &copied
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	ldapURL := "ldaps://ldap.new:636"
	useTLS := true
	err = svc.UpdateSchoolConfig(context.Background(), 4111010006, UpdateSchoolConfigInput{
		LDAPConfig: &SchoolLDAPConfigInput{
			URL:    &ldapURL,
			UseTLS: &useTLS,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.JSONEq(t, `{"url":"ldaps://ldap.new:636","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":true}`, string(captured.LDAPConfig))
}

func TestUpdateSchoolConfig_SchoolNotFoundReturnsError(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSchoolConfig(context.Background(), 4111099999, UpdateSchoolConfigInput{})
	assert.ErrorIs(t, err, ErrSchoolNotFound)
}

func TestUpdateSchoolConfig_InvalidManualFieldConfigReturnsError(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	fields := []ManualFieldDescriptor{{Key: "", Label: "空 key", Type: "text", Required: true}}
	err = svc.UpdateSchoolConfig(context.Background(), 4111010006, UpdateSchoolConfigInput{
		ManualFormFields: &fields,
	})
	assert.ErrorIs(t, err, ErrInvalidManualFieldConfig)
}

func TestUpdateSchoolConfig_EnabledLDAPRequiresAcademicTable(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				Enabled:            false,
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	method := VerifyMethodLDAP
	enabled := true
	err = svc.UpdateSchoolConfig(context.Background(), 4111010006, UpdateSchoolConfigInput{
		VerificationMethod: &method,
		Enabled:            &enabled,
	})

	assert.ErrorIs(t, err, ErrAcademicTableNotConfigured)
}

func TestUpdateSchoolConfig_EnabledLDAPRequiresValidLDAPConfig(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			table := "academic.buaa_students"
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodLDAP,
				AcademicDBTable:    &table,
				Enabled:            true,
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSchoolConfig(context.Background(), 4111010006, UpdateSchoolConfigInput{})

	assert.ErrorIs(t, err, ErrSchoolLDAPConfigMissing)
}

func TestUpdateSchoolConfig_ValidatesAcademicTableExists(t *testing.T) {
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			table := "academic.buaa_students"
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				AcademicDBTable:    &table,
				Enabled:            false,
			}, nil
		},
		onValidateAcademicDBTable: func(_ context.Context, tableName string) error {
			assert.Equal(t, "academic.buaa_students", tableName)
			return assert.AnError
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSchoolConfig(context.Background(), 4111010006, UpdateSchoolConfigInput{})

	assert.ErrorIs(t, err, ErrInvalidAcademicDBTable)
}

func TestUpdateSystemConfig_RejectsInvalidReviewAccessSchoolIDs(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), systemconfig.ReviewAccessSchoolIDsKey, `{"schoolID":"4111010006"}`)
	assert.ErrorIs(t, err, ErrInvalidSystemConfigValue)
}

func TestUpdateSystemConfig_RejectsInvalidReviewPreviewPercent(t *testing.T) {
	svc, err := NewService(&mockRepo{}, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), systemconfig.ReviewPreviewContentPercentKey, "120")
	assert.ErrorIs(t, err, ErrInvalidSystemConfigValue)
}

func TestUpdateSystemConfig_RejectsUnknownReviewAccessSchoolIDs(t *testing.T) {
	repo := &mockRepo{
		onListAllSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return []SchoolConfig{
				{SchoolID: 4111010006},
				{SchoolID: 4111010007},
			}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), systemconfig.ReviewAccessSchoolIDsKey, `["4111010006","4111099999"]`)
	assert.ErrorIs(t, err, ErrInvalidSystemConfigValue)
}

func TestUpdateSystemConfig_ReturnsNotFoundWhenKeyMissing(t *testing.T) {
	repo := &mockRepo{
		onUpdateSystemConfig: func(_ context.Context, key, value string) error {
			assert.Equal(t, "feature.missing", key)
			assert.Equal(t, "enabled", value)
			return ErrSystemConfigNotFound
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	err = svc.UpdateSystemConfig(context.Background(), "feature.missing", "enabled")
	assert.ErrorIs(t, err, ErrSystemConfigNotFound)
}

func TestLoadSystemConfigSnapshots_LoadsAuthAccessTokenTTL(t *testing.T) {
	t.Cleanup(systemconfig.InvalidateAuthTokenPolicySnapshot)

	repo := &mockRepo{
		onListSystemConfigs: func(_ context.Context) ([]SystemConfig, error) {
			return []SystemConfig{{
				Key:   systemconfig.AuthAccessTokenTTLSecondsKey,
				Value: "900",
			}}, nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	require.NoError(t, svc.LoadSystemConfigSnapshots(context.Background()))
	snapshot := systemconfig.GetAuthTokenPolicySnapshot()
	assert.Equal(t, 900, snapshot.AccessTokenTTLSeconds)
}

func TestUpdateSystemConfig_RefreshesAuthAccessTokenTTLSnapshot(t *testing.T) {
	t.Cleanup(systemconfig.InvalidateAuthTokenPolicySnapshot)

	repo := &mockRepo{
		onUpdateSystemConfig: func(_ context.Context, key, value string) error {
			assert.Equal(t, systemconfig.AuthAccessTokenTTLSecondsKey, key)
			assert.Equal(t, "1200", value)
			return nil
		},
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	require.NoError(t, svc.UpdateSystemConfig(context.Background(), systemconfig.AuthAccessTokenTTLSecondsKey, "1200"))
	snapshot := systemconfig.GetAuthTokenPolicySnapshot()
	assert.Equal(t, 1200, snapshot.AccessTokenTTLSeconds)
}
