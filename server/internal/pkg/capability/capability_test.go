package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestExpandRoles_ScopedRolesNeedExplicitScope(t *testing.T) {
	assert.Empty(t, ExpandRoles([]string{"school_admin"}))
	assert.Empty(t, ExpandRoles([]string{"section_moderator"}))
}

func TestRoleCapabilities_UsesCasdoorV2RoleCatalog(t *testing.T) {
	roles := make([]string, 0, len(GetRoleCapabilities()))
	for role := range GetRoleCapabilities() {
		roles = append(roles, role)
	}

	assert.ElementsMatch(t, []string{
		"super_admin",
		"school_admin",
		"section_admin",
		"section_moderator",
		"section_reviewer",
		"verified_student",
		"user",
	}, roles)
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
	assert.False(t, CanAccessAdmin(ExpandRoles([]string{"section_moderator"})))
	assert.False(t, CanAccessAdmin(ExpandRoles([]string{"moderator"})))
	assert.False(t, CanAccessAdmin(ExpandRoles([]string{"user"})))
	assert.False(t, CanAccessAdmin(ExpandRoles([]string{"verified_student"})))
}

func TestExpandRoleGrants_SchoolAdminUsesScopedSchoolIDs(t *testing.T) {
	grants := ExpandRoleGrants([]string{"school_admin"}, map[string][]string{
		"school_admin": {"1002", "1001", "1002"},
	})
	snapshot := BuildUserAccessSnapshot(grants)

	assert.Empty(t, snapshot.GlobalCapabilities)
	assert.Contains(t, snapshot.Capabilities, AdminReviewsManage)
	assert.Contains(t, snapshot.Capabilities, AdminReportsManage)
	assert.Contains(t, snapshot.Capabilities, UserStudentRead)
	assert.Contains(t, snapshot.Capabilities, UserSchoolUpdate)
	for _, grant := range snapshot.CapabilityGrants {
		assert.False(t, grant.Global)
		assert.Equal(t, []string{"1001", "1002"}, grant.ScopeSchoolIDs)
	}
}

func TestExpandRoleGrants_SchoolAdminWithoutOrgScopeGetsNoGrant(t *testing.T) {
	snapshot := BuildUserAccessSnapshot(ExpandRoleGrants([]string{"school_admin"}, nil))
	assert.Empty(t, snapshot.Capabilities)
	assert.Empty(t, snapshot.CapabilityGrants)
}

func TestExpandRoleGrants_SectionRolesUseScopedSectionIDs(t *testing.T) {
	grants := ExpandRoleGrants([]string{"section_moderator"}, map[string][]string{
		"section_moderator": {"school_10006_review_moderation"},
	})
	snapshot := BuildUserAccessSnapshot(grants)

	assert.Contains(t, snapshot.Capabilities, AdminReviewsManage)
	assert.Empty(t, snapshot.GlobalCapabilities)
	require.Len(t, snapshot.CapabilityGrants, 2)
	for _, grant := range snapshot.CapabilityGrants {
		assert.False(t, grant.Global)
		assert.Empty(t, grant.ScopeSchoolIDs)
		assert.Equal(t, []string{"school_10006_review_moderation"}, grant.ScopeSectionIDs)
	}
}

func TestExpandRoleGrants_SectionAdminDoesNotManageTeachers(t *testing.T) {
	grants := ExpandRoleGrants([]string{"section_admin"}, map[string][]string{
		"section_admin": {"school_10006_review_moderation"},
	})
	snapshot := BuildUserAccessSnapshot(grants)

	assert.NotContains(t, snapshot.Capabilities, AdminTeachersManage)
	for _, grant := range snapshot.CapabilityGrants {
		assert.NotEqual(t, AdminTeachersManage, grant.Name)
	}
}

func TestHasGrantInSchool(t *testing.T) {
	grants := []Grant{
		{Name: UserStudentRead, ScopeSchoolIDs: []string{"1001"}},
		{Name: UserSystemRead, Global: true},
	}
	assert.True(t, HasGrantInSchool(grants, UserStudentRead, "1001"))
	assert.False(t, HasGrantInSchool(grants, UserStudentRead, "1002"))
	assert.True(t, HasGrantInSchool(grants, UserSystemRead, "9999"))
	assert.True(t, HasGlobalGrant(grants, UserSystemRead))
	assert.False(t, HasGlobalGrant(grants, UserStudentRead))
}

func TestNormalize_DeduplicatesAndSorts(t *testing.T) {
	result := Normalize([]string{"c", "a", "b", "a", "", "c"})
	assert.Equal(t, []string{"a", "b", "c"}, result)
}
