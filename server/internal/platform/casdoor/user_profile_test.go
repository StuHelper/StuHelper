package casdoor

import (
	"context"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserProfileClientUpdatePhone(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserProfile
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client, err := newUserProfileClient(credential, users, &fakeSMSAPI{})
	require.NoError(t, err)

	err = client.UpdatePhone(context.Background(), UserPhoneUpdate{
		Subject: "casdoor-subject-1",
		Phone:   "+8613800138000",
	})

	require.NoError(t, err)
	assert.Equal(t, "casdoor-subject-1", users.gotSubject)
	require.NotNil(t, users.updated)
	assert.Equal(t, "alice", users.updated.Name)
	assert.Equal(t, "+8613800138000", users.updated.Phone)
	assert.Equal(t, "CN", users.updated.CountryCode)
	assert.Equal(t, []string{"phone", "countryCode"}, users.columns)
}

func TestUserProfileClientRejectsMissingSubject(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserProfile
	client, err := newUserProfileClient(credential, &fakeUserAPI{}, &fakeSMSAPI{})
	require.NoError(t, err)

	err = client.UpdatePhone(context.Background(), UserPhoneUpdate{Phone: "+8613800138000"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user subject is required")
}

func TestUserProfileClientGetPhone(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserProfile
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice", Phone: " +8613800138000 "}}
	client, err := newUserProfileClient(credential, users, &fakeSMSAPI{})
	require.NoError(t, err)

	phone, err := client.GetPhone(context.Background(), "casdoor-subject-1")

	require.NoError(t, err)
	assert.Equal(t, "casdoor-subject-1", users.gotSubject)
	assert.Equal(t, "+8613800138000", phone)
}

func TestUserProfileClientSendSMS(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserProfile
	sms := &fakeSMSAPI{}
	client, err := newUserProfileClient(credential, &fakeUserAPI{}, sms)
	require.NoError(t, err)

	err = client.Send(context.Background(), "+8613800138000", "123456")

	require.NoError(t, err)
	assert.Equal(t, "123456", sms.content)
	assert.Equal(t, []string{"+8613800138000"}, sms.receivers)
}

type fakeSMSAPI struct {
	content   string
	receivers []string
}

func (f *fakeSMSAPI) SendSms(content string, receivers ...string) error {
	f.content = content
	f.receivers = append([]string(nil), receivers...)
	return nil
}
