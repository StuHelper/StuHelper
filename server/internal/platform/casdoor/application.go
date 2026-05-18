package casdoor

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
)

const (
	casdoorApplicationResourceType = "casdoor.application"
	casdoorApplicationActionCreate = "created"
	casdoorApplicationActionUpdate = "updated"
	casdoorApplicationActionDelete = "deleted"
	casdoorApplicationActionRotate = "secret_rotated"
)

type ApplicationSpec struct {
	Name                 string
	DisplayName          string
	Logo                 string
	HomepageURL          string
	Description          string
	ClientID             string
	ClientSecret         string
	RedirectURIs         []string
	GrantTypes           []string
	TokenFormat          string
	TokenFields          []string
	ExpireInHours        float64
	RefreshExpireInHours float64
}

func (c *Client) CreateApplication(ctx context.Context, spec ApplicationSpec) error {
	app, err := c.buildApplication(spec)
	if err != nil {
		return err
	}
	return c.createApplication(ctx, app)
}

func (c *Client) UpdateApplication(ctx context.Context, spec ApplicationSpec) error {
	app, err := c.buildApplication(spec)
	if err != nil {
		return err
	}
	existing, err := c.getApplication(ctx, app.Name)
	if err != nil {
		return err
	}
	return c.updateApplication(ctx, app, existing)
}

func (c *Client) DeleteApplication(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if err := validateName("application name", name); err != nil {
		return err
	}
	app := &casdoorsdk.Application{Owner: "admin", Name: name}
	err := c.call(ctx, "delete application "+name, func() (bool, error) {
		return c.apps.DeleteApplication(app)
	})
	if err != nil {
		return err
	}
	auditCasdoorApplication(ctx, c.credential, name, casdoorApplicationActionDelete)
	return nil
}

func (c *Client) createApplication(ctx context.Context, app *casdoorsdk.Application) error {
	err := c.call(ctx, "create application "+app.Name, func() (bool, error) {
		return c.apps.AddApplication(app)
	})
	if err != nil {
		return err
	}
	auditCasdoorApplication(ctx, c.credential, app.Name, casdoorApplicationActionCreate)
	return nil
}

func (c *Client) updateApplication(
	ctx context.Context,
	app *casdoorsdk.Application,
	existing *casdoorsdk.Application,
) error {
	err := c.call(ctx, "update application "+app.Name, func() (bool, error) {
		return c.apps.UpdateApplication(app)
	})
	if err != nil {
		return err
	}
	action := casdoorApplicationAuditAction(existing, app)
	auditCasdoorApplication(ctx, c.credential, app.Name, action)
	return nil
}

func casdoorApplicationAuditAction(existing, desired *casdoorsdk.Application) string {
	if applicationSecretRotated(existing, desired) {
		return casdoorApplicationActionRotate
	}
	return casdoorApplicationActionUpdate
}

func applicationSecretRotated(existing, desired *casdoorsdk.Application) bool {
	if existing == nil || desired == nil {
		return false
	}
	oldSecret := strings.TrimSpace(existing.ClientSecret)
	newSecret := strings.TrimSpace(desired.ClientSecret)
	return oldSecret != "" && newSecret != "" && oldSecret != newSecret
}

func auditCasdoorApplication(ctx context.Context, credential Credential, appName string, action string) {
	audit.Log(audit.EventFromContext(ctx, casdoorApplicationAuditEvent(credential, appName, action)))
}

func casdoorApplicationAuditEvent(credential Credential, appName string, action string) audit.Event {
	return audit.Event{
		Type:         audit.EventType("iam.casdoor_app." + action),
		Category:     "admin_operation",
		ActorType:    "system",
		UserID:       string(credential.Purpose),
		ResourceType: casdoorApplicationResourceType,
		ResourceID:   appName,
		Action:       action,
		Result:       "success",
		Details: map[string]any{
			"purpose":           string(credential.Purpose),
			"organization":      credential.Organization,
			"admin_application": credential.Application,
			"app_name":          appName,
		},
	}
}

