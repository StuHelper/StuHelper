package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestRepositoryListAllSchoolConfigsIncludesDirectorySchools(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	insertDirectoryOnlySchool(t, fixture, 4111010001, "4111010001", "北京大学")

	configs, err := repo.ListAllSchoolConfigs(ctx)
	require.NoError(t, err)

	config := findSchoolConfig(t, configs, 4111010001)
	assert.Equal(t, "4111010001", config.SchoolCode)
	assert.Equal(t, "北京大学", config.SchoolName)
	assert.Equal(t, VerifyMethodManual, config.VerificationMethod)
	assert.Equal(t, "manual", config.ApprovalPolicy)
	assert.False(t, config.Enabled)
}

func TestRepositoryUpdateSchoolConfigUpsertsDirectorySchoolConfig(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	insertDirectoryOnlySchool(t, fixture, 4111010001, "4111010001", "北京大学")

	err := repo.UpdateSchoolConfig(ctx, &SchoolConfig{
		SchoolID:           4111010001,
		SchoolName:         "北京大学（测试）",
		VerificationMethod: VerifyMethodManual,
		ApprovalPolicy:     "manual",
		Enabled:            true,
	})
	require.NoError(t, err)

	config, err := repo.GetSchoolConfig(ctx, 4111010001)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "4111010001", config.SchoolCode)
	assert.Equal(t, "北京大学（测试）", config.SchoolName)
	assert.Equal(t, VerifyMethodManual, config.VerificationMethod)
	assert.Equal(t, "manual", config.ApprovalPolicy)
	assert.True(t, config.Enabled)
}

func TestRepositoryUpdateSchoolConfigRollsBackDirectoryNameOnConfigFailure(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	insertDirectoryOnlySchool(t, fixture, 4111010002, "4111010002", "清华大学")

	err := repo.UpdateSchoolConfig(ctx, &SchoolConfig{
		SchoolID:           4111010002,
		SchoolName:         "清华大学（测试）",
		VerificationMethod: "unsupported",
		ApprovalPolicy:     "manual",
		Enabled:            true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "UpdateSchoolConfig")
	assert.Equal(t, "清华大学", schoolDirectoryName(t, fixture, 4111010002))
}

func insertDirectoryOnlySchool(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	id int64,
	code string,
	name string,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO schools (id, code, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET code = EXCLUDED.code,
		    name = EXCLUDED.name
	`, id, code, name)
	require.NoError(t, err)
	_, err = fixture.Pool.Exec(context.Background(), `
		DELETE FROM school_configs
		WHERE school_id = $1
	`, id)
	require.NoError(t, err)
}

func schoolDirectoryName(t *testing.T, fixture *postgresfixture.Fixture, id int64) string {
	t.Helper()
	var name string
	err := fixture.Pool.QueryRow(context.Background(), `SELECT name FROM schools WHERE id = $1`, id).Scan(&name)
	require.NoError(t, err)
	return name
}

func findSchoolConfig(t *testing.T, configs []SchoolConfig, schoolID int64) SchoolConfig {
	t.Helper()
	for _, config := range configs {
		if config.SchoolID == schoolID {
			return config
		}
	}
	t.Fatalf("school config %d not found", schoolID)
	return SchoolConfig{}
}
