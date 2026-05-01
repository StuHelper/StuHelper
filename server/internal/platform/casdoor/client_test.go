package casdoor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppProvisioningClientValidatesPurpose(t *testing.T) {
	credential := validCredential()
	credential.Purpose = "casdoor-admin-role-sync"

	client, err := NewAppProvisioningClient(credential)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "unsupported admin purpose")
}

func TestNewAppProvisioningClientRequiresCredential(t *testing.T) {
	credential := validCredential()
	credential.ClientSecret = ""

	client, err := NewAppProvisioningClient(credential)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "admin client credential")
}

func TestNewAppProvisioningClientTrimsCredential(t *testing.T) {
	credential := validCredential()
	credential.Endpoint = " https://sso.example.com "

	client, err := NewAppProvisioningClient(credential)

	require.NoError(t, err)
	assert.Equal(t, "https://sso.example.com", client.credential.Endpoint)
}
