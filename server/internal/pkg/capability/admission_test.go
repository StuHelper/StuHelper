package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandRoles_FreshmanProvisionalMatchesVerifiedStudent(t *testing.T) {
	verified := ExpandRoles([]string{"verified_student"})
	provisional := ExpandRoles([]string{"freshman_provisional"})

	assert.ElementsMatch(t, verified, provisional)
	assert.Contains(t, GetRoleCapabilities(), "verified_student")
	assert.Contains(t, GetRoleCapabilities(), "freshman_provisional")
}

func TestExpandRoles_SuperAdminHasAdmissionCapabilities(t *testing.T) {
	caps := ExpandRoles([]string{"super_admin"})

	assert.Contains(t, caps, AdmissionPolicyRead)
	assert.Contains(t, caps, AdmissionPolicyUpdate)
	assert.Contains(t, caps, StudentManualReviewRead)
	assert.Contains(t, caps, StudentManualReviewDecide)
	assert.Contains(t, caps, AdmissionSessionRead)
	assert.Contains(t, caps, AdmissionSessionManage)
	assert.Contains(t, caps, MemberBlacklistRead)
	assert.Contains(t, caps, MemberBlacklistManage)
}
