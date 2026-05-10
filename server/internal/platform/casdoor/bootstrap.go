package casdoor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

type OrganizationSpec struct {
	Name               string
	DisplayName        string
	DefaultApplication string
}

type RoleSpec struct {
	Name        string
	DisplayName string
	Description string
}

type ProviderSpec struct {
	Name        string
	DisplayName string
	Category    string
	Type        string
	SubType     string
	Method      string
	ProviderURL string
	Endpoint    string
	Host        string
	Port        int
	DisableSSL  bool
	Title       string
	Content     string
	Metadata    string
}

type BootstrapPlan struct {
	Organization OrganizationSpec
	Applications []ApplicationSpec
	Roles        []RoleSpec
	Providers    []ProviderSpec
}

func (c *Client) Bootstrap(ctx context.Context, plan BootstrapPlan) error {
	if err := c.EnsureOrganization(ctx, plan.Organization); err != nil {
		return err
	}
	for _, app := range plan.Applications {
		if err := c.EnsureApplication(ctx, app); err != nil {
			return err
		}
	}
	for _, role := range plan.Roles {
		if err := c.EnsureRole(ctx, role); err != nil {
			return err
		}
	}
	for _, provider := range plan.Providers {
		if err := c.EnsureProvider(ctx, provider); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) EnsureOrganization(ctx context.Context, spec OrganizationSpec) error {
	org, err := c.buildOrganization(spec)
	if err != nil {
		return err
	}
	existing, err := c.getOrganization(ctx, org.Name)
	if err != nil {
		return err
	}
	if existing == nil {
		return c.call(ctx, "create organization "+org.Name, func() (bool, error) {
			return c.orgs.AddOrganization(org)
		})
	}
	return c.call(ctx, "update organization "+org.Name, func() (bool, error) {
		return c.orgs.UpdateOrganization(org)
	})
}

func (c *Client) EnsureApplication(ctx context.Context, spec ApplicationSpec) error {
	app, err := c.buildApplication(spec)
	if err != nil {
		return err
	}
	existing, err := c.getApplication(ctx, app.Name)
	if err != nil {
		return err
	}
	if existing == nil {
		return c.createApplication(ctx, app)
	}
	return c.updateApplication(ctx, app, existing)
}

func (c *Client) EnsureRole(ctx context.Context, spec RoleSpec) error {
	role, err := c.buildRole(spec)
	if err != nil {
		return err
	}
	existing, err := c.getRoleForBootstrap(ctx, role.Name)
	if err != nil {
		return err
	}
	if existing == nil {
		return c.call(ctx, "create role "+role.Name, func() (bool, error) {
			return c.roles.AddRole(role)
		})
	}
	existing.DisplayName = role.DisplayName
	existing.Description = role.Description
	existing.IsEnabled = role.IsEnabled
	return c.call(ctx, "update role "+role.Name, func() (bool, error) {
		return c.roles.UpdateRoleForColumns(existing, []string{"displayName", "description", "isEnabled"})
	})
}

func (c *Client) EnsureProvider(ctx context.Context, spec ProviderSpec) error {
	provider, err := c.buildProvider(spec)
	if err != nil {
		return err
	}
	existing, err := c.getProvider(ctx, provider.Name)
	if err != nil {
		return err
	}
	if existing == nil {
		return c.call(ctx, "create provider "+provider.Name, func() (bool, error) {
			return c.providers.AddProvider(provider)
		})
	}
	return c.call(ctx, "update provider "+provider.Name, func() (bool, error) {
		return c.providers.UpdateProvider(provider)
	})
}

func (c *Client) buildOrganization(spec OrganizationSpec) (*casdoorsdk.Organization, error) {
	spec, err := normalizeOrganizationSpec(spec)
	if err != nil {
		return nil, err
	}
	return newBootstrapOrganization(spec), nil
}

func (c *Client) buildRole(spec RoleSpec) (*casdoorsdk.Role, error) {
	spec, err := normalizeRoleSpec(spec)
	if err != nil {
		return nil, err
	}
	return &casdoorsdk.Role{
		Owner:       c.credential.Organization,
		Name:        spec.Name,
		DisplayName: spec.DisplayName,
		Description: spec.Description,
		IsEnabled:   true,
	}, nil
}

func (c *Client) buildProvider(spec ProviderSpec) (*casdoorsdk.Provider, error) {
	spec, err := normalizeProviderSpec(spec)
	if err != nil {
		return nil, err
	}
	return &casdoorsdk.Provider{
		Owner:       c.credential.Organization,
		Name:        spec.Name,
		DisplayName: spec.DisplayName,
		Category:    spec.Category,
		Type:        spec.Type,
		SubType:     spec.SubType,
		Method:      spec.Method,
		ProviderUrl: spec.ProviderURL,
		Endpoint:    spec.Endpoint,
		Host:        spec.Host,
		Port:        spec.Port,
		DisableSsl:  spec.DisableSSL,
		Title:       spec.Title,
		Content:     spec.Content,
		Metadata:    spec.Metadata,
	}, nil
}

func normalizeOrganizationSpec(spec OrganizationSpec) (OrganizationSpec, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	spec.DisplayName = strings.TrimSpace(spec.DisplayName)
	spec.DefaultApplication = strings.TrimSpace(spec.DefaultApplication)
	if err := validateName("organization name", spec.Name); err != nil {
		return OrganizationSpec{}, err
	}
	if spec.DisplayName == "" {
		return OrganizationSpec{}, errors.New("casdoor: organization display name is required")
	}
	return spec, nil
}

func normalizeRoleSpec(spec RoleSpec) (RoleSpec, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	spec.DisplayName = strings.TrimSpace(spec.DisplayName)
	spec.Description = strings.TrimSpace(spec.Description)
	if err := validateName("role name", spec.Name); err != nil {
		return RoleSpec{}, err
	}
	if spec.DisplayName == "" {
		return RoleSpec{}, errors.New("casdoor: role display name is required")
	}
	return spec, nil
}

func normalizeProviderSpec(spec ProviderSpec) (ProviderSpec, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	spec.DisplayName = strings.TrimSpace(spec.DisplayName)
	spec.Category = strings.TrimSpace(spec.Category)
	spec.Type = strings.TrimSpace(spec.Type)
	spec.SubType = strings.TrimSpace(spec.SubType)
	spec.Method = strings.TrimSpace(spec.Method)
	spec.ProviderURL = strings.TrimSpace(spec.ProviderURL)
	spec.Endpoint = strings.TrimSpace(spec.Endpoint)
	spec.Host = strings.TrimSpace(spec.Host)
	spec.Title = strings.TrimSpace(spec.Title)
	spec.Content = strings.TrimSpace(spec.Content)
	spec.Metadata = strings.TrimSpace(spec.Metadata)
	if err := validateProviderRequired(spec); err != nil {
		return ProviderSpec{}, err
	}
	return spec, nil
}

func validateProviderRequired(spec ProviderSpec) error {
	if err := validateName("provider name", spec.Name); err != nil {
		return err
	}
	if spec.DisplayName == "" || spec.Category == "" || spec.Type == "" {
		return errors.New("casdoor: provider display name, category, and type are required")
	}
	if spec.Port < 0 {
		return fmt.Errorf("casdoor: provider port must be >= 0 (got %d)", spec.Port)
	}
	return nil
}

func (c *Client) getApplication(ctx context.Context, name string) (*casdoorsdk.Application, error) {
	var app *casdoorsdk.Application
	err := withSDKConfig(ctx, c.credential, "get application "+name, func() error {
		var getErr error
		app, getErr = c.apps.GetApplication(name)
		return getErr
	})
	return app, err
}

func (c *Client) getOrganization(ctx context.Context, name string) (*casdoorsdk.Organization, error) {
	var org *casdoorsdk.Organization
	err := withSDKConfig(ctx, c.credential, "get organization "+name, func() error {
		var getErr error
		org, getErr = c.orgs.GetOrganization(name)
		return getErr
	})
	return org, err
}

func (c *Client) getRoleForBootstrap(ctx context.Context, name string) (*casdoorsdk.Role, error) {
	var role *casdoorsdk.Role
	err := withSDKConfig(ctx, c.credential, "get role "+name, func() error {
		var getErr error
		role, getErr = c.roles.GetRole(name)
		return getErr
	})
	return role, err
}

func (c *Client) getProvider(ctx context.Context, name string) (*casdoorsdk.Provider, error) {
	var provider *casdoorsdk.Provider
	err := withSDKConfig(ctx, c.credential, "get provider "+name, func() error {
		var getErr error
		provider, getErr = c.providers.GetProvider(name)
		return getErr
	})
	return provider, err
}
