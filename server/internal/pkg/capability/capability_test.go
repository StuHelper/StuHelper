package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandRoles_SuperAdminHasAllCapabilities(t *testing.T) {
	caps := ExpandRoles([]string{"super_admin"})
	assert.Contains(t, caps, AdminDashboardView)
	assert.Contains(t, caps, UserIdentityRead)
	assert.Contains(t, caps, UserSystemUpdate)
}

func TestExpandRoles_UserHasOnlyBriefReview(t *testing.T) {
	caps := ExpandRoles([]string{"user"})
	assert.Contains(t, caps, ReviewListBrief)
	assert.NotContains(t, caps, ReviewListFull)
	assert.NotContains(t, caps, AdminDashboardView)
}

func TestExpandRoles_VerifiedStudentCanPost(t *testing.T) {
	caps := ExpandRoles([]string{"verified_student"})
	assert.Contains(t, caps, ReviewCreate)
	assert.Contains(t, caps, ReviewListFull)
	assert.Contains(t, caps, ReviewEditOwn)
	assert.NotContains(t, caps, AdminReviewsManage)
}

func TestExpandRoles_MultipleSorted(t *testing.T) {
	caps := ExpandRoles([]string{"user", "verified_student"})
	assert.Contains(t, caps, ReviewListBrief)
	assert.Contains(t, caps, ReviewCreate)
	// should be deduplicated and sorted
	for i := 1; i < len(caps); i++ {
		assert.True(t, caps[i-1] < caps[i], "capabilities should be sorted: %s >= %s", caps[i-1], caps[i])
	}
}

func TestExpandRoles_UnknownRoleIgnored(t *testing.T) {
	caps := ExpandRoles([]string{"nonexistent_role"})
	assert.Empty(t, caps)
}

func TestHas(t *testing.T) {
	caps := []string{"a", "b", "c"}
	assert.True(t, Has(caps, "b"))
	assert.False(t, Has(caps, "d"))
}

func TestHasAny(t *testing.T) {
	caps := []string{"a", "b"}
	assert.True(t, HasAny(caps, "c", "b"))
	assert.False(t, HasAny(caps, "c", "d"))
}

func TestCanAccessAdmin(t *testing.T) {
	assert.True(t, CanAccessAdmin(ExpandRoles([]string{"super_admin"})))
	assert.True(t, CanAccessAdmin(ExpandRoles([]string{"moderator"})))
	assert.False(t, CanAccessAdmin(ExpandRoles([]string{"user"})))
	assert.False(t, CanAccessAdmin(ExpandRoles([]string{"verified_student"})))
}

func TestNormalize_DeduplicatesAndSorts(t *testing.T) {
	result := Normalize([]string{"c", "a", "b", "a", "", "c"})
	assert.Equal(t, []string{"a", "b", "c"}, result)
}
