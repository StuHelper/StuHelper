package casdoor

import (
	"context"
	"errors"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateApplicationMapsSpec(t *testing.T) {
	api := &fakeApplicationAPI{addOK: true}
	client := newTestClient(t, api)
	spec := validApplicationSpec()
	spec.RedirectURIs = append(spec.RedirectURIs, " https://app.example.com/callback ")
	spec.GrantTypes = append(spec.GrantTypes, "authorization_code")

	err := client.CreateApplication(context.Background(), spec)

	require.NoError(t, err)
	require.NotNil(t, api.added)
	assert.Equal(t, "admin", api.added.Owner)
	assert.Equal(t, "third-party-demo", api.added.Name)
	assert.Equal(t, "Third Party Demo", api.added.DisplayName)
	assert.Equal(t, "https://static.example.com/logo.png", api.added.Logo)
	assert.Equal(t, "https://app.example.com", api.added.HomepageUrl)
	assert.Equal(t, "Demo app", api.added.Description)
	assert.Equal(t, "stuhelper", api.added.Organization)
	assert.Equal(t, "cert-built-in", api.added.Cert)
	assert.True(t, api.added.EnablePassword)
	assert.True(t, api.added.EnableSignUp)
	assert.True(t, api.added.EnableSigninSession)
	assert.Equal(t, "client-demo", api.added.ClientId)
	assert.Equal(t, "secret-demo", api.added.ClientSecret)
	assert.Equal(t, []string{"https://app.example.com/callback"}, api.added.RedirectUris)
	assert.NotNil(t, api.added.Providers)
	assert.NotEmpty(t, api.added.SigninMethods)
	assert.NotEmpty(t, api.added.SignupItems)
	assert.NotEmpty(t, api.added.SigninItems)
	assert.Equal(t, []string{"authorization_code"}, api.added.GrantTypes)
	assert.Equal(t, "JWT", api.added.TokenFormat)
	assert.Equal(t, []string{}, api.added.TokenFields)
	assert.NotNil(t, api.added.TokenAttributes)
	assert.NotNil(t, api.added.Tags)
	assert.NotNil(t, api.added.SamlAttributes)
}

func TestUpdateApplicationDelegatesToSDK(t *testing.T) {
	api := &fakeApplicationAPI{updateOK: true}
	client := newTestClient(t, api)

	err := client.UpdateApplication(context.Background(), validApplicationSpec())

	require.NoError(t, err)
	assert.Equal(t, "third-party-demo", api.gotName)
	require.NotNil(t, api.updated)
	assert.Equal(t, "third-party-demo", api.updated.Name)
}

func TestCasdoorApplicationAuditActionDetectsSecretRotation(t *testing.T) {
	desired := &casdoorsdk.Application{Name: "third-party-demo", ClientSecret: "new-secret"}

	action := casdoorApplicationAuditAction(
		&casdoorsdk.Application{Name: "third-party-demo", ClientSecret: "old-secret"},
		desired,
	)
	assert.Equal(t, casdoorApplicationActionRotate, action)

	action = casdoorApplicationAuditAction(
		&casdoorsdk.Application{Name: "third-party-demo", ClientSecret: ""},
		desired,
	)
	assert.Equal(t, casdoorApplicationActionUpdate, action)

	action = casdoorApplicationAuditAction(nil, desired)
	assert.Equal(t, casdoorApplicationActionUpdate, action)
}

func TestDeleteApplicationUsesOwnedApplicationKey(t *testing.T) {
	api := &fakeApplicationAPI{deleteOK: true}
	client := newTestClient(t, api)

	err := client.DeleteApplication(context.Background(), "third-party-demo")

	require.NoError(t, err)
	require.NotNil(t, api.deleted)
	assert.Equal(t, "admin", api.deleted.Owner)
	assert.Equal(t, "third-party-demo", api.deleted.Name)
}

func TestApplicationValidationRejectsUnsafeRedirect(t *testing.T) {
	client := newTestClient(t, &fakeApplicationAPI{addOK: true})
	spec := validApplicationSpec()
	spec.RedirectURIs = []string{"https://*.example.com/callback"}

	err := client.CreateApplication(context.Background(), spec)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard redirect URI")
}

func TestApplicationValidationRequiresExplicitTokenFields(t *testing.T) {
	client := newTestClient(t, &fakeApplicationAPI{addOK: true})
	spec := validApplicationSpec()
	spec.TokenFields = nil

	err := client.CreateApplication(context.Background(), spec)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token fields must be explicit")
}

func TestApplicationValidationAllowsClientCredentialsWithoutRedirect(t *testing.T) {
	api := &fakeApplicationAPI{addOK: true}
	client := newTestClient(t, api)
	spec := validApplicationSpec()
	spec.Name = "casdoor-admin-role-sync"
	spec.GrantTypes = []string{"client_credentials"}
	spec.RedirectURIs = nil

	err := client.CreateApplication(context.Background(), spec)

	require.NoError(t, err)
	require.NotNil(t, api.added)
	assert.Empty(t, api.added.RedirectUris)
	assert.Equal(t, []string{"client_credentials"}, api.added.GrantTypes)
	assert.False(t, api.added.EnablePassword)
	assert.False(t, api.added.EnableSignUp)
	assert.False(t, api.added.EnableSigninSession)
	assert.NotNil(t, api.added.Providers)
	assert.Empty(t, api.added.SigninMethods)
	assert.Empty(t, api.added.SignupItems)
	assert.Empty(t, api.added.SigninItems)
}

func TestApplicationOperationRejected(t *testing.T) {
	client := newTestClient(t, &fakeApplicationAPI{addOK: false})

	err := client.CreateApplication(context.Background(), validApplicationSpec())

	require.ErrorIs(t, err, ErrOperationRejected)
}

func TestApplicationOperationError(t *testing.T) {
	client := newTestClient(t, &fakeApplicationAPI{addErr: errors.New("api down")})

	err := client.CreateApplication(context.Background(), validApplicationSpec())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "api down")
}

