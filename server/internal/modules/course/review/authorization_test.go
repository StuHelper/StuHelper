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

func TestModerationScopeSectionModeratorRequiresReviewModerationSection(t *testing.T) {
	scope := moderationScope{
		moderatorSections: map[string]struct{}{
			"school_10006_qa": {},
		},
	}

	assert.False(t, scope.canModerateSchool(10006))
	assert.Empty(t, scope.schoolIDs())
	assert.False(t, scope.hasModerationAccess())

	scope = moderationScope{
		moderatorSections: map[string]struct{}{
			reviewModerationSectionID(10006): {},
		},
	}

	assert.True(t, scope.canModerateSchool(10006))
	assert.Equal(t, []int64{10006}, scope.schoolIDs())
	assert.True(t, scope.hasModerationAccess())
}

func TestAdminReportsCacheKeyIncludesModerationScope(t *testing.T) {
	assert.Equal(t,
		"status=pending:page=1:size=20:scope=all",
		adminReportsCacheKey("pending", 1, 20, nil),
	)
	assert.Equal(t,
		"status=pending:page=1:size=20:scope=none",
		adminReportsCacheKey("pending", 1, 20, []int64{}),
	)
	assert.Equal(t,
		"status=pending:page=1:size=20:scope=schools:10006,10007",
		adminReportsCacheKey("pending", 1, 20, []int64{10007, 10006}),
	)
}
