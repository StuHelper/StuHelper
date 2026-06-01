package casdoor

import (
	"context"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapCreatesMissingObjects(t *testing.T) {
	apps := &fakeApplicationAPI{addOK: true}
	roles := &fakeRoleAPI{}
	orgs := &fakeOrganizationAPI{addOK: true}
	providers := &fakeProviderAPI{addOK: true}
	client := newBootstrapTestClient(t, bootstrapAPIs{apps: apps, roles: roles, orgs: orgs, providers: providers})

	err := client.Bootstrap(context.Background(), validBootstrapPlan())

	require.NoError(t, err)
	require.NotNil(t, orgs.added)
	assert.Equal(t, "stuhelper", orgs.added.Name)
	assert.Equal(t, "bcrypt", orgs.added.PasswordType)
	assert.NotEmpty(t, orgs.added.PasswordOptions)
	assert.NotEmpty(t, orgs.added.CountryCodes)
	assert.NotNil(t, orgs.added.UserTypes)
	assert.NotNil(t, orgs.added.Tags)
	assert.NotEmpty(t, orgs.added.Languages)
	assert.NotNil(t, orgs.added.NavItems)
	assert.NotNil(t, orgs.added.UserNavItems)
	assert.NotNil(t, orgs.added.WidgetItems)
	assert.NotNil(t, orgs.added.MfaItems)
	assert.NotEmpty(t, orgs.added.AccountItems)
	assert.NotNil(t, orgs.added.ThemeData)
	require.NotNil(t, apps.added)
	assert.Equal(t, "stuhelper-web", apps.added.Name)
	assert.Equal(t, "stuhelper", apps.added.Organization)
	require.NotNil(t, roles.added)
	assert.Equal(t, "verified_student", roles.added.Name)
	assert.Equal(t, "stuhelper", roles.added.Owner)
	require.NotNil(t, providers.added)
	assert.Equal(t, "stuhelper-sms", providers.added.Name)
	assert.Equal(t, "stuhelper", providers.added.Owner)
}

func TestBootstrapUpdatesExistingObjects(t *testing.T) {
	apps := &fakeApplicationAPI{existing: &casdoorsdk.Application{Name: "stuhelper-web"}, updateOK: true}
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "verified_student", DisplayName: "old"}}
	orgs := &fakeOrganizationAPI{existing: &casdoorsdk.Organization{Name: "stuhelper"}, updateOK: true}
	providers := &fakeProviderAPI{existing: &casdoorsdk.Provider{Name: "stuhelper-sms"}, updateOK: true}
	client := newBootstrapTestClient(t, bootstrapAPIs{apps: apps, roles: roles, orgs: orgs, providers: providers})

	err := client.Bootstrap(context.Background(), validBootstrapPlan())

	require.NoError(t, err)
	assert.Nil(t, orgs.added)
	assert.Equal(t, "StuHelper", orgs.updated.DisplayName)
	assert.Nil(t, apps.added)
	assert.Equal(t, "stuhelper-web", apps.updated.Name)
	assert.Nil(t, roles.added)
	assert.Equal(t, "Verified Student", roles.updated.DisplayName)
	assert.Nil(t, providers.added)
	assert.Equal(t, "https://api.example.com/internal/sms/send", providers.updated.Endpoint)
}

func TestBootstrapUsesTargetOrganizationWhenCredentialIsBuiltIn(t *testing.T) {
	apps := &fakeApplicationAPI{addOK: true}
	roles := &fakeRoleAPI{}
	orgs := &fakeOrganizationAPI{addOK: true}
	providers := &fakeProviderAPI{addOK: true}
	credential := validCredential()
	credential.Purpose = PurposeBootstrap
	credential.Organization = "built-in"
	client, err := newBootstrapClient(credential, apps, roles, orgs, providers)
	require.NoError(t, err)

	err = client.Bootstrap(context.Background(), validBootstrapPlan())

	require.NoError(t, err)
	require.NotNil(t, apps.added)
	assert.Equal(t, "stuhelper", apps.added.Organization)
	require.NotNil(t, roles.added)
	assert.Equal(t, "stuhelper", roles.added.Owner)
	require.NotNil(t, providers.added)
	assert.Equal(t, "stuhelper", providers.added.Owner)
}