func (c *Client) buildApplication(spec ApplicationSpec) (*casdoorsdk.Application, error) {
	normalized, err := normalizeApplicationSpec(spec)
	if err != nil {
		return nil, err
	}
	interactive := applicationGrantRequiresRedirect(normalized.GrantTypes)
	return &casdoorsdk.Application{
		Owner:                "admin",
		Name:                 normalized.Name,
		DisplayName:          normalized.DisplayName,
		Logo:                 normalized.Logo,
		HomepageUrl:          normalized.HomepageURL,
		Description:          normalized.Description,
		Organization:         c.credential.Organization,
		Cert:                 defaultApplicationCertificate,
		EnablePassword:       interactive,
		EnableSignUp:         interactive,
		EnableSigninSession:  interactive,
		ClientId:             normalized.ClientID,
		ClientSecret:         normalized.ClientSecret,
		RedirectUris:         normalized.RedirectURIs,
		Providers:            defaultProviderItems(),
		SigninMethods:        defaultSigninMethods(interactive),
		SignupItems:          defaultSignupItems(interactive),
		SigninItems:          defaultSigninItems(interactive),
		GrantTypes:           normalized.GrantTypes,
		TokenFormat:          normalized.TokenFormat,
		TokenFields:          normalized.TokenFields,
		TokenAttributes:      []*casdoorsdk.JwtItem{},
		Tags:                 []string{},
		SamlAttributes:       []*casdoorsdk.SamlItem{},
		ExpireInHours:        normalized.ExpireInHours,
		RefreshExpireInHours: normalized.RefreshExpireInHours,
	}, nil
}

func normalizeApplicationSpec(spec ApplicationSpec) (ApplicationSpec, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	spec.DisplayName = strings.TrimSpace(spec.DisplayName)
	spec.Logo = strings.TrimSpace(spec.Logo)
	spec.HomepageURL = strings.TrimSpace(spec.HomepageURL)
	spec.Description = strings.TrimSpace(spec.Description)
	spec.ClientID = strings.TrimSpace(spec.ClientID)
	spec.ClientSecret = strings.TrimSpace(spec.ClientSecret)
	spec.TokenFormat = strings.TrimSpace(spec.TokenFormat)
	if err := validateApplicationRequiredFields(spec); err != nil {
		return ApplicationSpec{}, err
	}
	grants, err := normalizeNonEmptyList("grant type", spec.GrantTypes)
	if err != nil {
		return ApplicationSpec{}, err
	}
	redirects, err := normalizeRedirectURIsForGrants(spec.RedirectURIs, grants)
	if err != nil {
		return ApplicationSpec{}, err
	}
	fields, err := normalizeTokenFields(spec.TokenFields)
	if err != nil {
		return ApplicationSpec{}, err
	}
	if err := validateHomepageURL(spec.HomepageURL); err != nil {
		return ApplicationSpec{}, err
	}
	spec.RedirectURIs = redirects
	spec.GrantTypes = grants
	spec.TokenFields = fields
	return spec, nil
}

func validateApplicationRequiredFields(spec ApplicationSpec) error {
	if err := validateName("application name", spec.Name); err != nil {
		return err
	}
	if spec.DisplayName == "" {
		return errors.New("casdoor: application display name is required")
	}
	if spec.ClientID == "" {
		return errors.New("casdoor: application client ID is required")
	}
	if spec.TokenFormat == "" {
		return errors.New("casdoor: application token format is required")
	}
	if spec.ExpireInHours <= 0 || spec.RefreshExpireInHours < 0 {
		return errors.New("casdoor: invalid application token TTL")
	}
	return nil
}

func normalizeRedirectURIs(values []string) ([]string, error) {
	redirects, err := normalizeNonEmptyList("redirect URI", values)
	if err != nil {
		return nil, err
	}
	for _, redirect := range redirects {
		if err := validateRedirectURI(redirect); err != nil {
			return nil, err
		}
	}
	return redirects, nil
}

func normalizeRedirectURIsForGrants(values []string, grantTypes []string) ([]string, error) {
	if applicationGrantRequiresRedirect(grantTypes) {
		return normalizeRedirectURIs(values)
	}
	redirects, err := normalizeList("redirect URI", values)
	if err != nil {
		return nil, err
	}
	for _, redirect := range redirects {
		if err := validateRedirectURI(redirect); err != nil {
			return nil, err
		}
	}
	return redirects, nil
}

func applicationGrantRequiresRedirect(grantTypes []string) bool {
	return containsString(grantTypes, "authorization_code") || containsString(grantTypes, "implicit")
}

func validateRedirectURI(redirect string) error {
	if strings.Contains(redirect, "*") {
		return fmt.Errorf("casdoor: wildcard redirect URI is forbidden: %s", redirect)
	}
	parsed, err := url.Parse(redirect)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("casdoor: invalid redirect URI: %s", redirect)
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("casdoor: redirect URI must not contain fragment: %s", redirect)
	}
	return nil
}

func validateHomepageURL(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("casdoor: invalid homepage URL: %s", value)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("casdoor: homepage URL must use http or https: %s", value)
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("casdoor: homepage URL must not contain fragment: %s", value)
	}
	return nil
}

func normalizeTokenFields(fields []string) ([]string, error) {
	if fields == nil {
		return nil, errors.New("casdoor: token fields must be explicit")
	}
	return normalizeList("token field", fields)
}
