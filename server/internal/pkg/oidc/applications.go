package oidc

import (
	"fmt"
	"net/url"
	"strings"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func oauth2ConfigsFromCasdoor(endpoint oauth2.Endpoint, cfg config.CasdoorConfig) (map[string]oauth2.Config, error) {
	apps := map[string]oauth2.Config{}
	for _, input := range oauth2ApplicationInputs(cfg) {
		oauth2Cfg, ok, err := oauth2ConfigFromInput(endpoint, input)
		if err != nil {
			return nil, err
		}
		if ok {
			apps[input.appKey] = oauth2Cfg
		}
	}
	return apps, nil
}

type oauth2ApplicationInput struct {
	appKey       string
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	required     bool
}

func oauth2ApplicationInputs(cfg config.CasdoorConfig) []oauth2ApplicationInput {
	return []oauth2ApplicationInput{
		{
			appKey:       ApplicationWeb,
			clientID:     cfg.ClientID,
			clientSecret: cfg.ClientSecret,
			redirectURI:  cfg.RedirectURI,
			scopes:       cfg.WebScopes,
			required:     true,
		},
		{
			appKey:       ApplicationAdmin,
			clientID:     cfg.AdminClientID,
			clientSecret: cfg.AdminClientSecret,
			redirectURI:  cfg.AdminRedirectURI,
			scopes:       cfg.AdminScopes,
		},
		{
			appKey:       ApplicationUniapp,
			clientID:     cfg.UniappClientID,
			clientSecret: cfg.UniappClientSecret,
			redirectURI:  cfg.UniappRedirectURI,
			scopes:       cfg.UniappScopes,
		},
	}
}

func oauth2ConfigFromInput(endpoint oauth2.Endpoint, input oauth2ApplicationInput) (oauth2.Config, bool, error) {
	if input.clientID == "" && input.clientSecret == "" && input.redirectURI == "" && !input.required {
		return oauth2.Config{}, false, nil
	}
	if input.clientID == "" || input.clientSecret == "" || input.redirectURI == "" {
		return oauth2.Config{}, false, fmt.Errorf("%w: %s requires client id, secret and redirect uri", ErrApplicationNotConfigured, input.appKey)
	}
	return oauth2.Config{
		ClientID:     input.clientID,
		ClientSecret: input.clientSecret,
		RedirectURL:  input.redirectURI,
		Endpoint:     endpoint,
		Scopes:       oauth2Scopes(input.scopes),
	}, true, nil
}

func oauth2Scopes(configured []string) []string {
	scopes := normalizeOAuth2Scopes(configured)
	if len(scopes) > 0 {
		return scopes
	}
	return []string{gooidc.ScopeOpenID, "profile", "email", "offline_access"}
}

func normalizeOAuth2Scopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (c *Client) oauth2ConfigForApplication(appKey string) (oauth2.Config, error) {
	appKey = strings.TrimSpace(appKey)
	if appKey == "" {
		appKey = ApplicationWeb
	}
	cfg, ok := c.oauth2Configs[appKey]
	if !ok {
		return oauth2.Config{}, fmt.Errorf("%w: %s", ErrApplicationNotConfigured, appKey)
	}
	return cfg, nil
}

func (c *Client) ApplicationKeyForClientID(clientID string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return ""
	}
	for appKey, cfg := range c.oauth2Configs {
		if cfg.ClientID == clientID {
			return appKey
		}
	}
	return ""
}

func (c *Client) GetAuthURLForApplication(appKey, state string) (string, string, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	return c.rewriteBrowserAuthURL(authURL), verifier, nil
}

func (c *Client) GetReauthURLForApplication(appKey, state string) (string, string, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "login"),
		oauth2.SetAuthURLParam("max_age", "0"),
	)
	return c.rewriteBrowserAuthURL(authURL), verifier, nil
}

func (c *Client) GetSignupURLForApplication(appKey, state string) (string, string, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	return c.rewriteBrowserAuthURL(casdoorSignupAuthURL(authURL)), verifier, nil
}

func (c *Client) GetStepUpAuthURLForApplication(appKey, state string) (string, string, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "login"),
		oauth2.SetAuthURLParam("max_age", "0"),
		oauth2.SetAuthURLParam("acr_values", "mfa"),
	)
	return c.rewriteBrowserAuthURL(authURL), verifier, nil
}

func (c *Client) GetAuthURL(clientID string, redirectURI string, scopes []string, state string) string {
	return BuildAuthURL(c.oauth2Cfg.Endpoint.AuthURL, clientID, redirectURI, scopes, state)
}

func (c *Client) AuthorizeEndpoint() string {
	if c == nil {
		return ""
	}
	return c.oauth2Cfg.Endpoint.AuthURL
}

func BuildAuthURL(authorizeEndpoint, clientID, redirectURI string, scopes []string, state string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", strings.TrimSpace(clientID))
	values.Set("redirect_uri", strings.TrimSpace(redirectURI))
	values.Set("scope", strings.Join(oauth2Scopes(scopes), " "))
	if trimmed := strings.TrimSpace(state); trimmed != "" {
		values.Set("state", trimmed)
	}
	return strings.TrimRight(authorizeEndpoint, "?") + "?" + values.Encode()
}

func casdoorSignupAuthURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if strings.HasSuffix(parsed.Path, "/login/oauth/authorize") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/login/oauth/authorize") + "/signup/oauth/authorize"
	}
	return parsed.String()
}

func (c *Client) rewriteBrowserAuthURL(rawURL string) string {
	base := strings.TrimRight(strings.TrimSpace(c.publicAuthBaseURL), "/")
	if base == "" {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	publicBase, err := url.Parse(base)
	if err != nil || publicBase.Scheme == "" || publicBase.Host == "" {
		return rawURL
	}
	if casdoorBrowserAuthPath(parsed.Path) && publicBase.Host != parsed.Host {
		return rawURL
	}
	parsed.Scheme = publicBase.Scheme
	parsed.Host = publicBase.Host
	if basePath := strings.TrimRight(publicBase.Path, "/"); basePath != "" {
		parsed.Path = basePath + parsed.Path
	}
	return parsed.String()
}

func casdoorBrowserAuthPath(path string) bool {
	return path == "/login/oauth/authorize" ||
		path == "/signup/oauth/authorize" ||
		strings.HasPrefix(path, "/login/oauth/") ||
		strings.HasPrefix(path, "/signup/oauth/")
}