func TestNewBootstrapClientRequiresBootstrapPurpose(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeAppProvisioning

	client, err := NewBootstrapClient(credential)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "credential purpose")
}

func TestBootstrapRejectsIncompleteProvider(t *testing.T) {
	client := newBootstrapTestClient(t, bootstrapAPIs{
		apps:      &fakeApplicationAPI{},
		roles:     &fakeRoleAPI{},
		orgs:      &fakeOrganizationAPI{},
		providers: &fakeProviderAPI{},
	})
	provider := validBootstrapPlan().Providers[0]
	provider.Category = ""

	err := client.EnsureProvider(context.Background(), provider)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider display name, category, and type are required")
}

func validBootstrapPlan() BootstrapPlan {
	return BootstrapPlan{
		Organization: OrganizationSpec{
			Name:               "stuhelper",
			DisplayName:        "StuHelper",
			DefaultApplication: "stuhelper-web",
		},
		Applications: []ApplicationSpec{{
			Name:                 "stuhelper-web",
			DisplayName:          "StuHelper Web",
			ClientID:             "stuhelper-web",
			ClientSecret:         "secret",
			RedirectURIs:         []string{"https://api.example.com/api/v1/auth/callback"},
			GrantTypes:           []string{"authorization_code", "refresh_token"},
			TokenFormat:          "JWT",
			TokenFields:          []string{},
			ExpireInHours:        1,
			RefreshExpireInHours: 24,
		}},
		Roles: []RoleSpec{{
			Name:        "verified_student",
			DisplayName: "Verified Student",
			Description: "StuHelper verified student projection",
		}},
		Providers: []ProviderSpec{{
			Name:        "stuhelper-sms",
			DisplayName: "StuHelper SMS",
			Category:    "SMS",
			Type:        "Custom HTTP SMS",
			Method:      "POST",
			Title:       "content",
			Endpoint:    "https://api.example.com/internal/sms/send",
		}},
	}
}

type bootstrapAPIs struct {
	apps      *fakeApplicationAPI
	roles     *fakeRoleAPI
	orgs      *fakeOrganizationAPI
	providers *fakeProviderAPI
}

func newBootstrapTestClient(t *testing.T, apis bootstrapAPIs) *Client {
	t.Helper()
	credential := validCredential()
	credential.Purpose = PurposeBootstrap
	client, err := newBootstrapClient(credential, apis.apps, apis.roles, apis.orgs, apis.providers)
	require.NoError(t, err)
	return client
}

type fakeOrganizationAPI struct {
	existing *casdoorsdk.Organization
	added    *casdoorsdk.Organization
	updated  *casdoorsdk.Organization
	addOK    bool
	updateOK bool
}

func (f *fakeOrganizationAPI) GetOrganization(string) (*casdoorsdk.Organization, error) {
	return f.existing, nil
}

func (f *fakeOrganizationAPI) AddOrganization(org *casdoorsdk.Organization) (bool, error) {
	f.added = org
	return f.addOK, nil
}

func (f *fakeOrganizationAPI) UpdateOrganization(org *casdoorsdk.Organization) (bool, error) {
	f.updated = org
	return f.updateOK, nil
}

type fakeProviderAPI struct {
	existing *casdoorsdk.Provider
	added    *casdoorsdk.Provider
	updated  *casdoorsdk.Provider
	addOK    bool
	updateOK bool
}

func (f *fakeProviderAPI) GetProvider(string) (*casdoorsdk.Provider, error) {
	return f.existing, nil
}

func (f *fakeProviderAPI) AddProvider(provider *casdoorsdk.Provider) (bool, error) {
	f.added = provider
	return f.addOK, nil
}

func (f *fakeProviderAPI) UpdateProvider(provider *casdoorsdk.Provider) (bool, error) {
	f.updated = provider
	return f.updateOK, nil
}
