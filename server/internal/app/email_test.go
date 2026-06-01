package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func TestNewSchoolEmailSenderUsesStudentVerificationSubjectDefault(t *testing.T) {
	sender, err := newSchoolEmailSender(config.EmailConfig{
		Enabled: true,
		Driver:  "blackhole",
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, sender)
	assert.Equal(t, "学生认证验证码", sender.subject)
}

func TestNewSchoolEmailSenderUsesConfiguredStudentVerificationSubject(t *testing.T) {
	sender, err := newSchoolEmailSender(config.EmailConfig{
		Enabled:                    true,
		Driver:                     "blackhole",
		StudentVerificationSubject: "BUAA 学生认证验证码",
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, sender)
	assert.Equal(t, "BUAA 学生认证验证码", sender.subject)
}
