package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFailClosedAuthorizationProvider_ReturnsConfigurationError(t *testing.T) {
	provider := NewFailClosedAuthorizationProvider()

	assert.ErrorIs(t,
		provider.WriteReviewRelations(context.Background(), "review-1", "user-1", "42", "10006"),
		errAuthorizationProviderNotConfigured,
	)
	assert.ErrorIs(t,
		provider.WriteReportRelations(context.Background(), "report-1", "user-1", "review-1", "10006"),
		errAuthorizationProviderNotConfigured,
	)
}

func TestReviewPermissionRelationForAction(t *testing.T) {
	assert.Equal(t, "can_hide", reviewPermissionRelationForAction("hide"))
	assert.Equal(t, "can_hide", reviewPermissionRelationForAction("restore"))
	assert.Equal(t, "can_delete", reviewPermissionRelationForAction("delete"))
}

func TestModerationScopeSectionModeratorRequiresReviewModerationSection(t *testing.T) {
	scope := moderationScope{
		moderatorSections: map[string]struct{}{
			"school_10006_qa": {},
		},
	}

	assert.False(t, scope.canModerateSchool(10006))
	assert.Empty(t, scope.schoolIDs())

	scope = moderationScope{
		moderatorSections: map[string]struct{}{
			reviewModerationSectionID(10006): {},
		},
	}

	assert.True(t, scope.canModerateSchool(10006))
	assert.Equal(t, []int64{10006}, scope.schoolIDs())
}
