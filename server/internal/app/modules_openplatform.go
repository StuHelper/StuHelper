package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	platformcasdoor "git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

func (rt *Runtime) initOpenPlatformModule(
	api *gin.RouterGroup,
	authMW gin.HandlerFunc,
	piiCipher *pii.Cipher,
	userIDResolver middleware.InternalUserIDResolver,
) (*openplatform.Handler, *openplatform.Service, error) {
	provisioner, err := rt.newCasdoorAppProvisioningClient()
	if err != nil {
		return nil, nil, err
	}
	tokenProber, err := rt.newOpenPlatformRuntimeTokenProber()
	if err != nil {
		return nil, nil, err
	}
	service, err := openplatform.NewService(
		openplatform.NewRepository(rt.database),
		rt.redisClient.GetClient(),
		openplatform.WithAppProvisioner(provisioner),
		openplatform.WithOIDCAuthURLBuilder(rt.oidcClient),
		openplatform.WithPhoneDecryptor(piiCipher),
		openplatform.WithResourceFGAClient(rt.fgaClient),
		openplatform.WithConsentBaseURL(rt.cfg.App.CORSOrigins[0]),
		openplatform.WithAccountBaseURL(rt.cfg.App.CORSOrigins[0]),
		openplatform.WithDisclosureRateLimits(openPlatformDisclosureRateLimitConfig(
			rt.cfg.OpenPlatform.DisclosureRateLimit,
		)),
		openplatform.WithRuntimeTokenProbe(
			tokenProber,
			rt.cfg.OpenPlatform.TokenProbe.RuntimeRequired,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize open platform service: %w", err)
	}
	handler := openplatform.NewHandler(
		service,
		openplatform.WithInternalUserIDResolver(userIDResolver),
		openplatform.WithAdminAuthorizers(openplatform.AdminAuthorizers{
			Manage: rbac.RequireGlobalCapability(capability.OpenPlatformManage),
		}),
	)
	handler.RegisterRoutes(api, authMW)
	return handler, service, nil
}

func (rt *Runtime) newOpenPlatformRuntimeTokenProber() (platformcasdoor.RuntimeTokenMinimizationProber, error) {
	cfg := rt.cfg.OpenPlatform.TokenProbe
	if cfg.RuntimeCommand == "" {
		return nil, nil
	}
	return platformcasdoor.NewCommandRuntimeTokenProber(platformcasdoor.RuntimeTokenProbeCommandConfig{
		Command: cfg.RuntimeCommand,
		Issuer:  rt.cfg.Casdoor.Issuer,
		Timeout: secondsDuration(cfg.RuntimeTimeoutSeconds),
	})
}

func openPlatformDisclosureRateLimitConfig(
	cfg config.OpenPlatformDisclosureRateLimitConfig,
) openplatform.DisclosureRateLimitConfig {
	return openplatform.DisclosureRateLimitConfig{
		AppLimit:            cfg.AppLimit,
		AppUserLimit:        cfg.AppUserLimit,
		EndpointLimit:       cfg.EndpointLimit,
		ConsentLimit:        cfg.ConsentLimit,
		ReplayLimit:         cfg.ReplayLimit,
		Window:              secondsDuration(cfg.WindowSeconds),
		ReplayWindow:        secondsDuration(cfg.ReplayWindowSeconds),
		ReplayAuditCooldown: secondsDuration(cfg.ReplayAuditCooldownSeconds),
	}
}

func secondsDuration(seconds int) time.Duration {
	if seconds == 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
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

func (g *casdoorUserProfileGateway) Send(ctx context.Context, phone, content string) error {
	return g.client.Send(ctx, phone, content)
}
