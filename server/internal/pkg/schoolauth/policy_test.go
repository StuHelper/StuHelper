package schoolauth

import (
	"strings"
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

func TestStudentIDValidationUsesEmailSafeCanonicalForm(t *testing.T) {
	for _, value := range []string{"20250001", "S-2025_01", "student.id"} {
		assert.True(t, IsValidStudentID(value), value)
	}
	for _, value := range []string{
		"",
		" student id ",
		"student@id",
		"-student",
		"学号2025",
		"student/id",
		string([]byte{0xff}),
	} {
		assert.False(t, IsValidStudentID(value), value)
	}
	assert.False(t, IsValidStudentID("1"+strings.Repeat("2", MaxStudentIDRunes)))
	assert.Equal(t, "s-2025_01@buaa.edu.cn", DeriveStudentEmail(" S-2025_01 ", "BUAA.EDU.CN"))
	assert.Empty(t, DeriveStudentEmail("student/id", "buaa.edu.cn"))
}

func TestAcademicNameValidationRejectsUnsafeOrOversizedValues(t *testing.T) {
	assert.True(t, IsValidAcademicName(" 张 三 "))
	assert.True(t, IsValidAcademicName("Mary-Jane O'Neil"))
	assert.False(t, IsValidAcademicName(""))
	assert.False(t, IsValidAcademicName("张\u200b三"))
	assert.False(t, IsValidAcademicName("张\x00三"))
	assert.False(t, IsValidAcademicName(strings.Repeat("张", MaxAcademicNameRunes+1)))
}
