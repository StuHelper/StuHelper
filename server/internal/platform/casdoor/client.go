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

const PurposeAppProvisioning AdminPurpose = "casdoor-admin-app-provisioning"

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

type sdkApplicationAPI struct{}

var sdkConfigMu sync.Mutex

func NewAppProvisioningClient(credential Credential) (*Client, error) {
	return newClient(credential, sdkApplicationAPI{})
}

func newClient(credential Credential, apps applicationAPI) (*Client, error) {
	normalized, err := validateCredential(credential)
	if err != nil {
		return nil, err
	}
	if apps == nil {
		return nil, errors.New("casdoor: application API is required")
	}
	return &Client{credential: normalized, apps: apps}, nil
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

func validateCredential(credential Credential) (Credential, error) {
	credential.Endpoint = strings.TrimSpace(credential.Endpoint)
	credential.ClientID = strings.TrimSpace(credential.ClientID)
	credential.ClientSecret = strings.TrimSpace(credential.ClientSecret)
	credential.Organization = strings.TrimSpace(credential.Organization)
	credential.Application = strings.TrimSpace(credential.Application)
	if credential.Purpose != PurposeAppProvisioning {
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
	if err := ctx.Err(); err != nil {
		return err
	}
	sdkConfigMu.Lock()
	defer sdkConfigMu.Unlock()
	c.initSDKConfig()
	ok, err := fn()
	if err != nil {
		return fmt.Errorf("casdoor: %s failed: %w", operation, err)
	}
	if !ok {
		return fmt.Errorf("%w: %s", ErrOperationRejected, operation)
	}
	return nil
}

func (c *Client) initSDKConfig() {
	casdoorsdk.InitConfig(
		c.credential.Endpoint,
		c.credential.ClientID,
		c.credential.ClientSecret,
		c.credential.Certificate,
		c.credential.Organization,
		c.credential.Application,
	)
}
