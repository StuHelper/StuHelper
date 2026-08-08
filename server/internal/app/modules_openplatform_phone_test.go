package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformcasdoor "github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

type recordingCasdoorUserProfileClient struct {
	phone      string
	err        error
	getCalls   int
	gotSubject string
}

func (c *recordingCasdoorUserProfileClient) GetPhone(_ context.Context, subject string) (string, error) {
	c.getCalls++
	c.gotSubject = subject
	return c.phone, c.err
}

func (*recordingCasdoorUserProfileClient) UpdatePhone(
	context.Context,
	platformcasdoor.UserPhoneUpdate,
) error {
	return nil
}

func (*recordingCasdoorUserProfileClient) ClearPhone(context.Context, string) error {
	return nil
}

func (*recordingCasdoorUserProfileClient) Send(context.Context, string, string) error {
	return nil
}

func TestCasdoorUserProfileGatewayResolvesInternalUserForPhoneLookup(t *testing.T) {
	client := &recordingCasdoorUserProfileClient{phone: "+8613800138000"}
	resolverCalls := 0
	gateway := newCasdoorUserProfileGateway(
		client,
		func(_ context.Context, userID int64) (string, error) {
			resolverCalls++
			assert.Equal(t, int64(42), userID)
			return " casdoor-subject-42 ", nil
		},
	)

	phone, err := gateway.GetPhone(context.Background(), 42)

	require.NoError(t, err)
	assert.Equal(t, "+8613800138000", phone)
	assert.Equal(t, 1, resolverCalls)
	assert.Equal(t, 1, client.getCalls)
	assert.Equal(t, "casdoor-subject-42", client.gotSubject)
}

func TestCasdoorUserProfileGatewayFailsClosedWhenSubjectResolutionFails(t *testing.T) {
	resolveErr := errors.New("identity mapping unavailable")
	client := &recordingCasdoorUserProfileClient{phone: "+8613800138000"}
	gateway := newCasdoorUserProfileGateway(
		client,
		func(context.Context, int64) (string, error) {
			return "", resolveErr
		},
	)

	phone, err := gateway.GetPhone(context.Background(), 42)

	require.ErrorIs(t, err, resolveErr)
	assert.Empty(t, phone)
	assert.Zero(t, client.getCalls)
}
