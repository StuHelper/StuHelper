package oidc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// ExchangeCode 用授权码 + PKCE code_verifier 交换 Token。
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	return c.ExchangeCodeForApplication(ctx, ApplicationWeb, code, codeVerifier)
}

func (c *Client) ExchangeCodeForApplication(ctx context.Context, appKey, code, codeVerifier string) (*oauth2.Token, error) {
	return c.ExchangeCodeForApplicationWithRedirectURI(ctx, appKey, code, codeVerifier, "")
}

func (c *Client) ExchangeCodeForApplicationWithRedirectURI(
	ctx context.Context,
	appKey, code, codeVerifier, redirectURI string,
) (*oauth2.Token, error) {
	var err error
	code, err = normalizeRequiredAuthorizationCode(code)
	if err != nil {
		return nil, err
	}
	codeVerifier, err = normalizeRequiredPKCEVerifier(codeVerifier)
	if err != nil {
		return nil, err
	}
	cfg, err := c.oauth2ConfigForApplicationWithRedirectURI(appKey, redirectURI)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	metrics.ObserveExternalRequest(c.metricName, "exchange_code", start, err)
	if err != nil {
		return nil, fmt.Errorf("oidc: code exchange failed: %w", classifyOAuthError(err))
	}
	return token, nil
}

// RefreshToken 用 refresh token 获取新的 token 对
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	return c.RefreshTokenForApplication(ctx, ApplicationWeb, refreshToken)
}

func (c *Client) RefreshTokenForApplication(ctx context.Context, appKey, refreshToken string) (*oauth2.Token, error) {
	var err error
	refreshToken, err = normalizeRequiredRefreshToken(refreshToken, "refresh")
	if err != nil {
		return nil, err
	}
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	tokenSource := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	token, err := tokenSource.Token()
	metrics.ObserveExternalRequest(c.metricName, "refresh_token", start, err)
	if err != nil {
		return nil, fmt.Errorf("oidc: token refresh failed: %w", classifyOAuthRefreshError(err))
	}
	return token, nil
}

// ExtractIDToken 从 OAuth2 Token 的 extra 字段中提取 id_token
func ExtractIDToken(token *oauth2.Token) string {
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(raw)
}
