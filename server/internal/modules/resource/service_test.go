package resource

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceObjectKeyUsesUUIDAndSanitizedSegments(t *testing.T) {
	key, err := newResourceObjectKey("oidc/user 1", "lecture notes/intro.txt")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(key, "resources/oidc-user_1/"))
	assert.True(t, strings.HasSuffix(key, "-lecture_notes-intro.txt"))
	assert.NotContains(t, key, "../")
	assert.Len(t, strings.Split(key, "/"), 3)

	objectID := strings.TrimSuffix(strings.TrimPrefix(key, "resources/oidc-user_1/"), "-lecture_notes-intro.txt")
	_, err = uuid.Parse(objectID)
	require.NoError(t, err)
}

func TestSanitizeObjectKeySegmentDefaultsUnsafeEmptyValues(t *testing.T) {
	assert.Equal(t, "unknown", sanitizeObjectKeySegment(" "))
	assert.Equal(t, "unknown", sanitizeObjectKeySegment("."))
	assert.Equal(t, "unknown", sanitizeObjectKeySegment(".."))
}

func TestResourceOperationsRejectInvalidResourceIDBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	for _, resourceID := range []int64{0, -1} {
		item, err := svc.GetResource(ctx, resourceID, "viewer")
		require.ErrorIs(t, err, ErrResourceIDInvalid)
		assert.Nil(t, item)

		updated, err := svc.UpdateResource(ctx, resourceID, "owner", UpdateRequest{Title: "title"})
		require.ErrorIs(t, err, ErrResourceIDInvalid)
		assert.Nil(t, updated)

		err = svc.DeleteResource(ctx, resourceID, "owner")
		require.ErrorIs(t, err, ErrResourceIDInvalid)

		url, err := svc.GetDownloadURL(ctx, resourceID, "viewer")
		require.ErrorIs(t, err, ErrResourceIDInvalid)
		assert.Empty(t, url)
	}
}
