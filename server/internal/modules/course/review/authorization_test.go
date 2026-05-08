package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFailClosedAuthorizationProvider_ReturnsConfigurationError(t *testing.T) {
	provider := NewFailClosedAuthorizationProvider()

	allowed, err := provider.Check(context.Background(), "user-1", "can_hide", "review-1")
	assert.False(t, allowed)
	assert.ErrorIs(t, err, errAuthorizationProviderNotConfigured)
	assert.ErrorIs(t,
		provider.WriteReviewRelations(context.Background(), "review-1", "user-1", "10006"),
		errAuthorizationProviderNotConfigured,
	)
	assert.ErrorIs(t,
		provider.WriteReportRelations(context.Background(), "report-1", "10006"),
		errAuthorizationProviderNotConfigured,
	)
}

func TestModerationScopeSectionModeratorRequiresReviewModerationSection(t *testing.T) {
	scope := moderationScope{
		moderatorSections: map[string]struct{}{
			"school_10006_qa": {},
		},
	}

	assert.Empty(t, scope.schoolIDs())

	scope = moderationScope{
		moderatorSections: map[string]struct{}{
			reviewModerationSectionID(10006): {},
		},
	}

	assert.Equal(t, []int64{10006}, scope.schoolIDs())
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
