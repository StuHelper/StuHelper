package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
)

func TestFailClosedAuthorizationProvider_ReturnsConfigurationError(t *testing.T) {
	provider := NewFailClosedAuthorizationProvider()

	allowed, err := provider.Check(context.Background(), "user-1", "can_hide", "review-1")
	assert.False(t, allowed)
	assert.ErrorIs(t, err, errAuthorizationProviderNotConfigured)
	assert.ErrorIs(t,
		provider.WriteReviewRelations(context.Background(), "review-1", "user-1", "4111010006"),
		errAuthorizationProviderNotConfigured,
	)
	assert.ErrorIs(t,
		provider.WriteReportRelations(context.Background(), "report-1", "4111010006"),
		errAuthorizationProviderNotConfigured,
	)
}

func TestModerationScopeSectionModeratorRequiresReviewModerationSection(t *testing.T) {
	scope := moderationScope{
		moderatorSections: map[string]struct{}{
			"school_4111010006_qa": {},
		},
	}

	assert.Empty(t, scope.schoolIDs())

	scope = moderationScope{
		moderatorSections: map[string]struct{}{
			reviewModerationSectionID(4111010006): {},
		},
	}

	assert.Equal(t, []int64{4111010006}, scope.schoolIDs())
}

func TestModerationScopeUsesTheRequestedCapabilityGrant(t *testing.T) {
	grants := capability.BuildUserAccessSnapshot([]capability.Grant{
		{
			Name:           capability.AdminReportsManage,
			ScopeSchoolIDs: []string{"4111010006", "invalid"},
		},
		{
			Name:            capability.AdminReviewsManage,
			ScopeSectionIDs: []string{reviewModerationSectionID(4111010007), "school_4111010008_qa"},
		},
	}).CapabilityGrants

	reportScope := moderationScopeFromCapabilityGrants(grants, capability.AdminReportsManage)
	assert.Equal(t, []int64{4111010006}, reportScope.schoolIDs())

	reviewScope := moderationScopeFromCapabilityGrants(grants, capability.AdminReviewsManage)
	assert.Equal(t, []int64{4111010007}, reviewScope.schoolIDs())
}

func TestModerationScopeGlobalGrantCoversAllSchools(t *testing.T) {
	grants := capability.BuildUserAccessSnapshot([]capability.Grant{{
		Name: capability.AdminReportsManage,
	}}).CapabilityGrants

	scope := moderationScopeFromCapabilityGrants(grants, capability.AdminReportsManage)
	assert.True(t, scope.global)
	assert.Nil(t, scope.schoolIDs())
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
		"status=pending:page=1:size=20:scope=schools:4111010006,4111010007",
		adminReportsCacheKey("pending", 1, 20, []int64{4111010007, 4111010006}),
	)
}
