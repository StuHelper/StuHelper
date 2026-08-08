package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/modules/openplatform"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	platformcasdoor "github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

func (rt *Runtime) initOpenPlatformModule(
	api *gin.RouterGroup,
	authMW gin.HandlerFunc,
	userProfileGateway *casdoorUserProfileGateway,
	userIDResolver middleware.InternalUserIDResolver,
) (*openplatform.Handler, *openplatform.Service, error) {
	provisionerClient, err := rt.newCasdoorAppProvisioningClient()
	if err != nil {
		return nil, nil, err
	}
	tokenProber, err := rt.newOpenPlatformRuntimeTokenProber()
	if err != nil {
		return nil, nil, err
	}
	consentBaseURL := rt.openPlatformConsentBaseURL()
	accountBaseURL := rt.openPlatformAccountBaseURL(consentBaseURL)
	service, err := openplatform.NewService(
		openplatform.NewRepository(rt.database),
		rt.redisClient.GetClient(),
		openplatform.WithAppProvisioner(newCasdoorOpenPlatformProvisioner(provisionerClient)),
		openplatform.WithOIDCAuthURLBuilder(rt.oidcClient),
		openplatform.WithAuthoritativePhoneReader(userProfileGateway),
		openplatform.WithResourceFGAClient(rt.fgaClient),
		openplatform.WithConsentBaseURL(consentBaseURL),
		openplatform.WithAccountBaseURL(accountBaseURL),
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
		openplatform.WithAdminAuthorizers(openPlatformAdminAuthorizers()),
	)
	handler.SetResourceAccessTokenVerifier(newOpenPlatformResourceAccessTokenVerifier(rt.oidcClient))
	handler.RegisterRoutes(api, authMW)
	return handler, service, nil
}

func (rt *Runtime) openPlatformConsentBaseURL() string {
	if rt.cfg.OpenPlatform.ConsentBaseURL != "" {
		return rt.cfg.OpenPlatform.ConsentBaseURL
	}
	if config.IsProductionLikeEnv(rt.cfg.App.Env) {
		return ""
	}
	if len(rt.cfg.App.CORSOrigins) == 0 {
		return ""
	}
	return rt.cfg.App.CORSOrigins[0]
}

func (rt *Runtime) openPlatformAccountBaseURL(consentBaseURL string) string {
	if rt.cfg.OpenPlatform.AccountBaseURL != "" {
		return rt.cfg.OpenPlatform.AccountBaseURL
	}
	return consentBaseURL
}

func (rt *Runtime) newOpenPlatformRuntimeTokenProber() (*casdoorOpenPlatformRuntimeTokenProber, error) {
	cfg := rt.cfg.OpenPlatform.TokenProbe
	command := strings.TrimSpace(cfg.RuntimeCommand)
	if command == "" {
		return nil, nil
	}
	prober, err := platformcasdoor.NewCommandRuntimeTokenProber(platformcasdoor.RuntimeTokenProbeCommandConfig{
		Command: command,
		Issuer:  rt.cfg.Casdoor.Issuer,
		Timeout: secondsDuration(cfg.RuntimeTimeoutSeconds),
	})
	if err != nil {
		return nil, err
	}
	return &casdoorOpenPlatformRuntimeTokenProber{prober: prober}, nil
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

type casdoorOpenPlatformProvisioner struct {
	client *platformcasdoor.Client
}

func newCasdoorOpenPlatformProvisioner(client *platformcasdoor.Client) *casdoorOpenPlatformProvisioner {
	if client == nil {
		return nil
	}
	return &casdoorOpenPlatformProvisioner{client: client}
}

func (p *casdoorOpenPlatformProvisioner) GetApplication(
	ctx context.Context,
	name string,
) (openplatform.ProvisionedApplicationSpec, error) {
	if p == nil || p.client == nil {
		return openplatform.ProvisionedApplicationSpec{}, fmt.Errorf("open platform Casdoor application reader is not configured")
	}
	spec, err := p.client.GetApplication(ctx, name)
	if err != nil {
		return openplatform.ProvisionedApplicationSpec{}, err
	}
	return openPlatformApplicationSpecFromCasdoor(spec), nil
}

func (p *casdoorOpenPlatformProvisioner) EnsureApplication(
	ctx context.Context,
	spec openplatform.ProvisionedApplicationSpec,
) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("open platform Casdoor application provisioner is not configured")
	}
	return p.client.EnsureApplication(ctx, casdoorApplicationSpecFromOpenPlatform(spec))
}

func (p *casdoorOpenPlatformProvisioner) DeleteApplication(ctx context.Context, name string) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("open platform Casdoor application provisioner is not configured")
	}
	return p.client.DeleteApplication(ctx, name)
}

type casdoorOpenPlatformRuntimeTokenProber struct {
	prober platformcasdoor.RuntimeTokenMinimizationProber
}

