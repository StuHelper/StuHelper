// Package casdoor is the only backend package allowed to import Casdoor SDK.
package casdoor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

type AdminPurpose string

const (
	PurposeAppProvisioning AdminPurpose = "casdoor-admin-app-provisioning"
	PurposeRoleSync        AdminPurpose = "casdoor-admin-role-sync"
	PurposeUserLookup      AdminPurpose = "casdoor-admin-user-lookup"
)

var ErrOperationRejected = errors.New("casdoor operation rejected")

type Credential struct {
	Purpose      AdminPurpose
	Endpoint     string
	ClientID     string
	ClientSecret string
	Certificate  string
	Organization string
	Application  string
}

type Client struct {
	credential Credential
	apps       applicationAPI
}

type applicationAPI interface {
	AddApplication(*casdoorsdk.Application) (bool, error)
	UpdateApplication(*casdoorsdk.Application) (bool, error)
	DeleteApplication(*casdoorsdk.Application) (bool, error)
}

type roleAPI interface {
	GetRole(string) (*casdoorsdk.Role, error)
	UpdateRoleForColumns(*casdoorsdk.Role, []string) (bool, error)
}

type userAPI interface {
	GetUserByUserId(string) (*casdoorsdk.User, error)
}

type sdkApplicationAPI struct{}
type sdkRoleAPI struct{}
type sdkUserAPI struct{}

var sdkConfigMu sync.Mutex

func NewAppProvisioningClient(credential Credential) (*Client, error) {
	return newClient(credential, sdkApplicationAPI{})
}

func newClient(credential Credential, apps applicationAPI) (*Client, error) {
	normalized, err := validateCredentialForPurpose(credential, PurposeAppProvisioning)
	if err != nil {
		return nil, err
	}
	if apps == nil {
		return nil, errors.New("casdoor: application API is required")
	}
	return &Client{credential: normalized, apps: apps}, nil
}

func validateCredentialForPurpose(credential Credential, purpose AdminPurpose) (Credential, error) {
	if credential.Purpose != purpose {
		return Credential{}, fmt.Errorf("casdoor: credential purpose %q does not match %q", credential.Purpose, purpose)
	}
	return validateCredential(credential)
}

func (sdkApplicationAPI) AddApplication(app *casdoorsdk.Application) (bool, error) {
	return casdoorsdk.AddApplication(app)
}

func (sdkApplicationAPI) UpdateApplication(app *casdoorsdk.Application) (bool, error) {
	return casdoorsdk.UpdateApplication(app)
}

func (sdkApplicationAPI) DeleteApplication(app *casdoorsdk.Application) (bool, error) {
	return casdoorsdk.DeleteApplication(app)
}

func (sdkRoleAPI) GetRole(name string) (*casdoorsdk.Role, error) {
	return casdoorsdk.GetRole(name)
}

func (sdkRoleAPI) UpdateRoleForColumns(role *casdoorsdk.Role, columns []string) (bool, error) {
	return casdoorsdk.UpdateRoleForColumns(role, columns)
}

func (sdkUserAPI) GetUserByUserId(subject string) (*casdoorsdk.User, error) {
	return casdoorsdk.GetUserByUserId(subject)
}

func validateCredential(credential Credential) (Credential, error) {
	credential.Endpoint = strings.TrimSpace(credential.Endpoint)
	credential.ClientID = strings.TrimSpace(credential.ClientID)
	credential.ClientSecret = strings.TrimSpace(credential.ClientSecret)
	credential.Organization = strings.TrimSpace(credential.Organization)
	credential.Application = strings.TrimSpace(credential.Application)
	switch credential.Purpose {
	case PurposeAppProvisioning, PurposeRoleSync, PurposeUserLookup:
	default:
		return Credential{}, fmt.Errorf("casdoor: unsupported admin purpose %q", credential.Purpose)
	}
	if credential.Endpoint == "" {
		return Credential{}, errors.New("casdoor: endpoint is required")
	}
	if credential.ClientID == "" || credential.ClientSecret == "" {
		return Credential{}, errors.New("casdoor: admin client credential is required")
	}
	if credential.Organization == "" || credential.Application == "" {
		return Credential{}, errors.New("casdoor: organization and application are required")
	}
	return credential, nil
}

func (c *Client) call(ctx context.Context, operation string, fn func() (bool, error)) error {
	return callWithCredential(ctx, c.credential, operation, fn)
}

func callWithCredential(ctx context.Context, credential Credential, operation string, fn func() (bool, error)) error {
	return withSDKConfig(ctx, credential, operation, func() error {
		ok, err := fn()
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %s", ErrOperationRejected, operation)
		}
		return nil
	})
}

func withSDKConfig(ctx context.Context, credential Credential, operation string, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sdkConfigMu.Lock()
	defer sdkConfigMu.Unlock()
	initSDKConfig(credential)
	if err := fn(); err != nil {
		return fmt.Errorf("casdoor: %s failed: %w", operation, err)
	}
	return nil
}

func initSDKConfig(credential Credential) {
	casdoorsdk.InitConfig(
		credential.Endpoint,
		credential.ClientID,
		credential.ClientSecret,
		credential.Certificate,
		credential.Organization,
		credential.Application,
	)
}
