package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	platformcasdoor "git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

func (rt *Runtime) initOpenPlatformModule(
	api *gin.RouterGroup,
	authMW gin.HandlerFunc,
	piiCipher *pii.Cipher,
	userProfileGateway *casdoorUserProfileGateway,
) (*openplatform.Handler, error) {
	provisioner, err := rt.newCasdoorAppProvisioningClient()
	if err != nil {
		return nil, err
	}
	service, err := openplatform.NewService(
		openplatform.NewRepository(rt.database),
		rt.redisClient.GetClient(),
		openplatform.WithAppProvisioner(provisioner),
		openplatform.WithOIDCAuthURLBuilder(rt.oidcClient),
		openplatform.WithPhoneDecryptor(piiCipher),
		openplatform.WithCasdoorPhoneReader(userProfileGateway),
		openplatform.WithConsentBaseURL(rt.cfg.App.CORSOrigins[0]),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize open platform service: %w", err)
	}
	handler := openplatform.NewHandler(service)
	handler.RegisterRoutes(api, authMW)
	return handler, nil
}

func (rt *Runtime) newCasdoorAppProvisioningClient() (*platformcasdoor.Client, error) {
	cfg := rt.cfg.Casdoor
	client, err := platformcasdoor.NewAppProvisioningClient(casdoorAppProvisioningCredential(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Casdoor app provisioning client: %w", err)
	}
	return client, nil
}

func (rt *Runtime) newCasdoorUserProfileClient() (*platformcasdoor.UserProfileClient, error) {
	cfg := rt.cfg.Casdoor
	client, err := platformcasdoor.NewUserProfileClient(casdoorUserProfileCredential(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Casdoor user profile client: %w", err)
	}
	return client, nil
}

func casdoorAppProvisioningCredential(cfg config.CasdoorConfig) platformcasdoor.Credential {
	return platformcasdoor.Credential{
		Purpose:      platformcasdoor.PurposeAppProvisioning,
		Endpoint:     cfg.Issuer,
		ClientID:     cfg.AppProvisioningClientID,
		ClientSecret: cfg.AppProvisioningClientSecret,
		Certificate:  cfg.AppProvisioningCertificate,
		Organization: cfg.Organization,
		Application:  cfg.AppProvisioningApplication,
	}
}

func casdoorUserProfileCredential(cfg config.CasdoorConfig) platformcasdoor.Credential {
	return platformcasdoor.Credential{
		Purpose:      platformcasdoor.PurposeUserProfile,
		Endpoint:     cfg.Issuer,
		ClientID:     cfg.UserProfileClientID,
		ClientSecret: cfg.UserProfileClientSecret,
		Certificate:  cfg.UserProfileCertificate,
		Organization: cfg.Organization,
		Application:  cfg.UserProfileApplication,
	}
}

type casdoorUserProfileGateway struct {
	client *platformcasdoor.UserProfileClient
}

func newCasdoorUserProfileGateway(client *platformcasdoor.UserProfileClient) *casdoorUserProfileGateway {
	return &casdoorUserProfileGateway{client: client}
}

func (g *casdoorUserProfileGateway) UpdatePhone(ctx context.Context, subject, phone string) error {
	return g.client.UpdatePhone(ctx, platformcasdoor.UserPhoneUpdate{
		Subject: subject,
		Phone:   phone,
	})
}

func (g *casdoorUserProfileGateway) GetPhone(ctx context.Context, subject string) (string, error) {
	return g.client.GetPhone(ctx, subject)
}

func (g *casdoorUserProfileGateway) Send(ctx context.Context, phone, content string) error {
	return g.client.Send(ctx, phone, content)
}
