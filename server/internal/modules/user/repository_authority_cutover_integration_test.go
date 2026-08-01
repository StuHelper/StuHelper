package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestAuthorityCutoverRepositoryListsOnlyLinkedIdentitiesInStableOrder(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	_, err := postgres.Pool.Exec(ctx, `
		INSERT INTO users (id, casdoor_subject, username)
		VALUES
			(82002, 'cutover-subject-two', 'cutover-user-two'),
			(82001, 'cutover-subject-one', 'cutover-user-one'),
			(82003, '', 'cutover-user-unlinked')
	`)
	require.NoError(t, err)

	identities, err := NewAuthorityCutoverRepository(postgres.DB).ListLinkedIdentities(ctx)

	require.NoError(t, err)
	assert.Equal(t, []AuthorityCutoverIdentity{
		{InternalUserID: 82001, ProviderSubject: "cutover-subject-one"},
		{InternalUserID: 82002, ProviderSubject: "cutover-subject-two"},
	}, identities)
}
