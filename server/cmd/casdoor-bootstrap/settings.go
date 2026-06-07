package main

import (
	"fmt"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

const (
	defaultAccessTokenHours  = 1
	defaultRefreshTokenHours = 24
	defaultTokenFormat       = "JWT"
	minimalTokenFormat       = "JWT-Custom"
	defaultApplicationLogo   = "https://stuhelper.com/android-chrome-512x512.png"
)

type envReader func(string) string

type bootstrapSettings struct {
	credential casdoor.Credential
	plan       casdoor.BootstrapPlan
}

type applicationBootstrapSettings struct {
	credential   casdoor.Credential
	applications []casdoor.ApplicationSpec
}

func loadSettings(getenv envReader) (bootstrapSettings, error) {
	targetOrganization, err := requiredValue(getenv, "CASDOOR_ORGANIZATION")
	if err != nil {
		return bootstrapSettings{}, err
	}
	credential, err := buildCredential(getenv, targetOrganization)
	if err != nil {
		return bootstrapSettings{}, err
	}
	plan, err := buildPlan(getenv, targetOrganization)
	if err != nil {
		return bootstrapSettings{}, err
	}
	return bootstrapSettings{credential: credential, plan: plan}, nil
}

func loadApplicationBootstrapSettings(getenv envReader) (applicationBootstrapSettings, error) {
	targetOrganization, err := requiredValue(getenv, "CASDOOR_ORGANIZATION")
	if err != nil {
		return applicationBootstrapSettings{}, err
	}
	credential, err := buildApplicationBootstrapCredential(getenv, targetOrganization)
	if err != nil {
		return applicationBootstrapSettings{}, err
	}
	apps, err := buildApplications(getenv)
	if err != nil {
		return applicationBootstrapSettings{}, err
	}
	return applicationBootstrapSettings{credential: credential, applications: apps}, nil
}

func buildCredential(getenv envReader, targetOrganization string) (casdoor.Credential, error) {
	endpoint, err := requiredFirst(getenv, "CASDOOR_BOOTSTRAP_ENDPOINT", "CASDOOR_ISSUER")
	if err != nil {
		return casdoor.Credential{}, err
	}
	clientID, err := requiredValue(getenv, "CASDOOR_BOOTSTRAP_CLIENT_ID")
	if err != nil {
		return casdoor.Credential{}, err
	}
	clientSecret, err := requiredValue(getenv, "CASDOOR_BOOTSTRAP_CLIENT_SECRET")
	if err != nil {
		return casdoor.Credential{}, err
	}
	application, err := requiredValue(getenv, "CASDOOR_BOOTSTRAP_APPLICATION")
	if err != nil {
		return casdoor.Credential{}, err
	}
	credentialOrganization := valueOrDefault(getenv("CASDOOR_BOOTSTRAP_ORGANIZATION"), targetOrganization)
	return casdoor.Credential{
		Purpose:      casdoor.PurposeBootstrap,
		Endpoint:     endpoint,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Certificate:  strings.TrimSpace(getenv("CASDOOR_BOOTSTRAP_CERTIFICATE")),
		Organization: credentialOrganization,
		Application:  application,
	}, nil
}

func buildApplicationBootstrapCredential(getenv envReader, targetOrganization string) (casdoor.Credential, error) {
	endpoint, err := requiredFirst(getenv, "CASDOOR_BOOTSTRAP_ENDPOINT", "CASDOOR_ISSUER")
	if err != nil {
		return casdoor.Credential{}, err
	}
	clientID, err := requiredFirst(getenv, "CASDOOR_APP_PROVISIONING_CLIENT_ID", "CASDOOR_BOOTSTRAP_CLIENT_ID")
	if err != nil {
		return casdoor.Credential{}, err
	}
	clientSecret, err := requiredFirst(getenv, "CASDOOR_APP_PROVISIONING_CLIENT_SECRET", "CASDOOR_BOOTSTRAP_CLIENT_SECRET")
	if err != nil {
		return casdoor.Credential{}, err
	}
	application, err := requiredFirst(getenv, "CASDOOR_APP_PROVISIONING_APPLICATION", "CASDOOR_BOOTSTRAP_APPLICATION")
	if err != nil {
		return casdoor.Credential{}, err
	}
	return casdoor.Credential{
		Purpose:      casdoor.PurposeAppProvisioning,
		Endpoint:     endpoint,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Certificate:  strings.TrimSpace(getenv("CASDOOR_APP_PROVISIONING_CERTIFICATE")),
		Organization: targetOrganization,
		Application:  application,
	}, nil
}

func buildPlan(getenv envReader, orgName string) (casdoor.BootstrapPlan, error) {
	apps, err := buildApplications(getenv)
	if err != nil {
		return casdoor.BootstrapPlan{}, err
	}
	providers, err := buildProviders(getenv)
	if err != nil {
		return casdoor.BootstrapPlan{}, err
	}
	return casdoor.BootstrapPlan{
		Organization: casdoor.OrganizationSpec{
			Name:               orgName,
			DisplayName:        valueOrDefault(getenv("CASDOOR_ORGANIZATION_DISPLAY_NAME"), "StuHelper"),
			DefaultApplication: "stuhelper-web",
		},
		Applications: apps,
		Roles:        flatRoleCatalog(),
		Providers:    providers,
	}, nil
}

func buildApplications(getenv envReader) ([]casdoor.ApplicationSpec, error) {
	web, err := appSpec(getenv, webAppEnv())
	if err != nil {
		return nil, err
	}
	admin, err := appSpec(getenv, prefixedAppEnv("CASDOOR_ADMIN", "stuhelper-admin", "StuHelper Admin"))
	if err != nil {
		return nil, err
	}
	uniapp, err := appSpec(getenv, prefixedAppEnv("CASDOOR_UNIAPP", "stuhelper-uniapp", "StuHelper UniApp"))
	if err != nil {
		return nil, err
	}
	adminServices, err := buildAdminServiceApplications(getenv)
	if err != nil {
		return nil, err
	}
	tokenProbeSmoke, err := tokenProbeSmokeAppSpec(getenv)
	if err != nil {
		return nil, err
	}
	apps := []casdoor.ApplicationSpec{web, admin, uniapp}
	apps = append(apps, adminServices...)
	return append(apps, tokenProbeSmoke), nil
}

func buildAdminServiceApplications(getenv envReader) ([]casdoor.ApplicationSpec, error) {
	defs := []serviceAppEnv{
		{prefix: "CASDOOR_APP_PROVISIONING", displayName: "StuHelper App Provisioning"},
		{prefix: "CASDOOR_INTROSPECTION", displayName: "StuHelper Token Introspection"},
		{prefix: "CASDOOR_USER_PROFILE", displayName: "StuHelper User Profile"},
		{prefix: "CASDOOR_ROLE_SYNC", displayName: "StuHelper Role Sync"},
		{prefix: "CASDOOR_USER_LOOKUP", displayName: "StuHelper User Lookup"},
	}
	apps := make([]casdoor.ApplicationSpec, 0, len(defs))
	for _, def := range defs {
		app, err := serviceAppSpec(getenv, def)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func buildProviders(getenv envReader) ([]casdoor.ProviderSpec, error) {
	var providers []casdoor.ProviderSpec
	sms, ok, err := providerSpec(getenv, "CASDOOR_SMS_PROVIDER")
	if err != nil {
		return nil, err
	}
	if ok {
		providers = append(providers, sms)
	}
	email, ok, err := providerSpec(getenv, "CASDOOR_EMAIL_PROVIDER")
	if err != nil {
		return nil, err
	}
	if ok {
		providers = append(providers, email)
	}
	return providers, nil
}

func flatRoleCatalog() []casdoor.RoleSpec {
	return []casdoor.RoleSpec{
		{Name: "super_admin", DisplayName: "Super Admin", Description: "Global StuHelper administrator"},
		{Name: "school_admin", DisplayName: "School Admin", Description: "School-scoped administrator projection"},
		{Name: "section_admin", DisplayName: "Section Admin", Description: "Section administrator projection"},
		{Name: "section_moderator", DisplayName: "Section Moderator", Description: "Section moderation projection"},
		{Name: "section_reviewer", DisplayName: "Section Reviewer", Description: "Section review projection"},
		{Name: "verified_student", DisplayName: "Verified Student", Description: "Verified student projection"},
		{Name: "freshman_provisional", DisplayName: "Freshman Provisional", Description: "Freshman admission provisional projection"},
		{Name: "user", DisplayName: "User", Description: "Default authenticated user projection"},
	}
}

type appEnv struct {
	name                  string
	displayName           string
	logoKey               string
	clientIDKey           string
	secretKey             string
	redirectKey           string
	additionalRedirectKey string
}

type serviceAppEnv struct {
	prefix      string
	displayName string
}

func webAppEnv() appEnv {
	return appEnv{
		name:                  "stuhelper-web",
		displayName:           "StuHelper Web",
		logoKey:               "CASDOOR_LOGO",
		clientIDKey:           "CASDOOR_CLIENT_ID",
		secretKey:             "CASDOOR_CLIENT_SECRET", // #nosec G101 -- env key name, not a secret value.
		redirectKey:           "CASDOOR_REDIRECT_URI",
		additionalRedirectKey: "CASDOOR_ADDITIONAL_REDIRECT_URIS",
	}
}

func prefixedAppEnv(prefix, name, displayName string) appEnv {
	return appEnv{
		name:                  name,
		displayName:           displayName,
		logoKey:               prefix + "_LOGO",
		clientIDKey:           prefix + "_CLIENT_ID",
		secretKey:             prefix + "_CLIENT_SECRET",
		redirectKey:           prefix + "_REDIRECT_URI",
		additionalRedirectKey: prefix + "_ADDITIONAL_REDIRECT_URIS",
	}
}

func appSpec(getenv envReader, env appEnv) (casdoor.ApplicationSpec, error) {
	redirects, err := requiredList(getenv, env.redirectKey)
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	redirects = append(redirects, optionalList(getenv, env.additionalRedirectKey)...)
	clientID, err := requiredValue(getenv, env.clientIDKey)
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	secret, err := requiredValue(getenv, env.secretKey)
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	return casdoor.ApplicationSpec{
		Name:                 env.name,
		DisplayName:          env.displayName,
		Logo:                 applicationLogo(getenv, env.logoKey),
		ClientID:             clientID,
		ClientSecret:         secret,
		RedirectURIs:         redirects,
		GrantTypes:           []string{"authorization_code", "refresh_token"},
		TokenFormat:          defaultTokenFormat,
		TokenFields:          []string{},
		ExpireInHours:        defaultAccessTokenHours,
		RefreshExpireInHours: defaultRefreshTokenHours,
	}, nil
}

func serviceAppSpec(getenv envReader, env serviceAppEnv) (casdoor.ApplicationSpec, error) {
	name, err := requiredValue(getenv, env.prefix+"_APPLICATION")
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	clientID, err := requiredValue(getenv, env.prefix+"_CLIENT_ID")
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	secret, err := requiredValue(getenv, env.prefix+"_CLIENT_SECRET")
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	return casdoor.ApplicationSpec{
		Name:                 name,
		DisplayName:          env.displayName,
		Logo:                 applicationLogo(getenv, env.prefix+"_LOGO"),
		ClientID:             clientID,
		ClientSecret:         secret,
		GrantTypes:           []string{"client_credentials"},
		TokenFormat:          defaultTokenFormat,
		TokenFields:          []string{},
		ExpireInHours:        defaultAccessTokenHours,
		RefreshExpireInHours: 0,
	}, nil
}

func tokenProbeSmokeAppSpec(getenv envReader) (casdoor.ApplicationSpec, error) {
	name, err := requiredValue(getenv, "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION")
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	spec, err := appSpec(getenv, appEnv{
		name:        name,
		displayName: "StuHelper Token Probe Smoke",
		logoKey:     "CASDOOR_TOKEN_PROBE_SMOKE_LOGO",
		clientIDKey: "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID",
		secretKey:   "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET", // #nosec G101 -- env key name, not a secret value.
		redirectKey: "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI",
	})
	if err != nil {
		return casdoor.ApplicationSpec{}, err
	}
	spec.TokenFormat = minimalTokenFormat
	return spec, nil
}

func applicationLogo(getenv envReader, key string) string {
	if logo := strings.TrimSpace(getenv(key)); logo != "" {
		return logo
	}

	baseURL := strings.TrimRight(strings.TrimSpace(getenv("WEB_PUBLIC_URL")), "/")
	if baseURL != "" {
		return baseURL + "/android-chrome-512x512.png"
	}
	return defaultApplicationLogo
}

func requiredValue(getenv envReader, key string) (string, error) {
	value := required(getenv, key)
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func required(getenv envReader, key string) string {
	return strings.TrimSpace(getenv(key))
}

func requiredFirst(getenv envReader, keys ...string) (string, error) {
	for _, key := range keys {
		if value := required(getenv, key); value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("one of %s is required", strings.Join(keys, ", "))
}

func requiredList(getenv envReader, key string) ([]string, error) {
	value := required(getenv, key)
	if value == "" {
		return nil, fmt.Errorf("%s is required", key)
	}
	return splitList(value), nil
}

func optionalList(getenv envReader, key string) []string {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	value := required(getenv, key)
	if value == "" {
		return nil
	}
	return splitList(value)
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func valueOrDefault(raw, fallback string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	return value
}
