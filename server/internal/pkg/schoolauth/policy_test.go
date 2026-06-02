package schoolauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeEmailAddressRequiresLocalAndDomain(t *testing.T) {
	assert.Equal(t, "student@buaa.edu.cn", NormalizeEmailAddress(" Student@BUAA.edu.cn "))
	assert.Empty(t, NormalizeEmailAddress("@buaa.edu.cn"))
	assert.Empty(t, NormalizeEmailAddress("student@"))
	assert.Empty(t, NormalizeEmailAddress("student@@buaa.edu.cn"))
	assert.Empty(t, NormalizeEmailAddress("stu dent@buaa.edu.cn"))
}

func TestEmailDomainAllowedRejectsEmptyEmailParts(t *testing.T) {
	domains := []string{"buaa.edu.cn"}

	assert.True(t, EmailDomainAllowed("student@BUAA.edu.cn", domains))
	assert.False(t, EmailDomainAllowed("@buaa.edu.cn", domains))
	assert.False(t, EmailDomainAllowed("student@", domains))
	assert.False(t, EmailDomainAllowed("student@other.example", domains))
}
