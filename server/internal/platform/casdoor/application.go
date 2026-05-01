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
)

type ApplicationSpec struct {
	Name                 string
	DisplayName          string
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
	err = c.call(ctx, "create application "+app.Name, func() (bool, error) {
		return c.apps.AddApplication(app)
	})
	if err != nil {
		return err
	}
	auditCasdoorApplication(ctx, c.credential, app.Name, casdoorApplicationActionCreate)
	return nil
}

func (c *Client) UpdateApplication(ctx context.Context, spec ApplicationSpec) error {
	app, err := c.buildApplication(spec)
	if err != nil {
		return err
	}
	err = c.call(ctx, "update application "+app.Name, func() (bool, error) {
		return c.apps.UpdateApplication(app)
	})
	if err != nil {
		return err
	}
	auditCasdoorApplication(ctx, c.credential, app.Name, casdoorApplicationActionUpdate)
	return nil
}

func (c *Client) DeleteApplication(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if err := validateName("application name", name); err != nil {
		return err
	}
	app := &casdoorsdk.Application{Owner: c.credential.Organization, Name: name}
	err := c.call(ctx, "delete application "+name, func() (bool, error) {
		return c.apps.DeleteApplication(app)
	})
	if err != nil {
		return err
	}
	auditCasdoorApplication(ctx, c.credential, name, casdoorApplicationActionDelete)
	return nil
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
	return &casdoorsdk.Application{
		Owner:                c.credential.Organization,
		Name:                 normalized.Name,
		DisplayName:          normalized.DisplayName,
		Organization:         c.credential.Organization,
		ClientId:             normalized.ClientID,
		ClientSecret:         normalized.ClientSecret,
		RedirectUris:         normalized.RedirectURIs,
		GrantTypes:           normalized.GrantTypes,
		TokenFormat:          normalized.TokenFormat,
		TokenFields:          normalized.TokenFields,
		ExpireInHours:        normalized.ExpireInHours,
		RefreshExpireInHours: normalized.RefreshExpireInHours,
	}, nil
}

func normalizeApplicationSpec(spec ApplicationSpec) (ApplicationSpec, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	spec.DisplayName = strings.TrimSpace(spec.DisplayName)
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

func normalizeTokenFields(fields []string) ([]string, error) {
	if fields == nil {
		return nil, errors.New("casdoor: token fields must be explicit")
	}
	return normalizeList("token field", fields)
}
