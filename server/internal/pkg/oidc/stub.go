package oidc

import (
	"golang.org/x/oauth2"
)

// NewStubClient 创建仅用于测试的 OIDC Client stub。
// 此函数仅供 _test.go 文件调用，不应被生产代码引用。
// authBaseURL 用于生成 AuthCodeURL。不可用于 ExchangeCode/VerifyIDToken 等需要真实 provider 的操作。
func NewStubClient(authBaseURL string) *Client {
	cfg := oauth2.Config{
		ClientID:    "test-client-id",
		RedirectURL: "http://localhost:3000/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  authBaseURL,
			TokenURL: authBaseURL + "/token",
		},
		Scopes: []string{"openid"},
	}
	adminCfg := cfg
	adminCfg.ClientID = "test-admin-client-id"
	uniappCfg := cfg
	uniappCfg.ClientID = "test-uniapp-client-id"
	return &Client{
		oauth2Cfg: cfg,
		oauth2Configs: map[string]oauth2.Config{
			ApplicationWeb:    cfg,
			ApplicationAdmin:  adminCfg,
			ApplicationUniapp: uniappCfg,
		},
		allowedClientIDs: map[string]struct{}{
			cfg.ClientID:       {},
			adminCfg.ClientID:  {},
			uniappCfg.ClientID: {},
		},
	}
}
