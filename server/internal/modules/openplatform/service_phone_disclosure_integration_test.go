package openplatform

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

type recordingAuthoritativePhoneReader struct {
	phone  string
	err    error
	calls  int
	userID int64
}

func (r *recordingAuthoritativePhoneReader) GetPhone(_ context.Context, userID int64) (string, error) {
	r.calls++
	r.userID = userID
	return r.phone, r.err
}

func TestAddPhonePayloadUsesAuthoritativePhoneReader(t *testing.T) {
	reader := &recordingAuthoritativePhoneReader{phone: " +86 138-0013-8000 "}
	service := &Service{phoneReader: reader}
	payload := map[string]any{}

	err := service.addPhonePayload(context.Background(), payload, &UserProjection{
		UserID:        42,
		PhoneVerified: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, reader.calls)
	assert.Equal(t, int64(42), reader.userID)
	assert.Equal(t, "13800138000", payload["phone"])
	assert.Equal(t, "138****8000", payload["phoneMasked"])
	assert.Equal(t, true, payload["phoneVerified"])
}

func TestAddPhonePayloadFailsClosedForUnavailableAuthoritativePhone(t *testing.T) {
	providerFailure := errors.New("provider unavailable")
	tests := []struct {
		name       string
		service    *Service
		projection *UserProjection
		reader     *recordingAuthoritativePhoneReader
	}{
		{
			name:       "reader is not configured",
			service:    &Service{},
			projection: &UserProjection{UserID: 41, PhoneVerified: true},
		},
		{
			name:    "internal user identity is missing",
			reader:  &recordingAuthoritativePhoneReader{phone: "13800138000"},
			service: &Service{},
			projection: &UserProjection{
				PhoneVerified: true,
			},
		},
		{
			name:    "Casdoor lookup fails",
			reader:  &recordingAuthoritativePhoneReader{err: providerFailure},
			service: &Service{},
			projection: &UserProjection{
				UserID:        42,
				PhoneVerified: true,
			},
		},
		{
			name:    "Casdoor phone is empty",
			reader:  &recordingAuthoritativePhoneReader{},
			service: &Service{},
			projection: &UserProjection{
				UserID:        43,
				PhoneVerified: true,
			},
		},
		{
			name:    "Casdoor phone has invalid format",
			reader:  &recordingAuthoritativePhoneReader{phone: "not-a-phone"},
			service: &Service{},
			projection: &UserProjection{
				UserID:        44,
				PhoneVerified: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.reader != nil {
				tt.service.phoneReader = tt.reader
			}
			payload := map[string]any{}
			err := tt.service.addPhonePayload(context.Background(), payload, tt.projection)

			require.ErrorIs(t, err, ErrDisclosureUnavailable)
			assert.NotContains(t, payload, "phone")
			assert.NotContains(t, payload, "phoneMasked")
			assert.NotContains(t, payload, "phoneVerified")
		})
	}
}

func TestAddPhonePayloadPreservesUnverifiedProjectionWithoutProviderCall(t *testing.T) {
	reader := &recordingAuthoritativePhoneReader{phone: "13800138000"}
	service := &Service{phoneReader: reader}
	payload := map[string]any{}

	err := service.addPhonePayload(context.Background(), payload, &UserProjection{
		UserID:        45,
		PhoneVerified: false,
	})

	require.NoError(t, err)
	assert.Zero(t, reader.calls)
	assert.Equal(t, false, payload["phoneVerified"])
	assert.NotContains(t, payload, "phone")
	assert.NotContains(t, payload, "phoneMasked")
}

func TestPhoneDisclosureUsesInternalUserIDForAuthoritativeLookup(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	reader := &recordingAuthoritativePhoneReader{phone: "+8613900139000"}
	service, err := NewService(
		repo,
		redis.Client,
		WithAuthoritativePhoneReader(reader),
	)
	require.NoError(t, err)

	ownerID := seedOpenPlatformUser(t, postgres, "phone-owner")
	userID := seedOpenPlatformUser(t, postgres, "phone-viewer")
	_, err = postgres.DB.Exec(ctx, `
		UPDATE users
		SET phone_enc = decode('0101', 'hex'),
		    phone_hash = $2,
		    phone_masked = '+86 139****9000',
		    phone_projection_state = 'synced',
		    phone_projection_synced_at = NOW(),
		    phone_encryption_key_version = 1,
		    phone_hmac_key_version = 1
		WHERE id = $1
	`, userID, strings.Repeat("a", 64))
	require.NoError(t, err)
	_, err = postgres.DB.Exec(ctx, `
		INSERT INTO phone_verification_credentials (
		    id, user_id, phone_hash, phone_display, method, assurance,
		    status, verified_at, last_confirmed_at
		)
		VALUES (
		    '00000000-0000-4000-8000-000000000002', $1, $2,
		    '+86 139****9000', 'sms_possession', 'current_possession',
		    'active', NOW(), NOW()
		)
	`, userID, strings.Repeat("a", 64))
	require.NoError(t, err)
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{ScopePhoneRead})
	require.NoError(t, repo.GrantConsents(ctx, Consent{
		AppID:       app.ID,
		UserID:      userID,
		GrantSource: "web",
		RequestID:   "phone-authoritative-consent",
	}, []string{ScopePhoneRead}))

	request := DisclosureRequest{
		ClientID:              app.ClientID,
		AuthenticatedClientID: app.ClientID,
		AuthenticatedByBearer: true,
		AccessTokenScopes:     []string{ScopePhoneRead},
		UserID:                userID,
		Scopes:                []string{ScopePhoneRead},
		RequestID:             "phone-authoritative-success",
	}
	payload, err := service.Phone(ctx, request)

	require.NoError(t, err)
	assert.Equal(t, userID, reader.userID)
	assert.Equal(t, "13900139000", payload["phone"])
	assert.Equal(t, "139****9000", payload["phoneMasked"])
	assert.Equal(t, true, payload["phoneVerified"])
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.granted", 1)

	identityPayload, err := service.UserInfoForIdentityToken(
		ctx,
		app.ClientID,
		userID,
		"oidc-subject-for-token",
		[]string{"openid", "phone"},
	)

	require.NoError(t, err)
	assert.Equal(t, "oidc-subject-for-token", identityPayload["sub"])
	assert.Equal(t, "13900139000", identityPayload["phone"])
	assert.Equal(t, true, identityPayload["phoneVerified"])
	assert.Equal(t, 2, reader.calls)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.granted", 2)

	reader.err = errors.New("Casdoor unavailable")
	request.RequestID = "phone-authoritative-failure"
	payload, err = service.Phone(ctx, request)

	require.ErrorIs(t, err, ErrDisclosureUnavailable)
	assert.Nil(t, payload)
	assert.Equal(t, 3, reader.calls)
	assertOpenPlatformAuditCount(t, postgres, app.ID, userID, "open_platform.disclosure.denied", 1)
}
