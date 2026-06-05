package serviceaccount

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifierVerifyUsesCredentialStore(t *testing.T) {
	store := &fakeCredentialStore{
		record: &credentialRecord{
			ID:        42,
			Name:      KoishiRuntimeCredentialName,
			Audiences: []string{AudienceBotAPI},
			Scopes:    []string{ScopeBotQQBindingConsume},
		},
	}
	verifier, err := newVerifier(store, []byte("test-service-account-hmac-key-32!"))
	require.NoError(t, err)

	err = verifier.Verify(context.Background(), "koishi-token", "/api/v1/bot/qq-binding/consume", ScopeBotQQBindingConsume)

	require.NoError(t, err)
	assert.NotEmpty(t, store.loadedTokenHash)
	assert.NotEqual(t, "koishi-token", store.loadedTokenHash)
	assert.Equal(t, int64(42), store.touchedID)
}

func TestNewVerifierRequiresCredentialStore(t *testing.T) {
	verifier, err := newVerifier(nil, []byte("test-service-account-hmac-key-32!"))

	require.Error(t, err)
	assert.Nil(t, verifier)
	assert.Contains(t, err.Error(), "credential store is required")
}

type fakeCredentialStore struct {
	record          *credentialRecord
	loadErr         error
	touchErr        error
	loadedTokenHash string
	touchedID       int64
}

func (f *fakeCredentialStore) EnsureBootstrapCredential(
	context.Context,
	BootstrapCredential,
	string,
) (BootstrapResult, error) {
	return BootstrapResult{}, nil
}

func (f *fakeCredentialStore) RevokeCredential(context.Context, string) (int64, error) {
	return 0, nil
}

func (f *fakeCredentialStore) LoadCredentialByTokenHash(_ context.Context, tokenHash string) (*credentialRecord, error) {
	f.loadedTokenHash = tokenHash
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.record, nil
}

func (f *fakeCredentialStore) TouchLastUsed(_ context.Context, id int64) error {
	f.touchedID = id
	return f.touchErr
}
