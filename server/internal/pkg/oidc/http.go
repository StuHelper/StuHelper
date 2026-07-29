package oidc

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/observability"
)

func newOIDCHTTPClient(cfg config.CasdoorConfig, clientName string) *http.Client {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   5,
	}

	internalAddr := strings.TrimSpace(cfg.InternalAddress)
	issuerAddr := issuerDialAddress(cfg.Issuer)
	if internalAddr != "" && issuerAddr != "" {
		internalAddr = internalDialAddress(internalAddr, issuerAddr)

		dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			if address == issuerAddr {
				address = internalAddr
			}
			return dialer.DialContext(ctx, network, address)
		}
	}

	return observability.WrapHTTPClient(&http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}, clientName)
}

func issuerDialAddress(issuer string) string {
	if issuer == "" {
		return ""
	}
	u, err := url.Parse(issuer)
	if err != nil || u.Host == "" {
		return ""
	}
	if port := u.Port(); port != "" {
		return net.JoinHostPort(u.Hostname(), port)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return net.JoinHostPort(u.Hostname(), "443")
	default:
		return net.JoinHostPort(u.Hostname(), "80")
	}
}

func internalDialAddress(internalAddr, issuerAddr string) string {
	internalAddr = strings.TrimSpace(internalAddr)
	if internalAddr == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(internalAddr); err == nil {
		return internalAddr
	}
	_, issuerPort, err := net.SplitHostPort(issuerAddr)
	if err != nil || issuerPort == "" {
		return internalAddr
	}
	return net.JoinHostPort(strings.Trim(internalAddr, "[]"), issuerPort)
}
