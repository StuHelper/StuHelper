package user

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestRepositoryEnsureVerificationCredentialTxInsertsAndDeduplicates(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO users (id, casdoor_subject, username)
		VALUES (6, 'subject-repository-credential', 'repository-credential')
		ON CONFLICT (id) DO NOTHING
	`)
	require.NoError(t, err)
	insertDirectoryOnlySchool(t, fixture, 4111010006, "4111010006", "北京航空航天大学")

	credential := VerificationCredentialProjection{
		UserID:         6,
		SchoolID:       4111010006,
		Kind:           userVerificationCredentialKindSchoolEmailOTP,
		SubjectHash:    "credential-subject-hash",
		SubjectDisplay: "2******8@buaa.edu.cn",
		VerifiedAt:     time.Now().UTC().Truncate(time.Second),
	}

	for i := 0; i < 2; i++ {
		err = repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			return repo.EnsureVerificationCredentialTx(ctx, tx, credential)
		})
		require.NoError(t, err)
	}

	var count int
	err = fixture.Pool.QueryRow(ctx, `
		SELECT count(*)
		FROM user_verification_credentials
		WHERE user_id = $1
		  AND school_id = $2
		  AND kind = $3
		  AND subject_hash = $4
		  AND revoked_at IS NULL
	`, credential.UserID, credential.SchoolID, credential.Kind, credential.SubjectHash).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