func validApplicationSpec() ApplicationSpec {
	return ApplicationSpec{
		Name:                 "third-party-demo",
		DisplayName:          "Third Party Demo",
		Logo:                 "https://static.example.com/logo.png",
		HomepageURL:          "https://app.example.com",
		Description:          "Demo app",
		ClientID:             "client-demo",
		ClientSecret:         "secret-demo",
		RedirectURIs:         []string{"https://app.example.com/callback"},
		GrantTypes:           []string{"authorization_code"},
		TokenFormat:          "JWT",
		TokenFields:          []string{},
		ExpireInHours:        1,
		RefreshExpireInHours: 24,
	}
}

func newTestClient(t *testing.T, api *fakeApplicationAPI) *Client {
	t.Helper()
	client, err := newClient(validCredential(), api)
	require.NoError(t, err)
	return client
}

func validCredential() Credential {
	return Credential{
		Purpose:      PurposeAppProvisioning,
		Endpoint:     "https://sso.example.com",
		ClientID:     "admin-client",
		ClientSecret: "admin-secret",
		Organization: "stuhelper",
		Application:  "casdoor-admin-app-provisioning",
	}
}

type fakeApplicationAPI struct {
	gotName  string
	existing *casdoorsdk.Application
	added    *casdoorsdk.Application
	updated  *casdoorsdk.Application
	deleted  *casdoorsdk.Application
	addOK    bool
	updateOK bool
	deleteOK bool
	addErr   error
}

func (f *fakeApplicationAPI) GetApplication(name string) (*casdoorsdk.Application, error) {
	f.gotName = name
	return f.existing, nil
}

func (f *fakeApplicationAPI) AddApplication(app *casdoorsdk.Application) (bool, error) {
	f.added = app
	return f.addOK, f.addErr
}

func (f *fakeApplicationAPI) UpdateApplication(app *casdoorsdk.Application) (bool, error) {
	f.updated = app
	return f.updateOK, nil
}

func (f *fakeApplicationAPI) DeleteApplication(app *casdoorsdk.Application) (bool, error) {
	f.deleted = app
	return f.deleteOK, nil
}
