package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestRepositoryIdentityReviewOnlyTransitionsPendingOnce(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()
	const userID = int64(81001)

	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO users (id, casdoor_subject, username)
		VALUES ($1, 'subject-identity-review-once', 'identity-review-once')
	`, userID)
	require.NoError(t, err)
	require.NoError(t, repo.CreateIdentity(ctx, &IdentityRecord{
		UserID:       userID,
		DocType:      DocTypePassport,
		DocNumberEnc: []byte("encrypted-document-number"),
		PersonUID:    "identity-review-once-person-uid",
		RealName:     "张三",
		Verified:     false,
	}))

	now := time.Now().UTC()
	method := VerifyMethodManual
	err = repo.UpdateIdentityReviewStatus(
		ctx,
		userID,
		true,
		&method,
		&now,
		&now,
		nil,
	)
	require.ErrorIs(t, err, ErrVerificationReviewStateConflict)

	_, err = fixture.Pool.Exec(ctx, `
		UPDATE user_identities
		SET doc_photo_front = 'identities/81001/2026/07/1777777777777777001-front.png',
		    doc_photo_selfie = 'identities/81001/2026/07/1777777777777777002-selfie.png'
		WHERE user_id = $1
	`, userID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateIdentityReviewStatus(
		ctx,
		userID,
		true,
		&method,
		&now,
		&now,
		nil,
	))

	err = repo.UpdateIdentityReviewStatus(
		ctx,
		userID,
		false,
		nil,
		&now,
		nil,
		nil,
	)
	require.ErrorIs(t, err, ErrVerificationReviewStateConflict)

	status, err := repo.GetIdentityStatusByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.True(t, status.Verified)
	require.NotNil(t, status.VerifyMethod)
	assert.Equal(t, VerifyMethodManual, *status.VerifyMethod)
}