func (p *casdoorOpenPlatformRuntimeTokenProber) ProbeTokenMinimization(
	ctx context.Context,
	spec openplatform.ProvisionedApplicationSpec,
) (openplatform.RuntimeTokenMinimizationProbeResult, error) {
	if p == nil || p.prober == nil {
		return openplatform.RuntimeTokenMinimizationProbeResult{}, fmt.Errorf("open platform runtime token prober is not configured")
	}
	return p.prober.ProbeTokenMinimization(ctx, casdoorApplicationSpecFromOpenPlatform(spec))
}

func openPlatformApplicationSpecFromCasdoor(
	spec platformcasdoor.ApplicationSpec,
) openplatform.ProvisionedApplicationSpec {
	return openplatform.ProvisionedApplicationSpec{
		Organization:         spec.Organization,
		Name:                 spec.Name,
		DisplayName:          spec.DisplayName,
		Logo:                 spec.Logo,
		HomepageURL:          spec.HomepageURL,
		Description:          spec.Description,
		ClientID:             spec.ClientID,
		ClientSecret:         spec.ClientSecret,
		RedirectURIs:         append([]string(nil), spec.RedirectURIs...),
		GrantTypes:           append([]string(nil), spec.GrantTypes...),
		TokenFormat:          spec.TokenFormat,
		TokenFields:          cloneExplicitStringSlice(spec.TokenFields),
		ExpireInHours:        spec.ExpireInHours,
		RefreshExpireInHours: spec.RefreshExpireInHours,
	}
}

func casdoorApplicationSpecFromOpenPlatform(
	spec openplatform.ProvisionedApplicationSpec,
) platformcasdoor.ApplicationSpec {
	return platformcasdoor.ApplicationSpec{
		Organization:         spec.Organization,
		Name:                 spec.Name,
		DisplayName:          spec.DisplayName,
		Logo:                 spec.Logo,
		HomepageURL:          spec.HomepageURL,
		Description:          spec.Description,
		ClientID:             spec.ClientID,
		ClientSecret:         spec.ClientSecret,
		RedirectURIs:         append([]string(nil), spec.RedirectURIs...),
		GrantTypes:           append([]string(nil), spec.GrantTypes...),
		TokenFormat:          spec.TokenFormat,
		TokenFields:          cloneExplicitStringSlice(spec.TokenFields),
		ExpireInHours:        spec.ExpireInHours,
		RefreshExpireInHours: spec.RefreshExpireInHours,
	}
}

func cloneExplicitStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

type casdoorUserProfileClient interface {
	GetPhone(context.Context, string) (string, error)
	UpdatePhone(context.Context, platformcasdoor.UserPhoneUpdate) error
	ClearPhone(context.Context, string) error
	Send(context.Context, string, string) error
}

type casdoorUserProfileGateway struct {
	client                 casdoorUserProfileClient
	casdoorSubjectByUserID func(context.Context, int64) (string, error)
}

func newCasdoorUserProfileGateway(
	client casdoorUserProfileClient,
	casdoorSubjectByUserID func(context.Context, int64) (string, error),
) *casdoorUserProfileGateway {
	return &casdoorUserProfileGateway{
		client:                 client,
		casdoorSubjectByUserID: casdoorSubjectByUserID,
	}
}

func (g *casdoorUserProfileGateway) UpdatePhone(ctx context.Context, subject, phone string) error {
	return g.client.UpdatePhone(ctx, platformcasdoor.UserPhoneUpdate{
		Subject: subject,
		Phone:   phone,
	})
}

func (g *casdoorUserProfileGateway) GetPhone(ctx context.Context, userID int64) (string, error) {
	subject, err := g.resolvePhoneSubject(ctx, userID)
	if err != nil {
		return "", err
	}
	return g.client.GetPhone(ctx, subject)
}

func (g *casdoorUserProfileGateway) SetPhone(ctx context.Context, userID int64, phone string) error {
	subject, err := g.resolvePhoneSubject(ctx, userID)
	if err != nil {
		return err
	}
	return g.client.UpdatePhone(ctx, platformcasdoor.UserPhoneUpdate{Subject: subject, Phone: phone})
}

func (g *casdoorUserProfileGateway) ClearPhone(ctx context.Context, userID int64) error {
	subject, err := g.resolvePhoneSubject(ctx, userID)
	if err != nil {
		return err
	}
	return g.client.ClearPhone(ctx, subject)
}

func (g *casdoorUserProfileGateway) resolvePhoneSubject(ctx context.Context, userID int64) (string, error) {
	if g == nil || g.client == nil {
		return "", fmt.Errorf("casdoor user profile reader is not configured")
	}
	if userID <= 0 {
		return "", fmt.Errorf("internal user ID is required for Casdoor phone lookup")
	}
	if g.casdoorSubjectByUserID == nil {
		return "", fmt.Errorf("casdoor user subject resolver is not configured")
	}
	subject, err := g.casdoorSubjectByUserID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("resolve Casdoor user for phone lookup: %w", err)
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", fmt.Errorf("resolved Casdoor user subject is empty")
	}
	return subject, nil
}

func (g *casdoorUserProfileGateway) Send(ctx context.Context, phone, content string) error {
	return g.client.Send(ctx, phone, content)
}
