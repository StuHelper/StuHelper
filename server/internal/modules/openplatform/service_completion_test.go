package openplatform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredProfileFieldsUseIdentityPortalActions(t *testing.T) {
	empty := ""
	unverified := "pending"
	fields := RequiredProfileFields(&UserProjection{
		Username:         "",
		Email:            "",
		AvatarURL:        &empty,
		PhoneVerified:    false,
		IdentityVerified: false,
		ProfileStatus:    &unverified,
	}, []string{
		ScopeProfileBasicRead,
		ScopeEmailRead,
		ScopePhoneRead,
		ScopeIdentityStatusRead,
		ScopeStudentStatusRead,
	})

	actionURLs := make(map[string]string, len(fields))
	for _, field := range fields {
		actionURLs[field.Key] = field.ActionURL
	}

	assert.Equal(t, "/account/profile", actionURLs[ProfileFieldUsername])
	assert.Equal(t, "/account/profile", actionURLs[ProfileFieldEmail])
	assert.Equal(t, "/account/profile", actionURLs[ProfileFieldAvatar])
	assert.Equal(t, "/user/phone-binding", actionURLs[ProfileFieldPhone])
	assert.Equal(t, "/user/identity-verification", actionURLs[ProfileFieldIdentity])
	assert.Equal(t, "/user/student-verification", actionURLs[ProfileFieldStudent])
	for _, actionURL := range actionURLs {
		assert.NotEqual(t, "/user", actionURL)
		assert.NotEqual(t, "/user/reviews", actionURL)
	}
}
