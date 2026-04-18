package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFailClosedAuthorizationProvider_DeniesChecksAndNoopsWrites(t *testing.T) {
	provider := NewFailClosedAuthorizationProvider()

	allowed, err := provider.Check(context.Background(), "user:1", "can_hide", "review:1")
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.NoError(t, provider.WriteReviewRelations(context.Background(), "review-1", "user-1", "42", "10006"))
	assert.NoError(t, provider.WriteReportRelations(context.Background(), "report-1", "user-1", "review-1", "10006"))
}

func TestReviewPermissionRelationForAction(t *testing.T) {
	assert.Equal(t, "can_hide", reviewPermissionRelationForAction("hide"))
	assert.Equal(t, "can_hide", reviewPermissionRelationForAction("restore"))
	assert.Equal(t, "can_delete", reviewPermissionRelationForAction("delete"))
}
