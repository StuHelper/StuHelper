package oidc

import (
	"golang.org/x/oauth2"
)

// NewStubClient 创建仅用于测试的 OIDC Client stub。
// authBaseURL 用于生成 AuthCodeURL。不可用于 ExchangeCode/VerifyIDToken 等需要真实 provider 的操作。
func NewStubClient(authBaseURL string) *Client {
	return &Client{
		oauth2Cfg: oauth2.Config{
			ClientID:    "test-client-id",
			RedirectURL: "http://localhost:3000/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  authBaseURL,
				TokenURL: authBaseURL + "/token",
			},
			Scopes: []string{"openid"},
		},
	}
}
