package openplatform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceLookupMethodsRejectInvalidIDsBeforeRepository(t *testing.T) {
	ctx := context.Background()
	service := &Service{}

	for _, appID := range []int64{0, -1} {
		app, err := service.AppByID(ctx, appID)
		require.ErrorIs(t, err, ErrAppNotFound)
		assert.Nil(t, app)
	}

	for _, userID := range []int64{0, -1} {
		projection, err := service.UserProjection(ctx, userID)
		require.ErrorIs(t, err, ErrDisclosureUnavailable)
		assert.Nil(t, projection)
	}
}
